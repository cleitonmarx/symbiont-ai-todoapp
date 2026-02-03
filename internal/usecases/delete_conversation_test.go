package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteConversationImpl_Execute(t *testing.T) {
	tests := map[string]struct {
		setExpectations func(repo *domain.MockChatMessageRepository)
		expectedErr     error
	}{
		"success": {
			expectedErr: nil,
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					DeleteConversation(mock.Anything).
					Return(nil).
					Once()
			},
		},
		"repository-error": {
			expectedErr: errors.New("database error"),
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					DeleteConversation(mock.Anything).
					Return(errors.New("database error")).
					Once()
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain.NewMockChatMessageRepository(t)
			tt.setExpectations(repo)

			uc := NewDeleteConversationImpl(repo)
			err := uc.Execute(context.Background())

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestInitDeleteConversation_Initialize(t *testing.T) {
	idc := InitDeleteConversation{}

	_, err := idc.Initialize(context.Background())
	assert.NoError(t, err)

	uc, err := depend.Resolve[DeleteConversation]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)

}
