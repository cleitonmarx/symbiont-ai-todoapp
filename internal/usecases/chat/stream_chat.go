package chat

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/metrics"
	"github.com/google/uuid"
)

const (
	// MAX_CHAT_HISTORY_MESSAGES is the maximum number of prior messages included in chat context.
	MAX_CHAT_HISTORY_MESSAGES = 100

	// MAX_REPEATED_ACTION_CALL_HIT is the limit for repeated action-call detections before aborting.
	MAX_REPEATED_ACTION_CALL_HIT = 5

	// CHAT_TEMPERATURE controls generation randomness for streamed chat turns.
	CHAT_TEMPERATURE = 0.2
	// CHAT_TOP_P controls nucleus sampling for streamed chat turns.
	CHAT_TOP_P = 0.7

	// MAX_SKILLS_PROMPT_CHARS is the maximum size of injected skill prompt content.
	MAX_SKILLS_PROMPT_CHARS = 4000
	// MAX_RECOVERY_MESSAGES is the maximum number of recovery messages retained in loops.
	MAX_RECOVERY_MESSAGES = 8
	// DEFAULT_CONTEXT_COMPACTION_TIMEOUT bounds synchronous pre-turn compaction.
	DEFAULT_CONTEXT_COMPACTION_TIMEOUT = 20 * time.Second
)

// StreamChatParams holds optional parameters for StreamChat execution.
type StreamChatParams struct {
	ConversationID *uuid.UUID
}

// StreamChatOption defines a functional option for configuring StreamChatParams.
type StreamChatOption func(*StreamChatParams)

// WithConversationID binds Execute to an existing conversation instead of creating a new one.
func WithConversationID(conversationID uuid.UUID) StreamChatOption {
	return func(params *StreamChatParams) {
		params.ConversationID = &conversationID
	}
}

// StreamChat defines the interface for the StreamChat use case
type StreamChat interface {
	// Execute streams a chat response and persists the conversation
	Execute(ctx context.Context, userMessage, model string, onEvent assistant.EventCallback, opts ...StreamChatOption) error
}

// StreamChatImpl is the implementation of the StreamChat use case
type StreamChatImpl struct {
	logger                *log.Logger
	timeProvider          core.CurrentTimeProvider
	conversationRepo      assistant.ConversationRepository
	conversationCompactor ConversationCompactor
	compactionPolicy      assistant.CompactionPolicy
	compactionTimeout     time.Duration
	maxActionCycles       int
	sessionBuilder        TurnSessionBuilder
	turnRunner            TurnRunner
	conversationCreator   ConversationCreator
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(
	logger *log.Logger,
	timeProvider core.CurrentTimeProvider,
	conversationRepo assistant.ConversationRepository,
	conversationCompactor ConversationCompactor,
	compactionPolicy assistant.CompactionPolicy,
	compactionTimeout time.Duration,
	maxActionCycles int,
	sessionBuilder TurnSessionBuilder,
	turnRunner TurnRunner,
	conversationCreator ConversationCreator,
) StreamChatImpl {
	return StreamChatImpl{
		logger:                logger,
		timeProvider:          timeProvider,
		conversationRepo:      conversationRepo,
		conversationCompactor: conversationCompactor,
		compactionPolicy:      compactionPolicy,
		compactionTimeout:     compactionTimeout,
		maxActionCycles:       maxActionCycles,
		sessionBuilder:        sessionBuilder,
		turnRunner:            turnRunner,
		conversationCreator:   conversationCreator,
	}
}

// Execute streams a chat response and persists the conversation
func (sc StreamChatImpl) Execute(ctx context.Context, userMessage, model string, onEvent assistant.EventCallback, opts ...StreamChatOption) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	if strings.TrimSpace(userMessage) == "" {
		return core.NewValidationErr("message cannot be empty")
	}

	if model == "" {
		return core.NewValidationErr("model cannot be empty")
	}

	params := &StreamChatParams{}
	for _, opt := range opts {
		opt(params)
	}

	conversation, conversationCreated, err := sc.createOrRetrieveConversation(spanCtx, params.ConversationID, userMessage)
	if telemetry.IsErrorRecorded(span, err) {
		return err
	}

	if err := sc.compactIfNeeded(spanCtx, conversation.ID, onEvent); telemetry.IsErrorRecorded(span, err) {
		return err
	}

	session, err := sc.sessionBuilder.Build(spanCtx, BuildSessionParams{
		UserMessage:         userMessage,
		Model:               model,
		MaxActionCycles:     sc.maxActionCycles,
		Conversation:        conversation,
		ConversationCreated: conversationCreated,
	})
	if telemetry.IsErrorRecorded(span, err) {
		return err
	}

	if err := sc.turnRunner.Run(spanCtx, session, onEvent); telemetry.IsErrorRecorded(span, err) {
		failedAt := sc.timeProvider.Now()
		if userMessage, ok := session.TryBuildUserMessage(uuid.Nil, failedAt); ok {
			if persistErr := sc.conversationCreator.CreateMessage(spanCtx, session.Conversation(), userMessage); telemetry.IsErrorRecorded(span, persistErr) {
				return persistErr
			}
		}
		if persistErr := sc.conversationCreator.CreateMessage(spanCtx, session.Conversation(), session.BuildFailureMessage(failedAt, err)); telemetry.IsErrorRecorded(span, persistErr) {
			return persistErr
		}
		return err
	}

	assistantMsg := session.BuildFinalAssistantMessage(sc.timeProvider.Now())

	if assistantMsg.Content == "" {
		assistantMsg.Content = "Sorry, I could not process your request. Please try again."
		if err := onEvent(ctx, assistant.EventType_MessageDelta,
			assistant.MessageDelta{
				Text: assistantMsg.Content + "\n",
			},
		); telemetry.IsErrorRecorded(span, err) {
			return err
		}
	}

	err = sc.conversationCreator.CreateMessage(spanCtx, session.Conversation(), assistantMsg)
	if telemetry.IsErrorRecorded(span, err) {
		return err
	}

	tokenUsage := session.TokenUsage()
	metrics.RecordLLMTokensUsed(spanCtx, tokenUsage.PromptTokens, tokenUsage.CompletionTokens)

	if err := onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
		AssistantMessageID: assistantMsg.ID.String(),
		CompletedAt:        sc.timeProvider.Now().Format(time.RFC3339),
		Usage:              tokenUsage,
	}); telemetry.IsErrorRecorded(span, err) {
		return err
	}
	return nil
}

// createOrRetrieveConversation resolves the target conversation and creates one when no conversation ID is supplied.
func (sc StreamChatImpl) createOrRetrieveConversation(
	ctx context.Context,
	conversationID *uuid.UUID,
	userMessage string,
) (assistant.Conversation, bool, error) {
	if conversationID == nil {
		title := assistant.GenerateAutoConversationTitle(userMessage)
		conversation, err := sc.conversationRepo.CreateConversation(ctx, title, assistant.ConversationTitleSource_Auto)
		if err != nil {
			return assistant.Conversation{}, false, err
		}
		return conversation, true, nil
	}

	conversation, found, err := sc.conversationRepo.GetConversation(ctx, *conversationID)
	if err != nil {
		return assistant.Conversation{}, false, err
	}
	if !found {
		return assistant.Conversation{}, false, core.NewValidationErr("conversation not found")
	}

	return conversation, false, nil
}

// compactIfNeeded evaluates and runs pre-turn context compaction while emitting the corresponding stream events.
func (sc StreamChatImpl) compactIfNeeded(
	ctx context.Context,
	conversationID uuid.UUID,
	onEvent assistant.EventCallback,
) error {
	if sc.conversationCompactor == nil {
		return nil
	}

	evalCtx, cancelEval := context.WithTimeout(ctx, sc.compactionTimeout)
	defer cancelEval()

	decision, err := sc.conversationCompactor.EvaluateConversationCompaction(evalCtx, conversationID, sc.compactionPolicy)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("context compaction evaluation timed out after %s", sc.compactionTimeout)
		}
		if sc.logger != nil {
			sc.logger.Printf("StreamChat: context compaction evaluation failed for conversation %s: %v", conversationID, err)
		}
		return onEvent(ctx, assistant.EventType_ContextCompactionFailed, assistant.ContextCompactionFailed{
			ConversationID:           conversationID,
			UnsummarizedMessageCount: 0,
			UnsummarizedTotalTokens:  0,
			Reason:                   assistant.ContextCompactionReasonNone,
			Error:                    err.Error(),
		})
	}

	if !decision.ShouldCompact {
		return nil
	}

	if err := onEvent(ctx, assistant.EventType_ContextCompactionStarted, assistant.ContextCompactionStarted{
		ConversationID:           conversationID,
		UnsummarizedMessageCount: decision.MessageCount,
		UnsummarizedTotalTokens:  decision.TotalTokens,
		Reason:                   decision.Reason,
	}); err != nil {
		return err
	}

	compactCtx, cancelCompact := context.WithTimeout(ctx, sc.compactionTimeout)
	defer cancelCompact()

	if err := sc.conversationCompactor.Compact(compactCtx, conversationID); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("context compaction timed out after %s", sc.compactionTimeout)
		}
		if sc.logger != nil {
			sc.logger.Printf("StreamChat: context compaction failed for conversation %s: %v", conversationID, err)
		}
		return onEvent(ctx, assistant.EventType_ContextCompactionFailed, assistant.ContextCompactionFailed{
			ConversationID:           conversationID,
			UnsummarizedMessageCount: decision.MessageCount,
			UnsummarizedTotalTokens:  decision.TotalTokens,
			Reason:                   decision.Reason,
			Error:                    err.Error(),
		})
	}

	return onEvent(ctx, assistant.EventType_ContextCompactionCompleted, assistant.ContextCompactionCompleted{
		ConversationID:           conversationID,
		UnsummarizedMessageCount: decision.MessageCount,
		UnsummarizedTotalTokens:  decision.TotalTokens,
		Reason:                   decision.Reason,
		CompactedAt:              sc.timeProvider.Now().Format(time.RFC3339),
	})
}
