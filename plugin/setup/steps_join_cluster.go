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
	"time"

	log "github.com/cihub/seelog"
)

// helper: resolve joinClusterConfig from a service.
func joinClusterCfg(svc *service) *joinClusterConfig {
	return svc.config.(*joinClusterConfig)
}

// ====================== Step: WriteJoinCerts =================================

// joinCertFiles maps enroll response field names to on-disk filenames.
// We always write all 6 files, regardless of security.enabled, so the node
// has cert material ready if security is later toggled on.
var joinCertFiles = []string{
	"http_ca.crt", "http_instance.crt", "http_instance.key",
	"transport_ca.crt", "transport_instance.crt", "transport_instance.key",
}

type stepWriteJoinCerts struct{}

func (s *stepWriteJoinCerts) IsAssetStep() bool { return true }

func (s *stepWriteJoinCerts) NameI18nKey() string { return "stepSetupCertificates" }

func (s *stepWriteJoinCerts) Execute(_ context.Context, svc *service) error {
	cfg := joinClusterCfg(svc)
	if cfg.FetchedEnrollInfo == nil {
		return fmt.Errorf("missing FetchedEnrollInfo")
	}

	sec := cfg.FetchedEnrollInfo.Security
	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, "config")

	// Map logical cert fields → file names.
	fileMap := map[string]string{
		"http_ca.crt":            sec.HTTPCaKey,
		"http_instance.crt":      sec.HTTPCert,
		"http_instance.key":      sec.HTTPKey,
		"transport_ca.crt":       sec.TransportCaCert,
		"transport_instance.crt": sec.TransportCert,
		"transport_instance.key": sec.TransportKey,
	}

	written := 0
	for name, content := range fileMap {
		if content == "" {
			continue
		}
		if err := os.WriteFile(filepath.Join(configDir, name), []byte(content), 0600); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
		written++
	}

	log.Infof("[setup] wrote %d certificate files from enroll response", written)
	return nil
}

func (s *stepWriteJoinCerts) Rollback(svc *service) error {
	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, "config")
	for _, name := range joinCertFiles {
		os.Remove(filepath.Join(configDir, name))
	}
	return nil
}

// ================== Step: GenerateJoinClusterConfig ===========================

type stepGenerateJoinClusterConfig struct{}

func (s *stepGenerateJoinClusterConfig) IsAssetStep() bool { return true }

func (s *stepGenerateJoinClusterConfig) NameI18nKey() string { return "stepGenerateConfig" }

func (s *stepGenerateJoinClusterConfig) Execute(ctx context.Context, svc *service) error {
	cfg := joinClusterCfg(svc)
	if cfg.FetchedEnrollInfo == nil {
		return fmt.Errorf("missing FetchedEnrollInfo")
	}
	info := cfg.FetchedEnrollInfo

	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return err
	}
	ymlPath := filepath.Join(home, "config", "easysearch.yml")

	// Back up original for rollback.
	backup := ymlPath + ".orig"
	if data, err := os.ReadFile(ymlPath); err == nil {
		_ = os.WriteFile(backup, data, 0644)
	}

	fields := map[string]interface{}{
		"cluster.name":         info.ClusterName,
		"node.name":            cfg.NodeName,
		"network.host":         cfg.Host,
		"http.port":            cfg.HTTPPort,
		"transport.port":       cfg.TransportPort,
		"discovery.seed_hosts": info.SeedAddresses,
	}

	if len(cfg.Roles) > 0 {
		fields["node.roles"] = cfg.Roles
	}

	if cfg.DataDirectory != "" {
		fields["path.data"] = cfg.DataDirectory
	}
	if cfg.LogDirectory != "" {
		fields["path.logs"] = cfg.LogDirectory
	}

	// Security configuration — mirrors what the existing cluster uses.
	sec := info.Security
	if sec.Enabled {
		fields["security.enabled"] = true
	}
	if sec.SSLTransportEnabled {
		fields["security.ssl.transport.enabled"] = true
		fields["security.ssl.transport.ca_file"] = "transport_ca.crt"
		fields["security.ssl.transport.cert_file"] = "transport_instance.crt"
		fields["security.ssl.transport.key_file"] = "transport_instance.key"
	}
	if sec.SSLHTTPEnabled {
		fields["security.ssl.http.enabled"] = true
		fields["security.ssl.http.ca_file"] = "http_ca.crt"
		fields["security.ssl.http.cert_file"] = "http_instance.crt"
		fields["security.ssl.http.key_file"] = "http_instance.key"
	}

	for key, val := range fields {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := UpdateConfigField(ymlPath, key, val); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}

	// Generate heap options file.
	heapDir := filepath.Join(home, "config", "jvm.options.d")
	if err := os.MkdirAll(heapDir, 0755); err != nil {
		return fmt.Errorf("create jvm.options.d: %w", err)
	}
	heapGB := int(cfg.JVMMemory)
	heapContent := fmt.Sprintf("-Xms%dg\n-Xmx%dg\n", heapGB, heapGB)
	heapFile := filepath.Join(heapDir, "heap.options")
	if err := os.WriteFile(heapFile, []byte(heapContent), 0644); err != nil {
		return fmt.Errorf("write heap.options: %w", err)
	}

	log.Infof("[setup] join-cluster config generated: cluster=%s node=%s host=%s http=%d transport=%d heap=%dg",
		info.ClusterName, cfg.NodeName, cfg.Host, cfg.HTTPPort, cfg.TransportPort, heapGB)
	return nil
}

func (s *stepGenerateJoinClusterConfig) Rollback(svc *service) error {
	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return err
	}
	ymlPath := filepath.Join(home, "config", "easysearch.yml")
	backup := ymlPath + ".orig"

	if data, err := os.ReadFile(backup); err == nil {
		_ = os.WriteFile(ymlPath, data, 0644)
		os.Remove(backup)
	}

	os.Remove(filepath.Join(home, "config", "jvm.options.d", "heap.options"))
	return nil
}

// ==================== Step: InstallJoinPlugins ================================

type stepInstallJoinPlugins struct{}

func (s *stepInstallJoinPlugins) IsAssetStep() bool { return true }

func (s *stepInstallJoinPlugins) NameI18nKey() string { return "stepInstallPlugins" }

func (s *stepInstallJoinPlugins) Execute(ctx context.Context, svc *service) error {
	cfg := joinClusterCfg(svc)
	if cfg.FetchedEnrollInfo == nil {
		return fmt.Errorf("missing FetchedEnrollInfo")
	}

	plugins := cfg.FetchedEnrollInfo.Plugins
	if len(plugins) == 0 {
		return nil
	}

	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return fmt.Errorf("resolve easysearch home: %w", err)
	}
	pluginBin := filepath.Join(home, "bin", "easysearch-plugin")

	for _, name := range plugins {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		log.Infof("[setup] installing plugin: %s", name)
		cmd := exec.CommandContext(ctx, pluginBin, "install", "--batch", name)
		cmd.Dir = home
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("install plugin %q: %w\nOutput: %s", name, err, string(output))
		}
	}

	return nil
}

func (s *stepInstallJoinPlugins) Rollback(svc *service) error {
	cfg := joinClusterCfg(svc)
	if cfg.FetchedEnrollInfo == nil {
		return nil
	}

	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return fmt.Errorf("resolve easysearch home: %w", err)
	}
	pluginBin := filepath.Join(home, "bin", "easysearch-plugin")

	for _, name := range cfg.FetchedEnrollInfo.Plugins {
		cmd := exec.Command(pluginBin, "remove", name)
		cmd.Dir = home
		_ = cmd.Run()
	}
	return nil
}

// ==================== Step: WaitForNodeJoin ===================================

type stepWaitForNodeJoin struct{}

func (s *stepWaitForNodeJoin) IsAssetStep() bool { return false }

func (s *stepWaitForNodeJoin) NameI18nKey() string { return "stepWaitForNodeJoin" }

func (s *stepWaitForNodeJoin) Execute(ctx context.Context, svc *service) error {
	cfg := joinClusterCfg(svc)
	if cfg.FetchedEnrollInfo == nil {
		return fmt.Errorf("missing FetchedEnrollInfo")
	}
	info := cfg.FetchedEnrollInfo

	// Poll the existing cluster (not the local node) to verify the new node
	// appears in the cluster membership.
	endpoint := info.Endpoint
	accessToken := info.ResponseAccessToken

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	const maxWait = 180 * time.Second
	const interval = 5 * time.Second
	deadline := time.Now().Add(maxWait)

	url := endpoint + "/_nodes/_all/name,roles"

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("node %q did not join cluster within %v", cfg.NodeName, maxWait)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("X-API-TOKEN", accessToken)

		resp, err := client.Do(req)
		if err == nil {
			body, _ := readBodyLimited(resp.Body, 1<<20)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				found, err := nodeExistsInResponse(body, cfg.NodeName)
				if err == nil && found {
					log.Infof("[setup] node %q successfully joined the cluster", cfg.NodeName)
					return nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func (s *stepWaitForNodeJoin) Rollback(_ *service) error {
	return nil
}

// nodeExistsInResponse checks whether nodeName appears in a
// GET /_nodes/_all/name,roles response body.
//
// Response shape: {"nodes": {"<id>": {"name": "...", "roles": [...]}}}
func nodeExistsInResponse(body []byte, nodeName string) (bool, error) {
	var resp struct {
		Nodes map[string]struct {
			Name string `json:"name"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return false, err
	}
	for _, n := range resp.Nodes {
		if n.Name == nodeName {
			return true, nil
		}
	}
	return false, nil
}

// readBodyLimited reads up to limit bytes from r.
func readBodyLimited(r io.ReadCloser, limit int64) ([]byte, error) {
	defer r.Close()
	lr := io.LimitReader(r, limit)
	return io.ReadAll(lr)
}
