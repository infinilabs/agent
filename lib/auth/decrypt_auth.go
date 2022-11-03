/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	"github.com/fsnotify/fsnotify"
	"infini.sh/agent/model"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/util"
	"os"
)

// DecryptAuthenticator get auth info from encrypted config file
type DecryptAuthenticator struct {
	cfg         DecryptConfig `config:"decrypt_auth"`
	userName    string
	encPassword string
	updateCallback func(authInfo *agent.BasicAuth)
}

func NewDecryptAuthenticator(c *config.Config, handle func(authInfo *agent.BasicAuth)) (Authenticator, error) {
	cfg := DecryptConfig{}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of node prospector processor: %s", err)
	}

	da := &DecryptAuthenticator{
		cfg: cfg,
		updateCallback: handle,
	}
	if !da.loadEnvConfig() {
		return nil, errors.New("please config environment variables")
	}
	err := da.loadAuthFile()
	if err != nil {
		return nil, err
	}
	da.registerAuthFileWatcher()
	return da, nil
}

type DecryptConfig struct {
	Enable  bool   `json:"enable" config:"enable"`
	Path    string `json:"path" config:"path"`
	EncKey  string `json:"enc_key" config:"enc_key"`
	EncIV   string `json:"enc_iv" config:"enc_iv"`
	EncType string `json:"enc_type" config:"enc_type"`
	ESUserName string `json:"es_username" config:"es_username"`
}

func (a *DecryptAuthenticator) Auth(clusterName string, endPoints ...string) (bool, *agent.BasicAuth, model.AuthType) {

	if !a.cfg.Enable || clusterName == "" || len(endPoints) == 0 {
		return false, nil, model.AuthTypeUnknown
	}
	pwd := opensslAesDecrypt(a.encPassword, a.cfg.EncKey, a.cfg.EncIV, a.cfg.EncType)
	if !a.validate(a.userName, pwd, endPoints...) {
		log.Debugf("decrypt auth fail, cluster: %s, username: %s, pwd: %s, endPoints: %s", clusterName, a.userName, a.encPassword, util.MustToJSON(endPoints))
		return false, nil, model.AuthTypeUnknown
	}
	log.Debugf("decrypt auth success, cluster: %s, username: %s, pwd: %s, endPoints: %s", clusterName, a.userName, a.encPassword, util.MustToJSON(endPoints))
	return true, &agent.BasicAuth{
		Username: a.userName,
		Password: pwd,
	}, model.AuthTypeEncrypt
}

func (a *DecryptAuthenticator) validate(userName string, password string, endPoints ...string) bool {
	var req *util.Request
	var clusterUUID string
	for _, url := range endPoints {
		req = util.NewGetRequest(url, nil)
		req.SetBasicAuth(userName, password)
		result, err := util.ExecuteRequest(req)
		if err != nil {
			//log.Error(err)
			continue
		}
		clusterUUID, err = jsonparser.GetString(result.Body, "cluster_uuid")
		if err != nil {
			//log.Error(err)
			continue
		}
		if clusterUUID != "" {
			return true
		}
	}
	return false
}

func (a *DecryptAuthenticator) loadAuthFile() error {
	content, err := os.ReadFile(a.cfg.Path)
	if err != nil {
		return err
	}
	var authInfo map[string]string
	err = json.Unmarshal(content, &authInfo)
	if err != nil {
		return err
	}
	encPWD, ok := authInfo[a.cfg.ESUserName]
	if !ok {
		return errors.New(fmt.Sprintf("can not find auth info from: %s", a.cfg.Path))
	}
	a.encPassword = encPWD
	a.userName = a.cfg.ESUserName
	return nil
}

func (a *DecryptAuthenticator) loadEnvConfig() bool {
	encKey := os.Getenv("AUTH_ENC_KEY")
	encIV := os.Getenv("AUTH_ENC_IV")
	encType := os.Getenv("AUTH_ENC_TYPE")
	if encKey == "" || encIV == "" || encType == "" {
		return false
	}
	a.cfg.EncIV = encIV
	a.cfg.EncKey = encKey
	a.cfg.EncType = encType
	return true
}

func (a *DecryptAuthenticator) registerAuthFileWatcher()  {
	config.AddPathToWatch(a.cfg.Path, func(file string, op fsnotify.Op) {
		log.Debug("auth file changed!!")
		err := a.loadAuthFile()
		if err != nil {
			log.Error("load auth file failed, %s", err)
			return
		}
		pwd := opensslAesDecrypt(a.encPassword, a.cfg.EncKey, a.cfg.EncIV, a.cfg.EncType)
		if pwd == "" {
			log.Error("decrypt auth file failed")
			return
		}
		a.updateCallback(&agent.BasicAuth{
			Username: a.cfg.ESUserName,
			Password: pwd,
		})
	})
}
