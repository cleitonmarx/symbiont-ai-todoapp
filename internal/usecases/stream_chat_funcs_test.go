package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// streamChatTestTableEntry defines the structure for test cases of StreamChatImpl's Execute method,
// including input parameters, expectations, and error scenarios.
type streamChatTestTableEntry struct {
	userMessage              string
	model                    string
	customSummaryExpectation bool
	options                  []StreamChatOption
	setExpectations          func(
		*domain.MockChatMessageRepository,
		*domain.MockConversationSummaryRepository,
		*domain.MockConversationRepository,
		*domain.MockCurrentTimeProvider,
		*domain.MockAssistant,
		*domain.MockAssistantActionRegistry,
		*domain.MockUnitOfWork,
		*domain.MockOutboxRepository,
	)
	expectErr       bool
	expectedContent string
	onEventErrType  domain.AssistantEventType
}

// testStreamChatImpl is a helper function that executes the StreamChatImpl use case with the provided test case entry,
// setting up mocks and validating expectations accordingly.
func testStreamChatImpl(t *testing.T, tt streamChatTestTableEntry) {
	t.Helper()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	chatRepo := domain.NewMockChatMessageRepository(t)
	summaryRepo := domain.NewMockConversationSummaryRepository(t)
	conversationRepo := domain.NewMockConversationRepository(t)
	timeProvider := domain.NewMockCurrentTimeProvider(t)
	assistant := domain.NewMockAssistant(t)
	actionRegistry := domain.NewMockAssistantActionRegistry(t)
	uow := domain.NewMockUnitOfWork(t)
	outbox := domain.NewMockOutboxRepository(t)

	if strings.TrimSpace(tt.userMessage) != "" && tt.model != "" && !tt.customSummaryExpectation {
		summaryRepo.EXPECT().
			GetConversationSummary(mock.Anything, conversationID).
			Return(domain.ConversationSummary{}, false, nil).
			Once()
	}

	if tt.setExpectations != nil {
		tt.setExpectations(chatRepo, summaryRepo, conversationRepo, timeProvider, assistant, actionRegistry, uow, outbox)
	}

	useCase := NewStreamChatImpl(
		chatRepo,
		summaryRepo,
		conversationRepo,
		timeProvider,
		assistant,
		actionRegistry,
		uow,
		"test-embedding-model",
		7,
	)

	var capturedContent string
	err := useCase.Execute(context.Background(), tt.userMessage, tt.model, func(eventType domain.AssistantEventType, data any) error {
		if tt.onEventErrType != "" && eventType == tt.onEventErrType {
			return errors.New("onEvent error")
		}
		if eventType == domain.AssistantEventType_MessageDelta {
			delta := data.(domain.AssistantMessageDelta)
			capturedContent += delta.Text
		}
		if eventType == domain.AssistantEventType_ActionStarted {
			actionCall := data.(domain.AssistantActionCall)
			capturedContent += actionCall.Text
		}
		return nil
	}, tt.options...)

	if tt.expectErr {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
		if tt.expectedContent != "" {
			assert.Equal(t, tt.expectedContent, capturedContent)
		}
	}

}

// toolFunctionCallback returns a mock assistant callback that simulates a
// tool call interaction, including meta, delta, and done events.
func toolFunctionCallback(userMsgID, assistantMsgID uuid.UUID, fixedTime time.Time) func(_ context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
	return func(ctx context.Context, req domain.AssistantTurnRequest, onEvent domain.AssistantEventCallback) error {
		if err := onEvent(domain.AssistantEventType_TurnStarted, domain.AssistantTurnStarted{
			UserMessageID:      userMsgID,
			AssistantMessageID: assistantMsgID,
		}); err != nil {
			return err
		}

		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Content == "Call a tool" {
			err := onEvent(domain.AssistantEventType_ActionRequested, domain.AssistantActionCall{
				ID:    "func-123",
				Name:  "list_todos",
				Input: `{"page": 1, "page_size": 5, "search_term": "searchTerm"}`,
			})
			return err
		}

		if lastMsg.Role == domain.ChatRole_Tool {
			if err := onEvent(domain.AssistantEventType_MessageDelta, domain.AssistantMessageDelta{Text: "Tool called successfully."}); err != nil {
				return err
			}
		}

		if err := onEvent(domain.AssistantEventType_TurnCompleted, domain.AssistantTurnCompleted{
			AssistantMessageID: assistantMsgID.String(),
			CompletedAt:        fixedTime.Format(time.RFC3339),
		}); err != nil {
			return err
		}
		return nil
	}
}

// persistCallExpectation describes one expected CreateChatMessages call.
type persistCallExpectation struct {
	Role             domain.ChatRole
	Content          string
	ID               *uuid.UUID
	MessageState     domain.ChatMessageState
	ErrorMessage     *string
	PromptTokens     *int
	CompletionTokens *int
	TotalTokens      *int
	ActionCallsLen   int
	HasActionCallID  bool
	CreateErr        error
}

// expectNowCalls enforces an exact number of current-time reads.
func expectNowCalls(timeProvider *domain.MockCurrentTimeProvider, fixedTime time.Time, times int) {
	timeProvider.EXPECT().
		Now().
		Return(fixedTime).
		Times(times)
}

// expectPersistSequence validates message persistence and matching outbox payloads in order.
func expectPersistSequence(
	t *testing.T,
	chatRepo *domain.MockChatMessageRepository,
	conversationRepo *domain.MockConversationRepository,
	uow *domain.MockUnitOfWork,
	outbox *domain.MockOutboxRepository,
	fixedTime time.Time,
	expectations []persistCallExpectation,
) {
	t.Helper()

	successCount := 0
	for _, exp := range expectations {
		if exp.CreateErr == nil {
			successCount++
		}
	}

	uow.EXPECT().
		Execute(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(uow domain.UnitOfWork) error) error {
			return fn(uow)
		}).
		Times(len(expectations))

	uow.EXPECT().
		ChatMessage().
		Return(chatRepo).
		Times(len(expectations))

	uow.EXPECT().
		Outbox().
		Return(outbox).
		Times(successCount)

	uow.EXPECT().
		Conversation().
		Return(conversationRepo).
		Times(successCount)

	var (
		createIdx         int
		successfulMessage []domain.ChatMessage
	)

	chatRepo.EXPECT().
		CreateChatMessages(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, msgs []domain.ChatMessage) error {
			assert.Len(t, msgs, 1)
			msg := msgs[0]
			exp := expectations[createIdx]
			createIdx++

			expectedState := exp.MessageState
			if expectedState == "" {
				expectedState = domain.ChatMessageState_Completed
			}

			assert.Equal(t, exp.Role, msg.ChatRole)
			assert.Equal(t, exp.Content, msg.Content)
			assert.Equal(t, expectedState, msg.MessageState)
			assert.Equal(t, exp.HasActionCallID, msg.ActionCallID != nil)
			assert.Len(t, msg.ActionCalls, exp.ActionCallsLen)
			assert.NotEqual(t, uuid.Nil, msg.TurnID)
			assert.Equal(t, int64(createIdx-1), msg.TurnSequence)
			assert.Equal(t, fixedTime, msg.CreatedAt)
			assert.Equal(t, fixedTime, msg.UpdatedAt)
			if exp.PromptTokens != nil {
				assert.Equal(t, *exp.PromptTokens, msg.PromptTokens)
			}
			if exp.CompletionTokens != nil {
				assert.Equal(t, *exp.CompletionTokens, msg.CompletionTokens)
			}
			if exp.TotalTokens != nil {
				assert.Equal(t, *exp.TotalTokens, msg.TotalTokens)
			}

			if exp.ID != nil {
				assert.Equal(t, *exp.ID, msg.ID)
			} else {
				assert.NotEqual(t, uuid.Nil, msg.ID)
			}

			if exp.ErrorMessage != nil {
				if assert.NotNil(t, msg.ErrorMessage) {
					assert.Equal(t, *exp.ErrorMessage, *msg.ErrorMessage)
				}
			} else {
				assert.Nil(t, msg.ErrorMessage)
			}

			if exp.CreateErr == nil {
				successfulMessage = append(successfulMessage, msg)
			}

			return exp.CreateErr
		}).
		Times(len(expectations))

	outboxCallIndex := 0
	outbox.EXPECT().
		CreateChatEvent(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, event domain.ChatMessageEvent) error {
			msg := successfulMessage[outboxCallIndex]
			outboxCallIndex++

			assert.Equal(t, domain.EventType_CHAT_MESSAGE_SENT, event.Type)
			assert.Equal(t, msg.ChatRole, event.ChatRole)
			assert.Equal(t, msg.ID, event.ChatMessageID)
			assert.Equal(t, msg.ConversationID, event.ConversationID)

			return nil
		}).
		Times(successCount)

	conversationRepo.EXPECT().
		UpdateConversation(mock.Anything, mock.MatchedBy(func(conv domain.Conversation) bool {
			return conv.LastMessageAt != nil && conv.UpdatedAt.Equal(fixedTime)
		})).
		Return(nil).
		Times(successCount)
}
