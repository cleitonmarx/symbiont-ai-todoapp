package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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

func TestConversation_CanBeLLMRetitled(t *testing.T) {
	assert.True(t, Conversation{TitleSource: ConversationTitleSource_Auto}.CanBeLLMRetitled())
	assert.False(t, Conversation{TitleSource: ConversationTitleSource_User}.CanBeLLMRetitled())
	assert.False(t, Conversation{TitleSource: ConversationTitleSource_LLM}.CanBeLLMRetitled())
}

func TestConversation_ApplyUserTitle(t *testing.T) {
	base := Conversation{
		ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Title:       "Auto title",
		TitleSource: ConversationTitleSource_Auto,
	}

	tests := map[string]struct {
		title      string
		wantErr    string
		wantTitle  string
		wantSource ConversationTitleSource
	}{
		"success-sets-user-source": {
			title:      "Plan Spring Cleaning",
			wantErr:    "",
			wantTitle:  "Plan Spring Cleaning",
			wantSource: ConversationTitleSource_User,
		},
		"empty-title-validation-error": {
			title:      "",
			wantErr:    "conversation title cannot be empty",
			wantTitle:  "Auto title",
			wantSource: ConversationTitleSource_Auto,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			conv := base
			err := conv.ApplyUserTitle(tt.title)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantTitle, conv.Title)
			assert.Equal(t, tt.wantSource, conv.TitleSource)
		})
	}
}

func TestConversation_ApplyLLMGeneratedTitle(t *testing.T) {
	longText := strings.Repeat("a", 80)
	base := Conversation{
		ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Title:       "I invited my Friend to...",
		TitleSource: ConversationTitleSource_Auto,
	}
	summary := "current_intent: Host dinner with friend"

	tests := map[string]struct {
		conversation Conversation
		rawTitle     string
		summary      string
		wantStatus   ConversationTitleApplyStatus
		wantTitle    string
		wantSource   ConversationTitleSource
	}{
		"success-update": {
			conversation: base,
			rawTitle:     "\"Dinner Hosting Plan\"",
			summary:      summary,
			wantStatus:   ConversationTitleApplyStatus_Updated,
			wantTitle:    "Dinner Hosting Plan",
			wantSource:   ConversationTitleSource_LLM,
		},
		"skip-non-auto-title-source": {
			conversation: Conversation{
				ID:          base.ID,
				Title:       "Manual title",
				TitleSource: ConversationTitleSource_User,
			},
			rawTitle:   "Dinner Hosting Plan",
			summary:    summary,
			wantStatus: ConversationTitleApplyStatus_SkippedNotEligible,
		},
		"skip-off-topic-title": {
			conversation: base,
			rawTitle:     "Weekly meeting agenda",
			summary:      summary,
			wantStatus:   ConversationTitleApplyStatus_SkippedNotGrounded,
		},
		"skip-empty-title": {
			conversation: base,
			rawTitle:     "New Conversation",
			summary:      summary,
			wantStatus:   ConversationTitleApplyStatus_SkippedEmpty,
		},
		"skip-empty-raw-title": {
			conversation: base,
			rawTitle:     "",
			summary:      summary,
			wantStatus:   ConversationTitleApplyStatus_SkippedEmpty,
		},
		"skip-unchanged-title": {
			conversation: Conversation{
				ID:          base.ID,
				Title:       "Dinner hosting plan",
				TitleSource: ConversationTitleSource_Auto,
			},
			rawTitle:   "Dinner Hosting Plan",
			summary:    summary,
			wantStatus: ConversationTitleApplyStatus_SkippedUnchanged,
		},
		"strip-markdown-and-list-prefix": {
			conversation: base,
			rawTitle:     "- **Trip Planning**",
			summary:      "current_intent: Trip planning for spring break",
			wantStatus:   ConversationTitleApplyStatus_Updated,
			wantTitle:    "Trip Planning",
			wantSource:   ConversationTitleSource_LLM,
		},
		"reject-multiline-title": {
			conversation: base,
			rawTitle:     "Review project timeline\nUpdate client contact info",
			summary:      "current_intent: Review project timeline",
			wantStatus:   ConversationTitleApplyStatus_SkippedEmpty,
		},
		"clamp-verbose-title-to-short-form": {
			conversation: base,
			rawTitle:     "Japan Trip planning with research flights accommodation visa checklist and packing",
			summary:      "current_intent: Japan trip planning",
			wantStatus:   ConversationTitleApplyStatus_Updated,
			wantTitle:    "Japan Trip planning with research flights accommodation",
			wantSource:   ConversationTitleSource_LLM,
		},
		"trim-trailing-punctuation": {
			conversation: base,
			rawTitle:     "Project cleanup!!!",
			summary:      "current_intent: Project cleanup before release",
			wantStatus:   ConversationTitleApplyStatus_Updated,
			wantTitle:    "Project cleanup",
			wantSource:   ConversationTitleSource_LLM,
		},
		"enforce-max-title-length": {
			conversation: base,
			rawTitle:     longText,
			summary:      "",
			wantStatus:   ConversationTitleApplyStatus_Updated,
			wantTitle:    strings.Repeat("a", 72),
			wantSource:   ConversationTitleSource_LLM,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			updated := tt.conversation
			status := updated.ApplyLLMGeneratedTitle(tt.rawTitle, tt.summary)
			assert.Equal(t, tt.wantStatus, status)

			if tt.wantStatus != ConversationTitleApplyStatus_Updated {
				assert.Equal(t, tt.conversation.Title, updated.Title)
				assert.Equal(t, tt.conversation.TitleSource, updated.TitleSource)
				return
			}

			assert.Equal(t, tt.wantTitle, updated.Title)
			assert.Equal(t, tt.wantSource, updated.TitleSource)
		})
	}
}
