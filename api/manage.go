package api

import (
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage"
	"infini.sh/agent/plugin/manage/instance"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/host"
	"infini.sh/framework/core/util"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
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
		}, http.StatusOK)
	}
}

func (handler *AgentAPI) HostBasicInfo() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		agentId := params.MustGetParameter("agent_id")
		if !strings.EqualFold(agentId, config.GetInstanceInfo().AgentID) {
			errorResponse("fail", "error params", handler, writer)
			return
		}
		hostInfo := host.HostInfo{
			OSInfo:  host.OS{},
			CPUInfo: host.CPU{},
		}
		var err error
		var bootTime uint64
		hostInfo.Name, bootTime, hostInfo.OSInfo.Platform, _, hostInfo.OSInfo.KernelVersion, hostInfo.OSInfo.KernelArch, err = instance.GetHostInfo()
		if err != nil {
			errorResponse("fail", fmt.Sprintf("get host basic info failed, %v", err), handler, writer)
			return
		}
		hostInfo.MemorySize, _, _, _, err = instance.GetMemoryInfo()
		if err != nil {
			errorResponse("fail", fmt.Sprintf("get host basic info failed, %v", err), handler, writer)
			return
		}
		hostInfo.DiskSize, _, _, _, err = instance.GetDiskInfo()
		if err != nil {
			errorResponse("fail", fmt.Sprintf("get host basic info failed, %v", err), handler, writer)
			return
		}
		hostInfo.CPUInfo.PhysicalCPU, hostInfo.CPUInfo.LogicalCPU, _, hostInfo.CPUInfo.Model, err = instance.GetCPUInfo()
		if err != nil {
			errorResponse("fail", fmt.Sprintf("get host info failed, %v", err), handler, writer)
			return
		}
		hostInfo.UpTime = time.Unix(int64(bootTime), 0)
		content, err := json.Marshal(hostInfo)
		if err != nil {
			errorResponse("fail", fmt.Sprintf("get host info failed, %v", err), handler, writer)
			return
		}
		handler.WriteJSON(writer, util.MapStr{
			"result": string(content),
		}, http.StatusOK)
	}
}

func (handler *AgentAPI) HostUsageInfo() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {

		//var err error
		//var bootTime uint64
		//hostInfo.Name, bootTime, hostInfo.OSInfo.Platform, _, hostInfo.OSInfo.KernelVersion, hostInfo.OSInfo.KernelArch, err = instance.GetHostInfo()
		//if err != nil {
		//	errorResponse("fail", fmt.Sprintf("get host info failed, %v", err), handler, writer)
		//	return
		//}
		//hostInfo.UpTime = time.Unix(int64(bootTime), 0)
		//content, err := json.Marshal(hostInfo)
		//if err != nil {
		//	errorResponse("fail", fmt.Sprintf("get host info failed, %v", err), handler, writer)
		//	return
		//}
		//handler.WriteJSON(writer, util.MapStr{
		//	"result": string(content),
		//}, http.StatusOK)

		handler.WriteJSON(writer, util.MapStr{
			"result": "updated",
		}, http.StatusOK)
	}
}

func errorResponse(errMsg string, description string, handler *AgentAPI, writer http.ResponseWriter) {
	handler.WriteJSON(writer, util.MapStr{
		"result": errMsg,
		"error":  description,
	}, http.StatusInternalServerError)
}
