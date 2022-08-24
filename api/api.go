/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"infini.sh/framework/core/api"
)

type AgentAPI struct {
	api.Handler
}

func (handler *AgentAPI) Init() {
	api.HandleAPIMethod(api.GET, "/stats/_local", handler.LocalStats())
	api.HandleAPIMethod(api.GET, "/task/:node_id/_enable", handler.EnableTask())
	api.HandleAPIMethod(api.GET, "/task/:node_id/_disable", handler.DisableTask())
	api.HandleAPIMethod(api.DELETE, "/manage/:agent_id", handler.DeleteAgent())
	api.HandleAPIMethod(api.POST, "/manage/register/:agent_id", handler.RegisterCallBack())
	api.HandleAPIMethod(api.GET, "/host/:agent_id/_basic", handler.HostBasicInfo())
	api.HandleAPIMethod(api.GET, "/host/:agent_id/_usage", handler.HostUsageInfo())
}
