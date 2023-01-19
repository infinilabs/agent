/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package cluster_stats

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

const processorName = "es_cluster_stats"

func init() {
	pipeline.RegisterProcessorPlugin(processorName, newProcessor)
}

func newProcessor(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{}
	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of %s processor: %s", processorName, err)
	}
	processor := ClusterStats{
		config: &cfg,
	}
	return &processor, nil
}

type Config struct {
	Elasticsearch string `config:"elasticsearch,omitempty"`
	Labels map[string]interface{} `config:"labels,omitempty"`
}

type ClusterStats struct {
	config *Config
}

func (p *ClusterStats) Name() string {
	return processorName
}

func (p *ClusterStats) Process(c *pipeline.Context) error {
	meta := elastic.GetMetadata(p.config.Elasticsearch)
	return p.Collect(p.config.Elasticsearch, meta)
}

func (p *ClusterStats) Collect(k string, v *elastic.ElasticsearchMetadata) error {

	log.Trace("collecting custer state metrics for :", k)

	client := elastic.GetClientNoPanic(k)
	if client == nil {
		return nil
	}

	var stats *elastic.ClusterStats
	var err error
	stats, err = client.GetClusterStatsSpecEndpoint("", v.Config.Endpoint)
	if err != nil {
		log.Error(v.Config.Name, " get cluster stats error: ", err)
		return err
	}
	labels := util.MapStr{
		"cluster_id": v.Config.ID,
		"cluster_uuid": v.Config.ClusterUUID,
	}
	if len(p.config.Labels) > 0 {
		for k, v := range p.config.Labels {
			labels[k] = v
		}
	}
	item := event.Event{
		Metadata: event.EventMetadata{
			Category: "elasticsearch",
			Name:     "cluster_stats",
			Datatype: "snapshot",
			Labels: labels,
		},
	}

	item.Fields = util.MapStr{
		"elasticsearch": util.MapStr{
			"cluster_stats": stats,
		},
	}

	return event.Save(item)
}