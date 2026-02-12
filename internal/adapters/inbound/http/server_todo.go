package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/usecases"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (api TodoAppServer) ListTodos(w http.ResponseWriter, r *http.Request, params gen.ListTodosParams) {
	resp := gen.ListTodosResp{
		Items: []gen.Todo{},
		Page:  params.Page,
	}
	var queryParams []usecases.ListTodoOptions
	if params.Status != nil {
		queryParams = append(queryParams, usecases.WithStatus(domain.TodoStatus(*params.Status)))
	}
	if params.Query != nil {
		queryParams = append(queryParams, usecases.WithSearchQuery(*params.Query))
	}
	if params.DateRange.DueAfter != nil && params.DateRange.DueBefore != nil {
		queryParams = append(queryParams, usecases.WithDueDateRange(params.DateRange.DueAfter.Time, params.DateRange.DueBefore.Time))
	}
	if params.Sort != nil {
		queryParams = append(queryParams, usecases.WithSortBy(string(*params.Sort)))
	}

	todos, hasMore, err := api.ListTodosUseCase.Query(r.Context(), params.Page, params.PageSize, queryParams...)
	if err != nil {
		api.Logger.Printf("Error listing todos: %v", err)
		respondError(w, toError(err))
		return
	}

	for _, t := range todos {
		resp.Items = append(resp.Items, toTodo(t))
	}
	if hasMore {
		nextPage := params.Page + 1
		resp.NextPage = &nextPage
	}
	if params.Page > 1 {
		prevPage := params.Page - 1
		resp.PreviousPage = &prevPage
	}

	respondJSON(w, http.StatusOK, resp)
}

func (api TodoAppServer) CreateTodo(w http.ResponseWriter, r *http.Request) {
	var req gen.CreateTodoJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp := gen.ErrorResp{}
		errResp.Error.Code = gen.BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: %v", err)

		respondError(w, errResp)
		return
	}

	todo, err := api.CreateTodoUseCase.Execute(r.Context(), req.Title, req.DueDate.Time)
	if err != nil {
		api.Logger.Printf("Error creating todo: %v", err)
		respondError(w, toError(err))
		return
	}

	respondJSON(w, http.StatusCreated, toTodo(todo))
}
func (api TodoAppServer) UpdateTodo(w http.ResponseWriter, r *http.Request, todoId openapi_types.UUID) {
	var req gen.UpdateTodoJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errResp := gen.ErrorResp{}
		errResp.Error.Code = gen.BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: %v", err)

		respondError(w, errResp)
		return
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		dueDate = &req.DueDate.Time
	}
	if req.Status != nil && *req.Status != gen.DONE && *req.Status != gen.OPEN {
		errResp := gen.ErrorResp{}
		errResp.Error.Code = gen.BADREQUEST
		errResp.Error.Message = fmt.Sprintf("invalid request body: unknown TodoStatus value: %s", *req.Status)
		respondError(w, errResp)
		return
	}

	todo, err := api.UpdateTodoUseCase.Execute(
		r.Context(),
		uuid.UUID(todoId),
		req.Title,
		(*domain.TodoStatus)(req.Status),
		dueDate,
	)
	if err != nil {
		api.Logger.Printf("Error updating todo: %v", err)
		respondError(w, toError(err))
		return
	}

	respondJSON(w, http.StatusOK, toTodo(todo))
}

func (api TodoAppServer) DeleteTodo(w http.ResponseWriter, r *http.Request, todoId openapi_types.UUID) {
	err := api.DeleteTodoUseCase.Execute(r.Context(), todoId)
	if err != nil {
		api.Logger.Printf("Error deleting todo: %v", err)
		respondError(w, toError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
