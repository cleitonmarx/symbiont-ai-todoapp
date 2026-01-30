package types

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDate_UnmarshalGQL(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    Date
		wantErr bool
	}{
		{
			name:    "valid-date-string",
			input:   "2024-01-29",
			want:    Date(time.Date(2024, 1, 29, 0, 0, 0, 0, time.UTC)),
			wantErr: false,
		},
		{
			name:    "invalid-date-string",
			input:   "2024-13-01",
			want:    Date{},
			wantErr: true,
		},
		{
			name:    "not-a-string",
			input:   12345,
			want:    Date{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Date
			err := d.UnmarshalGQL(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, d)
			}
		})
	}
}

func TestDate_MarshalGQL(t *testing.T) {
	tests := []struct {
		name     string
		input    Date
		expected string
	}{
		{
			name:     "marshal-valid-date",
			input:    Date(time.Date(2024, 1, 29, 0, 0, 0, 0, time.UTC)),
			expected: `"2024-01-29"`,
		},
		{
			name:     "marshal-zero-date",
			input:    Date(time.Time{}),
			expected: `"0001-01-01"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.input.MarshalGQL(&buf)
			require.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Date
		wantErr bool
	}{
		{
			name:    "valid-json-date",
			input:   `"2024-01-29"`,
			want:    Date(time.Date(2024, 1, 29, 0, 0, 0, 0, time.UTC)),
			wantErr: false,
		},
		{
			name:    "invalid-json-date",
			input:   `"2024-13-01"`,
			want:    Date{},
			wantErr: true,
		},
		{
			name:    "not-a-quoted-string",
			input:   `12345`,
			want:    Date{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Date
			err := json.Unmarshal([]byte(tt.input), &d)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, d)
			}
		})
	}
}

func TestDate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    Date
		expected string
	}{
		{
			name:     "marshal-valid-date",
			input:    Date(time.Date(2024, 1, 29, 0, 0, 0, 0, time.UTC)),
			expected: `"2024-01-29"`,
		},
		{
			name:     "marshal-zero-date",
			input:    Date(time.Time{}),
			expected: `"0001-01-01"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.input.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(b))
		})
	}
}
