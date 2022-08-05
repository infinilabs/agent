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
	"infini.sh/framework/core/kv"
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
	fmt.Printf("register agent: %v", string(body))
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
	fmt.Printf("response : %s", bodyC)

	//result := make(map[string]interface{})
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
				node.HttpPort = validatePort(node.NetWorkHost, cluster.UUID, cluster.UserName, cluster.Password, node.Ports)
			}
		}
	}
	return host, nil
}

func IsRegistered() bool {
	hostInfo := getAgentHostInfo()
	if hostInfo == nil {
		return false
	}
	if hostInfo.AgentID == "" {
		return false
	}
	return true
}

func validatePort(ip string, clusterID string, name string, pwd string, ports []int) int {
	if ports == nil {
		return 0
	}
	if ip == "" {
		ip = "localhost"
	}
	for _, port := range ports {
		url := fmt.Sprintf("http://%s:%d", ip, port)
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

func SaveAgentHostInfo(host *model.Host) error {
	byteInfo, err := json.Marshal(host)
	if err != nil {
		return errors.Wrap(err, "save agent host info failed")
	}
	err = kv.AddValue(config.KVAgentBucket, []byte(config.KVHostInfo), byteInfo)
	if err != nil {
		return errors.Wrap(err, "save agent host info failed")
	}
	return nil
}

func getAgentHostInfo() *model.Host {
	val, err := kv.GetValue(config.KVAgentBucket, []byte(config.KVHostInfo))
	if err != nil {
		return nil
	}
	var host *model.Host
	json.Unmarshal(val, &host)
	return host
}
