/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/golang-jwt/jwt"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
)

// ValidateLoginByAccessTokenSession checks for a JWT stored in the HTTP session cookie.
func ValidateLoginByAccessTokenSession(w http.ResponseWriter, r *http.Request) (claims *security.UserClaims, err error) {
	exists, sessToken := api.GetSession(w, r, UserAccessTokenSessionName)
	if !exists || sessToken == nil {
		return nil, errors.Error("invalid session")
	}

	tokenStr, ok := sessToken.(string)
	if !ok {
		return nil, errors.New("authorization token is empty")
	}

	claims = security.NewUserClaims()

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		secret, err := GetSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to get secret key: %v", err)
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if token.Valid {
		if !claims.IsValid() {
			return nil, errors.New("user info is not valid")
		}
		if !claims.VerifyExpiresAt(time.Now(), true) {
			return nil, errors.New("token is expired")
		}
	}

	return claims, nil
}

// ValidateLoginByAuthorizationHeader checks for a Bearer JWT in the Authorization header.
func ValidateLoginByAuthorizationHeader(w http.ResponseWriter, r *http.Request) (claims *security.UserClaims, err error) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		return nil, errors.Error("Authorization not found")
	}

	fields := strings.Fields(authorization)
	if len(fields) != 2 || fields[0] != "Bearer" {
		return nil, errors.New("authorization header is invalid")
	}
	tokenString := fields[1]

	claims = security.NewUserClaims()

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		secret, err := GetSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to get secret key: %v", err)
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if token.Valid {
		if !claims.IsValid() {
			return nil, errors.New("user info is not valid")
		}
		if !claims.VerifyExpiresAt(time.Now(), true) {
			return nil, errors.New("token is expired")
		}
	}

	if claims == nil {
		return nil, errors.Error("invalid claims")
	}

	return claims, nil
}

// ValidateLogin tries authentication in order: session JWT → Bearer header.
// Returns the authenticated UserSessionInfo or an error.
func ValidateLogin(w http.ResponseWriter, r *http.Request) (session *security.UserSessionInfo, err error) {

	claims, err := ValidateLoginByAccessTokenSession(w, r)

	if claims == nil || !claims.UserSessionInfo.IsValid() {
		claims, err = ValidateLoginByAuthorizationHeader(w, r)
	}

	if claims == nil || !claims.UserSessionInfo.IsValid() || err != nil {
		if global.Env().IsDebug {
			log.Debugf("validate login failed: %v", err)
		}
		err = errors.Errorf("invalid user info: %v", err)
		return
	}

	return claims.UserSessionInfo, nil
}
