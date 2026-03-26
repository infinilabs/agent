/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

const UserAccessTokenSessionName = "user_session_access_token"

// GenerateJWTAccessToken creates a HS256 JWT with 24h expiry for the given user session.
func GenerateJWTAccessToken(user *security.UserSessionInfo) (map[string]interface{}, error) {
	t := time.Now()
	if user.LastLogin.Timestamp == nil {
		user.LastLogin.Timestamp = &t
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, security.UserClaims{
		UserSessionInfo: user,
		RegisteredClaims: &jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})

	secret, err := GetSecret()
	if err != nil {
		return nil, errors.Errorf("failed to get secret key: %v", err)
	}

	tokenString, err := token.SignedString([]byte(secret))
	if tokenString == "" || err != nil {
		return nil, errors.Errorf("failed to generate access_token for user: %v", user)
	}

	data := util.MapStr{
		"access_token": tokenString,
		"expire_in":    time.Now().Unix() + 86400, // 24h
		"status":       "ok",
	}

	return data, nil
}

// AddUserAccessTokenToSession generates a JWT and stores it in the HTTP session cookie.
func AddUserAccessTokenToSession(w http.ResponseWriter, r *http.Request, user *security.UserSessionInfo) (error, map[string]interface{}) {
	if user == nil {
		panic("invalid user")
	}

	token, err := GenerateJWTAccessToken(user)
	if err != nil {
		return err, nil
	}

	api.ForceSetSession(w, r, UserAccessTokenSessionName, token["access_token"], true)
	return nil, token
}
