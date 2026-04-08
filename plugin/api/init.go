/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/modules/security/http_filters"
)

type AgentAPI struct {
	api.Handler
}

func InitAPI() {
	agentAPI := AgentAPI{}

	// Discovery & logs — require login
	api.HandleUIMethod(api.GET, "/elasticsearch/node/_discovery", agentAPI.getESNodes, api.RequireLogin(), api.Feature(http_filters.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/elasticsearch/node/_discovery", agentAPI.getESNodes, api.RequireLogin(), api.Feature(http_filters.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/node/_info", agentAPI.getESNodeInfo, api.RequireLogin(), api.Feature(http_filters.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/elasticsearch/node/_info", agentAPI.getESNodeInfo, api.RequireLogin(), api.Feature(http_filters.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_list", agentAPI.getElasticLogFiles, api.RequireLogin(), api.Feature(http_filters.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/elasticsearch/logs/_list", agentAPI.getElasticLogFiles, api.RequireLogin(), api.Feature(http_filters.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_read", agentAPI.readElasticLogFile, api.RequireLogin(), api.Feature(http_filters.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/elasticsearch/logs/_read", agentAPI.readElasticLogFile, api.RequireLogin(), api.Feature(http_filters.FeatureCORS))
}
