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

//TODO 账号密码错误时
func Init() {
	if host.IsRegistered() {
		HeartBeat()
		checkHostUpdate()
	} else {
		go Register()
		if <-registerSuccess {
			log.Printf("register host success")
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
		Interval:    "10s",
		Task: func(ctx context.Context) {
			ok, err := host.IsHostInfoChanged()
			if err != nil {
				log.Printf("update host info failed : %v", err)
			}
			if ok {
				registerSuccess = make(chan bool)
				go Register()
				if <-registerSuccess {
					log.Printf("update host info success")
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
		log.Printf("register host failed:\n%v\n", err)
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
	hbClient := hearbeat.NewDefaultClient(time.Second*5, host.AgentID)
	go hbClient.Heartbeat(func() (string, error) {
		ht := config.GetHostInfo()
		fmt.Println("heart beat")
		return fmt.Sprintf("{'instance_id':%s}", ht.AgentID), nil
	}, func(content string) (bool, error) {
		//TODO 解析返回结果
		fmt.Printf("心跳API返回: %s", content)
		return true, nil
	})
}

func UploadNodeInfos(host *model.Host) *model.Host {
	nodeInfos := GetESNodeInfos(host.Clusters)
	if nodeInfos == nil {
		log.Panic("getESNodeInfos failed. all passwords are wrong?? es crashed??")
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
		log.Printf("uploadNodeInfos failed: \n %v", err)
		return nil
	}
	//TODO 解析返回结果
	fmt.Println("上传Node信息，返回: ")
	fmt.Println(string(result.Body))
	var resp model.UpNodeInfoResponse
	err = json.Unmarshal(result.Body, &resp)
	if err != nil {
		log.Printf("uploadNodeInfos failed: \n %v", err)
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
			url := fmt.Sprintf("%s://%s/_nodes/_local", cluster.GetSchema(), node.GetNetWorkHost(cluster.GetSchema()))
			var req = util.NewGetRequest(url, nil)
			if cluster.UserName != "" && cluster.Password != "" {
				req.SetBasicAuth(cluster.UserName, cluster.Password)
			}
			result, err := util.ExecuteRequest(req)
			if err != nil {
				fmt.Printf("账号密码错误: %v\n", err)
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
