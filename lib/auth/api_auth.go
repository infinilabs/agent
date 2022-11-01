/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"infini.sh/agent/model"
	"infini.sh/framework/core/agent"
)

// ApiAuthenticator get auth info from Console
type ApiAuthenticator struct {
}


func NewAPIAuthenticator() (Authenticator, error) {
	return &ApiAuthenticator{}, nil
}

func (a *ApiAuthenticator) Auth(clusterName, endPoint string, ports ...int) (bool, *agent.BasicAuth, model.AuthType)  {
	//TODO 调用Console的api获取认证信息,重构之前的认证流程
	return false, nil, model.AuthTypeUnknown
}