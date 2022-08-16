package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"infini.sh/agent/model"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/modules/elastic/adapter"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	Version       float32             `config:"version"`
	TLS           bool                `config:"tls"`
	Port          uint                `config:"port"`
	ConsoleConfig model.ConsoleConfig `config:"console"`
}

var EnvConfig AppConfig
var hostInfo *model.Host

const (
	KVHostInfo           string = "agent_host_info"
	KVAgentBucket               = "agent_bucket"
	ESClusterDefaultName        = "elasticsearch"
	ESConfigFileName            = "elasticsearch.yml"
)

type ChangeType int8

const (
	ChangeOfESConfigFile   ChangeType = 1 //es配置文件变更
	ChangeOfESConnect      ChangeType = 2 //es密码变更,es端口变更
	ChangeOfClusterNumbers ChangeType = 3 //当前主机包含的集群总数变更
	ChangeOfESNodeNumbers  ChangeType = 4 //当前主机包含的es节点总数变更
	ChangeOfLostInfo       ChangeType = 5 //本地kv中的主机信息丢失
	ChangeOfNothing        ChangeType = 6
	ChangeOfUnknown        ChangeType = 7
	ChangeOfAgentBasic     ChangeType = 8 //agent基础信息变更: agent的端口、tls
)

func InitConfig() {

	con := AppConfig{}
	ok, err := env.ParseConfig("agent", &con)
	bindAddress := global.Env().SystemConfig.APIConfig.NetworkConfig.Binding
	if bindAddress != "" {
		temps := strings.Split(bindAddress, ":")
		port, _ := strconv.Atoi(temps[1])
		con.Port = uint(port)
	}
	if err != nil {
		panic(err)
	}
	if ok {
		EnvConfig = con
	}
}

func UrlConsole() string {
	if EnvConfig.TLS {
		return fmt.Sprintf("%s://%s:%d", "https",
			EnvConfig.ConsoleConfig.Host,
			EnvConfig.ConsoleConfig.Port)
	} else {
		return fmt.Sprintf("%s://%s:%d", "http",
			EnvConfig.ConsoleConfig.Host,
			EnvConfig.ConsoleConfig.Port)
	}
}

func GetHostInfo() *model.Host {
	if hostInfo != nil {
		return hostInfo
	}
	hostInfo = getHostInfoFromKV()
	return hostInfo
}

func SetHostInfo(host *model.Host) error {
	if host == nil {
		return errors.New("host info can not be nil")
	}
	hostInfo = host
	hostByte, _ := json.Marshal(host)
	return kv.AddValue(KVAgentBucket, []byte(KVHostInfo), hostByte)
}

func DeleteHostInfo() error {
	return kv.DeleteKey(KVAgentBucket, []byte(KVHostInfo))
}

func ReloadHostInfo() {
	hostInfo = getHostInfoFromKV()
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

/**
创建elastic.API。

host： client最终发起请求时使用的ip地址

注意，和elastic.ElasticModule{}的逻辑稍有不同。
agent这边每个client实际需要操作的是具体的节点，不是集群。 因此传入esNodeId作为唯一标识。

取client的时候，也传入esNodeId即可
*/
func InitOrGetElasticClient(esNodeId string, userName string, password string, esVersion string, host string) (elastic.API, error) {
	client := elastic.GetClientNoPanic(esNodeId)
	if client != nil {
		return client, nil
	}

	var (
		ver string
	)

	if esNodeId == "" || host == "" {
		return nil, errors.New("InitOrGetElasticClient: params should not be empty")
	}

	if ver == "" && esVersion == "" {
		err := errors.New("no es version info")
		return nil, err
	}

	if strings.HasPrefix(ver, "8.") {
		api := new(adapter.ESAPIV8)
		api.Elasticsearch = esNodeId
		api.Version = ver
		client = api
	} else if strings.HasPrefix(ver, "7.") {
		api := new(adapter.ESAPIV7)
		api.Elasticsearch = esNodeId
		api.Version = ver
		client = api
	} else if strings.HasPrefix(ver, "6.") {
		api := new(adapter.ESAPIV6)
		api.Elasticsearch = esNodeId
		api.Version = ver
		client = api
	} else if strings.HasPrefix(ver, "5.") {
		api := new(adapter.ESAPIV5)
		api.Elasticsearch = esNodeId
		api.Version = ver
		client = api
	} else if strings.HasPrefix(ver, "2.") {
		api := new(adapter.ESAPIV2)
		api.Elasticsearch = esNodeId
		api.Version = ver
		client = api
	} else {
		api := new(adapter.ESAPIV0)
		api.Elasticsearch = esNodeId
		api.Version = ver
		client = api
	}

	elasticSearchConfig := &elastic.ElasticsearchConfig{BasicAuth: &struct {
		Username string `json:"username,omitempty" config:"username" elastic_mapping:"username:{type:keyword}"`
		Password string `json:"password,omitempty" config:"password" elastic_mapping:"password:{type:keyword}"`
	}{
		Username: userName,
		Password: password,
	}}
	elasticSearchConfig.ID = esNodeId
	elasticSearchConfig.Name = esNodeId
	elasticSearchConfig.Enabled = true
	elasticSearchConfig.Monitored = false
	elasticSearchConfig.Endpoint = host
	//TODO 这部分metadata的逻辑先关掉
	elasticSearchConfig.MetadataConfigs = &elastic.MetadataConfig{
		HealthCheck:           elastic.TaskConfig{Enabled: false},
		ClusterSettingsCheck:  elastic.TaskConfig{Enabled: false},
		MetadataRefresh:       elastic.TaskConfig{Enabled: false},
		NodeAvailabilityCheck: elastic.TaskConfig{Enabled: false},
	}
	elastic.RegisterInstance(*elasticSearchConfig, client)
	elastic.GetOrInitHost(host, esNodeId)
	elastic.InitMetadata(elasticSearchConfig, true)
	return client, nil
}
