package config

import "infini.sh/agent/model"

type AppConfig struct {
	Version       float32             `config:"version"`
	Schema        string              `config:"schema"`
	Port          uint                `config:"port"`
	ConsoleConfig model.ConsoleConfig `config:"console"`
}

var EnvConfig AppConfig

const (
	KVAgentID            string = "agent_client_id"
	KVAgentBucket               = "agent_bucket"
	ESClusterDefaultName        = "elasticsearch"
	ESConfigFileName            = "elasticsearch.yml"
)
