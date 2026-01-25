package http

import (
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/openapi"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func toOpenAPIError(err error) openapi.ErrorResp {
	errResp := openapi.ErrorResp{}
	switch e := err.(type) {
	case *domain.ValidationErr:
		errResp.Error.Code = openapi.BADREQUEST
		errResp.Error.Message = e.Error()
	case *domain.NotFoundErr:
		errResp.Error.Code = openapi.NOTFOUND
		errResp.Error.Message = e.Error()
	default:
		errResp.Error.Code = openapi.INTERNALERROR
		errResp.Error.Message = "internal server error"
	}
	return errResp
}

func toOpenAPITodo(t domain.Todo) openapi.Todo {
	return openapi.Todo{
		Id:        openapi_types.UUID(t.ID),
		Title:     t.Title,
		CreatedAt: t.CreatedAt,
		Status:    openapi.TodoStatus(t.Status),
		DueDate:   openapi_types.Date{Time: t.DueDate},
		UpdatedAt: t.UpdatedAt,
	}
}
