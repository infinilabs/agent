package manage

import (
	"encoding/json"
	"fmt"
	"infini.sh/agent/api"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage/host"
	"infini.sh/framework/core/util"
	"log"
)

/*
初始化agent。注册agent，上报主机、集群、节点信息给console
*/
func Init() {
	clientInfos := host.RegisterHost()
	if clientInfos != nil {
		UploadNodeInfos(clientInfos)
	}
}

func UploadNodeInfos(clientInfos []model.ESCluster) {
	nodeInfos := GetESNodeInfos(clientInfos)
	if nodeInfos == nil {
		log.Panic("getESNodeInfos failed. all passwords are wrong?? es crashed??")
		return
	}
	url := fmt.Sprintf("%s%s", api.UrlRegisterHost, api.UrlUploadNodeInfo)
	body, _ := json.Marshal(nodeInfos)
	var req = util.NewGetRequest(url, body)
	result, err := util.ExecuteRequest(req)
	if err != nil {
		log.Printf("uploadNodeInfos failed: \n %v", err)
		return
	}
	//TODO 解析返回结果
	fmt.Println(result.Body)
}

func GetESNodeInfos(clientInfos []model.ESCluster) []model.ESCluster {
	var clusters []model.ESCluster
	for _, cluster := range clientInfos {
		if cluster.Port == 0 {
			continue
		}
		url := fmt.Sprintf("http://localhost:%d/_nodes/_local", cluster.Port)
		var req = util.NewGetRequest(url, nil)
		if cluster.BasicAuth.Username != "" && cluster.BasicAuth.Password != "" {
			req.SetBasicAuth(cluster.BasicAuth.Username, cluster.BasicAuth.Password)
		}
		result, err := util.ExecuteRequest(req)
		if err != nil {
			continue //账号密码错误
		}
		nodeId := host.ParseNodeID(string(result.Body))
		if nodeId == "" {
			continue
		}
		cluster.Nodes = append(cluster.Nodes, nodeId)
		clusters = append(clusters, cluster)
	}
	return clusters
}
