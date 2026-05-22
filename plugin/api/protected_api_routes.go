package api

import (
	"net/http"

	httprouter "infini.sh/framework/core/api/router"
)

type protectedAPIRoute struct {
	method string
	path   string
}

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
}

func registerProtectedAPIRoutes(router *httprouter.Router, handle httprouter.Handle) {
	for _, route := range protectedAPIRoutes {
		router.Handle(route.method, route.path, handle)
	}
}
