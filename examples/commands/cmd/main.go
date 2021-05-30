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
	if err := commands.New().Run(args...); err != nil {
		log.Fatal(err)
	}
}
