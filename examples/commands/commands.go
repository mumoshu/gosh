package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mumoshu/gosh"
)

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	sh.Def("gogrep", func(ctx gosh.Context, pattern string) {
		scanner := bufio.NewScanner(ctx.Stdin())

		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, pattern) {
				fmt.Fprint(os.Stdout, line+"\n")
			}
		}
	})

	sh.Def("gocat", func(ctx gosh.Context, file ...string) error {
		var in io.Reader

		if len(file) == 1 {
			f, err := os.Open(file[0])
			if err != nil {
				return err
			}
			in = f
		} else if len(file) == 0 {
			in = ctx.Stdin()
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
