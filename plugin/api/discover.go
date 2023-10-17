/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/lib/process"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"net/http"
)

//local exists nodes, find new nodes in runtime
func (handler *AgentAPI) getESNodes(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	var configs []elastic.ElasticsearchConfig
	appCfg, err := getAppConfig()
	if err != nil {
		_, err = env.ParseConfig("elasticsearch", &configs)
	} else {
		_, err = env.ParseConfigSection(appCfg, "elasticsearch", &configs)
	}
	if err != nil {
		log.Debug(err)
	}
	for i := range configs {
		if configs[i].ID == "" {
			configs[i].ID = configs[i].Name
		}
	}

	//found local nodes
	result, err := process.DiscoverESNode(configs)
	if err != nil {
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	handler.WriteJSON(w, result, http.StatusOK)
}

func getAppConfig() (*config.Config, error) {
	configFile := global.Env().GetConfigFile()
	configDir := global.Env().GetConfigDir()
	parentCfg, err := config.LoadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: ", err, ", path: ", configFile)
	}
	childCfg, err := config.LoadPath(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config dir: ", err, ", path: ", configDir)
	}
	err = parentCfg.Merge(childCfg)
	return parentCfg, nil
}

//func (handler *AgentAPI) authESNode(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
//	authReq := elastic.ElasticsearchConfig{}
//	err := handler.DecodeJSON(req, &authReq)
//	if err != nil {
//		log.Error(err)
//		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	//clusterInfo, err := util.GetClusterVersion(authReq.Endpoint, authReq.BasicAuth)
//	//if err != nil {
//	//	log.Error(err)
//	//	status, _ := jsonparser.GetInt([]byte(err.Error()), "status")
//	//	if status == 0 {
//	//		status = http.StatusInternalServerError
//	//	}
//	//	handler.WriteError(w, err.Error(), int(status))
//	//	return
//	//}
//	authReq.Enabled = true
//	nodeInfo, err := process.DiscoverESNodeFromEndpoint(authReq)
//	//nodeID, nodeInfo, err := util.GetLocalNodeInfo(authReq.Endpoint, authReq.BasicAuth)
//	if err != nil {
//		log.Error(err)
//		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	handler.WriteJSON(w, nodeInfo, http.StatusOK)
//}
