package main

import (
	"log"

	"github.com/yourorg/github-code-agent/pkg/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
