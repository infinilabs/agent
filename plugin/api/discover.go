/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	"infini.sh/framework/core/agent"
	"net/http"
	"sort"

	log "github.com/cihub/seelog"

	"infini.sh/agent/lib/process"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
)

func (handler *AgentAPI) getESNodes(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	var configs []elastic.ElasticsearchConfig
	env.ParseConfig("elasticsearch", &configs)
	for i := range configs {
		if configs[i].ID == "" {
			configs[i].ID = configs[i].Name
		}
	}
	nodesM, err := process.DiscoverESNode(configs)
	if err != nil {
		log.Error(err)
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nodes := []agent.ESNodeInfo{}
	for _, node := range nodesM {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		v1 := fmt.Sprintf("%s%s", nodes[i].ClusterUuid, nodes[i].NodeName)
		v2 := fmt.Sprintf("%s%s", nodes[j].ClusterUuid, nodes[j].NodeName)
		return v1 > v2
	})
	handler.WriteJSON(w, nodes, http.StatusOK)
}

func (handler *AgentAPI) authESNode(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	authReq := elastic.ElasticsearchConfig{}
	err := handler.DecodeJSON(req, &authReq)
	if err != nil {
		log.Error(err)
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//clusterInfo, err := util.GetClusterVersion(authReq.Endpoint, authReq.BasicAuth)
	//if err != nil {
	//	log.Error(err)
	//	status, _ := jsonparser.GetInt([]byte(err.Error()), "status")
	//	if status == 0 {
	//		status = http.StatusInternalServerError
	//	}
	//	handler.WriteError(w, err.Error(), int(status))
	//	return
	//}
	authReq.Enabled = true
	nodeInfo, err := process.DiscoverESNodeFromEndpoint(authReq)
	//nodeID, nodeInfo, err := util.GetLocalNodeInfo(authReq.Endpoint, authReq.BasicAuth)
	if err != nil {
		log.Error(err)
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	handler.WriteJSON(w, nodeInfo, http.StatusOK)
}
