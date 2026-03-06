package assistant

import (
	"context"
)

// TurnRequest is the domain request for one assistant turn.
type TurnRequest struct {
	Model    string
	Messages []Message
	Stream   bool
	// Optional generation settings.
	Temperature      *float64
	TopP             *float64
	MaxTokens        *int
	FrequencyPenalty *float64
	AvailableActions []ActionDefinition
}

// TurnResponse contains the final assistant message and usage for non-stream mode.
type TurnResponse struct {
	Content string
	Usage   Usage
}

// Assistant defines assistant interaction in domain terms.
type Assistant interface {
	// RunTurn streams one assistant turn.
	RunTurn(ctx context.Context, req TurnRequest, onEvent EventCallback) error

	// RunTurnSync executes one assistant turn and returns the final response.
	RunTurnSync(ctx context.Context, req TurnRequest) (TurnResponse, error)
}
