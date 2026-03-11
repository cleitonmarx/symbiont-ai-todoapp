package chat

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
)

// DeleteConversation defines the interface for deleting a conversation usecase
type DeleteConversation interface {
	Execute(ctx context.Context, conversationID uuid.UUID) error
}

// DeleteConversationImpl implements the DeleteConversation usecase
type DeleteConversationImpl struct {
	uow transaction.UnitOfWork
}

// NewDeleteConversationImpl creates a new DeleteConversationImpl instance
func NewDeleteConversationImpl(uow transaction.UnitOfWork) *DeleteConversationImpl {
	return &DeleteConversationImpl{
		uow: uow,
	}
}

// Execute deletes all messages and summaries related to the specified conversation, effectively resetting the conversation history.
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
