/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"infini.sh/framework/core/api"
)

type AgentAPI struct {
	api.Handler
}

func InitAPI() {
	agentAPI := AgentAPI{}

	// Discovery & logs — require login
	api.HandleUIMethod(api.GET, "/elasticsearch/node/_discovery", agentAPI.getESNodes, api.RequireLogin(), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/node/_info", agentAPI.getESNodeInfo, api.RequireLogin(), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_list", agentAPI.getSearchLogFiles, api.RequireLogin(), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_read", agentAPI.readSearchLogFile, api.RequireLogin(), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	registerAgentReverseChannel()
}
