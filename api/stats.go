/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package api

import (
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"net/http"
	"os"
	"github.com/struCoder/pidusage"

)

type AgentAPI struct {
	api.Handler
}

func (handler AgentAPI) Init() {
	api.HandleAPIMethod(fasthttp.MethodGet,"/stats/_local", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		sysInfo, err := pidusage.GetStat(os.Getpid())
		if err!=nil{
			handler.Error(w,err)
			return
		}

		handler.WriteJSON(w,util.MapStr{
			"cpu":sysInfo.CPU,
			"memory_in_bytes":sysInfo.Memory,
			"memory":util.ByteSize(uint64(sysInfo.Memory)),
		},200)
	})
}
