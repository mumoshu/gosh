package gosh_test

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/goshtest"
	"github.com/stretchr/testify/assert"
)

func TestAtoi(t *testing.T) {
	sh := &gosh.Shell{}

	sh.Export("atoi", func(ctx gosh.Context, a string) (int, error) {
		v, err := strconv.Atoi(a)
		fmt.Fprintf(ctx.Stdout(), "%d\n", v)
		return v, err
	})

	goshtest.Run(t, sh, func() {
		t.Run("ok", func(t *testing.T) {
			fmt.Fprintf(os.Stderr, "%v\n", os.Args)
			var stdout bytes.Buffer

			var i int

			err := sh.Run(t, "atoi", "123", gosh.Out(&i), gosh.WriteStdout(&stdout))

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, "123\n", stdout.String())
			assert.Equal(t, i, 123)
		})

		t.Run("err", func(t *testing.T) {
			fmt.Fprintf(os.Stderr, "%v\n", os.Args)
			var stdout bytes.Buffer

			err := sh.Run(t, "atoi", "aaa", gosh.WriteStdout(&stdout))

			assert.Equal(t, err.Error(), "strconv.Atoi: parsing \"aaa\": invalid syntax")
		})
	})
}
