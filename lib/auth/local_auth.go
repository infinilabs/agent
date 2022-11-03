/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	config2 "infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/util"
	log "src/github.com/cihub/seelog"
)

// LocalAuthenticator get auth info from kv store
type LocalAuthenticator struct {
}

func NewLocalAuthenticator() (Authenticator, error) {
	return &LocalAuthenticator{}, nil
}

func (a *LocalAuthenticator) Auth(clusterName string, endPoints ...string) (bool, *agent.BasicAuth, model.AuthType)  {
	instanceInfo := config2.GetInstanceInfo()
	if instanceInfo == nil {
		return false, nil, model.AuthTypeUnknown
	}
	for _, cluster := range instanceInfo.Clusters {
		if cluster.Name == clusterName && clusterName != "" {
			log.Debugf("local auth success, cluster: %s, endPoints: %s", clusterName, util.MustToJSON(endPoints))
			return true, &agent.BasicAuth{
				Username: cluster.UserName,
				Password: cluster.Password,
			}, model.AuthTypeLocal
		}
	}
	log.Debugf("local auth fail, cluster: %s, endPoints: %s", clusterName, util.MustToJSON(endPoints))
	return false, nil, model.AuthTypeUnknown
}