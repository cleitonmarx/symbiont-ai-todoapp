package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChatMessageRepository_CreateChatMessages(t *testing.T) {
	fixedID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedTime := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	msg := domain.ChatMessage{
		ID:             fixedID,
		ConversationID: domain.GlobalConversationID,
		ChatRole:       domain.ChatRole("user"),
		Content:        "hello",
		Model:          "ai/gpt-oss",
		ToolCalls: []domain.LLMStreamEventToolCall{
			{
				ID:        "id",
				Function:  "test_func",
				Arguments: "{\"arg1\":0}",
			},
		},
		CreatedAt: fixedTime,
	}

	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    error
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO ai_chat_messages (id,conversation_id,chat_role,content,tool_call_id,tool_calls,model,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)").
					WithArgs(
						msg.ID,
						msg.ConversationID,
						msg.ChatRole,
						msg.Content,
						msg.ToolCallID,
						[]byte(`[{"ID":"id","Function":"test_func","Arguments":"{\"arg1\":0}","Text":""}]`),
						msg.Model,
						msg.CreatedAt,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: nil,
		},
		"database-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO ai_chat_messages (id,conversation_id,chat_role,content,tool_call_id,tool_calls,model,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)").
					WithArgs(
						msg.ID,
						msg.ConversationID,
						msg.ChatRole,
						msg.Content,
						msg.ToolCallID,
						[]byte(`[{"ID":"id","Function":"test_func","Arguments":"{\"arg1\":0}","Text":""}]`),
						msg.Model,
						msg.CreatedAt,
					).
					WillReturnError(errors.New("db error"))
			},
			err: errors.New("db error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.expect(mock)

			repo := NewChatMessageRepository(db)
			gotErr := repo.CreateChatMessages(context.Background(), []domain.ChatMessage{msg})
			assert.Equal(t, tt.err, gotErr)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatMessageRepository_ListChatMessages(t *testing.T) {
	fixedID1 := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedID2 := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedID3 := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")
	t1 := time.Date(2026, 1, 24, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 24, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)

	row := func(id uuid.UUID, ts time.Time) []driver.Value {
		return []driver.Value{
			id.String(), // UUID as string for driver.Value
			domain.GlobalConversationID,
			domain.ChatRole("user"),
			"content",
			nil,
			nil,
			"ai/gpt-oss",
			ts,
		}
	}

	tests := map[string]struct {
		limit           int
		expect          func(sqlmock.Sqlmock)
		expectedMsgs    []domain.ChatMessage
		expectedHasMore bool
		expectErr       bool
	}{
		"success-no-limit": {
			limit: 0,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(chatFields).
					AddRow(row(fixedID3, t3)...).
					AddRow(row(fixedID2, t2)...).
					AddRow(row(fixedID1, t1)...)
				m.ExpectQuery("SELECT id, conversation_id, chat_role, content, tool_call_id, tool_calls, model, created_at FROM ai_chat_messages WHERE conversation_id = $1 ORDER BY created_at DESC").
					WithArgs(domain.GlobalConversationID).
					WillReturnRows(rows)
			},
			expectedMsgs: []domain.ChatMessage{
				{ID: fixedID1, ConversationID: domain.GlobalConversationID, ChatRole: domain.ChatRole("user"), Content: "content", ToolCallID: nil, ToolCalls: nil, Model: "ai/gpt-oss", CreatedAt: t1},
				{ID: fixedID2, ConversationID: domain.GlobalConversationID, ChatRole: domain.ChatRole("user"), Content: "content", ToolCallID: nil, ToolCalls: nil, Model: "ai/gpt-oss", CreatedAt: t2},
				{ID: fixedID3, ConversationID: domain.GlobalConversationID, ChatRole: domain.ChatRole("user"), Content: "content", ToolCallID: nil, ToolCalls: nil, Model: "ai/gpt-oss", CreatedAt: t3},
			},
			expectedHasMore: false,
			expectErr:       false,
		},
		"success-with-limit-and-has-more": {
			limit: 2,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(chatFields).
					AddRow(row(fixedID3, t3)...).
					AddRow(row(fixedID2, t2)...).
					AddRow(row(fixedID1, t1)...)

				m.ExpectQuery("SELECT id, conversation_id, chat_role, content, tool_call_id, tool_calls, model, created_at FROM ai_chat_messages WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT 3").
					WithArgs(domain.GlobalConversationID).
					WillReturnRows(rows)
			},
			expectedMsgs: []domain.ChatMessage{
				{ID: fixedID2, ConversationID: domain.GlobalConversationID, ChatRole: domain.ChatRole("user"), Content: "content", Model: "ai/gpt-oss", CreatedAt: t2},
				{ID: fixedID3, ConversationID: domain.GlobalConversationID, ChatRole: domain.ChatRole("user"), Content: "content", Model: "ai/gpt-oss", CreatedAt: t3},
			},
			expectedHasMore: true,
			expectErr:       false,
		},
		"empty": {
			limit: 0,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(chatFields)
				m.ExpectQuery("SELECT id, conversation_id, chat_role, content, tool_call_id, tool_calls, model, created_at FROM ai_chat_messages WHERE conversation_id = $1 ORDER BY created_at DESC").
					WithArgs(domain.GlobalConversationID).
					WillReturnRows(rows)
			},
			expectedMsgs:    nil, // repository returns nil when no rows
			expectedHasMore: false,
			expectErr:       false,
		},
		"database-error": {
			limit: 0,
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, conversation_id, chat_role, content, tool_call_id, tool_calls, model, created_at FROM ai_chat_messages WHERE conversation_id = $1 ORDER BY created_at DESC").
					WithArgs(domain.GlobalConversationID).
					WillReturnError(errors.New("db error"))
			},
			expectedMsgs:    nil,
			expectedHasMore: false,
			expectErr:       true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() //nolint:errcheck

			tt.expect(mock)

			repo := NewChatMessageRepository(db)
			got, hasMore, gotErr := repo.ListChatMessages(context.Background(), tt.limit)
			if tt.expectErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.expectedMsgs, got)
				assert.Equal(t, tt.expectedHasMore, hasMore)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestChatMessageRepository_DeleteConversation(t *testing.T) {
	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    error
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM ai_chat_messages WHERE conversation_id = $1").
					WithArgs(domain.GlobalConversationID).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: nil,
		},
		"database-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM ai_chat_messages WHERE conversation_id = $1").
					WithArgs(domain.GlobalConversationID).
					WillReturnError(errors.New("db error"))
			},
			err: errors.New("db error"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.expect(mock)

			repo := NewChatMessageRepository(db)
			gotErr := repo.DeleteConversation(context.Background())
			assert.Equal(t, tt.err, gotErr)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInitChatMessageRepository_Initialize(t *testing.T) {
	i := &InitChatMessageRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	_, err = depend.Resolve[domain.ChatMessageRepository]()
	assert.NoError(t, err)
}
