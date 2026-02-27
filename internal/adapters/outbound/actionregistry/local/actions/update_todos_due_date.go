package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
)

// BulkTodoDueDateUpdaterAction is an assistant action for updating due dates in bulk.
type BulkTodoDueDateUpdaterAction struct {
	uow          domain.UnitOfWork
	updater      usecases.TodoUpdater
	timeProvider domain.CurrentTimeProvider
}

// NewBulkTodoDueDateUpdaterAction creates a new instance of BulkTodoDueDateUpdaterAction.
func NewBulkTodoDueDateUpdaterAction(uow domain.UnitOfWork, updater usecases.TodoUpdater, timeProvider domain.CurrentTimeProvider) BulkTodoDueDateUpdaterAction {
	return BulkTodoDueDateUpdaterAction{
		uow:          uow,
		updater:      updater,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the action execution.
func (a BulkTodoDueDateUpdaterAction) StatusMessage() string {
	return "📅 Updating due dates..."
}

// Definition returns the assistant action definition for BulkTodoDueDateUpdaterAction.
func (a BulkTodoDueDateUpdaterAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "update_todos_due_date",
		Description: "Update due dates for multiple todos.",
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"todos": {
					Type:        "array",
					Description: "List of due date updates. Each item: {id,due_date}. REQUIRED.",
					Required:    true,
					Items: &domain.AssistantActionField{
						Type:        "object",
						Description: "Todo item to update due date.",
						Fields: map[string]domain.AssistantActionField{
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
	}
}

// Execute executes BulkTodoDueDateUpdaterAction.
func (a BulkTodoDueDateUpdaterAction) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	params := struct {
		Todos []struct {
			ID      string `json:"id"`
			DueDate string `json:"due_date"`
		} `json:"todos"`
	}{}
	exampleArgs := `{"todos":[{"id":"<uuid>","due_date":"2026-04-30"},{"id":"<uuid>","due_date":"2026-05-01"}]}`

	err := unmarshalActionInput(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("invalid_arguments", err.Error(), exampleArgs),
		}
	}
	if len(params.Todos) == 0 {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
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
			return domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_todo_id", fmt.Sprintf("todo at index %d has invalid id: %s", i, parseErr.Error()), exampleArgs),
			}
		}

		dueDate, found := extractDateParam(todo.DueDate, conversationHistory, now)
		if !found {
			return domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
				ActionCallID: &call.ID,
				Content:      newActionError("invalid_due_date", fmt.Sprintf("todo at index %d has invalid due_date.", i), exampleArgs),
			}
		}

		items = append(items, updateItem{
			ID:      todoID,
			DueDate: dueDate,
		})
	}

	todos := make([]domain.Todo, 0, len(items))
	err = a.uow.Execute(ctx, func(uowCtx context.Context, uow domain.UnitOfWork) error {
		for i, item := range items {
			todo, updateErr := a.updater.Update(uowCtx, uow, item.ID, nil, nil, &item.DueDate)
			if updateErr != nil {
				return fmt.Errorf("todo at index %d: %w", i, updateErr)
			}
			todos = append(todos, todo)
		}
		return nil
	})
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      newActionError("update_todos_due_date_error", err.Error(), exampleArgs),
		}
	}

	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      formatTodosRows(todos),
	}
}
