/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import "infini.sh/framework/core/api"

type AgentAPI struct {
	api.Handler
}

func InitAPI() {
	agentAPI := AgentAPI{}

	// Public endpoints (no RequireLogin)
	api.HandleUIMethod(api.GET, "/provider/_info", agentAPI.providerInfoHandler, api.AllowPublicAccess())

	// Discovery & logs — require login
	api.HandleUIMethod(api.GET, "/elasticsearch/node/_discovery", agentAPI.getESNodes, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/elasticsearch/node/_info", agentAPI.getESNodeInfo, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_list", agentAPI.getElasticLogFiles, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_read", agentAPI.readElasticLogFile, api.RequireLogin())
}
