/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package host

import (
	"encoding/json"
	"fmt"
	"infini.sh/agent/api"
	"infini.sh/agent/model"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"log"
	"net/http"
	"src/github.com/buger/jsonparser"
	"src/gopkg.in/yaml.v2"
	"strings"
)

func getHostInfo() model.Host {

	host := model.Host{}
	host.IPs = util.GetLocalIPs()
	processInfos := getProcessInfo()
	pathPorts := getNodeConfigPaths(processInfos)
	host.Clusters = getClusterConfigs(pathPorts)
	host.TLS = false //TODO 这里要改成从yml读取
	return host
}

func RegisterHost() []*model.Cluster {

	if isRegistered() {
		return nil
	}

	host := getHostInfo()
	body, err := json.Marshal(host)
	if err != nil {
		log.Printf("get hostinfo failed %v", err)
		return nil
	}
	resp, err := http.Post(api.UrlRegisterHost, "application/json", strings.NewReader(string(body)))
	defer resp.Body.Close()
	if err != nil {
		log.Printf("register host failed\n%v", err)
		return nil
	}
	var tempModules []agent.ESCluster
	err = json.NewDecoder(resp.Body).Decode(tempModules)
	if err != nil {
		log.Printf("parse %s response failed\n%v", api.UrlRegisterHost, err)
		return nil
	}
	var resultESCluster []*model.Cluster
	esClusters := make(map[string]*agent.ESCluster)
	for _, module := range tempModules {
		esClusters[module.ClusterID] = &module
	}

	for _, cluster := range host.Clusters {
		retCluster := esClusters[cluster.Name]
		if retCluster == nil {
			continue
		}
		for _, node := range cluster.Nodes {
			port := validatePort(retCluster.ClusterUUID, retCluster.BasicAuth.Username, retCluster.BasicAuth.Password, node.Ports)
			node.HttpPort = port
			resultESCluster = append(resultESCluster, cluster)
		}
	}
	return resultESCluster
}

func isRegistered() bool {
	clientInfo := getAgentClientInfo()
	if clientInfo == nil {
		return false
	}

	if clientInfo.AgentID == "" {
		return false
	}
	return true
}

func validatePort(clusterID string, name string, pwd string, ports []int) int {
	for _, port := range ports {
		//TODO 需要考虑https吗？
		url := fmt.Sprintf("http://localhost:%d", port)
		var req = util.NewGetRequest(url, nil)
		if name != "" && pwd != "" {
			req.SetBasicAuth(name, pwd)
		}
		result, err := util.ExecuteRequest(req)
		if err != nil {
			log.Printf("%v", err)
			continue
		}
		clusterUuid, _ := jsonparser.GetString(result.Body, "cluster_uuid")
		if strings.EqualFold(clusterUuid, clusterID) {
			return port
		}
	}
	return 0
}

func saveAgentClientInfo(info []byte) {
	//info => console接口返回
	var agentModule model.Host //这个实体和console那边的保持一致
	err := yaml.Unmarshal(info, agentModule)
	if err != nil {
		log.Printf("save agent info failed\n %v", err)
		return
	}
	yml, _ := yaml.Marshal(agentModule)
	fileName := fmt.Sprintf("%s/agent_client.yml", global.Env().GetDataDir())
	_, err = util.FilePutContent(fileName, string(yml))
	if err != nil {
		log.Printf("save agent info failed\n %v", err)
		return
	}
}

func getAgentClientInfo() *model.Host {
	fileName := fmt.Sprintf("%s/agent_client.yml", global.Env().GetDataDir())
	content, err := util.FileGetContent(fileName)
	if err != nil {
		log.Printf("read agent_client.yml failed\n %v", err)
		return nil
	}
	var config model.Host
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		log.Printf("read agent_client.yml failed\n %v", err)
	}
	return &config
}
