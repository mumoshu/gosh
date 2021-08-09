package gosh

import (
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

	"github.com/mumoshu/gosh/context"

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
	Env        []string

	funcs map[string]FunWithOpts
}

func (c *App) HandleFuncs(ctx context.Context, args []interface{}, outs []Output) (bool, error) {
	retVals, ret, err := c.handleFuncs(ctx, args, outs, map[FunID]struct{}{})

	if err != nil {
		return ret, err
	}

	for i, o := range outs {
		o.value.Set(retVals[i])
	}

	return ret, nil
}

func (c *App) handleFuncs(ctx context.Context, args []interface{}, outs []Output, called map[FunID]struct{}) ([]reflect.Value, bool, error) {
	for i, arg := range args {
		// With ::: (Deprecated)
		if c.TriggerArg == "" || (arg == c.TriggerArg && len(args) > i+1) {
			for cmd, funWithOpts := range c.funcs {
				if cmd == args[i+1] {
					for _, d := range funWithOpts.Opts.Deps {
						_, v, err := c.handleFuncs(ctx, append([]interface{}{d.Name}, d.Args...), nil, called)
						if !v {
							return nil, true, fmt.Errorf("unable to start function %s due to dep error: %w", args[i+1], err)
						}
					}

					funID := NewFunID(Dependency{Name: args[i+1], Args: args[i+2:]})

					if _, v := called[funID]; v {
						// this function has been already called successfully. We don't
						// need to call it twice.
						return nil, true, nil
					}

					// fmt.Fprintf(os.Stderr, "gosh.App.handleFuncs :::: cmd=%s, funID=%s\n", cmd, funID)

					retVals, err := c.funcs[cmd].Fun.Call(ctx, args[i+2:])
					if err != nil {
						return nil, true, err
					}

					if len(outs) > len(retVals) {
						return nil, true, fmt.Errorf("%s: missing outputs: expected %d, got %d return values", cmd, len(outs), len(retVals))
					}

					called[NewFunID(Dependency{Name: args[i+1], Args: args[i+2:]})] = struct{}{}

					return retVals, true, nil
				}
			}

			return nil, false, fmt.Errorf("function %s not found", args[i+1])
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
			_, v, err := c.handleFuncs(ctx, append([]interface{}{d.Name}, d.Args...), nil, called)
			if !v {
				return nil, true, fmt.Errorf("unable to start function %s due to dep error: %w", args[0], err)
			}
		}

		funID := NewFunID(Dependency{Name: args[0], Args: args[1:]})

		if _, v := called[funID]; v {
			// this function has been already called successfully. Se don't
			// need to call it twice.
			return nil, true, nil
		}

		// fmt.Fprintf(os.Stderr, "gosh.App.handleFuncs: cmd=%s, funID=%s\n", fnName, funID)

		retVals, err := funWithOpts.Fun.Call(ctx, args[1:])
		if err != nil {
			return nil, true, err
		}

		if len(outs) > len(retVals) {
			return nil, true, fmt.Errorf("%s: missing outputs: expected %d, got %d return values", args[0], len(outs), len(retVals))
		}

		called[NewFunID(Dependency{Name: args[0], Args: args[1:]})] = struct{}{}

		return retVals, true, nil
	}

	return nil, false, nil
}

func (c *App) printEnv(file io.Writer, interactive bool) {
	if strings.HasSuffix(os.Args[0], ".test") && len(c.SelfArgs) == 0 {
		panic(fmt.Errorf("[bug] empty self args while running: %v", os.Args))
	}

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
	files, err := filepath.Glob(filepath.Join(c.Dir, "bashenv.*"))
	if err != nil {
		return "", fmt.Errorf("failed globbing bashenv files: %w", err)
	}

	if len(files) > 5 {
		return "", fmt.Errorf("too many bashenv files found (%d). perhaps you've falling into a infinite recursion?", len(files))
	}

	file, err := ioutil.TempFile(c.Dir, "bashenv.")
	if err != nil {
		return "", err
	}
	defer file.Close()

	c.printEnv(file, interactive)

	return file.Name(), nil
}

func (c *App) runInteractiveShell(ctx context.Context) (int, error) {
	var interactive bool

	osFile, isOsFile := context.Stdin(ctx).(*os.File)
	if isOsFile {
		interactive = terminal.IsTerminal(int(osFile.Fd()))
	}

	return c.runInternal(ctx, interactive, nil, RunConfig{})
}

func (c *App) runNonInteractiveShell(ctx context.Context, args []string, cfg RunConfig) (int, error) {
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

	return c.runInternal(ctx, false, bashArgs, cfg)
}

func (c *App) runInternal(ctx context.Context, interactive bool, args []string, cfg RunConfig) (int, error) {
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

	bashArgs := []string{}

	if interactive {
		bashArgs = append(bashArgs, "--rcfile", envfile)
	}

	bashArgs = append(bashArgs, args...)

	// println(fmt.Sprintf("App.run: running %v: bashArgs %v (%d)", args, bashArgs, len(bashArgs)))

	cmd := exec.Command(c.BashPath, bashArgs...)
	cmd.Env = os.Environ()
	if !interactive {
		cmd.Env = append(cmd.Env, "BASH_ENV="+envfile)
	}
	cmd.Dir = cfg.Dir
	cmd.Env = append(cmd.Env, cfg.Env...)
	cmd.Stdin = context.Stdin(ctx)
	cmd.Stdout = context.Stdout(ctx)
	cmd.Stderr = context.Stderr(ctx)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	errChan := make(chan error, 1)
	go func() {
		for sig := range signals {
			if sig != syscall.SIGCHLD {
				// fmt.Fprintf(os.Stderr, "signal received: %v\n", sig)
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

func (app *App) Run(ctx context.Context, args []interface{}, cfg RunConfig) error {
	outs := cfg.Outputs
	stdout := cfg.Stdout
	stderr := cfg.Stderr

	if len(args) == 1 && args[0] == "env" {
		app.printEnv(os.Stdout, true)

		return nil
	}

	if ctx == nil {
		ctx = context.Background()
		ctx = context.WithStdin(ctx, os.Stdin)
		ctx = context.WithStdout(ctx, os.Stdout)
		ctx = context.WithStderr(ctx, os.Stderr)
	}

	if stdout.w != nil {
		ctx = context.WithStdout(ctx, stdout.w)
	}

	if stderr.w != nil {
		ctx = context.WithStderr(ctx, stderr.w)
	}

	ctx = context.WithVariables(ctx, map[string]interface{}{})

	if len(args) == 0 {
		_, err := app.runInteractiveShell(ctx)

		return err
	}

	funExists, err := app.HandleFuncs(ctx, args, outs)
	if err != nil {
		fmt.Fprintf(context.Stderr(ctx), "%v\n", err)
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

	_, err = app.runNonInteractiveShell(ctx, shellArgs, cfg)

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
	Name string
	F    interface{}
	M    *reflect.Value
}

func (fn Fun) Call(ctx context.Context, args []interface{}) ([]reflect.Value, error) {
	// switch f := fn.F.(type) {
	// case func(Context, []string):
	// 	f(ctx, args)

	// 	return nil
	// default:

	if fn.M != nil {
		return CallMethod(ctx, fn.Name, *fn.M, args...)
	}

	return CallFunc(ctx, fn.Name, fn.F, args...)
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

func (t *Shell) Export(args ...interface{}) {
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
		} else if aType.NumMethod() > 0 {
			v := reflect.ValueOf(a)
			for i := 0; i < aType.NumMethod(); i++ {
				typeM := aType.Method(i)
				name := typeM.Name
				m := v.Method(i)
				t.export(strings.ToLower(name), nil, &m, opts)
			}
		} else if aType.Kind() == reflect.Struct || aType.Kind() == reflect.Ptr {
			panic("struct must have one or more public functions to exported")
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

	if fn != nil {
		t.export(name, fn, nil, opts)
	}
}

func (t *Shell) export(name string, fn interface{}, m *reflect.Value, opts []FunOption) {
	var funOpts FunOptions

	for _, o := range opts {
		o(&funOpts)
	}

	t.Diagf("registering func %s", name)

	if m != nil {
		t.funcs[name] = FunWithOpts{Fun: Fun{Name: name, M: m}, Opts: funOpts}
	} else if fn != nil {
		t.funcs[name] = FunWithOpts{Fun: Fun{Name: name, F: fn}, Opts: funOpts}
	} else {
		panic(fmt.Errorf("unexpected args passed to export %s: fn=%v, m=%v, opts=%v", name, fn, m, opts))
	}
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
	return ReflectValueToCmdName(v)
}

func ReflectValueToCmdName(v reflect.Value) string {
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

func (t *Shell) runPipeline(ctx context.Context, cmds []Command) error {
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

func Env(v ...string) RunOption {
	return func(rc *RunConfig) {
		for _, v := range v {
			rc.Env = append(rc.Env, v)
		}
	}
}

func Dir(dir string) RunOption {
	return func(rc *RunConfig) {
		rc.Dir = dir
	}
}

type StdoutSink struct {
	w io.Writer
}

func WriteStdout(w io.Writer) RunOption {
	return func(rc *RunConfig) {
		rc.Stdout = StdoutSink{
			w: w,
		}
	}
}

type StderrSink struct {
	w io.Writer
}

func WriteStderr(w io.Writer) RunOption {
	return func(rc *RunConfig) {
		rc.Stderr = StderrSink{
			w: w,
		}
	}
}

type RunOption func(*RunConfig)

type RunConfig struct {
	Outputs []Output
	Stdout  StdoutSink
	Stderr  StderrSink
	Env     []string
	Dir     string
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
	var ctx context.Context
	var cmds []Command
	var testCtx *testing.T
	var rc RunConfig
	for _, v := range vars {
		switch typed := v.(type) {
		case context.Context:
			ctx = typed
		case string:
			args = append(args, typed)
		case []string:
			args = append(args, typed)
		case Command:
			cmds = append(cmds, typed)
		case Output:
			rc.Outputs = append(rc.Outputs, typed)
		case StdoutSink:
			rc.Stdout = typed
		case StderrSink:
			rc.Stderr = typed
		case RunOption:
			typed(&rc)
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
		ctx = context.Background()
	}

	if rc.Stdout.w != nil {
		ctx = context.WithStdout(ctx, rc.Stdout.w)
	}

	if rc.Stderr.w != nil {
		ctx = context.WithStderr(ctx, rc.Stderr.w)
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

		const GoshTestNameEnv = "GOSH_TEST_NAME"

		var env []string

		if testCtx != nil {
			// selfArgs = append(selfArgs, os.Args[1:]...)
			selfArgs = append(selfArgs, "-test.run=^"+testCtx.Name()+"$")

			env = append(env, GoshTestNameEnv+"="+testCtx.Name())
		} else {
			// os.Args can be something like the below when run via test
			//  /tmp/go-build2810781305/b001/arctest.test -test.testlogfile=/tmp/go-build2810781305/b001/testlog.txt -test.paniconexit0 -test.timeout=30s -test.run=^TestAcc$ ::: hello world
			//
			// It's especially important to set/inherit `-test.run`, but not `-test.paniconexit0`.
			// The former is required to correctly redirect the recursive command to the test function that invoked it.
			// The latter is required to not pollute the recursively invoked command's stdout/stderr with go test output.
			var testRun string
			for _, a := range os.Args[1:] {
				if strings.HasPrefix(a, "-test.run=") {
					testRun = a
					break
				}
			}

			// Needed to only trigger the target command when you run all the go tests
			if testRun != "" {
				selfArgs = append(selfArgs, testRun)

				env = append(env, GoshTestNameEnv+"="+strings.TrimRight(strings.TrimLeft(strings.TrimLeft(testRun, "-test.run="), "^"), "$"))
			} else if strings.HasSuffix(os.Args[0], ".test") {
				panic(fmt.Errorf("missing testing.T object in Run() args: %v", args))
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
			Env:        env,
		}
	})

	if initErr != nil {
		return initErr
	}

	if t.app == nil {
		return fmt.Errorf("[bug] app is not initialized")
	}

	if len(cmds) > 0 {
		return t.runPipeline(ctx, cmds)
	}

	rc.Env = append(rc.Env, t.app.Env...)

	return t.app.Run(ctx, args, rc)
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

func (sh *Shell) GoRun(ctx context.Context, vars ...interface{}) <-chan error {
	err := make(chan error)

	go func() {
		vars = append([]interface{}{ctx}, vars...)
		e := sh.Run(vars...)
		err <- e
	}()

	return err
}
