package gotest

import "github.com/mumoshu/gosh"

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	sh.Def("hello", func(ctx gosh.Context, target string) {
		ctx.Stdout().Write([]byte("hello " + target + "\n"))
	})

	return sh
}

func MustExec(osArgs []string) {
	New().MustExec(osArgs)
}
