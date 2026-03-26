/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package filter

import (
	"net/http"

	log "github.com/cihub/seelog"
	agentsecurity "infini.sh/agent/plugin/security"
	"infini.sh/framework/core/api"
	common "infini.sh/framework/core/api/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

func init() {
	api.RegisterUIFilter(&AuthFilter{})
}

type AuthFilter struct {
	api.Handler
}

func (f *AuthFilter) GetPriority() int {
	return 200
}

func (f *AuthFilter) ApplyFilter(
	method string,
	pattern string,
	options *api.HandlerOptions,
	next httprouter.Handle,
) httprouter.Handle {

	// skip if auth not required on this route, or auth is globally disabled
	if options == nil || (!options.RequireLogin && !options.OptionLogin) || !common.IsAuthEnable() {
		log.Debug(method, ",", pattern, ",skip auth")
		return next
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		claims, err := agentsecurity.ValidateLogin(w, r)

		if global.Env().IsDebug {
			log.Debug(method, ",", pattern, ",", util.MustToJSON(claims), ",", err)
		}

		if claims != nil && claims.IsValid() {
			// Only resolve permissions via the RBAC pipeline in managed mode,
			// where Cloud provides the authorization backend. In non-managed
			// mode there is no Easysearch and the framework's
			// SecurityBackendProvider would panic on orm.GetV2.
			if global.Env().SystemConfig.WebAppConfig.Security.Managed {
				claims.UserAssignedPermission = security.GetUserPermissions(claims)
			}
			r = r.WithContext(security.AddUserToContext(r.Context(), claims))
		}

		if !options.OptionLogin {
			if claims == nil {
				o := api.PrepareErrorJson("invalid login", 401)
				f.WriteJSON(w, o, 401)
				return
			}

			if err != nil {
				f.WriteErrorObject(w, err, 401)
				return
			}
		}

		next(w, r, ps)
	}
}
