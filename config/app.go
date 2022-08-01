package config

type AppConfig struct {
}

type ConsoleConfig struct {
	Name string `config:"name"`
	IP   string `config:"ip"`
	Port int    `config:"port"`
	//xxxx
}
