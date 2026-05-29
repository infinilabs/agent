/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/security"
	configcommon "infini.sh/framework/modules/configs/common"
	"net/http"
)

type AgentAPI struct {
	api.Handler
}

func InitAPI() {
	agentAPI := AgentAPI{}
	if err := ensureAgentAccessToken(); err != nil {
		log.Errorf("failed to ensure agent access token: %v", err)
	}

	// Discovery & logs — require login or agent access token
	api.HandleUIMethod(api.GET, "/agent/_info", agentAPI.requireLoginOrAccessToken(agentAPI.getAgentInfo), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.GET, "/elasticsearch/node/_discovery", agentAPI.requireLoginOrAccessToken(agentAPI.getESNodes), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/node/_info", agentAPI.requireLoginOrAccessToken(agentAPI.getESNodeInfo), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_list", agentAPI.requireLoginOrAccessToken(agentAPI.getElasticLogFiles), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	api.HandleUIMethod(api.POST, "/elasticsearch/logs/_read", agentAPI.requireLoginOrAccessToken(agentAPI.readElasticLogFile), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
	agentAPI.registerProtectedWebAPIRoutes()
	registerAgentReverseChannel()
}

func ensureAgentAccessToken() error {
	_, err := configcommon.EnsureTokenInKeystore(configcommon.AgentAccessTokenKeystoreKey)
	return err
}

func (a AgentAPI) getAgentInfo(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	a.WriteJSON(w, model.GetInstanceInfo(), http.StatusOK)
}

func (a AgentAPI) registerProtectedWebAPIRoutes() {
	api.RegisterProtectedUIRoutes(api.DefaultProtectedAPIRoutes, a.requireLoginOrAccessToken(a.proxyProtectedAPI), api.AllowOPTIONSS(), api.Feature(api.FeatureCORS))
}

func (a AgentAPI) proxyProtectedAPI(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	proxyReq := req.Clone(req.Context())
	applyAgentReverseLocalAPIAuth(proxyReq)
	api.ServeRegisteredAPIRequest(w, proxyReq)
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
			next(w, req, ps)
			return
		}
		a.WriteError(w, "unauthorized", http.StatusUnauthorized)
	}
}

func validateRequestAccessToken(req *http.Request) bool {
	return api.ValidateManagedAccessTokenRequest(req)
}
