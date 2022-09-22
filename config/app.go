package config

import (
	"encoding/json"
	"errors"
	log "github.com/cihub/seelog"
	"infini.sh/agent/model"
	metadata "infini.sh/agent/plugin/manage/elastic-metadata"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"strconv"
	"strings"
	"time"
)

type AppConfig struct {
	MajorIpPattern string   `config:"major_ip_pattern"`
	Labels         Label    `config:"labels"`
	Tags           []string `config:"tags"`
	Manager        *Manager `config:"manager"`
}

type Label struct {
	Env string `config:"env"`
}

type Manager struct {
	Endpoint string `config:"endpoint"`
}

var EnvConfig *AppConfig
var hostInfo *model.Instance
var hostInfoObserver []func(newHostInfo *model.Instance)

const (
	ESClusterDefaultName string = "elasticsearch"
	ESConfigFileName            = "elasticsearch.yml"
	KVHostInfo                  = "host_info"
	KVHostBucket                = "host_bucket"
)

const (
	UrlUploadInstanceInfo string = "/agent/instance"
	UrlUpdateInstanceInfo        = "/agent/instance/:instance_id"
	UrlHearBeat                  = "/agent/instance/:instance_id/_heartbeat"
	UrlGetInstanceInfo           = "/agent/instance/:instance_id"
)

func InitConfig() {
	appConfig := &AppConfig{}
	ok, err := env.ParseConfig("agent", appConfig)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("config.InitConfig: can not find agent config")
	}
	EnvConfig = appConfig
	hostInfoObserver = make([]func(newHostInfo *model.Instance), 1)
	RegisterHostInfoObserver(metadata.HostInfoChanged)
}

func RegisterHostInfoObserver(fn func(newHostInfo *model.Instance)) {
	hostInfoObserver = append(hostInfoObserver, fn)
}

func NotifyHostInfoObserver(newHostInfo *model.Instance) {
	for i := 0; i < len(hostInfoObserver); i++ {
		if hostInfoObserver[i] != nil {
			hostInfoObserver[i](newHostInfo)
		}
	}
}

func GetManagerEndpoint() string {
	if EnvConfig == nil {
		return ""
	}
	endPoint := EnvConfig.Manager.Endpoint
	if strings.HasSuffix(endPoint, "/") {
		endPoint = endPoint[:len(endPoint)-1]
	}
	return EnvConfig.Manager.Endpoint
}

func GetListenPort() uint {
	if EnvConfig == nil {
		return 0
	}
	bindAddress := global.Env().SystemConfig.APIConfig.NetworkConfig.Binding
	if strings.Contains(bindAddress, ":") {
		temps := strings.Split(bindAddress, ":")
		port, _ := strconv.Atoi(temps[1])
		return uint(port)
	}
	return 0
}

func IsHTTPS() bool {
	return global.Env().SystemConfig.APIConfig.TLSConfig.TLSEnabled
}

func GetInstanceInfo() *model.Instance {
	if hostInfo != nil {
		return hostInfo
	}
	hostInfo = getInstanceInfoFromKV()
	return hostInfo
}

func UpdateAgentBootTime(){
	instanceInfo := GetInstanceInfo()
	instanceInfo.BootTime = time.Now().UnixMilli()
	SetInstanceInfo(instanceInfo)
}

func SetInstanceInfo(host *model.Instance) error {
	if host == nil {
		return errors.New("host info can not be nil")
	}

	hostInfo = host
	event.UpdateAgentID(hostInfo.AgentID)
	event.UpdateHostID(hostInfo.HostID)
	hostByte, _ := json.Marshal(host)
	if host.IsRunning {
		NotifyHostInfoObserver(hostInfo)
	}
	return kv.AddValue(agent.KVInstanceBucket, []byte(agent.KVInstanceInfo), hostByte)
}

func UpdateInstanceInfo(instanceNew *model.Instance) {
	//1. 新增加的集群和节点，状态设置为online
	//2. 之前存在但新的列表中不存在的集群/节点，状态设置为offline
	log.Debugf("UpdateInstanceInfo, new: %s",util.MustToJSON(instanceNew))
	clusterRet := make(map[string]*model.Cluster) //key: 集群ID, value: *model.Cluster
	nodeRet := make(map[string]*model.Node)       //key: 集群id+节点ID， value: *model.Node

	for _, cluster := range instanceNew.Clusters {
		clusterRet[cluster.UUID] = cluster
		for _, node := range cluster.Nodes {
			node.Status = model.NodeStatusOnline
			nodeRet[cluster.UUID+node.ESHomePath] = node
		}
	}

	instanceKV := GetInstanceInfo()
	log.Debugf("UpdateInstanceInfo, old(in kv): %s", util.MustToJSON(instanceKV))
	for _, cluster := range instanceKV.Clusters {
		_, ok := clusterRet[cluster.UUID]
		if !ok {
			clusterRet[cluster.UUID] = cluster
		}
		for _, node := range cluster.Nodes {
			_, ok = nodeRet[cluster.UUID+node.ESHomePath]
			if !ok {
				node.Status = model.NodeStatusOffline
				nodeRet[cluster.UUID+node.ESHomePath] = node
			}
		}
	}

	instanceNew.Clusters = nil
	for _, cluster := range clusterRet {
		cluster.Nodes = nil
		instanceNew.Clusters = append(instanceNew.Clusters, cluster)
		for key, node := range nodeRet {
			if strings.HasPrefix(key, cluster.UUID) {
				cluster.Nodes = append(cluster.Nodes, node)
			}
		}
	}
	log.Debugf("UpdateInstanceInfo, final: %s",util.MustToJSON(instanceNew))
	SetInstanceInfo(instanceNew)
}

func DeleteInstanceInfo() error {
	hostInfo = nil
	return kv.DeleteKey(agent.KVInstanceBucket, []byte(agent.KVInstanceInfo))
}

func ReloadHostInfo() {
	hostInf := getInstanceInfoFromKV()
	if hostInf == nil {
		return
	}
}

var host *model.Instance

func getInstanceInfoFromKV() *model.Instance {
	hs, err := kv.GetValue(agent.KVInstanceBucket, []byte(agent.KVInstanceInfo))
	if err != nil {
		log.Error(err)
		return nil
	}
	if hs == nil {
		return nil
	}
	err = json.Unmarshal(hs, &host)
	if err != nil {
		log.Errorf("config.getInstanceInfoFromKV: %v\n", err)
		return nil
	}
	return host
}
