/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"net/http"

	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

type APIHandler struct {
	api.Handler
}

func init() {
	global.RegisterFuncBeforeSetup(func() {
		initAuthToken()

		h := APIHandler{}

		// If this agent is not managed, users log in via the log in token.
		if !global.Env().SystemConfig.WebAppConfig.Security.Managed {
			api.HandleUIMethod(api.POST, "/login", h.tokenLoginHandler)
			api.HandleUIMethod(api.GET, "/account/profile", h.tokenProfileHandler, api.RequireLogin())
		}

		api.HandleUIMethod(api.GET, "/account/logout", h.logoutHandler, api.OptionLogin())
		api.HandleUIMethod(api.POST, "/account/logout", h.logoutHandler, api.OptionLogin())

	})
}

const (
	AgentTokenProvider = "agent_token"
	agentAdminLogin    = "Admin"
	agentAdminUserID   = "admin"
)

type loginRequest struct {
	Token string `json:"token"`
}

// tokenLoginHandler handles POST /login.
// Validates the startup-generated token and creates a session JWT on success.
func (h APIHandler) tokenLoginHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var body loginRequest
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if body.Token != loginToken {
		h.WriteError(w, "invalid token", http.StatusUnauthorized)
		return
	}

	sessionInfo := &security.UserSessionInfo{}
	sessionInfo.Provider = AgentTokenProvider
	sessionInfo.Login = agentAdminLogin
	sessionInfo.UserID = agentAdminUserID
	sessionInfo.Roles = []string{security.RoleAdmin}

	err, token := AddUserAccessTokenToSession(w, req, sessionInfo)
	if err != nil {
		h.ErrorInternalServer(w, "failed to create session")
		return
	}

	h.WriteOKJSON(w, token)
}

// logoutHandler handles GET/POST /account/logout.
func (h APIHandler) logoutHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	api.DestroySession(w, r)
	h.WriteOKJSON(w, util.MapStr{
		"status": "ok",
	})
}

// tokenProfileHandler returns a synthetic admin profile for token-based logins.
func (h APIHandler) tokenProfileHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !api.IsAuthEnable() {
		h.WriteError(w, "auth is not enabled", http.StatusInternalServerError)
		return
	}

	reqUser, err := security.GetUserFromContext(r.Context())
	if err != nil || reqUser == nil {
		h.WriteError(w, "invalid user", http.StatusUnauthorized)
		return
	}

	p := &security.UserProfile{
		Name:  reqUser.Login,
		Roles: []string{security.RoleAdmin},
	}
	p.ID = reqUser.UserID
	p.Permissions = security.MustGetPermissionKeysByRole([]string{security.RoleAdmin})

	h.WriteJSON(w, p, http.StatusOK)
}
