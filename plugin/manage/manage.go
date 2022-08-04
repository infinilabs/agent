package manage

import (
	"encoding/json"
	"fmt"
	"infini.sh/agent/api"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage/host"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"log"
)

/*
初始化agent。注册agent，上报主机、集群、节点信息给console
*/
func Init() {
	ok, err := kv.ExistsKey(config.KVAgentBucket, []byte(config.KVAgentID))
	if err != nil {
		fmt.Println(err)
	}
	if ok {
		fmt.Println(kv.GetValue(config.KVAgentBucket, []byte(config.KVAgentID)))
	} else {
		kv.AddValue(config.KVAgentBucket, []byte(config.KVAgentID), []byte("clusteridxxxxxxx001"))
	}
	//clientInfos, err := host.RegisterHost()
	//if err != nil {
	//	panic(err)
	//}
	//if clientInfos != nil {
	//	UploadNodeInfos(clientInfos.Clusters)
	//}
	//hbClinet := hearbeat.NewClient(time.Second * 10)
	//go hbClinet.Heartbeat(func() (string, error) {
	//	return "", nil
	//}, func(content string) (bool, error) {
	//	return true, nil
	//})
}

func UploadNodeInfos(clientInfos []*model.Cluster) {
	nodeInfos := GetESNodeInfos(clientInfos)
	if nodeInfos == nil {
		log.Panic("getESNodeInfos failed. all passwords are wrong?? es crashed??")
		return
	}
	url := fmt.Sprintf("%s%s", api.UrlConsole, api.UrlUploadNodeInfo)
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

func GetESNodeInfos(clusterInfos []*model.Cluster) []*model.Cluster {
	var clusters []*model.Cluster
	for _, cluster := range clusterInfos {
		for _, node := range cluster.Nodes {
			if node.HttpPort == 0 {
				continue
			}
			url := fmt.Sprintf("http://localhost:%d/_nodes/_local", node.HttpPort)
			var req = util.NewGetRequest(url, nil)
			if cluster.UserName != "" && cluster.Password != "" {
				req.SetBasicAuth(cluster.UserName, cluster.Password)
			}
			result, err := util.ExecuteRequest(req)
			if err != nil {
				continue //账号密码错误
			}
			nodeId := host.ParseNodeID(string(result.Body))
			if nodeId == "" {
				continue
			}
			node.ID = nodeId
			clusters = append(clusters, cluster)
		}
	}
	return clusters
}
