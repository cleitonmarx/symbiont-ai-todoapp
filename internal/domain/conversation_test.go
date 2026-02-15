package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAutoConversationTitle(t *testing.T) {
	tests := map[string]struct {
		userMessage string
		want        string
	}{
		"empty-message": {
			userMessage: "",
			want:        "New Conversation",
		},
		"single-word": {
			userMessage: "Hello",
			want:        "Hello",
		},
		"more-than-five-words": {
			userMessage: "Can you help me with this task please",
			want:        "Can you help me with...",
		},
		"many-words": {
			userMessage: "I need to finish the project report by tomorrow and I want it to be perfect",
			want:        "I need to finish the...",
		},
		"whitespace-only": {
			userMessage: "   ",
			want:        "New Conversation",
		},
		"multiple-spaces-between-words": {
			userMessage: "Hello    world    test",
			want:        "Hello world test",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := GenerateAutoConversationTitle(tt.userMessage)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConversation_Validate(t *testing.T) {
	now := time.Now()
	validID := uuid.New()

	tests := map[string]struct {
		conversation Conversation
		wantErr      bool
		errMsg       string
	}{
		"valid-conversation": {
			conversation: Conversation{
				ID:          validID,
				Title:       "Test Conversation",
				TitleSource: ConversationTitleSource_User,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: false,
		},
		"valid-with-llm-source": {
			conversation: Conversation{
				ID:          validID,
				Title:       "Generated Title",
				TitleSource: ConversationTitleSource_LLM,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: false,
		},
		"valid-with-auto-source": {
			conversation: Conversation{
				ID:          validID,
				Title:       "Auto Generated",
				TitleSource: ConversationTitleSource_Auto,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: false,
		},
		"empty-title": {
			conversation: Conversation{
				ID:          validID,
				Title:       "",
				TitleSource: ConversationTitleSource_User,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: true,
			errMsg:  "conversation title cannot be empty",
		},
		"invalid-title-source": {
			conversation: Conversation{
				ID:          validID,
				Title:       "Test",
				TitleSource: ConversationTitleSource("invalid"),
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: true,
			errMsg:  "invalid conversation title source",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.conversation.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
