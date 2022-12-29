/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package cluster_health

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

const processorName = "es_cluster_health"

func init() {
	pipeline.RegisterProcessorPlugin(processorName, newProcessor)
}

func newProcessor(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{}
	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of %s processor: %s", processorName, err)
	}
	processor := ClusterHealth{
		config: &cfg,
	}
	return &processor, nil
}

type Config struct {
	Elasticsearch string `config:"elasticsearch,omitempty"`
}

type ClusterHealth struct {
	config *Config
}

func (p *ClusterHealth) Name() string {
	return processorName
}

func (p *ClusterHealth) Process(c *pipeline.Context) error {
	meta := elastic.GetMetadata(p.config.Elasticsearch)
	return p.Collect(p.config.Elasticsearch, meta)
}

func (p *ClusterHealth) Collect(k string, v *elastic.ElasticsearchMetadata) error {

	log.Trace("collecting custer health metrics for :", k)

	client := elastic.GetClientNoPanic(k)
	if client == nil {
		return nil
	}
	var health *elastic.ClusterHealth
	var err error
	health, err = client.ClusterHealthSpecEndpoint(v.Config.Endpoint)
	if err != nil {
		log.Error(v.Config.Name, " get cluster health error: ", err)
		return err
	}

	labels := util.MapStr{
		"cluster_id": v.Config.ID,
		"cluster_uuid": v.Config.ClusterUUID,
	}
	item := event.Event{
		Metadata: event.EventMetadata{
			Category: "elasticsearch",
			Name:     "cluster_health",
			Datatype: "snapshot",
			Labels: labels,
		},
	}
	item.Fields = util.MapStr{
		"elasticsearch": util.MapStr{
			"cluster_health": health,
		},
	}
	return event.Save(item)
}
