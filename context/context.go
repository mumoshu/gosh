package context

import (
	"context"
	gocontext "context"
	"io"
	"os"
)

type Context = gocontext.Context

var TODO = gocontext.TODO
var Background = gocontext.Background

type stdinKey struct{}
type stdoutKey struct{}
type stderrKey struct{}
type errorKey struct{}

func WithStdin(ctx gocontext.Context, in io.Reader) Context {
	return context.WithValue(ctx, stdinKey{}, in)
}

func Stdin(ctx gocontext.Context) io.Reader {
	v := ctx.Value(stdinKey{})
	if v == nil {
		return os.Stdin
	}

	return v.(io.Reader)
}

func WithStdout(ctx gocontext.Context, out io.Writer) Context {
	return context.WithValue(ctx, stdoutKey{}, out)
}

func Stdout(ctx gocontext.Context) io.Writer {
	v := ctx.Value(stdoutKey{})
	if v == nil {
		return os.Stdout
	}

	return v.(io.Writer)
}

func WithStderr(ctx gocontext.Context, out io.Writer) Context {
	return context.WithValue(ctx, stderrKey{}, out)
}

func Stderr(ctx gocontext.Context) io.Writer {
	v := ctx.Value(stderrKey{})
	if v == nil {
		return os.Stderr
	}

	return v.(io.Writer)
}

func WithError(ctx gocontext.Context, err error) Context {
	return context.WithValue(ctx, errorKey{}, err)
}

func Error(ctx gocontext.Context) error {
	v := ctx.Value(errorKey{})
	if v == nil {
		return nil
	}

	return v.(error)
}
