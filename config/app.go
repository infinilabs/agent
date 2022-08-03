package config

type AppConfig struct {
}

type ConsoleConfig struct {
	Name string `config:"name"`
	IP   string `config:"ip"`
	Port int    `config:"port"`
	//xxxx
}

const (
	KVAgentID     string = "agent_client_id"
	KVAgentBucket string = "agent_bucket"
)
