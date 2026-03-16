package chat

import (
	"context"
	"log"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/google/uuid"
)

// TurnRunner executes the assistant turn loop, including recovery retries.
type TurnRunner interface {
	// Run executes the streaming loop until the turn completes or fails.
	Run(ctx context.Context, session TurnSession, onEvent assistant.EventCallback) error
}

// turnRunnerImpl executes the assistant streaming loop and delegates event processing.
type turnRunnerImpl struct {
	logger              *log.Logger
	assistant           assistant.Assistant
	timeProvider        core.CurrentTimeProvider
	conversationCreator ConversationCreator
	actionPipeline      ActionPipeline
}

// newTurnRunner builds the default streamed turn runner.
func newTurnRunner(
	logger *log.Logger,
	assistantClient assistant.Assistant,
	timeProvider core.CurrentTimeProvider,
	conversationCreator ConversationCreator,
	actionPipeline ActionPipeline,
) TurnRunner {
	return turnRunnerImpl{
		logger:              logger,
		assistant:           assistantClient,
		timeProvider:        timeProvider,
		conversationCreator: conversationCreator,
		actionPipeline:      actionPipeline,
	}
}

// Run executes the assistant stream loop until completion, failure, or recovery exhaustion.
func (r turnRunnerImpl) Run(ctx context.Context, session TurnSession, onEvent assistant.EventCallback) error {
	for continueStreaming := true; continueStreaming; {
		continueStreaming = false
		var streamEventErr error
		request := session.Request()

		err := r.assistant.RunTurn(ctx, request, func(turnCtx context.Context, eventType assistant.EventType, data any) error {
			continueStreamingRequested, eventErr := r.handleStreamEvent(turnCtx, eventType, data, session, onEvent)
			if continueStreamingRequested {
				continueStreaming = true
			}
			if eventErr != nil && streamEventErr == nil {
				streamEventErr = eventErr
			}
			return eventErr
		})
		if err != nil {
			if streamEventErr == nil && prepareRunTurnRecovery(err, session) {
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
func (r turnRunnerImpl) handleStreamEvent(
	ctx context.Context,
	eventType assistant.EventType,
	data any,
	session TurnSession,
	onEvent assistant.EventCallback,
) (bool, error) {
	switch eventType {
	case assistant.EventType_TurnStarted:
		return false, r.handleTurnStarted(ctx, data, session, onEvent)
	case assistant.EventType_ActionRequested:
		return r.actionPipeline.Handle(ctx, data.(assistant.ActionCall), session, onEvent)
	case assistant.EventType_MessageDelta:
		delta := data.(assistant.MessageDelta)
		session.AppendAssistantContent(delta.Text)
		return false, onEvent(ctx, assistant.EventType_MessageDelta, delta)
	case assistant.EventType_TurnCompleted:
		done := data.(assistant.TurnCompleted)
		session.AddTokenUsage(done.Usage)
		return false, nil
	default:
		return false, nil
	}
}

// handleTurnStarted persists the user message after the assistant assigns turn metadata.
func (r turnRunnerImpl) handleTurnStarted(
	ctx context.Context,
	data any,
	session TurnSession,
	onEvent assistant.EventCallback,
) error {
	if session.AssistantMessageID() != uuid.Nil {
		return nil
	}

	meta := data.(assistant.TurnStarted)
	meta.ConversationID = session.Conversation().ID
	meta.ConversationCreated = session.ConversationCreated()
	meta.TurnID = session.TurnID()
	meta.SelectedSkills = session.SelectedSkills()
	session.SetAssistantMessageID(meta.AssistantMessageID)
	now := r.timeProvider.Now()
	userMessage, _ := session.TryBuildUserMessage(meta.UserMessageID, now)
	if err := r.conversationCreator.CreateMessage(ctx, session.Conversation(), userMessage); err != nil {
		return err
	}
	return onEvent(ctx, assistant.EventType_TurnStarted, meta)
}

// prepareRunTurnRecovery rewrites the request for one retry after an internal streaming failure.
func prepareRunTurnRecovery(runErr error, session TurnSession) bool {
	if !session.TryMarkRunTurnRecoveryAttempted() {
		return false
	}
	session.UpdateRequest(func(request *assistant.TurnRequest) {
		request.AvailableActions = nil
		request.Messages = compactToLastMessages(request.Messages, MAX_RECOVERY_MESSAGES)
		request.Messages = append(request.Messages, assistant.Message{
			Role: assistant.ChatRole_System,
			Content: "The previous assistant turn failed due to an internal processing issue " +
				"(commonly tool execution failure or context size limit). " +
				"Internal error: " + truncateToFirstChars(strings.TrimSpace(runErr.Error()), 400) + ". " +
				"Reply to the user with a short apology and explain that the request failed due to an internal error. " +
				"Suggest retrying with a smaller scope. Do not claim actions succeeded.",
		})
	})

	return true
}
