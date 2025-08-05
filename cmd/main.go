package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/lemmego/cli"
	"log"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
