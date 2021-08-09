package pipeline

import (
	"fmt"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/context"
)

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	sh.Export("ctx3", func(ctx context.Context) error {
		b, lsErr := sh.Pipe(ctx, "echo", "footest")

		grepErr := sh.GoRun(b, "grep", "test", gosh.WriteStdout(context.Stdout(ctx)))

		var count int
		for {
			fmt.Fprintf(context.Stderr(ctx), "x count=%d\n", count)
			select {
			case err := <-lsErr:
				if err != nil {
					fmt.Fprintf(context.Stderr(ctx), "lserr %v\n", err)
					return err
				}
				fmt.Fprintf(context.Stderr(ctx), "ls completed\n")

				count++
			case err := <-grepErr:
				if err != nil {
					fmt.Fprintf(context.Stderr(ctx), "greperr\n")
					return err
				}
				fmt.Fprintf(context.Stderr(ctx), "grep completed.\n")
				count++
			}
			fmt.Fprintf(context.Stderr(ctx), "selected count=%d\n", count)
			if count == 2 {
				break
			}
		}

		fmt.Fprintf(context.Stderr(ctx), "exiting\n")

		return nil
	})

	return sh
}
