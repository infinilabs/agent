/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"
)

// esLocalAuth holds credentials for querying the local Easysearch node.
type esLocalAuth struct {
	username string
	password string
	// apiToken is sent as X-API-TOKEN; used for join-cluster nodes where
	// the node inherits the existing cluster's security setup.
	apiToken string
}

func newLocalESClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			// TODO: supply the CA cert once the install steps store it on disk,
			// and remove InsecureSkipVerify.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
		},
	}
}

func esGet(client *http.Client, url string, auth esLocalAuth) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if auth.apiToken != "" {
		req.Header.Set("X-API-TOKEN", auth.apiToken)
	} else if auth.username != "" {
		req.SetBasicAuth(auth.username, auth.password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

func queryClusterVersion(client *http.Client, endpoint string, auth esLocalAuth) (string, error) {
	body, err := esGet(client, endpoint+"/", auth)
	if err != nil {
		return "", err
	}
	var resp struct {
		Version struct {
			Number string `json:"number"`
		} `json:"version"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	return resp.Version.Number, nil
}

func queryClusterStatus(client *http.Client, endpoint string, auth esLocalAuth) (string, error) {
	body, err := esGet(client, endpoint+"/_cluster/health", auth)
	if err != nil {
		return "", err
	}
	var resp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	return resp.Status, nil
}

type nodeStatsResult struct {
	id                 string
	name               string
	roles              []string
	cpuPercent         float64
	jvmHeapMaxGB       float64
	jvmHeapUsedPercent float64
	diskUsedPercent    float64
}

func queryLocalNodeStats(client *http.Client, endpoint string, auth esLocalAuth) (nodeStatsResult, error) {
	var zero nodeStatsResult
	body, err := esGet(client, endpoint+"/_nodes/_local/stats", auth)
	if err != nil {
		return zero, err
	}

	// Response shape: {"nodes": {"<node_id>": { ... }}}
	var resp struct {
		Nodes map[string]struct {
			Name  string   `json:"name"`
			Roles []string `json:"roles"`
			OS    struct {
				CPU struct {
					Percent float64 `json:"percent"`
				} `json:"cpu"`
			} `json:"os"`
			JVM struct {
				Mem struct {
					HeapMaxInBytes  int64 `json:"heap_max_in_bytes"`
					HeapUsedInBytes int64 `json:"heap_used_in_bytes"`
				} `json:"mem"`
			} `json:"jvm"`
			FS struct {
				Total struct {
					TotalInBytes     int64 `json:"total_in_bytes"`
					AvailableInBytes int64 `json:"available_in_bytes"`
				} `json:"total"`
			} `json:"fs"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return zero, err
	}
	if len(resp.Nodes) == 0 {
		return zero, fmt.Errorf("_nodes/_local/stats returned no nodes")
	}

	// _local always returns exactly one entry.
	var r nodeStatsResult
	for id, n := range resp.Nodes {
		r.id = id
		r.name = n.Name
		r.roles = n.Roles
		r.cpuPercent = n.OS.CPU.Percent
		if n.JVM.Mem.HeapMaxInBytes > 0 {
			r.jvmHeapMaxGB = float64(n.JVM.Mem.HeapMaxInBytes) / (1 << 30)
			r.jvmHeapUsedPercent = float64(n.JVM.Mem.HeapUsedInBytes) /
				float64(n.JVM.Mem.HeapMaxInBytes) * 100
		}
		total := n.FS.Total.TotalInBytes
		avail := n.FS.Total.AvailableInBytes
		if total > 0 {
			r.diskUsedPercent = float64(total-avail) / float64(total) * 100
		}
		break
	}
	return r, nil
}

// buildLocalEndpoint returns the base URL for the locally running node.
func buildLocalEndpoint(host string, port int, useTLS bool) string {
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

// isHTTPS reports whether an endpoint URL uses the https scheme.
func isHTTPS(endpoint string) bool {
	return strings.HasPrefix(strings.ToLower(endpoint), "https://")
}

// buildClusterNodeInfo queries the locally running Easysearch node and
// populates a clusterNodeInfo from the service config.
func buildClusterNodeInfo(svc *service) (*clusterNodeInfo, error) {
	cfg := svc.config

	var (
		clusterName    string
		clusterAddress = cfg.GetHost()
		clusterPort    = cfg.GetHTTPPort()
		auth           esLocalAuth
		useTLS         bool
	)

	switch c := cfg.(type) {
	case *NewClusterConfig:
		clusterName = c.ClusterName
		useTLS = c.EnableSecurity
		auth = esLocalAuth{username: "admin", password: c.AdminPassword}

	case *joinClusterConfig:
		if c.FetchedEnrollInfo == nil {
			return nil, fmt.Errorf("join-cluster service is missing FetchedEnrollInfo")
		}
		clusterName = c.FetchedEnrollInfo.ClusterName
		useTLS = isHTTPS(c.FetchedEnrollInfo.Endpoint)
		auth = esLocalAuth{apiToken: c.FetchedEnrollInfo.ResponseAccessToken}
	}

	endpoint := buildLocalEndpoint(clusterAddress, clusterPort, useTLS)
	client := newLocalESClient()

	version, err := queryClusterVersion(client, endpoint, auth)
	if err != nil {
		log.Warnf("[setup] %s: get cluster version: %v", svc.ID(), err)
	}
	status, err := queryClusterStatus(client, endpoint, auth)
	if err != nil {
		log.Warnf("[setup] %s: get cluster health: %v", svc.ID(), err)
	}
	ns, err := queryLocalNodeStats(client, endpoint, auth)
	if err != nil {
		log.Warnf("[setup] %s: get node stats: %v", svc.ID(), err)
	}

	return &clusterNodeInfo{
		ClusterName:       clusterName,
		ClusterStatus:     status,
		ClusterVersion:    version,
		ClusterAddress:    clusterAddress,
		ClusterPort:       clusterPort,
		NodeName:          ns.name,
		NodeID:            ns.id,
		NodeRoles:         ns.roles,
		CPUUtilization:    ns.cpuPercent,
		JVMMemoryCapacity: ns.jvmHeapMaxGB,
		DiskUsage:         ns.diskUsedPercent,
		JVMUsage:          ns.jvmHeapUsedPercent,
	}, nil
}
