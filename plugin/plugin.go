/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package plugin aggregates the agent-side plugin packages so that their
// init() side effects (pipeline processor registrations, module hooks) run
// when the agent boots. main.go blank-imports this package.
//
// Note: the enterprise imports below require the closed-source plugin
// checkouts to be present in the build tree (see CI "Prepare work layout").
package plugin

import (
	_ "infini.sh/agent/plugin/elastic/logging"
	_ "infini.sh/agent/plugin/elastic/metric"
	_ "infini.sh/agent/plugin/enterprise"
	_ "infini.sh/agent/plugin/logs"

	// enterprise data-processing processors (dissect, field_standardize,
	// ...) — lives in framework/plugins/enterprise/processors (private repo).
	_ "infini.sh/framework/plugins/enterprise/processors"
	// OTLP/gRPC egress processor: ships processed log batches to the
	// gateway's OTLP intake (:4317).
	_ "infini.sh/framework/plugins/otlp/export"
)
