package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExtractTimeFromText(t *testing.T) {
	loc := time.UTC
	ref := time.Date(2026, 1, 27, 10, 0, 0, 0, loc) // Tuesday

	tests := map[string]struct {
		text     string
		ref      time.Time
		loc      *time.Location
		expected time.Time
		ok       bool
	}{
		"today": {
			text:     "I need to finish this today",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 27, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"tomorrow": {
			text:     "Let's do it tomorrow",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 28, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"yesterday": {
			text:     "I should have done it yesterday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"next-monday": {
			text:     "due next monday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 2, 2, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"next-friday": {
			text:     "finish next friday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 30, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"iso-date-format": {
			text:     "deadline is 2026-02-15",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 2, 15, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"month-day-year-format": {
			text:     "due January 10, 2026",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 10, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"abbreviated-month": {
			text:     "deadline: Feb 28, 2026",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 2, 28, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"case-insensitive": {
			text:     "finish by TOMORROW",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 28, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"no-date-found": {
			text:     "no date mentioned here",
			ref:      ref,
			loc:      loc,
			expected: time.Time{},
			ok:       false,
		},
		"invalid-date": {
			text:     "due on some random day",
			ref:      ref,
			loc:      loc,
			expected: time.Time{},
			ok:       false,
		},
		"multiple-dates-returns-first": {
			text:     "start today and finish tomorrow",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 27, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"next-day-same-weekday": {
			text:     "next tuesday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 2, 3, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"abbreviated-month-name": {
			text:     "due mar 15, 2026",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 3, 15, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"sept-abbreviation": {
			text:     "due sep 10, 2026",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 9, 10, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"next-sunday": {
			text:     "next sunday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 2, 1, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"next-saturday": {
			text:     "next saturday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 31, 0, 0, 0, 0, loc),
			ok:       true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := ExtractTimeFromText(tt.text, tt.ref, tt.loc)
			assert.Equal(t, tt.ok, ok, "expected ok to be %v, got %v", tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.expected, got, "expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestResolveRelative(t *testing.T) {
	loc := time.UTC
	ref := time.Date(2026, 1, 27, 10, 0, 0, 0, loc) // Tuesday

	tests := map[string]struct {
		token    string
		ref      time.Time
		loc      *time.Location
		expected time.Time
		ok       bool
	}{
		"today": {
			token:    "today",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 27, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"tomorrow": {
			token:    "tomorrow",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 28, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"yesterday": {
			token:    "yesterday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 26, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"next-monday": {
			token:    "next monday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 2, 2, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"next-tuesday": {
			token:    "next tuesday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 2, 3, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"next-friday": {
			token:    "next friday",
			ref:      ref,
			loc:      loc,
			expected: time.Date(2026, 1, 30, 0, 0, 0, 0, loc),
			ok:       true,
		},
		"invalid-token": {
			token:    "some random text",
			ref:      ref,
			loc:      loc,
			expected: time.Time{},
			ok:       false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := resolveRelative(tt.token, tt.ref, tt.loc)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestParseWeekday(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected time.Weekday
		ok       bool
	}{
		"sunday": {
			input:    "sunday",
			expected: time.Sunday,
			ok:       true,
		},
		"monday": {
			input:    "monday",
			expected: time.Monday,
			ok:       true,
		},
		"tuesday": {
			input:    "tuesday",
			expected: time.Tuesday,
			ok:       true,
		},
		"wednesday": {
			input:    "wednesday",
			expected: time.Wednesday,
			ok:       true,
		},
		"thursday": {
			input:    "thursday",
			expected: time.Thursday,
			ok:       true,
		},
		"friday": {
			input:    "friday",
			expected: time.Friday,
			ok:       true,
		},
		"saturday": {
			input:    "saturday",
			expected: time.Saturday,
			ok:       true,
		},
		"case-insensitive-upper": {
			input:    "MONDAY",
			expected: time.Monday,
			ok:       true,
		},
		"case-insensitive-mixed": {
			input:    "FrIdAy",
			expected: time.Friday,
			ok:       true,
		},
		"invalid": {
			input:    "funday",
			expected: 0,
			ok:       false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := parseWeekday(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestNextWeekday(t *testing.T) {
	ref := time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC) // Tuesday

	tests := map[string]struct {
		ref      time.Time
		target   time.Weekday
		expected time.Time
	}{
		"next-monday": {
			ref:      ref,
			target:   time.Monday,
			expected: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
		},
		"next-tuesday-same-day": {
			ref:      ref,
			target:   time.Tuesday,
			expected: time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC),
		},
		"next-friday": {
			ref:      ref,
			target:   time.Friday,
			expected: time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC),
		},
		"next-sunday": {
			ref:      ref,
			target:   time.Sunday,
			expected: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		"next-wednesday": {
			ref:      ref,
			target:   time.Wednesday,
			expected: time.Date(2026, 1, 28, 0, 0, 0, 0, time.UTC),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := nextWeekday(tt.ref, tt.target)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDateOnly(t *testing.T) {
	tests := map[string]struct {
		input    time.Time
		expected time.Time
	}{
		"strips-time": {
			input:    time.Date(2026, 1, 27, 15, 30, 45, 123456, time.UTC),
			expected: time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC),
		},
		"midnight": {
			input:    time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC),
		},
		"end-of-day": {
			input:    time.Date(2026, 1, 27, 23, 59, 59, 999999999, time.UTC),
			expected: time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := dateOnly(tt.input)
			assert.Equal(t, tt.expected, got)
			assert.Equal(t, 0, got.Hour())
			assert.Equal(t, 0, got.Minute())
			assert.Equal(t, 0, got.Second())
		})
	}
}
