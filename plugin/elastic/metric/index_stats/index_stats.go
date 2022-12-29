/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package index_stats

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

const processorName = "es_index_stats"

func init() {
	pipeline.RegisterProcessorPlugin(processorName, newProcessor)
}

func newProcessor(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		AllIndexStats: true,
		IndexPrimaryStats: true,
		IndexTotalStats: true,
	}
	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of %s processor: %s", processorName, err)
	}
	processor := IndexStats{
		config: &cfg,
	}
	return &processor, nil
}

type Config struct {
	Elasticsearch string `config:"elasticsearch,omitempty"`
	AllIndexStats bool `config:"all_index_stats,omitempty"`
	IndexPrimaryStats bool `config:"index_primary_stats"`
	IndexTotalStats   bool `config:"index_total_stats"`
}

type IndexStats struct {
	config *Config
}

func (p *IndexStats) Name() string {
	return processorName
}

func (p *IndexStats) Process(c *pipeline.Context) error {
	meta := elastic.GetMetadata(p.config.Elasticsearch)
	return p.Collect(p.config.Elasticsearch, meta)
}

func (p *IndexStats) Collect(k string, v *elastic.ElasticsearchMetadata) error {
	var (
		shards []elastic.CatShardResponse
		err error
	)
	client := elastic.GetClientNoPanic(k)
	if client == nil {
		return nil
	}
	shards, err = client.CatShardsSpecEndpoint(v.Config.Endpoint)
	if err != nil {
		return err
	}

	clusterUUID := v.Config.ClusterUUID
	indexStats, err := client.GetStats()
	if err != nil {
		return err
	}

	if indexStats != nil {
		var indexInfos *map[string]elastic.IndexInfo
		shardInfos := map[string][]elastic.CatShardResponse{}

		if v.IsAvailable() {
			indexInfos, err = client.GetIndices("")
			if err != nil {
				log.Error(v.Config.Name, " get indices info error: ", err)
			}

			for _, item := range shards {
				if _, ok := shardInfos[item.Index]; !ok {
					shardInfos[item.Index] = []elastic.CatShardResponse{
						item,
					}
				} else {
					shardInfos[item.Index] = append(shardInfos[item.Index], item)
				}
			}
		}

		if p.config.AllIndexStats {
			p.SaveIndexStats(v.Config.ID, clusterUUID, "_all", "_all", indexStats.All.Primaries, indexStats.All.Total, nil, nil)
		}

		for x, y := range indexStats.Indices {
			var indexInfo elastic.IndexInfo
			var shardInfo []elastic.CatShardResponse
			if indexInfos != nil {
				indexInfo = (*indexInfos)[x]
			}
			if shardInfos != nil {
				shardInfo = shardInfos[x]
			}
			p.SaveIndexStats(v.Config.ID, clusterUUID, y.Uuid, x, y.Primaries, y.Total, &indexInfo, shardInfo)
		}

	}
	return nil
}

func (p *IndexStats) SaveIndexStats(clusterId, clusterUUID, indexID, indexName string, primary, total elastic.IndexLevelStats, info *elastic.IndexInfo, shardInfo []elastic.CatShardResponse) {
	newIndexID := fmt.Sprintf("%s:%s", clusterId, indexName)
	if indexID == "_all" {
		newIndexID = indexID
	}
	labels := util.MapStr{
		"cluster_id": clusterId,
		"index_id":   newIndexID,
		"index_uuid": indexID,
		"index_name": indexName,
	}
	if clusterUUID != "" {
		labels["cluster_uuid"] = clusterUUID
	}
	item := event.Event{
		Metadata: event.EventMetadata{
			Category: "elasticsearch",
			Name:     "index_stats",
			Datatype: "snapshot",
			Labels: labels,
		},
	}

	mtr := util.MapStr{}
	if p.config.IndexPrimaryStats {
		mtr["primaries"] = primary
		mtr["total"] = total
		mtr["index_info"] = info
		mtr["shard_info"] = shardInfo
	}

	item.Fields = util.MapStr{
		"elasticsearch": util.MapStr{
			"index_stats": mtr,
		},
	}

	event.Save(item)
}
