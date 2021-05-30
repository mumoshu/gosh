package goshtest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/mumoshu/gosh"
)

var TestEnvName = "FOO"

func Run(testCtx *testing.T, t *gosh.Shell, f func()) {
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
		if err := t.Run(append(runArgs, gosh.WriteStdout(&stdout), gosh.WriteStderr(&stderr))...); err != nil {
			testCtx.Error(err)
		}

		fmt.Fprint(origStderr, stderr.String())
		fmt.Fprint(origStdout, stdout.String())

		return
	}

	os.Setenv(TestEnvName, "foobar")

	f()
}
