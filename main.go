/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package main

import (
	_ "expvar"
	"infini.sh/agent/config"
	_ "infini.sh/agent/plugin/diagnostics"
	"infini.sh/framework"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/api"
	"infini.sh/framework/modules/elastic"
	"infini.sh/framework/modules/filter"
	"infini.sh/framework/modules/metrics"
	"infini.sh/framework/modules/pipeline"
	queue2 "infini.sh/framework/modules/queue/disk_queue"
	stats2 "infini.sh/framework/modules/stats"
	"infini.sh/framework/modules/task"
	stats "infini.sh/framework/plugins/stats_statsd"
)

func main() {

	terminalHeader := ("   _      ___   __    __  _____ \n")
	terminalHeader += ("  /_\\    / _ \\ /__\\/\\ \\ \\/__   \\\n")
	terminalHeader += (" //_\\\\  / /_\\//_\\ /  \\/ /  / /\\/\n")
	terminalHeader += ("/  _  \\/ /_\\\\//__/ /\\  /  / /   \n")
	terminalHeader += ("\\_/ \\_/\\____/\\__/\\_\\ \\/   \\/    \n\n")

	terminalFooter := ""

	app := framework.NewApp("agent", "A light-weight, powerful and high-performance elasticsearch agent.",
		util.TrimSpaces(config.Version), util.TrimSpaces(config.BuildNumber), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.Init(nil)

	defer app.Shutdown()

	if app.Setup(func() {

		//load core modules first
		module.RegisterSystemModule(&stats2.SimpleStatsModule{})
		module.RegisterSystemModule(&elastic.ElasticModule{})
		module.RegisterSystemModule(&filter.FilterModule{})
		module.RegisterSystemModule(&queue2.DiskQueue{})
		module.RegisterSystemModule(&api.APIModule{})
		module.RegisterSystemModule(&pipeline.PipeModule{})
		module.RegisterSystemModule(&task.TaskModule{})
		module.RegisterUserPlugin(&stats.StatsDModule{})

		module.RegisterUserPlugin(&metrics.MetricsModule{})
		//module.RegisterSystemModule(&diagnostics.DiagnosticsAnalysisModule{})

		//pipe.RegisterProcessorPlugin("json_indexing", json_indexing.New)

		//api1 := api2.AgentAPI{}
		//api1.Init()

		//con := config.AppConfig{}
		//ok, err := env.ParseConfig("console", con)
		//if err != nil {
		//	panic(err)
		//}
		//if ok {
		//	config.EnvConfig = con
		//}
		//manage.Init()
	}, func() {

		//start each module, with enabled provider
		module.Start()
	}, nil) {
		app.Run()
		//host.Init()
	}

}
