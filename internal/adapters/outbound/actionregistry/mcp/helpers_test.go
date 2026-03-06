package mcp

import (
	"context"
	"net/http"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeConnector struct {
	session mcpSession
	err     error
}

func (c *fakeConnector) Connect(context.Context) (mcpSession, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.session, nil
}

type fakeSession struct {
	listResults    []*mcp.ListToolsResult
	listErr        error
	callResult     *mcp.CallToolResult
	callErr        error
	lastCallParams *mcp.CallToolParams
	listCalls      int
	closeCalls     int
}

func (s *fakeSession) ListTools(_ context.Context, _ *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if len(s.listResults) == 0 {
		return &mcp.ListToolsResult{}, nil
	}
	index := min(s.listCalls, len(s.listResults)-1)
	s.listCalls++
	return s.listResults[index], nil
}

func (s *fakeSession) CallTool(_ context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	s.lastCallParams = params
	if s.callErr != nil {
		return nil, s.callErr
	}
	return s.callResult, nil
}

func (s *fakeSession) Close() error {
	s.closeCalls++
	return nil
}

type fakeRenderer struct {
	message assistant.Message
	ok      bool
}

func (r fakeRenderer) Render(_ assistant.ActionCall, _ assistant.Message) (assistant.Message, bool) {
	return r.message, r.ok
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
