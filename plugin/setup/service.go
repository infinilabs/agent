/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	log "github.com/cihub/seelog"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

type serviceStatus string

// Define the possible states for Easysearch service
const (
	serviceStatusCreating       serviceStatus = "creating"        // The service is downloading resources, configuring, or starting Easysearch and waiting
	serviceStatusCreationPaused serviceStatus = "creation_paused" // The creation process has been paused by the user
	serviceStatusCreationFailed serviceStatus = "creation_failed" // The creation process encountered an error
	serviceStatusRunning        serviceStatus = "running"         // The Easysearch service is running
	serviceStatusPaused         serviceStatus = "paused"          // The Easysearch service was stopped by the user
	serviceStatusProcessing     serviceStatus = "processing"      // Transitioning between running and stopped states
)

/*
 * serviceCreationStatus and serviceCreationStepStatus are created for the
 * creationProgress() function.
 */
type serviceCreationStatus string

const (
	serviceCreationStatusRunning serviceCreationStatus = "running"
	serviceCreationStatusFailed  serviceCreationStatus = "failed"
	serviceCreationStatusSuccess serviceCreationStatus = "success"
	serviceCreationStatusPaused  serviceCreationStatus = "paused"
)

type serviceCreationStepStatus string

const (
	serviceCreationStepStatusRunning serviceCreationStepStatus = "running"
	serviceCreationStepStatusFailed  serviceCreationStepStatus = "failed"
	serviceCreationStepStatusSuccess serviceCreationStepStatus = "success"
	serviceCreationStepStatusPending serviceCreationStepStatus = "pending"
)

func (ss serviceStatus) toCreationStatus() (serviceCreationStatus, error) {
	switch ss {
	case serviceStatusCreating:
		return serviceCreationStatusRunning, nil
	case serviceStatusCreationPaused:
		return serviceCreationStatusPaused, nil
	case serviceStatusCreationFailed:
		return serviceCreationStatusFailed, nil
	case serviceStatusRunning:
		return serviceCreationStatusSuccess, nil
	default:
		return "", fmt.Errorf("service creation was already complete, you cannot get its status")
	}
}

var ErrServiceNotFound = errors.New("service not found")

type errUnexpectedServiceStatus struct {
	Status serviceStatus
}

func (e *errUnexpectedServiceStatus) Error() string {
	return fmt.Sprintf("unexpected service status: %s", e.Status)
}

type serviceMode string

// serviceConfig is the interface for service configurations
type serviceConfig interface {
	Mode() serviceMode
	BuildSteps() []creationStep
	Validate() error
	// Common field accessors
	// TODO: consider removing these 2 getter, do type-cast instead.
	GetHost() string
	GetHTTPPort() int
}

// baseConfig holds common configuration fields that are shared between
// different tasks.
type baseConfig struct {
	Host          string  `json:"host"`
	HTTPPort      int     `json:"http_port"`
	TransportPort int     `json:"transport_port"`
	DataDirectory string  `json:"data_directory,omitempty"`
	LogDirectory  string  `json:"log_directory,omitempty"`
	JVMMemory     float64 `json:"jvm_memory"`
}

// GetHost returns the host
func (c *baseConfig) GetHost() string {
	return c.Host
}

// GetHTTPPort returns the HTTP port
func (c *baseConfig) GetHTTPPort() int {
	return c.HTTPPort
}

// GetTransportPort returns the transport port
func (c *baseConfig) GetTransportPort() int {
	return c.TransportPort
}

// GetDataDirectory returns the data directory
func (c *baseConfig) GetDataDirectory() string {
	return c.DataDirectory
}

// GetLogDirectory returns the log directory
func (c *baseConfig) GetLogDirectory() string {
	return c.LogDirectory
}

// GetJVMMemory returns the JVM memory in GB
func (c *baseConfig) GetJVMMemory() float64 {
	return c.JVMMemory
}

// Validate checks the common fields: port ranges, port availability, and
// directory writability. It is intended to be called from the embedding
// struct's own Validate() before applying mode-specific checks.
func (c *baseConfig) Validate() error {
	if c.HTTPPort <= 0 || c.HTTPPort > 65535 {
		return fmt.Errorf("http_port %d is out of valid range (1-65535)", c.HTTPPort)
	}
	if c.TransportPort <= 0 || c.TransportPort > 65535 {
		return fmt.Errorf("transport_port %d is out of valid range (1-65535)", c.TransportPort)
	}
	if c.HTTPPort == c.TransportPort {
		return fmt.Errorf("http_port and transport_port must be different")
	}

	httpAvailable, err := checkPortAvailable(c.HTTPPort)
	if err != nil {
		return fmt.Errorf("check http_port: %w", err)
	}
	if !httpAvailable {
		return fmt.Errorf("http_port %d is already in use", c.HTTPPort)
	}

	transportAvailable, err := checkPortAvailable(c.TransportPort)
	if err != nil {
		return fmt.Errorf("check transport_port: %w", err)
	}
	if !transportAvailable {
		return fmt.Errorf("transport_port %d is already in use", c.TransportPort)
	}

	if c.DataDirectory != "" {
		result, err := CheckDirectory(c.DataDirectory)
		if err != nil {
			return fmt.Errorf("check data_directory: %w", err)
		}
		if !result.Available {
			return fmt.Errorf("data_directory %q is not available (not writable or invalid path)", c.DataDirectory)
		}
	}

	if c.LogDirectory != "" {
		result, err := CheckDirectory(c.LogDirectory)
		if err != nil {
			return fmt.Errorf("check log_directory: %w", err)
		}
		if !result.Available {
			return fmt.Errorf("log_directory %q is not available (not writable or invalid path)", c.LogDirectory)
		}
	}

	return nil
}

type service struct {
	// RWMutex that protects this service
	mu sync.RWMutex

	// Service UUID
	id     string
	status serviceStatus
	config serviceConfig

	/*
	  Fields that only appear if this service is being created
	*/
	creationSteps []creationStep
	// This value will be len(creationSteps) if creation completes
	creationCurrentStep int
	// Error if creation failed
	creationError string
	// A dedicated goroutine will be spawned to create the service, i.e.,
	// execute the creation_steps.
	//
	// You call this function to stop it.
	cancelFunc context.CancelFunc
	// Stop callback to ensure goroutine stops
	stoppedCh chan struct{}
}

func absoluteServicesDir() (string, error) {
	dataDir := global.Env().GetDataDir()
	servicesDir := filepath.Join(dataDir, "setup", "services")
	dir, err := filepath.Abs(servicesDir)
	if err != nil {
		return "", err
	}

	return dir, nil
}

// AbsoluteWorkspacePath returns the absolute workspace directory path
// for the service.
//
// Directory layout:
//
// service_workspace/
// +-- metadata/
// |   +-- assets_ready (contains either a number or a string "yes")
// |   |
// |   +-- service_config.json
// |   |
// |   +-- service_mode (contains either "new_cluster" or "join_cluster")
// |
// +-- assets/
// |   +-- easysearch-2.0.2-2499-mac-arm64/
// |   |
// |   +-- easysearch
// |   |
// |   +-- graalvm-jdk-21.0.9+7.1/
// |
// +-- easysearch.pid
// |
// +-- easysearch_stdout.log
func (s *service) AbsoluteWorkspacePath() (string, error) {
	servicesDir, err := absoluteServicesDir()
	if err != nil {
		return "", nil
	}
	return filepath.Join(servicesDir, s.id), nil
}

func (s *service) AbsoluteMetadataDirPath() (string, error) {
	ws, err := s.AbsoluteWorkspacePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(ws, "metadata"), nil
}

func (s *service) AbsoluteAssetsDirPath() (string, error) {
	ws, err := s.AbsoluteWorkspacePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(ws, "assets"), nil
}

func (s *service) absoluteEasysearchHome() (string, error) {
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(assets, "easysearch"), nil
}

func (s *service) AbsoluteServiceConfigPath() (string, error) {
	ws, err := s.AbsoluteMetadataDirPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(ws, "service_config.json"), nil
}

func (s *service) AbsoluteAssetsReadyPath() (string, error) {
	ws, err := s.AbsoluteMetadataDirPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(ws, "assets_ready"), nil
}

type serviceCreationProgress struct {
	Status   serviceCreationStatus `json:"status"`
	Progress int                   `json:"progress"`
	Error    string                `json:"error,omitempty"`
	Steps    []creationStepInfo    `json:"steps"`
}

type creationStepInfo struct {
	NameI18nKey string                    `json:"name_i18n_key"`
	Status      serviceCreationStepStatus `json:"status"`
}

func (s *service) creationProgress() (serviceCreationProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, err := s.status.toCreationStatus()
	if err != nil {
		return serviceCreationProgress{}, err
	}
	progress := int(float64(s.creationCurrentStep) / float64(len(s.creationSteps)) * 100)
	error := s.creationError

	var steps = make([]creationStepInfo, 0)
	for idx, step := range s.creationSteps {
		if idx < s.creationCurrentStep {
			steps = append(steps, creationStepInfo{
				NameI18nKey: step.NameI18nKey(),
				Status:      serviceCreationStepStatusSuccess,
			})
		} else if idx == s.creationCurrentStep {
			var status serviceCreationStepStatus

			switch s.status {
			case serviceStatusCreating:
				status = serviceCreationStepStatusRunning
			case serviceStatusCreationFailed:
				status = serviceCreationStepStatusFailed
			default:
				// FIXME
				panic("unreachable")
			}
			steps = append(steps, creationStepInfo{
				NameI18nKey: step.NameI18nKey(),
				Status:      status,
			})
		} else {
			steps = append(steps, creationStepInfo{
				NameI18nKey: step.NameI18nKey(),
				Status:      serviceCreationStepStatusPending,
			})
		}

	}

	return serviceCreationProgress{
		Status:   status,
		Progress: progress,
		Error:    error,
		Steps:    steps,
	}, nil
}

type serviceManager struct {
	mu       sync.RWMutex
	services map[string]*service
}

// Return nil if any error happens.
func newServiceManager() *serviceManager {
	services := make(map[string]*service)

	servicesDir, err := absoluteServicesDir()
	if err != nil {
		log.Errorf("[setup] resolve services directory: %v", err)
		return nil
	}

	entries, err := os.ReadDir(servicesDir)
	if err != nil && !os.IsNotExist(err) {
		log.Errorf("[setup] read services directory %s: %v", servicesDir, err)
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		workspace := filepath.Join(servicesDir, id)

		svc, err := loadServiceFromWorkspace(id, workspace)
		if err != nil {
			log.Warnf("[setup] removing corrupted service workspace %s: %v", workspace, err)
			if removeErr := os.RemoveAll(workspace); removeErr != nil {
				log.Errorf("[setup] failed to remove workspace %s: %v", workspace, removeErr)
			}
			continue
		}

		services[id] = svc
		log.Infof("[setup] loaded service %s (status=%s)", id, svc.status)
	}

	return &serviceManager{
		services: services,
	}
}

// loadServiceFromWorkspace reads the metadata of a service workspace directory
// and reconstructs a *service.
//
// Returns an error if the metadata is missing or corrupted — the caller is
// responsible for deleting the workspace in that case.
func loadServiceFromWorkspace(id, workspace string) (*service, error) {
	metaDir := filepath.Join(workspace, "metadata")

	// --- service_mode ---
	modePath := filepath.Join(metaDir, "service_mode")
	modeBytes, err := os.ReadFile(modePath)
	if err != nil {
		return nil, fmt.Errorf("read service_mode: %w", err)
	}
	mode := serviceMode(strings.TrimSpace(string(modeBytes)))

	// --- service_config.json ---
	configPath := filepath.Join(metaDir, "service_config.json")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read service_config.json: %w", err)
	}

	var cfg serviceConfig
	switch mode {
	case serviceModeNewCluster:
		var c NewClusterConfig
		if err := json.Unmarshal(configBytes, &c); err != nil {
			return nil, fmt.Errorf("parse service_config.json (new_cluster): %w", err)
		}
		cfg = &c
	case serviceModeJoinCluster:
		var c joinClusterConfig
		if err := json.Unmarshal(configBytes, &c); err != nil {
			return nil, fmt.Errorf("parse service_config.json (join_cluster): %w", err)
		}
		cfg = &c
	default:
		return nil, fmt.Errorf("unknown service_mode %q", mode)
	}

	// --- assets_ready ---
	// The file is written only after at least one step completes, so its
	// absence is valid: the process crashed before finishing any step.
	assetsReadyPath := filepath.Join(metaDir, "assets_ready")
	assetsReadyBytes, err := os.ReadFile(assetsReadyPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read assets_ready: %w", err)
	}
	assetsReady := strings.TrimSpace(string(assetsReadyBytes))

	steps := cfg.BuildSteps()

	svc := &service{
		id:            id,
		config:        cfg,
		creationSteps: steps,
	}

	switch assetsReady {
	case "yes":
		// Assets are fully prepared; check whether Easysearch is still running.
		pidFile := filepath.Join(workspace, "easysearch.pid")
		if isEasysearchRunning(pidFile) {
			svc.status = serviceStatusRunning
		} else {
			svc.status = serviceStatusPaused
		}
		svc.creationCurrentStep = len(steps)
	case "":
		// File absent: process crashed before completing any step. Resume from 0.
		svc.status = serviceStatusCreationPaused
		svc.creationCurrentStep = 0
	default:
		// Must be a committed step index (integer).
		n, err := strconv.Atoi(assetsReady)
		if err != nil {
			return nil, fmt.Errorf("assets_ready contains unexpected value %q", assetsReady)
		}
		if n < 0 || n >= len(steps) {
			return nil, fmt.Errorf("assets_ready index %d out of range (steps=%d)", n, len(steps))
		}
		svc.status = serviceStatusCreationPaused
		svc.creationCurrentStep = n + 1
	}

	return svc, nil
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

// unsafeGetService looks up a service by id without acquiring any lock.
// Callers must hold sm.mu (read or write) for the duration of access.
func (sm *serviceManager) unsafeGetService(serviceId string) (*service, bool) {
	svc, exists := sm.services[serviceId]
	return svc, exists
}

func (sm *serviceManager) getService(serviceId string) (*service, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.unsafeGetService(serviceId)
}

func (sm *serviceManager) createService(config serviceConfig) (*service, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	id := util.GetUUID()
	steps := config.BuildSteps()

	svc := &service{
		id:            id,
		status:        serviceStatusCreating,
		config:        config,
		creationSteps: steps,
		stoppedCh:     make(chan struct{}),
	}

	// Write metadata before the goroutine starts so that a crash during
	// creation leaves a recoverable workspace.
	if err := svc.writeCreationMetadata(config); err != nil {
		return nil, fmt.Errorf("write creation metadata: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	svc.cancelFunc = cancel

	sm.mu.Lock()
	sm.services[id] = svc
	sm.mu.Unlock()

	go svc.executeLoop(ctx)

	return svc, nil
}

// writeCreationMetadata persists service_mode and service_config.json so the
// workspace can be recovered by newServiceManager() after a restart.
func (s *service) writeCreationMetadata(config serviceConfig) error {
	metaDir, err := s.AbsoluteMetadataDirPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return fmt.Errorf("create metadata dir: %w", err)
	}

	modePath := filepath.Join(metaDir, "service_mode")
	if err := atomicWriteFile(modePath, []byte(config.Mode())); err != nil {
		return fmt.Errorf("write service_mode: %w", err)
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal service config: %w", err)
	}
	configPath := filepath.Join(metaDir, "service_config.json")
	if err := atomicWriteFile(configPath, configBytes); err != nil {
		return fmt.Errorf("write service_config.json: %w", err)
	}

	return nil
}

// executeLoop runs the creation steps sequentially. It is the goroutine body
// spawned by createService() and resumeCreation().
func (s *service) executeLoop(ctx context.Context) {
	defer close(s.stoppedCh)

	for {
		s.mu.RLock()
		currentIdx := s.creationCurrentStep
		status := s.status
		s.mu.RUnlock()

		if status != serviceStatusCreating {
			return
		}

		if currentIdx >= len(s.creationSteps) {
			s.mu.Lock()
			s.status = serviceStatusRunning
			s.mu.Unlock()
			return
		}

		select {
		case <-ctx.Done():
			// Cancelled before the step started: nothing to roll back.
			s.mu.Lock()
			s.status = serviceStatusCreationPaused
			s.mu.Unlock()
			return
		default:
		}

		step := s.creationSteps[currentIdx]
		err := step.Execute(ctx, s)

		if err != nil {
			if ctx.Err() != nil {
				// Cancelled mid-step: roll back so the step is re-runnable on resume.
				if rollbackErr := step.Rollback(s); rollbackErr != nil {
					log.Warnf("[setup] rollback step [%s] on pause: %v", step.NameI18nKey(), rollbackErr)
				}
				s.mu.Lock()
				s.status = serviceStatusCreationPaused
				s.mu.Unlock()
				return
			}

			// Genuine failure.
			if rollbackErr := step.Rollback(s); rollbackErr != nil {
				log.Warnf("[setup] rollback step [%s] on failure: %v", step.NameI18nKey(), rollbackErr)
			}
			s.mu.Lock()
			s.status = serviceStatusCreationFailed
			s.creationError = fmt.Sprintf("step [%s] failed: %v", step.NameI18nKey(), err)
			s.mu.Unlock()
			return
		}

		// Step succeeded: advance the index and commit to disk so a restart
		// can resume from the next step.
		s.mu.Lock()
		s.creationCurrentStep++
		s.mu.Unlock()

		// Persist progress for asset-preparation steps so a restart can resume
		// from the correct index. Post-asset steps (start, wait) do not update
		// assets_ready: stepMarkAssetsReady already wrote "yes", and those
		// steps have no crash-recovery need.
		if step.IsAssetStep() {
			if err := s.commitCreationStep(currentIdx); err != nil {
				log.Warnf("[setup] persist committed step %d: %v", currentIdx, err)
			}
		}
	}
}

// commitCreationStep writes the completed step index to metadata/assets_ready
// so newServiceManager can reconstruct creationCurrentStep after a restart.
func (s *service) commitCreationStep(completedIdx int) error {
	path, err := s.AbsoluteAssetsReadyPath()
	if err != nil {
		return err
	}
	return atomicWriteFile(path, []byte(strconv.Itoa(completedIdx)))
}

func (sm *serviceManager) recreateService(serviceId string) (*service, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.unsafeGetService(serviceId)
	if !exists {
		return nil, ErrServiceNotFound
	}

	// Atomically validate status and claim ownership by transitioning to
	// processing. A second concurrent recreate will now fail the status check.
	svc.mu.Lock()
	status := svc.status
	switch status {
	case serviceStatusCreating, serviceStatusCreationPaused, serviceStatusCreationFailed:
		// allowed — fall through
	default:
		svc.mu.Unlock()
		return nil, &errUnexpectedServiceStatus{Status: status}
	}
	cancelFunc := svc.cancelFunc
	stoppedCh := svc.stoppedCh
	svc.status = serviceStatusProcessing
	svc.mu.Unlock()

	// Stop any running creation goroutine outside svc.mu to avoid deadlock
	// (executeLoop also acquires svc.mu).
	if status == serviceStatusCreating && cancelFunc != nil {
		cancelFunc()
		if stoppedCh != nil {
			<-stoppedCh
		}
	}

	// Remove assets/ and assets_ready so the next run starts clean.
	assetsDir, err := svc.AbsoluteAssetsDirPath()
	if err != nil {
		return nil, fmt.Errorf("resolve assets dir: %w", err)
	}
	if err := os.RemoveAll(assetsDir); err != nil {
		return nil, fmt.Errorf("remove assets dir: %w", err)
	}
	assetsReadyPath, err := svc.AbsoluteAssetsReadyPath()
	if err != nil {
		return nil, fmt.Errorf("resolve assets_ready path: %w", err)
	}
	if err := os.Remove(assetsReadyPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove assets_ready: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Reset in-memory state and hand off the new cancel/stoppedCh.
	svc.mu.Lock()
	svc.creationSteps = svc.config.BuildSteps()
	svc.creationCurrentStep = 0
	svc.creationError = ""
	svc.status = serviceStatusCreating
	svc.stoppedCh = make(chan struct{})
	svc.cancelFunc = cancel
	svc.mu.Unlock()

	go svc.executeLoop(ctx)

	return svc, nil
}

func (sm *serviceManager) creationProgress(serviceId string) (serviceCreationProgress, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.unsafeGetService(serviceId)
	if !exists {
		return serviceCreationProgress{}, ErrServiceNotFound
	}
	return svc.creationProgress()
}

func (sm *serviceManager) pauseCreation(serviceId string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.unsafeGetService(serviceId)
	if !exists {
		return ErrServiceNotFound
	}

	svc.mu.Lock()
	if svc.status != serviceStatusCreating {
		status := svc.status
		svc.mu.Unlock()
		return &errUnexpectedServiceStatus{Status: status}
	}
	cancelFunc := svc.cancelFunc
	stoppedCh := svc.stoppedCh
	svc.status = serviceStatusProcessing
	svc.mu.Unlock()

	if cancelFunc != nil {
		cancelFunc()
	}
	if stoppedCh != nil {
		<-stoppedCh
	}
	// executeLoop sets svc.status = serviceStatusCreationPaused before closing
	// stoppedCh, so no status write-back is needed here.
	return nil
}

func (sm *serviceManager) retryCreation(serviceId string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.unsafeGetService(serviceId)
	if !exists {
		return ErrServiceNotFound
	}

	svc.mu.Lock()
	if svc.status != serviceStatusCreationFailed {
		status := svc.status
		svc.mu.Unlock()
		return &errUnexpectedServiceStatus{Status: status}
	}
	// executeLoop already rolled back the failed step, and creationCurrentStep
	// still points at it, so the goroutine will re-run it from scratch.
	svc.creationError = ""
	svc.status = serviceStatusCreating
	svc.stoppedCh = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	svc.cancelFunc = cancel
	svc.mu.Unlock()

	go svc.executeLoop(ctx)
	return nil
}

func (sm *serviceManager) resumeCreation(serviceId string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.unsafeGetService(serviceId)
	if !exists {
		return ErrServiceNotFound
	}

	svc.mu.Lock()
	if svc.status != serviceStatusCreationPaused {
		status := svc.status
		svc.mu.Unlock()
		return &errUnexpectedServiceStatus{Status: status}
	}
	// creationCurrentStep already points to the next step to execute.
	svc.status = serviceStatusCreating
	svc.stoppedCh = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	svc.cancelFunc = cancel
	svc.mu.Unlock()

	go svc.executeLoop(ctx)
	return nil
}

type clusterNodeInfo struct {
	ClusterName       string   `json:"cluster_name"`
	ClusterStatus     string   `json:"cluster_status"`
	ClusterVersion    string   `json:"cluster_version"`
	ClusterAddress    string   `json:"cluster_address"`
	ClusterPort       int      `json:"cluster_port"`
	NodeName          string   `json:"node_name"`
	NodeID            string   `json:"node_id"`
	NodeRoles         []string `json:"node_roles"`
	CPUUtilization    float64  `json:"cpu_utilization"`     // whole percent, e.g. 34
	JVMMemoryCapacity float64  `json:"jvm_memory_capacity"` // GB, max heap
	DiskUsage         float64  `json:"disk_usage"`          // disk used percent, 0-100
	JVMUsage          float64  `json:"jvm_usage"`           // heap used percent
}

func (sm *serviceManager) getClusterNodeInfo(serviceId string) (clusterNodeInfo, error) {
	sm.mu.RLock()
	svc, exists := sm.unsafeGetService(serviceId)
	sm.mu.RUnlock()
	if !exists {
		return clusterNodeInfo{}, ErrServiceNotFound
	}

	svc.mu.RLock()
	status := svc.status
	svc.mu.RUnlock()
	if status != serviceStatusRunning {
		return clusterNodeInfo{}, &errUnexpectedServiceStatus{Status: status}
	}

	result, err := buildClusterNodeInfo(svc)
	if err != nil {
		return clusterNodeInfo{}, err
	}
	return *result, nil
}

func (sm *serviceManager) getServiceConfig(serviceId string) (serviceConfig, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.unsafeGetService(serviceId)
	if !exists {
		return nil, ErrServiceNotFound
	}
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	return svc.config, nil
}

func (sm *serviceManager) updateServiceConfig(serviceId string, config serviceConfig) error {
	// Hold sm.mu.RLock for the entire operation. This prevents a concurrent
	// deleteService (which needs sm.mu.Lock) from removing the service between
	// our lookup and the disk+memory write.
	//
	// Multiple concurrent updates to *different* services still proceed in
	// parallel (RLock is shared). Updates to the *same* service are serialized
	// by svc.mu.Lock below.
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.unsafeGetService(serviceId)
	if !exists {
		return ErrServiceNotFound
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal service config: %w", err)
	}

	configPath, err := svc.AbsoluteServiceConfigPath()
	if err != nil {
		return fmt.Errorf("resolve service config path: %w", err)
	}
	if err := atomicWriteFile(configPath, data); err != nil {
		return fmt.Errorf("write service_config.json: %w", err)
	}

	svc.config = config
	return nil
}

func (sm *serviceManager) deleteService(serviceId string) error {
	sm.mu.Lock()
	svc, exists := sm.services[serviceId]
	if !exists {
		sm.mu.Unlock()
		return ErrServiceNotFound
	}
	// Remove from the map immediately so no new operations can find this
	// service. Slow cleanup below happens without holding the manager lock.
	delete(sm.services, serviceId)
	sm.mu.Unlock()

	// Snapshot the fields we need under the service's own lock.
	svc.mu.Lock()
	status := svc.status
	cancelFunc := svc.cancelFunc
	stoppedCh := svc.stoppedCh
	svc.mu.Unlock()

	// If a creation goroutine is running, stop it and wait for it to exit
	// before touching the workspace on disk.
	if status == serviceStatusCreating && cancelFunc != nil {
		cancelFunc()
		if stoppedCh != nil {
			<-stoppedCh
		}
	}

	// If Easysearch is (or may be) running, kill it.
	if status == serviceStatusRunning || status == serviceStatusProcessing {
		if err := killEasysearch(svc); err != nil {
			log.Warnf("[setup] kill easysearch for service %s: %v", serviceId, err)
		}
	}

	ws, err := svc.AbsoluteWorkspacePath()
	if err != nil {
		return fmt.Errorf("resolve workspace path: %w", err)
	}
	if err := os.RemoveAll(ws); err != nil {
		return fmt.Errorf("remove workspace: %w", err)
	}
	return nil
}

func (sm *serviceManager) startService(serviceId string) error {
	// Hold sm.mu.RLock for the entire operation to prevent deleteService from
	// racing with our launch (same pattern as updateServiceConfig).
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.unsafeGetService(serviceId)
	if !exists {
		return ErrServiceNotFound
	}

	svc.mu.Lock()
	if svc.status != serviceStatusPaused {
		status := svc.status
		svc.mu.Unlock()
		return &errUnexpectedServiceStatus{Status: status}
	}
	svc.status = serviceStatusProcessing
	svc.mu.Unlock() // Unlock so that frontend code could call listServices

	err := launchEasysearch(context.Background(), svc)

	svc.mu.Lock()
	if err != nil {
		svc.status = serviceStatusPaused
	} else {
		svc.status = serviceStatusRunning
	}
	svc.mu.Unlock()

	return err
}

func (sm *serviceManager) pauseService(serviceId string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, exists := sm.unsafeGetService(serviceId)
	if !exists {
		return ErrServiceNotFound
	}

	svc.mu.Lock()
	if svc.status != serviceStatusRunning {
		status := svc.status
		svc.mu.Unlock()
		return &errUnexpectedServiceStatus{Status: status}
	}
	svc.status = serviceStatusProcessing
	svc.mu.Unlock() // Unlock so that frontend code could call listServices

	err := killEasysearch(svc)

	svc.mu.Lock()
	if err != nil {
		svc.status = serviceStatusRunning
	} else {
		svc.status = serviceStatusPaused
	}
	svc.mu.Unlock()

	return err
}

// ID returns the service's unique identifier.
func (s *service) ID() string { return s.id }

type serviceInfo struct {
	Id                string        `json:"id"`                           // service ID
	ClusterName       string        `json:"cluster_name,omitempty"`       // Cluster name, optional
	NodeName          string        `json:"node_name"`                    // Node name
	EasysearchVersion string        `json:"easysearch_version,omitempty"` // Easysearch version
	Port              int           `json:"port"`                         // Easysearch HTTP port
	Status            serviceStatus `json:"status"`
}

func (sm *serviceManager) listServices() []serviceInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]serviceInfo, 0, len(sm.services))
	for _, svc := range sm.services {
		svc.mu.RLock()
		cfg := svc.config
		status := svc.status
		id := svc.id
		svc.mu.RUnlock()

		info := serviceInfo{
			Id:     id,
			Status: status,
			Port:   cfg.GetHTTPPort(),
		}

		switch c := cfg.(type) {
		case *NewClusterConfig:
			info.ClusterName = c.ClusterName
			info.NodeName = c.NodeName
			info.EasysearchVersion = c.Easysearch
		case *joinClusterConfig:
			info.NodeName = c.NodeName
			if c.FetchedEnrollInfo != nil {
				info.ClusterName = c.FetchedEnrollInfo.ClusterName
				info.EasysearchVersion = c.FetchedEnrollInfo.Version
			}
		}

		result = append(result, info)
	}
	return result
}
