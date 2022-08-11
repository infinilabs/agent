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

const (
	UrlUploadHostInfo string = "agent/instance"
	UrlUploadNodeInfo        = "agent/instance/:instance_id/_nodes"
	UrlHearBeat              = "agent/instance/:instance_id/_heartbeat"
	UrlGetHostInfo           = "/agent/instance/:instance_id"
)

func (handler *AgentAPI) Init() {
	api.HandleAPIMethod(api.GET, "/stats/_local", handler.LocalStats())
	api.HandleAPIMethod(api.GET, "/task/:node_id/_enable", handler.EnableTask())
	api.HandleAPIMethod(api.GET, "/task/:node_id/_disable", handler.DisableTask())
	api.HandleAPIMethod(api.GET, "/manage/:agent_id/_delete", handler.DeleteAgent())
}
