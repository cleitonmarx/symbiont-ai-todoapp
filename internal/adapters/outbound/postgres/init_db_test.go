package postgres

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitDB_Initialize(t *testing.T) {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	dbInit := InitDB{
		Logger:        logger,
		DBUser:        "testuser",
		DBPass:        "testpass",
		DBHost:        "localhost",
		DBPort:        "5432",
		DBName:        "testdb",
		skipMigration: true,
	}

	_, err := dbInit.Initialize(context.Background())
	assert.NoError(t, err)
	resolveDB, err := depend.Resolve[*sql.DB]()
	assert.NoError(t, err)
	assert.NotNil(t, resolveDB)

}

func TestInitDB_Close(t *testing.T) {
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
