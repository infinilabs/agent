/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// launchEasysearch spawns the Easysearch daemon process. It is used by both
// stepStartEasysearch (during creation) and startService (manual restart).
func launchEasysearch(ctx context.Context, s *service) error {
	// Resolve Easysearch home before spawning the child process. esHome already
	// returns an absolute path (derived from AbsoluteAssetsDirPath), so no
	// further filepath.Abs wrapping is needed.
	home, err := s.absoluteEasysearchHome()
	if err != nil {
		return fmt.Errorf("resolve easysearch home: %w", err)
	}
	// Resolve the workspace path as well so startup artifacts like --pidfile and
	// redirected stdout are written to the task workspace itself instead of being
	// re-resolved relative to Easysearch's working directory.
	ws, err := s.AbsoluteWorkspacePath()
	if err != nil {
		return err
	}
	pidFile := filepath.Join(ws, "easysearch.pid")
	stdoutLog := filepath.Join(ws, "easysearch_stdout.log")

	binary := filepath.Join(home, "bin", "easysearch")
	// Let Easysearch launch itself in daemon mode so it detaches from the
	// agent/terminal session and owns the pid file lifecycle itself.
	cmd := exec.CommandContext(ctx, binary, "-d", "--pidfile", pidFile)
	cmd.Dir = home

	logFile, err := os.OpenFile(stdoutLog,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("create stdout log: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start easysearch: %w", err)
	}
	logFile.Close()
	_ = cmd.Process.Release()

	log.Infof("[setup] easysearch daemon launched, pid file: %s, stdout log: %s", pidFile, stdoutLog)
	return nil
}

// killEasysearch reads the PID file in the workspace and terminates the
// Easysearch process gracefully, escalating to SIGKILL if needed.
//
// Strategy:
//  1. Send SIGINT once and poll every 500 ms for up to 10 s.
//  2. If still alive, send SIGKILL once and poll every 500 ms for up to 5 s.
//  3. If still alive after SIGKILL, the process is likely stuck in an
//     uninterruptible sleep (D-state, e.g. NFS hang) — log a warning and
//     return an error so the caller can surface the problem.
func killEasysearch(s *service) error {
	ws, err := s.AbsoluteWorkspacePath()
	if err != nil {
		return fmt.Errorf("resolve workspace path: %w", err)
	}
	pidFile := filepath.Join(ws, "easysearch.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return nil // PID file absent — nothing to kill
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID %q: %w", pidStr, err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil // Process not found — already gone
	}

	stopped := func(rounds int) bool {
		for range rounds {
			time.Sleep(500 * time.Millisecond)
			if !isProcessAlive(proc) {
				return true
			}
		}
		return false
	}

	// Phase 1: graceful shutdown via SIGINT.
	// It has 10 seconds to stop
	_ = proc.Signal(os.Interrupt)
	if stopped(20) {
		os.Remove(pidFile)
		log.Infof("[setup] easysearch process %d exited after SIGINT", pid)
		return nil
	}

	// Phase 2: forceful shutdown via SIGKILL.
	_ = proc.Kill()
	if stopped(10) {
		os.Remove(pidFile)
		log.Infof("[setup] easysearch process %d exited after SIGKILL", pid)
		return nil
	}

	log.Warnf("[setup] easysearch process %d did not exit after SIGKILL; it may be stuck in uninterruptible sleep (D-state)", pid)
	return fmt.Errorf("process %d did not exit after SIGKILL", pid)
}

// isEasysearchRunning reports whether the process recorded in pidFile is alive.
func isEasysearchRunning(pidFile string) bool {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; sending signal 0 checks liveness.
	return isProcessAlive(proc)
}
