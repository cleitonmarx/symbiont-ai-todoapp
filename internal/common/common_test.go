package common

import (
	"testing"
)

func TestPtr(t *testing.T) {
	type sample struct {
		A int
		B string
	}

	tests := []struct {
		name string
		val  any
		want any
	}{
		{"Int", 42, 42},
		{"String", "hello", "hello"},
		{"Struct", sample{A: 1, B: "test"}, sample{A: 1, B: "test"}},
		{"Bool", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch v := tt.val.(type) {
			case int:
				ptr := Ptr(v)
				if ptr == nil {
					t.Fatal("Ptr returned nil")
				}
				if *ptr != tt.want {
					t.Errorf("Ptr returned pointer to %v, want %v", *ptr, tt.want)
				}
			case string:
				ptr := Ptr(v)
				if ptr == nil {
					t.Fatal("Ptr returned nil")
				}
				if *ptr != tt.want {
					t.Errorf("Ptr returned pointer to %q, want %q", *ptr, tt.want)
				}
			case sample:
				ptr := Ptr(v)
				if ptr == nil {
					t.Fatal("Ptr returned nil")
				}
				if *ptr != tt.want {
					t.Errorf("Ptr returned pointer to %+v, want %+v", *ptr, tt.want)
				}
			case bool:
				ptr := Ptr(v)
				if ptr == nil {
					t.Fatal("Ptr returned nil")
				}
				if *ptr != tt.want {
					t.Errorf("Ptr returned pointer to %v, want %v", *ptr, tt.want)
				}
			default:
				t.Fatalf("unsupported type %T", v)
			}
		})
	}
}
