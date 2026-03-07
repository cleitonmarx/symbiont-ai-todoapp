package main

import (
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/app"
)

func main() {
	err := app.NewGraphQLAPI().Run()
	if err != nil {
		log.Fatalf("Failed to run the GraphQL API: %v", err)
	}
}
