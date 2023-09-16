/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	es_common "infini.sh/framework/modules/elastic/common"
	"net/http"
)


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
