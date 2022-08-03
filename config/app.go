package config

import "infini.sh/agent/model"

type AppConfig struct {
	model.ConsoleConfig
}

var EnvConfig AppConfig

const (
	KVAgentID     string = "agent_client_id"
	KVAgentBucket string = "agent_bucket"
)
