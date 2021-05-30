package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mumoshu/gosh"
	. "github.com/mumoshu/gosh"
)

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	sh.Export("setup1", func(ctx gosh.Context, s []string) {
		fmt.Fprintf(ctx.Stdout(), "running setup1\n")
	})

	sh.Export("setup2", func(ctx gosh.Context, s []string) {
		ctx.Set("dir", s[0])
	})

	sh.Export("foo", func(ctx gosh.Context, s []string) {
		dir := ctx.Get("dir").(string)

		fmt.Fprintf(ctx.Stdout(), "dir="+dir+"\n")
		fmt.Fprintf(ctx.Stdout(), strings.Join(s, " ")+"\n")
		fmt.Fprintf(ctx.Stdout(), strings.Join(s, " ")+"\n")
		// fmt.Fprintf(os.Stdout, strings.Join(s, " ")+"\n")
		// fmt.Fprintf(os.Stdout, strings.Join(s, " ")+"\n")
		// fmt.Fprintf(os.Stdout, strings.Join(s, " "))
	}, Dep("setup1"), Dep("setup2", "aa"))

	sh.Export("hello", func(sub string) {
		println("hello " + sub)
	})

	sh.Export("ctx1", func(ctx gosh.Context, num int, b bool, args []string) {
		ctx.Stdout().Write([]byte(fmt.Sprintf("num=%v, b=%v, args=%v\n", num, b, args)))
	})

	sh.Export("ctx2", func(ctx gosh.Context, num int, b bool, args ...string) {
		ctx.Stdout().Write([]byte(fmt.Sprintf("num=%v, b=%v, args=%v\n", num, b, args)))

		sh.Run(ctx, "hello", "world")
		sh.Run(ctx, "ls", "-lah")
	})

	sh.Export("ctx3", func(ctx gosh.Context) error {
		b, lsErr := sh.GoPipe(ctx, "ls", "-lah")

		grepErr := sh.GoRun(b, "grep", "test")

		var count int
		for {
			fmt.Fprintf(os.Stderr, "x count=%d\n", count)
			select {
			case err := <-lsErr:
				if err != nil {
					fmt.Fprintf(os.Stderr, "lserr %v\n", err)
					return err
				}
				fmt.Fprintf(os.Stderr, "ls\n")

				count++
			case err := <-grepErr:
				if err != nil {
					fmt.Fprintf(os.Stderr, "greperr\n")
					return err
				}
				fmt.Fprintf(os.Stderr, "grep\n")
				count++
			}
			fmt.Fprintf(os.Stderr, "selected count=%d\n", count)
			if count == 2 {
				break
			}
		}

		fmt.Fprintf(os.Stderr, "exiting\n")

		return fmt.Errorf("some error")
	})

	sh.Export("ctx4", func(ctx gosh.Context) error {
		return sh.Run(ctx, Cmd("ls", "-lah"), Cmd("grep", "test"))
	})

	sh.Export("ctx5", func(ctx gosh.Context) error {
		return sh.Run(ctx, Cmd("bash -c 'ls -lah | grep test'"))
	})

	return sh
}

func main() {
	println(fmt.Sprintf("starting abc=%v", os.Args))
	if err := New().Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
