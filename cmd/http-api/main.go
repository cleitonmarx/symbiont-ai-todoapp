package main

import (
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/app"
)

func main() {
	err := app.NewHTTPAPI().Run()
	if err != nil {
		log.Fatalf("Failed to run the HTTP API: %v", err)
	}
}
