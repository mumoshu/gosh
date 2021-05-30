package main

import (
	"os"

	"github.com/mumoshu/gosh/run/examples/gotest"
)

func main() {
	gotest.MustExec(os.Args)
}
