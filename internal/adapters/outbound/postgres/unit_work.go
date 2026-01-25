package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
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
func (u *UnitOfWork) Execute(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	uow := &UnitOfWork{
		db: u.db,
		tx: tx,
	}

	err = fn(uow)
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

// Outbox returns the OutboxRepository for this UnitOfWork.
func (u *UnitOfWork) Outbox() domain.OutboxRepository {
	return NewOutboxRepository(u.getBaseRunner())
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
