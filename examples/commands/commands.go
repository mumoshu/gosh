package commands

import (
	"bufio"
	"fmt"
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

	sh.Def("gocat", func(ctx gosh.Context, file string) error {
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprint(os.Stdout, line+"\n")
		}

		return nil
	})

	return sh
}
