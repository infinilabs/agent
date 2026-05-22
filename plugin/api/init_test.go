package api

import (
	"testing"

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
