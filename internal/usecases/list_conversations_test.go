package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListConversationsImpl_Query(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		setExpectations       func(repo *domain.MockConversationRepository)
		page                  int
		pageSize              int
		expectedConversations []domain.Conversation
		expectedHasMore       bool
		expectedErr           error
	}{
		"success": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *domain.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 1, 10).Return([]domain.Conversation{
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Title:       "Conversation 1",
						TitleSource: domain.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
						Title:       "Conversation 2",
						TitleSource: domain.ConversationTitleSource_LLM,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, true, nil)
			},
			expectedConversations: []domain.Conversation{
				{
					ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
					Title:       "Conversation 1",
					TitleSource: domain.ConversationTitleSource_User,
					CreatedAt:   fixedTime,
					UpdatedAt:   fixedTime,
				},
				{
					ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
					Title:       "Conversation 2",
					TitleSource: domain.ConversationTitleSource_LLM,
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
			setExpectations: func(repo *domain.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 1, 5).Return([]domain.Conversation{
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
						Title:       "First Conversation",
						TitleSource: domain.ConversationTitleSource_Auto,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, true, nil)
			},
			expectedConversations: []domain.Conversation{
				{
					ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
					Title:       "First Conversation",
					TitleSource: domain.ConversationTitleSource_Auto,
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
			setExpectations: func(repo *domain.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 3, 5).Return([]domain.Conversation{
					{
						ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"),
						Title:       "Last Conversation",
						TitleSource: domain.ConversationTitleSource_User,
						CreatedAt:   fixedTime,
						UpdatedAt:   fixedTime,
					},
				}, false, nil)
			},
			expectedConversations: []domain.Conversation{
				{
					ID:          uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"),
					Title:       "Last Conversation",
					TitleSource: domain.ConversationTitleSource_User,
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
			setExpectations: func(repo *domain.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 1, 10).Return([]domain.Conversation{}, false, nil)
			},
			expectedConversations: []domain.Conversation{},
			expectedHasMore:       false,
			expectedErr:           nil,
		},
		"repository-error": {
			page:     1,
			pageSize: 10,
			setExpectations: func(repo *domain.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 1, 10).Return(nil, false, errors.New("database error"))
			},
			expectedConversations: nil,
			expectedHasMore:       false,
			expectedErr:           errors.New("database error"),
		},
		"invalid-page-number": {
			page:     0,
			pageSize: 10,
			setExpectations: func(repo *domain.MockConversationRepository) {
				repo.EXPECT().ListConversations(mock.Anything, 0, 10).Return(nil, false, domain.NewValidationErr("page must be greater than 0"))
			},
			expectedConversations: nil,
			expectedHasMore:       false,
			expectedErr:           domain.NewValidationErr("page must be greater than 0"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain.NewMockConversationRepository(t)
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

func TestInitListConversations_Initialize(t *testing.T) {
	ilc := InitListConversations{}

	ctx, err := ilc.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredListConversations, err := depend.Resolve[ListConversations]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredListConversations)
}
