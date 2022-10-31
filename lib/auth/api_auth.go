/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import "infini.sh/framework/core/agent"

// ApiAuthenticator get auth info from Console
type ApiAuthenticator struct {
}

func (a *ApiAuthenticator) Auth(clusterName, endPoint string, ports ...int) (bool, *agent.BasicAuth)  {
	//TODO 调用Console的api获取认证信息,重构之前的认证流程
	return false, nil
}