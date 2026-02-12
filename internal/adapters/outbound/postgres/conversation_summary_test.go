package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestConversationSummaryRepository_GetConversationSummary(t *testing.T) {
	summaryID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	messageID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	updatedAt := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		expect       func(sqlmock.Sqlmock)
		expected     domain.ConversationSummary
		expectedFind bool
		expectErr    bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(conversationSummaryFields).
					AddRow(summaryID, "global", "current state", messageID, updatedAt)
				m.ExpectQuery("SELECT id, conversation_id, current_state_summary, last_summarized_message_id, updated_at FROM conversations_summary WHERE conversation_id = $1 LIMIT 1").
					WithArgs("global").
					WillReturnRows(rows)
			},
			expected: domain.ConversationSummary{
				ID:                      summaryID,
				ConversationID:          "global",
				CurrentStateSummary:     "current state",
				LastSummarizedMessageID: &messageID,
				UpdatedAt:               updatedAt,
			},
			expectedFind: true,
			expectErr:    false,
		},
		"not-found": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, conversation_id, current_state_summary, last_summarized_message_id, updated_at FROM conversations_summary WHERE conversation_id = $1 LIMIT 1").
					WithArgs("global").
					WillReturnError(sql.ErrNoRows)
			},
			expected:     domain.ConversationSummary{},
			expectedFind: false,
			expectErr:    false,
		},
		"database-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, conversation_id, current_state_summary, last_summarized_message_id, updated_at FROM conversations_summary WHERE conversation_id = $1 LIMIT 1").
					WithArgs("global").
					WillReturnError(errors.New("db error"))
			},
			expected:     domain.ConversationSummary{},
			expectedFind: false,
			expectErr:    true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() //nolint:errcheck

			tt.expect(mock)

			repo := NewConversationSummaryRepository(db)
			got, found, gotErr := repo.GetConversationSummary(context.Background(), "global")
			if tt.expectErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.expectedFind, found)
				assert.Equal(t, tt.expected, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestConversationSummaryRepository_StoreConversationSummary(t *testing.T) {
	summaryID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	messageID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	updatedAt := time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC)

	summary := domain.ConversationSummary{
		ID:                      summaryID,
		ConversationID:          "global",
		CurrentStateSummary:     "current state",
		LastSummarizedMessageID: &messageID,
		UpdatedAt:               updatedAt,
	}

	tests := map[string]struct {
		expect    func(sqlmock.Sqlmock)
		expectErr bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`INSERT INTO conversations_summary (id,conversation_id,current_state_summary,last_summarized_message_id,updated_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (conversation_id) DO UPDATE SET current_state_summary = EXCLUDED.current_state_summary, last_summarized_message_id = EXCLUDED.last_summarized_message_id, updated_at = EXCLUDED.updated_at`).
					WithArgs(summary.ID, summary.ConversationID, summary.CurrentStateSummary, summary.LastSummarizedMessageID, summary.UpdatedAt).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectErr: false,
		},
		"database-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`INSERT INTO conversations_summary (id,conversation_id,current_state_summary,last_summarized_message_id,updated_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (conversation_id) DO UPDATE SET current_state_summary = EXCLUDED.current_state_summary, last_summarized_message_id = EXCLUDED.last_summarized_message_id, updated_at = EXCLUDED.updated_at`).
					WithArgs(summary.ID, summary.ConversationID, summary.CurrentStateSummary, summary.LastSummarizedMessageID, summary.UpdatedAt).
					WillReturnError(errors.New("db error"))
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() //nolint:errcheck

			tt.expect(mock)

			repo := NewConversationSummaryRepository(db)
			gotErr := repo.StoreConversationSummary(context.Background(), summary)
			if tt.expectErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInitConversationSummaryRepository_Initialize(t *testing.T) {
	i := &InitConversationSummaryRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	_, err = depend.Resolve[domain.ConversationSummaryRepository]()
	assert.NoError(t, err)
}
