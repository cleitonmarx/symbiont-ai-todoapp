package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChatMessageRepository_CreateChatMessages(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	fixedID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	turnID := uuid.MustParse("323e4567-e89b-12d3-a456-426614174100")
	turnSequence := int64(7)
	fixedTime := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	updatedAt := fixedTime.Add(2 * time.Second)
	msg := domain.ChatMessage{
		ID:             fixedID,
		ConversationID: conversationID,
		TurnID:         turnID,
		TurnSequence:   turnSequence,
		ChatRole:       domain.ChatRole("user"),
		Content:        "hello",
		Model:          "ai/gpt-oss",
		MessageState:   domain.ChatMessageState_Completed,
		ToolCalls: []domain.LLMStreamEventToolCall{
			{
				ID:        "id",
				Function:  "test_func",
				Arguments: "{\"arg1\":0}",
			},
		},
		CreatedAt: fixedTime,
		UpdatedAt: updatedAt,
	}

	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    error
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO ai_chat_messages (id,conversation_id,turn_id,turn_sequence,chat_role,content,tool_call_id,tool_calls,model,message_state,error_message,prompt_tokens,completion_tokens,total_tokens,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)").
					WithArgs(
						msg.ID,
						msg.ConversationID,
						msg.TurnID,
						msg.TurnSequence,
						msg.ChatRole,
						msg.Content,
						msg.ToolCallID,
						[]byte(`[{"id":"id","function":"test_func","arguments":"{\"arg1\":0}","text":""}]`),
						msg.Model,
						msg.MessageState,
						nil,
						msg.PromptTokens,
						msg.CompletionTokens,
						msg.TotalTokens,
						msg.CreatedAt,
						msg.UpdatedAt,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: nil,
		},
		"database-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("INSERT INTO ai_chat_messages (id,conversation_id,turn_id,turn_sequence,chat_role,content,tool_call_id,tool_calls,model,message_state,error_message,prompt_tokens,completion_tokens,total_tokens,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)").
					WithArgs(
						msg.ID,
						msg.ConversationID,
						msg.TurnID,
						msg.TurnSequence,
						msg.ChatRole,
						msg.Content,
						msg.ToolCallID,
						[]byte(`[{"id":"id","function":"test_func","arguments":"{\"arg1\":0}","text":""}]`),
						msg.Model,
						msg.MessageState,
						nil,
						msg.PromptTokens,
						msg.CompletionTokens,
						msg.TotalTokens,
						msg.CreatedAt,
						msg.UpdatedAt,
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
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	fixedID1 := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedID2 := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedID3 := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")
	t1 := time.Date(2026, 1, 24, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 24, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	turnID1 := uuid.MustParse("423e4567-e89b-12d3-a456-426614174003")
	turnID2 := uuid.MustParse("523e4567-e89b-12d3-a456-426614174004")
	turnID3 := uuid.MustParse("623e4567-e89b-12d3-a456-426614174005")

	row := func(id uuid.UUID, conversationID uuid.UUID, turnID uuid.UUID, turnSequence int64, ts time.Time) []driver.Value {
		return []driver.Value{
			id.String(),
			conversationID.String(),
			turnID.String(),
			turnSequence,
			domain.ChatRole("user"),
			"content",
			nil,
			nil,
			"ai/gpt-oss",
			string(domain.ChatMessageState_Completed),
			nil,
			0,
			0,
			0,
			ts,
			ts,
		}
	}

	tests := map[string]struct {
		page            int
		pageSize        int
		expect          func(sqlmock.Sqlmock)
		expectedMsgs    []domain.ChatMessage
		expectedHasMore bool
		expectErr       bool
	}{
		"success-first-page": {
			page:     1,
			pageSize: 10,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(chatFields).
					AddRow(row(fixedID3, conversationID, turnID3, 2, t3)...).
					AddRow(row(fixedID2, conversationID, turnID2, 1, t2)...).
					AddRow(row(fixedID1, conversationID, turnID1, 0, t1)...)
				m.ExpectQuery("SELECT id, conversation_id, turn_id, turn_sequence, chat_role, content, tool_call_id, tool_calls, model, message_state, error_message, prompt_tokens, completion_tokens, total_tokens, created_at, updated_at FROM ai_chat_messages WHERE conversation_id = $1 ORDER BY created_at DESC, id DESC LIMIT 11").
					WithArgs(conversationID).
					WillReturnRows(rows)
			},
			expectedMsgs: []domain.ChatMessage{
				{ID: fixedID1, ConversationID: conversationID, TurnID: turnID1, TurnSequence: 0, ChatRole: domain.ChatRole("user"), Content: "content", ToolCallID: nil, ToolCalls: nil, Model: "ai/gpt-oss", MessageState: domain.ChatMessageState_Completed, CreatedAt: t1, UpdatedAt: t1},
				{ID: fixedID2, ConversationID: conversationID, TurnID: turnID2, TurnSequence: 1, ChatRole: domain.ChatRole("user"), Content: "content", ToolCallID: nil, ToolCalls: nil, Model: "ai/gpt-oss", MessageState: domain.ChatMessageState_Completed, CreatedAt: t2, UpdatedAt: t2},
				{ID: fixedID3, ConversationID: conversationID, TurnID: turnID3, TurnSequence: 2, ChatRole: domain.ChatRole("user"), Content: "content", ToolCallID: nil, ToolCalls: nil, Model: "ai/gpt-oss", MessageState: domain.ChatMessageState_Completed, CreatedAt: t3, UpdatedAt: t3},
			},
			expectedHasMore: false,
			expectErr:       false,
		},
		"success-with-pagination-and-has-more": {
			page:     1,
			pageSize: 2,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(chatFields).
					AddRow(row(fixedID3, conversationID, turnID3, 2, t3)...).
					AddRow(row(fixedID2, conversationID, turnID2, 1, t2)...).
					AddRow(row(fixedID1, conversationID, turnID1, 0, t1)...)

				m.ExpectQuery("SELECT id, conversation_id, turn_id, turn_sequence, chat_role, content, tool_call_id, tool_calls, model, message_state, error_message, prompt_tokens, completion_tokens, total_tokens, created_at, updated_at FROM ai_chat_messages WHERE conversation_id = $1 ORDER BY created_at DESC, id DESC LIMIT 3").
					WithArgs(conversationID).
					WillReturnRows(rows)
			},
			expectedMsgs: []domain.ChatMessage{
				{ID: fixedID2, ConversationID: conversationID, TurnID: turnID2, TurnSequence: 1, ChatRole: domain.ChatRole("user"), Content: "content", Model: "ai/gpt-oss", MessageState: domain.ChatMessageState_Completed, CreatedAt: t2, UpdatedAt: t2},
				{ID: fixedID3, ConversationID: conversationID, TurnID: turnID3, TurnSequence: 2, ChatRole: domain.ChatRole("user"), Content: "content", Model: "ai/gpt-oss", MessageState: domain.ChatMessageState_Completed, CreatedAt: t3, UpdatedAt: t3},
			},
			expectedHasMore: true,
			expectErr:       false,
		},
		"success-second-page": {
			page:     2,
			pageSize: 2,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(chatFields).
					AddRow(row(fixedID1, conversationID, turnID1, 0, t1)...)

				m.ExpectQuery("SELECT id, conversation_id, turn_id, turn_sequence, chat_role, content, tool_call_id, tool_calls, model, message_state, error_message, prompt_tokens, completion_tokens, total_tokens, created_at, updated_at FROM ai_chat_messages WHERE conversation_id = $1 ORDER BY created_at DESC, id DESC LIMIT 3 OFFSET 2").
					WithArgs(conversationID).
					WillReturnRows(rows)
			},
			expectedMsgs: []domain.ChatMessage{
				{ID: fixedID1, ConversationID: conversationID, TurnID: turnID1, TurnSequence: 0, ChatRole: domain.ChatRole("user"), Content: "content", Model: "ai/gpt-oss", MessageState: domain.ChatMessageState_Completed, CreatedAt: t1, UpdatedAt: t1},
			},
			expectedHasMore: false,
			expectErr:       false,
		},
		"empty-page": {
			page:     1,
			pageSize: 10,
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(chatFields)
				m.ExpectQuery("SELECT id, conversation_id, turn_id, turn_sequence, chat_role, content, tool_call_id, tool_calls, model, message_state, error_message, prompt_tokens, completion_tokens, total_tokens, created_at, updated_at FROM ai_chat_messages WHERE conversation_id = $1 ORDER BY created_at DESC, id DESC LIMIT 11").
					WithArgs(conversationID).
					WillReturnRows(rows)
			},
			expectedMsgs:    nil,
			expectedHasMore: false,
			expectErr:       false,
		},
		"database-error": {
			page:     1,
			pageSize: 10,
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, conversation_id, turn_id, turn_sequence, chat_role, content, tool_call_id, tool_calls, model, message_state, error_message, prompt_tokens, completion_tokens, total_tokens, created_at, updated_at FROM ai_chat_messages WHERE conversation_id = $1 ORDER BY created_at DESC, id DESC LIMIT 11").
					WithArgs(conversationID).
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
			got, hasMore, gotErr := repo.ListChatMessages(context.Background(), conversationID, tt.page, tt.pageSize)
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

func TestChatMessageRepository_ListChatMessages_WithOptionalParameters(t *testing.T) {
	fixedID1 := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	fixedID2 := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedID3 := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")
	fixedID4 := uuid.MustParse("423e4567-e89b-12d3-a456-426614174003")
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	fixedTime := time.Date(2026, 1, 24, 10, 0, 0, 0, time.UTC)
	turnID := uuid.MustParse("523e4567-e89b-12d3-a456-426614174004")

	row := func(id uuid.UUID, turnID uuid.UUID, turnSequence int64, ts time.Time) []driver.Value {
		return []driver.Value{
			id.String(),
			conversationID.String(),
			turnID.String(),
			turnSequence,
			domain.ChatRole("user"),
			"content",
			nil,
			nil,
			"ai/gpt-oss",
			string(domain.ChatMessageState_Completed),
			nil,
			0,
			0,
			0,
			ts,
			ts,
		}
	}

	tests := map[string]struct {
		page            int
		pageSize        int
		options         []domain.ListChatMessagesOption
		expect          func(sqlmock.Sqlmock)
		expectedMsgs    []domain.ChatMessage
		expectedHasMore bool
		expectErr       bool
	}{
		"success-with-after-message-option": {
			page:     1,
			pageSize: 2,
			options: []domain.ListChatMessagesOption{
				domain.WithChatMessagesAfterMessageID(fixedID1),
			},
			expect: func(m sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(chatFields).
					AddRow(row(fixedID2, turnID, 1, fixedTime)...).
					AddRow(row(fixedID3, turnID, 2, fixedTime)...).
					AddRow(row(fixedID4, turnID, 3, fixedTime)...)
				m.ExpectQuery("SELECT id, conversation_id, turn_id, turn_sequence, chat_role, content, tool_call_id, tool_calls, model, message_state, error_message, prompt_tokens, completion_tokens, total_tokens, created_at, updated_at FROM ai_chat_messages LEFT JOIN ( SELECT created_at AS checkpoint_created_at, id AS checkpoint_id FROM ai_chat_messages WHERE conversation_id = $1 AND id = $2 LIMIT 1 ) checkpoint ON TRUE WHERE conversation_id = $3 AND (checkpoint.checkpoint_id IS NULL OR ai_chat_messages.created_at > checkpoint.checkpoint_created_at OR (ai_chat_messages.created_at = checkpoint.checkpoint_created_at AND ai_chat_messages.id > checkpoint.checkpoint_id)) ORDER BY created_at ASC, id ASC LIMIT 3").
					WithArgs(conversationID, fixedID1, conversationID).
					WillReturnRows(rows)
			},
			expectedMsgs: []domain.ChatMessage{
				{ID: fixedID2, ConversationID: conversationID, TurnID: turnID, TurnSequence: 1, ChatRole: domain.ChatRole("user"), Content: "content", ToolCallID: nil, ToolCalls: nil, Model: "ai/gpt-oss", MessageState: domain.ChatMessageState_Completed, CreatedAt: fixedTime, UpdatedAt: fixedTime},
				{ID: fixedID3, ConversationID: conversationID, TurnID: turnID, TurnSequence: 2, ChatRole: domain.ChatRole("user"), Content: "content", ToolCallID: nil, ToolCalls: nil, Model: "ai/gpt-oss", MessageState: domain.ChatMessageState_Completed, CreatedAt: fixedTime, UpdatedAt: fixedTime},
			},
			expectedHasMore: true,
			expectErr:       false,
		},
		"after-message-query-error": {
			page:     1,
			pageSize: 10,
			options: []domain.ListChatMessagesOption{
				domain.WithChatMessagesAfterMessageID(fixedID1),
			},
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectQuery("SELECT id, conversation_id, turn_id, turn_sequence, chat_role, content, tool_call_id, tool_calls, model, message_state, error_message, prompt_tokens, completion_tokens, total_tokens, created_at, updated_at FROM ai_chat_messages LEFT JOIN ( SELECT created_at AS checkpoint_created_at, id AS checkpoint_id FROM ai_chat_messages WHERE conversation_id = $1 AND id = $2 LIMIT 1 ) checkpoint ON TRUE WHERE conversation_id = $3 AND (checkpoint.checkpoint_id IS NULL OR ai_chat_messages.created_at > checkpoint.checkpoint_created_at OR (ai_chat_messages.created_at = checkpoint.checkpoint_created_at AND ai_chat_messages.id > checkpoint.checkpoint_id)) ORDER BY created_at ASC, id ASC LIMIT 11").
					WithArgs(conversationID, fixedID1, conversationID).
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
			got, hasMore, gotErr := repo.ListChatMessages(context.Background(), conversationID, tt.page, tt.pageSize, tt.options...)
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

func TestChatMessageRepository_DeleteConversationMessages(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tests := map[string]struct {
		expect func(sqlmock.Sqlmock)
		err    error
	}{
		"success": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM ai_chat_messages WHERE conversation_id = $1").
					WithArgs(conversationID).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			err: nil,
		},
		"database-error": {
			expect: func(m sqlmock.Sqlmock) {
				m.ExpectExec("DELETE FROM ai_chat_messages WHERE conversation_id = $1").
					WithArgs(conversationID).
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
			gotErr := repo.DeleteConversationMessages(context.Background(), conversationID)
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
