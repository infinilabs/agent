/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const serviceModeNewCluster serviceMode = "create_cluster"

// CertificateMode represents the certificate provisioning mode.
type CertificateMode string

const (
	CertificateModeAuto   CertificateMode = "auto"
	CertificateModeManual CertificateMode = "manual"
)

// JDKSource represents the source of JDK
type JDKSource string

const (
	JDKSourceRemote JDKSource = "remote"
	JDKSourceLocal  JDKSource = "local"
)

// JDKConfig represents JDK configuration
type JDKConfig struct {
	Source  JDKSource `json:"source"`
	Version string    `json:"version"`
	Path    string    `json:"path,omitempty"`
}

// CertificateConfig represents certificate configuration
type CertificateConfig struct {
	Mode CertificateMode `json:"mode"`
	/*
	   The following 6 fields only appear if Mode is "manual"
	*/
	HttpCaCertificate        string `json:"http_ca_certificate,omitempty"`
	HttpNodeCertificate      string `json:"http_node_certificate,omitempty"`
	HttpPrivateKey           string `json:"http_private_key,omitempty"`
	TransportCaCertificate   string `json:"transport_ca_certificate,omitempty"`
	TransportNodeCertificate string `json:"transport_node_certificate,omitempty"`
	TransportPrivateKey      string `json:"transport_private_key,omitempty"`
}

// PluginConfig represents a single plugin entry in the setup request.
// The JSON form is polymorphic: either a plain string (named plugin key) or
// an object with a "url" field (custom URL plugin). So it is essential
// an enum.
//
//	["analysis-ik", {"url": "file:///path/to/plugin.zip"}]
type PluginConfig struct {
	// Name is set when the plugin is specified as a string key, e.g. "analysis-ik".
	Name string
	// URL is set when the plugin is specified as {"url": "..."} .
	URL string
}

func (p *PluginConfig) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("empty plugin value")
	}

	*p = PluginConfig{}

	switch data[0] {
	case '"':
		var name string
		if err := json.Unmarshal(data, &name); err != nil {
			return fmt.Errorf("parse plugin name: %w", err)
		}
		if name == "" {
			return fmt.Errorf("plugin name must be non-empty")
		}
		p.Name = name
		return nil

	case '{':
		var obj struct {
			URL string `json:"url"`
		}

		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&obj); err != nil {
			return fmt.Errorf("parse plugin object: %w", err)
		}
		if obj.URL == "" {
			return fmt.Errorf("plugin object must have a non-empty \"url\" field")
		}
		p.URL = obj.URL
		return nil

	default:
		return fmt.Errorf("plugin must be a string or an object with \"url\", got: %.20q", data)
	}
}

func (p PluginConfig) MarshalJSON() ([]byte, error) {
	hasName := p.Name != ""
	hasURL := p.URL != ""

	switch {
	case hasName && hasURL:
		return nil, fmt.Errorf("plugin config must have either name or url, not both")
	case hasName:
		return json.Marshal(p.Name)
	case hasURL:
		return json.Marshal(struct {
			URL string `json:"url"`
		}{URL: p.URL})
	default:
		return nil, fmt.Errorf("plugin config must have either name or url")
	}
}

// NewClusterConfig represents configuration for creating a new cluster
type NewClusterConfig struct {
	baseConfig
	Easysearch     string            `json:"easysearch"`
	JDK            JDKConfig         `json:"jdk"`
	ClusterName    string            `json:"cluster_name"`
	NodeName       string            `json:"node_name"`
	EnableSecurity bool              `json:"enable_security"`
	CertConfig     CertificateConfig `json:"certificate_configuration,omitempty"`
	AdminPassword  string            `json:"admin_password"`
	Plugins        []PluginConfig    `json:"plugins,omitempty"`
}

// Mode returns the task mode
func (c *NewClusterConfig) Mode() serviceMode {
	return serviceModeNewCluster
}

// Validate validates the configuration
func (c *NewClusterConfig) Validate() error {
	if err := c.baseConfig.Validate(); err != nil {
		return err
	}

	if c.Easysearch == "" {
		return fmt.Errorf("easysearch version is required")
	}
	if c.ClusterName == "" {
		return fmt.Errorf("cluster_name is required")
	}
	if c.NodeName == "" {
		return fmt.Errorf("node_name is required")
	}
	if c.AdminPassword == "" {
		return fmt.Errorf("admin_password is required")
	}
	if c.EnableSecurity {
		if c.CertConfig.Mode != CertificateModeAuto && c.CertConfig.Mode != CertificateModeManual {
			return fmt.Errorf("invalid certificate mode configuration: [%s]", c.CertConfig.Mode)
		}
	}

	return nil
}

// BuildSteps builds the steps for new cluster setup
func (c *NewClusterConfig) BuildSteps() []creationStep {
	steps := []creationStep{
		&stepPrepareWorkspace{},
		&stepDownloadEasysearch{},
		&stepUnpackEasysearch{},
	}

	if c.JDK.Source == JDKSourceRemote {
		steps = append(steps, &stepDownloadJDK{}, &stepUnpackJDK{})
	}

	steps = append(steps,
		&stepLinkJDK{},
	)

	if c.EnableSecurity {
		steps = append(steps, &stepSetupCertificates{})
	}

	steps = append(steps,
		&stepGenerateNewClusterConfig{},
		&stepSetAdminPassword{},
		&stepInstallPlugins{},
		&stepMarkAssetsReady{},
		&stepStartEasysearch{},
		&stepWaitForReady{},
	)
	return steps
}
