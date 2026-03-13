/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import "fmt"

const serviceModeJoinCluster = "join_cluster"

// joinClusterConfig represents configuration for joining an existing cluster.
// The Easysearch version and JDK version are NOT part of the request body;
// they come from the enroll response (FetchedEnrollInfo).
type joinClusterConfig struct {
	baseConfig
	EnrollToken string   `json:"enroll_token"`
	Roles       []string `json:"roles,omitempty"`
	NodeName    string   `json:"node_name"`

	// FetchedEnrollInfo is populated by the POST /setup/cluster/join handler
	// before the task is created. It is not part of the JSON request body.
	FetchedEnrollInfo *EnrollNodeResponse `json:"-"`
}

// Mode returns the task mode
func (c *joinClusterConfig) Mode() serviceMode {
	return serviceModeJoinCluster
}

// Validate validates the configuration
func (c *joinClusterConfig) Validate() error {
	if err := c.baseConfig.Validate(); err != nil {
		return err
	}
	if c.EnrollToken == "" {
		return fmt.Errorf("enroll_token is required")
	}
	if c.NodeName == "" {
		return fmt.Errorf("node_name is required")
	}

	if len(c.FetchedEnrollInfo.SeedAddresses) == 0 {
		return fmt.Errorf("seed_addresses should not be empty")
	}

	return nil
}

// BuildSteps builds the steps for joining an existing cluster.
func (c *joinClusterConfig) BuildSteps() []creationStep {
	steps := []creationStep{
		&stepPrepareWorkspace{},
		&stepDownloadEasysearch{},
		&stepUnpackEasysearch{},
		&stepDownloadJDK{},
		&stepUnpackJDK{},
		&stepLinkJDK{},
		&stepWriteJoinCerts{},
		&stepGenerateJoinClusterConfig{},
		&stepInstallJoinPlugins{},
		&stepMarkAssetsReady{},
		&stepStartEasysearch{},
		&stepWaitForNodeJoin{},
	}
	return steps
}
