package chat

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// ConversationCreator owns message creation and related conversation side effects.
type ConversationCreator interface {
	// CreateMessage stores one chat message and related side effects.
	CreateMessage(ctx context.Context, conversation assistant.Conversation, message assistant.ChatMessage) error
	// RepairTurn deletes dangling assistant action-call messages left by a failed turn.
	RepairTurn(ctx context.Context, conversation assistant.Conversation, turnID uuid.UUID) error
}

// ConversationCreatorImpl implements the ConversationCreator interface, coordinating message persistence, outbox event creation, and conversation timestamp updates.
type ConversationCreatorImpl struct {
	uow       transaction.UnitOfWork
	tokenizer assistant.Tokenizer
}

// NewConversationCreatorImpl builds the default conversation creator for stream chat.
func NewConversationCreatorImpl(
	uow transaction.UnitOfWork,
	tokenizer assistant.Tokenizer,
) ConversationCreatorImpl {
	return ConversationCreatorImpl{
		uow:       uow,
		tokenizer: tokenizer,
	}
}

// CreateMessage stores one chat message together with its outbox event and conversation timestamp updates.
func (p ConversationCreatorImpl) CreateMessage(
	ctx context.Context,
	conversation assistant.Conversation,
	message assistant.ChatMessage,
) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	message.ContextTokensEstimate = p.estimateContextTokens(spanCtx, message)

	return p.uow.Execute(spanCtx, func(uowCtx context.Context, scope transaction.Scope) error {
		if err := scope.ChatMessage().CreateChatMessages(uowCtx, []assistant.ChatMessage{message}); err != nil {
			return err
		}

		if err := scope.Outbox().CreateChatEvent(uowCtx, outbox.ChatMessageEvent{
			Type:           outbox.EventType_CHAT_MESSAGE_SENT,
			ChatRole:       message.ChatRole,
			ChatMessageID:  message.ID,
			ConversationID: message.ConversationID,
			CreatedAt:      message.CreatedAt,
		}); err != nil {
			return err
		}

		lastMessageAt := message.CreatedAt
		if conversation.LastMessageAt == nil || message.CreatedAt.After(*conversation.LastMessageAt) {
			conversation.LastMessageAt = &lastMessageAt
		}
		if message.CreatedAt.After(conversation.UpdatedAt) {
			conversation.UpdatedAt = message.CreatedAt
		}
		if err := scope.Conversation().UpdateConversation(uowCtx, conversation); err != nil {
			return err
		}

		return nil
	})
}

// RepairTurn removes unmatched assistant tool-call messages so future turns do not replay invalid history.
func (p ConversationCreatorImpl) RepairTurn(
	ctx context.Context,
	conversation assistant.Conversation,
	turnID uuid.UUID,
) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	return p.uow.Execute(spanCtx, func(uowCtx context.Context, scope transaction.Scope) error {
		messages, _, err := scope.ChatMessage().ListChatMessages(uowCtx, conversation.ID, 1, 0)
		if err != nil {
			return err
		}

		danglingMessageIDs := danglingAssistantActionCallMessageIDs(messages, turnID)
		if len(danglingMessageIDs) == 0 {
			return nil
		}

		if err := scope.ChatMessage().DeleteChatMessages(uowCtx, danglingMessageIDs); err != nil {
			return err
		}

		refreshedConversation, found, err := scope.Conversation().GetConversation(uowCtx, conversation.ID)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}

		updateConversationAfterMessageDeletion(&refreshedConversation, messages, danglingMessageIDs)
		if err := scope.Conversation().UpdateConversation(uowCtx, refreshedConversation); err != nil {
			return err
		}

		return nil
	})
}

// estimateContextTokens computes the persisted context footprint for a chat message.
func (p ConversationCreatorImpl) estimateContextTokens(ctx context.Context, message assistant.ChatMessage) int {
	input := assistant.BuildChatMessageTokenizationInput(message)
	if strings.TrimSpace(input) == "" {
		return 0
	}

	if p.tokenizer != nil {
		count, err := p.tokenizer.CountTokens(ctx, message.Model, input)
		if err == nil && count >= 0 {
			return count
		}
	}

	return assistant.EstimateTokenCountFallback(input)
}

// danglingAssistantActionCallMessageIDs returns assistant tool-call messages in one turn that do not have a matching tool result.
func danglingAssistantActionCallMessageIDs(
	messages []assistant.ChatMessage,
	turnID uuid.UUID,
) []uuid.UUID {
	toolResultsByActionCallID := make(map[string]struct{})
	for _, message := range messages {
		if message.TurnID != turnID || message.ChatRole != assistant.ChatRole_Tool || message.ActionCallID == nil {
			continue
		}
		toolResultsByActionCallID[*message.ActionCallID] = struct{}{}
	}

	danglingMessageIDs := make([]uuid.UUID, 0)
	for _, message := range messages {
		if message.TurnID != turnID || message.ChatRole != assistant.ChatRole_Assistant || len(message.ActionCalls) == 0 {
			continue
		}

		hasDanglingActionCall := false
		for _, actionCall := range message.ActionCalls {
			if _, found := toolResultsByActionCallID[actionCall.ID]; !found {
				hasDanglingActionCall = true
				break
			}
		}
		if hasDanglingActionCall {
			danglingMessageIDs = append(danglingMessageIDs, message.ID)
		}
	}

	return danglingMessageIDs
}

// updateConversationAfterMessageDeletion recalculates message timestamps after cancellation repair deletes dangling rows.
func updateConversationAfterMessageDeletion(
	conversation *assistant.Conversation,
	messages []assistant.ChatMessage,
	deletedMessageIDs []uuid.UUID,
) {
	if len(deletedMessageIDs) == 0 {
		return
	}

	var latestMessageAt *time.Time
	for _, message := range messages {
		if slices.Contains(deletedMessageIDs, message.ID) {
			continue
		}
		if latestMessageAt == nil || message.CreatedAt.After(*latestMessageAt) {
			ts := message.CreatedAt
			latestMessageAt = &ts
		}
	}

	conversation.LastMessageAt = latestMessageAt
	conversation.UpdatedAt = conversation.CreatedAt
	if latestMessageAt != nil && latestMessageAt.After(conversation.UpdatedAt) {
		conversation.UpdatedAt = *latestMessageAt
	}
}
