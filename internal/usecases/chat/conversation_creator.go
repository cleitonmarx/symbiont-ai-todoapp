package chat

import (
	"context"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
)

// ConversationCreator owns message creation and related conversation side effects.
type ConversationCreator interface {
	// CreateMessage stores one chat message and related side effects.
	CreateMessage(ctx context.Context, conversation assistant.Conversation, message assistant.ChatMessage) error
}

// conversationCreator persists chat messages, outbox events, and token estimates.
type conversationCreator struct {
	uow       transaction.UnitOfWork
	tokenizer assistant.Tokenizer
}

// newConversationCreator builds the default conversation creator for stream chat.
func newConversationCreator(
	uow transaction.UnitOfWork,
	tokenizer assistant.Tokenizer,
) ConversationCreator {
	return conversationCreator{
		uow:       uow,
		tokenizer: tokenizer,
	}
}

// CreateMessage stores one chat message together with its outbox event and conversation timestamp updates.
func (p conversationCreator) CreateMessage(
	ctx context.Context,
	conversation assistant.Conversation,
	message assistant.ChatMessage,
) error {
	message.ContextTokensEstimate = p.estimateContextTokens(ctx, message)

	return p.uow.Execute(ctx, func(uowCtx context.Context, scope transaction.Scope) error {
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
func (p conversationCreator) estimateContextTokens(ctx context.Context, message assistant.ChatMessage) int {
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
