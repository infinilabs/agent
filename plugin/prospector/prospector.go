/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package prospector

import (
	"fmt"
	log "github.com/cihub/seelog"
	config2 "infini.sh/agent/config"
	"infini.sh/agent/lib/auth"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage/instance"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	"strings"
)

type NodeProspectorProcessor struct {
	cfg Config `config:"prospector"`
}

type Config struct {
	DecryptAuthConfig *config.Config `config:"decrypt_auth"`
}

func init() {
	pipeline.RegisterProcessorPlugin("prospector", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of node prospector processor: %s", err)
	}

	localAuth, err := auth.NewLocalAuthenticator()
	if err != nil {
		log.Error(err)
	}
	auth.RegisterAuth(localAuth)
	decryptAuth, err := auth.NewDecryptAuthenticator(cfg.DecryptAuthConfig)
	if err != nil {
		log.Error(err)
	}
	auth.RegisterAuth(decryptAuth)
	return &NodeProspectorProcessor{
		cfg: cfg,
	}, nil
}

func (p *NodeProspectorProcessor) Name() string {
	return "prospector"
}

func (p *NodeProspectorProcessor) Process(c *pipeline.Context) error {
	// step1: merge new nodes(into old cluster)
	// step2: merge new clusters
	if !config2.IsAgentActivated() {
		return nil
	}
	onlineCluster := p.getOnlineClusters()
	processCluster := p.getClustersFromProcess()
	p.MergeNewNode(onlineCluster, processCluster)
	p.MergeNewCluster(onlineCluster, processCluster)
	return nil
}

func (p *NodeProspectorProcessor) MergeNewNode(onlineCluster, processCluster []*model.Cluster) {
	for _, cluster := range processCluster {
		for _, olCluster := range onlineCluster {
			if strings.EqualFold(cluster.Name, olCluster.Name) {
				olCluster.MergeNodes(cluster.Nodes)
			}
		}
	}
	instanceInfo := config2.GetInstanceInfo()
	instanceInfo.Clusters = onlineCluster
	config2.SetInstanceInfo(instanceInfo)
}

func (p *NodeProspectorProcessor) MergeNewCluster(onlineCluster, processCluster []*model.Cluster) {
	unAuthClusters := p.getUnAuthCluster(onlineCluster, processCluster)
	if len(unAuthClusters) == 0 {
		return
	}
	log.Debugf("get un auth cluster: %s", util.MustToJSON(unAuthClusters))
	var ok bool
	var authInfo *agent.BasicAuth
	var authedCluster []*model.Cluster
	for _, cluster := range unAuthClusters {
		if len(cluster.Nodes) == 0 {
			continue
		}
		ok, authInfo = auth.Auth(cluster.Name, cluster.GetEndPoint(), cluster.Nodes[0].GetPorts()...)
		if !ok {
			continue
		}
		cluster.UserName = authInfo.Username
		cluster.Password = authInfo.Password
		cluster.RefreshClusterInfo()
		authedCluster = append(authedCluster, cluster)
	}
	instanceInfo := config2.GetInstanceInfo()
	instanceInfo.MergeClusters(authedCluster)
	config2.SetInstanceInfo(instanceInfo)
}

func (p *NodeProspectorProcessor) getOnlineClusters() []*model.Cluster {
	instanceInfo := config2.GetInstanceInfo()
	return instanceInfo.Clusters
}

func (p *NodeProspectorProcessor) getClustersFromProcess() []*model.Cluster {
	instanceInfo, err := instance.GetInstanceInfo()
	if err != nil {
		return nil
	}
	return instanceInfo.Clusters
}

func (p *NodeProspectorProcessor) getUnAuthCluster(onlineCluster, processCluster []*model.Cluster) []*model.Cluster {
	if onlineCluster == nil || processCluster == nil {
		return nil
	}
	var result []*model.Cluster
	for _, pCluster := range processCluster {
		for _, olCluster := range onlineCluster {
			if pCluster.Name == olCluster.Name {
				goto NEXT
			}
		}
		result = append(result, pCluster)
	NEXT:
	}
	return result
}
