package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
)

// DeleteConversation defines the interface for deleting a conversation usecase
type DeleteConversation interface {
	Execute(ctx context.Context) error
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

// Execute deletes all messages in the global conversation
func (uc *DeleteConversationImpl) Execute(ctx context.Context) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	err := uc.uow.Execute(spanCtx, func(uow domain.UnitOfWork) error {
		if err := uow.ChatMessage().DeleteConversation(spanCtx); err != nil {
			return err
		}
		if err := uow.ConversationSummary().DeleteConversationSummary(spanCtx, domain.GlobalConversationID); err != nil {
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
