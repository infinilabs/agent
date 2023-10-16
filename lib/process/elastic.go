/* Copyright © INFINI Ltd. All rights reserved.
* Web: https://infinilabs.com
* Email: hello#infini.ltd */

package process

import (
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	util2 "infini.sh/agent/lib/util"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func DiscoverESNodeFromEndpoint(config elastic.ElasticsearchConfig) (*model.ESNodeInfo, error){
	var (
		nodeID string
		nodeInfo *elastic.NodesInfo
		err error
	)
	nodeID, nodeInfo, err = util2.GetLocalNodeInfo(config.GetAnyEndpoint(), config.BasicAuth)
	if err != nil {
		return nil, fmt.Errorf("get nodes error: %w", err)
	}
	clusterInfo, err := util2.GetClusterVersion(config.GetAnyEndpoint(), config.BasicAuth)
	if err != nil {
		return nil, fmt.Errorf("get cluster info error: %w", err)
	}

	settings := util.MapStr(nodeInfo.Settings)
	homePath, _ := settings.GetValue("path.home")
	logsPath, _ := settings.GetValue("path.logs")
	dataPath, _ := settings.GetValue("path.data")

	var (
		port string
		schema string
	)
	tempurl, _ := url.Parse(config.GetAnyEndpoint())
	port = tempurl.Port()
	_, rport, err := net.SplitHostPort(nodeInfo.Http.PublishAddress)
	if err != nil {
		log.Error(err)
	}else{
		port = rport
	}
	schema = getNodeSchema(tempurl.Scheme, nodeInfo.Http.PublishAddress, config.BasicAuth)


	esNode := model.ESNodeInfo{
		AgentID: global.Env().SystemConfig.NodeConfig.ID,
		ClusterUuid: clusterInfo.ClusterUUID,
		ClusterName: clusterInfo.ClusterName,
		NodeUUID: nodeID,
		NodeName: nodeInfo.Name,
		Version: nodeInfo.Version,
		Timestamp: time.Now().UnixMilli(),
		PublishAddress: nodeInfo.GetHttpPublishHost(),
		HttpPort: port,
		Schema: schema,
		Status: "online",
		ProcessInfo: model.ProcessInfo{
			//ListenAddresses: listenAddresses,
		},
	}
	if v, ok := nodeInfo.Process["id"].(float64); ok {
		esNode.ProcessInfo.PID = int(v)
	}
	if v, ok := homePath.(string); ok {
		esNode.Path.Home = v
	}
	if v, ok := logsPath.(string); ok {
		esNode.Path.Logs = v
	}
	if v, ok := dataPath.(string); ok {
		esNode.Path.Data = v
	}

	return &esNode, nil
}

func getNodeSchema(schema, pubAddr string, auth *model.BasicAuth) string {
	url := fmt.Sprintf("%s://%s", schema, pubAddr)
	_, err := util2.GetClusterVersion(url, auth)
	if err != nil {
		log.Debug(err)
		if schema == "http" {
			return "https"
		}
		return "http"
	}
	return schema
}

func getListenAddresses(boundAddresses []string) []model.ListenAddr{
	var listenAddresses []model.ListenAddr
	for _, boundAddr := range boundAddresses {
		if idx := strings.LastIndex(boundAddr, ":"); idx > -1 {
			addr := model.ListenAddr{
				IP: boundAddr[:idx],
			}
			if idx < len(boundAddr) - 1 {
				addr.Port, _ = strconv.Atoi(boundAddr[idx+1:])
			}
			listenAddresses = append(listenAddresses, addr)
		}
	}
	return listenAddresses
}


func DiscoverESNode(cfgs []elastic.ElasticsearchConfig) (map[string]model.ESNodeInfo, error){
	nodes := map[string]model.ESNodeInfo{}
	for _, cfg := range cfgs {
		if cfg.Enabled {
			node, err := DiscoverESNodeFromEndpoint(cfg)
			if err != nil {
				continue
			}
			nodes[node.NodeUUID] = *node
		}
	}
	processInfos, err := Discover(ElasticFilter)
	if err != nil {
		return nil, err
	}
	localNodes := map[string]model.ESNodeInfo{}
	var cfgsFromProcess []elastic.ElasticsearchConfig
	for _, processInfo := range processInfos {
		if nodeID, exists := isProcessExists(processInfo.PID, nodes); exists {
			node := nodes[nodeID]
			node.ProcessInfo = processInfo
			localNodes[nodeID] = node
			continue
		}
		//try connect
		for _, addr := range processInfo.ListenAddresses {
			endpoint, info, err := tryGetESClusterInfo(addr)
			if info != nil && info.ClusterUUID != "" {
				cfg := elastic.ElasticsearchConfig{
					Endpoint: endpoint,
					Enabled: true,
					ClusterUUID: info.ClusterUUID,
					Name: info.ClusterName,
					Version: info.Version.Number,
				}
				cfg.ID = info.ClusterUUID
				cfgsFromProcess = append(cfgsFromProcess, cfg)
				break
			}
			if err == ErrUnauthorized {
				tempUrl, _ := url.Parse(endpoint)
				esNode := model.ESNodeInfo{
					Timestamp: time.Now().UnixMilli(),
					PublishAddress: tempUrl.Host,
					Schema: tempUrl.Scheme,
					Status: "online",
					HttpPort: tempUrl.Port(),
					ProcessInfo: processInfo,
				}
				err = parseNodeInfoFromCmdline(processInfo.Cmdline, &esNode)
				if err != nil {
					log.Debug(err)
				}
				localNodes[tempUrl.Port()] = esNode

				break
			}
		}

	}
	for _, cfg := range cfgsFromProcess {
		if cfg.Enabled {
			node, err := DiscoverESNodeFromEndpoint(cfg)
			if err != nil {
				log.Error(err)
				continue
			}
			node.ProcessInfo = processInfos[node.ProcessInfo.PID]
			localNodes[node.NodeUUID] = *node
		}
	}
	return localNodes, nil
}

func isProcessExists(pid int, nodes map[string]model.ESNodeInfo) (string, bool) {
	for _, node := range nodes {
		if node.ProcessInfo.PID == pid {
			return  node.NodeUUID, true
		}
	}
	return "", false
}

var ErrUnauthorized = errors.New(http.StatusText(http.StatusUnauthorized))

func tryGetESClusterInfo(addr model.ListenAddr) (string, *elastic.ClusterInformation, error) {
	var ip = addr.IP
	if ip == "*" {
		_, ip, _, _ = util.GetPublishNetworkDeviceInfo(".*")
	}
	schemas := []string{"http", "https"}
	clusterInfo := &elastic.ClusterInformation{}
	var endpoint string
	for _, schema := range schemas {

		if util.ContainStr(ip,":") && !util.PrefixStr(ip, "["){
			ip = fmt.Sprintf("[%s]", ip)
		}

		endpoint = fmt.Sprintf("%s://%s:%d", schema, ip, addr.Port)
		req := &util.Request{
			Method: util.Verb_GET,
			Url: endpoint,
		}
		result, err := util.ExecuteRequest(req)
		if err != nil {
			if !strings.Contains(err.Error(), "transport connection broken") && !strings.Contains(err.Error(), "EOF"){
				return endpoint, nil, err
			}
			log.Debug(err)
			continue
		}
		if result.StatusCode == http.StatusUnauthorized {
			return endpoint, nil, ErrUnauthorized
		}

		err = util.FromJSONBytes(result.Body, &clusterInfo)
		if err == nil {
			return endpoint, clusterInfo, err
		}
	}
	return endpoint, clusterInfo, nil
}

func parseNodeInfoFromCmdline(cmdline string, nodeInfo *model.ESNodeInfo) (err error) {
	nodeInfo.Path.Home, err = parsePathValue(cmdline, `\-Des\.path\.home=(.*?)\s+`)
	if err != nil {
		return
	}
	nodeInfo.Path.Config, err = parsePathValue(cmdline, `\-Des\.path\.conf=(.*?)\s+`)
	return err
}

func parsePathValue(cmdline string, regStr string) (string, error) {
	reg, err := regexp.Compile(regStr)
	if err != nil {
		return "", err
	}
	matches := reg.FindStringSubmatch(cmdline)
	if len(matches) > 1 {
		return matches[1], nil
	}
	return "", nil
}