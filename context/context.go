package context

import (
	"context"
	"io"
	"os"
)

type Context = context.Context

var TODO = context.TODO
var Background = context.Background

type stdinKey struct{}
type stdoutKey struct{}
type stderrKey struct{}
type errorKey struct{}
type varsKey struct{}

func WithStdin(ctx context.Context, in io.Reader) Context {
	return context.WithValue(ctx, stdinKey{}, in)
}

func Stdin(ctx context.Context) io.Reader {
	v := ctx.Value(stdinKey{})
	if v == nil {
		return os.Stdin
	}

	return v.(io.Reader)
}

func WithStdout(ctx context.Context, out io.Writer) Context {
	return context.WithValue(ctx, stdoutKey{}, out)
}

func Stdout(ctx context.Context) io.Writer {
	v := ctx.Value(stdoutKey{})
	if v == nil {
		return os.Stdout
	}

	return v.(io.Writer)
}

func WithStderr(ctx context.Context, out io.Writer) Context {
	return context.WithValue(ctx, stderrKey{}, out)
}

func Stderr(ctx context.Context) io.Writer {
	v := ctx.Value(stderrKey{})
	if v == nil {
		return os.Stderr
	}

	return v.(io.Writer)
}

func WithError(ctx context.Context, err error) Context {
	return context.WithValue(ctx, errorKey{}, err)
}

func Error(ctx context.Context) error {
	v := ctx.Value(errorKey{})
	if v == nil {
		return nil
	}

	return v.(error)
}

func WithVariables(ctx context.Context, vars map[string]interface{}) Context {
	return context.WithValue(ctx, varsKey{}, &Variables{vars: vars})
}

func Get(ctx context.Context, key string) interface{} {
	vars := getVars(ctx)
	if vars == nil {
		return nil
	}

	return vars.Get(key)
}

func Set(ctx context.Context, key string, value interface{}) {
	vars := getVars(ctx)
	if vars == nil {
		return
	}

	vars.Set(key, value)
}

func getVars(ctx context.Context) *Variables {
	v := ctx.Value(varsKey{})
	if v == nil {
		return nil
	}

	return v.(*Variables)
}
