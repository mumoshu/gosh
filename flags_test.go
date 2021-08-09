package gosh_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/context"
	"github.com/mumoshu/gosh/goshtest"
	"github.com/stretchr/testify/assert"
)

func TestFlags(t *testing.T) {
	sh := &gosh.Shell{}

	type Opts struct {
		UpperCase bool `flag:"upper-case"`
	}

	sh.Export("hello", func(ctx context.Context, a string, opts Opts) {
		a = "hello " + a
		if opts.UpperCase {
			a = strings.ToUpper(a)
		}
		fmt.Fprintf(context.Stdout(ctx), "%s\n", a)
	})

	goshtest.Run(t, sh, func() {
		t.Run("direct", func(t *testing.T) {
			fmt.Fprintf(os.Stderr, "%v\n", os.Args)
			var stdout bytes.Buffer

			err := sh.Run(t, "hello", "world", "-upper-case", gosh.WriteStdout(&stdout))

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, "HELLO WORLD\n", stdout.String())
		})

		t.Run("flags", func(t *testing.T) {
			fmt.Fprintf(os.Stderr, "%v\n", os.Args)
			var stdout bytes.Buffer

			err := sh.Run(t, "hello", "world", Opts{UpperCase: true}, gosh.WriteStdout(&stdout))

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, "HELLO WORLD\n", stdout.String())
		})
	})
}
