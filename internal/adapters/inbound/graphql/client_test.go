package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/graphql/types"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockRoundTripper implements http.RoundTripper for testing HTTP requests.
type mockRoundTripper struct {
	handler func(*http.Request) *http.Response
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req), nil
}

// setTestHTTPClient replaces http.DefaultClient's Transport for testing.
func setTestHTTPClient(handler func(*http.Request) *http.Response) func() {
	orig := http.DefaultClient.Transport
	http.DefaultClient.Transport = &mockRoundTripper{handler: handler}
	return func() { http.DefaultClient.Transport = orig }
}

func TestClient_UpdateTodos(t *testing.T) {
	todoIDs := []uuid.UUID{uuid.New(), uuid.New()}
	dueDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	todos := []*gen.Todo{
		{ID: todoIDs[0], Title: "Test", Status: gen.TodoStatus("OPEN"), DueDate: types.Date(dueDate)},
		{ID: todoIDs[1], Title: "Test2", Status: gen.TodoStatus("OPEN"), DueDate: types.Date(dueDate)},
	}
	tests := map[string]struct {
		params      []gen.UpdateTodoParams
		mockHandler func(*http.Request) *http.Response
		expectErr   bool
	}{
		"success": {
			params: []gen.UpdateTodoParams{
				{ID: todoIDs[0], Title: common.Ptr("Test3")},
				{ID: todoIDs[1], Title: common.Ptr("Test4")},
			},
			mockHandler: func(req *http.Request) *http.Response {
				resp := response[*gen.Todo]{
					Data: map[string]*gen.Todo{"updateTodo0": todos[0], "updateTodo1": todos[1]},
				}
				b, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: 200,
					Body:       ioNopCloser(b),
					Header:     make(http.Header),
				}
			},
			expectErr: false,
		},
		"error-response": {
			params: []gen.UpdateTodoParams{{ID: todoIDs[0], Title: common.Ptr("Test")}},
			mockHandler: func(req *http.Request) *http.Response {
				resp := response[*gen.Todo]{
					Errors: []gqlError{{Message: "fail"}},
				}
				b, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: 400,
					Body:       ioNopCloser(b),
					Header:     make(http.Header),
				}
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			restore := setTestHTTPClient(tt.mockHandler)
			defer restore()
			client := NewClient("http://fake")
			out, err := client.UpdateTodos(context.Background(), tt.params)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, out)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, todos, out)
			}
		})
	}
}

func TestClient_DeleteTodos(t *testing.T) {
	tests := map[string]struct {
		ids         []uuid.UUID
		mockHandler func(*http.Request) *http.Response
		expectErr   bool
	}{
		"success": {
			ids: []uuid.UUID{uuid.New(), uuid.New()},
			mockHandler: func(req *http.Request) *http.Response {
				resp := response[bool]{
					Data: map[string]bool{"deleteTodo0": true, "deleteTodo1": true},
				}
				b, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: 200,
					Body:       ioNopCloser(b),
					Header:     make(http.Header),
				}
			},
			expectErr: false,
		},
		"error-response": {
			ids: []uuid.UUID{uuid.New()},
			mockHandler: func(req *http.Request) *http.Response {
				resp := response[bool]{
					Errors: []gqlError{{Message: "fail"}},
				}
				b, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: 400,
					Body:       ioNopCloser(b),
					Header:     make(http.Header),
				}
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			restore := setTestHTTPClient(tt.mockHandler)
			defer restore()
			client := NewClient("http://fake")
			out, err := client.DeleteTodos(context.Background(), tt.ids)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, out)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, []bool{true, true}, out)
			}
		})
	}
}

func TestClient_ListTodos(t *testing.T) {
	page := &gen.TodoPage{Page: 1}
	tests := map[string]struct {
		status      *gen.TodoStatus
		page        int
		pageSize    int
		mockHandler func(*http.Request) *http.Response
		expectErr   bool
	}{
		"success": {
			status:   nil,
			page:     1,
			pageSize: 10,
			mockHandler: func(req *http.Request) *http.Response {
				resp := response[*gen.TodoPage]{
					Data: map[string]*gen.TodoPage{"listTodos": page},
				}
				b, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: 200,
					Body:       ioNopCloser(b),
					Header:     make(http.Header),
				}
			},
			expectErr: false,
		},
		"error-response": {
			status:   nil,
			page:     1,
			pageSize: 10,
			mockHandler: func(req *http.Request) *http.Response {
				resp := response[*gen.TodoPage]{
					Errors: []gqlError{{Message: "fail"}},
				}
				b, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: 400,
					Body:       ioNopCloser(b),
					Header:     make(http.Header),
				}
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			restore := setTestHTTPClient(tt.mockHandler)
			defer restore()
			client := NewClient("http://fake")
			out, err := client.ListTodos(context.Background(), tt.status, tt.page, tt.pageSize)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, out)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, page, out)
			}
		})
	}
}

// ioNopCloser returns a ReadCloser from bytes.
func ioNopCloser(b []byte) *nopCloser {
	return &nopCloser{data: b}
}

type nopCloser struct {
	data []byte
	read bool
}

func (n *nopCloser) Read(p []byte) (int, error) {
	if n.read {
		return 0, errors.New("EOF")
	}
	copy(p, n.data)
	n.read = true
	return len(n.data), nil
}
func (n *nopCloser) Close() error { return nil }
