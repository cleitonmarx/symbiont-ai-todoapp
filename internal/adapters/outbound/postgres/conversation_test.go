package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestConversationRepository_CreateConversation(t *testing.T) {
	fixedID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		title       string
		titleSource domain.ConversationTitleSource
		expect      func(sqlmock.Sqlmock)
		expected    domain.Conversation
		expectErr   bool
	}{
		"success": {
			title:       "Plan Japan trip",
			titleSource: domain.ConversationTitleSource_Auto,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(conversationFields).
					AddRow(fixedID, "Plan Japan trip", domain.ConversationTitleSource_Auto, nil, fixedTime, fixedTime)
				m.ExpectQuery("INSERT INTO conversations (id,title,title_source,last_message_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, title, title_source, last_message_at, created_at, updated_at").
					WithArgs(sqlmock.AnyArg(), "Plan Japan trip", domain.ConversationTitleSource_Auto, nil, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			expected: domain.Conversation{
				ID:          fixedID,
				Title:       "Plan Japan trip",
				TitleSource: domain.ConversationTitleSource_Auto,
				CreatedAt:   fixedTime,
				UpdatedAt:   fixedTime,
			},
			expectErr: false,
		},
		"validation-error-empty-title": {
			title:       "",
			titleSource: domain.ConversationTitleSource_Auto,
			expect:      func(sqlmock.Sqlmock) {},
			expectErr:   true,
		},
		"database-error": {
			title:       "Plan Japan trip",
			titleSource: domain.ConversationTitleSource_Auto,
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("INSERT INTO conversations (id,title,title_source,last_message_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, title, title_source, last_message_at, created_at, updated_at").
					WithArgs(sqlmock.AnyArg(), "Plan Japan trip", domain.ConversationTitleSource_Auto, nil, sqlmock.AnyArg(), sqlmock.AnyArg()).
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

			repo := NewConversationRepository(db)
			got, gotErr := repo.CreateConversation(context.Background(), tt.title, tt.titleSource)
			if tt.expectErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.expected, got)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestConversationRepository_GetConversation(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	lastMessageAt := time.Date(2026, 2, 16, 13, 0, 0, 0, time.UTC)
	fixedTime := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		expect       func(sqlmock.Sqlmock)
		expected     domain.Conversation
		expectedFind bool
		expectErr    bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(conversationFields).
					AddRow(conversationID, "Trip", domain.ConversationTitleSource_User, lastMessageAt, fixedTime, fixedTime)
				m.ExpectQuery("SELECT id, title, title_source, last_message_at, created_at, updated_at FROM conversations WHERE id = $1 LIMIT 1").
					WithArgs(conversationID).
					WillReturnRows(rows)
			},
			expected: domain.Conversation{
				ID:            conversationID,
				Title:         "Trip",
				TitleSource:   domain.ConversationTitleSource_User,
				LastMessageAt: &lastMessageAt,
				CreatedAt:     fixedTime,
				UpdatedAt:     fixedTime,
			},
			expectedFind: true,
			expectErr:    false,
		},
		"not-found": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, title, title_source, last_message_at, created_at, updated_at FROM conversations WHERE id = $1 LIMIT 1").
					WithArgs(conversationID).
					WillReturnError(sql.ErrNoRows)
			},
			expectedFind: false,
			expectErr:    false,
		},
		"database-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, title, title_source, last_message_at, created_at, updated_at FROM conversations WHERE id = $1 LIMIT 1").
					WithArgs(conversationID).
					WillReturnError(errors.New("db error"))
			},
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

			repo := NewConversationRepository(db)
			got, found, gotErr := repo.GetConversation(context.Background(), conversationID)
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

func TestConversationRepository_UpdateConversation(t *testing.T) {
	lastMessageAt := time.Date(2026, 2, 16, 13, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 16, 14, 0, 0, 0, time.UTC)
	conversation := domain.Conversation{
		ID:            uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Title:         "Renamed",
		TitleSource:   domain.ConversationTitleSource_User,
		LastMessageAt: &lastMessageAt,
		UpdatedAt:     updatedAt,
	}

	tests := map[string]struct {
		conversation domain.Conversation
		expect       func(sqlmock.Sqlmock)
		expectErr    bool
	}{
		"success": {
			conversation: conversation,
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE conversations SET title = $1, title_source = $2, last_message_at = $3, updated_at = $4 WHERE id = $5").
					WithArgs(conversation.Title, conversation.TitleSource, conversation.LastMessageAt, conversation.UpdatedAt, conversation.ID).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectErr: false,
		},
		"id-mismatch": {
			conversation: domain.Conversation{
				ID:          uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
				Title:       "Renamed",
				TitleSource: domain.ConversationTitleSource_User,
			},
			expect:    func(sqlmock.Sqlmock) {},
			expectErr: true,
		},
		"database-error": {
			conversation: conversation,
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("UPDATE conversations SET title = $1, title_source = $2, last_message_at = $3, updated_at = $4 WHERE id = $5").
					WithArgs(conversation.Title, conversation.TitleSource, conversation.LastMessageAt, conversation.UpdatedAt, conversation.ID).
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

			repo := NewConversationRepository(db)
			gotErr := repo.UpdateConversation(t.Context(), tt.conversation)
			if tt.expectErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestConversationRepository_ListConversations(t *testing.T) {
	c1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	c2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	c3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	createdAt := time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 16, 11, 0, 0, 0, time.UTC)
	lastMessageAt := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		page            int
		pageSize        int
		expect          func(sqlmock.Sqlmock)
		expected        []domain.Conversation
		expectedHasMore bool
		expectErr       bool
	}{
		"success-with-has-more": {
			page:     1,
			pageSize: 2,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(conversationFields).
					AddRow(c1, "C1", domain.ConversationTitleSource_Auto, lastMessageAt, createdAt, updatedAt).
					AddRow(c2, "C2", domain.ConversationTitleSource_User, nil, createdAt, updatedAt).
					AddRow(c3, "C3", domain.ConversationTitleSource_LLM, nil, createdAt, updatedAt)
				m.ExpectQuery("SELECT id, title, title_source, last_message_at, created_at, updated_at FROM conversations ORDER BY last_message_at DESC NULLS LAST, updated_at DESC, created_at DESC LIMIT 3 OFFSET 0").
					WillReturnRows(rows)
			},
			expected: []domain.Conversation{
				{
					ID:            c1,
					Title:         "C1",
					TitleSource:   domain.ConversationTitleSource_Auto,
					LastMessageAt: &lastMessageAt,
					CreatedAt:     createdAt,
					UpdatedAt:     updatedAt,
				},
				{
					ID:          c2,
					Title:       "C2",
					TitleSource: domain.ConversationTitleSource_User,
					CreatedAt:   createdAt,
					UpdatedAt:   updatedAt,
				},
			},
			expectedHasMore: true,
			expectErr:       false,
		},
		"invalid-page": {
			page:      0,
			pageSize:  2,
			expect:    func(sqlmock.Sqlmock) {},
			expectErr: true,
		},
		"database-error": {
			page:     1,
			pageSize: 2,
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, title, title_source, last_message_at, created_at, updated_at FROM conversations ORDER BY last_message_at DESC NULLS LAST, updated_at DESC, created_at DESC LIMIT 3 OFFSET 0").
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

			repo := NewConversationRepository(db)
			got, hasMore, gotErr := repo.ListConversations(t.Context(), tt.page, tt.pageSize)
			if tt.expectErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.expected, got)
				assert.Equal(t, tt.expectedHasMore, hasMore)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestConversationRepository_DeleteConversation(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tests := map[string]struct {
		expect    func(sqlmock.Sqlmock)
		expectErr bool
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM conversations WHERE id = $1").
					WithArgs(conversationID).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectErr: false,
		},
		"database-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM conversations WHERE id = $1").
					WithArgs(conversationID).
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

			repo := NewConversationRepository(db)
			gotErr := repo.DeleteConversation(t.Context(), conversationID)
			if tt.expectErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInitConversationRepository_Initialize(t *testing.T) {
	i := &InitConversationRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	_, err = depend.Resolve[domain.ConversationRepository]()
	assert.NoError(t, err)
}
