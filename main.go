package main

import (
	"log"

	"skill-eval-harness/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
