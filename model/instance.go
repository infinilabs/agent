/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package model

import (
	"errors"
	"infini.sh/framework/core/model"
	"strings"
)

type Instance struct {
	IPs       []string           `json:"ip,omitempty"`
	MajorIP   string             `json:"major_ip,omitempty"`
	TLS       bool               `json:"tls" yaml:"tls"`
	AgentPort uint               `json:"agent_port" yaml:"agent_port"`
	AgentID   string             `json:"agent_id" yaml:"agent_id"`
	HostID    string             `json:"host_id"`
	Nodes     []model.ESNodeInfo `json:"nodes" yaml:"nodes"`
	Host      model.HostInfo     `json:"host"`
	IsRunning bool               `json:"is_running"`
	BootTime  int64              `json:"boot_time"`
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
	Instance model.Instance `json:"_source"`
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
