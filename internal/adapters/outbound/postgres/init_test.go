package postgres

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitBoardSummaryRepository_Initialize(t *testing.T) {
	t.Parallel()

	i := &InitBoardSummaryRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	_, err = depend.Resolve[todo.BoardSummaryRepository]()
	assert.NoError(t, err)
}

func TestInitChatMessageRepository_Initialize(t *testing.T) {
	t.Parallel()

	i := &InitChatMessageRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	_, err = depend.Resolve[assistant.ChatMessageRepository]()
	assert.NoError(t, err)
}

func TestInitConversationSummaryRepository_Initialize(t *testing.T) {
	t.Parallel()

	i := &InitConversationSummaryRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	_, err = depend.Resolve[assistant.ConversationSummaryRepository]()
	assert.NoError(t, err)
}

func TestInitConversationRepository_Initialize(t *testing.T) {
	t.Parallel()

	i := &InitConversationRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	_, err = depend.Resolve[assistant.ConversationRepository]()
	assert.NoError(t, err)
}

func TestInitLocker_Initialize(t *testing.T) {
	t.Parallel()

	i := &InitLocker{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	_, err = depend.Resolve[core.Locker]()
	assert.NoError(t, err)
}

func TestInitDB_Initialize(t *testing.T) {
	t.Parallel()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	dbInit := InitDB{
		Logger:        logger,
		DBUser:        "testuser",
		DBPass:        "testpass",
		DBHost:        "localhost",
		DBPort:        "5432",
		DBName:        "testdb",
		SkipMigration: true,
	}

	_, err := dbInit.Initialize(t.Context())
	assert.NoError(t, err)
	resolveDB, err := depend.Resolve[*sql.DB]()
	assert.NoError(t, err)
	assert.NotNil(t, resolveDB)

}

func TestInitDB_Close(t *testing.T) {
	t.Parallel()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		dbInit          *InitDB
		shouldClose     bool
	}{
		"close-success": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose()
			},
			dbInit: &InitDB{
				Logger: logger,
			},
			shouldClose: true,
		},
		"close-log-error": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectClose().WillReturnError(sql.ErrConnDone)
			},
			dbInit: &InitDB{
				Logger: logger,
			},
			shouldClose: true,
		},
		"close-with-nil-db": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				// No expectations for nil db
			},
			dbInit: &InitDB{
				Logger: logger,
				db:     nil,
			},
			shouldClose: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.shouldClose {
				db, mock, err := sqlmock.New()
				assert.NoError(t, err)

				tt.setExpectations(mock)
				tt.dbInit.db = db

				tt.dbInit.Close()
				assert.NoError(t, mock.ExpectationsWereMet())
			} else {
				tt.dbInit.Close()
				assert.Nil(t, tt.dbInit.db)
			}
		})
	}
}

func TestInitTodoRepository_Initialize(t *testing.T) {
	t.Parallel()

	i := &InitTodoRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	_, err = depend.Resolve[todo.Repository]()
	assert.NoError(t, err)
}

func TestInitUnitOfWork_Initialize(t *testing.T) {
	t.Parallel()

	i := &InitUnitOfWork{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	_, err = depend.Resolve[transaction.UnitOfWork]()
	assert.NoError(t, err)

}
