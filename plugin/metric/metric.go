package metric

import (
	"context"
	"fmt"
	agentconfig "infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	log "src/github.com/cihub/seelog"
	"time"
)

type MetricConfig struct {
	Enabled  bool `config:"enabled"`
	Interval uint `config:"interval"`
}

type MetricDataModule struct {
	config *MetricConfig
}

var moduleTask *task.ScheduleTask

func (module *MetricDataModule) Name() string {
	return "nodemetric"
}

func (module *MetricDataModule) Setup(cfg *config.Config) {
	//cfg.Unpack(&module)
	module.config = &MetricConfig{}
	ok, err := env.ParseConfig("nodemetric", module.config)
	if err != nil {
		panic(err)
	}
	if !ok {
		module.config.Enabled = false
		return
	}
}

func (module *MetricDataModule) Start() error {
	if !module.config.Enabled {
		return nil
	}
	moduleTask := task.ScheduleTask{
		Description: "agent collect metric data",
		Type:        "interval",
		Interval:    fmt.Sprintf("%ds", module.config.Interval),
		Task: func(ctx context.Context) {
			hostInfo := agentconfig.GetHostInfo()
			if hostInfo == nil || hostInfo.Clusters == nil {
				return
			}
			for _, cluster := range hostInfo.Clusters {
				if !cluster.TaskOwner {
					continue
				}
				if err := collectNodeState(cluster); err != nil {
					log.Error(cluster.Name, "metric.Start: get node info error: ", err)
				}
				//当前集群没有节点被指派任务，则跳过
				taskNode := cluster.GetTaskOwnerNode()
				if taskNode == nil {
					continue
				}
				fmt.Printf("节点:%s || %s 收到任务\n", taskNode.ID, taskNode.Name)
				client, err := agentconfig.InitOrGetElasticClient(taskNode.ID,
					cluster.UserName, cluster.Password, cluster.Version, taskNode.GetNetWorkHost(cluster.GetSchema()))
				if err != nil {
					log.Error(cluster.Name, "metric.Start: get elastic client error: ", err)
					continue
				}
				if err := collectClusterHealth(client, cluster); err != nil {
					log.Error(cluster.Name, "metric.Start: get cluster health error: ", err)
				}
				if err := collectClusterState(client, cluster); err != nil {
					log.Error(cluster.Name, "metric.Start: get cluster state error: ", err)
				}
				if err := collectIndexState(client, cluster); err != nil {
					log.Error(cluster.Name, "metric.Start: get cluster state error: ", err)
				}
			}
		},
	}
	task.RegisterScheduleTask(moduleTask)
	return nil
}

func CollectDataTask() {

}

func (module *MetricDataModule) Stop() error {

	return nil
}

func collectNodeState(cluster *model.Cluster) error {
	//TODO 这里需要优化，没必要把所有的节点数据查出来。给client增加查单个节点信息的方法
	if len(cluster.Nodes) <= 0 {
		return nil
	}
	//这里不限制使用哪个节点了，只要能拿到就行。
	nodeTemp := cluster.Nodes[0]
	clientTemp, err := agentconfig.InitOrGetElasticClient(nodeTemp.ID,
		cluster.UserName,
		cluster.Password,
		cluster.Version,
		nodeTemp.GetNetWorkHost(cluster.GetSchema()),
	)
	if clientTemp == nil {
		return errors.New("metric.collectNodeState: elastic client is nil")
	}
	if err != nil {
		return err
	}
	t1 := time.Now()
	shards, err := clientTemp.CatShards()
	log.Trace("time of CatShards:", time.Since(t1).String())

	if err != nil {
		return errors.Wrap(err, "metric.collectNodeState: get shards info error")
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

	for _, node := range cluster.Nodes {
		client, err := agentconfig.InitOrGetElasticClient(node.ID, cluster.UserName,
			cluster.Password, cluster.Version, node.GetNetWorkHost(cluster.GetSchema()))
		if err != nil {
			log.Errorf("metric.collectNodeState: get node stats of %s error: %v", cluster.Name)
			continue
		}
		nodeHost := node.GetNetWorkHost(cluster.GetSchema())
		stats := client.GetNodesStats(node.ID, nodeHost)

		if stats.ErrorObject != nil {
			log.Errorf("metric.collectNodeState: get node stats of %s error: %v", cluster.Name, stats.ErrorObject)
			continue
		}
		if _, ok := shardInfos[node.ID]; ok {
			shardInfos[node.ID]["indices_count"] = len(indexInfos[node.ID])
		}
		SaveNodeStats(cluster.ID, node.ID, stats.Nodes[node.ID], shardInfos[node.ID])
	}
	return nil
}

func SaveNodeStats(clusterId, nodeID string, f interface{}, shardInfo interface{}) {
	x, ok := f.(map[string]interface{})
	if !ok {
		log.Errorf("metric.SaveNodeStats: invalid node stats for [%v] [%v]", clusterId, nodeID)
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
	item := event.Event{
		Metadata: event.EventMetadata{
			Category: "elasticsearch",
			Name:     "node_stats",
			Datatype: "snapshot",
			Labels: util.MapStr{
				"cluster_id": clusterId,
				//"cluster_uuid": stats.ClusterUUID,
				"node_id":           nodeID,
				"node_name":         nodeName,
				"ip":                nodeIP,
				"transport_address": nodeAddress,
			},
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

func collectClusterHealth(client elastic.API, cluster *model.Cluster) error {
	//这里需要限制，用console指定的节点来获取集群数据
	health, err := client.ClusterHealth()
	if err != nil {
		return err
	}

	item := event.Event{
		Metadata: event.EventMetadata{
			Category: "elasticsearch",
			Name:     "cluster_health",
			Datatype: "snapshot",
			Labels: util.MapStr{
				"cluster_id": cluster.ID,
			},
		},
	}
	item.Fields = util.MapStr{
		"elasticsearch": util.MapStr{
			"cluster_health": health,
		},
	}
	return event.Save(item)
}

func collectClusterState(client elastic.API, cluster *model.Cluster) error {

	stats, err := client.GetClusterStats("")
	if err != nil {
		return err
	}

	item := event.Event{
		Metadata: event.EventMetadata{
			Category: "elasticsearch",
			Name:     "cluster_stats",
			Datatype: "snapshot",
			Labels: util.MapStr{
				"cluster_id": cluster.ID,
			},
		},
	}

	item.Fields = util.MapStr{
		"elasticsearch": util.MapStr{
			"cluster_stats": stats,
		},
	}

	return event.Save(item)
}

func collectIndexState(client elastic.API, cluster *model.Cluster) error {

	shards, err := client.CatShards() //TODO 这里可以和其他方法共用一次CatShards()
	indexStats, err := client.GetStats()
	if err != nil {
		log.Error(cluster.Name, "metric.collectIndexState: get indices stats error: ", err)
		return nil
	}

	if indexStats != nil {
		var indexInfos *map[string]elastic.IndexInfo
		shardInfos := map[string][]elastic.CatShardResponse{}

		indexInfos, err = client.GetIndices("")
		if err != nil {
			log.Error(cluster.Name, "metric.collectIndexState: get indices info error: ", err)
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

		//TODO
		AllIndexStats := true
		IndexStats := true
		if AllIndexStats {
			SaveIndexStats(cluster.ID, "_all", "_all", indexStats.All.Primaries, indexStats.All.Total, nil, nil)
		}

		if IndexStats {
			for x, y := range indexStats.Indices {
				var indexInfo elastic.IndexInfo
				var shardInfo []elastic.CatShardResponse
				if indexInfos != nil {
					indexInfo = (*indexInfos)[x]
				}
				if shardInfos != nil {
					shardInfo = shardInfos[x]
				}
				SaveIndexStats(cluster.ID, y.Uuid, x, y.Primaries, y.Total, &indexInfo, shardInfo)
			}
		}
	}
	return nil
}

func SaveIndexStats(clusterId, indexID, indexName string, primary, total elastic.IndexLevelStats, info *elastic.IndexInfo, shardInfo []elastic.CatShardResponse) {
	newIndexID := fmt.Sprintf("%s:%s", clusterId, indexName)
	if indexID == "_all" {
		newIndexID = indexID
	}
	item := event.Event{
		Metadata: event.EventMetadata{
			Category: "elasticsearch",
			Name:     "index_stats",
			Datatype: "snapshot",
			Labels: util.MapStr{
				"cluster_id": clusterId,
				//"cluster_uuid": clusterId,
				"index_id":   newIndexID,
				"index_uuid": indexID,
				"index_name": indexName,
			},
		},
	}

	mtr := util.MapStr{}
	//TODO
	IndexPrimaryStats := true
	if IndexPrimaryStats {
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
