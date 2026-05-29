/* Copyright © INFINI Ltd. All rights reserved.
* Web: https://infinilabs.com
* Email: hello#infini.ltd */

package process

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	util2 "infini.sh/agent/lib/util"
	"infini.sh/framework/core/elastic"
	log "infini.sh/framework/core/log"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
)

type esDiscoveryProbe struct {
	Schema string
	Auth   *model.BasicAuth
}

const esDiscoveryProbeTimeout = time.Second

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
	// check if we are in kubernetes environment
	var httpPort = 0
	if util2.IsKubernetes() {
		httpPort = getESNodeHttpPort()
	}
	probeHints := buildESDiscoveryProbeHints(cfgs)
	nodes := map[string]*elastic.LocalNodeInfo{}
	processInfos, err := DiscoverESProcessors(ElasticFilter)
	if err != nil {
		return nil, err
	}

	unknowProcess := []model.ProcessInfo{}
	findPIds := map[int]string{}
	for _, processInfo := range processInfos {
		//try connect
		for _, addr := range prioritizeESListenAddresses(processInfo.ListenAddresses, probeHints) {
			if httpPort > 0 && addr.Port != httpPort {
				continue // skip if port does not match in Kubernetes environment
			}
			endpoint, info, auth, err := tryGetESClusterInfo(addr, probeHints[addr.Port])
			if info != nil && info.ClusterUUID != "" {

				nodeID, nodeInfo, err := util2.GetLocalNodeInfo(endpoint, auth)
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

// getESNodeHttpPort retrieves the HTTP port for Elasticsearch nodes from the environment variable "http.port".
// If the variable is not set or contains an invalid value, it returns -1.
func getESNodeHttpPort() int {
	strPort := os.Getenv("http.port")
	if strPort == "" {
		return -1
	}
	port, err := strconv.Atoi(strPort)
	if err != nil {
		log.Error("Invalid http.port value: ", strPort)
		return -1
	}
	return port
}

var ErrUnauthorized = errors.New(http.StatusText(http.StatusUnauthorized))

func tryGetESClusterInfo(addr model.ListenAddr, preferredProbes []esDiscoveryProbe) (string, *elastic.ClusterInformation, *model.BasicAuth, error) {
	var ip = normalizeESProbeIP(addr.IP)
	var (
		lastEndpoint     string
		lastErr          error
		unauthorizedSeen bool
	)
	clusterInfo := &elastic.ClusterInformation{}
	for _, probe := range buildESDiscoveryProbeCandidates(preferredProbes) {
		lastEndpoint = fmt.Sprintf("%s://%s:%d", probe.Schema, ip, addr.Port)
		info, err := getESClusterVersion(lastEndpoint, probe.Auth)
		if err != nil {
			lastErr = err
			if errors.Is(err, ErrUnauthorized) {
				unauthorizedSeen = true
			}
			log.Debug(err)
			continue
		}
		return lastEndpoint, info, probe.Auth, nil
	}
	if unauthorizedSeen {
		return lastEndpoint, nil, nil, ErrUnauthorized
	}
	return lastEndpoint, clusterInfo, nil, lastErr
}

func normalizeESProbeIP(ip string) string {
	if ip == "*" {
		_, ip, _, _ = util.GetPublishNetworkDeviceInfo(".*")
	}
	if util.ContainStr(ip, ":") && !util.PrefixStr(ip, "[") {
		ip = fmt.Sprintf("[%s]", ip)
	}
	return ip
}

func buildESDiscoveryProbeCandidates(preferredProbes []esDiscoveryProbe) []esDiscoveryProbe {
	probes := make([]esDiscoveryProbe, 0, len(preferredProbes)+2)
	seen := map[string]struct{}{}
	appendProbe := func(probe esDiscoveryProbe) {
		schema := strings.ToLower(strings.TrimSpace(probe.Schema))
		if schema == "" {
			return
		}
		key := schema + "|" + basicAuthKey(probe.Auth)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		probes = append(probes, esDiscoveryProbe{Schema: schema, Auth: probe.Auth})
	}
	for _, probe := range preferredProbes {
		appendProbe(probe)
	}
	appendProbe(esDiscoveryProbe{Schema: "http"})
	appendProbe(esDiscoveryProbe{Schema: "https"})
	return probes
}

func getESClusterVersion(endpoint string, auth *model.BasicAuth) (*elastic.ClusterInformation, error) {
	req := util.Request{
		Method: util.Verb_GET,
		Url:    endpoint,
	}
	if auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password.Get())
	}
	ctx, cancel := context.WithTimeout(context.Background(), esDiscoveryProbeTimeout)
	defer cancel()
	req.Context = ctx
	resp, err := util.ExecuteRequest(&req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", string(resp.Body))
	}

	clusterInfo := elastic.ClusterInformation{}
	if err := util.FromJSONBytes(resp.Body, &clusterInfo); err != nil {
		return nil, err
	}
	return &clusterInfo, nil
}

func buildESDiscoveryProbeHints(cfgs []elastic.ElasticsearchConfig) map[int][]esDiscoveryProbe {
	hints := map[int][]esDiscoveryProbe{}
	for _, cfg := range cfgs {
		for _, endpoint := range getESDiscoveryConfigEndpoints(cfg) {
			target, err := url.Parse(strings.TrimSpace(endpoint))
			if err != nil || target == nil {
				continue
			}
			scheme := strings.ToLower(strings.TrimSpace(target.Scheme))
			host := strings.TrimSpace(target.Host)
			if scheme == "" || host == "" {
				continue
			}
			port := target.Port()
			if port == "" {
				switch scheme {
				case "http", "ws":
					port = "80"
				case "https", "wss":
					port = "443"
				default:
					continue
				}
			}
			portNumber, err := strconv.Atoi(port)
			if err != nil {
				continue
			}
			hints[portNumber] = appendUniqueESDiscoveryProbe(hints[portNumber], esDiscoveryProbe{
				Schema: normalizeESProbeScheme(scheme),
				Auth:   cfg.BasicAuth,
			})
		}
	}
	return hints
}

func getESDiscoveryConfigEndpoints(cfg elastic.ElasticsearchConfig) []string {
	endpoints := []string{}
	appendEndpoint := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		endpoints = append(endpoints, value)
	}
	appendEndpoint(cfg.Endpoint)
	for _, endpoint := range cfg.Endpoints {
		appendEndpoint(endpoint)
	}
	if cfg.Schema != "" {
		appendEndpoint(fmt.Sprintf("%s://%s", cfg.Schema, cfg.Host))
		for _, host := range cfg.Hosts {
			appendEndpoint(fmt.Sprintf("%s://%s", cfg.Schema, host))
		}
	}
	return endpoints
}

func normalizeESProbeScheme(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "https", "wss":
		return "https"
	default:
		return "http"
	}
}

func appendUniqueESDiscoveryProbe(items []esDiscoveryProbe, probe esDiscoveryProbe) []esDiscoveryProbe {
	key := strings.ToLower(strings.TrimSpace(probe.Schema)) + "|" + basicAuthKey(probe.Auth)
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Schema))+"|"+basicAuthKey(item.Auth) == key {
			return items
		}
	}
	return append(items, probe)
}

func basicAuthKey(auth *model.BasicAuth) string {
	if auth == nil {
		return ""
	}
	return strings.TrimSpace(auth.Username) + ":" + auth.Password.Get()
}

func prioritizeESListenAddresses(addresses []model.ListenAddr, probeHints map[int][]esDiscoveryProbe) []model.ListenAddr {
	if len(addresses) <= 1 {
		return addresses
	}
	items := make([]model.ListenAddr, 0, len(addresses))
	seen := map[string]struct{}{}
	for _, addr := range addresses {
		key := fmt.Sprintf("%s:%d", addr.IP, addr.Port)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, addr)
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := esListenAddressPriority(items[i], probeHints)
		right := esListenAddressPriority(items[j], probeHints)
		if left != right {
			return left < right
		}
		return items[i].Port < items[j].Port
	})
	return items
}

func esListenAddressPriority(addr model.ListenAddr, probeHints map[int][]esDiscoveryProbe) int {
	if len(probeHints[addr.Port]) > 0 {
		return 0
	}
	switch {
	case addr.Port == 9200:
		return 1
	case addr.Port > 9200 && addr.Port < 9300:
		return 2
	case addr.Port == 80 || addr.Port == 443:
		return 3
	case addr.Port >= 9300 && addr.Port < 9400:
		return 5
	default:
		return 4
	}
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
