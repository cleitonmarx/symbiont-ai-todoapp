package usecases

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
)

type TodoDueDateUpdaterTool struct {
	uow          domain.UnitOfWork
	updater      TodoUpdater
	timeProvider domain.CurrentTimeProvider
}

// NewTodoDueDateUpdaterTool creates a new instance of TodoDueDateUpdaterTool.
func NewTodoDueDateUpdaterTool(uow domain.UnitOfWork, updater TodoUpdater, timeProvider domain.CurrentTimeProvider) TodoDueDateUpdaterTool {
	return TodoDueDateUpdaterTool{
		uow:          uow,
		updater:      updater,
		timeProvider: timeProvider,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoDueDateUpdaterTool) StatusMessage() string {
	return "📅 Updating the due date..."
}

// Tool returns the LLMTool definition for the TodoDueDateUpdaterTool.
func (tdut TodoDueDateUpdaterTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "update_todo_due_date",
			Description: "Update due date for exactly one existing todo. Required keys: id (UUID string) and due_date (YYYY-MM-DD). Use this tool only for due-date changes. No extra keys. Valid: {\"id\":\"<uuid>\",\"due_date\":\"2026-04-30\"}. Invalid: {\"id\":\"<uuid>\",\"status\":\"DONE\"}.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
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
		},
	}
}

// Call executes the TodoDueDateUpdaterTool with the provided function call.
func (tdut TodoDueDateUpdaterTool) Call(ctx context.Context, call domain.LLMStreamEventToolCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		ID      uuid.UUID `json:"id"`
		DueDate string    `json:"due_date"`
	}{}

	exampleArgs := `{"id":"<uuid>","due_date":"2026-04-30"}`

	err := unmarshalToolArguments(call.Arguments, &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	if params.ID == uuid.Nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_todo_id","details":"Todo ID cannot be nil.", "example":%s}`, exampleArgs),
		}
	}

	now := tdut.timeProvider.Now()
	dueDate, found := extractDateParam(params.DueDate, conversationHistory, now)
	if !found {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_due_date","details":"Due date cannot be empty. ISO 8601 string is required.", "example":%s}`, exampleArgs),
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
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"update_due_date_error","details":"%s"}`, err.Error()),
		}
	}
	return domain.LLMChatMessage{
		Role:       domain.ChatRole_Tool,
		ToolCallID: &call.ID,
		Content:    fmt.Sprintf(`{"message":"The due date was updated successfully! Updated todo: %s"}`, todo.ToLLMInput()),
	}
}
