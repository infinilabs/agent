/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage/instance"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	es_common "infini.sh/framework/modules/elastic/common"
	"net/http"
)

func (handler *AgentAPI) getAgentInfo(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	agentInfo, err := getAgentInfo()
	if err != nil {
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	handler.WriteJSON(w, agentInfo, http.StatusOK)
}

func getAgentInfo() (*model.AgentInfo, error){
	ai := model.AgentInfo{
		ID: global.Env().SystemConfig.NodeConfig.ID,
		Name: global.Env().SystemConfig.NodeConfig.Name,
		Version: util.MapStr{
			"number":       global.Env().GetVersion(),
			"build_date":   global.Env().GetBuildDate(),
			"build_hash":   global.Env().GetLastCommitHash(),
			"build_number": global.Env().GetBuildNumber(),
			"eol_date":     global.Env().GetEOLDate(),
		},
	}
	ai.IPs = util.GetLocalIPs()
	_, majorIp, _, err := util.GetPublishNetworkDeviceInfo(config.EnvConfig.MajorIpPattern)
	if err != nil {
		log.Errorf("get publish network: %v", err)
	}
	ai.MajorIP = majorIp
	hostInfo, err := instance.CollectHostInfo()
	if err != nil {
		log.Errorf("collect host info: %v", err)
	}
	ai.Host = hostInfo
	return &ai, nil
}

func (handler *AgentAPI) registerESNode(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	reqObj := []elastic.ElasticsearchConfig{}
	err := handler.DecodeJSON(req, &reqObj)
	if err != nil {
		log.Errorf("failed to decode request, err: %v", err)
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, cfg := range reqObj {
		cfg.Source = "api"
		// NOTE: needed by GetMetadata, fix it later
		if cfg.ID == "" {
			cfg.ID = cfg.Name
		}
		_, err = es_common.InitElasticInstance(cfg)
		if err != nil {
			 err = log.Errorf("failed to register %s, err: %v", cfg.ID, err)
			handler.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	handler.WriteAckOKJSON(w)
}
