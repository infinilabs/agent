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

// NodeProspectorProcessor
//  detect es node information changes.
//  get auth info from api/encrypted config file/local cluster info(kv)
type NodeProspectorProcessor struct {
	cfg Config `config:"prospector"`
}

type Config struct {
}

func init() {
	pipeline.RegisterProcessorPlugin("prospector", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of node prospector processor: %s", err)
	}
	auth.InitDecryptAuth(c, authInfoChangeCallback)
	auth.InitAPIAuth(c)
	auth.InitLocalAuth(c)
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
	//if !config2.IsAgentActivated() {
	//	return nil
	//}
	onlineClusters := p.getOnlineClusters()
	processClusters := p.getClustersFromProcess()
	if !p.isClusterChanged(onlineClusters, processClusters) && p.isClusterInfoComplete(onlineClusters){
		return nil
	}
	p.MergeNewNode(onlineClusters, processClusters)
	p.MergeNewCluster(onlineClusters, processClusters)
	p.RefreshClusterInfo()
	return nil
}

// TODO remove isClusterInfoComplete
func (p *NodeProspectorProcessor) isClusterInfoComplete(onlineClusters []*model.Cluster) bool {
	for _, cluster := range onlineClusters {
		if cluster.UUID == "" {
			return false
		}
		for _, node := range cluster.Nodes {
			if node.ID == "" {
				return false
			}
		}
	}
	return true
}

func (p *NodeProspectorProcessor) isClusterChanged(onlineClusters, processClusters []*model.Cluster) bool {
	//集群变化: 集群数量、集群信息
	//节点变化: 节点数量、节点信息

	//数量不一致
	if len(onlineClusters) != len(processClusters) {
		return true
	}
	//数量一致, 信息不一致
	olClusterMap := make(map[string]*model.Cluster)
	pClusterMap := make(map[string]*model.Cluster)
	olNodes := make(map[string]*model.Node)
	pNodes := make(map[string]*model.Node)
	for _, cluster := range onlineClusters {
		olClusterMap[cluster.Name] = cluster
		for _, node := range cluster.Nodes {
			olNodes[node.ESHomePath] = node
		}
	}
	for _, cluster := range processClusters {
		pClusterMap[cluster.Name] = cluster
		for _, node := range cluster.Nodes {
			pNodes[node.ESHomePath] = node
		}
	}
	// 节点信息的变化通过es节点配置文件的变化来判断
	var tempNode *model.Node
	var ok bool
	for path, node := range olNodes {
		tempNode, ok = pNodes[path]
		if !ok { //在线节点 在 进程中找不到
			return true
		}
		// 节点找到了，但配置文件有变更
		if !strings.EqualFold(string(node.ConfigFileContent), string(tempNode.ConfigFileContent)) {
			return true
		}
	}
	for path, _ := range pNodes {
		tempNode, ok = olNodes[path]
		if !ok { //进程节点 在 在线节点中找不到
			return true
		}
	}
	return false
}

func (p *NodeProspectorProcessor) RefreshClusterInfo()  {
	instanceInfo := config2.GetInstanceInfo()
	for _, cluster := range instanceInfo.Clusters {
		cluster.RefreshClusterInfo()
	}
}

func (p *NodeProspectorProcessor) MergeNewNode(onlineCluster, processCluster []*model.Cluster) {
	for _, cluster := range processCluster {
		for _, olCluster := range onlineCluster {
			if strings.EqualFold(cluster.Name, olCluster.Name) {
				olCluster.MergeNodes(cluster.Nodes)
			}
		}
	}
	instanceInfo := config2.GetOrInitInstanceInfo()
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
	var authType model.AuthType
	for _, cluster := range unAuthClusters {
		if len(cluster.Nodes) == 0 {
			continue
		}
		ok, authInfo, authType = auth.Auth(cluster.Name, cluster.GetEndPoint(), cluster.Nodes[0].GetPorts()...)
		if !ok {
			continue
		}
		cluster.UserName = authInfo.Username
		cluster.Password = authInfo.Password
		cluster.AuthType = authType
		authedCluster = append(authedCluster, cluster)
	}
	if len(authedCluster) == 0 {
		return
	}
	instanceInfo := config2.GetOrInitInstanceInfo()
	instanceInfo.MergeClusters(authedCluster)
	config2.SetInstanceInfo(instanceInfo)
}

func (p *NodeProspectorProcessor) getOnlineClusters() []*model.Cluster {
	instanceInfo := config2.GetOrInitInstanceInfo()
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
	if processCluster == nil {
		return nil
	}
	var result []*model.Cluster
	for _, pCluster := range processCluster {
		for _, olCluster := range onlineCluster {
			if pCluster.Name == olCluster.Name && p.isClusterInfoComplete([]*model.Cluster{olCluster}){
				goto NEXT
			}
		}
		result = append(result, pCluster)
	NEXT:
	}
	return result
}

func authInfoChangeCallback(authInfo *agent.BasicAuth)  {
	instanceInfo := config2.GetOrInitInstanceInfo()
	if instanceInfo == nil || len(instanceInfo.Clusters) == 0 {
		return
	}
	for _, cluster := range instanceInfo.Clusters {
		if cluster.AuthType == model.AuthTypeEncrypt {
			cluster.UserName = authInfo.Username
			cluster.Password = authInfo.Password
			cluster.RefreshClusterInfo()
		}
	}
	config2.SetInstanceInfo(instanceInfo)
}