package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListChatMessagesImpl_Query(t *testing.T) {
	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tests := map[string]struct {
		setExpectations func(repo *domain.MockChatMessageRepository)
		page            int
		pageSize        int
		expectedLen     int
		expectedHasMore bool
		expectedErr     error
	}{
		"success-with-more-available": {
			page:            1,
			pageSize:        50,
			expectedLen:     50,
			expectedHasMore: true,
			expectedErr:     nil,
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 50).
					Return(createChatMessages(50, domain.ChatRole_User), true, nil).
					Once()
			},
		},
		"success-without-more-available": {
			page:            3,
			pageSize:        50,
			expectedLen:     30,
			expectedHasMore: false,
			expectedErr:     nil,
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 3, 50).
					Return(createChatMessages(30, domain.ChatRole_Assistant), false, nil).
					Once()
			},
		},
		"repository-error": {
			page:            1,
			pageSize:        50,
			expectedLen:     0,
			expectedHasMore: false,
			expectedErr:     errors.New("database error"),
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 50).
					Return(nil, false, errors.New("database error")).
					Once()
			},
		},
		"empty-result-set": {
			page:            1,
			pageSize:        50,
			expectedLen:     0,
			expectedHasMore: false,
			expectedErr:     nil,
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 50).
					Return([]domain.ChatMessage{}, false, nil).
					Once()
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain.NewMockChatMessageRepository(t)
			tt.setExpectations(repo)

			uc := NewListChatMessagesImpl(repo)
			got, hasMore, gotErr := uc.Query(context.Background(), conversationID, tt.page, tt.pageSize)

			assert.Equal(t, tt.expectedErr, gotErr)
			assert.Equal(t, tt.expectedLen, len(got))
			assert.Equal(t, tt.expectedHasMore, hasMore)
		})
	}
}

func createChatMessages(count int, role domain.ChatRole) []domain.ChatMessage {
	messages := make([]domain.ChatMessage, count)
	for i := range count {
		messages[i] = domain.ChatMessage{
			ChatRole: role,
			Content:  "Test message",
		}
	}
	return messages
}

func TestInitListChatMessages_Initialize(t *testing.T) {
	idc := InitListChatMessages{}

	_, err := idc.Initialize(context.Background())
	assert.NoError(t, err)

	uc, err := depend.Resolve[ListChatMessages]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)

}
