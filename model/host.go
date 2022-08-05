/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package model

import (
	"infini.sh/framework/core/agent"
)

type Host struct {
	IPs       []string   `json:"ip,omitempty"`
	TLS       bool       `json:"tls"`
	AgentPort uint       `json:"agent_port"`
	AgentID   string     `json:"agent_id" yaml:"agent_id"`
	Clusters  []*Cluster `json:"clusters" yaml:"clusters"`
}

func (h *Host) ToAgentInstance() *agent.Instance {
	instance := agent.Instance{
		ID:   h.AgentID,
		Port: h.AgentPort,
		IPS:  h.IPs,
	}
	if h.TLS {
		instance.Schema = "https"
	} else {
		instance.Schema = "http"
	}
	for _, cluster := range h.Clusters {
		instance.Clusters = append(instance.Clusters, *cluster.ConvertToESCluster())
	}
	return &instance
}

type Cluster struct {
	ID       string  `json:"cluster.id" yaml:"cluster.id"`
	Name     string  `json:"cluster.name,omitempty" yaml:"cluster.name"`
	UUID     string  `json:"cluster.uuid,omitempty" yaml:"cluster.uuid"`
	UserName string  `json:"username,omitempty" yaml:"username"`
	Password string  `json:"password,omitempty" yaml:"password"`
	Nodes    []*Node `json:"nodes" yaml:"nodes"`
}

type Node struct {
	ID          string `json:"id" yaml:"id"`
	ClusterName string `json:"-" yaml:"cluster.name,omitempty"`
	HttpPort    int    `json:"http.port,omitempty" yaml:"http.port,omitempty"`
	LogPath     string `json:"-" yaml:"path.logs,omitempty"`    //解析elasticsearch.yml
	NetWorkHost string `json:"-" yaml:"network.host,omitempty"` //解析elasticsearch.yml
	ConfigPath  string `json:"-" yaml:"-"`
	Ports       []int  `json:"-" yaml:"-"` //之所以是数组，因为从进程信息中获取到端口会有多个(通常为2个)，需要二次验证。这个字段只做缓存
}

type ConsoleConfig struct {
	Name string `json:"name" config:"name"`
	Host string `json:"host" config:"host"`
	Port int    `json:"port" config:"port"`
	TLS  bool   `json:"tls" config:"tls"`
}

type RegisterResponse struct {
	AgentId  string                 `json:"agent_id"`
	Clusters map[string]ClusterResp `json:"clusters"`
}

type ClusterResp struct {
	ClusterId   string        `json:"cluster_id"`
	ClusterUUID string        `json:"cluster_uuid"`
	BasicAuth   BasicAuthResp `json:"basic_auth"`
}

type BasicAuthResp struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

func (cluster *Cluster) ConvertToESCluster() *agent.ESCluster {
	esc := &agent.ESCluster{}
	esc.ClusterID = cluster.ID
	esc.ClusterUUID = cluster.UUID
	esc.ClusterName = cluster.Name
	esc.BasicAuth.Username = cluster.UserName
	esc.BasicAuth.Password = cluster.Password
	for _, node := range cluster.Nodes {
		esc.Nodes = append(esc.Nodes, node.ID)
	}
	return esc
}
