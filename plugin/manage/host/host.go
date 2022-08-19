/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package host

import (
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/api"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
	"io/ioutil"
	"net/http"
	"src/github.com/buger/jsonparser"
	"strings"
)

func WindowsTest() (*model.Host, error) {
	return GetHostInfo()
}

func GetHostInfo() (*model.Host, error) {

	host := &model.Host{}
	host.IPs = util.GetLocalIPs()
	processInfos := getProcessInfo()
	pathPorts := getNodeConfigPaths(processInfos)
	clusters, err := getClusterConfigs(pathPorts)
	if err != nil {
		return nil, errors.Wrap(err, "host.getHostInfo: getClusterConfigs failed")
	}
	host.Clusters = clusters
	return host, nil
}

func RegisterHost() (*model.Host, error) {

	host, err := GetHostInfo()
	if err != nil {
		return nil, errors.Wrap(err, "host.RegisterHost: registerHost failed")
	}
	host.TLS = config.IsHTTPS()
	host.AgentPort = config.GetListenPort()

	instance := host.ToConsoleModel()
	body, err := json.Marshal(instance)
	if err != nil {
		return nil, errors.Wrap(err, "host.RegisterHost: get hostinfo failed")
	}
	log.Debugf("host.RegisterHost: request to: %s , body: %v\n", api.UrlUploadHostInfo, string(body))
	url := fmt.Sprintf("%s%s", config.GetManagerEndpoint(), api.UrlUploadHostInfo)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.Wrap(err, "host.RegisterHost: register host failed")
	}
	defer resp.Body.Close()
	bodyC, _ := ioutil.ReadAll(resp.Body)
	if strings.Contains(string(bodyC), "already exists") {
		return nil, errors.New(fmt.Sprintf("\ncurrent cluster registered\nplease delete first in console\n"))
	}
	log.Debugf("host.RegisterHost, resp: %s\n", string(bodyC))
	var registerResp *model.RegisterResponse
	util.MustFromJSONBytes(bodyC, registerResp)
	host.AgentID = registerResp.AgentId
	//if result is "acknowledged" => console receive register info, but need user review this request. if passed, console will callback from api
	if registerResp.Result == "acknowledged" {
		host.IsRunning = false
		return host, nil
	}
	return UpdateClusterInfoFromResp(host, registerResp)
}

func UpdateClusterInfoFromResp(host *model.Host, registerResp *model.RegisterResponse) (*model.Host, error) {

	for _, cluster := range host.Clusters {
		if respCluster, ok := registerResp.Clusters[cluster.Name]; ok {
			cluster.ID = respCluster.ClusterId
			cluster.UserName = respCluster.BasicAuth.Username
			cluster.Password = respCluster.BasicAuth.Password
		}
		for _, node := range cluster.Nodes {
			if node.HttpPort == 0 {
				node.HttpPort = ValidatePort(node.NetWorkHost, cluster.GetSchema(), cluster.UUID, cluster.UserName, cluster.Password, node.Ports)
			}
		}
	}
	// if clusterId is empty => this cluster not register in console => ignore
	var resultCluster []*model.Cluster
	for _, clus := range host.Clusters {
		if clus.ID != "" {
			resultCluster = append(resultCluster, clus)
		}
	}
	host.IsRunning = true
	host.Clusters = resultCluster
	return host, nil
}

func IsHostInfoChanged() (bool, error) {
	originHost := config.GetHostInfo()
	if originHost == nil {
		log.Error("host.IsHostInfoChanged: host info in kv lost")
		return true, nil
	}

	//判断es配置文件是否变化(集群名称、节点名、端口等). 任意一个节点配置文件变化，都触发更新
	for _, v := range originHost.Clusters {
		for _, node := range v.Nodes {
			currentFileContent, err := util.FileGetContent(node.ConfigPath)
			if err != nil {
				//读取文件失败，这种错误暂不处理
				log.Errorf("host.IsHostInfoChanged: es node(%s) read config file failed, path: \n%s\n", node.ID, node.ConfigPath)
				return false, nil
			}
			if !strings.EqualFold(RemoveCommentInFile(string(currentFileContent)), string(node.ConfigFileContent)) {
				log.Errorf("host.IsHostInfoChanged: es node(%s) config file changed. file path: %s\n", node.ID, node.ConfigPath)
				_ = currentFileContent
				return true, nil
			}
			_ = currentFileContent
		}
	}

	//判断es节点是否都还活着
	for _, cluster := range originHost.Clusters {
		for _, node := range cluster.Nodes {
			if !node.IsAlive(cluster.GetSchema(), cluster.UserName, cluster.Password, cluster.Version) {
				log.Debugf("host.IsHostInfoChanged: es node not alive: \nid: %s, \nname: %s, \nclusterName: %s, \nip: %s, \npath: %s\n\n", node.ID, node.Name, node.ClusterName, node.NetWorkHost, node.ConfigPath)
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
		return true, errors.Wrap(err, "host.IsHostInfoChanged: getClusterConfigs failed")
	}
	currentHost.Clusters = currentClusters
	currentHost.TLS = config.IsHTTPS()

	//TODO 当前主机包含的集群数量变化。 如果有一个集群，用户并不想注册到console，那这里会一直检测到有新集群。
	if len(currentClusters) != len(originHost.Clusters) {
		log.Debugf("host.IsHostInfoChanged: cluster total number changed")
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
		log.Debugf("host.IsHostInfoChanged: es node total number changed")
		return true, nil
	}
	return false, nil
}

func IsRegistered() bool {
	if config.GetHostInfo() != nil {
		if config.GetHostInfo().AgentID == "" {
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
	return true
}

func ValidatePort(ip string, schema string, clusterID string, name string, pwd string, ports []int) int {
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
			log.Errorf("%v", err)
			continue
		}
		clusterUuid, _ := jsonparser.GetString(result.Body, "cluster_uuid")
		if strings.EqualFold(clusterUuid, clusterID) {
			return port
		}
	}
	return 0
}
