package metrics

import (
	"infini.sh/agent/metrics/host/network"
	. "infini.sh/framework/core/config"
	"infini.sh/framework/core/task"
	log "github.com/cihub/seelog"
)

type MetricsModule struct {
}

func (this *MetricsModule) Name() string {
	return "metrics"
}

func (module *MetricsModule) Setup(cfg *Config) {

}

func (module *MetricsModule) Start() error {
	net,_:=network.New()

	var task1= task.ScheduleTask{
		Description: "fetch network metrics",
		Type:        "schedule",
		Interval:    "10s",
		Task: func() {
			log.Debug("collecting network metrics")
			net.Collect()
		},
	}
	task.RegisterScheduleTask(task1)

	return nil
}

func (module *MetricsModule) Stop() error {

	return nil
}
