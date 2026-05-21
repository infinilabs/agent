package api

import (
	"net/http"
	"testing"
)

func TestBuildAgentReverseChannelURL(t *testing.T) {
	testCases := []struct {
		name   string
		server string
		expect string
	}{
		{
			name:   "http root",
			server: "http://console.example.com",
			expect: "ws://console.example.com/ws",
		},
		{
			name:   "https base path",
			server: "https://console.example.com/console",
			expect: "wss://console.example.com/console/ws",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := buildAgentReverseChannelURL(tc.server)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tc.expect {
				t.Fatalf("unexpected ws url, got %s want %s", actual, tc.expect)
			}
		})
	}
}

func TestExecuteAgentReverseRequestUnknownPath(t *testing.T) {
	status, body := executeAgentReverseRequest(http.MethodGet, "/not-found", nil)
	if status != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", status)
	}
	if len(body) == 0 {
		t.Fatal("expected error body")
	}
}
