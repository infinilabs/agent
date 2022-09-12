/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package api

import (
	"infini.sh/agent/config"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
	"net/http"
	"os"
	"src/github.com/struCoder/pidusage"
	"time"
)

func (handler *AgentAPI) LocalStats() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {

		sysInfo, err := pidusage.GetStat(os.Getpid())
		if err != nil {
			handler.Error(writer, err)
			return
		}
		upTime := time.Now().Unix() - config.GetInstanceInfo().BootTime
		handler.WriteJSON(writer, util.MapStr{
			"cpu":             sysInfo.CPU,
			"memory_in_bytes": sysInfo.Memory,
			"memory":          util.ByteSize(uint64(sysInfo.Memory)),
			"uptime_in_second": upTime,
		}, 200)
	}
}
