package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// extractDateParam tries to extract a date from the provided parameter
// or from the user message history.
func extractDateParam(param string, history []domain.AssistantMessage, referenceDate time.Time) (time.Time, bool) {
	// First, try to extract from the provided parameter
	if dueDate, ok := domain.ExtractTimeFromText(param, referenceDate, referenceDate.Location()); ok {
		return dueDate, true
	}

	// Next, scan the message history for date phrases
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Role != domain.ChatRole_User {
			continue
		}
		if dueDate, ok := domain.ExtractTimeFromText(msg.Content, referenceDate, referenceDate.Location()); ok {
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
func parseDueDateParams(dueAfter, dueBefore *string, exampleArgs string) (*time.Time, *time.Time, *domain.AssistantMessage) {
	var (
		dueAfterTime  *time.Time
		dueBeforeTime *time.Time
		now           = time.Now().UTC()
	)

	if dueAfter != nil {
		parsedTime, ok := domain.ExtractTimeFromText(*dueAfter, now, now.Location())
		if !ok {
			errMsg := domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
				ActionCallID: nil,
				Content:      newActionError("invalid_due_after", "could not parse due_after date", exampleArgs),
			}
			return nil, nil, &errMsg
		}
		dueAfterTime = &parsedTime
	}

	if dueBefore != nil {
		parsedTime, ok := domain.ExtractTimeFromText(*dueBefore, now, now.Location())
		if !ok {
			errMsg := domain.AssistantMessage{
				Role:         domain.ChatRole_Tool,
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
	var validationErr *domain.ValidationErr
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
