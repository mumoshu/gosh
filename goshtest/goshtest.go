package goshtest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/mumoshu/gosh"
)

var TestEnvName = "GOSH_TEST_NAME"

func Run(testCtx *testing.T, t *gosh.Shell, f func()) {
	testCtx.Helper()

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

		if len(runArgs) == 0 {
			testCtx.Error("runArgs is zero length. This means that your test target script called the test binary without any args, which shoudln't happen.")
		}

		fmt.Fprintf(os.Stderr, "ARGS=%v\n", runArgs)
		err = t.Run(append(runArgs, gosh.WriteStdout(&stdout), gosh.WriteStderr(&stderr))...)
		if err != nil {
			testCtx.Error(fmt.Errorf("failed running %s: %v", strings.Join(osArgs, " "), err))
		}

		fmt.Fprint(origStderr, stderr.String())
		fmt.Fprint(origStdout, stdout.String())

		// This requires that we omit `-test.paniconexit0` on recursively running gosh-provided command.
		if err != nil {
			os.Exit(1)
		}

		os.Exit(0)

		return
	}

	f()
}

// Cleanup is similar to t.Cleanup(), but it only runs the cleanup function only when the whole test has succeded.
// If the test as a whole failed, or go-test binary was instructed to run a specific subtest, the cleanup function isn't called,
// so that you can iterate quicker.
func Cleanup(t *testing.T, f func()) {
	t.Helper()

	t.Cleanup(func() {
		var run string
		for i := range os.Args {
			// `go test -run $RUN` results in `/tmp/path/to/some.test -test.run $RUN` being run,
			// and hence we check for -test.run
			if os.Args[i] == "-test.run" {
				runIdx := i + 1
				run = os.Args[runIdx]
				break
			} else if strings.HasPrefix(os.Args[i], "-test.run=") {
				split := strings.Split(os.Args[i], "-test.run=")
				run = split[1]
			}
		}

		if t.Failed() {
			return
		}

		// Do not delete the cluster so that we can accelerate interation on tests
		if run == "" || (run != "" && run != "^"+t.Name()+"$" && !strings.HasPrefix(run, "^"+t.Name()+"/")) {
			// This should be printed to the debug console for visibility
			t.Logf("Skipped stopping due to run being %q", run)
			return
		}

		f()
	})

}
