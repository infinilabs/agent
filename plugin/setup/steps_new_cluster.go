/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	log "github.com/cihub/seelog"
	"golang.org/x/crypto/bcrypt"
)

// helper: resolve NewClusterConfig from a service.
func newClusterCfg(svc *service) *NewClusterConfig {
	return svc.config.(*NewClusterConfig)
}

// ==================== Step 7: SetupCertificates ==============================

// autoCertFiles are written when mode=auto (shared for http + transport).
var autoCertFiles = []string{"ca.crt", "instance.crt", "instance.key"}

// manualCertFiles are written when mode=manual (separate per layer).
var manualCertFiles = []string{
	"http_ca.crt", "http_instance.crt", "http_instance.key",
	"transport_ca.crt", "transport_instance.crt", "transport_instance.key",
}

type stepSetupCertificates struct{}

func (s *stepSetupCertificates) IsAssetStep() bool { return true }

func (s *stepSetupCertificates) NameI18nKey() string { return "stepSetupCertificates" }

func (s *stepSetupCertificates) Execute(_ context.Context, svc *service) error {
	cfg := newClusterCfg(svc)
	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, "config")

	switch cfg.CertConfig.Mode {
	case CertificateModeAuto:
		bundle, err := generateCertBundle()
		if err != nil {
			return fmt.Errorf("generate certificates: %w", err)
		}
		fileMap := map[string][]byte{
			"ca.crt":       bundle.CACertPEM,
			"instance.crt": bundle.NodeCertPEM,
			"instance.key": bundle.NodeKeyPEM,
		}
		for name, data := range fileMap {
			if err := os.WriteFile(filepath.Join(configDir, name), data, 0600); err != nil {
				return fmt.Errorf("write %s: %w", name, err)
			}
		}
		log.Info("[setup] auto-generated TLS certificates (ca.crt, instance.crt, instance.key)")

	case CertificateModeManual:
		fileMap := map[string]string{
			"http_ca.crt":            cfg.CertConfig.HttpCaCertificate,
			"http_instance.crt":      cfg.CertConfig.HttpNodeCertificate,
			"http_instance.key":      cfg.CertConfig.HttpPrivateKey,
			"transport_ca.crt":       cfg.CertConfig.TransportCaCertificate,
			"transport_instance.crt": cfg.CertConfig.TransportNodeCertificate,
			"transport_instance.key": cfg.CertConfig.TransportPrivateKey,
		}
		for name, content := range fileMap {
			if content == "" {
				return fmt.Errorf("certificate field for %s is empty", name)
			}
			if err := os.WriteFile(filepath.Join(configDir, name), []byte(content), 0600); err != nil {
				return fmt.Errorf("write %s: %w", name, err)
			}
		}
		log.Info("[setup] wrote manual TLS certificate files")

	default:
		return fmt.Errorf("unknown certificate mode: %s", cfg.CertConfig.Mode)
	}
	return nil
}

func (s *stepSetupCertificates) Rollback(svc *service) error {
	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, "config")
	for _, name := range append(autoCertFiles, manualCertFiles...) {
		os.Remove(filepath.Join(configDir, name))
	}
	return nil
}

// ======================= Step 8: GenerateNewClusterConfig ==============================

type stepGenerateNewClusterConfig struct{}

func (s *stepGenerateNewClusterConfig) IsAssetStep() bool { return true }

func (s *stepGenerateNewClusterConfig) NameI18nKey() string { return "stepGenerateConfig" }

func (s *stepGenerateNewClusterConfig) Execute(ctx context.Context, svc *service) error {
	cfg := newClusterCfg(svc)
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

	// Apply config fields.
	fields := map[string]interface{}{
		"cluster.name":                 cfg.ClusterName,
		"node.name":                    cfg.NodeName,
		"network.host":                 cfg.Host,
		"http.port":                    cfg.HTTPPort,
		"transport.port":               cfg.TransportPort,
		"cluster.initial_master_nodes": []string{cfg.NodeName},
	}

	if cfg.DataDirectory != "" {
		fields["path.data"] = cfg.DataDirectory
	}
	if cfg.LogDirectory != "" {
		fields["path.logs"] = cfg.LogDirectory
	}

	// Security configuration.
	if cfg.EnableSecurity {
		fields["security.enabled"] = true
		fields["security.ssl.transport.enabled"] = true
		fields["security.ssl.http.enabled"] = true

		if cfg.CertConfig.Mode == CertificateModeAuto {
			fields["security.ssl.transport.cert_file"] = "instance.crt"
			fields["security.ssl.transport.key_file"] = "instance.key"
			fields["security.ssl.transport.ca_file"] = "ca.crt"
			fields["security.ssl.http.cert_file"] = "instance.crt"
			fields["security.ssl.http.key_file"] = "instance.key"
			fields["security.ssl.http.ca_file"] = "ca.crt"
		} else {
			fields["security.ssl.transport.cert_file"] = "transport_instance.crt"
			fields["security.ssl.transport.key_file"] = "transport_instance.key"
			fields["security.ssl.transport.ca_file"] = "transport_ca.crt"
			fields["security.ssl.http.cert_file"] = "http_instance.crt"
			fields["security.ssl.http.key_file"] = "http_instance.key"
			fields["security.ssl.http.ca_file"] = "http_ca.crt"
		}
	} else {
		fields["security.enabled"] = false
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

	log.Infof("[setup] configuration generated: cluster=%s node=%s host=%s http=%d transport=%d heap=%dg",
		cfg.ClusterName, cfg.NodeName, cfg.Host, cfg.HTTPPort, cfg.TransportPort, heapGB)
	return nil
}

func (s *stepGenerateNewClusterConfig) Rollback(svc *service) error {
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

// ====================== Step 9: SetAdminPassword =============================

type stepSetAdminPassword struct{}

func (s *stepSetAdminPassword) IsAssetStep() bool { return true }

func (s *stepSetAdminPassword) NameI18nKey() string { return "stepSetAdminPassword" }

func (s *stepSetAdminPassword) Execute(_ context.Context, svc *service) error {
	cfg := newClusterCfg(svc)

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), 12)
	if err != nil {
		return fmt.Errorf("bcrypt hash password: %w", err)
	}
	hash := string(hashBytes)

	userYML := generateUserYML(hash)

	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return err
	}
	userYMLPath := filepath.Join(home, "config", "security", "user.yml")
	if err := os.MkdirAll(filepath.Dir(userYMLPath), 0755); err != nil {
		return fmt.Errorf("create security config dir: %w", err)
	}
	if err := os.WriteFile(userYMLPath, []byte(userYML), 0644); err != nil {
		return fmt.Errorf("write user.yml: %w", err)
	}

	log.Info("[setup] admin password set successfully")
	return nil
}

// generateUserYML produces the security user.yml content.
func generateUserYML(bcryptHash string) string {
	return fmt.Sprintf(`---
# This is the internal user database
# The hash value is a bcrypt hash and can be generated with hash_password.sh

_meta:
  type: "user"
  config_version: 2

# Define your internal users here

## Default users
admin:
  hash: "%s"
  reserved: true
  hidden: true
  external_roles:
    - "admin"
  description: "System default admin user"
infini:
  hash: "%s"
  reserved: true
  hidden: true
  description: "System default monitor user"
infini_agent:
  hash: "%s"
  reserved: true
  hidden: true
  description: "System default agent user"
`, bcryptHash, bcryptHash, bcryptHash)
}

func (s *stepSetAdminPassword) Rollback(_ *service) error {
	return nil
}

// ======================= Step 10: InstallPlugins ==============================

type stepInstallPlugins struct{}

func (s *stepInstallPlugins) IsAssetStep() bool { return true }

func (s *stepInstallPlugins) NameI18nKey() string { return "stepInstallPlugins" }

func (s *stepInstallPlugins) Execute(ctx context.Context, svc *service) error {
	cfg := newClusterCfg(svc)
	if len(cfg.Plugins) == 0 {
		return nil
	}

	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return fmt.Errorf("resolve easysearch home: %w", err)
	}
	pluginBin := filepath.Join(home, "bin", "easysearch-plugin")

	for _, p := range cfg.Plugins {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		arg := p.Name
		if p.URL != "" {
			arg = p.URL
		}

		log.Infof("[setup] installing plugin: %s", arg)
		cmd := exec.CommandContext(ctx, pluginBin, "install", "--batch", arg)
		cmd.Dir = home
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("install plugin %q: %w\nOutput: %s", arg, err, string(output))
		}
	}

	return nil
}

func (s *stepInstallPlugins) Rollback(svc *service) error {
	cfg := newClusterCfg(svc)
	if len(cfg.Plugins) == 0 {
		return nil
	}

	home, err := svc.absoluteEasysearchHome()
	if err != nil {
		return fmt.Errorf("resolve easysearch home: %w", err)
	}
	pluginBin := filepath.Join(home, "bin", "easysearch-plugin")

	for _, p := range cfg.Plugins {
		name := p.Name
		if name == "" {
			continue
		}
		cmd := exec.Command(pluginBin, "remove", name)
		cmd.Dir = home
		_ = cmd.Run()
	}
	return nil
}

// ====================== Step 11: WaitForReady ================================

type stepWaitForReady struct{}

func (s *stepWaitForReady) IsAssetStep() bool { return false }

func (s *stepWaitForReady) NameI18nKey() string { return "stepWaitForReady" }

func (s *stepWaitForReady) Execute(ctx context.Context, svc *service) error {
	cfg := newClusterCfg(svc)

	scheme := "http"
	if cfg.EnableSecurity {
		scheme = "https"
	}
	endpoint := fmt.Sprintf("%s://%s:%d/", scheme, cfg.Host, cfg.HTTPPort)

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	const maxWait = 120 * time.Second
	const interval = 3 * time.Second
	deadline := time.Now().Add(maxWait)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("easysearch did not become ready within %v", maxWait)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.SetBasicAuth("admin", cfg.AdminPassword)

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Infof("[setup] easysearch is ready at %s", endpoint)
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func (s *stepWaitForReady) Rollback(_ *service) error {
	return nil
}
