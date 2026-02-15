package usecases

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
)

// DeleteConversation defines the interface for deleting a conversation usecase
type DeleteConversation interface {
	Execute(ctx context.Context, conversationID uuid.UUID) error
}

// DeleteConversationImpl implements the DeleteConversation usecase
type DeleteConversationImpl struct {
	uow domain.UnitOfWork
}

// NewDeleteConversationImpl creates a new DeleteConversationImpl instance
func NewDeleteConversationImpl(uow domain.UnitOfWork) *DeleteConversationImpl {
	return &DeleteConversationImpl{
		uow: uow,
	}
}

// Execute deletes all messages and summaries related to the specified conversation, effectively resetting the conversation history.
func (uc *DeleteConversationImpl) Execute(ctx context.Context, conversationID uuid.UUID) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	err := uc.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		_, found, err := uow.Conversation().GetConversation(spanCtx, conversationID)
		if err != nil {
			return err
		}
		if !found {
			return domain.NewNotFoundErr(fmt.Sprintf("conversation with ID %s not found", conversationID))
		}

		if err := uow.ChatMessage().DeleteConversationMessages(spanCtx, conversationID); err != nil {
			return err
		}
		if err := uow.ConversationSummary().DeleteConversationSummary(spanCtx, conversationID); err != nil {
			return err
		}
		if err := uow.Conversation().DeleteConversation(spanCtx, conversationID); err != nil {
			return err
		}
		return nil
	})

	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// InitDeleteConversation is the initializer for the DeleteConversation usecase
type InitDeleteConversation struct {
	Uow domain.UnitOfWork `resolve:""`
}

// Initialize registers the DeleteConversation usecase in the dependency container
func (i InitDeleteConversation) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[DeleteConversation](NewDeleteConversationImpl(i.Uow))
	return ctx, nil
}
