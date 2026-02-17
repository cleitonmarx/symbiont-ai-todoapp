package usecases

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
)

type TodoDeleterTool struct {
	uow     domain.UnitOfWork
	deleter TodoDeleter
}

// NewTodoDeleterTool creates a new instance of TodoDeleterTool.
func NewTodoDeleterTool(uow domain.UnitOfWork, deleter TodoDeleter) TodoDeleterTool {
	return TodoDeleterTool{
		uow:     uow,
		deleter: deleter,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoDeleterTool) StatusMessage() string {
	return "🗑️ Deleting the todo..."
}

// Tool returns the LLMTool definition for the TodoDeleterTool.
func (tdt TodoDeleterTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "delete_todo",
			Description: "Delete exactly one todo by id. Required key: id (UUID string). No extra keys. If id is unknown, call fetch_todos first. Valid: {\"id\":\"<uuid>\"}. Invalid: {\"id\":\"<uuid>\",\"confirm\":true}.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"id": {
						Type:        "string",
						Description: "Todo UUID. REQUIRED.",
						Required:    true,
					},
				},
			},
		},
	}
}

// Call executes the TodoDeleterTool with the provided function call.
func (tdt TodoDeleterTool) Call(ctx context.Context, call domain.LLMStreamEventToolCall, conversationHistory []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		ID uuid.UUID `json:"id"`
	}{}

	exampleArgs := `{"id":"<uuid>"}`

	err := unmarshalToolArguments(call.Arguments, &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	err = tdt.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		return tdt.deleter.Delete(ctx, uow, params.ID)
	})
	if err != nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"delete_todo_error","details":"%s"}`, err.Error()),
		}
	}

	return domain.LLMChatMessage{
		Role:       domain.ChatRole_Tool,
		ToolCallID: &call.ID,
		Content:    `{"message":"The todo was deleted successfully!"}`,
	}
}
