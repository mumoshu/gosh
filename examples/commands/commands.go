package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/context"
)

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	sh.Export("gogrep", func(ctx context.Context, pattern string) {
		scanner := bufio.NewScanner(context.Stdin(ctx))

		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, pattern) {
				fmt.Fprint(os.Stdout, line+"\n")
			}
		}
	})

	sh.Export("gocat", func(ctx context.Context, file ...string) error {
		var in io.Reader

		if len(file) == 1 {
			f, err := os.Open(file[0])
			if err != nil {
				return err
			}
			in = f
		} else if len(file) == 0 {
			in = context.Stdin(ctx)
		} else {
			return fmt.Errorf("unexpected length of args %d: %v", len(file), file)
		}

		scanner := bufio.NewScanner(in)

		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprint(os.Stdout, line+"\n")
		}

		return nil
	})

	return sh
}
