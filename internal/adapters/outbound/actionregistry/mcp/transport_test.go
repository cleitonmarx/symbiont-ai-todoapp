package mcp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithAPIKey(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		headerName string
		apiKey     string
		wantHeader string
		wantValue  string
	}{
		"injects-header":   {headerName: "Authorization", apiKey: "Bearer test-token", wantHeader: "Authorization", wantValue: "Bearer test-token"},
		"no-key-no-header": {headerName: "Authorization", apiKey: "", wantHeader: "Authorization", wantValue: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var gotHeaderVal string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotHeaderVal = r.Header.Get(tt.wantHeader)
				_, _ = io.WriteString(w, "ok")
			}))
			defer server.Close()

			client := withAPIKey(nil, tt.headerName, tt.apiKey)
			resp, err := client.Get(server.URL)
			assert.NoError(t, err)
			defer resp.Body.Close() //nolint:errcheck

			assert.Equal(t, tt.wantValue, gotHeaderVal)
		})
	}
}

func TestAuthRoundTripper_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		headerName string
		apiKey     string
		want       string
	}{
		"sets-header": {headerName: "Authorization", apiKey: "Bearer token", want: "Bearer token"},
		"empty-key":   {headerName: "Authorization", apiKey: "", want: ""},
	}

	for name, tt := range tests {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			transport := authRoundTripper{
				base: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, tt.want, req.Header.Get(tt.headerName))
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
				}),
				headerName: tt.headerName,
				headerVal:  tt.apiKey,
			}

			req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			assert.NoError(t, err)
			resp, err := transport.RoundTrip(req)
			assert.NoError(t, err)
			defer resp.Body.Close() //nolint:errcheck
		})
	}
}
