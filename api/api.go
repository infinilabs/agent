/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	"infini.sh/agent/model"
	"infini.sh/console/config"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/env"
)

type AgentAPI struct {
	api.Handler
	config.AppConfig
}

var UrlRegisterHost string

const (
	UrlUploadNodeInfo string = ""
)

func (handler AgentAPI) Init() {
	getConsoleAddress()
	api.HandleAPIMethod(api.GET, "/stats/_local", handler.LocalStats())
	api.HandleAPIMethod(api.GET, "/stats/_local", handler.LocalStats())
}

func getConsoleAddress() {
	console := model.ConsoleConfig{}
	env.ParseConfig("console", console)
	if console.TLS {
		UrlRegisterHost = fmt.Sprintf("https://%s:%d/", console.Host, console.Port)
	} else {
		UrlRegisterHost = fmt.Sprintf("http://%s:%d/", console.Host, console.Port)
	}
}
