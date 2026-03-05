package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListChatMessagesImpl_Query(t *testing.T) {
	t.Parallel()

	conversationID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	turnID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	userMsgID := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	actionMsgID := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	toolMsgID := uuid.MustParse("40000000-0000-0000-0000-000000000001")
	assistantMsgID := uuid.MustParse("50000000-0000-0000-0000-000000000001")
	fixedTime := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	approvalStatus := domain.ChatMessageApprovalStatus_Approved
	actionExecuted := true

	tests := map[string]struct {
		setExpectations  func(repo *domain.MockChatMessageRepository)
		page             int
		pageSize         int
		expectedMessages []domain.ChatMessage
		expectedHasMore  bool
		expectedErr      error
	}{
		"projects-turn-messages-and-folds-action-details": {
			page:     1,
			pageSize: 50,
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 50).
					Return([]domain.ChatMessage{
						{
							ID:        userMsgID,
							TurnID:    turnID,
							ChatRole:  domain.ChatRole_User,
							Content:   "Delete todo 1",
							CreatedAt: fixedTime,
							UpdatedAt: fixedTime,
						},
						{
							ID:       actionMsgID,
							TurnID:   turnID,
							ChatRole: domain.ChatRole_Assistant,
							ActionCalls: []domain.AssistantActionCall{
								{
									ID:    "call-1",
									Name:  "delete_todos",
									Input: `{"todos":[{"id":"1"}]}`,
									Text:  "Deleting todos...",
								},
							},
							CreatedAt: fixedTime.Add(time.Second),
							UpdatedAt: fixedTime.Add(time.Second),
						},
						{
							ID:                     toolMsgID,
							TurnID:                 turnID,
							ChatRole:               domain.ChatRole_Tool,
							ActionCallID:           common.Ptr("call-1"),
							Content:                "todo deleted",
							MessageState:           domain.ChatMessageState_Completed,
							ApprovalStatus:         &approvalStatus,
							ApprovalDecisionReason: common.Ptr("approved by user"),
							ApprovalDecidedAt:      common.Ptr(fixedTime.Add(2 * time.Second)),
							ActionExecuted:         &actionExecuted,
							CreatedAt:              fixedTime.Add(2 * time.Second),
							UpdatedAt:              fixedTime.Add(2 * time.Second),
						},
						{
							ID:             assistantMsgID,
							TurnID:         turnID,
							ChatRole:       domain.ChatRole_Assistant,
							Content:        "Done.",
							SelectedSkills: []domain.AssistantSelectedSkill{{Name: "delete_todos", Source: "skills/delete_todos.md", Tools: []string{"fetch_todos", "delete_todos"}}},
							CreatedAt:      fixedTime.Add(3 * time.Second),
							UpdatedAt:      fixedTime.Add(3 * time.Second),
						},
					}, false, nil).
					Once()
			},
			expectedMessages: []domain.ChatMessage{
				{
					ID:        userMsgID,
					TurnID:    turnID,
					ChatRole:  domain.ChatRole_User,
					Content:   "Delete todo 1",
					CreatedAt: fixedTime,
					UpdatedAt: fixedTime,
				},
				{
					ID:             assistantMsgID,
					TurnID:         turnID,
					ChatRole:       domain.ChatRole_Assistant,
					Content:        "Done.",
					SelectedSkills: []domain.AssistantSelectedSkill{{Name: "delete_todos", Source: "skills/delete_todos.md", Tools: []string{"fetch_todos", "delete_todos"}}},
					ActionDetails: []domain.ChatMessageActionDetail{
						{
							ActionCallID:           "call-1",
							Name:                   "delete_todos",
							Input:                  `{"todos":[{"id":"1"}]}`,
							Text:                   "Deleting todos...",
							Output:                 "todo deleted",
							MessageState:           domain.ChatMessageState_Completed,
							ApprovalStatus:         &approvalStatus,
							ApprovalDecisionReason: common.Ptr("approved by user"),
							ApprovalDecidedAt:      common.Ptr(fixedTime.Add(2 * time.Second)),
							ActionExecuted:         &actionExecuted,
						},
					},
					CreatedAt: fixedTime.Add(3 * time.Second),
					UpdatedAt: fixedTime.Add(3 * time.Second),
				},
			},
			expectedHasMore: false,
			expectedErr:     nil,
		},
		"returns-failed-assistant-message-even-with-empty-content": {
			page:     1,
			pageSize: 50,
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 50).
					Return([]domain.ChatMessage{
						{
							ID:        userMsgID,
							TurnID:    turnID,
							ChatRole:  domain.ChatRole_User,
							Content:   "Delete todo 1",
							CreatedAt: fixedTime,
							UpdatedAt: fixedTime,
						},
						{
							ID:           assistantMsgID,
							TurnID:       turnID,
							ChatRole:     domain.ChatRole_Assistant,
							MessageState: domain.ChatMessageState_Failed,
							CreatedAt:    fixedTime.Add(time.Second),
							UpdatedAt:    fixedTime.Add(time.Second),
						},
					}, false, nil).
					Once()
			},
			expectedMessages: []domain.ChatMessage{
				{
					ID:        userMsgID,
					TurnID:    turnID,
					ChatRole:  domain.ChatRole_User,
					Content:   "Delete todo 1",
					CreatedAt: fixedTime,
					UpdatedAt: fixedTime,
				},
				{
					ID:           assistantMsgID,
					TurnID:       turnID,
					ChatRole:     domain.ChatRole_Assistant,
					MessageState: domain.ChatMessageState_Failed,
					CreatedAt:    fixedTime.Add(time.Second),
					UpdatedAt:    fixedTime.Add(time.Second),
				},
			},
			expectedHasMore: false,
			expectedErr:     nil,
		},
		"repository-error": {
			page:     1,
			pageSize: 50,
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 50).
					Return(nil, false, errors.New("database error")).
					Once()
			},
			expectedMessages: nil,
			expectedHasMore:  false,
			expectedErr:      errors.New("database error"),
		},
		"empty-result-set": {
			page:     1,
			pageSize: 50,
			setExpectations: func(repo *domain.MockChatMessageRepository) {
				repo.EXPECT().
					ListChatMessages(mock.Anything, conversationID, 1, 50).
					Return([]domain.ChatMessage{}, false, nil).
					Once()
			},
			expectedMessages: nil,
			expectedHasMore:  false,
			expectedErr:      nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			repo := domain.NewMockChatMessageRepository(t)
			tt.setExpectations(repo)

			uc := NewListChatMessagesImpl(repo)
			got, hasMore, gotErr := uc.Query(context.Background(), conversationID, tt.page, tt.pageSize)

			assert.Equal(t, tt.expectedErr, gotErr)
			assert.Equal(t, tt.expectedMessages, got)
			assert.Equal(t, tt.expectedHasMore, hasMore)
		})
	}
}

func TestInitListChatMessages_Initialize(t *testing.T) {
	t.Parallel()

	idc := InitListChatMessages{}

	_, err := idc.Initialize(context.Background())
	assert.NoError(t, err)

	uc, err := depend.Resolve[ListChatMessages]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)
}
