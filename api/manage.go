package api

import (
	"fmt"
	"infini.sh/agent/config"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
	"log"
	"net/http"
	"strings"
)

func (handler *AgentAPI) EnableTask() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.MustGetParameter("node_id")
		if id == "" {
			handler.WriteJSON(writer, util.MapStr{
				"result": "fail",
				"error":  fmt.Sprintf("nodeID:%s, could not be empty", id),
			}, http.StatusInternalServerError)
			return
		}
		log.Printf("api.EnableTask, nodeID: %s\n", id)
		host := config.GetHostInfo()
		for _, cluster := range host.Clusters {
			for _, node := range cluster.Nodes {
				if strings.EqualFold(id, node.ID) {
					node.TaskOwner = true
					config.SetHostInfo(host)
					handler.WriteJSON(writer, util.MapStr{
						"result": "success",
					}, http.StatusOK)
					return
				}
			}
		}
		handler.WriteJSON(writer, util.MapStr{
			"result": "fail",
			"error":  fmt.Sprintf("nodeID:%s, could not be found", id),
		}, http.StatusInternalServerError)
	}
}

func (handler *AgentAPI) DisableTask() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.MustGetParameter("node_id")
		if id == "" {
			handler.WriteJSON(writer, util.MapStr{
				"result": "fail",
				"error":  fmt.Sprintf("nodeID:%s, could not be empty", id),
			}, http.StatusInternalServerError)
			return
		}
		log.Printf("api.DisableTask, nodeID: %s\n", id)
		host := config.GetHostInfo()
		for _, cluster := range host.Clusters {
			for _, node := range cluster.Nodes {
				if strings.EqualFold(id, node.ID) {
					node.TaskOwner = false
					config.SetHostInfo(host)
					handler.WriteJSON(writer, util.MapStr{
						"result": "success",
					}, http.StatusOK)
					return
				}
			}
		}
		handler.WriteJSON(writer, util.MapStr{
			"result": "fail",
			"error":  fmt.Sprintf("nodeID:%s, could not be found", id),
		}, http.StatusInternalServerError)
	}
}

func (handler AgentAPI) DeleteAgent() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {

		handler.WriteJSON(writer, util.MapStr{
			"result": "fail",
			"error":  "unknown",
		}, http.StatusInternalServerError)
	}
}
