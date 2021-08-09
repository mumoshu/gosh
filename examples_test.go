package gosh_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/context"
	"github.com/mumoshu/gosh/goshtest"
	"github.com/stretchr/testify/assert"
)

func TestAtoiBasic(t *testing.T) {
	sh := &gosh.Shell{}

	sh.Export("atoi", func(ctx context.Context, a string) (int, error) {
		v, err := strconv.Atoi(a)
		fmt.Fprintf(context.Stdout(ctx), "%d\n", v)
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

func atoi(ctx context.Context, a string) (int, error) {
	v, err := strconv.Atoi(a)
	fmt.Fprintf(context.Stdout(ctx), "%d\n", v)
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

func (v Strconv) Atoi(ctx context.Context, a string) (int, error) {
	return atoi(ctx, a)
}

func TestInputRedirectionFromFile(t *testing.T) {
	sh := &gosh.Shell{}

	sh.Export("run", func(ctx context.Context) error {
		data, err := io.ReadAll(context.Stdin(ctx))
		if err != nil {
			return err
		}

		fmt.Fprintf(context.Stdout(ctx), "%s", string(data))
		return err
	})

	goshtest.Run(t, sh, func() {
		t.Run("ok", func(t *testing.T) {
			var stdout bytes.Buffer

			f, err := ioutil.TempFile(t.TempDir(), "input")
			assert.NoError(t, err)

			f.Close()

			err = os.WriteFile(f.Name(), []byte("hello\n"), 0644)
			assert.NoError(t, err)

			f, err = os.Open(f.Name())
			assert.NoError(t, err)

			err = sh.Run(t, context.WithStdin(context.Background(), f), "run", gosh.WriteStdout(&stdout))
			defer f.Close()

			assert.NoError(t, err)

			assert.Equal(t, "hello\n", stdout.String())
		})
	})
}

func TestStdoutRedirectionToFile(t *testing.T) {
	sh := &gosh.Shell{}

	sh.Export("run", func(ctx context.Context) error {
		data, err := io.ReadAll(context.Stdin(ctx))
		if err != nil {
			return err
		}

		fmt.Fprintf(context.Stdout(ctx), "%s", string(data))
		return err
	})

	goshtest.Run(t, sh, func() {
		t.Run("ok", func(t *testing.T) {
			f, err := ioutil.TempFile(t.TempDir(), "output")
			assert.NoError(t, err)
			f.Close()

			f, err = os.Create(f.Name())
			assert.NoError(t, err)

			err = sh.Run(t, context.WithStdin(context.Background(), bytes.NewBufferString("hello\n")), "run", gosh.WriteStdout(f))
			f.Close()

			assert.NoError(t, err)

			f, err = os.Open(f.Name())
			assert.NoError(t, err)

			data, err := io.ReadAll(f)
			assert.NoError(t, err)

			assert.Equal(t, "hello\n", string(data))
		})
	})
}

func TestStderrRedirectionToFile(t *testing.T) {
	sh := &gosh.Shell{}

	sh.Export("run", func(ctx context.Context) error {
		data, err := io.ReadAll(context.Stdin(ctx))
		if err != nil {
			return err
		}

		fmt.Fprintf(context.Stderr(ctx), "%s", string(data))
		return err
	})

	goshtest.Run(t, sh, func() {
		t.Run("ok", func(t *testing.T) {
			f, err := ioutil.TempFile(t.TempDir(), "output")
			assert.NoError(t, err)
			f.Close()

			f, err = os.Create(f.Name())
			assert.NoError(t, err)

			err = sh.Run(t, context.WithStdin(context.Background(), bytes.NewBufferString("hello\n")), "run", gosh.WriteStderr(f))
			f.Close()

			assert.NoError(t, err)

			f, err = os.Open(f.Name())
			assert.NoError(t, err)

			data, err := io.ReadAll(f)
			assert.NoError(t, err)

			assert.Equal(t, "hello\n", string(data))
		})
	})
}
