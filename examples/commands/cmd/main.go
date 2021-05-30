package main

import (
	"log"
	"os"

	"github.com/mumoshu/gosh/examples/commands"
)

func main() {
	var args []interface{}
	for _, a := range os.Args[1:] {
		args = append(args, a)
	}
	log.Fatal(commands.New().Run(args...))
}
