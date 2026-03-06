package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListConversationsImpl_Query(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setExpectations       func(repo *assistant.MockConversationRepository)
		page                  int
		pageSize              int
		expectedConversations []assistant.Conversation
		expectedHasMore       bool
		expectedErr           error
	}{
		"success": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *assistant.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 1, 10).Return([]assistant.Conversation{
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Title:       "Conversation 1",
						TitleSource: assistant.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
						Title:       "Conversation 2",
						TitleSource: assistant.ConversationTitleSource_LLM,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, true, nil)
			},
			expectedConversations: []assistant.Conversation{
				{
					ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
					Title:       "Conversation 1",
					TitleSource: assistant.ConversationTitleSource_User,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
				{
					ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
					Title:       "Conversation 2",
					TitleSource: assistant.ConversationTitleSource_LLM,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
			},
			expectedHasMore: true,
			expectedErr:     nil,
		},
		"success-first-page": {
			page:     1,
			pageSize: 5,
			setExpectations: func(repo *assistant.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 1, 5).Return([]assistant.Conversation{
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Title:       "First Conversation",
						TitleSource: assistant.ConversationTitleSource_Auto,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, true, nil)
			},
			expectedConversations: []assistant.Conversation{
				{
					ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
					Title:       "First Conversation",
					TitleSource: assistant.ConversationTitleSource_Auto,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
			},
			expectedHasMore: true,
			expectedErr:     nil,
		},
		"success-last-page": {
			page:     3,
			pageSize: 5,
			setExpectations: func(repo *assistant.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 3, 5).Return([]assistant.Conversation{
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"),
						Title:       "Last Conversation",
						TitleSource: assistant.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, false, nil)
			},
			expectedConversations: []assistant.Conversation{
				{
					ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"),
					Title:       "Last Conversation",
					TitleSource: assistant.ConversationTitleSource_User,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
			},
			expectedHasMore: false,
			expectedErr:     nil,
		},
		"success-empty-list": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *assistant.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 1, 10).Return([]assistant.Conversation{}, false, nil)
			},
			expectedConversations: []assistant.Conversation{},
			expectedHasMore:       false,
			expectedErr:           nil,
		},
		"repository-error": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *assistant.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 1, 10).Return(nil, false, errors.New("database error"))
			},
			expectedConversations: nil,
			expectedHasMore:       false,
			expectedErr:           errors.New("database error"),
		},
		"invalid-page-number": {
			page:     0,
			pageSize: 10,
			setExpectations: func(repo *assistant.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 0, 10).Return(nil, false, core.NewValidationErr("page must be greater than 0"))
			},
			expectedConversations: nil,
			expectedHasMore:       false,
			expectedErr:           core.NewValidationErr("page must be greater than 0"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := assistant.NewMockConversationRepository(t)
			if tt.setExpectations != nil {
				tt.setExpectations(repo)
			}

			lc := NewListConversationsImpl(repo)

			got, hasMore, gotErr := lc.Query(context.Background(), tt.page, tt.pageSize)
			assert.Equal(t, tt.expectedErr, gotErr)
			assert.Equal(t, tt.expectedConversations, got)
			assert.Equal(t, tt.expectedHasMore, hasMore)
		})
	}
}
