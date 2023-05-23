/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package model

import (
	"errors"
	"infini.sh/framework/core/agent"
	"strings"
)

type Instance struct {
	IPs       []string       `json:"ip,omitempty"`
	MajorIP   string         `json:"major_ip,omitempty"`
	TLS       bool           `json:"tls" yaml:"tls"`
	AgentPort uint           `json:"agent_port" yaml:"agent_port"`
	AgentID   string         `json:"agent_id" yaml:"agent_id"`
	HostID    string         `json:"host_id"`
	Nodes  []agent.ESNodeInfo     `json:"nodes" yaml:"nodes"`
	Host      agent.HostInfo `json:"host"`
	IsRunning bool           `json:"is_running"`
	BootTime  int64          `json:"boot_time"`
}

const (
	NodeStatusOnline = "online"
	NodeStatusOffline = "offline"
)

//type Node struct {
//	ID                string `json:"id" yaml:"id"` //节点在es中的id
//	Name              string `json:"node.name" yaml:"node.name"`
//	ClusterName       string `json:"cluster.name" yaml:"cluster.name,omitempty"`
//	HttpPort          int    `json:"http.port,omitempty" yaml:"http.port,omitempty"`
//	LogPath           string `json:"path.logs" yaml:"path.logs,omitempty"`       //解析elasticsearch.yml
//	NetWorkHost       string `json:"network.host" yaml:"network.host,omitempty"` //解析elasticsearch.yml
//	ESHomePath        string `json:"es_home_path"`
//	ConfigPath        string `json:"config_path" yaml:"-"`
//	ConfigFileContent []byte `json:"config_file_content"` //把配置文件的内容整个存储，用来判断配置文件内容是否变更
//	Ports             []int  `json:"-" yaml:"-"`          //之所以是数组，因为从进程信息中获取到端口会有多个(通常为2个)，需要二次验证。这个字段只做缓存
//	PID               int32  `json:"pid"`                 //es节点的进程id
//	Status            string `json:"status"`
//	SSL               SSL    `json:"ssl" yaml:"xpack.security.http.ssl,omitempty"` //解析elasticsearch.yml
//	IsSSL             bool   `json:"is_ssl" yaml:"xpack.security.http.ssl.enabled"`
//}
//
//type SSL struct {
//	Enabled bool `json:"enabled" yaml:"enabled"`
//	Path string `json:"keystore.path" yaml:"keystore.path"`
//}

type RegisterResponse struct {
	AgentId  string                 `json:"_id"`
	Clusters map[string]ClusterResp `json:"clusters"`
	Result   string                 `json:"result"`
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

type UpNodeInfoResponse struct {
	AgentId string                 `json:"_id"`
	Result  string                 `json:"result"`
	Cluster map[string]interface{} `json:"clusters"`
}

type GetAgentInfoResponse struct {
	AgentId  string         `json:"_id"`
	Found    bool           `json:"found"`
	Instance agent.Instance `json:"_source"`
}

type ReadLogRequest struct {
	NodeId   string `json:"node_id"`
	FileName string `json:"file_name"`
	Offset   int64    `json:"offset"`
	Lines    int    `json:"lines"`
}

func (r *ReadLogRequest) ValidateParams() error {
	if r.NodeId == "" {
		return errors.New("error params: node id")
	}
	if r.FileName == "" {
		return errors.New("error params: file name")
	}
	if !strings.HasSuffix(r.FileName,".json") && !strings.HasSuffix(r.FileName,".log") {
		return errors.New("error params: file name")
	}
	if r.Lines <= 0 {
		r.Lines = 10
	}
	return nil
}

func (u *UpNodeInfoResponse) IsSuccessed() bool {
	if strings.EqualFold(u.Result, "updated") {
		return true
	}
	return false
}
