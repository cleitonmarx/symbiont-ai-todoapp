package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/google/uuid"
)

// request represents a GraphQL request payload.
type request struct {
	Query     string `json:"query"`
	Variables any    `json:"variables,omitempty"`
	url       string `json:"-"`
}

// location represents the location of an error in a GraphQL query.
type location struct {
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
}

// gqlError represents a GraphQL error response.
type gqlError struct {
	Err        error          `json:"-"`
	Message    string         `json:"message"`
	Locations  []location     `json:"locations,omitempty"`
	Extensions map[string]any `json:"extensions,omitempty"`
	Rule       string         `json:"-"`
}

// response represents a GraphQL response payload.
type response[T any] struct {
	Data       map[string]T   `json:"data"`
	Extensions map[string]any `json:"extensions,omitempty"`
	Errors     []gqlError     `json:"errors,omitempty"`
}

// Client is a GraphQL client for interacting with the todo app GraphQL API.
type Client struct {
	url string
}

// NewClient creates a new GraphQL client with the specified endpoint.
func NewClient(endpoint string) *Client {
	return &Client{
		url: endpoint,
	}
}

// UpdateTodos updates multiple todos based on the provided parameters.
func (c *Client) UpdateTodos(ctx context.Context, params []gen.UpdateTodoParams) ([]*gen.Todo, error) {
	req := request{
		url: c.url,
	}
	sb := &strings.Builder{}
	variables := make(map[string]any)
	sb.WriteString("mutation UpdateTodos(")
	for i, p := range params {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(sb, "$params%d: updateTodoParams!", i)
		variables[fmt.Sprintf("params%d", i)] = p
	}
	sb.WriteString(") {")
	for i := range params {
		fmt.Fprintf(sb, " updateTodo%d: updateTodo(params: $params%d) { id title status due_date created_at updated_at } ", i, i)
	}
	sb.WriteString("}")

	req.Query = sb.String()
	req.Variables = variables

	resp, err := makeRequest[*gen.Todo](ctx, req)
	if err != nil {
		return nil, err
	}

	output := make([]*gen.Todo, len(params))
	for i := range params {
		output[i] = resp.Data[fmt.Sprintf("updateTodo%d", i)]
	}

	return output, nil
}

// DeleteTodos deletes multiple todos by their IDs.
func (c *Client) DeleteTodos(ctx context.Context, ids []uuid.UUID) ([]bool, error) {
	req := request{
		url: c.url,
	}
	sb := &strings.Builder{}
	variables := make(map[string]any)
	sb.WriteString("mutation DeleteTodos(")
	for i, id := range ids {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(sb, "$id%d: UUID!", i)
		variables[fmt.Sprintf("id%d", i)] = id
	}
	sb.WriteString(") {")
	for i := range ids {
		fmt.Fprintf(sb, " deleteTodo%d: deleteTodo(id: $id%d) ", i, i)
	}
	sb.WriteString("}")

	req.Query = sb.String()
	req.Variables = variables

	resp, err := makeRequest[bool](ctx, req)
	if err != nil {
		return nil, err
	}

	output := make([]bool, len(ids))
	for i := range ids {
		output[i] = resp.Data[fmt.Sprintf("deleteTodo%d", i)]
	}

	return output, nil
}

// ListTodos retrieves a paginated list of todos, optionally filtered by status.
func (c *Client) ListTodos(ctx context.Context, status *gen.TodoStatus, page int, pageSize int) (*gen.TodoPage, error) {
	req := request{
		Query:     listTodosQuery,
		Variables: map[string]any{"status": status, "page": page, "pageSize": pageSize},
		url:       c.url,
	}

	resp, err := makeRequest[*gen.TodoPage](ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Data["listTodos"], nil
}

// makeRequest sends a GraphQL request and decodes the response.
func makeRequest[T any](ctx context.Context, req request) (response[T], error) {
	body, err := json.Marshal(req)
	if err != nil {
		return response[T]{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, req.url, bytes.NewReader(body))
	if err != nil {
		return response[T]{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return response[T]{}, err
	}
	defer httpResp.Body.Close() //nolint:errcheck

	var gqlResp response[T]

	if err = json.NewDecoder(httpResp.Body).Decode(&gqlResp); err != nil {
		return response[T]{}, fmt.Errorf("failed to decode error response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return response[T]{}, fmt.Errorf("response failed with status %d: %v", httpResp.StatusCode, gqlResp.Errors)
	}
	return gqlResp, nil
}

var listTodosQuery = `
query ListTodos($status: TodoStatus, $page: Int, $pageSize: Int) {
  listTodos(status: $status, page: $page, pageSize: $pageSize) {
    items { id title status due_date created_at updated_at }
    page nextPage previousPage
  }
}`
