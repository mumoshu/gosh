//+build project

package main

import (
	"os"

	. "github.com/mumoshu/gosh"
)

// dsl
var (
	sh       = &Shell{}
	Task     = sh.Export
	Run      = sh.Run
	MustExec = sh.MustExec
)

func main() {
	Task("all", Dep("build"), Dep("test"), func() {

	})

	Task("build", func() {
		var examples = []string{
			"arctest",
			"commands",
			"getting-started",
			"ginkgotest",
			"gotest",
			"pipeline",
		}

		const dir = "examples"

		for _, name := range examples {
			Run("go", "build", "-o", "bin/"+name, "./"+name, Dir(dir))
		}
	})

	Task("test", func() {
		Run("go", "test", "./...")
		Run("go", "test", "./...", Dir("examples"))
	})

	MustExec(os.Args)
}
