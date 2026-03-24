/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
)

// authToken is the in-memory API token generated at startup.
// All API requests must present this value via the X-API-TOKEN header.
var authToken string

func initAuthToken() {
	authToken = util.GetUUID()
	log.Infof("[agent] api token: %s", authToken)
}
