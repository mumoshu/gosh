package main

import (
	"log"
	"os"

	"github.com/mumoshu/gosh/examples/arctest"
)

func main() {
	var args []interface{}
	for _, a := range os.Args[1:] {
		args = append(args, a)
	}
	log.Fatal(arctest.New().Run(args...))
}
