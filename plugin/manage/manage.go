package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"infini.sh/agent/api"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage/hearbeat"
	"infini.sh/agent/plugin/manage/host"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"log"
	"strings"
	"time"
)

/*
初始化agent。注册agent，上报主机、集群、节点信息给console
*/

func Init() {
	//host基础信息，每次启动的时候更新一次
	//集群相关的，定时任务更新
	//UpdateHostBasicInfo()
	if host.IsRegistered() {
		//isAgentLiving()
		HeartBeat()
		checkHostUpdate()
	} else {
		registerChan := make(chan bool)
		go Register(registerChan)
		select {
		case ok := <-registerChan:
			log.Printf("manage.Init: register host %t", ok)
			HeartBeat()
			checkHostUpdate()
		case <-time.After(time.Second * 60):
			log.Printf("manage.Init: register timeout.")
		}
		//close(registerChan)
	}
}

//从console获取agent信息，主要用来判断agent有没有被删掉
func isAgentLiving() {
	//TODO 未完成
	GetHostInfoFromConsole(config.GetHostInfo().AgentID)
}

func GetHostInfoFromConsole(agentID string) (*model.Host, error) {
	reqPath := strings.ReplaceAll(api.UrlUpdateHostInfo, ":instance_id", agentID)
	url := fmt.Sprintf("%s/%s", config.UrlConsole(), reqPath)
	log.Printf("register: %s\n", agentID)
	var req = util.NewGetRequest(url, []byte(""))
	result, err := util.ExecuteRequest(req)
	if err != nil {
		log.Printf("manage.UploadNodeInfos: uploadNodeInfos failed: %v\n", err)
		return nil, err
	}
	log.Printf("manage.UploadNodeInfos: upNodeInfo resp: %s\n", string(result.Body))
	var resp model.UpNodeInfoResponse
	err = json.Unmarshal(result.Body, &resp)
	return nil, err
}

func UpdateHostBasicInfo() {
	config.GetHostInfo().TLS = config.EnvConfig.TLS
	config.GetHostInfo().AgentPort = config.EnvConfig.Port
}

/**
更新主机信息: ip、es集群
*/
func checkHostUpdate() {
	//TODO 先检查任务是否有变化。主要针对agent挂掉之后，新启动的时候。
	hostUpdateTask := task.ScheduleTask{
		Description: "update agent host info",
		Type:        "interval",
		Interval:    "15s",
		Task: func(ctx context.Context) {
			changeType, err := host.IsHostInfoChanged()
			if err != nil {
				log.Printf("manage.checkHostUpdate: update host info failed : %v", err)
				return
			}
			if changeType == config.ChangeOfNothing {
				return
			}
			updateChan := make(chan bool)
			go UpdateHostInfo(updateChan, changeType)

			select {
			case ok := <-updateChan:
				log.Printf("manage.checkHostUpdate: update host info %t", ok)
			case <-time.After(time.Second * 60):
			}
			//close(updateChan)
		},
	}
	task.RegisterScheduleTask(hostUpdateTask)
}

func UpdateHostInfo(isSuccess chan bool, changeType config.ChangeType) {

	////更新集群信息。 注意动态端口的情况
	hostKV := config.GetHostInfo()     //kv当前存储的
	hostPid, err := host.GetHostInfo() //从进程里新解析出来的
	if err != nil {
		log.Printf("get host info failed: %v", err)
		return
	}
	hostPid.AgentID = hostKV.AgentID
	hostPid.AgentPort = hostKV.AgentPort
	for _, cluster := range hostPid.Clusters {
		for _, clusterKv := range hostKV.Clusters {
			if clusterKv.Name == cluster.Name {
				cluster.ID = clusterKv.ID
				cluster.UserName = clusterKv.UserName
				cluster.Password = clusterKv.Password
			}
		}
	}
	UploadNodeInfos(hostPid)
	//fmt.Printf("%v", host)

	//1. 先把新增的集群合并过来
	//2. 新增的节点获取端口号
	//3.

	////基础信息
	//hostNew := &model.Host{}
	//hostNew.TLS = config.EnvConfig.TLS
	//hostNew.AgentPort = config.EnvConfig.Port
	//hostNew.IPs = hostPid.IPs
	//hostNew.AgentID = hostKV.AgentID

	//switch changeType {
	//case config.ChangeOfESConfigFile:
	//	changeESConfigFile()
	//case config.ChangeOfESNodeNumbers:
	//	changeESNodeNumbers()
	//case config.ChangeOfLostInfo:
	//	//待定. 重新注册？ 还是通过别的条件，找console匹配？
	//	log.Printf("hostinfo in kv lost. not handle now")
	//case config.ChangeOfESConnect:
	//	//找console要新的密码，这个之后实现
	//	log.Printf("es password change or port change. not handle now")
	//case config.ChangeOfClusterNumbers:
	//	changeESClusterNumbers()
	//case config.ChangeOfAgentBasic:
	//	changeBasicInfo()
	//}

	////集群节点信息
	//hostNew.Clusters = hostPid.Clusters
	////已有集群的，把新增或者减少的节点处理好，以及现有节点的信息变动。注意动态端口的情况
	////新增集群的，把新的集群信息提交，并把console返回的账号密码更新，并验证更新端口信息。
	//for _, cluster := range hostNew.Clusters {
	//	for _, kvCluster := range hostKV.Clusters {
	//		if cluster.Name == kvCluster.Name {
	//			cluster.UserName = kvCluster.UserName
	//			cluster.Password = kvCluster.Password
	//			cluster.ID = kvCluster.ID
	//			cluster.TLS = kvCluster.TLS
	//			cluster.UUID = kvCluster.UUID
	//			cluster.Task = kvCluster.Task
	//			mergeNodeInfo(cluster, kvCluster)
	//		}
	//	}
	//}
	////GetHostInfoFromConsole(hostNew.AgentID)
	//if UpdateHostInfoToConsole(hostNew) {
	//
	//}
	isSuccess <- true
}

func changeBasicInfo() {
	hostInfo := config.GetHostInfo()
	hostInfo.TLS = config.EnvConfig.TLS
	hostInfo.AgentPort = config.EnvConfig.Port
	UpdateHostInfoToConsole(hostInfo)
}

func changeESConnect() {

}

func changeESNodeNumbers() {
	hostInfo := config.GetHostInfo()
	hostInfoNew, err := host.GetHostInfo()
	if err != nil {
		log.Println(err)
		return
	}
	//先找出新增加/减少的节点
	var oldESConfigPath map[string]string
	for _, cluster := range hostInfo.Clusters {
		for _, node := range cluster.Nodes {
			oldESConfigPath[node.ConfigPath] = node.ConfigPath
		}
	}
	var newESConfigPath map[string]string
	for _, cluster := range hostInfoNew.Clusters {
		for _, node := range cluster.Nodes {
			newESConfigPath[node.ConfigPath] = node.ConfigPath
		}
	}
	var addedESPath map[string]string
	for _, path := range newESConfigPath {
		if _, ok := oldESConfigPath[path]; !ok {
			addedESPath[path] = path
		}
	}
	var removedESPath map[string]string
	for _, path := range oldESConfigPath {
		if _, ok := newESConfigPath[path]; !ok {
			removedESPath[path] = path
		}
	}
	//新增加的节点，需要查询出节点uuid等信息，再更新到console
	//删除的节点, 从hostInfo中删掉即可

	//先处理删掉了的
	for _, cluster := range hostInfo.Clusters {
		for i, node := range cluster.Nodes {
			if _, ok := removedESPath[node.ConfigPath]; ok {
				cluster.Nodes = append(cluster.Nodes[:i], cluster.Nodes[i+1:]...)
			}
		}
	}

	////处理增加的节点
	//for _, cluster := range hostInfo.Clusters {
	//	for _, clusterNew := range hostInfoNew.Clusters {
	//		for _, node := range clusterNew.Nodes {
	//
	//		}
	//	}
	//}
}

func changeESClusterNumbers() {

}

func changeESConfigFile() {
	hostInfo := config.GetHostInfo()
	for _, cluster := range hostInfo.Clusters {
		for _, node := range cluster.Nodes {
			fileContent, err := util.FileGetContent(node.ConfigPath)
			if err != nil {
				log.Printf("read es config file failed : %s", node.ConfigPath)
				continue
			}
			node.ConfigFileContent = fileContent
		}
	}
	UpdateHostInfoToConsole(hostInfo)
}

func UpdateHostInfoToConsole(host *model.Host) bool {
	consoleHost := host.ToConsoleModel()
	body, err := json.Marshal(consoleHost)
	if err != nil {
		log.Println(err)
		return false
	}
	reqPath := strings.ReplaceAll(api.UrlUpdateHostInfo, ":instance_id", host.AgentID)
	url := fmt.Sprintf("%s%s", config.UrlConsole(), reqPath)
	log.Printf("update host info: %s\n", body)
	var req = util.NewPutRequest(url, body)
	result, err := util.ExecuteRequest(req)
	if err != nil {
		log.Println(err)
		return false
	}
	/**
	{
	  "_id": "cbr188lath20g5m72m30",
	  "new_clusters": null,
	  "result": "updated"
	}
	*/
	var resultMap map[string]string
	util.MustFromJSONBytes(result.Body, &resultMap)
	if v, ok := resultMap["result"]; ok && v == "updated" {
		config.SetHostInfo(host)
	}
	return false
}

func mergeNodeInfo(clusterFromPID *model.Cluster, clusterFromKV *model.Cluster) {
	for _, node := range clusterFromPID.Nodes {
		for _, kvNode := range clusterFromKV.Nodes {
			if node.Name == kvNode.Name {
				node.ID = kvNode.ID
				node.TaskOwner = kvNode.TaskOwner
				//node.ConfigPath = kvNode.ConfigPath
				//node.NetWorkHost = kvNode.NetWorkHost
				//node.ClusterName = kvNode.ClusterName
				//node.HttpPort = kvNode.HttpPort
				//node.ConfigFileContent = kvNode.ConfigFileContent
				//node.LogPath = kvNode.LogPath
				//node.Ports = kvNode.Ports
			}
		}
	}
}

func Register(success chan bool) {
	hostInfo, err := host.RegisterHost()
	if err != nil {
		log.Printf("manage.Register: register host failed:\n%v\n", err)
		success <- false
		return
	}
	if hostInfo != nil {
		tmpHostInfo := UploadNodeInfos(hostInfo)
		if tmpHostInfo != nil {
			config.SetHostInfo(tmpHostInfo) //到这一步，才算真正的完成agent注册
			success <- true
			return
		}
	}
	success <- false
}

func HeartBeat() {
	host := config.GetHostInfo()
	if config.GetHostInfo() == nil {
		return
	}
	hbClient := hearbeat.NewDefaultClient(time.Second*10, host.AgentID)
	go hbClient.Heartbeat(func() string {
		ht := config.GetHostInfo()
		if ht == nil {
			return ""
		}
		return fmt.Sprintf("{'instance_id':%s}", ht.AgentID)
	}, func(content string) bool {
		var resp model.HeartBeatResp
		err := json.Unmarshal([]byte(content), &resp)
		if err != nil {
			log.Printf("manage.HeartBeat: heart beat failed: %s , resp: %s", err, content)
			return false
		}
		if resp.Result != "ok" {
			log.Printf("heartbeat failed: %s", resp.Result)
			return false
		}
		taskMap := resp.TaskState
		hostInfo := config.GetHostInfo()
		clusters := hostInfo.Clusters
		clusterTaskOwner := make(map[string]string)
		for k, val := range taskMap {
			if val.ClusterMetric == "" {
				continue
			}
			clusterTaskOwner[k] = val.ClusterMetric
		}
		for _, cluster := range clusters {
			if v, ok := clusterTaskOwner[cluster.ID]; ok && v != "" {
				cluster.Task.ClusterMetric.Owner = true
				cluster.Task.ClusterMetric.TaskNodeID = v
			}
		}
		config.SetHostInfo(host)
		return true
	})
}

func UploadNodeInfos(host *model.Host) *model.Host {
	newClusterInfos := GetESNodeInfos(host.Clusters)
	if newClusterInfos == nil {
		log.Panic("manage.UploadNodeInfos: getESNodeInfos failed. all passwords are wrong?? es crashed??")
		return nil
	}
	host.Clusters = newClusterInfos
	reqPath := strings.ReplaceAll(api.UrlUpdateHostInfo, ":instance_id", host.AgentID)
	url := fmt.Sprintf("%s%s", config.UrlConsole(), reqPath)
	instance := host.ToConsoleModel()
	body, _ := json.Marshal(instance)
	log.Printf("UploadNodeInfos, 请求: %s\n", string(body))
	var req = util.NewPutRequest(url, body)
	result, err := util.ExecuteRequest(req)
	if err != nil {
		log.Printf("manage.UploadNodeInfos: uploadNodeInfos failed: %v\n", err)
		return nil
	}
	log.Printf("manage.UploadNodeInfos: upNodeInfo resp: %s\n", string(result.Body))
	var resp model.UpNodeInfoResponse
	err = json.Unmarshal(result.Body, &resp)
	if err != nil {
		log.Printf("manage.UploadNodeInfos: uploadNodeInfos failed: %v\n", err)
		return nil
	}

	var resultCluster []*model.Cluster
	for clusterName, val := range resp.Cluster {
		for _, cluster := range host.Clusters {
			if cluster.Name == clusterName {
				if valMap, ok := val.(map[string]interface{}); ok {
					if authInfo, ok := valMap["basic_auth"]; ok {
						if auth, ok := authInfo.(map[string]string); ok {
							cluster.UserName = auth["username"]
							cluster.Password = auth["password"]
						}
					}
					if clusterid, ok := valMap["cluster_id"]; ok {
						cluster.ID = clusterid.(string)
					}
					resultCluster = append(resultCluster, cluster)
				}
			}
		}
	}
	host.Clusters = resultCluster
	if resp.IsSuccessed() {
		config.SetHostInfo(host)
		return host
	}
	return nil
}

func GetESNodeInfos(clusterInfos []*model.Cluster) []*model.Cluster {
	var clusters []*model.Cluster
	for _, cluster := range clusterInfos {
		for _, node := range cluster.Nodes {
			if node.HttpPort == 0 {
				continue
			}
			url := fmt.Sprintf("%s/_nodes/_local", node.GetNetWorkHost(cluster.GetSchema()))
			var req = util.NewGetRequest(url, nil)
			if cluster.UserName != "" && cluster.Password != "" {
				req.SetBasicAuth(cluster.UserName, cluster.Password)
			}
			result, err := util.ExecuteRequest(req)
			if err != nil {
				log.Printf("manage.GetESNodeInfos: username or password error: %v\n", err)
				continue //账号密码错误
			}
			resultMap := host.ParseNodeInfo(string(result.Body))
			if v, ok := resultMap["node_id"]; ok {
				node.ID = v
			}
			if v, ok := resultMap["node_name"]; ok {
				node.Name = v
			}
			if v, ok := resultMap["version"]; ok {
				cluster.Version = v
			}
		}
		clusters = append(clusters, cluster)
	}
	return clusters
}
