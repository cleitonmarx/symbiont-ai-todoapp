package transaction

import (
	"context"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
)

// Scope provides repository access for operations executed inside a unit of work.
type Scope interface {
	// Todo returns the todo repository for the current transaction scope.
	Todo() todo.Repository
	// Conversation returns the conversation repository for the current transaction scope.
	Conversation() assistant.ConversationRepository
	// ChatMessage returns the chat message repository for the current transaction scope.
	ChatMessage() assistant.ChatMessageRepository
	// ConversationSummary returns the conversation summary repository for the current transaction scope.
	ConversationSummary() assistant.ConversationSummaryRepository
	// Outbox returns the outbox repository for the current transaction scope.
	Outbox() outbox.Repository
}

// UnitOfWork coordinates atomic execution of a function.
type UnitOfWork interface {
	// Execute runs fn in a transactional context.
	// Returning an error rolls the transaction back; returning nil commits it.
	Execute(ctx context.Context, fn func(ctx context.Context, scope Scope) error) error
}
