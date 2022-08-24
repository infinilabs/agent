package config

import (
	"encoding/json"
	"errors"
	log "github.com/cihub/seelog"
	"infini.sh/agent/model"
	metadata "infini.sh/agent/plugin/manage/elastic-metadata"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"strconv"
	"strings"
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
	KVInstanceInfo       string = "agent_instance_info"
	KVInstanceBucket            = "agent_bucket"
	ESClusterDefaultName        = "elasticsearch"
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

func SetInstanceInfo(host *model.Instance) error {
	if host == nil {
		return errors.New("host info can not be nil")
	}

	hostInfo = host
	hostByte, _ := json.Marshal(host)
	NotifyHostInfoObserver(hostInfo)
	return kv.AddValue(KVInstanceBucket, []byte(KVInstanceInfo), hostByte)
}

func DeleteInstanceInfo() error {
	hostInfo = nil
	return kv.DeleteKey(KVInstanceBucket, []byte(KVInstanceInfo))
}

func ReloadHostInfo() {
	hostInf := getInstanceInfoFromKV()
	if hostInf == nil {
		return
	}
}

var host *model.Instance

func getInstanceInfoFromKV() *model.Instance {
	hs, err := kv.GetValue(KVInstanceBucket, []byte(KVInstanceInfo))
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
