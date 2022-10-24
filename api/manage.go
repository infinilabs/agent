package api

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/lib/reader/harvester"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage"
	"infini.sh/agent/plugin/manage/instance"
	httprouter "infini.sh/framework/core/api/router"
	. "infini.sh/framework/core/host"
	"infini.sh/framework/core/util"
	"io"
	"io/ioutil"
	"net/http"
	"src/github.com/shirou/gopsutil/process"
	"strings"
	"time"
)

func (handler *AgentAPI) EnableTask() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.MustGetParameter("node_id")
		if id == "" {
			errorResponse("fail", fmt.Sprintf("nodeID:%s, could not be empty", id), handler, writer)
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
					log.Infof("receive task, cluster: %s(%s), node: %s\n", cluster.Name, cluster.ID, id)
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

func (handler *AgentAPI) ExtraTask() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			handler.WriteJSON(writer, util.MapStr{
				"success": false,
				"error":   "parse request error",
			}, http.StatusInternalServerError)
			return
		}
		var extra map[string][]string
		err = json.Unmarshal(body, &extra)
		if err != nil {
			handler.WriteJSON(writer, util.MapStr{
				"success": false,
				"error":   "parse request error",
			}, http.StatusInternalServerError)
			return
		}
		instanceInfo := config.GetInstanceInfo()
		for clusterId, nodeIds := range extra {
			for _, cluster := range instanceInfo.Clusters {
				if cluster.ID == clusterId {
					cluster.Task.NodeMetric = &model.NodeMetricTask{
						Owner:      true,
						ExtraNodes: nodeIds,
					}
					log.Infof("received extra task, cluster: %s(id: %s), node:%s", cluster.Name, cluster.ID, nodeIds)
				}
			}
		}
		config.SetInstanceInfo(instanceInfo)
		handler.WriteJSON(writer, util.MapStr{
			"success": true,
		}, http.StatusOK)
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
		ok, err := manage.RegisterCallback(&registerResp)
		if err != nil {
			log.Debugf("api.RegisterCallBack: %v", err)
			errorResponse("fail", "parse request body failed.", handler, writer)
			return
		}
		if !ok {
			errorResponse("fail", "update agent status failed", handler, writer)
			return
		}
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

//
// HostDiscovered
//  @Description: when host discovered in console. this api will be call.
//
func (handler *AgentAPI) HostDiscovered() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {

		content,err := ioutil.ReadAll(request.Body)
		if err != nil {
			errorResponseNew("bad request params",handler,writer)
			return
		}
		var bodyMap map[string]string
		err = json.Unmarshal(content,&bodyMap)
		if err != nil {
			errorResponseNew("bad request params",handler,writer)
		}
		hostID := bodyMap["host_id"]
		if hostID == "" {
			errorResponseNew("bad request params",handler,writer)
		}
		instanceInfo := config.GetInstanceInfo()
		instanceInfo.HostID = hostID
		config.SetInstanceInfo(instanceInfo)
		handler.WriteJSON(writer, util.MapStr{
			"success": true,
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
		processes, err := process.Processes()
		if err != nil {
			log.Error(err)
			errorResponseNew("parse process info failed",handler,writer)
		}
		var pidInfos []util.MapStr
		for _, cluster := range instanceInfo.Clusters {
			for _, node := range cluster.Nodes {
				status, createTime, err := getPIDStatusAndCreateTime(processes, node.PID, node.Name)
				if err != nil {
					log.Error(err)
					continue
				}
				pidInfos = append(pidInfos,
					util.MapStr{
						"pid":          node.PID,
						"pid_status":   status,
						"cluster_name": cluster.Name,
						"cluster_uuid": cluster.UUID,
						"cluster_id":   cluster.ID,
						"node_id":      node.ID,
						"node_name":    node.Name,
						"uptime_in_ms": time.Now().UnixMilli() - createTime,
					})
			}
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

func (handler *AgentAPI) LogsFileList() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		content,err := ioutil.ReadAll(request.Body)
		if err != nil {
			errorResponseNew("bad request params",handler,writer)
			return
		}
		var bodyMap map[string]string
		err = json.Unmarshal(content,&bodyMap)
		if err != nil {
			errorResponseNew("bad request params",handler,writer)
		}
		nodeId := bodyMap["node_id"]
		suffix := ""
		//suffix := params.MustGetParameter("suffix")
		//if !strings.EqualFold(suffix,".log") && !strings.EqualFold(suffix,".json") {
		//	suffix = ".json"
		//}
		if nodeId == "" {
			errorResponseNew("error params: nodeId", handler, writer)
			return
		}
		instanceInfo := config.GetInstanceInfo()
		if instanceInfo == nil {
			errorResponseNew("no instance info found", handler, writer)
			return
		}
		node := instanceInfo.FindNodeById(nodeId)
		if node == nil {
			errorResponseNew("can not find node info", handler, writer)
			return
		}
		fileInfos, err := ioutil.ReadDir(node.LogPath)
		if err != nil {
			log.Error(err)
			errorResponseNew("can not read log files", handler, writer)
			return
		}
		var files []util.MapStr
		for _, info := range fileInfos {
			if suffix == "" {
				if strings.HasSuffix(info.Name(), ".log") || strings.HasSuffix(info.Name(), ".json") {
					files = append(files,util.MapStr{
						"name": info.Name(),
						"size_in_bytes": info.Size(),
						"modify_time": info.ModTime(),
					})
				}
			} else {
				if strings.HasSuffix(info.Name(), suffix) {
					files = append(files,util.MapStr{
						"name": info.Name(),
						"size_in_bytes": info.Size(),
						"modify_time": info.ModTime(),
					})
				}
			}
		}
		if len(files) == 0 {
			errorResponseNew("can not find log files", handler, writer)
			return
		}

		handler.WriteJSON(writer, util.MapStr{
			"result":  files,
			"success": true,
		}, http.StatusOK)
	}
}

func (handler *AgentAPI) ReadLogFile() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		byteBody, err := ioutil.ReadAll(request.Body)
		if err != nil {
			errorResponseNew("error params", handler, writer)
			return
		}
		var requestModel model.ReadLogRequest
		err = json.Unmarshal(byteBody, &requestModel)
		if err != nil {
			log.Error(err)
			errorResponseNew("error params", handler, writer)
			return
		}
		err = requestModel.ValidateParams()
		if err != nil {
			log.Error(err)
			errorResponseNew(err.Error(), handler, writer)
			return
		}

		instanceInfo := config.GetInstanceInfo()
		if instanceInfo == nil {
			errorResponseNew("no instance info found", handler, writer)
			return
		}
		node := instanceInfo.FindNodeById(requestModel.NodeId)
		if node == nil {
			errorResponseNew("can not find node info", handler, writer)
			return
		}
		filePath := node.LogPath
		if !strings.HasSuffix(filePath, "/") {
			filePath = fmt.Sprintf("%s/", node.LogPath)
		}
		filePath = fmt.Sprintf("%s%s", filePath, requestModel.FileName)
		h, err := harvester.NewHarvester(filePath, requestModel.Offset)
		if err != nil {
			log.Error(err)
			errorResponseNew("harvester: can not read log files", handler, writer)
			return
		}
		r, err := h.NewPlainTextRead()
		if err != nil {
			log.Error(err)
			errorResponseNew("harvester: can not read log files", handler, writer)
			return
		}
		var msgs []util.MapStr
		isEOF := false
		for i := 0; i < requestModel.Lines; i++ {
			msg, err := r.Next()
			if err != nil {
				if err == io.EOF {
					isEOF = true
					break
				} else {
					log.Error(err)
					errorResponseNew("harvester: read log file error", handler, writer)
					return
				}
			}
			msgs = append(msgs, util.MapStr{
				"content": string(msg.Content),
				"bytes": msg.Bytes,
				"offset": msg.Offset,
				"line_number": coverLineNumbers(msg.LineNumbers),
			})
		}
		if h.Close() != nil {
			log.Error(err)
			errorResponseNew("harvester: close reader error", handler, writer)
			return
		}
		handler.WriteJSON(writer, util.MapStr{
			"result":  msgs,
			"success": true,
			"EOF": isEOF,
		}, http.StatusOK)
	}
}

func coverLineNumbers(numbers []int) interface{}{
	if len(numbers) == 1 {
		return numbers[0]
	} else {
		return numbers
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
