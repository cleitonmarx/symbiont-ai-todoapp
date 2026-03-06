package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
)

// CreateTodosAction is an assistant action for creating multiple todos.
type CreateTodosAction struct {
	uow          transaction.UnitOfWork
	creator      todouc.Creator
	timeProvider core.CurrentTimeProvider
}

// NewCreateTodosAction creates a new instance of CreateTodosAction.
func NewCreateTodosAction(uow transaction.UnitOfWork, creator todouc.Creator, timeProvider core.CurrentTimeProvider) CreateTodosAction {
	return CreateTodosAction{
		uow:          uow,
		creator:      creator,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the action execution.
func (a CreateTodosAction) StatusMessage() string {
	return "📝 Creating your todos..."
}

// Renderer returns the deterministic result renderer for created todos.
func (a CreateTodosAction) Renderer() (assistant.ActionResultRenderer, bool) {
	return createTodosRenderer{}, true
}

// Definition returns the assistant action definition for CreateTodosAction.
func (a CreateTodosAction) Definition() assistant.ActionDefinition {
	return assistant.ActionDefinition{
		Name:        "create_todos",
		Description: "Create multiple todo items in one call (batch).",
		Input: assistant.ActionInput{
			Type: "object",
			Fields: map[string]assistant.ActionField{
				"todos": {
					Type:        "array",
					Description: "List of todos to create. Each item: {title, due_date}. REQUIRED.",
					Required:    true,
					Items: &assistant.ActionField{
						Type:        "object",
						Description: "Todo item to create.",
						Fields: map[string]assistant.ActionField{
							"title": {
								Type:        "string",
								Description: "Title of the todo. REQUIRED.",
								Required:    true,
							},
							"due_date": {
								Type:        "string",
								Description: "Due date in YYYY-MM-DD format. REQUIRED.",
								Required:    true,
								Format:      "date",
							},
						},
					},
				},
			},
		},
	}
}

// Execute executes CreateTodosAction.
func (a CreateTodosAction) Execute(ctx context.Context, call assistant.ActionCall, conversationHistory []assistant.Message) assistant.Message {
	params := struct {
		Todos []struct {
			Title   string `json:"title"`
			DueDate string `json:"due_date"`
		} `json:"todos"`
	}{}
	exampleArgs := `{"todos":[{"title":"Pay rent","due_date":"2026-04-30"},{"title":"Buy groceries","due_date":"2026-05-01"}]}`

	err := unmarshalActionInput(call.Input, &params)
	if err != nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", err.Error(), exampleArgs),
		}
	}
	if len(params.Todos) == 0 {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", "todos must not be empty.", exampleArgs),
		}
	}

	now := a.timeProvider.Now()
	type createItem struct {
		Title   string
		DueDate time.Time
	}
	items := make([]createItem, 0, len(params.Todos))
	for i, td := range params.Todos {
		title := strings.TrimSpace(td.Title)
		if title == "" {
			return assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_title", fmt.Sprintf("todo at index %d has an empty title.", i), exampleArgs),
			}
		}

		dueDate, found := extractDateParam(td.DueDate, conversationHistory, now)
		if !found {
			return assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_due_date", fmt.Sprintf("todo at index %d has invalid due_date.", i), exampleArgs),
			}
		}

		items = append(items, createItem{Title: title, DueDate: dueDate})
	}

	todos := make([]todo.Todo, 0, len(items))
	err = a.uow.Execute(ctx, func(uowCtx context.Context, scope transaction.Scope) error {
		for i, item := range items {
			todo, createErr := a.creator.Create(uowCtx, scope, item.Title, item.DueDate)
			if createErr != nil {
				return fmt.Errorf("todo at index %d: %w", i, createErr)
			}
			todos = append(todos, todo)
		}
		return nil
	})
	if err != nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("create_todos_error", err.Error(), exampleArgs),
		}
	}

	return assistant.Message{
		Role:         assistant.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      formatTodosRows(todos),
	}
}
