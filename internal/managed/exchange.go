package managed

import (
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/keystore"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	ucfg "infini.sh/framework/lib/go-ucfg"
	"infini.sh/framework/modules/configs/client"
	"infini.sh/framework/modules/security/access_token"
)

const (
	tokenExchangeAPI       = "/instance/_exchange_token"
	agentAPIAccessTokenKey = "AGENT_API_ACCESS_TOKEN"
	managerAccessTokenKey  = "CONFIGS_MANAGER_ACCESS_TOKEN"
)

type tokenExchangeRequest struct {
	InstanceID    string `json:"instance_id,omitempty"`
	AgentAPIToken string `json:"agent_api_token,omitempty"`
}

type tokenExchangeResponse struct {
	ManagerAPIToken string `json:"manager_api_token,omitempty"`
}

func getOrCreateAgentAPIToken(instance model.Instance) (string, error) {
	if tokenBytes, err := keystore.GetValue(agentAPIAccessTokenKey); err == nil {
		token := strings.TrimSpace(string(tokenBytes))
		if token != "" {
			return token, nil
		}
	}

	user := &security.UserSessionInfo{
		Provider: "managed_agent",
		Login:    instance.ID,
	}
	user.SetUserID(instance.ID)
	user.Set("instance_id", instance.ID)
	user.Set("instance_name", instance.Name)
	user.Set("endpoint", instance.Endpoint)

	res, err := access_token.CreateAPIToken(user, fmt.Sprintf("%s agent api", instance.Name), "managed_agent_api", -1, nil)
	if err != nil {
		return "", err
	}
	token, ok := res["access_token"].(string)
	if !ok || strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("failed to create agent api access token")
	}
	if err := keystore.SetValue(agentAPIAccessTokenKey, []byte(token)); err != nil {
		return "", err
	}
	return token, nil
}

func ExchangeTokens() error {
	if !global.Env().SystemConfig.Configs.Managed || len(global.Env().SystemConfig.Configs.Servers) == 0 {
		return nil
	}
	managerAPIToken := strings.TrimSpace(global.Env().SystemConfig.Configs.ManagerConfig.AccessToken.Get())
	if managerAPIToken == "" {
		return nil
	}

	instance := model.GetInstanceInfo()
	agentAPIToken, err := getOrCreateAgentAPIToken(instance)
	if err != nil {
		return err
	}

	reqBody := tokenExchangeRequest{
		InstanceID:    instance.ID,
		AgentAPIToken: agentAPIToken,
	}
	req := util.Request{
		Method:      util.Verb_POST,
		Path:        tokenExchangeAPI,
		ContentType: "application/json",
		Body:        util.MustToJSONBytes(reqBody),
	}
	server, res, err := client.DoManagerRequest(&req)
	if err != nil {
		return err
	}
	if res == nil {
		return fmt.Errorf("empty response from %s", server)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("token exchange failed on %s, status: %d, body: %s", server, res.StatusCode, string(res.Body))
	}

	resp := tokenExchangeResponse{}
	if err := util.FromJSONBytes(res.Body, &resp); err != nil {
		return err
	}
	if strings.TrimSpace(resp.ManagerAPIToken) == "" {
		return fmt.Errorf("manager api token is empty")
	}
	if err := keystore.SetValue(managerAccessTokenKey, []byte(resp.ManagerAPIToken)); err != nil {
		return err
	}
	global.Env().SystemConfig.Configs.ManagerConfig.AccessToken = ucfg.SecretString(resp.ManagerAPIToken)
	log.Infof("exchanged managed access token from %s", server)
	return nil
}

func RegisterTokenExchangeCallback() {
	client.AddPostRegisterHook(ExchangeTokens)
}
