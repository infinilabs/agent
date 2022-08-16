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
	_, err := isAgentAliveInConsole()
	if err != nil {
		log.Printf("manage.Init: %v", err)
		return
	}
	doManage()
}

func doManage() {
	if host.IsRegistered() {
		HeartBeat()
		checkHostUpdate()
	} else {
		registerChan := make(chan bool)
		go Register(registerChan)
		select {
		case ok := <-registerChan:
			log.Printf("manage.Init: register host %t", ok)
			if ok {
				HeartBeat()
				checkHostUpdate()
			}
		case <-time.After(time.Second * 60):
			log.Printf("manage.Init: register timeout.")
		}
	}
}

//判断agent在console那边是否还存在。 如果存在，则获取信息，并更新任务状态。 不存在的话，清空本地KV
func isAgentAliveInConsole() (bool, error) {
	hostInfo := config.GetHostInfo()
	if hostInfo == nil {
		return false, nil
	}

	resp, err := GetHostInfoFromConsole(hostInfo.AgentID)
	if err != nil {
		return false, err
	}
	if !resp.Found {
		config.DeleteHostInfo()
		return false, nil
	}
	for _, cluster := range hostInfo.Clusters {
		for _, esCluster := range resp.Instance.Clusters {
			if cluster.UUID == esCluster.ClusterUUID {
				cluster.UpdateTask(&esCluster.Task)
			}
		}
	}
	config.SetHostInfo(hostInfo)
	return true, nil
}

func GetHostInfoFromConsole(agentID string) (*model.GetAgentInfoResponse, error) {
	reqPath := strings.ReplaceAll(api.UrlGetHostInfo, ":instance_id", agentID)
	url := fmt.Sprintf("%s/%s", config.UrlConsole(), reqPath)
	var req = util.NewGetRequest(url, []byte(""))
	result, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	//log.Printf("manage.GetHostInfoFromConsole: getAgentInfo resp:\n %s\n", string(result.Body))
	var resp model.GetAgentInfoResponse
	err = json.Unmarshal(result.Body, &resp)
	return &resp, err
}

/**
更新主机信息: ip、es集群
*/
func checkHostUpdate() {
	//TODO 先检查任务是否有变化。主要针对agent挂掉之后，新启动的时候。
	hostUpdateTask := task.ScheduleTask{
		Description: "update agent host info",
		Type:        "interval",
		Interval:    "10s",
		Task: func(ctx context.Context) {
			if config.GetHostInfo() == nil {
				return
			}
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
		},
	}
	task.RegisterScheduleTask(hostUpdateTask)
}

func UpdateHostInfo(isSuccess chan bool, changeType config.ChangeType) {

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
	isSuccess <- true
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
	} else {
		success <- false
	}

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
		log.Printf("heartbear: %s\n", ht.AgentID)
		return fmt.Sprintf("{'instance_id':%s}", ht.AgentID)
	}, func(content string) bool {
		//log.Printf("heartbeat resp: %s\n", content)
		if strings.Contains(content, "record not found") {
			config.DeleteHostInfo()
			log.Panic("agent deleted in console. please restart agent\n")
			return false
		}
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
			if v, ok := clusterTaskOwner[cluster.ID]; ok {
				if v == "" {
					cluster.Task.ClusterMetric.Owner = false
					cluster.Task.ClusterMetric.TaskNodeID = ""
					cluster.Task.NodeMetric.Owner = false
				} else {
					cluster.Task.ClusterMetric.Owner = true
					cluster.Task.ClusterMetric.TaskNodeID = v
					cluster.Task.NodeMetric.Owner = true
				}
			} else {
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
				}
			}
		}
	}
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
