/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
)

// loginToken is the in-memory API token generated at startup.
//
// Users can login via the token, which grants administrator privileges
// after successful login.
var loginToken string

func initAuthToken() {
	loginToken = util.GetUUID()
	log.Infof("[agent] login token: %s", loginToken)
}
