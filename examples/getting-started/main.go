package main

import (
	"os"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/context"
)

func main() {
	sh := &gosh.Shell{}

	sh.Export("hello", func(ctx context.Context, target string) {
		// sh.Diagf("My own debug message someData=%s someNumber=%d", "foobar", 123)

		context.Stdout(ctx).Write([]byte("hello " + target + "\n"))
	})

	sh.MustExec(os.Args)
}
