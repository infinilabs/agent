/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package plugin aggregates the agent-side plugin packages so that their
// init() side effects (pipeline processor registrations, module hooks) run
// when the agent boots. main.go blank-imports this package.
package plugin

import (
	_ "infini.sh/agent/plugin/elastic/logging"
	_ "infini.sh/agent/plugin/elastic/metric"
	_ "infini.sh/agent/plugin/logs"

	// kafka queue backend: enables routing the "logs" queue to Kafka
	// purely via configuration (kafka_queue.default: true)
	_ "infini.sh/framework/plugins/queue/kafka_queue"
)
