package chat

import (
	"context"
	"log"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

const (
	// MAX_RECOVERY_MESSAGES is the maximum number of recovery messages retained in loops.
	MAX_RECOVERY_MESSAGES = 8
)

// TurnRunner drives the assistant streaming loop for one turn state.
type TurnRunner interface {
	// Run streams the turn until it completes, fails, or exhausts recovery.
	Run(ctx context.Context, state TurnState, onEvent assistant.EventCallback) error
}

// TurnRunnerImpl implements TurnRunner.
type TurnRunnerImpl struct {
	logger         *log.Logger
	assistant      assistant.Assistant
	actionPipeline ActionPipeline
}

// NewTurnRunnerImpl creates a TurnRunnerImpl.
func NewTurnRunnerImpl(
	logger *log.Logger,
	assistantClient assistant.Assistant,
	actionPipeline ActionPipeline,
) TurnRunnerImpl {
	return TurnRunnerImpl{
		logger:         logger,
		assistant:      assistantClient,
		actionPipeline: actionPipeline,
	}
}

// Run implements TurnRunner.
func (r TurnRunnerImpl) Run(ctx context.Context, state TurnState, onEvent assistant.EventCallback) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	if err := onEvent(spanCtx, assistant.EventType_TurnStarted, assistant.TurnStarted{
		ConversationID:      state.Conversation().ID,
		ConversationCreated: state.ConversationCreated(),
		TurnID:              state.TurnID(),
		SelectedSkills:      state.SelectedSkills(),
	}); err != nil {
		return err
	}

	runTurnRecoveryAttempted := false
	for continueStreaming := true; continueStreaming; {
		continueStreaming = false
		var streamEventErr error
		request := state.Request()

		err := r.assistant.RunTurn(spanCtx, request, func(turnCtx context.Context, eventType assistant.EventType, data any) error {
			continueStreamingRequested, eventErr := r.handleStreamEvent(turnCtx, eventType, data, state, onEvent)
			if continueStreamingRequested {
				continueStreaming = true
			}
			if eventErr != nil && streamEventErr == nil {
				streamEventErr = eventErr
			}
			return eventErr
		})
		if err != nil {
			if streamEventErr == nil && prepareRunTurnRecovery(err, state, &runTurnRecoveryAttempted) {
				continueStreaming = true
				r.logger.Printf("StreamChat: encountered error during RunTurn, but prepared recovery. err=%v", err)
				continue
			}
			return err
		}
	}

	return nil
}

// handleStreamEvent processes one assistant stream event and returns loop control output.
func (r TurnRunnerImpl) handleStreamEvent(
	ctx context.Context,
	eventType assistant.EventType,
	data any,
	state TurnState,
	onEvent assistant.EventCallback,
) (bool, error) {
	switch eventType {
	case assistant.EventType_ActionRequested:
		return r.actionPipeline.Handle(ctx, data.(assistant.ActionCall), state, onEvent)
	case assistant.EventType_MessageDelta:
		delta := data.(assistant.MessageDelta)
		state.AppendAssistantContent(delta.Text)
		return false, onEvent(ctx, assistant.EventType_MessageDelta, delta)
	case assistant.EventType_TurnCompleted:
		done := data.(assistant.TurnCompleted)
		state.AccumulateTokenUsage(done.Usage)
		return false, nil
	default:
		return false, nil
	}
}

// prepareRunTurnRecovery rewrites the request for one retry after an internal streaming failure.
func prepareRunTurnRecovery(runErr error, state TurnState, attempted *bool) bool {
	if *attempted {
		return false
	}
	*attempted = true
	state.ApplyRecoveryPolicy(runErr, MAX_RECOVERY_MESSAGES)

	return true
}
