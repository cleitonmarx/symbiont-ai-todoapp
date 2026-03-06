package actions

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/toon-format/toon-go"
)

// createTodosRenderer renders successful create_todos tool results.
type createTodosRenderer struct{}

// Render converts a successful create_todos tool result into an assistant message.
func (createTodosRenderer) Render(_ assistant.ActionCall, result assistant.Message) (assistant.Message, bool) {
	todos, ok := parseRenderedTodos(result)
	if !ok {
		return assistant.Message{}, false
	}
	return assistant.Message{Role: assistant.ChatRole_Assistant, Content: renderTodoMutationResult("Created", todos)}, true
}

// updateTodosRenderer renders successful update_todos tool results.
type updateTodosRenderer struct{}

// Render converts a successful update_todos tool result into an assistant message.
func (updateTodosRenderer) Render(_ assistant.ActionCall, result assistant.Message) (assistant.Message, bool) {
	todos, ok := parseRenderedTodos(result)
	if !ok {
		return assistant.Message{}, false
	}
	return assistant.Message{Role: assistant.ChatRole_Assistant, Content: renderTodoMutationResult("Updated", todos)}, true
}

// updateTodosDueDateRenderer renders successful update_todos_due_date tool results.
type updateTodosDueDateRenderer struct{}

// Render converts a successful update_todos_due_date tool result into an assistant message.
func (updateTodosDueDateRenderer) Render(_ assistant.ActionCall, result assistant.Message) (assistant.Message, bool) {
	todos, ok := parseRenderedTodos(result)
	if !ok {
		return assistant.Message{}, false
	}
	return assistant.Message{Role: assistant.ChatRole_Assistant, Content: renderTodoMutationResult("Updated due date for", todos)}, true
}

// deleteTodosRenderer renders successful delete_todos tool results.
type deleteTodosRenderer struct{}

// Render converts a successful delete_todos tool result into an assistant message.
func (deleteTodosRenderer) Render(actionCall assistant.ActionCall, result assistant.Message) (assistant.Message, bool) {
	count, ok := parseDeletedRowsCount(result)
	if !ok {
		return assistant.Message{}, false
	}
	titles := parseDeleteTitles(actionCall.Input)
	return assistant.Message{Role: assistant.ChatRole_Assistant, Content: renderDeleteResult(count, titles)}, true
}

// renderedTodo is the minimal todo projection needed for deterministic rendering.
type renderedTodo struct {
	Title   string
	DueDate string
	Status  string
}

// parseRenderedTodos extracts todo rows from the compact local action result format.
func parseRenderedTodos(result assistant.Message) ([]renderedTodo, bool) {
	if result.Role != assistant.ChatRole_Tool || !result.IsActionCallSuccess() {
		return nil, false
	}

	payload := struct {
		Todos []struct {
			Title   string `toon:"title"`
			DueDate string `toon:"due_date"`
			Status  string `toon:"status"`
		} `toon:"todos"`
	}{}
	if err := toon.UnmarshalString(strings.TrimSpace(result.Content), &payload); err != nil {
		return nil, false
	}
	todos := make([]renderedTodo, 0, len(payload.Todos))
	for _, todo := range payload.Todos {
		todos = append(todos, renderedTodo{
			Title:   strings.TrimSpace(todo.Title),
			DueDate: strings.TrimSpace(todo.DueDate),
			Status:  strings.TrimSpace(todo.Status),
		})
	}
	return todos, true
}

// parseDeletedRowsCount extracts the deleted row count from the compact delete result format.
func parseDeletedRowsCount(result assistant.Message) (int, bool) {
	if result.Role != assistant.ChatRole_Tool || !result.IsActionCallSuccess() {
		return 0, false
	}

	payload := struct {
		Todos []struct {
			Deleted bool `toon:"deleted"`
		} `toon:"todos"`
	}{}
	if err := toon.UnmarshalString(strings.TrimSpace(result.Content), &payload); err != nil {
		return 0, false
	}
	return len(payload.Todos), true
}

// parseDeleteTitles extracts todo titles from the original delete action input when present.
func parseDeleteTitles(input string) []string {
	params := struct {
		Todos []struct {
			Title string `json:"title"`
		} `json:"todos"`
	}{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil
	}

	titles := make([]string, 0, len(params.Todos))
	for _, todo := range params.Todos {
		title := strings.TrimSpace(todo.Title)
		if title != "" {
			titles = append(titles, title)
		}
	}
	return titles
}

// renderTodoMutationResult formats a list of created or updated todos for the assistant response.
func renderTodoMutationResult(verb string, todos []renderedTodo) string {
	if len(todos) == 0 {
		return verb + " 0 todos."
	}
	if len(todos) == 1 {
		return fmt.Sprintf("%s %s.", verb, formatRenderedTodo(todos[0]))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s %d todos:", verb, len(todos))
	for _, todo := range todos {
		b.WriteString("\n")
		b.WriteString(formatRenderedTodo(todo))
	}
	return b.String()
}

// renderDeleteResult formats a delete confirmation using the deleted count and optional titles.
func renderDeleteResult(count int, titles []string) string {
	if count <= 0 {
		return "Deleted 0 todos."
	}
	if count == 1 && len(titles) > 0 {
		return fmt.Sprintf("Deleted **%s**.", titles[0])
	}
	return fmt.Sprintf("Deleted %d todos.", count)
}

// formatRenderedTodo formats one rendered todo row for assistant-facing output.
func formatRenderedTodo(todo renderedTodo) string {
	title := strings.TrimSpace(todo.Title)
	if title == "" {
		title = "Untitled"
	}
	return fmt.Sprintf("**%s** (Due: %s) - %s", title, formatRenderedDueDate(todo.DueDate), strings.TrimSpace(todo.Status))
}

// formatRenderedDueDate formats a raw YYYY-MM-DD date for assistant-facing output.
func formatRenderedDueDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "N/A"
	}
	parsed, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return raw
	}
	return parsed.Format("Jan 02, 2006")
}
