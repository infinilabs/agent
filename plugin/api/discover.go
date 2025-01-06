/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	"net/http"

	log "github.com/cihub/seelog"
	"infini.sh/agent/lib/process"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

// local exists nodes, find new nodes in runtime
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
		return nil, fmt.Errorf("failed to load config file: %v, path: %s", err, configFile)
	}
	childCfg, err := config.LoadPath(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config dir: %v, path: %s", err, configDir)
	}
	err = parentCfg.Merge(childCfg)
	return parentCfg, nil
}

func (handler *AgentAPI) getESNodeInfo(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	esConfig := elastic.ElasticsearchConfig{}
	err := handler.DecodeJSON(req, &esConfig)
	if err != nil {
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if global.Env().IsDebug {
		log.Debug("esConfig: ", util.MustToJSON(esConfig))
	}

	localNodeInfo, err := process.DiscoverESNodeFromEndpoint(esConfig.GetAnyEndpoint(), esConfig.BasicAuth)
	if err != nil {
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	handler.WriteJSON(w, localNodeInfo, http.StatusOK)
}
