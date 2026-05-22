/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/security"
	"net/http"
)

type AgentAPI struct {
	api.Handler
}

func InitAPI() {
	agentAPI := AgentAPI{}

	// Discovery & logs — require login
	api.HandleUIMethod(api.GET, "/elasticsearch/node/_discovery", agentAPI.requireLogin(agentAPI.getESNodes), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/node/_info", agentAPI.requireLogin(agentAPI.getESNodeInfo), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_list", agentAPI.requireLogin(agentAPI.getElasticLogFiles), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_read", agentAPI.requireLogin(agentAPI.readElasticLogFile), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))

	for _, route := range protectedAPIRoutes {
		api.HandleUIMethod(api.Method(route.method), route.path, agentAPI.requireLogin(agentAPI.proxyProtectedAPI))
	}
	registerAgentReverseChannel()
}

func (a AgentAPI) proxyProtectedAPI(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	api.ServeRegisteredAPIRequest(w, req)
}

func (a AgentAPI) requireLogin(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		if api.IsAuthEnable() {
			user, err := security.ValidateLogin(w, req)
			if err != nil {
				a.WriteError(w, err.Error(), http.StatusUnauthorized)
				return
			}
			req = req.WithContext(security.AddUserToContext(req.Context(), user))
		}

		next(w, req, ps)
	}
}
