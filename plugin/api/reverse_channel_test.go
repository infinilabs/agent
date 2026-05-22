package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"infini.sh/framework/core/global"
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

func TestShouldServeRegisteredAPIReverse(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		path   string
		expect bool
	}{
		{name: "queue stats", method: http.MethodGet, path: "/queue/stats", expect: true},
		{name: "task search with query", method: http.MethodGet, path: "/pipeline/tasks/?size=20", expect: true},
		{name: "config runtime", method: http.MethodGet, path: "/config/runtime", expect: true},
		{name: "logger setting", method: http.MethodPost, path: "/setting/logger", expect: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := shouldServeRegisteredAPIReverse(tc.method, tc.path); actual != tc.expect {
				t.Fatalf("unexpected match result: %v", actual)
			}
		})
	}
}

func TestExecuteAgentRegisteredAPIReverse(t *testing.T) {
	oldResolver := agentReverseAPIProxyTargetResolver
	oldClientFactory := agentReverseHTTPClientFactory
	t.Cleanup(func() {
		agentReverseAPIProxyTargetResolver = oldResolver
		agentReverseHTTPClientFactory = oldClientFactory
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, password, ok := req.BasicAuth()
		if !ok || user != "api-user" || password != "api-pass" {
			t.Fatalf("unexpected basic auth: %v %s %s", ok, user, password)
		}
		if req.URL.Path != "/api/queue/stats" || req.URL.RawQuery != "size=20" {
			t.Fatalf("unexpected proxied url: %s", req.URL.String())
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("proxied"))
	}))
	defer server.Close()

	agentReverseAPIProxyTargetResolver = func() (agentReverseProxyTarget, error) {
		return agentReverseProxyTarget{
			endpoint:      server.URL,
			basePath:      "/api",
			basicAuthUser: "api-user",
			basicAuthPass: "api-pass",
		}, nil
	}
	agentReverseHTTPClientFactory = func(agentReverseProxyTarget) (*http.Client, error) {
		return server.Client(), nil
	}

	status, body := executeAgentRegisteredAPIReverse(http.MethodGet, "/queue/stats?size=20", nil)
	if status != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", status)
	}
	if string(body) != "proxied" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestResolveAgentReverseAPIProxyTarget(t *testing.T) {
	oldAPIConfig := global.Env().SystemConfig.APIConfig
	oldWebConfig := global.Env().SystemConfig.WebAppConfig
	t.Cleanup(func() {
		global.Env().SystemConfig.APIConfig = oldAPIConfig
		global.Env().SystemConfig.WebAppConfig = oldWebConfig
	})

	apiURL, _ := url.Parse("http://127.0.0.1:9000")
	global.Env().SystemConfig.APIConfig.Enabled = true
	global.Env().SystemConfig.APIConfig.NetworkConfig.Publish = apiURL.Host
	global.Env().SystemConfig.APIConfig.BasePath = "/api"
	global.Env().SystemConfig.APIConfig.Security.Username = "api-user"
	global.Env().SystemConfig.APIConfig.Security.Password = "api-pass"

	target, err := resolveAgentReverseAPIProxyTarget()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.endpoint != "http://"+apiURL.Host {
		t.Fatalf("unexpected endpoint: %s", target.endpoint)
	}
	if target.basePath != "/api" {
		t.Fatalf("unexpected base path: %s", target.basePath)
	}
	if target.basicAuthUser != "api-user" || target.basicAuthPass != "api-pass" {
		t.Fatalf("unexpected auth target: %#v", target)
	}
}

func TestBuildAgentReverseProxyURL(t *testing.T) {
	actual, err := buildAgentReverseProxyURL("http://127.0.0.1:9000", "/api", "/queue/stats?size=20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actual != "http://127.0.0.1:9000/api/queue/stats?size=20" {
		t.Fatalf("unexpected proxy url: %s", actual)
	}
}
