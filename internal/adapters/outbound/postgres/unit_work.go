package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
)

// UnitOfWork is the Postgres implementation of transaction.UnitOfWork.
type UnitOfWork struct {
	db *sql.DB
	tx *sql.Tx
}

// NewUnitOfWork builds a UnitOfWork bound to a database handle.
func NewUnitOfWork(db *sql.DB) *UnitOfWork {
	return &UnitOfWork{
		db: db,
	}
}

// Execute opens a transaction, runs fn, and commits or rolls back.
func (u *UnitOfWork) Execute(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	tx, err := u.db.BeginTx(spanCtx, nil)
	if err != nil {
		return err
	}

	uow := &UnitOfWork{
		db: u.db,
		tx: tx,
	}

	err = fn(spanCtx, uow)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction rollback error: %v, original error: %w", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}

// Todo returns a todo repository bound to the current runner (tx when present).
func (u *UnitOfWork) Todo() todo.Repository {
	return NewTodoRepository(u.getBaseRunner())
}

// Conversation returns a conversation repository bound to the current runner.
func (u *UnitOfWork) Conversation() assistant.ConversationRepository {
	return NewConversationRepository(u.getBaseRunner())
}

// ChatMessage returns a chat message repository bound to the current runner.
func (u *UnitOfWork) ChatMessage() assistant.ChatMessageRepository {
	return NewChatMessageRepository(u.getBaseRunner())
}

// Outbox returns an outbox repository bound to the current runner.
func (u *UnitOfWork) Outbox() outbox.Repository {
	return NewOutboxRepository(u.getBaseRunner())
}

// ConversationSummary returns a conversation summary repository bound to the current runner.
func (u *UnitOfWork) ConversationSummary() assistant.ConversationSummaryRepository {
	return NewConversationSummaryRepository(u.getBaseRunner())
}

// getBaseRunner picks the transaction runner when available, otherwise the DB handle.
func (u *UnitOfWork) getBaseRunner() squirrel.BaseRunner {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}
