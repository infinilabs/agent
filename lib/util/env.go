/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package util

import "os"

func IsKubernetes() bool {
	// Check if the environment variable KUBERNETES_SERVICE_HOST is set)
	_, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	// If it exists, we are likely running in a Kubernetes environment
	// This is a common environment variable set by Kubernetes for service discovery
	return exists
}
