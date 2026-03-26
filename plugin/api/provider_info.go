/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"net/http"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

const agentSSOLoginURL = "/sso/login/cloud?product=agent"

// providerInfoHandler handles GET /provider/_info.
// Returns managed mode flag and SSO URL (when managed).
func (h *AgentAPI) providerInfoHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	managed := global.Env().SystemConfig.WebAppConfig.Security.Managed

	ssoURL := ""
	if managed {
		ssoURL = agentSSOLoginURL
	}

	h.WriteJSON(w, util.MapStr{
		"managed": managed,
		"auth_provider": util.MapStr{
			"sso": util.MapStr{
				"url": ssoURL,
			},
		},
	}, 200)
}
