/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
	httprouter "infini.sh/framework/core/api/router"
)

type CheckPortResponse struct {
	Available bool `json:"available"`
}

type CheckDirectoryRequest struct {
	Path string `json:"path"`
}

type CheckDirectoryResponse struct {
	Available  bool   `json:"available"`
	MountPoint string `json:"mount_point,omitempty"`
	// In GiB
	TotalDiskCapacity float64 `json:"total_disk_capacity,omitempty"`
	// In GiB
	AvailableDiskSpace float64 `json:"available_disk_space,omitempty"`
	DiskType           string  `json:"disk_type,omitempty"`
}

// Errors that checkDirectory() could return.
var (
	errPathRequired    = errors.New("path is required")
	errPathNotAbsolute = errors.New("path must be absolute")
	errPathInvalid     = errors.New("invalid directory path")
)

// checkPort handles POST /setup/check/port/:port.
func (h *SetupAPI) checkPort(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	portRaw := strings.TrimSpace(params.ByName("port"))
	port, err := strconv.Atoi(portRaw)
	if err != nil || port <= 0 || port > 65535 {
		h.WriteError(w, "invalid port", http.StatusBadRequest)
		return
	}

	available, err := checkPortAvailable(port)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, CheckPortResponse{Available: available}, http.StatusOK)
}

// checkDirectory handles POST /setup/check/directory.
func (h *SetupAPI) checkDirectory(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	request := CheckDirectoryRequest{}
	if err := h.DecodeJSON(req, &request); err != nil {
		h.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := CheckDirectory(request.Path)
	if err != nil {
		if errors.Is(err, errPathRequired) || errors.Is(err, errPathNotAbsolute) || errors.Is(err, errPathInvalid) {
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, result, http.StatusOK)
}

// checkPortAvailable checks whether a TCP port can be bound on this host.
// This is extracted for reuse by setup tasks before starting Easysearch.
func checkPortAvailable(port int) (bool, error) {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// For setup UX, bind failures mean "not available" rather than API failure.
		return false, nil
	}
	_ = listener.Close()
	return true, nil
}

// CheckDirectory checks whether a directory path can be used by setup logic.
// It returns availability plus best-effort disk metadata for frontend display.
func CheckDirectory(dirPath string) (CheckDirectoryResponse, error) {
	result := CheckDirectoryResponse{Available: false}

	// Pre-process the directory
	expandedPath, err := expandHomeDir(dirPath)
	if err != nil {
		return result, err
	}
	expandedPath = filepath.Clean(expandedPath)
	if expandedPath == "" {
		return result, errPathInvalid
	}
	if !filepath.IsAbs(expandedPath) {
		return result, errPathNotAbsolute
	}

	// Then check availability
	checkBase, available, err := checkDirectoryAvailability(expandedPath)
	if err != nil {
		return result, err
	}
	if !available {
		return result, nil
	}

	// It is available, get further needed information.
	result.Available = true

	usage, mountPoint, fsType := getDiskInfo(checkBase)
	if usage != nil {
		result.MountPoint = mountPoint
		result.TotalDiskCapacity = bytesToGB(usage.Total)
		result.AvailableDiskSpace = bytesToGB(usage.Free)
		result.DiskType = fsType
	}

	return result, nil
}

func checkDirectoryAvailability(path string) (string, bool, error) {
	// If path exists, validate writability on target dir. (Symlink handled)
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return "", false, nil
		}

		if !canWriteToDir(path) {
			return "", false, nil
		}
		return path, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}

	// Path does not exist. Find nearest existing directory in the path chain.
	ancestor, err := nearestExistingDirectory(path)
	if err != nil {
		return "", false, err
	}
	if ancestor == "" {
		return "", false, nil
	}
	if !canWriteToDir(ancestor) {
		return "", false, nil
	}
	return ancestor, true, nil
}

func nearestExistingDirectory(path string) (string, error) {
	current := path
	for {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				// Existing file on the path blocks mkdir -p semantics.
				return "", nil
			}
			return current, nil
		}

		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", nil
		}
		current = parent
	}
}

func expandHomeDir(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errPathRequired
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home directory: %w", err)
		}
		if trimmed == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(trimmed, "~/")), nil
	}

	return trimmed, nil
}

func canWriteToDir(dir string) bool {
	tmp, err := os.CreateTemp(dir, ".agent-dir-check-*")
	if err != nil {
		return false
	}

	// cleanup (try our best)
	maxRetries := 3
	for range maxRetries {
		err = tmp.Close()
		if err == nil {
			break
		}
	}
	for range maxRetries {
		err = os.Remove(tmp.Name())
		if err == nil {
			break
		}
	}

	return true
}

func getDiskInfo(path string) (*disk.UsageStat, string, string) {
	usage, err := disk.Usage(path)
	if err != nil {
		return nil, "", ""
	}

	mountPoint := usage.Path
	fsType := ""

	partitions, err := disk.Partitions(false)
	if err == nil {
		bestLen := 0
		cleanPath := filepath.Clean(path)
		for _, p := range partitions {
			mp := filepath.Clean(p.Mountpoint)
			if strings.HasPrefix(cleanPath, mp) && len(mp) > bestLen {
				bestLen = len(mp)
				mountPoint = p.Mountpoint
				fsType = p.Fstype
			}
		}
	}

	return usage, mountPoint, fsType
}

func bytesToGB(v uint64) float64 {
	return float64(v) / (1024 * 1024 * 1024)
}
