/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"infini.sh/framework/core/api"
)

// Initialize the package, should be invoked when app starts.
func Init() {
	globalServiceManager = newServiceManager()

	/*
		Register API
	*/
	initAPI()
}

type SetupAPI struct {
	api.Handler
}

// Set up API handlers.
func initAPI() {
	apiHandler := SetupAPI{}
	// Environment check APIs
	api.HandleAPIMethod(api.GET, "/setup/env-check/entries", apiHandler.getEnvCheckEntries)
	api.HandleAPIMethod(api.GET, "/setup/env-check/:id", apiHandler.envCheckByID)
	api.HandleAPIMethod(api.POST, "/setup/check/port/:port", apiHandler.checkPort)
	api.HandleAPIMethod(api.POST, "/setup/check/directory", apiHandler.checkDirectory)
	// Version info APIs
	api.HandleAPIMethod(api.GET, "/setup/easysearch/versions", apiHandler.getEasysearchVersions)
	api.HandleAPIMethod(api.GET, "/setup/jdk_easysearch_plugins/versions", apiHandler.getJdkEasysearchPluginsVersions)
	// Get Memory info and recommended JVM setting
	api.HandleAPIMethod(api.GET, "/setup/jvm-memory-recommendation", apiHandler.getJVMMemoryRecommendation)
	// Cluster setup APIs
	api.HandleAPIMethod(api.POST, "/setup/cluster/new", apiHandler.createCluster)
	api.HandleAPIMethod(api.POST, "/setup/cluster/join", apiHandler.joinCluster)
	// Service management APIs
	api.HandleAPIMethod(api.GET, "/setup/services", apiHandler.listServicesHandler)
	api.HandleAPIMethod(api.DELETE, "/setup/services/:service_id", apiHandler.deleteServiceHandler)
	api.HandleAPIMethod(api.GET, "/setup/services/:service_id/creation/progress", apiHandler.getCreationProgressHandler)
	api.HandleAPIMethod(api.POST, "/setup/services/:service_id/creation/recreate", apiHandler.recreateServiceHandler)
	api.HandleAPIMethod(api.POST, "/setup/services/:service_id/creation/pause", apiHandler.pauseCreationHandler)
	api.HandleAPIMethod(api.POST, "/setup/services/:service_id/creation/retry", apiHandler.retryCreationHandler)
	api.HandleAPIMethod(api.POST, "/setup/services/:service_id/creation/resume", apiHandler.resumeCreationHandler)
	api.HandleAPIMethod(api.GET, "/setup/services/:service_id/cluster_node_info", apiHandler.getClusterNodeInfoHandler)
	api.HandleAPIMethod(api.GET, "/setup/services/:service_id/service_config", apiHandler.getServiceConfigHandler)
	api.HandleAPIMethod(api.PUT, "/setup/services/:service_id/service_config", apiHandler.updateServiceConfigHandler)
	api.HandleAPIMethod(api.POST, "/setup/services/:service_id/start", apiHandler.startServiceHandler)
	api.HandleAPIMethod(api.POST, "/setup/services/:service_id/pause", apiHandler.pauseServiceHandler)
}
