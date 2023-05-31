/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package main

import (
	_ "expvar"
	api2 "infini.sh/agent/api"
	"infini.sh/agent/config"
	_ "infini.sh/agent/plugin"
	api3 "infini.sh/agent/plugin/api"
	"infini.sh/agent/plugin/manage"
	"infini.sh/framework"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/api"
	"infini.sh/framework/modules/elastic"
	"infini.sh/framework/modules/keystore"
	"infini.sh/framework/modules/metrics"
	"infini.sh/framework/modules/pipeline"
	queue2 "infini.sh/framework/modules/queue/disk_queue"
	stats2 "infini.sh/framework/modules/stats"
	"infini.sh/framework/modules/task"
	_ "infini.sh/framework/plugins"
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
		module.RegisterSystemModule(&elastic.ElasticModule{})
		module.RegisterSystemModule(&stats2.SimpleStatsModule{})
		module.RegisterSystemModule(&queue2.DiskQueue{})

		module.RegisterSystemModule(&api.APIModule{})
		module.RegisterSystemModule(&pipeline.PipeModule{})
		module.RegisterSystemModule(&task.TaskModule{})

		module.RegisterUserPlugin(&metrics.MetricsModule{})
		module.RegisterUserPlugin(&keystore.KeystoreModule{})
		config.InitConfig()

		api1 := api2.AgentAPI{}
		api1.Init()
		api3.InitAPI()
	}, func() {

		//start each module, with enabled provider
		module.Start()
		config.ReloadHostInfo()
		if config.GetManagerEndpoint() != "" {
			manage.Init()
		}
	}, nil) {
		app.Run()
	}

}
