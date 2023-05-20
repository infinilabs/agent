/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	log "github.com/cihub/seelog"
	"infini.sh/agent/plugin/manage/instance"
	httprouter "infini.sh/framework/core/api/router"
	. "infini.sh/framework/core/host"
	"infini.sh/framework/core/util"
	"net/http"
	"time"

)

func (handler *AgentAPI) getHostBasicInfo(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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
