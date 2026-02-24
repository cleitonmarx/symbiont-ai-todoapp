package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
)

// UnitOfWork implements the domain.UnitOfWork interface for Postgres.
type UnitOfWork struct {
	db *sql.DB
	tx *sql.Tx
}

// NewUnitOfWork creates a new instance of UnitOfWork.
func NewUnitOfWork(db *sql.DB) *UnitOfWork {
	return &UnitOfWork{
		db: db,
	}
}

// Execute runs the provided function within a database transaction.
func (u *UnitOfWork) Execute(ctx context.Context, fn func(context.Context, domain.UnitOfWork) error) error {
	spanCtx, span := telemetry.Start(ctx)
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

// Todo returns the TodoRepository for this UnitOfWork.
func (u *UnitOfWork) Todo() domain.TodoRepository {
	return NewTodoRepository(u.getBaseRunner())
}

// Conversation returns the ConversationRepository for this UnitOfWork.
func (u *UnitOfWork) Conversation() domain.ConversationRepository {
	return NewConversationRepository(u.getBaseRunner())
}

// ChatMessage returns the ChatMessageRepository for this UnitOfWork.
func (u *UnitOfWork) ChatMessage() domain.ChatMessageRepository {
	return NewChatMessageRepository(u.getBaseRunner())
}

// Outbox returns the OutboxRepository for this UnitOfWork.
func (u *UnitOfWork) Outbox() domain.OutboxRepository {
	return NewOutboxRepository(u.getBaseRunner())
}

func (u *UnitOfWork) ConversationSummary() domain.ConversationSummaryRepository {
	return NewConversationSummaryRepository(u.getBaseRunner())
}

// getBaseRunner returns the appropriate BaseRunner (transaction or DB) for the UnitOfWork.
func (u *UnitOfWork) getBaseRunner() squirrel.BaseRunner {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}

// InitUnitOfWork is responsible for initializing the UnitOfWork dependency.
type InitUnitOfWork struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the UnitOfWork in the dependency container.
func (iuw InitUnitOfWork) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.UnitOfWork](NewUnitOfWork(iuw.DB))
	return ctx, nil
}
