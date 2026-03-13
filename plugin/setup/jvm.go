/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"fmt"
	"math"
	"net/http"

	"github.com/shirou/gopsutil/v3/mem"
	httprouter "infini.sh/framework/core/api/router"
)

// JVMMemoryRecommendationResponse is the response for GET /setup/jvm-memory-recommendation
type JVMMemoryRecommendationResponse struct {
	Total      float64 `json:"total"`      // Total system memory in GB
	Available  float64 `json:"available"`  // Available system memory in GB
	Suggestion float64 `json:"suggestion"` // Suggested JVM heap size in GB
}

const maxCompressedOopsHeapGB = 31.0

// getJVMMemoryRecommendation handles GET /setup/jvm-memory-recommendation
func (h *SetupAPI) getJVMMemoryRecommendation(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	recommendation, err := GetJVMMemoryRecommendation()
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, recommendation, http.StatusOK)
}

// GetJVMMemoryRecommendation returns memory information and JVM heap recommendation
func GetJVMMemoryRecommendation() (*JVMMemoryRecommendationResponse, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("get virtual memory info: %w", err)
	}

	totalGB := float64(vm.Total) / (1024 * 1024 * 1024)
	availableGB := float64(vm.Available) / (1024 * 1024 * 1024)

	// Round to 1 decimal place
	totalGB = roundDownToTenths(totalGB)
	availableGB = roundDownToTenths(availableGB)
	suggestion := roundDownToTenths(recommendJVMMemoryGB(totalGB, availableGB))

	return &JVMMemoryRecommendationResponse{
		Total:      totalGB,
		Available:  availableGB,
		Suggestion: suggestion,
	}, nil
}

func recommendJVMMemoryGB(totalGB, availableGB float64) float64 {
	return minFloat64(totalGB/2, availableGB, maxCompressedOopsHeapGB)
}

func roundDownToTenths(v float64) float64 {
	return math.Floor(v*10) / 10
}

func minFloat64(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}

	min := values[0]
	for _, value := range values[1:] {
		if value < min {
			min = value
		}
	}
	return min
}
