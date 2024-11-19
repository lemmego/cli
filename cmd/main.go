package main

import (
	"github.com/lemmego/cli"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("env file not loaded")
	}

	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
