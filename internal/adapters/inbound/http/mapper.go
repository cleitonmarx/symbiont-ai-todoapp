package http

import (
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func toError(err error) gen.ErrorResp {
	errResp := gen.ErrorResp{}
	switch e := err.(type) {
	case *domain.ValidationErr:
		errResp.Error.Code = gen.BADREQUEST
		errResp.Error.Message = e.Error()
	case *domain.NotFoundErr:
		errResp.Error.Code = gen.NOTFOUND
		errResp.Error.Message = e.Error()
	default:
		errResp.Error.Code = gen.INTERNALERROR
		errResp.Error.Message = "internal server error"
	}
	return errResp
}

func toTodo(t domain.Todo) gen.Todo {
	return gen.Todo{
		Id:        openapi_types.UUID(t.ID),
		Title:     t.Title,
		CreatedAt: t.CreatedAt,
		Status:    gen.TodoStatus(t.Status),
		DueDate:   openapi_types.Date{Time: t.DueDate},
		UpdatedAt: t.UpdatedAt,
	}
}

func toChatMessage(msg domain.ChatMessage) gen.ChatMessage {
	return gen.ChatMessage{
		Id:        msg.ID,
		Role:      gen.ChatMessageRole(msg.ChatRole),
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
}

func toBoardSummary(summary domain.BoardSummary) gen.BoardSummary {
	resp := gen.BoardSummary{
		Counts: gen.TodoStatusCounts{
			DONE: summary.Content.Counts.Done,
			OPEN: summary.Content.Counts.Open,
		},
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
