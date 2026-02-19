package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases"
	"github.com/google/uuid"
)

type TodoDueDateUpdaterAction struct {
	uow          domain.UnitOfWork
	updater      usecases.TodoUpdater
	timeProvider domain.CurrentTimeProvider
}

// NewTodoDueDateUpdaterAction creates a new instance of TodoDueDateUpdaterAction.
func NewTodoDueDateUpdaterAction(uow domain.UnitOfWork, updater usecases.TodoUpdater, timeProvider domain.CurrentTimeProvider) TodoDueDateUpdaterAction {
	return TodoDueDateUpdaterAction{
		uow:          uow,
		updater:      updater,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the action execution.
func (t TodoDueDateUpdaterAction) StatusMessage() string {
	return "📅 Updating the due date..."
}

// Definition returns the assistant action definition for TodoDueDateUpdaterAction.
func (tdut TodoDueDateUpdaterAction) Definition() domain.AssistantActionDefinition {
	return domain.AssistantActionDefinition{
		Name:        "update_todo_due_date",
		Description: "Update due date for exactly one existing todo. Required keys: id (UUID string) and due_date (YYYY-MM-DD). Use this tool only for due-date changes. No extra keys. Valid: {\"id\":\"<uuid>\",\"due_date\":\"2026-04-30\"}. Invalid: {\"id\":\"<uuid>\",\"status\":\"DONE\"}.",
		Input: domain.AssistantActionInput{
			Type: "object",
			Fields: map[string]domain.AssistantActionField{
				"id": {
					Type:        "string",
					Description: "Todo UUID. REQUIRED.",
					Required:    true,
				},
				"due_date": {
					Type:        "string",
					Description: "Due date. REQUIRED. Use YYYY-MM-DD.",
					Required:    true,
				},
			},
		},
	}
}

// Execute executes TodoDueDateUpdaterAction.
func (tdut TodoDueDateUpdaterAction) Execute(ctx context.Context, call domain.AssistantActionCall, conversationHistory []domain.AssistantMessage) domain.AssistantMessage {
	params := struct {
		ID      uuid.UUID `json:"id"`
		DueDate string    `json:"due_date"`
	}{}

	exampleArgs := `{"id":"<uuid>","due_date":"2026-04-30"}`

	err := unmarshalActionInput(call.Input, &params)
	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	if params.ID == uuid.Nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"invalid_todo_id","details":"Todo ID cannot be nil.", "example":%s}`, exampleArgs),
		}
	}

	now := tdut.timeProvider.Now()
	dueDate, found := extractDateParam(params.DueDate, conversationHistory, now)
	if !found {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"invalid_due_date","details":"Due date cannot be empty. ISO 8601 string is required.", "example":%s}`, exampleArgs),
		}
	}

	var todo domain.Todo
	err = tdut.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		td, err := tdut.updater.Update(ctx, uow, params.ID, nil, nil, &dueDate)
		if err != nil {
			return err
		}
		todo = td
		return nil
	})

	if err != nil {
		return domain.AssistantMessage{
			Role:         domain.ChatRole_Tool,
			ActionCallID: &call.ID,
			Content:      fmt.Sprintf(`{"error":"update_due_date_error","details":"%s"}`, err.Error()),
		}
	}
	return domain.AssistantMessage{
		Role:         domain.ChatRole_Tool,
		ActionCallID: &call.ID,
		Content:      fmt.Sprintf(`{"message":"Your todo was updated successfully! todo: {"id":"%s", "title":"%s", "due_date":"%s", "status":"%s"}"}`, todo.ID, todo.Title, todo.DueDate.Format(time.DateOnly), todo.Status),
	}
}
