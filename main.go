/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package main

import (
	"context"
	_ "expvar"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	_ "infini.sh/agent/plugin"
	api3 "infini.sh/agent/plugin/api"
	"infini.sh/framework"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	task2 "infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/api"
	"infini.sh/framework/modules/elastic"
	"infini.sh/framework/modules/keystore"
	"infini.sh/framework/modules/metrics"
	"infini.sh/framework/modules/pipeline"
	queue2 "infini.sh/framework/modules/queue/disk_queue"
	stats2 "infini.sh/framework/modules/stats"
	"infini.sh/framework/modules/task"
	_ "infini.sh/framework/plugins/elastic/bulk_indexing"
	_ "infini.sh/framework/plugins/elastic/indexing_merge"
	_ "infini.sh/framework/plugins/http"
	_ "infini.sh/framework/plugins/queue/consumer"
	"infini.sh/framework/plugins/simple_kv"
	"os"
	"runtime"
)

func main() {

	terminalHeader := ("   _      ___   __    __  _____ \n")
	terminalHeader += ("  /_\\    / _ \\ /__\\/\\ \\ \\/__   \\\n")
	terminalHeader += (" //_\\\\  / /_\\//_\\ /  \\/ /  / /\\/\n")
	terminalHeader += ("/  _  \\/ /_\\\\//__/ /\\  /  / /   \n")
	terminalHeader += ("\\_/ \\_/\\____/\\__/\\_\\ \\/   \\/    \n\n")

	terminalFooter := ""

	app := framework.NewApp("agent", "A light-weight but powerful cloud agent.",
		config.Version, config.BuildNumber, config.LastCommitLog, config.BuildDate, config.EOLDate, terminalHeader, terminalFooter)

	app.Init(nil)

	defer app.Shutdown()

	if app.Setup(func() {

		//load core modules first
		module.RegisterSystemModule(&elastic.ElasticModule{})
		module.RegisterSystemModule(&stats2.SimpleStatsModule{})
		module.RegisterSystemModule(&simple_kv.SimpleKV{})
		module.RegisterSystemModule(&queue2.DiskQueue{})

		module.RegisterSystemModule(&api.APIModule{})
		module.RegisterSystemModule(&pipeline.PipeModule{})
		module.RegisterSystemModule(&task.TaskModule{})

		module.RegisterUserPlugin(&metrics.MetricsModule{})
		module.RegisterUserPlugin(&keystore.KeystoreModule{})

		api3.InitAPI()
	}, func() {
		defer func() {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				log.Errorf("error on start module [%v]", v)
				log.Flush()
				os.Exit(1)
			}
		}()
		//start each module, with enabled provider
		module.Start()
		if global.Env().SystemConfig.Configs.AllowGeneratedMetricsTasks {
			taskID := util.GetUUID()
			task2.RegisterScheduleTask(task2.ScheduleTask{
				ID:          taskID,
				Description: "generated metrics tasks for agent",
				Type:        "interval",
				Interval:    "20s",
				Task: func(ctx context.Context) {
					err := generatedMetricsTasksConfig()
					if err != nil {
						log.Error("error generating metrics tasks config: ", err)
						return
					}
					//clean up task after success
					task2.DeleteTask(taskID)
				},
			})
		}
		//if agent is mark as deleted, cleanup local configs

	}, nil) {
		app.Run()
	}

}
