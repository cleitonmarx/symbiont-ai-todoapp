package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
)

// ListChatMessages defines the interface for the ListChatMessages use case
type ListChatMessages interface {
	Query(ctx context.Context, page int, pageSize int) ([]domain.ChatMessage, bool, error)
}

// ListChatMessagesImpl is the implementation of the ListChatMessages use case
type ListChatMessagesImpl struct {
	ChatMessageRepo domain.ChatMessageRepository `resolve:""`
}

// NewListChatMessagesImpl creates a new instance of ListChatMessagesImpl
func NewListChatMessagesImpl(chatMessageRepo domain.ChatMessageRepository) ListChatMessagesImpl {
	return ListChatMessagesImpl{
		ChatMessageRepo: chatMessageRepo,
	}
}

// Query retrieves chat messages with pagination support
func (lcm ListChatMessagesImpl) Query(ctx context.Context, page int, pageSize int) ([]domain.ChatMessage, bool, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	messages, hasMore, err := lcm.ChatMessageRepo.ListChatMessages(spanCtx, pageSize)
	if tracing.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}

	// Filter out tool messages before returning to the user
	messagesToReturnToUser := []domain.ChatMessage{}
	for _, msg := range messages {
		if msg.ChatRole != domain.ChatRole_Tool && len(msg.Content) > 0 {
			messagesToReturnToUser = append(messagesToReturnToUser, msg)
		}
	}

	return messagesToReturnToUser, hasMore, nil
}

// InitListChatMessages is the initializer for the ListChatMessages use case
type InitListChatMessages struct {
	Repo domain.ChatMessageRepository `resolve:""`
}

// Initialize registers the ListChatMessages use case in the dependency container
func (i InitListChatMessages) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListChatMessages](NewListChatMessagesImpl(i.Repo))
	return ctx, nil
}
