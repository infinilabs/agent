package api

import (
	"fmt"
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
)

func (handler *AgentAPI) EnableTask() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.MustGetParameter("node_id")
		fmt.Println(id)
		//从kv中获取host信息，找到对应node，开始抓取数据
	}
}

func (handler *AgentAPI) DisableTask() httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		id := params.MustGetParameter("node_id")
		fmt.Println(id)
		//关闭任务
	}
}
