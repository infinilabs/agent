package manage

import (
	"encoding/json"
	"fmt"
	"infini.sh/agent/api"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage/hearbeat"
	"infini.sh/agent/plugin/manage/host"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"log"
	"strings"
	"time"
)

/*
初始化agent。注册agent，上报主机、集群、节点信息给console
*/
var registerSuccess = make(chan bool)

func Init() {

	//if host.IsRegistered() {
	//	go HeartBeat()
	//} else {
	//	Register()
	//	if <-registerSuccess {
	//		go HeartBeat()
	//	}
	//}
}

func Register() {
	defer close(registerSuccess)
	hostInfo, err := host.RegisterHost()
	if err != nil {
		log.Printf("register host failed:\n%v\n", err)
		registerSuccess <- false
		return
	}
	if hostInfo != nil {
		fmt.Printf("注册成功: \n%v\n", hostInfo)
		tmpHostInfo := UploadNodeInfos(hostInfo)
		if tmpHostInfo != nil {
			host.SaveAgentHostInfo(tmpHostInfo) //到这一步，才算真正的完成agent注册
			registerSuccess <- true
			return
		}
	}
	registerSuccess <- false
}

func HeartBeat() {
	host := config.GetHostInfoFromKV()
	if host == nil {
		return
	}
	hbClinet := hearbeat.NewClient(time.Second*10, host.AgentID)
	go hbClinet.Heartbeat(func() (string, error) {
		hst, err := kv.GetValue(config.KVAgentBucket, []byte(config.KVHostInfo))
		if err != nil {
			return "", err
		}
		var host model.Host
		err = json.Unmarshal(hst, &host)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("{'instance_id':%s}", host.AgentID), nil
	}, func(content string) (bool, error) {
		//TODO 解析返回结果
		return true, nil
	})
}

func UploadNodeInfos(host *model.Host) *model.Host {
	nodeInfos := GetESNodeInfos(host.Clusters)
	if nodeInfos == nil {
		log.Panic("getESNodeInfos failed. all passwords are wrong?? es crashed??")
		return nil
	}
	var reqPath string
	strings.ReplaceAll(api.UrlUploadNodeInfo, ":instance_id", host.AgentID)
	url := fmt.Sprintf("%s/%s", config.UrlConsole(), reqPath)
	var esClusters []*agent.ESCluster
	for _, info := range nodeInfos {
		esClusters = append(esClusters, info.ConvertToESCluster())
	}
	body, _ := json.Marshal(esClusters)
	var req = util.NewPutRequest(url, body)
	result, err := util.ExecuteRequest(req)
	if err != nil {
		log.Printf("uploadNodeInfos failed: \n %v", err)
		return nil
	}
	//TODO 解析返回结果
	fmt.Println("上传Node信息，返回: ")
	fmt.Println(result.Body)

	return host
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
