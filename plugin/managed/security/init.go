/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"fmt"
	"net/http"
	"net/url"

	log "github.com/cihub/seelog"

	agentsecurity "infini.sh/agent/plugin/security"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/security/oauth_client/provider"
)

const UserProfileBucketKey = "user_profile"

type APIHandler struct {
	api.Handler
}

func init() {

	global.RegisterFuncBeforeSetup(func() {

		if global.Env().SystemConfig.WebAppConfig.Security.Managed {

			// Register the "cloud" OAuth provider so the framework's
			// /sso/login/cloud and /sso/callback/cloud routes can fetch
			// user profiles from INFINI Cloud.
			provider.RegisterOAuthProvider("cloud", &CloudProfileProvider{})

			provider.RegisterOAuthCallback("*", provider.OAuthCallback{MatchFunc: func(s string) bool {
				return true
			}, CallbackFunc: func(w http.ResponseWriter, req *http.Request, ps httprouter.Params, profile *security.UserExternalProfile) bool {
				if profile == nil {
					panic("invalid oauth callback, profile can't be nil")
				}

				tenantID, userID := GetTenantInfo(profile)
				if global.Env().SystemConfig.WebAppConfig.Security.Managed {
					if tenantID == "" || userID == "" {
						panic("invalid oauth callback, tenant id or user id can't be empty")
					}
				}

				// build local profile from the cloud-provided identity
				var user = &security.UserProfile{
					Name: profile.Name,
				}
				user.ID = userID
				user.Avatar = profile.Avatar
				user.Email = profile.Email

				log.Debug("tenant ", tenantID, " user ", userID, " login: ", profile.Login, ", id:", profile.ID)

				sessionInfo := security.UserSessionInfo{}
				sessionInfo.Provider = profile.AuthProvider
				sessionInfo.Login = profile.Login
				sessionInfo.UserID = userID

				if global.Env().IsDebug {
					log.Debug("external profile: ", util.ToJson(profile, true))
				}

				teamsV, ok := profile.GetSystemValue(orm.TeamsIDKey)
				if ok {
					if teamsV != nil {
						teamsID, ok := teamsV.([]string)
						if ok {
							if len(teamsID) > 0 {
								user.SetSystemValue(orm.TeamsIDKey, teamsID)
								sessionInfo.Set(orm.TeamsIDKey, teamsID)
								if global.Env().IsDebug {
									log.Debug("tenant ", tenantID, " user ", userID, " login: ", profile.Login, ", teamsID:", teamsID)
								}
							}
						}
					}
				}

				SetTenantInfo(user, tenantID, userID)
				SetUserSessionWithTenantInfo(&sessionInfo, tenantID, userID)

				profileKey := fmt.Sprintf("%v:%v", tenantID, userID)

				// cache profile locally for each login
				err := kv.AddValue(UserProfileBucketKey, []byte(profileKey), util.MustToJSONBytes(user))
				if err != nil {
					panic(err)
				}

				err, _ = agentsecurity.AddUserAccessTokenToSession(w, req, &sessionInfo)
				if err != nil {
					panic(err)
				}

				redirectURL := "/"
				session, err := api.GetSessionStore(req, "oauth-session")
				if session == nil {
					panic("session is nil")
				}
				if v, ok := session.Values["redirect_url"].(string); ok && v != "" {
					redirectURL = v
				}

				if pathPrefix := req.URL.Query().Get("oauth_proxy_prefix"); pathPrefix != "" {
					redirectURL, err = url.JoinPath(pathPrefix, redirectURL)
					if err != nil {
						panic(err)
					}
					log.Debug("redirect to proxy path: ", redirectURL)
				}

				http.Redirect(w, req, redirectURL, 301)
				return false
			}})
		}
	})

}
