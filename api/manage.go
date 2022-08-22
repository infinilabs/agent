package api

import (
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
	"io/ioutil"
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
		log.Debugf("api.EnableTask, nodeID: %s\n", id)
		host := config.GetInstanceInfo()
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
					config.SetInstanceInfo(host)
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
		log.Debugf("api.DisableTask, nodeID: %s\n", id)
		host := config.GetInstanceInfo()
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
					config.SetInstanceInfo(host)
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
		log.Debugf("request to delete agent: %s", agentId)
		if agentId == config.GetInstanceInfo().AgentID {
			config.DeleteInstanceInfo()
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

func (handler *AgentAPI) RegisterCallBack() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		agentId := params.MustGetParameter("agent_id")
		if agentId == "" || !strings.EqualFold(agentId, config.GetInstanceInfo().AgentID) {
			errorResponse("fail", "bad request params", handler, writer)
			return
		}
		registerResp := model.RegisterResponse{}
		registerResp.AgentId = agentId
		contentBytes, err := ioutil.ReadAll(request.Body)
		log.Debugf("api.RegisterCallBack, agentId: %s, request body: %s\n", agentId, string(contentBytes))
		if err != nil {
			errorResponse("fail", fmt.Sprintf("read request body failed, %v", err), handler, writer)
			return
		}
		err = json.Unmarshal(contentBytes, &registerResp.Clusters)
		if err != nil {
			errorResponse("fail", fmt.Sprintf("parse request body failed, %v", err), handler, writer)
			return
		}
		ok, err := manage.RegisterCallback(&registerResp)
		if err != nil {
			errorResponse("fail", fmt.Sprintf("%v", err), handler, writer)
			return
		}
		if !ok {
			errorResponse("fail", "update agent status failed", handler, writer)
			return
		}
		handler.WriteJSON(writer, util.MapStr{
			"result": "updated",
		}, http.StatusInternalServerError)
	}
}

func errorResponse(errMsg string, description string, handler *AgentAPI, writer http.ResponseWriter) {
	handler.WriteJSON(writer, util.MapStr{
		"result": errMsg,
		"error":  description,
	}, http.StatusInternalServerError)
}
