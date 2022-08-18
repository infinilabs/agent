package metadata

import (
	"infini.sh/agent/model"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/modules/elastic/common"
	"log"
)

func Init(hostInfo *model.Host) {
	RegisterMetadata(hostInfo)
}

func RegisterMetadata(hostInfo *model.Host) {
	for i := 0; i < len(hostInfo.Clusters); i++ {
		cluster := hostInfo.Clusters[i]
		if cluster.Task != nil && cluster.Task.ClusterMetric == (model.ClusterMetricTask{}) && !cluster.Task.ClusterMetric.Owner {
			continue //当前是console在采集，无需注册
		}
		for i := 0; i < len(cluster.Nodes); i++ {
			node := cluster.Nodes[i]
			clusterTaskOwner := false
			if node.ID == cluster.Task.ClusterMetric.TaskNodeID {
				clusterTaskOwner = true
			}
			err := InitOrUpdateElasticClient(node.ID, cluster.UserName, cluster.Password, cluster.Version, node.GetEndPoint(cluster.GetSchema()), clusterTaskOwner)
			if err != nil {
				log.Printf("RegisterMetadata Failed, %v\n", err)
				continue
			}
		}
	}

}

func HostInfoChanged(newHostInfo *model.Host) {
	log.Printf("metadata:  rigister/update\n")
	if newHostInfo == nil {
		return
	}
	RegisterMetadata(newHostInfo)
}

func InitOrUpdateElasticClient(esNodeId string, userName string, password string, esVersion string, host string, clusterOwner bool) error {

	elasticSearchConfig := &elastic.ElasticsearchConfig{BasicAuth: &struct {
		Username string `json:"username,omitempty" config:"username" elastic_mapping:"username:{type:keyword}"`
		Password string `json:"password,omitempty" config:"password" elastic_mapping:"password:{type:keyword}"`
	}{
		Username: userName,
		Password: password,
	}}
	elasticSearchConfig.ID = esNodeId
	elasticSearchConfig.Version = esVersion
	elasticSearchConfig.Name = esNodeId
	elasticSearchConfig.Enabled = true
	elasticSearchConfig.Monitored = true
	elasticSearchConfig.Endpoint = host
	elasticSearchConfig.MetadataConfigs = &elastic.MetadataConfig{
		HealthCheck:           elastic.TaskConfig{Enabled: true},
		ClusterSettingsCheck:  elastic.TaskConfig{Enabled: false},
		MetadataRefresh:       elastic.TaskConfig{Enabled: false},
		NodeAvailabilityCheck: elastic.TaskConfig{Enabled: true},
	}
	client, err := common.InitClientWithConfig(*elasticSearchConfig)
	if err != nil {
		return err
	}
	elastic.RegisterInstance(*elasticSearchConfig, client)
	elastic.GetOrInitHost(host, esNodeId)
	metadata := elastic.GetOrInitMetadata(elasticSearchConfig)
	nodesInfo := make(map[string]elastic.NodesInfo)
	nodesInfo[esNodeId] = elastic.NodesInfo{
		Http: struct {
			BoundAddress            []string `json:"bound_address"`
			PublishAddress          string   `json:"publish_address,omitempty"`
			MaxContentLengthInBytes int64    `json:"max_content_length_in_bytes,omitempty"`
		}{
			PublishAddress: host,
		},
	}
	metadata.Nodes = &nodesInfo
	metadata.Config.MonitorConfigs = &elastic.MonitorConfig{
		ClusterStats: elastic.TaskConfig{
			Enabled:  false,
			Interval: "10s",
		},
		NodeStats: elastic.TaskConfig{
			Enabled:  true,
			Interval: "10s",
		},
		ClusterHealth: elastic.TaskConfig{
			Enabled:  clusterOwner,
			Interval: "10s",
		},
		IndexStats: elastic.TaskConfig{
			Enabled:  false,
			Interval: "10s",
		},
	}
	elastic.SetMetadata(esNodeId, metadata)
	return nil
}
