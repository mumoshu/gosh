package gotest

import (
	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/context"
)

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	sh.Export("hello", func(ctx context.Context, target string) {
		context.Stdout(ctx).Write([]byte("hello " + target + "\n"))
	})

	return sh
}

func MustExec(osArgs []string) {
	New().MustExec(osArgs)
}
