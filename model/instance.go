/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package model

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/util"
	"strings"
	"time"
)

type Instance struct {
	IPs       []string       `json:"ip,omitempty"`
	MajorIP   string         `json:"major_ip,omitempty"`
	TLS       bool           `json:"tls" yaml:"tls"`
	AgentPort uint           `json:"agent_port" yaml:"agent_port"`
	AgentID   string         `json:"agent_id" yaml:"agent_id"`
	HostID    string         `json:"host_id"`
	Clusters  []*Cluster     `json:"clusters" yaml:"clusters"`
	Host      agent.HostInfo `json:"host"`
	IsRunning bool           `json:"is_running"`
	BootTime  int64          `json:"boot_time"`
}

type Cluster struct {
	ID       string  `json:"cluster.id" yaml:"cluster.id"`
	Name     string  `json:"cluster.name,omitempty" yaml:"cluster.name"`
	UUID     string  `json:"cluster.uuid,omitempty" yaml:"cluster.uuid"`
	UserName string  `json:"username,omitempty" yaml:"username"`
	Password string  `json:"password,omitempty" yaml:"password"`
	Nodes    []*Node `json:"nodes" yaml:"nodes"`
	Version  string  `json:"version" yaml:"version"`
	TLS      bool    `json:"tls" yaml:"tls"`
	Task     *Task   `json:"task"`
}

func (c *Cluster) GetEndPoint() string {
	if len(c.Nodes) > 0 {
		return c.Nodes[0].GetEndPoint(c.GetSchema())
	}
	return ""
}

func (c *Cluster) UpdateTask(task *agent.Task) {
	if c.Task == nil {
		return
	}
	empty1 := ClusterMetricTask{}
	empty3 := agent.ClusterMetricTask{}
	if c.Task.ClusterMetric != empty1 && task.ClusterMetric != empty3 {
		c.Task.ClusterMetric.Owner = task.ClusterMetric.Owner
		c.Task.ClusterMetric.TaskNodeID = task.ClusterMetric.TaskNodeID
	}
	if c.Task.NodeMetric != nil && task.NodeMetric != nil {
		c.Task.NodeMetric.ExtraNodes = task.NodeMetric.ExtraNodes
		c.Task.NodeMetric.Owner = task.NodeMetric.Owner
	}
}

type Task struct {
	ClusterMetric ClusterMetricTask `json:"cluster_metric,omitempty"`
	NodeMetric    *NodeMetricTask   `json:"node_metric,omitempty"`
}

type ClusterMetricTask struct {
	Owner      bool   `json:"owner"`
	TaskNodeID string `json:"task_node_id"`
}

type NodeMetricTask struct {
	Owner      bool     `json:"owner"`
	ExtraNodes []string `json:"extra_nodes,omitempty"`
}

func (c *Cluster) IsClusterTaskOwner() bool {
	return c.Task.ClusterMetric.Owner
}

type NodeStatus string
const Online = NodeStatus("online")
const Offline = NodeStatus("offline")

type Node struct {
	ID                string `json:"id" yaml:"id"` //节点在es中的id
	Name              string `json:"node.name" yaml:"node.name"`
	ClusterName       string `json:"cluster.name" yaml:"cluster.name,omitempty"`
	HttpPort          int    `json:"http.port,omitempty" yaml:"http.port,omitempty"`
	LogPath           string `json:"path.logs" yaml:"path.logs,omitempty"`       //解析elasticsearch.yml
	NetWorkHost       string `json:"network.host" yaml:"network.host,omitempty"` //解析elasticsearch.yml
	ESHomePath        string `json:"es_home_path"`
	ConfigPath        string `json:"config_path" yaml:"-"`
	ConfigFileContent []byte `json:"config_file_content"` //把配置文件的内容整个存储，用来判断配置文件内容是否变更
	Ports             []int  `json:"-" yaml:"-"`          //之所以是数组，因为从进程信息中获取到端口会有多个(通常为2个)，需要二次验证。这个字段只做缓存
	PID               int32  `json:"pid"`                 //es节点的进程id
	Status            NodeStatus `json:"status"`
}

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

type HeartBeatResp struct {
	AgentId   string                    `json:"agent_id"`
	Success   bool                      `json:"success"`
	Timestamp int64                     `json:"timestamp"`
	TaskState map[string]*HeartTaskResp `json:"task_state"`
}

type HeartTaskResp struct {
	ClusterMetric string `json:"cluster_metric"`
}

func (u *UpNodeInfoResponse) IsSuccessed() bool {
	if strings.EqualFold(u.Result, "updated") {
		return true
	}
	return false
}

func (n *Node) GetEndPoint(schema string) string {
	if schema == "" {
		schema = "http"
	}
	if n.NetWorkHost == "" {
		return fmt.Sprintf("%s://localhost:%d", schema, n.HttpPort)
	}
	return fmt.Sprintf("%s://%s:%d", schema, n.NetWorkHost, n.HttpPort)
}

func (c *Cluster) ToConsoleModel() *agent.ESCluster {
	esc := &agent.ESCluster{}
	esc.BasicAuth = &agent.BasicAuth{}
	esc.Task = agent.Task{
		ClusterMetric: agent.ClusterMetricTask{},
		NodeMetric:    &agent.NodeMetricTask{},
	}
	esc.ClusterID = c.ID
	esc.ClusterUUID = c.UUID
	esc.ClusterName = c.Name
	esc.BasicAuth.Username = c.UserName
	esc.BasicAuth.Password = c.Password
	esc.Task.ClusterMetric.TaskNodeID = c.Task.ClusterMetric.TaskNodeID
	esc.Task.ClusterMetric.Owner = c.Task.ClusterMetric.Owner
	esc.Task.NodeMetric.Owner = c.Task.NodeMetric.Owner
	esc.Task.NodeMetric.ExtraNodes = c.Task.NodeMetric.ExtraNodes
	for _, node := range c.Nodes {
		esc.Nodes = append(esc.Nodes,
			agent.ESNode{
				UUID: node.ID,
				Name: node.Name,
			})
	}
	return esc
}

func (h *Instance) ToConsoleModel() *agent.Instance {
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
		//TODO: 和console那边的node status 做对接
		instance.Clusters = append(instance.Clusters, *cluster.ToConsoleModel())
	}
	instance.Host = h.Host
	instance.MajorIP = h.MajorIP
	return &instance
}

func (h *Instance) GetSchema() string {
	if h.TLS {
		return "https"
	} else {
		return "http"
	}
}

func (h *Instance) GetUpTimeInSecond() int64 {
	return time.Now().Unix() - h.BootTime
}

func (c *Cluster) GetSchema() string {
	if c.TLS {
		return "https"
	} else {
		return "http"
	}
}

//获取执行集群指标任务的节点信息
func (c *Cluster) GetClusterTaskOwnerNode() *Node {
	for _, node := range c.Nodes {
		if node.ID == c.Task.ClusterMetric.TaskNodeID {
			return node
		}
	}
	return nil
}

func (c *Cluster) IsNeedCollectNodeMetric() bool {
	if c.Task != nil && c.Task.NodeMetric != nil {
		return c.Task.NodeMetric.Owner
	}
	return false
}

func (n *Node) IsAlive(schema string, userName string, password string, esVersion string) bool {
	url := fmt.Sprintf("%s://%s:%d/_nodes/_local", schema, n.NetWorkHost, n.HttpPort)
	var req = util.NewGetRequest(url, nil)
	if userName != "" && password != "" {
		req.SetBasicAuth(userName, password)
	}
	result, err := util.ExecuteRequest(req)
	if err != nil {
		log.Errorf("%v", err)
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

func (h *Instance) GetOnlineClusterOnCurrentHost() []*Cluster  {
	var retCluster []*Cluster
	for _, cluster := range h.Clusters {
		if len(cluster.GetOnlineNodes()) > 0 {
			retCluster = append(retCluster, cluster)
		}
	}
	return retCluster
}

func (c *Cluster) GetOnlineNodes() []*Node {
	var retNodes []*Node
	for _, node := range c.Nodes {
		if node.Status == Online {
			retNodes = append(retNodes, node)
		}
	}
	return retNodes
}