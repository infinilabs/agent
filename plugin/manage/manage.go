package manage

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage/hearbeat"
	"infini.sh/agent/plugin/manage/instance"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"strings"
	"time"
)

func Init() {
	_, err := isAgentAliveInConsole()
	if err != nil {
		log.Errorf("manage.Init: %v", err)
		return
	}

	if instance.IsRegistered() {
		HeartBeat()
		checkInstanceUpdate()
		config.UpdateAgentBootTime()
	} else {
		registerChan := make(chan bool)
		go Register(registerChan)
		select {
		case ok := <-registerChan:
			log.Debugf("manage.Init: register host %t", ok)
			if ok {
				HeartBeat()
				checkInstanceUpdate()
				config.UpdateAgentBootTime()
			}
		case <-time.After(time.Second * 60):
			log.Error("manage.Init: register timeout.")
		}
	}
}

//get agent info from console. if nil => delete kv. if not => update task info.
func isAgentAliveInConsole() (bool, error) {
	hostInfo := config.GetInstanceInfo()
	if hostInfo == nil {
		return false, nil
	}

	resp, err := GetHostInfoFromConsole(hostInfo.AgentID)
	if err != nil {
		return false, err
	}
	if !resp.Found {
		config.DeleteInstanceInfo()
		return false, nil
	}
	if !hostInfo.IsRunning {
		log.Debug("agent not running, wait console confirm")
		return false, nil
	}
	for _, cluster := range hostInfo.Clusters {
		for _, esCluster := range resp.Instance.Clusters {
			if cluster.UUID == esCluster.ClusterUUID {
				cluster.UpdateTask(&esCluster.Task)
			}
		}
	}
	hostInfo.AgentPort = config.GetListenPort()
	hostInfo.TLS = config.IsHTTPS()
	config.SetInstanceInfo(hostInfo)
	return true, nil
}

func GetHostInfoFromConsole(agentID string) (*model.GetAgentInfoResponse, error) {
	reqPath := strings.ReplaceAll(config.UrlGetInstanceInfo, ":instance_id", agentID)
	url := fmt.Sprintf("%s%s", config.GetManagerEndpoint(), reqPath)
	var req = util.NewGetRequest(url, []byte(""))
	result, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	var resp model.GetAgentInfoResponse
	err = json.Unmarshal(result.Body, &resp)
	return &resp, err
}

func checkInstanceUpdate() {
	hostUpdateTask := task.ScheduleTask{
		Description: "update agent host info",
		Type:        "interval",
		Interval:    "10s",
		Task: func(ctx context.Context) {
			if config.GetInstanceInfo() == nil || !config.GetInstanceInfo().IsRunning {
				return
			}
			instance.UpdateProcessInfo()
			isChanged, err := instance.IsHostInfoChanged()
			if err != nil {
				log.Errorf("manage.checkInstanceUpdate: update host info failed : %v", err)
				return
			}
			if !isChanged {
				return
			}
			log.Debugf("manage.checkInstanceUpdate: host info change")
			updateChan := make(chan bool)
			go UpdateInstanceInfo(updateChan)

			select {
			case ok := <-updateChan:
				log.Debugf("manage.checkInstanceUpdate: update host info %t", ok)
			case <-time.After(time.Second * 60):
			}
		},
	}
	task.RegisterScheduleTask(hostUpdateTask)
}

func UpdateInstanceInfo(isSuccess chan bool) {

	hostKV := config.GetInstanceInfo()
	hostPid, err := instance.GetInstanceInfo()
	if err != nil {
		log.Errorf("get host info failed: %v", err)
		isSuccess <- false
		return
	}
	//没报错，但没有进程信息，说明当前没有es实例在运行了
	if hostPid.Clusters == nil {
		hostKV.Clusters = nil
		config.SetInstanceInfo(hostKV)
		UploadNodeInfos(hostKV)
		isSuccess <- true
		return
	}
	hostPid.IsRunning = hostKV.IsRunning
	hostPid.AgentID = hostKV.AgentID
	hostPid.AgentPort = hostKV.AgentPort
	hostPid.HostID = hostKV.HostID
	hostPid.BootTime = hostKV.BootTime
	hostPid.TLS = hostKV.TLS
	hostPid.MajorIP = hostKV.MajorIP
	count := 0
	for _, cluster := range hostPid.Clusters {
		for _, clusterKv := range hostKV.Clusters {
			if clusterKv.Name == cluster.Name {
				cluster.ID = clusterKv.ID
				cluster.UserName = clusterKv.UserName
				cluster.Password = clusterKv.Password
				count++
			}
		}
	}

	log.Debugf("manage.UpdateInstanceInfo: %v\n", hostPid)
	if count != len(hostPid.Clusters) {
		//new cluster added -> 1. get auth info from console. 2. upload node info.
		if UploadNodeInfos(hostPid) != nil {
			UploadNodeInfos(config.GetInstanceInfo())
		}
	} else {
		UploadNodeInfos(hostPid)
	}
	isSuccess <- true
}

func Register(success chan bool) {
	log.Info("register agent to console")
	instanceInfo, err := instance.RegisterInstance()
	if err != nil {
		log.Errorf("manage.Register: %v\n", err)
		success <- false
		return
	}
	if instanceInfo == nil {
		log.Errorf("manage.Register: register agent Failed. all passwords are wrong?? es crashed?? cluster not register in console??\n")
		success <- false
		return
	}
	if instanceInfo != nil {
		var tmpInstanceInfo *model.Instance
		if instanceInfo.IsRunning {
			log.Debugf("manage.Register: %v\n", instanceInfo)
			tmpInstanceInfo = UploadNodeInfos(instanceInfo)
			if tmpInstanceInfo != nil {
				config.SetInstanceInfo(tmpInstanceInfo)
				success <- true
				return
			}
		} else {
			log.Info("registering, waiting for review")
			config.SetInstanceInfo(instanceInfo)
			success <- true
		}
	} else {
		success <- false
	}
}

func RegisterCallback(resp *model.RegisterResponse) (bool, error) {
	log.Debugf("manage.RegisterCallback: %v\n", util.MustToJSON(resp))
	instanceInfo, err := instance.UpdateClusterInfoFromResp(config.GetInstanceInfo(), resp)
	if err != nil {
		return false, err
	}
	if UploadNodeInfos(instanceInfo) == nil {
		return false, nil
	}
	instanceInfo.IsRunning = true
	config.SetInstanceInfo(instanceInfo)
	return true, nil
}

func HeartBeat() {
	instanceInfo := config.GetInstanceInfo()
	if instanceInfo == nil {
		return
	}
	hbClient := hearbeat.NewDefaultClient(time.Second*10, instanceInfo.AgentID)
	go hbClient.Heartbeat(func() string {
		ht := config.GetInstanceInfo()
		if ht == nil {
			return ""
		}
		return fmt.Sprintf("{'instance_id':%s}", ht.AgentID)
	}, func(content string) bool {
		if strings.Contains(content, "record not found") {
			config.DeleteInstanceInfo()
			panic("agent deleted in console. please restart\n")
		}
		var resp model.HeartBeatResp
		err := json.Unmarshal([]byte(content), &resp)
		if err != nil {
			log.Errorf("manage.HeartBeat: heart beat failed: %s , resp: %s", err, content)
			return false
		}
		if !resp.Success {
			log.Errorf("heartbeat failed, resp: %s", content)
			return false
		}
		taskMap := resp.TaskState
		instance := config.GetInstanceInfo()
		clusters := instance.Clusters
		clusterTaskOwner := make(map[string]string)
		for k, val := range taskMap {
			if val.ClusterMetric == "" {
				continue
			}
			clusterTaskOwner[k] = val.ClusterMetric
		}

		changed := 0
		for _, cluster := range clusters {
			if v, ok := clusterTaskOwner[cluster.ID]; ok {
				if cluster.Task.ClusterMetric.TaskNodeID == v {
					continue
				}
				cluster.Task.ClusterMetric.Owner = true
				cluster.Task.ClusterMetric.TaskNodeID = v
				cluster.Task.NodeMetric.Owner = true
				changed++
			} else {
				if cluster.Task != nil && cluster.Task.ClusterMetric.TaskNodeID != "" {
					changed++
				}
				cluster.Task.ClusterMetric.Owner = false
				cluster.Task.ClusterMetric.TaskNodeID = ""
				cluster.Task.NodeMetric.Owner = false
			}
		}
		if changed > 0 {
			config.SetInstanceInfo(instance)
		}
		return true
	})
}

func UploadNodeInfos(instanceInfo *model.Instance) *model.Instance {
	newClusterInfos := GetESNodeInfos(instanceInfo.Clusters)
	if newClusterInfos == nil {
		log.Errorf("manage.UploadNodeInfos: getESNodeInfos failed. please check: \n1. all passwords are wrong?  \n2. es crashed? \n3. cluster not register in console?")
		return nil
	}
	instanceInfo.Clusters = newClusterInfos
	reqPath := strings.ReplaceAll(config.UrlUpdateInstanceInfo, ":instance_id", instanceInfo.AgentID)
	url := fmt.Sprintf("%s%s", config.GetManagerEndpoint(), reqPath)
	instance := instanceInfo.ToConsoleModel()
	body, _ := json.Marshal(instance)
	log.Debugf("UploadNodeInfos, request body: %s\n", string(body))
	var req = util.NewPutRequest(url, body)
	result, err := util.ExecuteRequest(req)
	if err != nil {
		log.Errorf("manage.UploadNodeInfos: uploadNodeInfos failed: %v\n", err)
		return nil
	}
	log.Debugf("manage.UploadNodeInfos: upNodeInfo resp: %s\n", string(result.Body))
	var resp model.UpNodeInfoResponse
	err = json.Unmarshal(result.Body, &resp)
	if err != nil {
		log.Errorf("manage.UploadNodeInfos: uploadNodeInfos failed: %v\n", err)
		return nil
	}

	var clustersResult []*model.Cluster
	for clusterName, val := range resp.Cluster {
		for _, cluster := range instanceInfo.Clusters {
			if cluster.Name == clusterName {
				if valMap, ok := val.(map[string]interface{}); ok {
					if authInfo, ok := valMap["basic_auth"]; ok {
						if auth, ok := authInfo.(map[string]string); ok {
							cluster.UserName = auth["username"]
							cluster.Password = auth["password"]
						}
					}
					if clusterId, ok := valMap["cluster_id"]; ok {
						cluster.ID = clusterId.(string)
					}
				}
				clustersResult = append(clustersResult, cluster)
			}
		}
	}
	if clustersResult != nil {
		instanceInfo.Clusters = clustersResult
	}
	if resp.IsSuccessed() {
		config.SetInstanceInfo(instanceInfo)
		return instanceInfo
	}
	return nil
}

func GetESNodeInfos(clusterInfos []*model.Cluster) []*model.Cluster {
	var clusters []*model.Cluster
	for _, cluster := range clusterInfos {
		log.Debugf("manage.GetESNodeInfos: %v\n", cluster)
		for _, node := range cluster.Nodes {
			if node.HttpPort == 0 {
				validatePort := instance.ValidatePort(node.NetWorkHost,cluster.GetSchema(),cluster.UUID,cluster.UserName,cluster.Password,node.Ports)
				if validatePort == 0 {
					continue
				}
				node.HttpPort = validatePort
			}
			url := fmt.Sprintf("%s/_nodes/_local", node.GetEndPoint(cluster.GetSchema()))
			var req = util.NewGetRequest(url, nil)
			if cluster.UserName != "" && cluster.Password != "" {
				req.SetBasicAuth(cluster.UserName, cluster.Password)
			}
			result, err := util.ExecuteRequest(req)
			if err != nil {
				log.Errorf("manage.GetESNodeInfos: username or password error: %v\n", err)
				continue
			}
			log.Debugf("manage.GetESNodeInfos: %s\n", string(result.Body))
			resultMap := instance.ParseNodeInfo(string(result.Body))
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
	log.Debugf("manage.GetESNodeInfos: %v\n", clusters)
	return clusters
}
