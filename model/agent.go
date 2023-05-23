/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package model

import "infini.sh/framework/core/agent"

type AgentInfo struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Version interface{} `json:"version"`
	IPs       []string       `json:"ips,omitempty"`
	MajorIP   string         `json:"major_ip,omitempty"`
	Host      *agent.HostInfo `json:"host"`
}
