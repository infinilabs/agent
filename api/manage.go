package api

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/shirou/gopsutil/v3/process"
	"infini.sh/agent/config"
	process2 "infini.sh/agent/lib/process"
	//"infini.sh/agent/lib/reader/harvester"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage/instance"
	httprouter "infini.sh/framework/core/api/router"
	. "infini.sh/framework/core/host"
	"infini.sh/framework/core/util"
	//"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

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
			log.Debugf("api.RegisterCallBack: %v", err)
			errorResponse("fail", "read request body failed", handler, writer)
			return
		}
		err = json.Unmarshal(contentBytes, &registerResp.Clusters)
		if err != nil {
			log.Debugf("api.RegisterCallBack: %v", err)
			errorResponse("fail", "parse request body failed", handler, writer)
			return
		}
		//ok, err := manage.RegisterCallback(&registerResp)
		//if err != nil {
		//	log.Debugf("api.RegisterCallBack: %v", err)
		//	errorResponse("fail", "parse request body failed.", handler, writer)
		//	return
		//}
		//if !ok {
		//	errorResponse("fail", "update agent status failed", handler, writer)
		//	return
		//}
		log.Infof("reviewed and approved, register successfully")
		handler.WriteJSON(writer, util.MapStr{
			"result": "updated",
		}, http.StatusOK)
	}
}

func (handler *AgentAPI) HostBasicInfo() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		if !validateAgentId(params) {
			errorResponse("fail", "error params", handler, writer)
			return
		}
		hostInfo := HostInfo{
			OSInfo:  OS{},
			CPUInfo: CPU{},
		}
		var err error
		var bootTime uint64
		hostInfo.Name, bootTime, hostInfo.OSInfo.Platform, hostInfo.OSInfo.PlatformVersion, hostInfo.OSInfo.KernelVersion, hostInfo.OSInfo.KernelArch, err = instance.GetOSInfo()
		if err != nil {
			log.Error(err)
		}
		hostInfo.MemorySize, _, _, _, err = instance.GetMemoryInfo()
		if err != nil {
			log.Error(err)
		}
		hostInfo.DiskSize, _, _, _, err = instance.GetDiskInfo()
		if err != nil {
			log.Error(err)
		}
		hostInfo.CPUInfo.PhysicalCPU, hostInfo.CPUInfo.LogicalCPU, _, hostInfo.CPUInfo.Model, err = instance.GetCPUInfo()
		if err != nil {
			log.Error(err)
		}
		hostInfo.UpTime = time.Unix(int64(bootTime), 0)
		handler.WriteJSON(writer, util.MapStr{
			"success": true,
			"result":  hostInfo,
		}, http.StatusOK)
	}
}

func (handler *AgentAPI) HostUsageInfo() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		if !validateAgentId(params) {
			errorResponse("fail", "error params", handler, writer)
			return
		}
		cate := params.ByName("category")
		if cate == "" {
			cate = "all"
		}

		category := UsageCategory(cate)
		usage := &Usage{}
		var err error
		switch category {
		case AllUsage:
			usage, err = instance.GetAllUsageInfo()
		case MemoryUsage:
			usage.MemoryUsage, usage.SwapMemoryUsage, err = instance.GetMemoryUsage()
		case DiskUsage:
			usage.DiskUsage, err = instance.GetDiskUsage()
		case CPUUsage:
			usage.CPUPercent = instance.GetCPUUsageInfo()
		case DiskIOUsage:
			usage.DiskIOUsage, err = instance.GetDiskIOUsageInfo()
		case NetIOUsage:
			usage.NetIOUsage, err = instance.GetNetIOUsage()
		case ESProcessInfo:
			ret, err := instance.GetNodeInfoFromProcess()
			if err != nil {
				errorResponse("fail", fmt.Sprintf("api.ESProcessInfo: failed, %v", err), handler, writer)
				return
			}
			retByte, err := json.Marshal(ret)
			if err != nil {
				errorResponse("fail", fmt.Sprintf("api.ESProcessInfo: failed, %v", err), handler, writer)
				return
			}
			usage.ESProcessInfo = string(retByte)
		}
		if err != nil {
			errorResponse("fail", fmt.Sprintf("api.HostUsageInfo: failed, %v", err), handler, writer)
			return
		}
		usage.AgentID = config.GetInstanceInfo().AgentID
		handler.WriteJSON(writer, util.MapStr{
			"result": usage,
		}, http.StatusOK)
	}
}

func (handler *AgentAPI) ElasticProcessInfo() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		if !validateAgentId(params) {
			errorResponse("fail", "error params", handler, writer)
			return
		}
		instanceInfo := config.GetInstanceInfo()
		if instanceInfo == nil {
			errorResponseNew("no instance info found",handler,writer)
			return
		}

		nodes, err := process2.DiscoverESNode(nil)
		if err != nil {
			log.Error(err)
			errorResponseNew("parse process info failed",handler,writer)
		}
		var pidInfos []util.MapStr
		for _, node := range nodes {
			pidInfos = append(pidInfos,
				util.MapStr{
					"pid":          node.ProcessInfo.PID,
					"pid_status":   node.ProcessInfo.Status,
					"cluster_name": node.ClusterName,
					"cluster_uuid": node.ClusterUuid,
					"cluster_id":   "",
					"node_id":      node.NodeUUID,
					"node_name":    node.NodeName,
					"uptime_in_ms": time.Now().UnixMilli() - node.ProcessInfo.CreateTime,
				})
		}
		if len(pidInfos) == 0 {
			errorResponseNew("no es process found", handler, writer)
			return
		}
		handler.WriteJSON(writer, util.MapStr{
			"elastic_process": pidInfos,
			"success": true,
		}, http.StatusOK)
	}
}

func getPIDStatusAndCreateTime(processes []*process.Process, pid int32, nodeName string) (status string, createTime int64, err error) {
	for _, process := range processes {
		if process.Pid == pid {
			values, err := process.Status()
			if err != nil {
				status = "N/A"
			} else {
				if len(values) > 0 {
					status = values[0]
				}
			}
			createTime, err = process.CreateTime()
			if err != nil {
				createTime = 0
			}
			err = nil
			return status,createTime,err
		}
	}
	return "", 0, errors.New(fmt.Sprintf("es process info not found,nodeName: %s(pid: %d)\n",nodeName,pid))
}

func validateAgentId(params httprouter.Params) bool {
	agentId := params.MustGetParameter("agent_id")
	if !strings.EqualFold(agentId, config.GetInstanceInfo().AgentID) {
		return false
	}
	return true
}

func errorResponse(errMsg string, description string, handler *AgentAPI, writer http.ResponseWriter) {
	handler.WriteJSON(writer, util.MapStr{
		"result": errMsg,
		"error":  description,
	}, http.StatusInternalServerError)
}

func errorResponseNew(description string, handler *AgentAPI, writer http.ResponseWriter) {
	handler.WriteJSON(writer, util.MapStr{
		"success": false,
		"error":   description,
	}, http.StatusInternalServerError)
}
