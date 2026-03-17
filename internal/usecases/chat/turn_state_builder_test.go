package chat

import (
	"context"
	"io"
	"log"
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

func TestTurnStateBuilder_Build(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	summaryRepo := assistant.NewMockConversationSummaryRepository(t)
	chatRepo := assistant.NewMockChatMessageRepository(t)
	skillRegistry := assistant.NewMockSkillRegistry(t)
	actionRegistry := assistant.NewMockActionRegistry(t)
	timeProvider := core.NewMockCurrentTimeProvider(t)

	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	timeProvider.EXPECT().Now().Return(now).Once()
	summaryRepo.EXPECT().
		GetConversationSummary(mock.Anything, conversationID).
		Return(assistant.ConversationSummary{
			CurrentStateSummary: "summary context",
		}, true, nil).
		Once()
	chatRepo.EXPECT().
		ListChatMessages(mock.Anything, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES).
		Return([]assistant.ChatMessage{}, false, nil).
		Once()

	skills := []assistant.SkillDefinition{
		{
			Name:    "todo-skill",
			Tools:   []string{"fetch_todos", "update_todos"},
			Content: "1. Fetch ids first",
		},
	}
	skillRegistry.EXPECT().
		ListRelevant(mock.Anything, mock.MatchedBy(func(query assistant.SkillQueryContext) bool {
			if query.ConversationSummary != "summary context" {
				return false
			}
			if len(query.Messages) != 3 {
				return false
			}
			return query.Messages[2].Role == assistant.ChatRole_User && query.Messages[2].Content == "Update my todos"
		})).
		Return(skills).
		Once()

	actionRegistry.EXPECT().
		GetDefinition("fetch_todos").
		Return(assistant.ActionDefinition{Name: "todo_lookup"}, true).
		Once()
	actionRegistry.EXPECT().
		GetDefinition("update_todos").
		Return(assistant.ActionDefinition{Name: "todo_lookup"}, true).
		Once()

	builder := NewTurnStateBuilderImpl(
		summaryRepo,
		chatRepo,
		timeProvider,
		skillRegistry,
		actionRegistry,
	)

	state, err := builder.Build(t.Context(), BuildSessionParams{
		UserMessage:         "Update my todos",
		Model:               "test-model",
		MaxActionCycles:     7,
		Conversation:        assistant.Conversation{ID: conversationID},
		ConversationCreated: false,
	})
	require.NoError(t, err)
	request := state.Request()
	assert.Equal(t, conversationID, state.Conversation().ID)
	assert.False(t, state.ConversationCreated())
	assert.Len(t, state.SelectedSkills(), 1)
	assert.Equal(t, "todo-skill", state.SelectedSkills()[0].Name)
	assert.Len(t, request.AvailableActions, 1)
	assert.Equal(t, "todo_lookup", request.AvailableActions[0].Name)
	assert.Len(t, request.Messages, 4)
	assert.Equal(t, assistant.ChatRole_User, request.Messages[2].Role)
	assert.Equal(t, "Update my todos", request.Messages[2].Content)
	assert.Equal(t, assistant.ChatRole_System, request.Messages[3].Role)
	assert.True(t, strings.Contains(request.Messages[3].Content, "Skill runbooks for this turn"))
}

func TestStreamChatImpl_CompactIfNeeded(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	fixedTime := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
	compactionPolicy := assistant.CompactionPolicy{
		TriggerTokenCount: 8000,
	}

	tests := map[string]struct {
		setExpectations func(*MockConversationCompactor)
		wantEvents      []assistant.EventType
		wantTimeCalled  bool
		timeout         time.Duration
	}{
		"skips-when-not-triggered": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.CompactionDecision{
					ShouldCompact: false,
					Reason:        assistant.ContextCompactionReasonNone,
				}, nil).Once()
			},
			wantEvents:     []assistant.EventType{},
			wantTimeCalled: false,
		},
		"emits-started-and-completed-when-triggered": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.CompactionDecision{
					ShouldCompact: true,
					Reason:        assistant.ContextCompactionReasonTokenCountThreshold,
					MessageCount:  7,
					TotalTokens:   1200,
				}, nil).Once()
				compactor.EXPECT().Compact(mock.Anything, conversationID).Return(nil).Once()
			},
			wantEvents:     []assistant.EventType{assistant.EventType_ContextCompactionStarted, assistant.EventType_ContextCompactionCompleted},
			wantTimeCalled: true,
		},
		"emits-failed-when-evaluation-errors": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.CompactionDecision{}, assert.AnError).Once()
			},
			wantEvents:     []assistant.EventType{assistant.EventType_ContextCompactionFailed},
			wantTimeCalled: false,
		},
		"emits-started-then-failed-when-compaction-errors": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.CompactionDecision{
					ShouldCompact: true,
					Reason:        assistant.ContextCompactionReasonTokenCountThreshold,
					MessageCount:  2,
					TotalTokens:   990,
				}, nil).Once()
				compactor.EXPECT().Compact(mock.Anything, conversationID).Return(assert.AnError).Once()
			},
			wantEvents:     []assistant.EventType{assistant.EventType_ContextCompactionStarted, assistant.EventType_ContextCompactionFailed},
			wantTimeCalled: false,
		},
		"emits-started-then-failed-when-compaction-times-out": {
			setExpectations: func(compactor *MockConversationCompactor) {
				compactor.EXPECT().EvaluateConversationCompaction(mock.Anything, conversationID, compactionPolicy).Return(assistant.CompactionDecision{
					ShouldCompact: true,
					Reason:        assistant.ContextCompactionReasonTokenCountThreshold,
					MessageCount:  5,
					TotalTokens:   980,
				}, nil).Once()
				compactor.EXPECT().Compact(mock.Anything, conversationID).RunAndReturn(func(ctx context.Context, _ uuid.UUID) error {
					<-ctx.Done()
					return ctx.Err()
				}).Once()
			},
			wantEvents:     []assistant.EventType{assistant.EventType_ContextCompactionStarted, assistant.EventType_ContextCompactionFailed},
			wantTimeCalled: false,
			timeout:        10 * time.Millisecond,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			compactor := NewMockConversationCompactor(t)
			tt.setExpectations(compactor)

			timeProvider := core.NewMockCurrentTimeProvider(t)
			if tt.wantTimeCalled {
				timeProvider.EXPECT().Now().Return(fixedTime).Once()
			}

			useCase := StreamChatImpl{
				logger:                log.New(io.Discard, "", 0),
				timeProvider:          timeProvider,
				conversationCompactor: compactor,
				compactionPolicy:      compactionPolicy,
				compactionTimeout:     tt.timeout,
			}

			gotEvents := make([]assistant.EventType, 0, 2)
			err := useCase.compactIfNeeded(t.Context(), conversationID, func(_ context.Context, eventType assistant.EventType, _ any) error {
				gotEvents = append(gotEvents, eventType)
				return nil
			})

			assert.NoError(t, err)
			assert.Equal(t, tt.wantEvents, gotEvents)
		})
	}
}
