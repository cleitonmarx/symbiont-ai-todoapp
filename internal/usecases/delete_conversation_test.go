package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteConversationImpl_Execute(t *testing.T) {
	tests := map[string]struct {
		setExpectations func(repo *mocks.MockChatMessageRepository)
		expectedErr     error
	}{
		"success": {
			expectedErr: nil,
			setExpectations: func(repo *mocks.MockChatMessageRepository) {
				repo.EXPECT().
					DeleteConversation(mock.Anything).
					Return(nil).
					Once()
			},
		},
		"repository-error": {
			expectedErr: errors.New("database error"),
			setExpectations: func(repo *mocks.MockChatMessageRepository) {
				repo.EXPECT().
					DeleteConversation(mock.Anything).
					Return(errors.New("database error")).
					Once()
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := mocks.NewMockChatMessageRepository(t)
			tt.setExpectations(repo)

			uc := NewDeleteConversationImpl(repo)
			err := uc.Execute(context.Background())

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
