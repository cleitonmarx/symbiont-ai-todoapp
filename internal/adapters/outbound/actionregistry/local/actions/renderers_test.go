package actions

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/stretchr/testify/assert"
	"github.com/toon-format/toon-go"
)

func TestActionResultRenderers(t *testing.T) {
	t.Parallel()

	mustMarshal := func(t *testing.T, value any) string {
		t.Helper()
		content, err := toon.MarshalString(value)
		if err != nil {
			t.Fatalf("toon.MarshalString: %v", err)
		}
		return content
	}

	tests := map[string]struct {
		renderer   assistant.ActionResultRenderer
		actionCall assistant.ActionCall
		result     assistant.Message
		want       string
		wantOK     bool
	}{
		"create-todos": {
			renderer:   createTodosRenderer{},
			actionCall: assistant.ActionCall{Name: "create_todos"},
			result: assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: common.Ptr("call-1"),
				Content: mustMarshal(t, struct {
					Todos []struct {
						ID      string `toon:"id"`
						Title   string `toon:"title"`
						DueDate string `toon:"due_date"`
						Status  string `toon:"status"`
					} `toon:"todos"`
				}{
					Todos: []struct {
						ID      string `toon:"id"`
						Title   string `toon:"title"`
						DueDate string `toon:"due_date"`
						Status  string `toon:"status"`
					}{
						{ID: "1", Title: "Call Alice", DueDate: "2026-03-02", Status: "OPEN"},
					},
				}),
			},
			want:   "Created **Call Alice** (Due: Mar 02, 2026) - OPEN.",
			wantOK: true,
		},
		"update-todos": {
			renderer:   updateTodosRenderer{},
			actionCall: assistant.ActionCall{Name: "update_todos"},
			result: assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: common.Ptr("call-2"),
				Content: mustMarshal(t, struct {
					Todos []struct {
						ID      string `toon:"id"`
						Title   string `toon:"title"`
						DueDate string `toon:"due_date"`
						Status  string `toon:"status"`
					} `toon:"todos"`
				}{
					Todos: []struct {
						ID      string `toon:"id"`
						Title   string `toon:"title"`
						DueDate string `toon:"due_date"`
						Status  string `toon:"status"`
					}{
						{ID: "1", Title: "Call Alice", DueDate: "2026-03-02", Status: "OPEN"},
						{ID: "2", Title: "Buy milk", DueDate: "2026-03-03", Status: "DONE"},
					},
				}),
			},
			want:   "Updated 2 todos:\n**Call Alice** (Due: Mar 02, 2026) - OPEN\n**Buy milk** (Due: Mar 03, 2026) - DONE",
			wantOK: true,
		},
		"update-todos-due-date": {
			renderer:   updateTodosDueDateRenderer{},
			actionCall: assistant.ActionCall{Name: "update_todos_due_date"},
			result: assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: common.Ptr("call-3"),
				Content: mustMarshal(t, struct {
					Todos []struct {
						ID      string `toon:"id"`
						Title   string `toon:"title"`
						DueDate string `toon:"due_date"`
						Status  string `toon:"status"`
					} `toon:"todos"`
				}{
					Todos: []struct {
						ID      string `toon:"id"`
						Title   string `toon:"title"`
						DueDate string `toon:"due_date"`
						Status  string `toon:"status"`
					}{
						{ID: "1", Title: "Call Alice", DueDate: "2026-03-04", Status: "OPEN"},
					},
				}),
			},
			want:   "Updated due date for **Call Alice** (Due: Mar 04, 2026) - OPEN.",
			wantOK: true,
		},
		"delete-todos": {
			renderer: deleteTodosRenderer{},
			actionCall: assistant.ActionCall{
				Name:  "delete_todos",
				Input: `{"todos":[{"title":"Call Alice"}]}`,
			},
			result: assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: common.Ptr("call-4"),
				Content: mustMarshal(t, struct {
					Todos []struct {
						ID      string `toon:"id"`
						Deleted bool   `toon:"deleted"`
					} `toon:"todos"`
				}{
					Todos: []struct {
						ID      string `toon:"id"`
						Deleted bool   `toon:"deleted"`
					}{
						{ID: "1", Deleted: true},
					},
				}),
			},
			want:   "Deleted **Call Alice**.",
			wantOK: true,
		},
		"returns-false-for-malformed-content": {
			renderer:   updateTodosRenderer{},
			actionCall: assistant.ActionCall{Name: "update_todos"},
			result: assistant.Message{
				Role:         assistant.ChatRole_Tool,
				ActionCallID: common.Ptr("call-5"),
				Content:      "malformed",
			},
		},
	}

	for name, tt := range tests {

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := tt.renderer.Render(tt.actionCall, tt.result)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, assistant.ChatRole_Assistant, got.Role)
				assert.Equal(t, tt.want, got.Content)
			}
		})
	}
}
