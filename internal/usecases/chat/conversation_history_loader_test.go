package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBuildSkillsPrompt(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		skills       []assistant.SkillDefinition
		expectEmpty  bool
		expectedText []string
		notContains  []string
		maxLen       int
	}{
		"returns-empty-when-no-skills": {
			skills:      nil,
			expectEmpty: true,
		},
		"includes-skill-content-and-tools": {
			skills: []assistant.SkillDefinition{
				{
					Name:      "todo-mutation-safety",
					UseWhen:   "User asks to update todos",
					AvoidWhen: "User only wants summaries",
					Tools:     []string{"update_todos", "fetch_todos", "update_todos"},
					Content:   "1. Fetch ids first\n2. Confirm intent\n3. Mutate",
				},
			},
			expectedText: []string{
				"Skill runbooks for this turn:",
				"Skill: todo-mutation-safety",
				"Use when: User asks to update todos",
				"Avoid when: User only wants summaries",
				"Tools: update_todos, fetch_todos",
				"Workflow:",
				"1. Fetch ids first",
			},
		},
		"ignores-empty-skill-name": {
			skills: []assistant.SkillDefinition{
				{Name: "   ", Content: "ignore me"},
			},
			expectedText: []string{
				"Skill runbooks for this turn:",
			},
			notContains: []string{
				"Skill:",
				"ignore me",
			},
		},
		"truncates-long-prompt": {
			skills: []assistant.SkillDefinition{
				{
					Name:    "verbose_skill",
					UseWhen: strings.Repeat("x", MAX_SKILLS_PROMPT_CHARS),
				},
			},
			maxLen: MAX_SKILLS_PROMPT_CHARS,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := buildSkillsPrompt(tt.skills)

			if tt.expectEmpty {
				assert.Equal(t, "", got)
				return
			}

			for _, expect := range tt.expectedText {
				assert.Contains(t, got, expect)
			}
			for _, denied := range tt.notContains {
				assert.NotContains(t, got, denied)
			}
			if tt.maxLen > 0 {
				assert.LessOrEqual(t, len([]rune(got)), tt.maxLen)
			}
		})
	}
}

func TestCompactToLastMessages(t *testing.T) {
	t.Parallel()

	msgs := []assistant.Message{
		{Role: assistant.ChatRole_User, Content: "one"},
		{Role: assistant.ChatRole_Assistant, Content: "two"},
		{Role: assistant.ChatRole_User, Content: "three"},
	}

	tests := map[string]struct {
		messages  []assistant.Message
		max       int
		wantTexts []string
	}{
		"returns-nil-when-max-non-positive": {
			messages: msgs,
			max:      0,
		},
		"returns-nil-when-empty": {
			max: 2,
		},
		"returns-copy-when-within-limit": {
			messages:  msgs[:2],
			max:       3,
			wantTexts: []string{"one", "two"},
		},
		"returns-last-messages-when-over-limit": {
			messages:  msgs,
			max:       2,
			wantTexts: []string{"two", "three"},
		},
	}

	for name, tt := range tests {

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := compactToLastMessages(tt.messages, tt.max)
			assert.Len(t, got, len(tt.wantTexts))
			for i, want := range tt.wantTexts {
				assert.Equal(t, want, got[i].Content)
			}
			if len(got) > 0 && len(tt.messages) > 0 {
				assert.NotSame(t, &tt.messages[0], &got[0])
			}
		})
	}
}

func TestTruncateToFirstChars(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		max   int
		want  string
	}{
		"returns-empty-when-max-non-positive": {
			input: "hello",
			max:   0,
			want:  "",
		},
		"trims-and-keeps-when-short-enough": {
			input: "  hello  ",
			max:   10,
			want:  "hello",
		},
		"truncates-by-rune": {
			input: "ábcdef",
			max:   3,
			want:  "ábc",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, truncateToFirstChars(tt.input, tt.max))
		})
	}
}

func TestTurnStateBuilder_LoadMessagesHistory(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	checkpointID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	summaryRepo := assistant.NewMockConversationSummaryRepository(t)
	chatRepo := assistant.NewMockChatMessageRepository(t)
	fixedTime := time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC)
	timeProvider := core.NewMockCurrentTimeProvider(t)

	summaryRepo.EXPECT().
		GetConversationSummary(mock.Anything, conversationID).
		Return(assistant.ConversationSummary{
			ConversationID:          conversationID,
			CurrentStateSummary:     "Summary state",
			LastSummarizedMessageID: &checkpointID,
		}, true, nil).
		Once()
	chatRepo.EXPECT().
		ListChatMessages(
			mock.Anything,
			conversationID,
			1,
			MAX_CHAT_HISTORY_MESSAGES,
			mock.MatchedBy(func(options []assistant.ListChatMessagesOption) bool {
				params := assistant.ListChatMessagesParams{}
				for _, opt := range options {
					opt(&params)
				}
				return params.AfterMessageID != nil && *params.AfterMessageID == checkpointID
			}),
		).
		Return([]assistant.ChatMessage{
			{ChatRole: assistant.ChatRole_Tool, Content: "orphan tool"},
			{ChatRole: assistant.ChatRole_User, Content: "Hello"},
			{ChatRole: assistant.ChatRole_Assistant, Content: "Hi"},
		}, false, nil).
		Once()

	timeProvider.EXPECT().Now().Return(fixedTime).Once()

	builder := NewTurnStateBuilderImpl(
		summaryRepo,
		chatRepo,
		timeProvider,
		nil,
		nil,
	)

	messages, summaryContext, err := builder.loadMessagesHistory(context.Background(), conversationID)
	require.NoError(t, err)
	assert.Equal(t, "Summary state", summaryContext)
	require.GreaterOrEqual(t, len(messages), 4)
	assert.Equal(t, assistant.ChatRole_System, messages[0].Role)
	assert.Equal(t, assistant.ChatRole_User, messages[len(messages)-2].Role)
	assert.Equal(t, "Hello", messages[len(messages)-2].Content)
	assert.Equal(t, assistant.ChatRole_Assistant, messages[len(messages)-1].Role)
	assert.Equal(t, "Hi", messages[len(messages)-1].Content)
}
