/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package common

import (
	"fmt"
	"infini.sh/framework/core/util"
	"time"
)

type Event struct {
	Agent        Agent       `json:"agent"`
	Timestamp    time.Time   `json:"timestamp,omitempty" elastic_mapping:"timestamp: { type: date }"`
	MetricFields util.MapStr `json:"metric"`
}

func (e *Event) String() string {
	return fmt.Sprintf("%v-%v,%v,%v", e.Timestamp.UTC().Unix(), e.MetricFields, e.Agent.Tags, e.Agent.Labels)
}

type Agent struct {
	AgentID  string   `json:"id"`
	Hostname string   `json:"hostname"`
	MajorIP  string   `json:"ip"`
	IP       []string `json:"binding_ip"`

	Tags   []string          `json:"tags"`
	Labels map[string]string `json:"labels"`
}
