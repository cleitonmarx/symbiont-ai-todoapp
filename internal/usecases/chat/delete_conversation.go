package chat

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// DeleteConversation deletes a conversation and all persisted data derived from it.
type DeleteConversation interface {
	// Execute removes the conversation, its messages, and its summaries.
	Execute(ctx context.Context, conversationID uuid.UUID) error
}

// DeleteConversationImpl implements DeleteConversation.
type DeleteConversationImpl struct {
	uow transaction.UnitOfWork
}

// NewDeleteConversationImpl creates a DeleteConversationImpl.
func NewDeleteConversationImpl(uow transaction.UnitOfWork) *DeleteConversationImpl {
	return &DeleteConversationImpl{
		uow: uow,
	}
}

// Execute implements DeleteConversation.
func (uc *DeleteConversationImpl) Execute(ctx context.Context, conversationID uuid.UUID) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	err := uc.uow.Execute(spanCtx, func(uowCtx context.Context, scope transaction.Scope) error {
		_, found, err := scope.Conversation().GetConversation(uowCtx, conversationID)
		if err != nil {
			return err
		}
		if !found {
			return core.NewNotFoundErr(fmt.Sprintf("conversation with ID %s not found", conversationID))
		}

		if err := scope.ChatMessage().DeleteConversationMessages(uowCtx, conversationID); err != nil {
			return err
		}
		if err := scope.ConversationSummary().DeleteConversationSummary(uowCtx, conversationID); err != nil {
			return err
		}
		if err := scope.Conversation().DeleteConversation(uowCtx, conversationID); err != nil {
			return err
		}
		return nil
	})

	if telemetry.IsErrorRecorded(span, err) {
		return err
	}
	return nil
}
