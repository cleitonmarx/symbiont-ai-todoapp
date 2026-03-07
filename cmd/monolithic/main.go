package main

import (
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/app"
)

func main() {
	err := app.NewMonolithic().Run()
	if err != nil {
		log.Fatalf("Failed to run the TodoApp: %v", err)
	}
}
