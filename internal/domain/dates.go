package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

var datePhraseRe = regexp.MustCompile(
	`(?i)\b(` +
		`\d{4}-\d{2}-\d{2}` + // YYYY-MM-DD
		`|` +
		`(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\s+\d{1,2},?\s+\d{4}` +
		`|` +
		`today|tomorrow|yesterday` +
		`|` +
		`next\s+(?:monday|tuesday|wednesday|thursday|friday|saturday|sunday)` +
		`)`,
)

// ExtractTimeFromText tries to extract a date from the given text.
func ExtractTimeFromText(
	text string,
	ref time.Time,
	loc *time.Location,
) (time.Time, bool) {

	m := datePhraseRe.FindStringSubmatch(text)
	if len(m) < 2 {
		return time.Time{}, false
	}

	token := strings.ToLower(strings.TrimSpace(m[1]))

	if iso, ok := resolveRelative(token, ref, loc); ok {
		return iso, true
	}

	t, err := dateparse.ParseIn(token, loc)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}

func resolveRelative(token string, ref time.Time, loc *time.Location) (time.Time, bool) {
	ref = ref.In(loc)
	ref = dateOnly(ref)

	switch token {
	case "today":
		return ref, true
	case "tomorrow":
		return ref.AddDate(0, 0, 1), true
	case "yesterday":
		return ref.AddDate(0, 0, -1), true
	}

	if after, ok := strings.CutPrefix(token, "next "); ok {
		wd, ok := parseWeekday(after)
		if !ok {
			return time.Time{}, false
		}
		return nextWeekday(ref, wd), true
	}

	return time.Time{}, false
}

func parseWeekday(s string) (time.Weekday, bool) {
	switch strings.ToLower(s) {
	case "sunday":
		return time.Sunday, true
	case "monday":
		return time.Monday, true
	case "tuesday":
		return time.Tuesday, true
	case "wednesday":
		return time.Wednesday, true
	case "thursday":
		return time.Thursday, true
	case "friday":
		return time.Friday, true
	case "saturday":
		return time.Saturday, true
	default:
		return 0, false
	}
}

func nextWeekday(ref time.Time, target time.Weekday) time.Time {
	cur := ref.Weekday()
	delta := (int(target) - int(cur) + 7) % 7
	if delta == 0 {
		delta = 7
	}
	return ref.AddDate(0, 0, delta)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
