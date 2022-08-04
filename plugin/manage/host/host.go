/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package host

import (
	"encoding/json"
	"fmt"
	"infini.sh/agent/api"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"io/ioutil"
	"log"
	"net/http"
	"src/github.com/buger/jsonparser"
	"src/gopkg.in/yaml.v2"
	"strings"
)

func getHostInfo() (*model.Host, error) {

	host := &model.Host{}
	host.IPs = util.GetLocalIPs()
	processInfos := getProcessInfo()
	pathPorts := getNodeConfigPaths(processInfos)
	clusters, err := getClusterConfigs(pathPorts)
	if err != nil {
		return nil, errors.Wrap(err, "getClusterConfigs failed")
	}
	host.Clusters = clusters
	if config.EnvConfig.Schema == "https" {
		host.TLS = true
	} else {
		host.TLS = false
	}
	return host, nil
}

func RegisterHost() (*model.Host, error) {

	if isRegistered() {
		return nil, nil
	}

	host, err := getHostInfo()
	if err != nil {
		return nil, errors.Wrap(err, "RegisterHost failed")
	}
	consoleHost := agent.Instance{
		Schema: config.EnvConfig.Schema,
		Port:   config.EnvConfig.Port,
		IPS:    host.IPs,
	}
	for _, clusLoc := range host.Clusters {
		consoleHost.Clusters = append(consoleHost.Clusters, agent.ESCluster{
			ClusterName: clusLoc.Name,
			ClusterUUID: "KuvqunvmTAyEwx_b13eshg",
		})
	}
	body, err := json.Marshal(consoleHost)
	if err != nil {
		return nil, errors.Wrap(err, "get hostinfo failed")
	}
	fmt.Printf("注册agent: %v", string(body))
	url := fmt.Sprintf("%s/%s", api.UrlConsole, api.UrlUploadHostInfo)
	fmt.Println(url)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.Wrap(err, "register host failed")
	}
	defer resp.Body.Close()
	bodyC, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("返回: %s", bodyC)
	var tempModules []agent.ESCluster
	err = json.NewDecoder(resp.Body).Decode(tempModules)
	if err != nil {
		log.Printf("parse %s response failed\n%v", url, err)
		return nil, errors.Wrapf(err, "parse %s response failed", url)
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
		cluster.UserName = retCluster.BasicAuth.Username
		cluster.Password = retCluster.BasicAuth.Password
		cluster.UUID = retCluster.ClusterUUID
		for _, node := range cluster.Nodes {
			port := validatePort(retCluster.ClusterUUID, retCluster.BasicAuth.Username, retCluster.BasicAuth.Password, node.Ports)
			node.HttpPort = port
			resultESCluster = append(resultESCluster, cluster)
		}
	}
	host.Clusters = resultESCluster
	return host, nil
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
	if ports == nil {
		return 0
	}
	for _, port := range ports {
		//TODO https？
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
