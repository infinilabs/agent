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
	cfg := Config{
		Level: "shards", //cluster,indices,shards
	}
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
	Elasticsearch string                 `config:"elasticsearch,omitempty"`
	NodeUUIDs     []string               `config:"node_uuids,omitempty" json:"node_uuids"`
	Labels        map[string]interface{} `config:"labels,omitempty"`
	Level         string                 `config:"level,omitempty"`
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
		err error
	)
	client := elastic.GetClientNoPanic(k)
	if client == nil {
		return nil
	}

	//get shards info
	shardInfos := map[string]map[string]interface{}{}
	indexInfos := map[string]map[string]bool{}

	//TODO remove, deprecate after 2.0
	if p.config.Level == "" {
		var shards []elastic.CatShardResponse
		shards, err = client.CatShards()
		if err != nil {
			return err
		}

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
	}

	clusterUUID := v.Config.ClusterUUID

	host := v.GetActiveHost()
	nodeUUID := strings.Join(p.config.NodeUUIDs, ",")
	stats := client.GetNodesStats(nodeUUID, host, p.config.Level)
	if stats.ErrorObject != nil {
		log.Errorf("error on get node stats: %v %v", host, stats.ErrorObject)
	} else {
		for nodeID, nodeStats := range stats.Nodes {
			shardsCount := 0
			indexCount := 0
			shardInfo := util.MapStr{}
			if p.config.Level == "shards" {
				nodeData, ok := nodeStats.(map[string]interface{})
				if ok {
					nodeHost:=nodeData["host"].(string)
					indexData, ok := nodeData["indices"].(map[string]interface{})
					if ok {
						//shards
						x, ok := indexData["shards"].(map[string]interface{})
						if ok {
							indexCount = len(x)
							for indexName, f := range x {
								//e is index name
								//f is shards in array type
								indexUUID:="" //TODO get index uuid
								u, ok := f.([]interface{})
								if ok {
									for _, g := range u {
										m, ok := g.(map[string]interface{})
										if ok {
											for shardID, i := range m {
												shardsCount++

												p.SaveShardStats(v.Config.ID, clusterUUID, nodeID,nodeHost,indexName,indexUUID, shardID, i)
											}
										}
									}
								}
							}
						}

						shardInfo.Put("indices_count", indexCount)
						shardInfo.Put("shard_count", shardsCount)
						//shardsInfo.Put("replicas_count",shardsCount)

						//delete shards data
						delete(indexData, "shards")
					}
				}
			} else {
				if _, ok := shardInfos[nodeID]; ok {
					shardInfos[nodeID]["indices_count"] = len(indexInfos[nodeID])
				}
				shardInfo = shardInfos[nodeID]
			}
			p.SaveNodeStats(v.Config.ID, clusterUUID, nodeID, nodeStats, shardInfo)
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
		"cluster_id":        clusterId,
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
			Labels:   labels,
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

func (p *NodeStats) SaveShardStats(clusterId, clusterUUID, nodeID,host string, indexName, indexUUID string, shardID string, f interface{}) {

	x, ok := f.(map[string]interface{})
	if !ok {
		log.Errorf("invalid shard stats for [%v] [%v] [%v] [%v]", clusterId, nodeID, indexName, shardID)
		return
	}

	newIndexID := fmt.Sprintf("%s:%s", clusterUUID, indexName)
	if indexName == "_all" {
		newIndexID = indexName
	} //TODO ??

	labels := util.MapStr{
		"cluster_id": clusterId,
		"node_id":    nodeID,
		"index_name": indexName,
		"index_id":   newIndexID,
		"ip":   host,
		"shard":   shardID,
		"shard_id": fmt.Sprintf("%s:%s:%s", nodeID, indexName, shardID),
	}

	if clusterUUID != "" {
		labels["cluster_uuid"] = clusterUUID
	}

	if indexUUID != "" {
		labels["index_uuid"] = indexUUID
	}

	if len(p.config.Labels) > 0 {
		for k, v := range p.config.Labels {
			labels[k] = v
		}
	}
	item := event.Event{
		Metadata: event.EventMetadata{
			Category: "elasticsearch",
			Name:     "shard_stats",
			Datatype: "snapshot",
			Labels:   labels,
		},
	}
	item.Fields = util.MapStr{
		"elasticsearch": util.MapStr{
			"shard_stats": x,
		},
	}
	err := event.Save(item)
	if err != nil {
		log.Error(err)
	}
}
