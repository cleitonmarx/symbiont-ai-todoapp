package http

import (
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func toError(err error) gen.ErrorResp {
	errResp := gen.ErrorResp{}
	switch e := err.(type) {
	case *core.ValidationErr:
		errResp.Error.Code = gen.BADREQUEST
		errResp.Error.Message = e.Error()
	case *core.NotFoundErr:
		errResp.Error.Code = gen.NOTFOUND
		errResp.Error.Message = e.Error()
	default:
		errResp.Error.Code = gen.INTERNALERROR
		errResp.Error.Message = "internal server error"
	}
	return errResp
}

func toTodo(t todo.Todo) gen.Todo {
	return gen.Todo{
		Id:        openapi_types.UUID(t.ID),
		Title:     t.Title,
		CreatedAt: t.CreatedAt,
		Status:    gen.TodoStatus(t.Status),
		DueDate:   openapi_types.Date{Time: t.DueDate},
		UpdatedAt: t.UpdatedAt,
	}
}

func toConversation(c assistant.Conversation) gen.Conversation {
	return gen.Conversation{
		Id:          c.ID,
		Title:       c.Title,
		TitleSource: gen.ConversationTitleSource(c.TitleSource),
		UpdatedAt:   c.UpdatedAt,
		CreatedAt:   c.CreatedAt,
	}
}

func toChatMessage(msg assistant.ChatMessage) gen.ChatMessage {
	resp := gen.ChatMessage{
		Id:        msg.ID,
		Role:      gen.ChatMessageRole(msg.ChatRole),
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
	if msg.TurnID != uuid.Nil {
		turnID := openapi_types.UUID(msg.TurnID)
		resp.TurnId = &turnID
	}
	if msg.ActionExecuted != nil {
		resp.ActionExecuted = msg.ActionExecuted
	}
	if len(msg.SelectedSkills) > 0 {
		selectedSkills := make([]gen.SelectedSkill, 0, len(msg.SelectedSkills))
		for _, skill := range msg.SelectedSkills {
			tools := make([]string, len(skill.Tools))
			copy(tools, skill.Tools)
			selectedSkills = append(selectedSkills, gen.SelectedSkill{
				Name:   skill.Name,
				Source: skill.Source,
				Tools:  tools,
			})
		}
		resp.SelectedSkills = &selectedSkills
	}
	if len(msg.ActionDetails) > 0 {
		actionDetails := make([]gen.ChatMessageActionDetail, 0, len(msg.ActionDetails))
		for _, detail := range msg.ActionDetails {
			actionDetail := gen.ChatMessageActionDetail{
				ActionCallId: detail.ActionCallID,
				Input:        detail.Input,
				MessageState: gen.ChatMessageActionDetailMessageState(detail.MessageState),
				Name:         detail.Name,
				Output:       detail.Output,
				Text:         detail.Text,
			}
			if detail.ErrorMessage != nil {
				actionDetail.ErrorMessage = detail.ErrorMessage
			}
			if detail.ApprovalStatus != nil {
				status := gen.ChatMessageActionDetailApprovalStatus(*detail.ApprovalStatus)
				actionDetail.ApprovalStatus = &status
			}
			if detail.ApprovalDecisionReason != nil {
				actionDetail.ApprovalDecisionReason = detail.ApprovalDecisionReason
			}
			if detail.ApprovalDecidedAt != nil {
				actionDetail.ApprovalDecidedAt = detail.ApprovalDecidedAt
			}
			if detail.ActionExecuted != nil {
				actionDetail.ActionExecuted = detail.ActionExecuted
			}
			actionDetails = append(actionDetails, actionDetail)
		}
		resp.ActionDetails = &actionDetails
	}
	return resp
}

func toBoardSummary(summary todo.BoardSummary) gen.BoardSummary {
	resp := gen.BoardSummary{
		Counts: gen.TodoStatusCounts{
			DONE: summary.Content.Counts.Done,
			OPEN: summary.Content.Counts.Open,
		},
		GeneratedAt:  summary.GeneratedAt,
		NearDeadline: summary.Content.NearDeadline,
		NextUp:       []gen.NextUpTodoItem{},
		Overdue:      summary.Content.Overdue,
		Summary:      summary.Content.Summary,
	}
	for _, item := range summary.Content.NextUp {
		resp.NextUp = append(resp.NextUp, gen.NextUpTodoItem{
			Title:  item.Title,
			Reason: item.Reason,
		})
	}
	return resp
}
