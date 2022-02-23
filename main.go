package main

import (
	"fmt"
	"os"

	_ "github.com/joho/godotenv/autoload"
)

var TmpDir = "/tmp/shortique"

func mustGetEnv(key string) string {
	v := os.Getenv(key)

	if v == "" {
		panic(fmt.Errorf("%s env variable is required", key))
	}

	return v
}

func main() {
	if err := RunBot(mustGetEnv("TOKEN")); err != nil {
		panic(err)
	}
}
