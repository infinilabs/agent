package config

import (
	"encoding/json"
	"errors"
	"infini.sh/agent/model"
	metadata "infini.sh/agent/plugin/manage/elastic-metadata"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"io"
	"log"
	"os"
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
var hostInfo *model.Host
var hostInfoObserver []func(newHostInfo *model.Host)

const (
	KVHostInfo           string = "agent_host_info"
	KVAgentBucket               = "agent_bucket"
	ESClusterDefaultName        = "elasticsearch"
	ESConfigFileName            = "elasticsearch.yml"
)

func InitConfig() {
	OutputLogsToStd()
	appConfig := &AppConfig{}
	ok, err := env.ParseConfig("agent", appConfig)
	if err != nil {
		panic(err)
	}
	if !ok {
		panic("config.InitConfig: can not find agent config")
	}
	EnvConfig = appConfig
	hostInfoObserver = make([]func(newHostInfo *model.Host), 1)
	RegisterHostInfoObserver(metadata.HostInfoChanged)
}

func RegisterHostInfoObserver(fn func(newHostInfo *model.Host)) {
	hostInfoObserver = append(hostInfoObserver, fn)
}

func NotifyHostInfoObserver(newHostInfo *model.Host) {
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

func GetHostInfo() *model.Host {
	//if hostInfo != nil {
	//	return hostInfo
	//}
	hostInfo = getHostInfoFromKV()
	return hostInfo
}

func SetHostInfo(host *model.Host) error {
	if host == nil {
		return errors.New("host info can not be nil")
	}

	hostInfo = host
	hostByte, _ := json.Marshal(host)
	NotifyHostInfoObserver(hostInfo)
	return kv.AddValue(KVAgentBucket, []byte(KVHostInfo), hostByte)
}

func DeleteHostInfo() error {
	return kv.DeleteKey(KVAgentBucket, []byte(KVHostInfo))
}

func ReloadHostInfo() {
	hostInf := getHostInfoFromKV()
	if hostInf == nil {
		return
	}
	//hostInf.AgentPort = uint(GetListenPort())
	//hostInf.TLS = IsHTTPS()
	//SetHostInfo(hostInf)
}

func OutputLogsToStd() {
	logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

var host *model.Host

func getHostInfoFromKV() *model.Host {
	hs, err := kv.GetValue(KVAgentBucket, []byte(KVHostInfo))
	if err != nil {
		log.Println(err)
		return nil
	}
	if hs == nil {
		return nil
	}
	err = json.Unmarshal(hs, &host)
	if err != nil {
		log.Printf("config.getHostInfoFromKV: %v\n", err)
		return nil
	}
	return host
}
