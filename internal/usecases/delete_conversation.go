package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
)

// DeleteConversation defines the interface for deleting a conversation usecase
type DeleteConversation interface {
	Execute(ctx context.Context) error
}

// DeleteConversationImpl implements the DeleteConversation usecase
type DeleteConversationImpl struct {
	chatMessageRepo domain.ChatMessageRepository
}

// NewDeleteConversationImpl creates a new DeleteConversationImpl instance
func NewDeleteConversationImpl(chatMessageRepo domain.ChatMessageRepository) *DeleteConversationImpl {
	return &DeleteConversationImpl{
		chatMessageRepo: chatMessageRepo,
	}
}

// Execute deletes all messages in the global conversation
func (uc *DeleteConversationImpl) Execute(ctx context.Context) error {
	return uc.chatMessageRepo.DeleteConversation(ctx)
}

// InitDeleteConversation is the initializer for the DeleteConversation usecase
type InitDeleteConversation struct {
	Repo domain.ChatMessageRepository `resolve:""`
}

// Initialize registers the DeleteConversation usecase in the dependency container
func (i InitDeleteConversation) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[DeleteConversation](NewDeleteConversationImpl(i.Repo))
	return ctx, nil
}
