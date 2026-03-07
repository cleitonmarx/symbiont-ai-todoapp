package main

import (
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/app"
)

func main() {
	err := app.NewBoardSummaryGenerator().Run()
	if err != nil {
		log.Fatalf("Failed to run the Board Summary Generator: %v", err)
	}
}
