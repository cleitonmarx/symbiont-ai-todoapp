package usecases

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
)

// UpdateConversation defines the interface for the UpdateConversation use case
type UpdateConversation interface {
	// Execute partially updates a conversation, such as the title.
	Execute(ctx context.Context, conversationID uuid.UUID, title string) (domain.Conversation, error)
}

// UpdateConversationImpl is the implementation of the UpdateConversation use case
type UpdateConversationImpl struct {
	uow          domain.UnitOfWork
	timeProvider domain.CurrentTimeProvider
}

// NewUpdateConversationImpl creates a new instance of UpdateConversationImpl
func NewUpdateConversationImpl(uow domain.UnitOfWork, timeProvider domain.CurrentTimeProvider) *UpdateConversationImpl {
	return &UpdateConversationImpl{
		uow:          uow,
		timeProvider: timeProvider,
	}
}

// Execute partially updates a conversation, such as the title.
func (uc *UpdateConversationImpl) Execute(ctx context.Context, conversationID uuid.UUID, title string) (domain.Conversation, error) {
	var updatedConv domain.Conversation
	err := uc.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		conv, found, err := uow.Conversation().GetConversation(ctx, conversationID)
		if err != nil {
			return err
		}
		if !found {
			return domain.NewNotFoundErr(fmt.Sprintf("conversation with ID %s not found", conversationID))
		}

		conv.Title = title
		conv.TitleSource = domain.ConversationTitleSource_User
		if err := conv.Validate(); err != nil {
			return err
		}
		conv.UpdatedAt = uc.timeProvider.Now().UTC()
		if err := uow.Conversation().UpdateConversation(ctx, conv); err != nil {
			return err
		}
		updatedConv = conv
		return nil
	})
	if err != nil {
		return domain.Conversation{}, err
	}
	return updatedConv, nil
}

// InitUpdateConversation initializes the UpdateConversation use case and registers it in the dependency container.
type InitUpdateConversation struct {
	Uow          domain.UnitOfWork          `resolve:""`
	TimeProvider domain.CurrentTimeProvider `resolve:""`
}

// Initialize initializes the UpdateConversationImpl use case.
func (i InitUpdateConversation) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[UpdateConversation](NewUpdateConversationImpl(i.Uow, i.TimeProvider))
	return ctx, nil
}
