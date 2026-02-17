package usecases

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
)

// TodoUpdaterTool is an LLM tool for updating todos.
type TodoUpdaterTool struct {
	uow     domain.UnitOfWork
	updater TodoUpdater
}

// NewTodoUpdaterTool creates a new instance of TodoUpdaterTool.
func NewTodoUpdaterTool(uow domain.UnitOfWork, updater TodoUpdater) TodoUpdaterTool {
	return TodoUpdaterTool{
		uow:     uow,
		updater: updater,
	}
}

// StatusMessage returns a status message about the tool execution.
func (t TodoUpdaterTool) StatusMessage() string {
	return "✏️ Updating your todo..."
}

// Tool returns the LLMTool definition for the TodoUpdaterTool.
func (tut TodoUpdaterTool) Definition() domain.LLMToolDefinition {
	return domain.LLMToolDefinition{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "update_todo",
			Description: "Update metadata for exactly one existing todo. Required key: id (UUID). Optional keys: title and status. Use this tool only for title/status changes (never due date). status must be OPEN or DONE. No extra keys. Valid: {\"id\":\"<uuid>\",\"status\":\"DONE\"}. Invalid: {\"id\":\"<uuid>\",\"due_date\":\"2026-04-30\"}.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"id": {
						Type:        "string",
						Description: "Todo UUID. REQUIRED.",
						Required:    true,
					},
					"title": {
						Type:        "string",
						Description: "New title (optional).",
						Required:    false,
					},
					"status": {
						Type:        "string",
						Description: "OPEN or DONE (optional).",
						Required:    false,
					},
				},
			},
		},
	}
}

// Call executes the TodoMetaUpdaterTool with the provided function call.
func (tut TodoUpdaterTool) Call(ctx context.Context, call domain.LLMStreamEventToolCall, _ []domain.LLMChatMessage) domain.LLMChatMessage {
	params := struct {
		ID     string  `json:"id"`
		Title  *string `json:"title"`
		Status *string `json:"status"`
	}{}

	exampleArgs := `{"id":"<uuid>","status":"DONE", "title":"New title"}`

	err := unmarshalToolArguments(call.Arguments, &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_arguments","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	todoID, err := uuid.Parse(params.ID)
	if err != nil {
		return domain.LLMChatMessage{
			Role:       domain.ChatRole_Tool,
			ToolCallID: &call.ID,
			Content:    fmt.Sprintf(`{"error":"invalid_todo_id","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	var todo domain.Todo
	err = tut.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		td, err := tut.updater.Update(ctx, uow, todoID, params.Title, (*domain.TodoStatus)(params.Status), nil)
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
			Content:    fmt.Sprintf(`{"error":"update_todo_error","details":"%s", "example":%s}`, err.Error(), exampleArgs),
		}
	}

	return domain.LLMChatMessage{
		Role:       domain.ChatRole_Tool,
		ToolCallID: &call.ID,
		Content:    "Your todo was updated successfully! Updated todo: " + todo.ToLLMInput(),
	}
}
