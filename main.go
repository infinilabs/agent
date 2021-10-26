/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package main

import (
	_ "expvar"
	"infini.sh/agent/config"
	"infini.sh/agent/metrics"
	"infini.sh/framework"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/api"
	"infini.sh/framework/modules/elastic"
	"infini.sh/framework/modules/filter"
	"infini.sh/framework/modules/pipeline"
	"infini.sh/framework/modules/queue"
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
		util.TrimSpaces(config.Version), util.TrimSpaces(config.LastCommitLog), util.TrimSpaces(config.BuildDate), util.TrimSpaces(config.EOLDate), terminalHeader, terminalFooter)

	app.Init(nil)

	defer app.Shutdown()

	app.Start(func() {

		//load core modules first
		module.RegisterSystemModule(elastic.ElasticModule{})
		module.RegisterSystemModule(filter.FilterModule{})
		module.RegisterSystemModule(&queue.DiskQueue{})
		module.RegisterSystemModule(&api.APIModule{})
		module.RegisterSystemModule(&pipeline.PipeModule{})
		module.RegisterSystemModule(&task.TaskModule{})
		module.RegisterUserPlugin(&stats.StatsDModule{})
		module.RegisterUserPlugin(&metrics.MetricsModule{})

		//start each module, with enabled provider
		module.Start()

	}, func() {
	})

}
