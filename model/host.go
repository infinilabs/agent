/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package model

import (
	"fmt"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/util"
	"log"
	"strings"
)

type Host struct {
	IPs       []string   `json:"ip,omitempty"`
	TLS       bool       `json:"tls" yaml:"tls"`
	AgentPort uint       `json:"agent_port" yaml:"agent_port"`
	AgentID   string     `json:"agent_id" yaml:"agent_id"`
	Clusters  []*Cluster `json:"clusters" yaml:"clusters"`
}

type Cluster struct {
	ID        string  `json:"cluster.id" yaml:"cluster.id"`
	Name      string  `json:"cluster.name,omitempty" yaml:"cluster.name"`
	UUID      string  `json:"cluster.uuid,omitempty" yaml:"cluster.uuid"`
	UserName  string  `json:"username,omitempty" yaml:"username"`
	Password  string  `json:"password,omitempty" yaml:"password"`
	Nodes     []*Node `json:"nodes" yaml:"nodes"`
	Version   string  `json:"version" yaml:"version"`
	TLS       bool    `json:"tls" yaml:"tls"`
	TaskOwner bool    `json:"task_owner" yaml:"task_owner"`
}

type Node struct {
	ID                string `json:"id" yaml:"id"` //节点在es中的id
	Name              string `json:"name" yaml:"name"`
	ClusterName       string `json:"cluster.name" yaml:"cluster.name,omitempty"`
	HttpPort          int    `json:"http.port,omitempty" yaml:"http.port,omitempty"`
	LogPath           string `json:"path.logs" yaml:"path.logs,omitempty"`       //解析elasticsearch.yml
	NetWorkHost       string `json:"network.host" yaml:"network.host,omitempty"` //解析elasticsearch.yml
	TaskOwner         bool   `json:"task_owner" yaml:"task_owner"`               //console是否指派当前节点来获取集群数据
	ConfigPath        string `json:"config_path" yaml:"-"`
	ConfigFileContent []byte `json:"config_file_content"` //把配置文件的内容整个存储，用来判断配置文件内容是否变更
	Ports             []int  `json:"-" yaml:"-"`          //之所以是数组，因为从进程信息中获取到端口会有多个(通常为2个)，需要二次验证。这个字段只做缓存
}

type ConsoleConfig struct {
	Name string `json:"name" config:"name"`
	Host string `json:"host" config:"host"`
	Port int    `json:"port" config:"port"`
	TLS  bool   `json:"tls" config:"tls"`
}

type RegisterResponse struct {
	AgentId  string                 `json:"_id"`
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

type UpNodeInfoResponse struct {
	AgentId string `json:"_id"`
	Result  string `json:"result"`
}

type HeartBeatResp struct {
	AgentId   string                 `json:"agent_id"`
	Result    string                 `json:"result"`
	Timestamp int64                  `json:"timestamp"`
	TaskState map[string]interface{} `json:"task_state"`
}

func (u *UpNodeInfoResponse) IsSuccessed() bool {
	if strings.EqualFold(u.Result, "updated") {
		return true
	}
	return false
}

func (n *Node) GetNetWorkHost(schema string) string {
	if schema == "" {
		schema = "http"
	}
	if n.NetWorkHost == "" {
		return fmt.Sprintf("%s://localhost:%d", schema, n.HttpPort)
	}
	return fmt.Sprintf("%s://%s:%d", schema, n.NetWorkHost, n.HttpPort)
}

func (c *Cluster) ConvertToESCluster() *agent.ESCluster {
	esc := &agent.ESCluster{}
	esc.BasicAuth = &agent.BasicAuth{}
	esc.ClusterID = c.ID
	esc.ClusterUUID = c.UUID
	esc.ClusterName = c.Name
	esc.BasicAuth.Username = c.UserName
	esc.BasicAuth.Password = c.Password
	for _, node := range c.Nodes {
		esc.Nodes = append(esc.Nodes,
			agent.ESNode{
				UUID: node.ID,
				Name: node.Name,
			})
	}
	return esc
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

func (h *Host) GetSchema() string {
	if h.TLS {
		return "https"
	} else {
		return "http"
	}
}

func (c *Cluster) GetSchema() string {
	if c.TLS {
		return "https"
	} else {
		return "http"
	}
}

func (c *Cluster) GetTaskOwnerNode() *Node {
	for _, node := range c.Nodes {
		if node.TaskOwner {
			return node
		}
	}
	return nil
}

func (n *Node) IsAlive(schema string, userName string, password string, esVersion string) bool {
	url := fmt.Sprintf("%s://%s:%d/_nodes/_local", schema, n.NetWorkHost, n.HttpPort)
	var req = util.NewGetRequest(url, nil)
	if userName != "" && password != "" {
		req.SetBasicAuth(userName, password)
	}
	result, err := util.ExecuteRequest(req)
	if err != nil {
		log.Printf("%v", err)
		return false
	}

	//判断用户名密码是否正确
	resultMap := make(map[string]interface{})
	util.MustFromJSONBytes(result.Body, &resultMap)
	//有错误，则认为是无法正常访问es了 => 更新host信息
	if _, ok := resultMap["error"]; ok {
		return false
	}

	nodesInfo := map[string]interface{}{}
	util.MustFromJSONBytes(result.Body, &nodesInfo)
	//这里虽然是遍历，但实际返回的只有当前节点的信息
	if nodes, ok := nodesInfo["nodes"]; ok {
		if nodesMap, ok := nodes.(map[string]interface{}); ok {
			for id, v := range nodesMap {
				resultMap[id] = id
				if nodeInfo, ok := v.(map[string]interface{}); ok {
					resultMap[nodeInfo["name"].(string)] = nodeInfo["name"].(string)
					resultMap["version"] = nodeInfo["version"].(string)
				}
			}
		}
	}
	//接下来的3个判断，实际是比较极端的情况： 配置文件没变，但es实例已经不是之前的那个实例了。
	if _, ok := resultMap[n.ID]; !ok {
		return false
	}
	if _, ok := resultMap[n.Name]; !ok {
		return false
	}
	if versionStr, ok := resultMap["version"]; ok {
		if !strings.EqualFold(esVersion, versionStr.(string)) {
			return false
		}
	}
	return true
}
