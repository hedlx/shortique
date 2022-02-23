package main

import (
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"
)

var TmpDir = "/tmp/shortique"

func mustGetEnv(key string) string {
	v := os.Getenv(key)

	if v == "" {
		log.Fatal(key, " env variable is required")
	}

	return v
}

func main() {
	token := mustGetEnv("TOKEN")
	if err := Setup(); err != nil {
		log.Fatal(err.Error())
	}

	if err := RunBot(token); err != nil {
		log.Fatal(err.Error())
	}
}
