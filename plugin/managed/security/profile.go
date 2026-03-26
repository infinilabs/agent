/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"fmt"
	"net/http"

	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

func init() {
	global.RegisterFuncBeforeSetup(func() {
		if global.Env().SystemConfig.WebAppConfig.Security.Managed {
			h := APIHandler{}
			api.HandleUIMethod(api.GET, "/account/profile", h.profileHandler, api.RequireLogin())
		}
	})
}

// profileHandler returns the KV-cached user profile for SSO logins.
func (h APIHandler) profileHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if !api.IsAuthEnable() {
		h.WriteError(w, "auth is not enabled", http.StatusInternalServerError)
		return
	}

	reqUser, err := security.GetUserFromContext(r.Context())
	if err != nil || reqUser == nil {
		h.WriteError(w, "invalid user", http.StatusUnauthorized)
		return
	}

	tenantID, userID := GetTenantInfoFromUserSession(reqUser)
	if tenantID == "" || userID == "" {
		h.WriteError(w, "invalid user", http.StatusUnauthorized)
		return
	}

	profileKey := fmt.Sprintf("%v:%v", tenantID, userID)
	data, err := kv.GetValue(UserProfileBucketKey, []byte(profileKey))
	if err != nil {
		h.WriteError(w, "failed to read profile", http.StatusInternalServerError)
		return
	}

	p := &security.UserProfile{}
	util.MustFromJSONBytes(data, p)

	h.WriteJSON(w, p, http.StatusOK)
}
