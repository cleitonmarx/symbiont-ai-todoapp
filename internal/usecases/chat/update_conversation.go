package chat

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/google/uuid"
)

// UpdateConversation defines the interface for the UpdateConversation use case
type UpdateConversation interface {
	// Execute partially updates a conversation, such as the title.
	Execute(ctx context.Context, conversationID uuid.UUID, title string) (assistant.Conversation, error)
}

// UpdateConversationImpl is the implementation of the UpdateConversation use case
type UpdateConversationImpl struct {
	uow          transaction.UnitOfWork
	timeProvider core.CurrentTimeProvider
}

// NewUpdateConversationImpl creates a new instance of UpdateConversationImpl
func NewUpdateConversationImpl(uow transaction.UnitOfWork, timeProvider core.CurrentTimeProvider) *UpdateConversationImpl {
	return &UpdateConversationImpl{
		uow:          uow,
		timeProvider: timeProvider,
	}
}

// Execute partially updates a conversation, such as the title.
func (uc *UpdateConversationImpl) Execute(ctx context.Context, conversationID uuid.UUID, title string) (assistant.Conversation, error) {
	var updatedConv assistant.Conversation
	err := uc.uow.Execute(ctx, func(uowCtx context.Context, scope transaction.Scope) error {
		conversationRepo := scope.Conversation()

		conv, found, err := conversationRepo.GetConversation(uowCtx, conversationID)
		if err != nil {
			return err
		}
		if !found {
			return core.NewNotFoundErr(fmt.Sprintf("conversation with ID %s not found", conversationID))
		}

		if err := conv.ApplyUserTitle(title); err != nil {
			return err
		}

		conv.UpdatedAt = uc.timeProvider.Now()
		if err := conversationRepo.UpdateConversation(uowCtx, conv); err != nil {
			return err
		}
		updatedConv = conv
		return nil
	})
	if err != nil {
		return assistant.Conversation{}, err
	}
	return updatedConv, nil
}
