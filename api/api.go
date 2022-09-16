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
	api.HandleAPIMethod(api.POST, "/task/_extra", handler.ExtraTask())
	api.HandleAPIMethod(api.GET, "/task/:node_id/_disable", handler.DisableTask())
	api.HandleAPIMethod(api.DELETE, "/manage/:agent_id", handler.DeleteAgent())
	api.HandleAPIMethod(api.POST, "/manage/register/:agent_id", handler.RegisterCallBack())
	api.HandleAPIMethod(api.GET, "/agent/:agent_id/host/_basic", handler.HostBasicInfo())
	api.HandleAPIMethod(api.GET, "/agent/:agent_id/host/usage/:category", handler.HostUsageInfo())
	api.HandleAPIMethod(api.PUT, "/host/discover", handler.HostDiscovered())
	api.HandleAPIMethod(api.GET, "/agent/:agent_id/process/_elastic", handler.ElasticProcessInfo())
	api.HandleAPIMethod(api.GET, "/agent/:agent_id/logs/list/:node_id/:suffix", handler.LogsFileList())
	api.HandleAPIMethod(api.POST, "/agent/:agent_id/logs/_read", handler.ReadLogFile())
}
