package main

import (
	"log"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/app"
)

func main() {
	err := app.NewTodoApp().Run()
	if err != nil {
		log.Fatalf("Failed to run the TodoApp: %v", err)
	}
}
