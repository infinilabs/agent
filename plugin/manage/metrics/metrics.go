/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package metrics

import (
	"context"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/plugin/manage/instance"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

//
//func Init() {
//	meta := event.AgentMeta{
//		MajorIP:   publicIP,
//		Hostname:  util.GetHostName(),
//		IP:        util.GetLocalIPs(),
//		QueueName: util.StringDefault(module.config.Queue, "metrics"),
//		Labels:    module.config.Labels,
//		Tags:      module.config.Tags}
//	event.RegisterMeta(&meta)
//}

func Collect() {

	var task1 = task.ScheduleTask{
		Description: "collect host usage metrics",
		Type:        "interval",
		Interval:    "10s",
		Task: func(ctx context.Context) {
			if !config.GetInstanceInfo().IsRunning {
				return
			}
			usage, err := instance.GetAllUsageInfo()
			usage.AgentID = config.GetInstanceInfo().AgentID
			if err != nil {
				log.Debugf("collect usage metrics failed, %v", err)
				return
			}

			item := event.Event{
				Metadata: event.EventMetadata{
					Category: "elasticsearch",
					Name:     "host_usages",
					Datatype: "snapshot",
					Labels: util.MapStr{
						"agent_id": usage.AgentID,
					},
				},
			}
			item.Fields = util.MapStr{
				"elasticsearch": util.MapStr{
					"host_usage": usage,
				},
			}
			err = event.Save(item)
			if err != nil {
				log.Errorf("metrics.Collect: %v\n", err)
			}
		},
	}
	task.RegisterScheduleTask(task1)
}
