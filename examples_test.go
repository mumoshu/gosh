package gosh_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/goshtest"
	"github.com/stretchr/testify/assert"
)

func TestAtoiBasic(t *testing.T) {
	sh := &gosh.Shell{}

	sh.Export("atoi", func(ctx gosh.Context, a string) (int, error) {
		v, err := strconv.Atoi(a)
		fmt.Fprintf(ctx.Stdout(), "%d\n", v)
		return v, err
	})

	goshtest.Run(t, sh, func() {
		t.Run("ok", func(t *testing.T) {
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
			var stdout bytes.Buffer

			err := sh.Run(t, "atoi", "aaa", gosh.WriteStdout(&stdout))

			assert.Equal(t, err.Error(), "strconv.Atoi: parsing \"aaa\": invalid syntax")
		})
	})
}

func TestAtoiFunc(t *testing.T) {
	sh := &gosh.Shell{}

	sh.Export(atoi)

	goshtest.Run(t, sh, func() {
		t.Run("ok", func(t *testing.T) {
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
			var stdout bytes.Buffer

			err := sh.Run(t, "atoi", "aaa", gosh.WriteStdout(&stdout))

			assert.Equal(t, err.Error(), "strconv.Atoi: parsing \"aaa\": invalid syntax")
		})
	})
}

func atoi(ctx gosh.Context, a string) (int, error) {
	v, err := strconv.Atoi(a)
	fmt.Fprintf(ctx.Stdout(), "%d\n", v)
	return v, err
}

func TestAtoiMethod(t *testing.T) {
	sh := &gosh.Shell{}

	conv := &Strconv{}

	sh.Export(conv.Atoi)

	goshtest.Run(t, sh, func() {
		t.Run("ok", func(t *testing.T) {
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
			var stdout bytes.Buffer

			err := sh.Run(t, "atoi", "aaa", gosh.WriteStdout(&stdout))

			assert.Equal(t, err.Error(), "strconv.Atoi: parsing \"aaa\": invalid syntax")
		})
	})
}

func TestAtoiStruct(t *testing.T) {
	sh := &gosh.Shell{}

	conv := &Strconv{}

	sh.Export(conv)

	goshtest.Run(t, sh, func() {
		t.Run("ok", func(t *testing.T) {
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
			var stdout bytes.Buffer

			err := sh.Run(t, "atoi", "aaa", gosh.WriteStdout(&stdout))

			assert.Equal(t, err.Error(), "strconv.Atoi: parsing \"aaa\": invalid syntax")
		})
	})
}

type Strconv struct {
}

func (v Strconv) Atoi(ctx gosh.Context, a string) (int, error) {
	return atoi(ctx, a)
}
