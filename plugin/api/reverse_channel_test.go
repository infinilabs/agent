package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httprouter "infini.sh/framework/core/api/router"
	framework_reverse "infini.sh/framework/core/api/websocket/reverse"
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
	status, body := executeAgentReverseRequest(http.MethodGet, "/not-found", nil, framework_reverse.RequestMessage{})
	if status != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", status)
	}
	if len(body) == 0 {
		t.Fatal("expected error body")
	}
}

func TestProtectedAPIRouter(t *testing.T) {
	router := newProtectedAPIRouter(func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusAccepted)
	})

	t.Run("match and serve protected route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/queue/stats?size=20", nil)
		if !router.Match(req.Method, req.URL.RequestURI()) {
			t.Fatal("expected protected route to match")
		}
		recorder := httptest.NewRecorder()
		if !router.ServeHTTP(recorder, req) {
			t.Fatal("expected protected route to be served")
		}
		if recorder.Code != http.StatusAccepted {
			t.Fatalf("unexpected status code: %d", recorder.Code)
		}
	})

	t.Run("reject unknown route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
		if router.Match(req.Method, req.URL.RequestURI()) {
			t.Fatal("expected unknown route not to match")
		}
		recorder := httptest.NewRecorder()
		if router.ServeHTTP(recorder, req) {
			t.Fatal("expected unknown route not to be served")
		}
	})
}

func TestReverseAPIRouterMatchesSpecialAndProtectedRoutes(t *testing.T) {
	router := newReverseAPIRouter(AgentAPI{})

	testCases := []struct {
		name   string
		method string
		path   string
		expect bool
	}{
		{name: "agent info", method: http.MethodGet, path: "/agent/_info", expect: true},
		{name: "discovery", method: http.MethodGet, path: "/elasticsearch/node/_discovery", expect: true},
		{name: "protected queue route", method: http.MethodGet, path: "/queue/stats", expect: true},
		{name: "unknown route", method: http.MethodGet, path: "/not-found", expect: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := router.Match(tc.method, tc.path); actual != tc.expect {
				t.Fatalf("unexpected match result: %v", actual)
			}
		})
	}
}
