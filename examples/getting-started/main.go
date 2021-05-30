package main

import (
	"os"

	"github.com/mumoshu/gosh"
)

func main() {
	sh := &gosh.Shell{}

	sh.Export("hello", func(ctx gosh.Context, target string) {
		// sh.Diagf("My own debug message someData=%s someNumber=%d", "foobar", 123)

		ctx.Stdout().Write([]byte("hello " + target + "\n"))
	})

	sh.MustExec(os.Args)
}
