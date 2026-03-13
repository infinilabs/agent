/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
)

// Will be initialized in init()
var globalServiceManager *serviceManager

// writeServiceError maps service-layer errors to appropriate HTTP status codes.
func (h *SetupAPI) writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrServiceNotFound) {
		h.WriteError(w, err.Error(), http.StatusNotFound)
		return
	}
	var statusErr *errUnexpectedServiceStatus
	if errors.As(err, &statusErr) {
		h.WriteError(w, err.Error(), http.StatusConflict)
		return
	}
	h.WriteError(w, err.Error(), http.StatusInternalServerError)
}

// createCluster handles POST /setup/cluster/new
func (h *SetupAPI) createCluster(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var config NewClusterConfig
	if err := h.DecodeJSON(req, &config); err != nil {
		h.WriteError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	svc, err := globalServiceManager.createService(&config)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.WriteAckJSON(w, true, http.StatusCreated, map[string]interface{}{"service_id": svc.ID()})
}

// joinCluster handles POST /setup/cluster/join
func (h *SetupAPI) joinCluster(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var config joinClusterConfig
	if err := h.DecodeJSON(req, &config); err != nil {
		h.WriteError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	_, enrollInfo, err := fetchEnrollInfo(config.EnrollToken)
	if err != nil {
		if errors.Is(err, ErrEnrollTokenInvalid) {
			h.WriteError(w, err.Error(), http.StatusBadRequest)
		} else {
			log.Warnf("enroll failed: %v", err)
			h.WriteError(w, "enroll_token validation failed: "+err.Error(), http.StatusBadGateway)
		}
		return
	}
	config.FetchedEnrollInfo = enrollInfo

	svc, err := globalServiceManager.createService(&config)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.WriteAckJSON(w, true, http.StatusCreated, map[string]interface{}{"service_id": svc.ID()})
}

// recreateServiceHandler handles POST /setup/services/:service_id/creation/recreate
func (h *SetupAPI) recreateServiceHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	if _, err := globalServiceManager.recreateService(serviceId); err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteAckJSON(w, true, http.StatusOK, nil)
}

// getCreationProgressHandler handles GET /setup/services/:service_id/creation/progress
func (h *SetupAPI) getCreationProgressHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	progress, err := globalServiceManager.creationProgress(serviceId)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteJSON(w, progress, http.StatusOK)
}

// pauseCreationHandler handles POST /setup/services/:service_id/creation/stop
func (h *SetupAPI) pauseCreationHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	if err := globalServiceManager.pauseCreation(serviceId); err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteAckJSON(w, true, http.StatusOK, nil)
}

// retryCreationHandler handles POST /setup/services/:service_id/creation/retry
func (h *SetupAPI) retryCreationHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	if err := globalServiceManager.retryCreation(serviceId); err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteAckJSON(w, true, http.StatusOK, nil)
}

// resumeCreationHandler handles POST /setup/services/:service_id/creation/resume
func (h *SetupAPI) resumeCreationHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	if err := globalServiceManager.resumeCreation(serviceId); err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteAckJSON(w, true, http.StatusOK, nil)
}

// getClusterNodeInfoHandler handles GET /setup/services/:service_id/cluster_node_info
func (h *SetupAPI) getClusterNodeInfoHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	info, err := globalServiceManager.getClusterNodeInfo(serviceId)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteJSON(w, info, http.StatusOK)
}

// deleteServiceHandler handles DELETE /setup/services/:service_id
func (h *SetupAPI) deleteServiceHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	if err := globalServiceManager.deleteService(serviceId); err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteAckJSON(w, true, http.StatusOK, nil)
}

// getServiceConfigHandler handles GET /setup/services/:service_id/service_config
func (h *SetupAPI) getServiceConfigHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	config, err := globalServiceManager.getServiceConfig(serviceId)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteJSON(w, config, http.StatusOK)
}

// updateServiceConfigHandler handles PUT /setup/services/:service_id/service_config
func (h *SetupAPI) updateServiceConfigHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")

	// Read body once — req.Body is a stream and can only be consumed once.
	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.WriteError(w, "read request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Determine the concrete config type from the existing service so we
	// unmarshal into the right struct.
	current, err := globalServiceManager.getServiceConfig(serviceId)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	var updated serviceConfig
	switch current.(type) {
	case *NewClusterConfig:
		var c NewClusterConfig
		if err := json.Unmarshal(body, &c); err != nil {
			h.WriteError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		updated = &c
	case *joinClusterConfig:
		var c joinClusterConfig
		if err := json.Unmarshal(body, &c); err != nil {
			h.WriteError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		updated = &c
	default:
		h.WriteError(w, "unknown service config type", http.StatusInternalServerError)
		return
	}

	if err := globalServiceManager.updateServiceConfig(serviceId, updated); err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteAckJSON(w, true, http.StatusOK, nil)
}

// startServiceHandler handles POST /setup/services/:service_id/start
func (h *SetupAPI) startServiceHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	if err := globalServiceManager.startService(serviceId); err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteAckJSON(w, true, http.StatusOK, nil)
}

// stopServiceHandler handles POST /setup/services/:service_id/pause
func (h *SetupAPI) pauseServiceHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	serviceId := ps.ByName("service_id")
	if err := globalServiceManager.pauseService(serviceId); err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.WriteAckJSON(w, true, http.StatusOK, nil)
}

// listServicesHandler handles GET /setup/services
func (h *SetupAPI) listServicesHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	h.WriteJSON(w, globalServiceManager.listServices(), http.StatusOK)
}
