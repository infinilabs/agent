/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package metrics

import (
	"context"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/plugin/manage/instance"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
)

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
			if orm.Create(usage) != nil {
				log.Debugf("collect usage metrics, orm failed, %v", err)
			}
		},
	}
	task.RegisterScheduleTask(task1)
}
