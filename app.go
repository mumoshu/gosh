package gosh

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

// This program is inspired by https://github.com/progrium/go-basher
// Much appreciation to the author!

type App struct {
	BashPath   string
	Dir        string
	TriggerArg string
	SelfPath   string
	SelfArgs   []string
	Pkg        string
	Debug      bool

	funcs map[string]FunWithOpts
}

func (c *App) HandleFuncs(ctx Context, args []interface{}, outs []Output) bool {
	return c.handleFuncs(ctx, args, outs, map[FunID]struct{}{})
}

func (c *App) handleFuncs(ctx Context, args []interface{}, outs []Output, called map[FunID]struct{}) bool {
	for i, arg := range args {
		// With ::: (Deprecated)
		if c.TriggerArg == "" || (arg == c.TriggerArg && len(args) > i+1) {
			for cmd, funWithOpts := range c.funcs {
				if cmd == args[i+1] {
					for _, d := range funWithOpts.Opts.Deps {
						w := WrapContext(ctx)
						v := c.HandleFuncs(w, append([]interface{}{d.Name}, d.Args...), nil)
						if !v {
							ctx.Err(fmt.Sprintf("unable to start function %s due to dep error: %v", args[i+1], w.GetErr()))
							return true
						}
					}

					funID := NewFunID(Dependency{Name: args[i+1], Args: args[i+2:]})

					if _, v := called[funID]; v {
						// this function has been already called successfully. Se don't
						// need to call it twice.
						return true
					}

					fmt.Fprintf(os.Stderr, "gosh.App.handleFuncs :::: cmd=%s, funID=%s\n", cmd, funID)

					retVals := c.funcs[cmd].Fun.Call(ctx, args[i+2:])

					if len(outs) > len(retVals) {
						ctx.Err(fmt.Sprintf("%s: missing outputs: expected %d, got %d return values", cmd, len(outs), len(retVals)))
						return true
					}

					for i, o := range outs {
						o.value.Set(retVals[i])
					}

					called[NewFunID(Dependency{Name: args[i+1], Args: args[i+2:]})] = struct{}{}

					return true
				}
			}

			ctx.Err(fmt.Sprintf("function %s not found", args[i+1]))

			return false
		}
	}

	var fnName string
	switch typed := args[0].(type) {
	case string:
		fnName = typed
	default:
		fnName = FuncOrMethodToCmdName(typed)
	}

	// Without :::
	if funWithOpts, ok := c.funcs[fnName]; ok {
		for _, d := range funWithOpts.Opts.Deps {
			w := WrapContext(ctx)
			v := c.HandleFuncs(w, append([]interface{}{d.Name}, d.Args...), nil)
			if !v {
				ctx.Err(fmt.Sprintf("unable to start function %s due to dep error: %v", args[0], w.GetErr()))
				return true
			}
		}

		funID := NewFunID(Dependency{Name: args[0], Args: args[1:]})

		if _, v := called[funID]; v {
			// this function has been already called successfully. Se don't
			// need to call it twice.
			return true
		}

		fmt.Fprintf(os.Stderr, "gosh.App.handleFuncs: cmd=%s, funID=%s\n", fnName, funID)

		retVals := funWithOpts.Fun.Call(ctx, args[1:])

		if len(outs) > len(retVals) {
			ctx.Err(fmt.Sprintf("%s: missing outputs: expected %d, got %d return values", args[0], len(outs), len(retVals)))
			return true
		}

		for i, o := range outs {
			o.value.Set(retVals[i])
		}

		called[NewFunID(Dependency{Name: args[0], Args: args[1:]})] = struct{}{}

		return true
	}

	return false
}

func (c *App) printEnv(file io.Writer, interactive bool) {
	var selfArgs []string

	selfArgs = append(selfArgs, c.SelfArgs...)

	var buildArgs []string

	if buildTag := os.Getenv("GOSH_BUILD_TAG"); buildTag != "" {
		buildArgs = append(buildArgs, "-tags="+buildTag)
	}
	buildArgs = append(buildArgs, c.Pkg)

	// variables
	file.Write([]byte("unset BASH_ENV\n")) // unset for future calls to bash
	file.Write([]byte("export SELF=" + os.Args[0] + "\n"))
	file.Write([]byte("export SELF_ARGS=\"" + strings.Join(selfArgs, " ") + "\"\n"))
	file.Write([]byte("export SELF_EXECUTABLE='" + c.SelfPath + "'\n"))

	// file.Write([]byte("export PS0='exec go run ./run'\n"))
	// functions
	if len(c.funcs) > 0 {
		file.Write([]byte(`
mkdir -p .cmds
export PATH=$(pwd)/.cmds:$PATH
`))
	}
	for cmd := range c.funcs {
		file.Write([]byte(`
cat <<'EOS' > .cmds/` + cmd + `
#!/usr/bin/env bash
$SELF_EXECUTABLE $SELF_ARGS ::: ` + cmd + ` "$@"
EOS
chmod +x .cmds/` + cmd + `
`))
		// file.Write([]byte(cmd + "() { $SELF_EXECUTABLE ::: " + cmd + " \"$@\"; }\n"))
	}
	// 	file.Write([]byte(`
	// _gosh_hook() {
	// 	local previous_exit_status=$?;
	// 	trap -- '' SIGINT;
	// 	go build -o run2 ./run;
	// 	export SELF_EXECUTABLE=$(pwd)/run2;
	// 	trap - SIGINT;
	// 	return $previous_exit_status;
	// };
	// if ! [[ "${PROMPT_COMMAND:-}" =~ _direnv_hook ]]; then
	// 	PROMPT_COMMAND="_gosh_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
	// fi
	// `))
	if c.Pkg != "" && interactive {
		file.Write([]byte(`
preexec () { :; }
preexec_invoke_exec () {
[ -n "$COMP_LINE" ] && return  # do nothing if completing
[ "$BASH_COMMAND" = "$PROMPT_COMMAND" ] && return # don't cause a preexec for $PROMPT_COMMAND
NEWBIN=run2
go build -o $NEWBIN ` + strings.Join(buildArgs, " ") + `
export SELF_EXECUTABLE=$(pwd)/$NEWBIN
eval "$(./$NEWBIN env)"
}
trap 'preexec_invoke_exec' DEBUG
`))
	}
}

func (c *App) buildEnvfile(interactive bool) (string, error) {
	file, err := ioutil.TempFile(c.Dir, "bashenv.")
	if err != nil {
		return "", err
	}
	defer file.Close()

	c.printEnv(file, interactive)

	return file.Name(), nil
}

func (c *App) runInteractiveShell(ctx Context) (int, error) {
	var interactive bool

	osFile, isOsFile := ctx.Stdin().(*os.File)
	if isOsFile {
		interactive = terminal.IsTerminal(int(osFile.Fd()))
	}

	return c.runInternal(ctx, interactive, nil)
}

func (c *App) runNonInteractiveShell(ctx Context, args []string) (int, error) {
	var isCmd bool

	if len(args) > 0 {
		if info, _ := os.Stat(args[0]); info == nil {
			isCmd = true
		}
	}

	var bashArgs []string

	if isCmd {
		bashArgs = append(bashArgs, "-c")
		var bashCmd []string
		for _, a := range args {
			bashCmd = append(bashCmd, `"`+strings.ReplaceAll(a, `"`, `\"`)+`"`)
		}
		bashArgs = append(bashArgs, strings.Join(bashCmd, " "))
	} else {
		bashArgs = append(bashArgs, args...)
	}

	return c.runInternal(ctx, false, bashArgs)
}

func (c *App) runInternal(ctx Context, interactive bool, args []string) (int, error) {
	envfile, err := c.buildEnvfile(interactive)
	if err != nil {
		return 0, err
	}
	if !c.Debug {
		defer os.Remove(envfile)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals)
	// Avoid receiving "urgent I/O condition" signals
	// See https://golang.hateblo.jp/entry/golang-signal-urgent-io-condition
	signal.Ignore(syscall.Signal(0x17))

	bashArgs := []string{"--rcfile", envfile}

	bashArgs = append(bashArgs, args...)

	println(fmt.Sprintf("App.run: running %v: bashArgs %v (%d)", args, bashArgs, len(bashArgs)))

	cmd := exec.Command(c.BashPath, bashArgs...)
	cmd.Env = append(os.Environ(), "BASH_ENV="+envfile)
	cmd.Stdin = ctx.Stdin()
	cmd.Stdout = ctx.Stdout()
	cmd.Stderr = ctx.Stderr()
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	errChan := make(chan error, 1)
	go func() {
		for sig := range signals {
			if sig != syscall.SIGCHLD {
				fmt.Fprintf(os.Stderr, "signal received: %v\n", sig)
				err = cmd.Process.Signal(sig)
				if err != nil {
					errChan <- err
				}
			}
		}
	}()
	go func() {
		errChan <- cmd.Wait()
	}()
	err = <-errChan
	return exitStatus(err)
}

func (app *App) Run(ctx Context, args []interface{}, cfg RunConfig) error {
	outs := cfg.Outputs
	stdout := cfg.Stdout
	stderr := cfg.Stderr

	if len(args) == 1 && args[0] == "env" {
		app.printEnv(os.Stdout, true)

		return nil
	}

	if len(args) == 0 {
		_, err := app.runInteractiveShell(ctx)

		return err
	}

	if ctx == nil {
		c := &context{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}

		if stdout.w != nil {
			c.stdout = stdout.w
		}

		if stderr.w != nil {
			c.stderr = stderr.w
		}

		ctx = c
	}

	funExists := app.HandleFuncs(ctx, args, outs)

	if err := ctx.GetErr(); err != nil {
		fmt.Fprintf(ctx.Stderr(), "%v\n", err)
		return err
	}

	if funExists {
		return nil
	}

	var shellArgs []string
	for _, v := range args {
		if s, ok := v.(string); !ok {
			return fmt.Errorf("%v(%T) cannot be converted to string", v, v)
		} else {
			shellArgs = append(shellArgs, s)
		}
	}

	_, err := app.runNonInteractiveShell(ctx, shellArgs)

	return err
}

func exitStatus(err error) (int, error) {
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// There is no platform independent way to retrieve
			// the exit code, but the following will work on Unix
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if status.ExitStatus() != -1 {
					return status.ExitStatus(), err
				} else {
					// The process hasn't exited or was terminated by a signal.
					return int(status), err
				}
			}
		}
		return 0, err
	}
	return 0, nil
}

type FunOption func(*FunOptions)

type FunOptions struct {
	Deps []Dependency
}

type Dependency struct {
	Name interface{}
	Args []interface{}
}

type Fun struct {
	F interface{}
}

func (fn Fun) Call(ctx Context, args []interface{}) []reflect.Value {
	// switch f := fn.F.(type) {
	// case func(Context, []string):
	// 	f(ctx, args)

	// 	return nil
	// default:
	return Call(ctx, fn.F, args...)
	// }
}

type FunWithOpts struct {
	Fun  Fun
	Opts FunOptions
}

type Diagnostic struct {
	Timestamp time.Time
	Message   string
}

func (d Diagnostic) String() string {
	ts := d.Timestamp.Format(time.RFC3339)
	return fmt.Sprintf("%s\t%s", ts, d.Message)
}

type Diagnostics []Diagnostic

func (d Diagnostics) String() string {
	var diags []string
	for _, a := range d {
		diags = append(diags, fmt.Sprintf("%s", a))
	}
	summary := strings.Join(diags, ", ")
	return summary
}

type Shell struct {
	sync.Mutex

	diags Diagnostics
	funcs map[string]FunWithOpts

	sync.Once

	app *App

	additionalCallerSkip int
}

func (t *Shell) Def(args ...interface{}) {
	t.Lock()
	defer t.Unlock()

	if t.funcs == nil {
		t.funcs = map[string]FunWithOpts{}
	}

	var fn interface{}
	var opts []FunOption

	funOptionType := reflect.TypeOf(FunOption(func(fo *FunOptions) {}))

	var name string

	for i, a := range args {
		aType := reflect.TypeOf(a)
		if aType.AssignableTo(funOptionType) {
			opts = append(opts, a.(FunOption))
		} else {
			if i == 0 {
				s, ok := a.(string)
				if ok {
					name = s
				} else {
					name = FuncOrMethodToCmdName(a)
					fn = a
				}

				continue
			}

			if fn != nil {
				panic("you cannot have two or more fns")
			}

			fn = a
		}
	}

	var funOpts FunOptions

	for _, o := range opts {
		o(&funOpts)
	}

	t.Diagf("registering func %s", name)

	t.funcs[name] = FunWithOpts{Fun: Fun{fn}, Opts: funOpts}
}

func (t *Shell) Diagf(format string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)

	callerInfo := fmt.Sprintf("%s:%d\t", filepath.Base(file), line)
	diag := Diagnostic{Timestamp: time.Now(), Message: callerInfo + fmt.Sprintf(format, args...)}

	t.diags = append(t.diags, diag)

	diagsOut := os.NewFile(3, "diagnostics")
	if diagsOut != nil {
		fmt.Fprintf(diagsOut, "%s\n", diag)
	}
}

func FuncOrMethodToCmdName(f interface{}) string {
	v := reflect.ValueOf(f)
	name := runtime.FuncForPC(v.Pointer()).Name()
	vs := strings.Split(name, ".")

	base := vs[len(vs)-1]

	// https://stackoverflow.com/questions/32925344/why-is-there-a-fm-suffix-when-getting-a-functions-name-in-go
	if strings.HasSuffix(base, "-fm") {
		base = strings.TrimSuffix(base, "-fm")
	}

	return strings.ToLower(base)
}

func Dep(name string, args ...interface{}) FunOption {
	return func(o *FunOptions) {
		o.Deps = append(o.Deps, Dependency{Name: name, Args: args})
	}
}

type Command struct {
	Vars []interface{}
}

func Cmd(vars ...interface{}) Command {
	return Command{Vars: vars}
}

func (t *Shell) runPipeline(ctx Context, cmds []Command) error {
	precedents, final := cmds[:len(cmds)-1], cmds[len(cmds)-1]

	errs := make([]error, len(cmds))

	var wg sync.WaitGroup

	for i := range precedents {
		i := i
		var errCh <-chan error
		ctx, errCh = t.GoPipe(ctx, precedents[i].Vars...)
		wg.Add(1)
		go func() {
			errs[i] = <-errCh
			wg.Done()
		}()
	}

	var errCh <-chan error
	errCh = t.GoRun(ctx, final.Vars...)
	wg.Add(1)
	go func() {
		errs[len(cmds)-1] = <-errCh
		wg.Done()
	}()

	wg.Wait()

	for i, err := range errs {
		if err != nil {
			return fmt.Errorf("command %v at index %d failed: %v", cmds[i].Vars, i, err)
		}
	}

	return nil
}

type Output struct {
	value reflect.Value
}

func Out(p interface{}) Output {
	return Output{value: reflect.ValueOf(p).Elem()}
}

type StdoutSink struct {
	w io.Writer
}

func WriteStdout(w io.Writer) StdoutSink {
	return StdoutSink{
		w: w,
	}
}

type StderrSink struct {
	w io.Writer
}

func WriteStderr(w io.Writer) StderrSink {
	return StderrSink{
		w: w,
	}
}

type RunConfig struct {
	Outputs []Output
	Stdout  StdoutSink
	Stderr  StderrSink
}

var TestEnvName = "FOO"

func (t *Shell) In(testCtx *testing.T, f func()) {
	if os.Getenv(TestEnvName) != "" {
		var osArgs []string

		var i int
		var a string

		for i, a = range os.Args {
			if a == ":::" {
				break
			}
		}

		osArgs = os.Args[i+1:]

		var runArgs []interface{}
		for _, a := range osArgs {
			runArgs = append(runArgs, a)
		}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		origStdout := os.Stdout
		origStderr := os.Stderr

		tempDir := os.Getenv("ARCTEST_TEMPDIR")

		// Note that panics aren't redirected to this log file.
		// See https://github.com/golang/go/issues/325
		//
		// Also, from what I have observed, println aren't redirect to the log file, too.
		if tempDir == "" {
			tempDir = testCtx.TempDir()
		}

		logFile, err := ioutil.TempFile(tempDir, "stdoutandstderr.log")
		if err != nil {
			testCtx.Fatal(err)
		}

		os.Stdout = logFile
		os.Stderr = logFile

		fmt.Fprintf(os.Stderr, "ARGS=%v\n", runArgs)
		if err := t.Run(append(runArgs, WriteStdout(&stdout), WriteStderr(&stderr))...); err != nil {
			testCtx.Error(err)
		}

		fmt.Fprint(origStderr, stderr.String())
		fmt.Fprint(origStdout, stdout.String())

		return
	}

	f()
}

func (t *Shell) MustExec(osArgs []string) {
	var args []interface{}
	for _, a := range osArgs[1:] {
		args = append(args, a)
	}

	t.additionalCallerSkip = 1

	if err := t.Run(args...); err != nil {
		log.Fatal(err)
	}
}

func (t *Shell) Run(vars ...interface{}) error {
	var args []interface{}
	var ctx Context
	var cmds []Command
	var outs []Output
	var stdout StdoutSink
	var stderr StderrSink
	var testCtx *testing.T

	for _, v := range vars {
		switch typed := v.(type) {
		case Context:
			ctx = typed
		case string:
			args = append(args, typed)
		case []string:
			args = append(args, typed)
		case Command:
			cmds = append(cmds, typed)
		case Output:
			outs = append(outs, typed)
		case StdoutSink:
			stdout = typed
		case StderrSink:
			stderr = typed
		case *testing.T:
			testCtx = typed
		default:
			if reflect.TypeOf(v).Kind() == reflect.Func {
				args = append(args, FuncOrMethodToCmdName(v))
				continue
			}

			args = append(args, typed)
			// panic(fmt.Errorf("unexpected vars: %v", vars))
		}
	}

	if ctx == nil {
		c := &context{
			stdin:  os.Stdin,
			stdout: os.Stdout,
			stderr: os.Stderr,
		}

		if stdout.w != nil {
			c.stdout = stdout.w
		}

		if stderr.w != nil {
			c.stderr = stderr.w
		}

		ctx = c
	}

	t.Diagf("Running %v", args)

	var initErr error

	t.Once.Do(func() {
		ex, err := os.Executable()
		if err != nil {
			println(err.Error())
			initErr = err
			return
		}

		dir, err := os.Getwd()
		if err != nil {
			println(err.Error())
			initErr = err
			return
		}

		// Without sync.Once, the number should be 1
		_, filename, _, _ := runtime.Caller(4 + t.additionalCallerSkip)

		var pkg string

		if _, err := os.Stat(filename); err == nil {
			pkg = filepath.Dir(filename)
		}

		var selfArgs []string

		if testCtx != nil {
			os.Setenv(TestEnvName, "foobar")

			selfArgs = append(selfArgs, os.Args[1:]...)

			var testRunExists bool
			for _, a := range os.Args[1:] {
				if strings.HasPrefix(a, "-test.run=") {
					testRunExists = true
					break
				}
			}

			// Needed to only trigger the target command when you run all the go tests
			if !testRunExists {
				selfArgs = append(selfArgs, "-test.run=^"+testCtx.Name()+"$")
			}
		}

		t.app = &App{
			funcs:      t.funcs,
			Pkg:        pkg,
			BashPath:   "/bin/bash",
			Dir:        dir,
			TriggerArg: ":::",
			SelfPath:   ex,
			SelfArgs:   selfArgs,
		}
	})

	if initErr != nil {
		return initErr
	}

	if len(cmds) > 0 {
		return t.runPipeline(ctx, cmds)
	}

	return t.app.Run(ctx, args, RunConfig{
		Outputs: outs,
		Stdout:  stdout,
		Stderr:  stderr,
	})
}

func (c *App) Dep(args ...interface{}) error {
	return nil
}

func (c *App) DepString(args ...interface{}) (string, error) {
	return "", nil
}

func (c *App) DepStringMap(args ...interface{}) (map[string]string, error) {
	return nil, nil
}

func (sh *Shell) GoPipe(ctx Context, vars ...interface{}) (Context, <-chan error) {
	a, b, close := sh.PipeFromContext(ctx)

	err := make(chan error)

	go func() {
		vars = append([]interface{}{a}, vars...)
		e := sh.Run(vars...)
		close()
		err <- e
	}()

	return b, err
}

func (sh *Shell) GoRun(ctx Context, vars ...interface{}) <-chan error {
	err := make(chan error)

	go func() {
		vars = append([]interface{}{ctx}, vars...)
		e := sh.Run(vars...)
		err <- e
	}()

	return err
}

func (sh *Shell) PipeFromContext(ctx Context) (Context, Context, func()) {
	a, b := &context{}, &context{}

	r, w := io.Pipe()

	a.stdin = ctx.Stdin()
	a.stdout = w
	a.stderr = ctx.Stderr()

	b.stdin = r
	b.stdout = ctx.Stdout()
	b.stderr = ctx.Stderr()

	return a, b, func() {
		if err := w.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
