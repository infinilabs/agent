/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"net/http"

	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
)

type loginRequest struct {
	Token string `json:"token"`
}

// loginHandler handles POST /login.
// This route is exempt from tokenAuthFilter so that callers can verify
// their token and receive a structured response.
func (h *AgentAPI) loginHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var body loginRequest
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.Token != authToken {
		o := api.PrepareErrorJson("invalid login", 401)
		h.WriteJSON(w, o, 401)
		return
	}
	h.WriteAckJSON(w, true, http.StatusOK, nil)
}
