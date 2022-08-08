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
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
	"io/ioutil"
	"log"
	"net/http"
	"src/github.com/buger/jsonparser"
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
	host.TLS = config.EnvConfig.TLS
	return host, nil
}

func RegisterHost() (*model.Host, error) {

	host, err := getHostInfo()
	if err != nil {
		return nil, errors.Wrap(err, "RegisterHost failed")
	}
	host.TLS = config.EnvConfig.TLS
	host.AgentPort = config.EnvConfig.Port

	instance := host.ToAgentInstance()
	body, err := json.Marshal(instance)
	if err != nil {
		return nil, errors.Wrap(err, "get hostinfo failed")
	}
	fmt.Printf("register agent: %v\n", string(body))
	url := fmt.Sprintf("%s/%s", config.UrlConsole(), api.UrlUploadHostInfo)
	fmt.Println(url)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.Wrap(err, "register host failed")
	}
	defer resp.Body.Close()
	bodyC, _ := ioutil.ReadAll(resp.Body)
	if strings.Contains(string(bodyC), "already exists") {
		return nil, errors.New(fmt.Sprintf("\ncurrent cluster registered\nplease delete first in console\n"))
	}

	var registerResp model.RegisterResponse
	util.MustFromJSONBytes(bodyC, &registerResp)
	host.AgentID = registerResp.AgentId
	for _, cluster := range host.Clusters {
		if respCluster, ok := registerResp.Clusters[cluster.Name]; ok {
			cluster.ID = respCluster.ClusterId
			cluster.UserName = respCluster.BasicAuth.Username
			cluster.Password = respCluster.BasicAuth.Password
		}
		for _, node := range cluster.Nodes {
			if node.HttpPort == 0 {
				node.HttpPort = validatePort(node.NetWorkHost, cluster.GetSchema(), cluster.UUID, cluster.UserName, cluster.Password, node.Ports)
			}
		}
	}
	return host, nil
}

func IsHostInfoChanged() (bool, error) {
	originHost := config.GetHostInfo()
	if originHost == nil {
		return true, nil
	}
	//判断es配置文件是否变化(集群名称、节点名、端口等). 任意一个节点配置文件变化，都触发更新
	for _, v := range originHost.Clusters {
		for _, node := range v.Nodes {
			currentFileContent, err := util.FileGetContent(node.ConfigPath)
			if err != nil {
				//读取文件失败，则认为es的文件发生了变化，如: 被删除了。 需要更新主机信息
				log.Printf("read config file failed, path: \n%s\n", node.ConfigPath)
				return true, nil
			}
			if !strings.EqualFold(string(currentFileContent), string(node.ConfigFileContent)) {
				return true, nil
			}
		}
	}

	//判断es节点是否都还活着
	for _, cluster := range originHost.Clusters {
		for _, node := range cluster.Nodes {
			if !node.IsAlive(cluster.GetSchema(), cluster.UserName, cluster.Password, cluster.Version) {
				return true, nil
			}
		}
	}

	currentHost := &model.Host{}
	currentHost.IPs = util.GetLocalIPs()
	processInfos := getProcessInfo()
	pathPorts := getNodeConfigPaths(processInfos)
	currentClusters, err := getClusterConfigs(pathPorts)
	if err != nil {
		return false, errors.Wrap(err, "getClusterConfigs failed")
	}
	currentHost.Clusters = currentClusters
	currentHost.TLS = config.EnvConfig.TLS

	//当前主机包含的集群数量变化
	if len(currentClusters) != len(originHost.Clusters) {
		return true, nil
	}
	//节点数量变化
	currentNodeNums := 0
	for _, cluster := range currentClusters {
		currentNodeNums += len(cluster.Nodes)
	}
	originNodeNums := 0
	for _, cluster := range originHost.Clusters {
		originNodeNums += len(cluster.Nodes)
	}
	if originNodeNums != currentNodeNums {
		return true, nil
	}
	return false, nil
}

func IsRegistered() bool {
	if config.HostInfo != nil {
		if config.HostInfo.AgentID == "" {
			return false
		}
		return true
	}
	hostInfo := config.GetHostInfo()
	if hostInfo == nil {
		return false
	}
	if hostInfo.AgentID == "" {
		return false
	}
	config.HostInfo = hostInfo
	return true
}

func validatePort(ip string, schema string, clusterID string, name string, pwd string, ports []int) int {
	if ports == nil {
		return 0
	}
	if ip == "" {
		ip = "localhost"
	}
	for _, port := range ports {
		url := fmt.Sprintf("%s://%s:%d", schema, ip, port)
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
