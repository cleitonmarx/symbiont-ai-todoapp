package common

import (
	"strings"
)

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// NormalizeWhitespace trims and compacts repeated whitespace.
func NormalizeWhitespace(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}
