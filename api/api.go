/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"infini.sh/console/config"
	"infini.sh/framework/core/api"
)

type AgentAPI struct {
	api.Handler
	config.AppConfig
}

func (handler AgentAPI) Init() {
	api.HandleAPIMethod(api.GET, "/stats/_local", handler.LocalStats())
	api.HandleAPIMethod(api.GET, "/stats/_local", handler.LocalStats())

}
