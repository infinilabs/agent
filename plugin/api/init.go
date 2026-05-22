/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/security"
	"net/http"
	"strings"
)

type AgentAPI struct {
	api.Handler
}

func InitAPI() {
	agentAPI := AgentAPI{}

	// Discovery & logs — require login or agent access token
	api.HandleUIMethod(api.GET, "/agent/_info", agentAPI.requireLoginOrAccessToken(agentAPI.getAgentInfo), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.GET, "/elasticsearch/node/_discovery", agentAPI.requireLoginOrAccessToken(agentAPI.getESNodes), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/node/_info", agentAPI.requireLoginOrAccessToken(agentAPI.getESNodeInfo), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_list", agentAPI.requireLoginOrAccessToken(agentAPI.getElasticLogFiles), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_read", agentAPI.requireLoginOrAccessToken(agentAPI.readElasticLogFile), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))

	for _, route := range protectedAPIRoutes {
		api.HandleUIMethod(api.Method(route.method), route.path, agentAPI.requireLoginOrAccessToken(agentAPI.proxyProtectedAPI))
	}
	registerAgentReverseChannel()
}

func (a AgentAPI) getAgentInfo(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	a.WriteJSON(w, model.GetInstanceInfo(), http.StatusOK)
}

func (a AgentAPI) proxyProtectedAPI(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	api.ServeRegisteredAPIRequest(w, req)
}

func (a AgentAPI) requireLoginOrAccessToken(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		if validateRequestAccessToken(req) {
			next(w, req, ps)
			return
		}
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

func validateRequestAccessToken(req *http.Request) bool {
	if req == nil {
		return false
	}
	value := strings.TrimSpace(req.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return false
	}
	return validateAgentAccessToken(strings.TrimSpace(value[7:]))
}
