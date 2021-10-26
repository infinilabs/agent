/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package common

import (
	"fmt"
	"infini.sh/framework/core/util"
	"time"
)

type Event struct {
	Timestamp    time.Time         `json:"timestamp"`
	Tags         []string          `json:"tags"`
	Labels       map[string]string `json:"labels"`
	MetricFields util.MapStr       `json:"metric"`
}

func (e *Event)String()string  {
	return fmt.Sprintf("%v-%v,%v,%v",e.Timestamp.UTC().Unix(),e.MetricFields,e.Tags,e.Labels)
}
