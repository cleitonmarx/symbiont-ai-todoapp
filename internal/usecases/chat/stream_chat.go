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
	// DEFAULT_CONTEXT_COMPACTION_TIMEOUT bounds synchronous pre-turn compaction.
	DEFAULT_CONTEXT_COMPACTION_TIMEOUT = 20 * time.Second
	// DEFAULT_CANCELED_TURN_REPAIR_TIMEOUT bounds cleanup work after a canceled turn.
	DEFAULT_CANCELED_TURN_REPAIR_TIMEOUT = 3 * time.Second
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
	stateBuilder          TurnStateBuilder
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
	stateBuilder TurnStateBuilder,
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
		stateBuilder:          stateBuilder,
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

	state, err := sc.stateBuilder.Build(spanCtx, BuildSessionParams{
		UserMessage:         userMessage,
		Model:               model,
		MaxActionCycles:     sc.maxActionCycles,
		Conversation:        conversation,
		ConversationCreated: conversationCreated,
	})
	if telemetry.IsErrorRecorded(span, err) {
		return err
	}

	now := sc.timeProvider.Now()
	userChatMessage := assistant.ChatMessage{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		TurnID:         state.TurnID(),
		TurnSequence:   state.NextTurnSequence(),
		ChatRole:       assistant.ChatRole_User,
		Content:        userMessage,
		Model:          model,
		MessageState:   assistant.ChatMessageState_Completed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := sc.conversationCreator.CreateMessage(spanCtx, state.Conversation(), userChatMessage); telemetry.IsErrorRecorded(span, err) {
		return err
	}

	if err := sc.turnRunner.Run(spanCtx, state, onEvent); telemetry.IsErrorRecorded(span, err) {
		if state.HasPersistedActionCall() {
			if repairErr := sc.repairFailedTurn(ctx, state); telemetry.IsErrorRecorded(span, repairErr) {
				return errors.Join(err, repairErr)
			}
		}
		if isCanceledTurnError(err) {
			return err
		}
		failedAt := sc.timeProvider.Now()
		if persistErr := sc.conversationCreator.CreateMessage(spanCtx, state.Conversation(), sc.buildFailureAssistantMessage(state, failedAt, err)); telemetry.IsErrorRecorded(span, persistErr) {
			return persistErr
		}
		return err
	}

	completedAt := sc.timeProvider.Now()
	assistantMsg := assistant.ChatMessage{
		ID:               uuid.New(),
		ConversationID:   state.Conversation().ID,
		TurnID:           state.TurnID(),
		TurnSequence:     state.NextTurnSequence(),
		ChatRole:         assistant.ChatRole_Assistant,
		Content:          state.AssistantContent(),
		SelectedSkills:   state.SelectedSkills(),
		Model:            state.Model(),
		MessageState:     assistant.ChatMessageState_Completed,
		PromptTokens:     state.TokenUsage().PromptTokens,
		CompletionTokens: state.TokenUsage().CompletionTokens,
		TotalTokens:      state.TokenUsage().TotalTokens,
		CreatedAt:        completedAt,
		UpdatedAt:        completedAt,
	}

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

	err = sc.conversationCreator.CreateMessage(spanCtx, state.Conversation(), assistantMsg)
	if telemetry.IsErrorRecorded(span, err) {
		return err
	}

	tokenUsage := state.TokenUsage()
	metrics.RecordLLMTokensUsed(spanCtx, tokenUsage.PromptTokens, tokenUsage.CompletionTokens)

	if err := onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
		Usage: tokenUsage,
	}); telemetry.IsErrorRecorded(span, err) {
		return err
	}
	return nil
}

// repairFailedTurn performs detached cleanup so failed turns do not leave dangling assistant tool-call messages in history.
func (sc StreamChatImpl) repairFailedTurn(ctx context.Context, state TurnState) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), DEFAULT_CANCELED_TURN_REPAIR_TIMEOUT)
	defer cancel()

	return sc.conversationCreator.RepairTurn(cleanupCtx, state.Conversation(), state.TurnID())
}

// isCanceledTurnError reports whether the turn ended due to cancellation rather than an internal assistant failure.
func isCanceledTurnError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// buildFailureAssistantMessage creates the persisted assistant failure message from the use-case-owned turn state.
func (sc StreamChatImpl) buildFailureAssistantMessage(
	state TurnState,
	now time.Time,
	streamErr error,
) assistant.ChatMessage {
	content := strings.TrimSpace(state.AssistantContent())
	if content == "" {
		content = "Sorry, I could not process your request. Please try again."
	}

	errorMessage := streamErr.Error()
	tokenUsage := state.TokenUsage()

	return assistant.ChatMessage{
		ID:               uuid.New(),
		ConversationID:   state.Conversation().ID,
		TurnID:           state.TurnID(),
		TurnSequence:     state.NextTurnSequence(),
		ChatRole:         assistant.ChatRole_Assistant,
		Content:          content,
		SelectedSkills:   state.SelectedSkills(),
		Model:            state.Model(),
		MessageState:     assistant.ChatMessageState_Failed,
		ErrorMessage:     &errorMessage,
		PromptTokens:     tokenUsage.PromptTokens,
		CompletionTokens: tokenUsage.CompletionTokens,
		TotalTokens:      tokenUsage.TotalTokens,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
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
