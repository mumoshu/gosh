//+build project

package main

import (
	"os"

	. "github.com/mumoshu/gosh"
)

// dsl
var (
	sh       = &Shell{}
	Export   = sh.Export
	Run      = sh.Run
	MustExec = sh.MustExec
)

func main() {
	Export("all", Dep("build"), Dep("test"), func() {

	})

	Export("build", func() {
		Run("go", "build", "-o", "getting-started", "./examples/getting-started")
	})

	Export("test", func() {
		Run("go", "test", "./...")
	})

	MustExec(os.Args)
}
