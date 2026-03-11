package postgres

import (
	"context"
	"database/sql"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont/depend"
)

// InitBoardSummaryRepository is a Symbiont initializer for BoardSummaryRepository.
type InitBoardSummaryRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the BoardSummaryRepository in the dependency container.
func (ibsr InitBoardSummaryRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[todo.BoardSummaryRepository](NewBoardSummaryRepository(ibsr.DB))
	return ctx, nil
}

// InitChatMessageRepository is a Symbiont initializer for ChatMessageRepository.
type InitChatMessageRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the ChatMessageRepository in the dependency container.
func (r InitChatMessageRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[assistant.ChatMessageRepository](NewChatMessageRepository(r.DB))
	return ctx, nil
}

// InitConversationRepository is a Symbiont initializer for ConversationRepository.
type InitConversationRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the ConversationRepository in the dependency container.
func (i InitConversationRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[assistant.ConversationRepository](NewConversationRepository(i.DB))
	return ctx, nil
}

// InitLocker is a Symbiont initializer for core.Locker.
type InitLocker struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the core.Locker in the dependency container.
func (i InitLocker) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[core.Locker](NewAdvisoryLocker(i.DB))
	return ctx, nil
}

// InitConversationSummaryRepository is a Symbiont initializer for ConversationSummaryRepository.
type InitConversationSummaryRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the ConversationSummaryRepository in the dependency container.
func (i InitConversationSummaryRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[assistant.ConversationSummaryRepository](NewConversationSummaryRepository(i.DB))
	return ctx, nil
}

// InitTodoRepository is a Symbiont initializer for TodoRepository.
type InitTodoRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the TodoRepository in the dependency container.
func (tr InitTodoRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[todo.Repository](NewTodoRepository(tr.DB))
	return ctx, nil
}

// InitUnitOfWork is responsible for initializing the UnitOfWork dependency.
type InitUnitOfWork struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the UnitOfWork implementation in the dependency container.
func (iuw InitUnitOfWork) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[transaction.UnitOfWork](NewUnitOfWork(iuw.DB))
	return ctx, nil
}
