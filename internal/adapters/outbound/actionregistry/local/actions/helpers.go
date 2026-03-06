package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/google/uuid"
	"github.com/toon-format/toon-go"
)

// extractDateParam tries to extract a date from the provided parameter
// or from the user message history.
func extractDateParam(param string, history []assistant.Message, referenceDate time.Time) (time.Time, bool) {
	// First, try to extract from the provided parameter
	if dueDate, ok := core.ExtractTimeFromText(param, referenceDate, referenceDate.Location()); ok {
		return dueDate, true
	}

	// Next, scan the message history for date phrases
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Role != assistant.ChatRole_User {
			continue
		}
		if dueDate, ok := core.ExtractTimeFromText(msg.Content, referenceDate, referenceDate.Location()); ok {
			return dueDate, true
		}
	}
	return time.Time{}, false
}

// unmarshalActionInput unmarshals the action input from a JSON string into
// the target struct, ensuring that only a single JSON object is present and that there are no unknown fields.
func unmarshalActionInput(arguments string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}

	// Reject trailing JSON values after the first object.
	var extra any
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("action arguments must contain a single JSON object")
}

// newActionError constructs a standardized error message for assistant actions.
func newActionError(errorType, details, example string) string {
	return fmt.Sprintf("errors[1]{error,details,example}%s,%s,%s", errorType, details, example)
}

// parseDueDateParams parses and validates due date parameters, returning pointers to parsed times.
func parseDueDateParams(dueAfter, dueBefore *string, exampleArgs string) (*time.Time, *time.Time, *assistant.Message) {
	var (
		dueAfterTime  *time.Time
		dueBeforeTime *time.Time
		now           = time.Now().UTC()
	)

	if dueAfter != nil {
		parsedTime, ok := core.ExtractTimeFromText(*dueAfter, now, now.Location())
		if !ok {
			errMsg := assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: nil,
				Content:      newActionError("invalid_due_after", "could not parse due_after date", exampleArgs),
			}
			return nil, nil, &errMsg
		}
		dueAfterTime = &parsedTime
	}

	if dueBefore != nil {
		parsedTime, ok := core.ExtractTimeFromText(*dueBefore, now, now.Location())
		if !ok {
			errMsg := assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: nil,
				Content:      newActionError("invalid_due_before", "could not parse due_before date", exampleArgs),
			}
			return nil, nil, &errMsg
		}
		dueBeforeTime = &parsedTime
	}

	return dueAfterTime, dueBeforeTime, nil
}

// mapTodoFilterBuildErrCode maps errors from building todo search options to specific error codes for better client handling.
func mapTodoFilterBuildErrCode(err error) string {
	var validationErr *core.ValidationErr
	if errors.As(err, &validationErr) {
		switch err.Error() {
		case "due_after and due_before must be provided together":
			return "invalid_due_range"
		case "due_after must be less than or equal to due_before":
			return "invalid_due_range"
		case "search_by_similarity is required when using similarity sorting":
			return "missing_search_by_similarity_for_similarity_sort"
		case "sort_by is invalid":
			return "invalid_sort_by"
		case "only one search query is allowed":
			return "multiple_search_queries"
		case "status must be either OPEN or DONE":
			return "invalid_status"
		default:
			return "invalid_filters"
		}
	}
	return "embedding_error"
}

// formatTodosRows formats todos as a compact table-like payload consumed by the assistant.
func formatTodosRows(todos []todo.Todo) string {
	type todoRow struct {
		ID      string `toon:"id"`
		Title   string `toon:"title"`
		DueDate string `toon:"due_date"`
		Status  string `toon:"status"`
	}
	type payload struct {
		Todos []todoRow `toon:"todos"`
	}

	rows := make([]todoRow, 0, len(todos))
	for _, todo := range todos {
		rows = append(rows, todoRow{
			ID:      todo.ID.String(),
			Title:   todo.Title,
			DueDate: todo.DueDate.Format(time.DateOnly),
			Status:  string(todo.Status),
		})
	}

	content, err := toon.MarshalString(payload{Todos: rows})
	if err != nil {
		return newActionError("marshal_error", err.Error(), "")
	}
	return content
}

// formatDeletedRows formats deleted todo ids as a compact table-like payload.
func formatDeletedRows(ids []uuid.UUID) string {
	type deletedRow struct {
		ID      string `toon:"id"`
		Deleted bool   `toon:"deleted"`
	}
	type payload struct {
		Todos []deletedRow `toon:"todos"`
	}

	rows := make([]deletedRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, deletedRow{
			ID:      id.String(),
			Deleted: true,
		})
	}

	content, err := toon.MarshalString(payload{Todos: rows})
	if err != nil {
		return newActionError("marshal_error", err.Error(), "")
	}
	return content
}
