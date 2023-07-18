/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package node_stats

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/elastic/adapter"
	"strings"
)

const processorName = "es_node_stats"

func init() {
	pipeline.RegisterProcessorPlugin(processorName, newProcessor)
}

func newProcessor(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{}
	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of %s processor: %s", processorName, err)
	}
	var nodeUUIDs = []string{"_local"}
	cfg.NodeUUIDs = append(nodeUUIDs, cfg.NodeUUIDs...)
	processor := NodeStats{
		config: &cfg,
	}
	_, err := adapter.GetClusterUUID(processor.config.Elasticsearch)
	if err != nil {
		log.Error(" get cluster uuid error: ", err)
	}
	return &processor, nil
}

type Config struct {
	Elasticsearch string `config:"elasticsearch,omitempty"`
	NodeUUIDs []string `config:"node_uuids,omitempty" json:"node_uuids"`
	Labels map[string]interface{} `config:"labels,omitempty"`
}

type NodeStats struct {
	config *Config
}

func (p *NodeStats) Name() string {
	return processorName
}

func (p *NodeStats) Process(c *pipeline.Context) error {
	meta := elastic.GetMetadata(p.config.Elasticsearch)
	return p.Collect(p.config.Elasticsearch, meta)
}

func (p *NodeStats) Collect(k string, v *elastic.ElasticsearchMetadata) error {
	var (
		shards []elastic.CatShardResponse
		err error
	)
	client := elastic.GetClientNoPanic(k)
	if client == nil {
		return nil
	}
	shards, err = client.CatShards()
	if err != nil {
		return err
	}
	shardInfos := map[string]map[string]interface{}{}
	indexInfos := map[string]map[string]bool{}
	for _, item := range shards {
		if item.State == "UNASSIGNED" {
			continue
		}
		if _, ok := shardInfos[item.NodeID]; !ok {
			shardInfos[item.NodeID] = map[string]interface{}{
				"shard_count":    0,
				"replicas_count": 0,
				"indices_count":  0,
				"shards":         []interface{}{},
			}
		}
		if _, ok := indexInfos[item.NodeID]; !ok {
			indexInfos[item.NodeID] = map[string]bool{}
		}
		if item.ShardType == "p" {
			shardInfos[item.NodeID]["shard_count"] = shardInfos[item.NodeID]["shard_count"].(int) + 1
		} else {
			shardInfos[item.NodeID]["replicas_count"] = shardInfos[item.NodeID]["replicas_count"].(int) + 1
		}
		shardInfos[item.NodeID]["shards"] = append(shardInfos[item.NodeID]["shards"].([]interface{}), item)
		indexInfos[item.NodeID][item.Index] = true
	}
	clusterUUID := v.Config.ClusterUUID

	host := v.GetActiveHost()
	nodeUUID := strings.Join(p.config.NodeUUIDs, ",")
	stats := client.GetNodesStats(nodeUUID, host)
	if stats.ErrorObject != nil {
		log.Errorf("error on get node stats: %v %v", host, stats.ErrorObject)
	} else {
		for nodeID, nodeStats := range stats.Nodes {
			if _, ok := shardInfos[nodeID]; ok {
				shardInfos[nodeID]["indices_count"] = len(indexInfos[nodeID])
			}
			p.SaveNodeStats(v.Config.ID, clusterUUID, nodeID, nodeStats, shardInfos[nodeID])
		}
	}
	return nil
}

func (p *NodeStats) SaveNodeStats(clusterId, clusterUUID, nodeID string, f interface{}, shardInfo interface{}) {
	//remove adaptive_selection
	x, ok := f.(map[string]interface{})
	if !ok {
		log.Errorf("invalid node stats for [%v] [%v]", clusterId, nodeID)
		return
	}

	if ok {
		delete(x, "adaptive_selection")
		delete(x, "ingest")
		util.MapStr(x).Delete("indices.segments.max_unsafe_auto_id_timestamp")
		x["shard_info"] = shardInfo
	}
	nodeName := x["name"]
	nodeIP := x["ip"]
	nodeAddress := x["transport_address"]
	labels := util.MapStr{
		"cluster_id": clusterId,
		"node_id":           nodeID,
		"node_name":         nodeName,
		"ip":                nodeIP,
		"transport_address": nodeAddress,
	}
	if clusterUUID != "" {
		labels["cluster_uuid"] = clusterUUID
	}
	if len(p.config.Labels) > 0 {
		for k, v := range p.config.Labels {
			labels[k] = v
		}
	}
	item := event.Event{
		Metadata: event.EventMetadata{
			Category: "elasticsearch",
			Name:     "node_stats",
			Datatype: "snapshot",
			Labels: labels,
		},
	}
	item.Fields = util.MapStr{
		"elasticsearch": util.MapStr{
			"node_stats": x,
		},
	}
	err := event.Save(item)
	if err != nil {
		log.Error(err)
	}
}
