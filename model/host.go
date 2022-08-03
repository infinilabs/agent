/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package model

type Host struct {
	IPs       []string   `json:"ip,omitempty"`
	TLS       bool       `json:"tls"`
	AgentPort int        `json:"agent_port"`
	AgentID   string     `json:"agent_id" yaml:"agent_id"`
	Clusters  []*Cluster `json:"clusters" yaml:"clusters"`
}

type Cluster struct {
	Name     string  `json:"cluster.name,omitempty" yaml:"cluster.name"`
	UUID     string  `json:"cluster.uuid,omitempty" yaml:"cluster.uuid"`
	UserName string  `json:"username,omitempty" yaml:"username"`
	Password string  `json:"password,omitempty" yaml:"password"`
	Nodes    []*Node `json:"nodes" yaml:"nodes"`
}

type Node struct {
	ID         string `json:"id" yaml:"id"`
	HttpPort   int    `json:"http.port,omitempty" yaml:"http.port,omitempty"`
	ConfigPath string `json:"-" yaml:"-"`
	Ports      []int  `json:"-" yaml:"-"` //之所以是数组，因为从进程信息中获取到端口会有多个(通常为2个)，需要二次验证。这个字段只做缓存
}

type ConsoleConfig struct {
	Name string `json:"name" yaml:"name"`
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
	TLS  bool   `json:"tls" yaml:"tls"`
}
