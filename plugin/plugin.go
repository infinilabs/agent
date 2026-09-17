/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package plugin aggregates the agent-side plugin packages so that their
// init() side effects (pipeline processor registrations, module hooks) run
// when the agent boots. main.go blank-imports this package.
//
// Enterprise plugins are deliberately NOT imported here: make update-plugins
// generates plugin/generated_plugins.go from the plugin folders present in
// the build tree, so private checkouts are picked up automatically when
// they exist and public builds stay self-contained.
package plugin

import (
	_ "infini.sh/agent/plugin/elastic/logging"
	_ "infini.sh/agent/plugin/elastic/metric"
	_ "infini.sh/agent/plugin/logs"

	// kafka queue backend: enables routing the "logs" queue to Kafka
	// purely via configuration (kafka_queue.default: true)
	_ "infini.sh/framework/plugins/queue/kafka_queue"
)
