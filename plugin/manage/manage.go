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
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/task"
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
	if host.IsRegistered() {
		HeartBeat()
		checkHostUpdate()
	} else {
		go Register()
		if <-registerSuccess {
			log.Printf("manage.Init: register host success")
			HeartBeat()
			checkHostUpdate()
		}
	}
}

/**
更新主机信息: ip、es集群
*/
func checkHostUpdate() {
	hostUpdateTask := task.ScheduleTask{
		Description: "update agent host info",
		Type:        "interval",
		Interval:    "5s",
		Task: func(ctx context.Context) {
			ok, err := host.IsHostInfoChanged()
			if err != nil {
				log.Printf("manage.checkHostUpdate: update host info failed : %v", err)
			}
			if ok {
				registerSuccess = make(chan bool)
				go Register()
				if <-registerSuccess {
					log.Printf("manage.checkHostUpdate: update host info success")
				}
			}
		},
	}
	task.RegisterScheduleTask(hostUpdateTask)
}

func Register() {
	defer close(registerSuccess)
	hostInfo, err := host.RegisterHost()
	if err != nil {
		log.Printf("manage.Register: register host failed:\n%v\n", err)
		registerSuccess <- false
		return
	}
	if hostInfo != nil {
		tmpHostInfo := UploadNodeInfos(hostInfo)
		if tmpHostInfo != nil {
			config.SetHostInfo(tmpHostInfo) //到这一步，才算真正的完成agent注册
			registerSuccess <- true
			return
		}
	}
	registerSuccess <- false
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
			log.Printf("manage.HeartBeat: heart beat failed: %s\n", err)
			return false
		}
		if resp.Result != "ok" {
			return false
		}
		taskMap := resp.TaskState
		hostInfo := config.GetHostInfo()
		clusters := hostInfo.Clusters
		clusterTaskOwner := make(map[string]bool)
		for k, v := range taskMap {
			if val, ok := v.(string); ok && val != "" {
				clusterTaskOwner[k] = true
			}
		}
		for _, cluster := range clusters {
			if v, ok := clusterTaskOwner[cluster.ID]; ok && v {
				cluster.TaskOwner = true
			}
		}
		config.SetHostInfo(host)
		return true
	})
}

func UploadNodeInfos(host *model.Host) *model.Host {
	nodeInfos := GetESNodeInfos(host.Clusters)
	if nodeInfos == nil {
		log.Panic("manage.UploadNodeInfos: getESNodeInfos failed. all passwords are wrong?? es crashed??")
		return nil
	}
	reqPath := strings.ReplaceAll(api.UrlUploadNodeInfo, ":instance_id", host.AgentID)
	url := fmt.Sprintf("%s/%s", config.UrlConsole(), reqPath)
	fmt.Printf("注册节点信息,url: %s\n", url)
	var esClusters []*agent.ESCluster
	for _, info := range nodeInfos {
		esClusters = append(esClusters, info.ConvertToESCluster())
	}
	body, _ := json.Marshal(esClusters)
	fmt.Printf("注册节点: %v\n", string(body))
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
	if resp.IsSuccessed() {
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
			clusters = append(clusters, cluster)
		}
	}
	return clusters
}
