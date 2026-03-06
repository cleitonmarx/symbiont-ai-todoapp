package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	todouc "github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/todo"
	"github.com/google/uuid"
)

// UpdateTodosDueDateAction is an assistant action for updating due dates in bulk.
type UpdateTodosDueDateAction struct {
	uow          transaction.UnitOfWork
	updater      todouc.Updater
	timeProvider core.CurrentTimeProvider
}

// NewUpdateTodosDueDateAction creates a new instance of UpdateTodosDueDateAction.
func NewUpdateTodosDueDateAction(uow transaction.UnitOfWork, updater todouc.Updater, timeProvider core.CurrentTimeProvider) UpdateTodosDueDateAction {
	return UpdateTodosDueDateAction{
		uow:          uow,
		updater:      updater,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the action execution.
func (a UpdateTodosDueDateAction) StatusMessage() string {
	return "📅 Updating due dates..."
}

// Renderer returns the deterministic result renderer for due date updates.
func (a UpdateTodosDueDateAction) Renderer() (assistant.ActionResultRenderer, bool) {
	return updateTodosDueDateRenderer{}, true
}

// Definition returns the assistant action definition for UpdateTodosDueDateAction.
func (a UpdateTodosDueDateAction) Definition() assistant.ActionDefinition {
	return assistant.ActionDefinition{
		Name:        "update_todos_due_date",
		Description: "Update due dates for multiple todos.",
		Input: assistant.ActionInput{
			Type: "object",
			Fields: map[string]assistant.ActionField{
				"todos": {
					Type:        "array",
					Description: "List of due date updates. Each item: {id,due_date}. REQUIRED.",
					Required:    true,
					Items: &assistant.ActionField{
						Type:        "object",
						Description: "Todo item to update due date.",
						Fields: map[string]assistant.ActionField{
							"id": {
								Type:        "string",
								Description: "ID of the todo to update. REQUIRED.",
								Required:    true,
							},
							"due_date": {
								Type:        "string",
								Description: "New due date for the todo in YYYY-MM-DD format. REQUIRED.",
								Required:    true,
								Format:      "date",
							},
						},
					},
				},
			},
		},
		Approval: assistant.ActionApproval{
			Required:    true,
			Title:       "Confirm update of todo due dates",
			Description: "Updating due dates will modify existing todos. Please confirm.",
			PreviewFields: []string{
				"todos[].id",
				"todos[].due_date",
			},
			Timeout: 2 * time.Minute,
		},
	}
}

// Execute executes UpdateTodosDueDateAction.
func (a UpdateTodosDueDateAction) Execute(ctx context.Context, call assistant.ActionCall, conversationHistory []assistant.Message) assistant.Message {
	params := struct {
		Todos []struct {
			ID      string `json:"id"`
			DueDate string `json:"due_date"`
		} `json:"todos"`
	}{}
	exampleArgs := `{"todos":[{"id":"<uuid>","due_date":"2026-04-30"},{"id":"<uuid>","due_date":"2026-05-01"}]}`

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
	type updateItem struct {
		ID      uuid.UUID
		DueDate time.Time
	}
	items := make([]updateItem, 0, len(params.Todos))
	for i, todo := range params.Todos {
		todoID, parseErr := uuid.Parse(todo.ID)
		if parseErr != nil {
			return assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_todo_id", fmt.Sprintf("todo at index %d has invalid id: %s", i, parseErr.Error()), exampleArgs),
			}
		}

		dueDate, found := extractDateParam(todo.DueDate, conversationHistory, now)
		if !found {
			return assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_due_date", fmt.Sprintf("todo at index %d has invalid due_date.", i), exampleArgs),
			}
		}

		items = append(items, updateItem{
			ID:      todoID,
			DueDate: dueDate,
		})
	}

	todos := make([]todo.Todo, 0, len(items))
	err = a.uow.Execute(ctx, func(uowCtx context.Context, scope transaction.Scope) error {
		for i, item := range items {
			todo, updateErr := a.updater.Update(uowCtx, scope, item.ID, nil, nil, &item.DueDate)
			if updateErr != nil {
				return fmt.Errorf("todo at index %d: %w", i, updateErr)
			}
			todos = append(todos, todo)
		}
		return nil
	})
	if err != nil {
		return assistant.Message{
			Role:         assistant.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("update_todos_due_date_error", err.Error(), exampleArgs),
		}
	}

	return assistant.Message{
		Role:         assistant.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      formatTodosRows(todos),
	}
}
