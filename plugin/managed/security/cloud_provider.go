/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"golang.org/x/oauth2"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/security/oauth_client/provider"
)

// CloudProfileProvider implements provider.OAuthProvider for INFINI Cloud SSO.
// Replicated from framework/plugins/enterprise/managed/profile.go to avoid
// importing the managed package (which pulls in Coco-specific init logic).
type CloudProfileProvider struct {
	api.Handler
}

// GetUserProfileResponse is the JSON envelope returned by the Cloud profile endpoint.
type GetUserProfileResponse struct {
	Source CloudUserProfile `json:"_source,omitempty"`
	Found  bool             `json:"found,omitempty"`
	ID     string           `json:"_id,omitempty"`
}

// CloudUserProfile represents the user profile payload from INFINI Cloud.
type CloudUserProfile struct {
	orm.ORMObjectBase

	Tenant struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"tenant"`

	Tenants []struct {
		Id     string `json:"id"`
		Name   string `json:"name"`
		Domain string `json:"domain"`
		Status string `json:"status"`
		Teams  []struct {
			Id             string `json:"id"`
			Name           string `json:"name"`
			IsDefault      bool   `json:"is_default"`
			DefaultProject struct {
				Id   string `json:"id"`
				Name string `json:"name"`
			} `json:"default_project"`
		} `json:"teams"`
		Roles []struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"roles"`
	} `json:"tenants"`

	AuthProvider  string      `json:"auth_provider,omitempty"`
	Username      string      `json:"username,omitempty"`
	Password      string      `json:"password,omitempty"`
	Nickname      string      `json:"nickname,omitempty"`
	Email         string      `json:"email,omitempty"`
	EmailVerified bool        `json:"email_verified"`
	Phone         string      `json:"phone,omitempty"`
	Tags          []string    `json:"tags,omitempty"`
	AvatarUrl     string      `json:"avatar_url,omitempty"`
	Payload       interface{} `json:"payload,omitempty"`
}

func (u *CloudUserProfile) GetDisplayName() string {
	if u.Nickname != "" {
		return u.Nickname
	}
	if u.Username != "" {
		return u.Username
	}
	if u.Email != "" {
		if util.ContainStr(u.Email, "@") {
			arr := strings.Split(u.Email, "@")
			return arr[0]
		}
		return u.Email
	}
	if u.ID != "" {
		return u.ID
	}
	return "N/A"
}

// GetProfile fetches the user profile from INFINI Cloud using the OAuth2 token
// and builds a UserExternalProfile with tenant/team information.
func (handler *CloudProfileProvider) GetProfile(ctx *orm.Context, appConfig *config.OAuthConfig, cfg *oauth2.Config, tkn *oauth2.Token) *security.UserExternalProfile {

	httpClient := api.GetHttpClient("oauth_" + appConfig.Provider)
	ctxHTTP := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)

	client := cfg.Client(ctxHTTP, tkn)
	if appConfig.ProfileUrl == "" {
		panic("invalid profile URL")
	}

	resp, err := client.Get(appConfig.ProfileUrl)
	if err != nil {
		panic(fmt.Errorf("failed to fetch user info: %w", err))
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		panic(errors.NewWithHTTPCode(401, fmt.Sprintf("unexpected status code fetching user info: %d", resp.StatusCode)))
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("failed to read user info response: %w", err))
	}

	var userInfo GetUserProfileResponse
	if err := util.FromJSONBytes(bytes, &userInfo); err != nil {
		panic(errors.NewWithHTTPCode(401, fmt.Sprintf("failed to parse user info: %v", err)))
	}

	profile := security.UserExternalProfile{}

	tenantID := userInfo.Source.Tenant.Id
	userID := userInfo.ID
	if tenantID == "" {
		if len(userInfo.Source.Tenants) == 0 {
			panic(errors.NewWithHTTPCode(401, "invalid tenant info"))
		}
		tenantID = userInfo.Source.Tenants[0].Id
		log.Warnf("invalid tenant id, pick the first tenant info: %v, %v", tenantID, string(bytes))
	}

	if userID == "" {
		panic(errors.NewWithHTTPCode(401, "invalid user id"))
	}

	teamsID := []string{}
	for _, tenant := range userInfo.Source.Tenants {
		if tenant.Id == tenantID {
			for _, team := range tenant.Teams {
				teamsID = append(teamsID, team.Id)
			}
		}
	}

	profile.ID = provider.GetExternalUserProfileID("cloud", userID)
	profile.AuthProvider = "cloud"
	profile.Login = userID
	profile.Email = userInfo.Source.Email
	profile.Name = userInfo.Source.GetDisplayName()
	profile.Avatar = userInfo.Source.AvatarUrl
	profile.Payload = userInfo.Source
	t := time.Now()
	profile.Created = &t
	profile.Updated = &t

	profile.SetOwnerID(userID)
	profile.SetSystemValue(orm.TenantIDKey, tenantID)

	if len(teamsID) > 0 {
		profile.SetSystemValue(orm.TeamsIDKey, teamsID)
	}

	return &profile
}

// GetOauthConfig returns nil — Agent always uses the static YAML-configured OAuth config.
func (handler *CloudProfileProvider) GetOauthConfig() *config.OAuthConfig {
	return nil
}
