package main

import (
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/app"
)

func main() {
	err := app.NewChatSummaryGenerator().Run()
	if err != nil {
		log.Fatalf("Failed to run the Chat Summary Generator: %v", err)
	}
}
