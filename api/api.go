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
	if config.EnvConfig.TLS {
		UrlRegisterHost = fmt.Sprintf("https://%s:%d/", config.EnvConfig.Host, config.EnvConfig.Port)
	} else {
		UrlRegisterHost = fmt.Sprintf("http://%s:%d/", config.EnvConfig.Host, config.EnvConfig.Port)
	}
}
