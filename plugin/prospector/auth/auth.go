/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"infini.sh/agent/model"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/config"
	log "src/github.com/cihub/seelog"
)

var authenticators []Authenticator

func RegisterAuth(auth Authenticator) {
	authenticators = append(authenticators, auth)
}

func Auth(clusterName string, endPoints ...string) (bool, *agent.BasicAuth, model.AuthType) {
	if len(authenticators) == 0 {
		log.Error("please init authenticators first")
		return false, nil, model.AuthTypeUnknown
	}
	for _, authenticator := range authenticators {
		ok, basicAuth, authType := authenticator.Auth(clusterName, endPoints...)
		if ok {
			return true, basicAuth, authType
		}
	}
	return false, nil, model.AuthTypeUnknown
}

type Authenticator interface {
	Auth(clusterName string, endPoints ...string) (bool, *agent.BasicAuth, model.AuthType)
}

//func InitAuthenticators(cfg *config.Config) {
//	InitDecryptAuth(cfg)
//	InitAPIAuth(cfg)
//	InitLocalAuth(cfg)
//}

func InitLocalAuth(cfg *config.Config){
	localAuthCfg, err := cfg.Child("local_auth", -1)
	if err != nil {
		log.Error(err)
		return
	}
	localAuthEnable, err := localAuthCfg.Bool("enable", -1)
	if err != nil {
		log.Error(err)
		return
	}
	if !localAuthEnable {
		return
	}
	localAuth, err := NewLocalAuthenticator()
	if err != nil {
		log.Error(err)
		return
	}
	RegisterAuth(localAuth)
}

func InitAPIAuth(cfg *config.Config){
	apiAuthCfg, err := cfg.Child("api_auth", -1)
	if err != nil {
		log.Error(err)
		return
	}
	apiAuthEnable, err := apiAuthCfg.Bool("enable", -1)
	if err != nil {
		log.Error(err)
		return
	}
	if !apiAuthEnable {
		return
	}
	apiAuth, err := NewAPIAuthenticator()
	if err != nil {
		log.Error(err)
		return
	}
	RegisterAuth(apiAuth)
}

func InitDecryptAuth(cfg *config.Config, callback func(authInfo *agent.BasicAuth)) {
	decryptCfg, err := cfg.Child("decrypt_auth", -1)
	if err != nil {
		log.Error(err)
		return
	}
	decryptAuthEnable, err := decryptCfg.Bool("enable", -1)
	if err != nil {
		log.Error(err)
		return
	}
	if !decryptAuthEnable {
		return
	}
	decryptAuth, err := NewDecryptAuthenticator(decryptCfg, callback)
	if err != nil {
		log.Error(err)
		return
	}
	RegisterAuth(decryptAuth)
}