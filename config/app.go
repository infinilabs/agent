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
	"sync"
	"time"
)

type AppConfig struct {
	MajorIpPattern string            `config:"major_ip_pattern"`
	Labels         map[string]string `config:"labels"`
	Tags           []string          `config:"tags"`
	Manager        *Manager          `config:"manager"`
}

type Manager struct {
	Endpoint string `config:"endpoint"`
}

var EnvConfig *AppConfig
var hostInfo *model.Instance
var hostInfoObserver []func(newHostInfo *model.Instance)
var instanceLock sync.RWMutex

const (
	ESClusterDefaultName string = "elasticsearch"
	ESConfigFileName            = "elasticsearch.yml"
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

func IsAgentActivated() bool {
	instanceInfo := GetInstanceInfo()
	if instanceInfo == nil || !instanceInfo.IsRunning {
		return false
	}
	return true
}

func GetInstanceInfo() *model.Instance {
	if hostInfo != nil {
		return hostInfo
	}
	hostInfo = getInstanceInfoFromKV()
	return hostInfo
}

func GetOrInitInstanceInfo() *model.Instance {
	if hostInfo != nil {
		return hostInfo
	}
	hostInfo = getInstanceInfoFromKV()
	if hostInfo == nil {
		hostInfo = &model.Instance{
			IPs:       util.GetLocalIPs(),
			Host:      agent.HostInfo{},
		}
		_, majorIp, _, err := util.GetPublishNetworkDeviceInfo(EnvConfig.MajorIpPattern)
		if err != nil {
			log.Error(err)
		}
		hostInfo.MajorIP = majorIp
	}
	SetInstanceInfo(hostInfo)
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
	instanceLock.Lock()
	defer instanceLock.Unlock()
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
	//1. new es node => set status 'online'
	//2. nodes not in current list => set status 'offline'
	log.Debugf("UpdateInstanceInfo, new: %s", util.MustToJSON(instanceNew))
	clusterRet := make(map[string]*model.Cluster) //key: cluster id, value: *model.Cluster
	nodeRet := make(map[string]*model.Node)       //key: cluster id+ node.ESHomePath， value: *model.Node

	for _, cluster := range instanceNew.Clusters {
		if cluster.ID == "" {
			continue
		}
		clusterRet[cluster.ID] = cluster
		for _, node := range cluster.Nodes {
			node.Status = model.NodeStatusOnline
			nodeRet[cluster.ID+node.ESHomePath] = node
		}
	}

	instanceKV := GetInstanceInfo()
	log.Debugf("UpdateInstanceInfo, old(in kv): %s", util.MustToJSON(instanceKV))
	for _, cluster := range instanceKV.Clusters {
		if cluster.ID == "" {
			continue
		}
		_, ok := clusterRet[cluster.ID]
		if !ok {
			clusterRet[cluster.ID] = cluster
		}
		for _, node := range cluster.Nodes {
			_, ok = nodeRet[cluster.ID+node.ESHomePath]
			if !ok {
				node.Status = model.NodeStatusOffline
				nodeRet[cluster.ID+node.ESHomePath] = node
			}
		}
	}

	instanceNew.Clusters = nil
	for _, cluster := range clusterRet {
		cluster.Nodes = nil
		instanceNew.Clusters = append(instanceNew.Clusters, cluster)
		for key, node := range nodeRet {
			if strings.HasPrefix(key, cluster.ID) {
				cluster.Nodes = append(cluster.Nodes, node)
			}
		}
	}
	log.Debugf("UpdateInstanceInfo, final: %s", util.MustToJSON(instanceNew))
	SetInstanceInfo(instanceNew)
}

func SetInstanceInfoNoNotify(host *model.Instance) error {
	if host == nil {
		return errors.New("host info can not be nil")
	}

	instanceLock.Lock()
	defer instanceLock.Unlock()
	hostInfo = host
	event.UpdateAgentID(hostInfo.AgentID)
	event.UpdateHostID(hostInfo.HostID)
	hostByte, _ := json.Marshal(host)
	return kv.AddValue(agent.KVInstanceBucket, []byte(agent.KVInstanceInfo), hostByte)
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
	ret,_ := json.Marshal(hostInf)
	log.Debugf(string(ret))
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
