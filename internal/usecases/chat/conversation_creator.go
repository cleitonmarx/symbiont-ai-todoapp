package chat

import (
	"context"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// ConversationCreator owns message creation and related conversation side effects.
type ConversationCreator interface {
	// CreateMessage stores one chat message and related side effects.
	CreateMessage(ctx context.Context, conversation assistant.Conversation, message assistant.ChatMessage) error
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
