package metadata

import (
	log "github.com/cihub/seelog"
	"infini.sh/agent/model"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/modules/elastic/common"
	"strings"
)

func Init(hostInfo *model.Instance) {
	registerMetadata(hostInfo)
}

func registerMetadata(hostInfo *model.Instance) {
	for i := 0; i < len(hostInfo.Clusters); i++ {
		cluster := hostInfo.Clusters[i]
		if cluster.Task != nil && cluster.Task.ClusterMetric == (model.ClusterMetricTask{}) && !cluster.Task.ClusterMetric.Owner {
			continue //当前是console在采集，无需注册
		}
		if len(cluster.GetOnlineNodes()) == 0 {
			continue //集群没有活着的节点，跳过，无需采集
		}
		if err := initMetadata(cluster); err != nil {
			log.Warnf("initMetadata err: %v", err)
		}
	}
}

func HostInfoChanged(newHostInfo *model.Instance) {
	log.Debugf("metadata.HostInfoChanged:  register/update\n")
	if newHostInfo == nil {
		return
	}
	registerMetadata(newHostInfo)
}

func initMetadata(cluster *model.Cluster) error {
	escfg := &elastic.ElasticsearchConfig{BasicAuth: &struct {
		Username string `json:"username,omitempty" config:"username" elastic_mapping:"username:{type:keyword}"`
		Password string `json:"password,omitempty" config:"password" elastic_mapping:"password:{type:keyword}"`
	}{
		Username: cluster.UserName,
		Password: cluster.Password,
	}}
	clusterEndpoint := GetClusterTaskEndPoint(cluster)
	escfg.ID = cluster.ID
	escfg.Version = cluster.Version
	escfg.Name = cluster.UUID
	escfg.Enabled = true
	escfg.Monitored = true
	escfg.Endpoint = clusterEndpoint
	escfg.MetadataConfigs = &elastic.MetadataConfig{
		HealthCheck:           elastic.TaskConfig{Enabled: true},
		ClusterSettingsCheck:  elastic.TaskConfig{Enabled: false},
		MetadataRefresh:       elastic.TaskConfig{Enabled: false},
		NodeAvailabilityCheck: elastic.TaskConfig{Enabled: true},
	}
	client, err := common.InitClientWithConfig(*escfg)
	if err != nil {
		return err
	}
	elastic.RegisterInstance(*escfg, client)
	elastic.GetOrInitHost(removeUrlSchema(clusterEndpoint), cluster.ID)
	metadata := elastic.GetOrInitMetadata(escfg)
	metadata.IsAgentMode = true
	metadata.Nodes = GetNodesInfo(cluster)

	metadata.Config.MonitorConfigs = &elastic.MonitorConfig{
		ClusterStats: elastic.TaskConfig{
			Enabled:  true,
			Interval: "10s",
		},
		NodeStats: elastic.TaskConfig{
			Enabled:  true,
			Interval: "10s",
		},
		ClusterHealth: elastic.TaskConfig{
			Enabled:  true,
			Interval: "10s",
		},
		IndexStats: elastic.TaskConfig{
			Enabled:  true,
			Interval: "10s",
		},
	}
	elastic.SetMetadata(cluster.ID, metadata)
	return nil
}

func GetNodesInfo(cluster *model.Cluster) *map[string]elastic.NodesInfo {

	nodesInfo := make(map[string]elastic.NodesInfo)
	for i := 0; i < len(cluster.Nodes); i++ {
		node := cluster.Nodes[i]
		if node.Status == model.Offline {
			continue
		}
		nodesInfo[node.ID] = elastic.NodesInfo{
			Http: struct {
				BoundAddress            []string `json:"bound_address"`
				PublishAddress          string   `json:"publish_address,omitempty"`
				MaxContentLengthInBytes int64    `json:"max_content_length_in_bytes,omitempty"`
			}{
				BoundAddress:   []string{removeUrlSchema(node.GetEndPoint(cluster.GetSchema()))},
				PublishAddress: removeUrlSchema(node.GetEndPoint(cluster.GetSchema())),
			},
		}
	}

	if cluster.Task == nil || cluster.Task.NodeMetric == nil {
		return &nodesInfo
	}
	for _, nodeId := range cluster.Task.NodeMetric.ExtraNodes {
		nodesInfo[nodeId] = elastic.NodesInfo{
			Http: struct {
				BoundAddress            []string `json:"bound_address"`
				PublishAddress          string   `json:"publish_address,omitempty"`
				MaxContentLengthInBytes int64    `json:"max_content_length_in_bytes,omitempty"`
			}{
				BoundAddress:   []string{removeUrlSchema(cluster.GetEndPoint())},
				PublishAddress: removeUrlSchema(cluster.GetEndPoint()),
			},
		}
	}
	return &nodesInfo
}

// GetClusterTaskEndPoint
// @Description: 要根据console指定的节点来拿endpoint，用来采集集群层面的指标。
//
func GetClusterTaskEndPoint(cluster *model.Cluster) string {
	for i := 0; i < len(cluster.Nodes); i++ {
		node := cluster.Nodes[i]
		if node.ID == cluster.Task.ClusterMetric.TaskNodeID {
			return node.GetEndPoint(cluster.GetSchema())
		}
	}
	return cluster.GetEndPoint()
}

func removeUrlSchema(url string) string {
	if url == "" {
		return ""
	}
	if strings.Contains(url, "https://") {
		return strings.ReplaceAll(url, "https://", "")
	}
	return strings.ReplaceAll(url, "http://", "")
}
