/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/config"
	"sync"
)

var authenticators []Authenticator
var once sync.Once

func RegisterAuth(auth Authenticator) {
	authenticators = append(authenticators, auth)
}

func Auth(clusterName, endPoint string, ports ...int) (bool, *agent.BasicAuth) {
	if len(authenticators) == 0 {
		log.Error("please init authenticators first")
		return false, nil
	}
	for _, authenticator := range authenticators {
		ok, basicAuth := authenticator.Auth(clusterName, endPoint, ports...)
		if ok {
			return true, basicAuth
		}
	}
	return false, nil
}

type Authenticator interface {
	Auth(clusterName, endPoint string, ports ...int) (bool, *agent.BasicAuth)
}

func InitDefaultAuthenticators(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	once.Do(func() {
		localAuth, err := NewLocalAuthenticator()
		if err != nil {
			log.Error(err)
		}
		RegisterAuth(localAuth)
		decryptAuth, err := NewDecryptAuthenticator(cfg)
		if err != nil {
			log.Error(err)
		}
		RegisterAuth(decryptAuth)
	})
	return true
}
