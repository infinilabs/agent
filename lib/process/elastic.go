/* Copyright © INFINI Ltd. All rights reserved.
* Web: https://infinilabs.com
* Email: hello#infini.ltd */

package process

import (
	"context"
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
	"time"
)

func DiscoverESNodeFromEndpoint(endpoint string, auth *model.BasicAuth) (*elastic.LocalNodeInfo, error) {
	localNodeInfo := elastic.LocalNodeInfo{}
	var (
		nodeInfo *elastic.NodesInfo
		err      error
		nodeID   string
	)

	nodeID, nodeInfo, err = util2.GetLocalNodeInfo(endpoint, auth)
	if err != nil {
		return nil, fmt.Errorf("get node info error: %w", err)
	}

	clusterInfo, err := util2.GetClusterVersion(endpoint, auth)
	if err != nil {
		return nil, fmt.Errorf("get cluster info error: %w", err)
	}

	localNodeInfo.NodeInfo = nodeInfo
	localNodeInfo.ClusterInfo = clusterInfo
	localNodeInfo.NodeUUID = nodeID

	return &localNodeInfo, nil
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
	nodes := map[string]*elastic.LocalNodeInfo{}
	processInfos, err := DiscoverESProcessors(ElasticFilter)
	if err != nil {
		return nil, err
	}

	unknowProcess := []model.ProcessInfo{}
	findPIds := map[int]string{}
	for _, processInfo := range processInfos {
		//try connect
		for _, addr := range processInfo.ListenAddresses {
			endpoint, info, err := tryGetESClusterInfo(addr)
			if info != nil && info.ClusterUUID != "" {

				nodeID, nodeInfo, err := util2.GetLocalNodeInfo(endpoint, nil)
				if err != nil {
					log.Error(err)
					continue
				}

				if nodeInfo.Process.Id == processInfo.PID {
					localNodeInfo := elastic.LocalNodeInfo{}
					localNodeInfo.NodeInfo = nodeInfo
					localNodeInfo.ClusterInfo = info
					localNodeInfo.NodeUUID = nodeID
					nodes[localNodeInfo.NodeUUID] = &localNodeInfo
					findPIds[localNodeInfo.NodeInfo.Process.Id] = localNodeInfo.NodeUUID
				}
				break
			}
			if err == ErrUnauthorized {
				unknowProcess = append(unknowProcess, processInfo)
				break
			}
		}
	}

	newProcess := []model.ProcessInfo{}
	for _, process := range unknowProcess {
		if _, ok := findPIds[process.PID]; !ok {
			newProcess = append(newProcess, process)
		}
	}

	result := elastic.DiscoveryResult{
		Nodes:          nodes,
		UnknownProcess: newProcess,
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
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		req := &util.Request{
			Method:  util.Verb_GET,
			Url:     endpoint,
			Context: ctx,
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
