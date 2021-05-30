//+build project

package main

import (
	"os"

	. "github.com/mumoshu/gosh"
)

// dsl
var (
	sh       = &Shell{}
	Def      = sh.Def
	Run      = sh.Run
	MustExec = sh.MustExec
)

func main() {
	Def("all", Dep("build"), Dep("test"), func() {

	})

	Def("build", func() {
		Run("go", "build", "-o", "getting-started", "./examples/getting-started")
	})

	Def("test", func() {
		Run("go", "test", "./...")
	})

	MustExec(os.Args)
}
