package main

import (
	"os"

	"github.com/mumoshu/gosh/run/examples/ginkgotest"
)

func main() {
	ginkgotest.MustExec(os.Args)
}
