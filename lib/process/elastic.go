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
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func DiscoverESNodeFromEndpoint(config elastic.ElasticsearchConfig) (*elastic.ClusterInformation,string, *elastic.NodesInfo, error) {
	var (
		nodeInfo *elastic.NodesInfo
		err      error
		nodeID   string
	)
	nodeID, nodeInfo, err = util2.GetLocalNodeInfo(config.GetAnyEndpoint(), config.BasicAuth)
	if err != nil {
		return nil, "",nil, fmt.Errorf("get nodes error: %w", err)
	}
	clusterInfo, err := util2.GetClusterVersion(config.GetAnyEndpoint(), config.BasicAuth)
	if err != nil {
		return nil, "",nil, fmt.Errorf("get cluster info error: %w", err)
	}
	return clusterInfo,nodeID, nodeInfo, nil
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

func getListenAddresses(boundAddresses []string) []model.ListenAddr {
	var listenAddresses []model.ListenAddr
	for _, boundAddr := range boundAddresses {
		if idx := strings.LastIndex(boundAddr, ":"); idx > -1 {
			addr := model.ListenAddr{
				IP: boundAddr[:idx],
			}
			if idx < len(boundAddr)-1 {
				addr.Port, _ = strconv.Atoi(boundAddr[idx+1:])
			}
			listenAddresses = append(listenAddresses, addr)
		}
	}
	return listenAddresses
}

func DiscoverESNode(cfgs []elastic.ElasticsearchConfig) (*elastic.DiscoveryResult, error) {
	nodes := map[string]elastic.LocalNodeInfo{}
	processInfos, err := DiscoverESProcessors(ElasticFilter)
	if err != nil {
		return nil, err
	}

	unknowProcess := []model.ProcessInfo{}
	for _, processInfo := range processInfos {
		//try connect
		for _, addr := range processInfo.ListenAddresses {
			endpoint, info, err := tryGetESClusterInfo(addr)
			if info != nil && info.ClusterUUID != "" {
				cfg := elastic.ElasticsearchConfig{
					Endpoint:    endpoint,
					Enabled:     true,
					ClusterUUID: info.ClusterUUID,
					Name:        info.ClusterName,
					Version:     info.Version.Number,
				}
				cfg.ID = info.ClusterUUID
				cfgs = append(cfgs, cfg)
				break
			}
			if err == ErrUnauthorized {
				unknowProcess = append(unknowProcess, processInfo)
				break
			}
		}
	}

	for _, cfg := range cfgs {
		if cfg.Enabled {
			cluster, nodeID,node, err := DiscoverESNodeFromEndpoint(cfg)
			if err != nil {
				continue
			}
			localNodeInfo := elastic.LocalNodeInfo{
				ClusterInfo: cluster,
				NodeInfo:    node,
			}
			nodes[nodeID] = localNodeInfo
		}
	}

	result:=elastic.DiscoveryResult{
		Nodes: nodes,
		UnknownProcess: unknowProcess,
	}

	return &result, nil
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

		if util.ContainStr(ip, ":") && !util.PrefixStr(ip, "[") {
			ip = fmt.Sprintf("[%s]", ip)
		}

		endpoint = fmt.Sprintf("%s://%s:%d", schema, ip, addr.Port)
		req := &util.Request{
			Method: util.Verb_GET,
			Url:    endpoint,
		}
		result, err := util.ExecuteRequest(req)
		if err != nil {
			if !strings.Contains(err.Error(), "transport connection broken") && !strings.Contains(err.Error(), "EOF") {
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

func parseNodeInfoFromCmdline(cmdline string) (pathHome, pathConfig string, err error) {
	pathHome, err = parsePathValue(cmdline, `\-Des\.path\.home=(.*?)\s+`)
	if err != nil {
		return
	}
	pathConfig, err = parsePathValue(cmdline, `\-Des\.path\.conf=(.*?)\s+`)
	return pathHome, pathConfig, err
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
