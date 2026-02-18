package actions

import (
	"encoding/json"
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
