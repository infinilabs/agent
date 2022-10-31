/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/config"
	"os"
)

const ESUserName = "elastic"

// DecryptAuthenticator get auth info from encrypted config file
type DecryptAuthenticator struct {
	cfg         DecryptConfig `config:"decrypt_auth"`
	userName    string
	encPassword string
}

func NewDecryptAuthenticator(c *config.Config) (Authenticator, error) {
	cfg := DecryptConfig{}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of node prospector processor: %s", err)
	}

	da := &DecryptAuthenticator{
		cfg: cfg,
	}
	err := da.LoadAuthFile()
	if err != nil {
		return nil, err
	}
	return da, nil
}

type DecryptConfig struct {
	Enable  bool   `json:"enable" config:"enable"`
	Path    string `json:"path" config:"path"`
	EncKey  string `json:"enc_key" config:"enc_key"`
	EncIV   string `json:"enc_iv" config:"enc_iv"`
	EncType string `json:"enc_type" config:"enc_type"`
}

func (a *DecryptAuthenticator) Auth(clusterName, endPoint string, ports ...int) (bool, *agent.BasicAuth) {

	if !a.cfg.Enable || clusterName == "" || endPoint == "" || len(ports) == 0 {
		return false, nil
	}
	pwd := decrypt(a.encPassword, a.cfg.EncKey, a.cfg.EncIV, a.cfg.EncType)
	return true, &agent.BasicAuth{
		Username: a.userName,
		Password: pwd,
	}
}

func (a *DecryptAuthenticator) LoadAuthFile() error {
	content, err := os.ReadFile(a.cfg.Path)
	if err != nil {
		return err
	}
	var authInfo map[string]string
	err = json.Unmarshal(content, &authInfo)
	if err != nil {
		return err
	}
	encPWD, ok := authInfo[ESUserName]
	if !ok {
		return errors.New(fmt.Sprintf("can not find auth info from: %s", a.cfg.Path))
	}
	a.encPassword = encPWD
	a.userName = ESUserName
	return nil
}
