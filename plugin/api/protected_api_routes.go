package api

import (
	"net/http"
	"net/url"
	"strings"

	httprouter "infini.sh/framework/core/api/router"
)

type protectedAPIRoute struct {
	method string
	path   string
}

type protectedAPIRouter struct {
	router *httprouter.Router
}

var noopProtectedAPIRouteHandle = func(http.ResponseWriter, *http.Request, httprouter.Params) {}

var protectedAPIRoutes = []protectedAPIRoute{
	{http.MethodGet, "/stats"},
	{http.MethodGet, "/queue/stats"},
	{http.MethodGet, "/queue/:id/stats"},
	{http.MethodGet, "/queue/:id/_scroll"},
	{http.MethodDelete, "/queue/:id"},
	{http.MethodDelete, "/queue/_search"},
	{http.MethodPut, "/queue/:id/consumer/:consumer_id/offset"},
	{http.MethodGet, "/queue/:id/consumer/:consumer_id/offset"},
	{http.MethodDelete, "/queue/:id/consumer/:consumer_id"},
	{http.MethodDelete, "/queue/consumer/_search"},
	{http.MethodGet, "/pipeline/tasks/"},
	{http.MethodPost, "/pipeline/tasks/_search"},
	{http.MethodPost, "/pipeline/task/:id/_start"},
	{http.MethodPost, "/pipeline/task/:id/_stop"},
	{http.MethodGet, "/pipeline/task/:id"},
	{http.MethodDelete, "/pipeline/task/:id"},
	{http.MethodGet, "/config/"},
	{http.MethodPut, "/config/"},
	{http.MethodGet, "/config/runtime"},
	{http.MethodGet, "/setting/logger"},
	{http.MethodPost, "/setting/logger"},
}

func newProtectedAPIRouter(handle httprouter.Handle) *protectedAPIRouter {
	router := &protectedAPIRouter{router: httprouter.New(nil)}
	registerProtectedAPIRoutes(router.router, handle)
	return router
}

func newReverseAPIRouter(agentAPI AgentAPI) *protectedAPIRouter {
	router := &protectedAPIRouter{router: httprouter.New(nil)}
	router.Handle(http.MethodGet, "/agent/_info", agentAPI.getAgentInfo)
	router.Handle(http.MethodGet, "/elasticsearch/node/_discovery", agentAPI.getESNodes)
	router.Handle(http.MethodPost, "/elasticsearch/node/_info", agentAPI.getESNodeInfo)
	router.Handle(http.MethodPost, "/elasticsearch/logs/_list", agentAPI.getElasticLogFiles)
	router.Handle(http.MethodPost, "/elasticsearch/logs/_read", agentAPI.readElasticLogFile)
	registerProtectedAPIRoutes(router.router, noopProtectedAPIRouteHandle)
	return router
}

func (r *protectedAPIRouter) Handle(method, path string, handle httprouter.Handle) {
	if r == nil || r.router == nil {
		return
	}
	r.router.Handle(method, path, handle)
}

func (r *protectedAPIRouter) Match(method, rawPath string) bool {
	if r == nil || r.router == nil {
		return false
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawPath))
	if err != nil || parsed.Path == "" {
		return false
	}
	handle, _, _ := r.router.Lookup(strings.ToUpper(method), parsed.Path)
	return handle != nil
}

func (r *protectedAPIRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) bool {
	if r == nil || r.router == nil || req == nil || req.URL == nil {
		return false
	}
	handle, _, _ := r.router.Lookup(strings.ToUpper(req.Method), req.URL.Path)
	if handle == nil {
		return false
	}
	r.router.ServeHTTP(w, req)
	return true
}

func registerProtectedAPIRoutes(router *httprouter.Router, handle httprouter.Handle) {
	for _, route := range protectedAPIRoutes {
		router.Handle(route.method, route.path, handle)
	}
}
