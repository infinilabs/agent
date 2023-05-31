/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"net/http"
	"path"
)

func (handler *AgentAPI) saveDynamicConfig(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	reqBody := DynamicConfigReq{}
	err := handler.DecodeJSON(req, &reqBody)
	if err != nil {
		log.Error(err)
		handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cfgDir := global.Env().GetConfigDir()
	for name, content := range reqBody.Configs {
		file := path.Join(cfgDir, fmt.Sprintf("%s.yml", name))
		_, err = util.FilePutContent(file, content)
		if err != nil {
			log.Error(err)
			handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	handler.WriteAckOKJSON(w)
}
