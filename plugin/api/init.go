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

	//discovery local nodes
	api.HandleAPIMethod(api.GET, "/elasticsearch/node/_discovery", agentAPI.getESNodes)
	api.HandleAPIMethod(api.POST, "/elasticsearch/node/_info", agentAPI.getESNodeInfo) //get node info by connect to this node

	api.HandleAPIMethod(api.POST, "/elasticsearch/_register", agentAPI.registerESNode)
	api.HandleAPIMethod(api.POST, "/elasticsearch/logs/_list", agentAPI.getElasticLogFiles)
	api.HandleAPIMethod(api.POST, "/elasticsearch/logs/_read", agentAPI.readElasticLogFile)
}
