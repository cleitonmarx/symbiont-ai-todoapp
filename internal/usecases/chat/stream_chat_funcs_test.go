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
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/transaction"
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
		*assistant.MockChatMessageRepository,
		*assistant.MockConversationSummaryRepository,
		*assistant.MockConversationRepository,
		*core.MockCurrentTimeProvider,
		*assistant.MockAssistant,
		*assistant.MockActionRegistry,
		*assistant.MockSkillRegistry,
		*transaction.MockUnitOfWork,
		*outbox.MockRepository,
	)
	expectErr       bool
	expectedContent string
	onEventErrType  assistant.EventType
}

// testStreamChatImpl is a helper function that executes the StreamChatImpl use case with the provided test case entry,
// setting up mocks and validating expectations accordingly.
func testStreamChatImpl(t *testing.T, tt streamChatTestTableEntry) {
	t.Helper()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	chatRepo := assistant.NewMockChatMessageRepository(t)
	summaryRepo := assistant.NewMockConversationSummaryRepository(t)
	conversationRepo := assistant.NewMockConversationRepository(t)
	timeProvider := core.NewMockCurrentTimeProvider(t)
	assist := assistant.NewMockAssistant(t)
	actionRegistry := assistant.NewMockActionRegistry(t)
	skillRegistry := assistant.NewMockSkillRegistry(t)
	uow := transaction.NewMockUnitOfWork(t)
	outboxRepo := outbox.NewMockRepository(t)
	if strings.TrimSpace(tt.userMessage) != "" && tt.model != "" && !tt.customSummaryExpectation {
		summaryRepo.EXPECT().
			GetConversationSummary(mock.Anything, conversationID).
			Return(assistant.ConversationSummary{}, false, nil).
			Once()
	}

	if tt.setExpectations != nil {
		tt.setExpectations(chatRepo, summaryRepo, conversationRepo, timeProvider, assist, actionRegistry, skillRegistry, uow, outboxRepo)
	}

	actionRegistry.EXPECT().
		GetRenderer(mock.Anything).
		Return(nil, false).
		Maybe()

	useCase := NewStreamChatImpl(
		log.New(io.Discard, "", 0),
		chatRepo,
		summaryRepo,
		conversationRepo,
		timeProvider,
		assist,
		actionRegistry,
		skillRegistry,
		nil,
		uow,
		"test-embedding-model",
		7,
	)

	var capturedContent string
	err := useCase.Execute(context.Background(), tt.userMessage, tt.model, func(_ context.Context, eventType assistant.EventType, data any) error {
		if tt.onEventErrType != "" && eventType == tt.onEventErrType {
			return errors.New("onEvent error")
		}
		if eventType == assistant.EventType_MessageDelta {
			delta := data.(assistant.MessageDelta)
			capturedContent += delta.Text
		}
		if eventType == assistant.EventType_ActionStarted {
			actionCall := data.(assistant.ActionCall)
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

// actionFunctionCallback returns a mock assistant callback that simulates a
// action call interaction, including meta, delta, and done events.
func actionFunctionCallback(userMsgID, assistantMsgID uuid.UUID, fixedTime time.Time) func(_ context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
	return func(ctx context.Context, req assistant.TurnRequest, onEvent assistant.EventCallback) error {
		if err := onEvent(ctx, assistant.EventType_TurnStarted, assistant.TurnStarted{
			UserMessageID:      userMsgID,
			AssistantMessageID: assistantMsgID,
		}); err != nil {
			return err
		}

		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Content == "Call an action" {
			err := onEvent(ctx, assistant.EventType_ActionRequested, assistant.ActionCall{
				ID:    "func-123",
				Name:  "list_todos",
				Input: `{"page": 1, "page_size": 5, "search_term": "searchTerm"}`,
			})
			return err
		}

		if lastMsg.Role == assistant.ChatRole_Tool {
			if err := onEvent(ctx, assistant.EventType_MessageDelta, assistant.MessageDelta{Text: "Action called successfully."}); err != nil {
				return err
			}
		}

		if err := onEvent(ctx, assistant.EventType_TurnCompleted, assistant.TurnCompleted{
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
	Role                   assistant.ChatRole
	Content                string
	ID                     *uuid.UUID
	MessageState           assistant.ChatMessageState
	ErrorMessage           *string
	ApprovalStatus         *assistant.ChatMessageApprovalStatus
	ApprovalDecisionReason *string
	ApprovalDecidedAt      *time.Time
	PromptTokens           *int
	CompletionTokens       *int
	TotalTokens            *int
	ActionCallsLen         int
	HasActionCallID        bool
	SelectedSkills         []assistant.SelectedSkill
	ActionExecuted         *bool
	FirstActionCallText    *string
	CreateErr              error
}

// expectNowCalls enforces an exact number of current-time reads.
func expectNowCalls(timeProvider *core.MockCurrentTimeProvider, fixedTime time.Time, times int) {
	timeProvider.EXPECT().
		Now().
		Return(fixedTime).
		Times(times)
}

// expectPersistSequence validates message persistence and matching outbox payloads in order.
func expectPersistSequence(
	t *testing.T,
	chatRepo *assistant.MockChatMessageRepository,
	conversationRepo *assistant.MockConversationRepository,
	uow *transaction.MockUnitOfWork,
	outboxRepo *outbox.MockRepository,
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

	scope := transaction.NewMockScope(t)
	scope.EXPECT().ChatMessage().Return(chatRepo).Times(len(expectations))
	scope.EXPECT().Conversation().Return(conversationRepo).Times(successCount)
	scope.EXPECT().Outbox().Return(outboxRepo).Times(successCount)

	uow.EXPECT().
		Execute(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context, transaction.Scope) error) error {
			return fn(ctx, scope)
		}).
		Times(len(expectations))

	var (
		createIdx         int
		successfulMessage []assistant.ChatMessage
	)

	chatRepo.EXPECT().
		CreateChatMessages(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, msgs []assistant.ChatMessage) error {
			assert.Len(t, msgs, 1)
			msg := msgs[0]
			exp := expectations[createIdx]
			createIdx++

			expectedState := exp.MessageState
			if expectedState == "" {
				expectedState = assistant.ChatMessageState_Completed
			}

			assert.Equal(t, exp.Role, msg.ChatRole)
			assert.Equal(t, exp.Content, msg.Content)
			assert.Equal(t, expectedState, msg.MessageState)
			assert.Equal(t, exp.HasActionCallID, msg.ActionCallID != nil)
			assert.Len(t, msg.ActionCalls, exp.ActionCallsLen)
			assert.ElementsMatch(t, exp.SelectedSkills, msg.SelectedSkills)
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
			if exp.ApprovalStatus != nil {
				if assert.NotNil(t, msg.ApprovalStatus) {
					assert.Equal(t, *exp.ApprovalStatus, *msg.ApprovalStatus)
				}
			} else {
				assert.Nil(t, msg.ApprovalStatus)
			}
			if exp.ApprovalDecisionReason != nil {
				if assert.NotNil(t, msg.ApprovalDecisionReason) {
					assert.Equal(t, *exp.ApprovalDecisionReason, *msg.ApprovalDecisionReason)
				}
			} else {
				assert.Nil(t, msg.ApprovalDecisionReason)
			}
			if exp.ApprovalDecidedAt != nil {
				if assert.NotNil(t, msg.ApprovalDecidedAt) {
					assert.Equal(t, *exp.ApprovalDecidedAt, *msg.ApprovalDecidedAt)
				}
			} else {
				assert.Nil(t, msg.ApprovalDecidedAt)
			}
			if exp.ActionExecuted != nil {
				if assert.NotNil(t, msg.ActionExecuted) {
					assert.Equal(t, *exp.ActionExecuted, *msg.ActionExecuted)
				}
			}
			if exp.FirstActionCallText != nil {
				if assert.NotEmpty(t, msg.ActionCalls) {
					assert.Equal(t, *exp.FirstActionCallText, msg.ActionCalls[0].Text)
				}
			}

			if exp.CreateErr == nil {
				successfulMessage = append(successfulMessage, msg)
			}

			return exp.CreateErr
		}).
		Times(len(expectations))

	outboxCallIndex := 0
	outboxRepo.EXPECT().
		CreateChatEvent(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, event outbox.ChatMessageEvent) error {
			msg := successfulMessage[outboxCallIndex]
			outboxCallIndex++

			assert.Equal(t, outbox.EventType_CHAT_MESSAGE_SENT, event.Type)
			assert.Equal(t, msg.ChatRole, event.ChatRole)
			assert.Equal(t, msg.ID, event.ChatMessageID)
			assert.Equal(t, msg.ConversationID, event.ConversationID)

			return nil
		}).
		Times(successCount)

	conversationRepo.EXPECT().
		UpdateConversation(mock.Anything, mock.MatchedBy(func(conv assistant.Conversation) bool {
			return conv.LastMessageAt != nil && conv.UpdatedAt.Equal(fixedTime)
		})).
		Return(nil).
		Times(successCount)
}
