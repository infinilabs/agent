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
	api.HandleAPIMethod(api.GET, "/elasticsearch/_nodes", agentAPI.getESNodes)
	api.HandleAPIMethod(api.POST, "/elasticsearch/_auth", agentAPI.authESNode)
	api.HandleAPIMethod(api.POST, "/elasticsearch/_register", agentAPI.registerESNode)
	api.HandleAPIMethod(api.GET, "/agent/host/_basic", agentAPI.getHostBasicInfo)
	//api.HandleAPIMethod(api.PUT, "/host/discover", agentAPI.HostDiscovered)
	//api.HandleAPIMethod(api.GET, "/agent/:agent_id/process/_elastic", agentAPI.ElasticProcessInfo)
	api.HandleAPIMethod(api.POST, "/agent/logs/elastic/list", agentAPI.getElasticLogFiles)
	api.HandleAPIMethod(api.POST, "/agent/logs/elastic/_read", agentAPI.readElasticLogFile)
	api.HandleAPIMethod(api.GET, "/agent/_info", agentAPI.getAgentInfo)
	api.HandleAPIMethod(api.POST, "/agent/config", agentAPI.saveDynamicConfig)

}