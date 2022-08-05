package config

import (
	"encoding/json"
	"fmt"
	"infini.sh/agent/model"
	"infini.sh/framework/core/kv"
	"log"
)

type AppConfig struct {
	Version       float32             `config:"version"`
	TLS           bool                `config:"tls"`
	Port          uint                `config:"port"`
	ConsoleConfig model.ConsoleConfig `config:"console"`
}

var EnvConfig AppConfig

const (
	KVHostInfo           string = "agent_host_info"
	KVAgentBucket               = "agent_bucket"
	ESClusterDefaultName        = "elasticsearch"
	ESConfigFileName            = "elasticsearch.yml"
)

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

var host *model.Host

func GetHostInfoFromKV() *model.Host {
	hs, err := kv.GetValue(KVAgentBucket, []byte(KVHostInfo))
	if err != nil {
		log.Println(err)
		return nil
	}
	err = json.Unmarshal(hs, &host)
	if err != nil {
		log.Println(err)
		return nil
	}
	return host
}
