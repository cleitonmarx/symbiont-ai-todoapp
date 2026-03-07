package main

import (
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/app"
)

func main() {
	err := app.NewMessageRelay().Run()
	if err != nil {
		log.Fatalf("Failed to run the Message Relay: %v", err)
	}
}
