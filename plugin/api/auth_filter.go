/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"net/http"

	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
)

const apiTokenHeader = "X-API-TOKEN"

// tokenAuthFilter implements infini.sh/framework/core/api/filter.Filter.
// It is registered globally, so every route registered through
// api.HandleAPIMethod / HandleAPIFunc is protected.
// The /login route is the only public exception.
type tokenAuthFilter struct {
	api.Handler
}

func (f *tokenAuthFilter) FilterHttpRouter(pattern string, h httprouter.Handle) httprouter.Handle {
	if pattern == "/login" {
		return h
	}
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if r.Header.Get(apiTokenHeader) != authToken {
			o := api.PrepareErrorJson("invalid login", 401)
			f.WriteJSON(w, o, 401)
			return
		}
		h(w, r, ps)
	}
}

func (f *tokenAuthFilter) FilterHttpHandlerFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	if pattern == "/login" {
		return handler
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(apiTokenHeader) != authToken {
			o := api.PrepareErrorJson("invalid login", 401)
			f.WriteJSON(w, o, 401)
			return
		}
		handler(w, r)
	}
}
