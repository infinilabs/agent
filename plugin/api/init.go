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
	api.HandleAPIMethod(api.GET, "/elasticsearch/nodes", agentAPI.getESNodes)
	api.HandleAPIMethod(api.POST, "/elasticsearch/_auth", agentAPI.authESNode)
	//api.HandleAPIMethod(api.GET, "/agent/:agent_id/host/_basic", agentAPI.HostBasicInfo)
	//api.HandleAPIMethod(api.PUT, "/host/discover", agentAPI.HostDiscovered)
	//api.HandleAPIMethod(api.GET, "/agent/:agent_id/process/_elastic", agentAPI.ElasticProcessInfo)
	//api.HandleAPIMethod(api.GET, "/agent/logs/_list", agentAPI.LogsFileList)
	//api.HandleAPIMethod(api.POST, "/agent/logs/_read", agentAPI.ReadLogFile)
}