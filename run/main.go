package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mumoshu/gosh"
	. "github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/context"
)

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	sh.Export("setup1", func(ctx context.Context, s []string) {
		fmt.Fprintf(context.Stdout(ctx), "running setup1\n")
	})

	sh.Export("setup2", func(ctx context.Context, s []string) {
		context.Set(ctx, "dir", s[0])
	})

	sh.Export("foo", func(ctx context.Context, s []string) {
		dir := context.Get(ctx, "dir").(string)

		fmt.Fprintf(context.Stdout(ctx), "dir="+dir+"\n")
		fmt.Fprintf(context.Stdout(ctx), strings.Join(s, " ")+"\n")
		fmt.Fprintf(context.Stdout(ctx), strings.Join(s, " ")+"\n")
		// fmt.Fprintf(os.Stdout, strings.Join(s, " ")+"\n")
		// fmt.Fprintf(os.Stdout, strings.Join(s, " ")+"\n")
		// fmt.Fprintf(os.Stdout, strings.Join(s, " "))
	}, Dep("setup1"), Dep("setup2", "aa"))

	sh.Export("hello", func(sub string) {
		println("hello " + sub)
	})

	sh.Export("ctx1", func(ctx context.Context, num int, b bool, args []string) {
		context.Stdout(ctx).Write([]byte(fmt.Sprintf("num=%v, b=%v, args=%v\n", num, b, args)))
	})

	sh.Export("ctx2", func(ctx context.Context, num int, b bool, args ...string) {
		context.Stdout(ctx).Write([]byte(fmt.Sprintf("num=%v, b=%v, args=%v\n", num, b, args)))

		sh.Run(ctx, "hello", "world")
		sh.Run(ctx, "ls", "-lah")
	})

	sh.Export("ctx3", func(ctx context.Context) error {
		b, lsErr := sh.Pipe(ctx, "ls", "-lah")

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

	sh.Export("ctx4", func(ctx context.Context) error {
		return sh.Run(ctx, Cmd("ls", "-lah"), Cmd("grep", "test"))
	})

	sh.Export("ctx5", func(ctx context.Context) error {
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
