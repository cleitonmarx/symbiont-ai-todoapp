package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoardSummaryContent_DiffersFrom(t *testing.T) {
	tests := map[string]struct {
		current  BoardSummaryContent
		previous BoardSummaryContent
		want     bool
	}{
		"same-content": {
			current: BoardSummaryContent{
				Counts:       TodoStatusCounts{Open: 1, Done: 2},
				NextUp:       []NextUpTodoItem{{Title: "A", Reason: "upcoming"}},
				Overdue:      []string{"B"},
				NearDeadline: []string{"C"},
			},
			previous: BoardSummaryContent{
				Counts:       TodoStatusCounts{Open: 1, Done: 2},
				NextUp:       []NextUpTodoItem{{Title: "A", Reason: "upcoming"}},
				Overdue:      []string{"B"},
				NearDeadline: []string{"C"},
			},
			want: false,
		},
		"different-counts": {
			current:  BoardSummaryContent{Counts: TodoStatusCounts{Open: 2, Done: 2}},
			previous: BoardSummaryContent{Counts: TodoStatusCounts{Open: 1, Done: 2}},
			want:     true,
		},
		"different-nextup": {
			current:  BoardSummaryContent{NextUp: []NextUpTodoItem{{Title: "A", Reason: "upcoming"}}},
			previous: BoardSummaryContent{NextUp: []NextUpTodoItem{{Title: "B", Reason: "upcoming"}}},
			want:     true,
		},
		"different-overdue": {
			current:  BoardSummaryContent{Overdue: []string{"A"}},
			previous: BoardSummaryContent{Overdue: []string{"B"}},
			want:     true,
		},
		"different-near-deadline": {
			current:  BoardSummaryContent{NearDeadline: []string{"A"}},
			previous: BoardSummaryContent{NearDeadline: []string{"B"}},
			want:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.current.DiffersFrom(tt.previous)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBoardSummaryContent_BuildComparisonHints(t *testing.T) {
	tests := map[string]struct {
		current  BoardSummaryContent
		previous BoardSummaryContent
		want     ComparisonHints
	}{
		"calculates-hints": {
			current: BoardSummaryContent{
				Counts: TodoStatusCounts{Open: 1, Done: 3},
				NextUp: []NextUpTodoItem{
					{Title: "Task A", Reason: "overdue"},
					{Title: "Task B", Reason: "due within 7 days"},
					{Title: "Task C", Reason: "upcoming"},
					{Title: "Task D", Reason: "future"},
				},
				Overdue:      []string{"Overdue 1"},
				NearDeadline: []string{"Near 1"},
			},
			previous: BoardSummaryContent{
				Counts:       TodoStatusCounts{Open: 1, Done: 1},
				NextUp:       []NextUpTodoItem{{Title: "Task A", Reason: "overdue"}},
				Overdue:      []string{"Overdue 1", "Overdue 2"},
				NearDeadline: []string{"Near 1", "Near 2"},
			},
			want: ComparisonHints{
				CompletedCandidates: []string{"Near 2", "Overdue 2"},
				DoneDelta:           2,
				OverdueTitles:       "Overdue 1",
				NearDeadlineTitles:  "Near 1",
				NextUpOverdue:       "Task A",
				NextUpDueSoon:       "Task B",
				NextUpUpcoming:      "Task C",
				NextUpFuture:        "Task D",
			},
		},
		"empty-hints": {
			current:  BoardSummaryContent{},
			previous: BoardSummaryContent{},
			want: ComparisonHints{
				CompletedCandidates: nil,
				DoneDelta:           0,
				OverdueTitles:       "none",
				NearDeadlineTitles:  "none",
				NextUpOverdue:       "none",
				NextUpDueSoon:       "none",
				NextUpUpcoming:      "none",
				NextUpFuture:        "none",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.current.BuildComparisonHints(tt.previous)
			assert.Equal(t, tt.want, got)
		})
	}
}
