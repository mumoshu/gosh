package gosh_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/goshtest"
	"github.com/stretchr/testify/assert"
)

func TestArgs(t *testing.T) {
	sh := &gosh.Shell{}

	sh.Export("add", func(ctx gosh.Context, a, b int) {
		fmt.Fprintf(ctx.Stdout(), "%d\n", a+b)
	})

	sh.Export("join1", func(ctx gosh.Context, delim string, elems []string) {
		v := strings.Join(elems, delim)
		fmt.Fprintf(ctx.Stdout(), "%s\n", v)
	})

	sh.Export("join2", func(ctx gosh.Context, delim string, elems ...string) {
		v := strings.Join(elems, delim)
		fmt.Fprintf(ctx.Stdout(), "%s\n", v)
	})

	goshtest.Run(t, sh, func() {
		t.Run("add", func(t *testing.T) {
			var stdout bytes.Buffer

			err := sh.Run(t, "add", 1, 2, gosh.WriteStdout(&stdout))

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, "3\n", stdout.String())
		})

		t.Run("join1", func(t *testing.T) {
			var stdout bytes.Buffer

			err := sh.Run(t, "join1", ",", "A", "B", gosh.WriteStdout(&stdout))

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, "A,B\n", stdout.String())
		})

		t.Run("join2", func(t *testing.T) {
			var stdout bytes.Buffer

			err := sh.Run(t, "join2", ",", "A", "B", gosh.WriteStdout(&stdout))

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, "A,B\n", stdout.String())
		})
	})
}
