/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	"infini.sh/agent/config"
	"infini.sh/framework/core/api"
)

type AgentAPI struct {
	api.Handler
}

var UrlConsole string

const (
	UrlUploadHostInfo string = "agent/instance"
	UrlUploadNodeInfo        = "agent/xxx"
	UrlHearBeat              = ""
)

func (handler AgentAPI) Init() {
	getConsoleAddress()
	api.HandleAPIMethod(api.GET, "/stats/_local", handler.LocalStats())
	api.HandleAPIMethod(api.GET, "/stats/_local", handler.LocalStats())
}

func getConsoleAddress() {
	UrlConsole = fmt.Sprintf("%s://%s:%d", config.EnvConfig.Schema,
		config.EnvConfig.ConsoleConfig.Host,
		config.EnvConfig.ConsoleConfig.Port)
}
