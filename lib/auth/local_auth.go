/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	config2 "infini.sh/agent/config"
	"infini.sh/framework/core/agent"
)

// LocalAuthenticator get auth info from kv store
type LocalAuthenticator struct {
}

func NewLocalAuthenticator() (Authenticator, error) {
	return &LocalAuthenticator{}, nil
}

func (a *LocalAuthenticator) Auth(clusterName, endPoint string, ports ...int) (bool, *agent.BasicAuth)  {
	instanceInfo := config2.GetInstanceInfo()
	if instanceInfo == nil {
		return false, nil
	}
	for _, cluster := range instanceInfo.Clusters {
		if cluster.Name == clusterName {
			return true, &agent.BasicAuth{
				Username: cluster.UserName,
				Password: cluster.Password,
			}
		}
	}
	return false, nil
}