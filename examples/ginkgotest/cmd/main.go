package main

import (
	"os"

	"github.com/mumoshu/gosh/examples/ginkgotest"
)

func main() {
	ginkgotest.MustExec(os.Args)
}
