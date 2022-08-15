package api

import (
	"fmt"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
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
					cluster.Task = &model.Task{
						ClusterMetric: model.ClusterMetricTask{
							Owner:      true,
							TaskNodeID: id,
						},
						NodeMetric: &model.NodeMetricTask{
							Owner:      true,
							ExtraNodes: nil,
						},
					}
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
					//node.TaskOwner = false
					cluster.Task = &model.Task{
						ClusterMetric: model.ClusterMetricTask{
							Owner:      false,
							TaskNodeID: "",
						},
						NodeMetric: &model.NodeMetricTask{
							Owner:      true,
							ExtraNodes: nil,
						},
					}
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

func (handler *AgentAPI) DeleteAgent() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		agentId := params.MustGetParameter("agent_id")
		log.Printf("request to delete agent: %s", agentId)
		if agentId == config.GetHostInfo().AgentID {
			config.DeleteHostInfo()
			handler.WriteJSON(writer, util.MapStr{
				"result": "deleted",
			}, http.StatusOK)
		} else {
			handler.WriteJSON(writer, util.MapStr{
				"result": "fail",
				"error":  "bad request",
			}, http.StatusInternalServerError)
		}
	}
}
