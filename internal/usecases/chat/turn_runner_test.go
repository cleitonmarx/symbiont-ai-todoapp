package chat

import (
	"context"
	"errors"
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

func TestTurnRunner_Run_RetriesAfterRecovery(t *testing.T) {
	t.Parallel()

	assistantClient := assistant.NewMockAssistant(t)
	actionPipeline := NewMockActionPipeline(t)
	conversationCreator := NewMockConversationCreator(t)
	timeProvider := core.NewMockCurrentTimeProvider(t)
	callCount := 0

	assistantClient.EXPECT().
		RunTurn(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, req assistant.TurnRequest, _ assistant.EventCallback) error {
			callCount++
			if callCount == 1 {
				return errors.New("stream failed")
			}
			assert.NotEmpty(t, req.Messages)
			assert.Equal(t, assistant.ChatRole_System, req.Messages[len(req.Messages)-1].Role)
			assert.True(t, strings.Contains(req.Messages[len(req.Messages)-1].Content, "stream failed"))
			return nil
		}).
		Twice()

	runner := newTurnRunner(
		log.New(io.Discard, "", 0),
		assistantClient,
		timeProvider,
		conversationCreator,
		actionPipeline,
	)

	session := NewTurnSession(assistant.Conversation{}, false, "Hello", nil, assistant.TurnRequest{
		Model:    "test-model",
		Messages: []assistant.Message{{Role: assistant.ChatRole_User, Content: "Hello"}},
	}, 7)

	err := runner.Run(t.Context(), session, func(context.Context, assistant.EventType, any) error { return nil })
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestTurnRunner_Run_ProcessesStreamEvents(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 14, 13, 0, 0, 0, time.UTC)
	assistantClient := assistant.NewMockAssistant(t)
	actionPipeline := NewMockActionPipeline(t)
	conversationCreator := NewMockConversationCreator(t)
	timeProvider := core.NewMockCurrentTimeProvider(t)
	runner := newTurnRunner(
		log.New(io.Discard, "", 0),
		assistantClient,
		timeProvider,
		conversationCreator,
		actionPipeline,
	)

	session := NewTurnSession(
		assistant.Conversation{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
		true,
		"Hello",
		nil,
		assistant.TurnRequest{Model: "test-model"},
		7,
	)

	actionPipeline.EXPECT().
		Handle(mock.Anything, assistant.ActionCall{ID: "call-1", Name: "list_todos"}, session, mock.Anything).
		Return(false, nil).
		Once()
	timeProvider.EXPECT().Now().Return(fixedTime).Once()
	conversationCreator.EXPECT().
		CreateMessage(mock.Anything, session.Conversation(), mock.Anything).
		Return(nil).
		Once()
	var request assistant.TurnRequest
	session.UpdateRequest(func(current *assistant.TurnRequest) {
		request = *current
	})
	assistantClient.EXPECT().
		RunTurn(mock.Anything, request, mock.Anything).
		RunAndReturn(func(ctx context.Context, _ assistant.TurnRequest, onEvent assistant.EventCallback) error {
			if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
				UserMessageID:      uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				AssistantMessageID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			}); err != nil {
				return err
			}
			if err := onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "Hello back"}); err != nil {
				return err
			}
			if err := onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{ID: "call-1", Name: "list_todos"}); err != nil {
				return err
			}
			return onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
				Usage: assistant.Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8},
			})
		}).
		Once()

	var turnStarted assistant.TurnStarted
	var eventTypes []assistant.EventType
	err := runner.Run(t.Context(), session, func(_ context.Context, eventType assistant.EventType, data any) error {
		eventTypes = append(eventTypes, eventType)
		if eventType == assistant.EventType_TurnStarted {
			turnStarted = data.(assistant.TurnStarted)
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, session.TurnID(), turnStarted.TurnID)
	assert.Equal(t, "Hello back", session.BuildFinalAssistantMessage(fixedTime).Content)
	assert.Equal(t, 3, session.TokenUsage().PromptTokens)
	assert.Equal(t, 5, session.TokenUsage().CompletionTokens)
	assert.Equal(t, 8, session.TokenUsage().TotalTokens)
	assert.Equal(t, []assistant.EventType{
		assistant.EventType_TurnStarted,
		assistant.EventType_MessageDelta,
	}, eventTypes)
}
