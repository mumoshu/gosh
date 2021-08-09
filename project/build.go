//+build project

package main

import (
	"context"
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

func init() {
	Task("all", Dep("build"), Dep("test"), func() {

	})

	Task("build", func(ctx context.Context) {
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
			Run(ctx, "go", "build", "-o", "bin/"+name, "./"+name, Dir(dir))
		}
	})

	Task("test", func() {
		Run("go", "test", "./...")
		Run("go", "test", "./...", Dir("examples"))
	})
}

func main() {
	MustExec(os.Args)
}
