package chat

import (
	"context"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestActionPipeline_Handle_SuccessWithRenderer(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 14, 14, 0, 0, 0, time.UTC)
	actionRegistry := assistant.NewMockActionRegistry(t)
	renderer := assistant.NewMockActionResultRenderer(t)
	transcriptWriter := NewMockConversationTranscriptWriter(t)
	timeProvider := core.NewMockCurrentTimeProvider(t)

	actionRegistry.EXPECT().StatusMessage("list_todos").Return("Listing todos").Once()
	actionRegistry.EXPECT().
		Execute(mock.Anything, assistant.ActionCall{
			ID:    "call-1",
			Name:  "list_todos",
			Input: `{"page":1}`,
			Text:  "Listing todos",
		}, mock.Anything).
		Return(assistant.Message{
			Role:         assistant.ChatRole_Tool,
			Content:      `{"items":["a","b"]}`,
			ActionCallID: common.Ptr("call-1"),
		}).
		Once()
	actionRegistry.EXPECT().
		GetRenderer("list_todos").
		Return(renderer, true).
		Once()
	renderer.EXPECT().
		Render(
			assistant.ActionCall{ID: "call-1", Name: "list_todos", Input: `{"page":1}`, Text: "Listing todos"},
			assistant.Message{
				Role:         assistant.ChatRole_Tool,
				Content:      `{"items":["a","b"]}`,
				ActionCallID: common.Ptr("call-1"),
			},
		).
		Return(assistant.Message{Role: assistant.ChatRole_Assistant, Content: "Found 2 todos."}, true).
		Once()

	pipeline := NewActionPipelineImpl(
		actionRegistry,
		nil,
		transcriptWriter,
		timeProvider,
	)

	state := NewTurnState(
		assistant.Conversation{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
		false,
		nil,
		assistant.TurnRequest{
			Model:    "test-model",
			Messages: []assistant.Message{{Role: assistant.ChatRole_User, Content: "List todos"}},
		},
		7,
	)

	var persistedMessages []assistant.ChatMessage
	timeProvider.EXPECT().Now().Return(fixedTime).Twice()
	transcriptWriter.EXPECT().
		WriteMessage(mock.Anything, state.Conversation(), mock.Anything).
		Run(func(_ context.Context, _ assistant.Conversation, message assistant.ChatMessage) {
			persistedMessages = append(persistedMessages, message)
		}).
		Return(nil).
		Twice()

	var eventTypes []assistant.EventType
	continueStreaming, err := pipeline.Handle(
		t.Context(),
		assistant.ActionCall{ID: "call-1", Name: "list_todos", Input: `{"page":1}`},
		state,
		func(_ context.Context, eventType assistant.EventType, _ any) error {
			eventTypes = append(eventTypes, eventType)
			return nil
		},
	)

	require.NoError(t, err)
	assert.True(t, continueStreaming)
	assert.Len(t, persistedMessages, 2)
	assert.Equal(t, assistant.ChatRole_Assistant, persistedMessages[0].ChatRole)
	assert.Equal(t, assistant.ChatRole_Tool, persistedMessages[1].ChatRole)
	assert.Equal(t, []assistant.EventType{
		assistant.EventType_ActionStarted,
		assistant.EventType_ActionCompleted,
		assistant.EventType_MessageDelta,
	}, eventTypes)
	assert.Equal(t, "Found 2 todos.", state.AssistantContent())
	request := state.Request()
	assert.Len(t, request.Messages, 4)
}
