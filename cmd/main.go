package main

import (
	"github.com/lemmego/cli"
	"log"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
