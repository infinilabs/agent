package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/model"
	configcommon "infini.sh/framework/modules/configs/common"
)

func TestEnsureAgentAccessToken(t *testing.T) {
	t.Setenv("KEYSTORE_PATH", t.TempDir())

	if err := ensureAgentAccessToken(); err != nil {
		t.Fatalf("ensure access token: %v", err)
	}

	value, err := configcommon.LoadTokenFromKeystore(configcommon.AgentAccessTokenKeystoreKey)
	if err != nil {
		t.Fatalf("load access token: %v", err)
	}
	if value == "" {
		t.Fatal("expected access token to be initialized")
	}
}

func TestProxyProtectedAPIRequiresAccessToken(t *testing.T) {
	t.Setenv("KEYSTORE_PATH", t.TempDir())

	if err := ensureAgentAccessToken(); err != nil {
		t.Fatalf("ensure access token: %v", err)
	}
	token, err := configcommon.LoadTokenFromKeystore(configcommon.AgentAccessTokenKeystoreKey)
	if err != nil {
		t.Fatalf("load access token: %v", err)
	}

	oldWebConfig := global.Env().SystemConfig.WebAppConfig
	oldAPIConfig := global.Env().SystemConfig.APIConfig
	t.Cleanup(func() {
		global.Env().SystemConfig.WebAppConfig = oldWebConfig
		global.Env().SystemConfig.APIConfig = oldAPIConfig
	})

	global.Env().SystemConfig.WebAppConfig.Security.Enabled = true
	global.Env().SystemConfig.APIConfig.Security.Enabled = true
	global.Env().SystemConfig.APIConfig.Security.Username = "api-user"
	global.Env().SystemConfig.APIConfig.Security.Password = "api-pass"

	api.HandleAPIMethod(api.GET, "/queue/:id/stats", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		user, password, ok := req.BasicAuth()
		if !ok || user != "api-user" || password != "api-pass" {
			t.Fatalf("unexpected basic auth: %v %s %s", ok, user, password)
		}
		if req.URL.Path != "/queue/test/stats" || req.URL.RawQuery != "size=20" {
			t.Fatalf("unexpected request url: %s", req.URL.String())
		}
		if ps.ByName("id") != "test" {
			t.Fatalf("unexpected route params: %#v", ps)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("served"))
	})

	handler := AgentAPI{}.requireLoginOrAccessToken(AgentAPI{}.proxyProtectedAPI)

	req := httptest.NewRequest(http.MethodGet, "/queue/test/stats?size=20", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	recorder := httptest.NewRecorder()
	handler(recorder, req, nil)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if recorder.Body.String() != "served" {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestProxyProtectedAPIAcceptsAPITokenHeader(t *testing.T) {
	t.Setenv("KEYSTORE_PATH", t.TempDir())

	if err := ensureAgentAccessToken(); err != nil {
		t.Fatalf("ensure access token: %v", err)
	}
	token, err := configcommon.LoadTokenFromKeystore(configcommon.AgentAccessTokenKeystoreKey)
	if err != nil {
		t.Fatalf("load access token: %v", err)
	}

	oldWebConfig := global.Env().SystemConfig.WebAppConfig
	t.Cleanup(func() {
		global.Env().SystemConfig.WebAppConfig = oldWebConfig
	})
	global.Env().SystemConfig.WebAppConfig.Security.Enabled = false

	handler := AgentAPI{}.requireLoginOrAccessToken(func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/queue/stats", nil)
	req.Header.Set(model.API_TOKEN, token)
	recorder := httptest.NewRecorder()
	handler(recorder, req, nil)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected X-API-TOKEN request to succeed, got %d", recorder.Code)
	}
}

func TestProxyProtectedAPIRejectsMissingTokenWhenLoginDisabled(t *testing.T) {
	oldWebConfig := global.Env().SystemConfig.WebAppConfig
	t.Cleanup(func() {
		global.Env().SystemConfig.WebAppConfig = oldWebConfig
	})
	global.Env().SystemConfig.WebAppConfig.Security.Enabled = false

	handler := AgentAPI{}.requireLoginOrAccessToken(func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/queue/stats", nil)
	recorder := httptest.NewRecorder()
	handler(recorder, req, nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing token to be rejected, got %d", recorder.Code)
	}
}
