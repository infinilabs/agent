/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/env"
)

// Environment check configuration
type setupEnvCheckConfig struct {
	OSCPUWhitelist []string `config:"os_cpu_whitelist"`
}

// Environment check result.
const (
	// Fully meets the requirements
	envCheckResultOptimal = "optimal"
	// Works but does not fully meet the recommended setup
	envCheckResultSuboptimal = "suboptimal"
	// Does not meet the minimum requirements
	envCheckResultFailed = "failed"
)

// By default, Agent supports these platforms.
var defaultOSCPUWhitelist = []string{"linux-amd64", "linux-aarch64", "mac-amd64", "mac-aarch64", "windows-amd64"}

type EnvCheckEntry struct {
	ID          string `json:"id"`
	NameI18nKey string `json:"name_i18n_key"`
}

// Check result. Return value of the
type EnvCheckResult struct {
	// Check result, see the envCheckResultXxx constants.
	Result string `json:"result"`
	// Value of this environment check entry, e.g., if the entry is
	// cpu_os, then its value could be "macOS 26/aarch64".
	Value string `json:"value"`
	// i18n key of the message that should be passed to the frontend.
	MessageI18nKey string `json:"message_i18n_key,omitempty"`
	// i18n key of the badge message that should be passed to the frontend.
	BadgeMessageI18nKey string `json:"badge_message_i18n_key"`
	// We may provide users with suggestions on how to improve its environment
	// check entry.
	//
	// This field contains its i18n key.
	SuggestionI18nKey string `json:"suggestion_i18n_key,omitempty"`
}

// Handler of "GET /setup/env-check/entries"
func (h *SetupAPI) getEnvCheckEntries(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	h.WriteJSON(w, listEnvCheckEntries(normalizeOs(runtime.GOOS)), http.StatusOK)
}

// Handler of "GET /setup/env-check/:id"
func (h *SetupAPI) envCheckByID(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	id := strings.TrimSpace(params.ByName("id"))
	if id == "" {
		h.WriteError(w, "missing env-check id", http.StatusBadRequest)
		return
	}

	var (
		result EnvCheckResult
		err    error
	)
	switch id {
	case "os_cpu":
		result, err = envCheckOSCPU()
	case "kernel":
		result, err = envCheckKernel()
	case "ram":
		result, err = envCheckRAM()
	case "jdk":
		result, err = envCheckJDK()
	case "open_fd_limits":
		result, err = envCheckOpenFDLimits()
	case "max_map_count":
		result, err = envCheckMaxMapCount()
	default:
		result = EnvCheckResult{}
		err = fmt.Errorf("%w: %s", errUnsupportedEnvCheckID, id)
	}

	if err != nil {
		if errors.Is(err, errUnsupportedEnvCheckID) {
			h.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, result, http.StatusOK)
}

var errUnsupportedEnvCheckID = errors.New("unsupported env-check id")

func listEnvCheckEntries(os string) []EnvCheckEntry {
	// Base checks available on all supported platforms.
	entries := []EnvCheckEntry{
		{ID: "os_cpu", NameI18nKey: "envCheckEntryOsCpu"},
		{ID: "kernel", NameI18nKey: "envCheckEntryKernel"},
		{ID: "ram", NameI18nKey: "envCheckEntryRam"},
		{ID: "jdk", NameI18nKey: "envCheckEntryJdk"},
	}

	if os == "linux" || os == "mac" {
		entries = append(entries, EnvCheckEntry{ID: "open_fd_limits", NameI18nKey: "envCheckEntryOpenFdLimits"})
	}

	if os == "linux" {
		entries = append(entries, EnvCheckEntry{ID: "max_map_count", NameI18nKey: "envCheckEntryMaxMapCount"})
	}

	return entries
}

func envCheckOSCPU() (EnvCheckResult, error) {
	platform := getCurrentPlatform()

	whitelist, err := loadOSCPUWhitelist()
	if err != nil {
		return EnvCheckResult{}, err
	}
	platformDisplayName := getPlatformDisplayName(platform)

	if slices.Contains(whitelist, platform) {
		return EnvCheckResult{
			Result:              envCheckResultOptimal,
			Value:               platformDisplayName,
			BadgeMessageI18nKey: "supported",
		}, nil
	}

	return EnvCheckResult{
		Result:              envCheckResultFailed,
		Value:               platformDisplayName,
		MessageI18nKey:      "envCheckEntryOsCpuNotSupported",
		BadgeMessageI18nKey: "notSupported",
	}, nil
}

func envCheckKernel() (EnvCheckResult, error) {
	info, err := host.Info()
	if err != nil {
		return EnvCheckResult{}, fmt.Errorf("detect kernel info: %w", err)
	}

	kernel := strings.TrimSpace(info.KernelVersion)
	if kernel == "" {
		kernel = "unknown"
	}

	osResult, err := envCheckOSCPU()
	if err != nil {
		return EnvCheckResult{}, err
	}

	// Follow product rule: kernel compatibility inherits OS compatibility.
	// If OS is unsupported, kernel check is failed even if a version string is available.
	if osResult.Result == envCheckResultFailed {
		return EnvCheckResult{
			Result:              envCheckResultFailed,
			Value:               kernel,
			BadgeMessageI18nKey: "incompatible",
		}, nil
	}

	return EnvCheckResult{
		Result:              envCheckResultOptimal,
		Value:               kernel,
		BadgeMessageI18nKey: "compatible",
	}, nil
}

func loadOSCPUWhitelist() ([]string, error) {
	cfg := setupConfig{}
	exist, err := env.ParseConfig("setup", &cfg)
	if err != nil {
		return nil, fmt.Errorf("load setup config: %w", err)
	}
	if !exist {
		return append([]string{}, defaultOSCPUWhitelist...), nil
	}

	if len(cfg.EnvCheck.OSCPUWhitelist) == 0 {
		return append([]string{}, defaultOSCPUWhitelist...), nil
	}

	return cfg.EnvCheck.OSCPUWhitelist, nil
}

func envCheckRAM() (EnvCheckResult, error) {
	const recommendedRam = 4

	vm, err := mem.VirtualMemory()
	if err != nil {
		return EnvCheckResult{}, fmt.Errorf("detect RAM: %w", err)
	}

	ramGB := float64(vm.Total) / (1024 * 1024 * 1024)
	value := fmt.Sprintf("%.1fGB", ramGB)

	if ramGB > recommendedRam {
		return EnvCheckResult{
			Result:              envCheckResultOptimal,
			Value:               value,
			BadgeMessageI18nKey: "recommended",
		}, nil
	}

	return EnvCheckResult{
		Result:              envCheckResultSuboptimal,
		Value:               value,
		MessageI18nKey:      "envCheckEntryRamLow",
		BadgeMessageI18nKey: "warning",
	}, nil
}

func envCheckJDK() (EnvCheckResult, error) {
	allJdks := findAllLocalJdks()

	if len(allJdks) == 0 {
		return EnvCheckResult{
			Result:              envCheckResultSuboptimal,
			MessageI18nKey:      "envCheckEntryJdkNotInstalled",
			BadgeMessageI18nKey: "notInstalled",
		}, nil
	}

	// Iterate all discovered JDKs, rate each, and keep the best result.
	// Rating: JDK 21 (optimal, rank 2) > JDK 11-20 (suboptimal, rank 1) > others (skip).
	var bestResult EnvCheckResult
	bestRank := -1

	for _, jdk := range allJdks {
		var result EnvCheckResult
		var rank int

		switch {
		case jdk.Major == 21:
			result = EnvCheckResult{
				Result:              envCheckResultOptimal,
				Value:               jdk.Version,
				BadgeMessageI18nKey: "installed",
			}
			rank = 2
		case jdk.Major >= 11 && jdk.Major < 21:
			result = EnvCheckResult{
				Result:              envCheckResultSuboptimal,
				Value:               jdk.Version,
				MessageI18nKey:      "envCheckEntryJdkCompatible",
				BadgeMessageI18nKey: "installed",
			}
			rank = 1
		default:
			continue
		}

		if rank > bestRank {
			bestResult = result
			bestRank = rank
		}
	}

	if bestRank < 0 {
		// Found JDKs but none in the supported 11-21 range.
		return EnvCheckResult{
			Result:              envCheckResultSuboptimal,
			MessageI18nKey:      "envCheckEntryJdkNoSupportedJdkFound",
			BadgeMessageI18nKey: "notSupported",
		}, nil
	}

	return bestResult, nil
}

func envCheckOpenFDLimits() (EnvCheckResult, error) {
	// This entry is not supported on Windows, so this function should never
	// be called, as a fallback, fail the check.
	if runtime.GOOS == "windows" {
		return EnvCheckResult{
			Result:              envCheckResultFailed,
			MessageI18nKey:      "envCheckEntryNotSupported",
			BadgeMessageI18nKey: "notSupported",
		}, nil
	}

	var limits syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limits); err != nil {
		return EnvCheckResult{}, fmt.Errorf("read open file descriptor limit: %w", err)
	}

	current := limits.Cur
	value := fmt.Sprintf("%d", current)

	// Thresholds follow setup spec:
	// [0,100] failed, [101,1024] suboptimal, >1024 optimal.
	switch {
	case current <= 100:
		return EnvCheckResult{
			Result:              envCheckResultFailed,
			Value:               value,
			MessageI18nKey:      "envCheckEntryOpenFdLimitsTooLow",
			BadgeMessageI18nKey: "optimizable",
			SuggestionI18nKey:   "envCheckEntryOpenFdLimitsHowToIncrease",
		}, nil
	case current <= 1024:
		return EnvCheckResult{
			Result:              envCheckResultSuboptimal,
			Value:               value,
			MessageI18nKey:      "envCheckEntryOpenFdLimitsLow",
			BadgeMessageI18nKey: "optimizable",
			SuggestionI18nKey:   "envCheckEntryOpenFdLimitsHowToIncrease",
		}, nil
	default:
		return EnvCheckResult{
			Result:              envCheckResultOptimal,
			Value:               value,
			BadgeMessageI18nKey: "optimized",
		}, nil
	}
}

func envCheckMaxMapCount() (EnvCheckResult, error) {
	// vm.max_map_count exists on Linux only; non-Linux is a capability mismatch,
	// not a transient runtime error.
	if runtime.GOOS != "linux" {
		return EnvCheckResult{
			Result:              envCheckResultFailed,
			Value:               "unsupported",
			MessageI18nKey:      "envCheckEntryNotSupported",
			BadgeMessageI18nKey: "notSupported",
		}, nil
	}

	content, err := os.ReadFile("/proc/sys/vm/max_map_count")
	if err != nil {
		return EnvCheckResult{}, fmt.Errorf("read vm.max_map_count: %w", err)
	}

	raw := strings.TrimSpace(string(content))
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return EnvCheckResult{}, fmt.Errorf("parse vm.max_map_count: %w", err)
	}

	if value >= 262144 {
		return EnvCheckResult{
			Result:              envCheckResultOptimal,
			Value:               raw,
			BadgeMessageI18nKey: "optimized",
		}, nil
	}

	return EnvCheckResult{
		Result:              envCheckResultSuboptimal,
		Value:               raw,
		MessageI18nKey:      "envCheckEntryMaxMapCountLow",
		BadgeMessageI18nKey: "optimizable",
	}, nil
}
