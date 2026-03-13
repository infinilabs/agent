/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	httprouter "infini.sh/framework/core/api/router"
)

const (
	easysearchVersionsURL = "https://release.infinilabs.com/easysearch/.versions.json"
	pluginsVersionsURL    = "https://release.infinilabs.com/easysearch/.plugins-versions.json"
	httpTimeout           = 30 * time.Second
)

// Remote API response structures

type easysearchVersionsResponse map[string]easysearchVersionInfo

type easysearchVersionInfo struct {
	Platforms   []string `json:"platforms"`
	BuildNumber int      `json:"build_number"`
}

type pluginsVersionsResponse map[string]pluginVersionInfo

type pluginVersionInfo struct {
	MinVersion  string            `json:"min_version,omitempty"`
	MaxVersion  string            `json:"max_version,omitempty"`
	Platforms   []string          `json:"platforms,omitempty"`
	Description pluginDescription `json:"description"`
}

type pluginDescription struct {
	En   string `json:"en"`
	ZhCN string `json:"zh_cn"`
}

// API response structures

// JdkInfo represents a JDK installation (remote or local)
type JdkInfo struct {
	Source  string `json:"source"`         // "remote" or "local"
	Version string `json:"version"`        // e.g., "21", "17"
	Path    string `json:"path,omitempty"` // Only for local JDKs
}

// PluginInfo represents a plugin compatible with the Easysearch version
type PluginInfo struct {
	Name        string            `json:"name"`
	Description pluginDescription `json:"description"`
}

// JdkEasysearchPluginsVersionsResponse is the response for GET /setup/jdk_easysearch_plugins/versions
type JdkEasysearchPluginsVersionsResponse struct {
	Jdk               []JdkInfo    `json:"jdk"`
	EasysearchPlugins []PluginInfo `json:"easysearch_plugins"`
}

// getEasysearchVersions handles GET /setup/easysearch/versions
func (h *SetupAPI) getEasysearchVersions(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	versions, err := FetchEasysearchVersions()
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusBadGateway)
		return
	}

	h.WriteJSON(w, versions, http.StatusOK)
}

// FetchEasysearchVersions fetches and filters Easysearch versions for current platform
func FetchEasysearchVersions() ([]string, error) {
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(easysearchVersionsURL)
	if err != nil {
		return nil, fmt.Errorf("fetch easysearch versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch easysearch versions: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read easysearch versions response: %w", err)
	}

	var versionsResp easysearchVersionsResponse
	if err := json.Unmarshal(body, &versionsResp); err != nil {
		return nil, fmt.Errorf("parse easysearch versions: %w", err)
	}

	currentPlatform := getCurrentPlatform()
	filteredVersions := make([]string, 0)

	for version, info := range versionsResp {
		for _, platform := range info.Platforms {
			normalizedPlatform, err := normalizePlatform(platform)
			if err != nil {
				return nil, fmt.Errorf("normalize remote platform %q: %w", platform, err)
			}
			if normalizedPlatform == currentPlatform {
				filteredVersions = append(filteredVersions, version)
				break
			}
		}
	}

	// Sort versions descending (newest first)
	sort.Slice(filteredVersions, func(i, j int) bool {
		cmp, _ := compareVersions(filteredVersions[i], filteredVersions[j])
		return cmp > 0
	})

	return filteredVersions, nil
}

// getJdkEasysearchPluginsVersions handles GET /setup/jdk_easysearch_plugins/versions
func (h *SetupAPI) getJdkEasysearchPluginsVersions(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	easysearchVersion := strings.TrimSpace(h.GetParameter(req, "easysearch_version"))
	if easysearchVersion == "" {
		h.WriteError(w, "missing easysearch_version parameter", http.StatusBadRequest)
		return
	}

	// Validate version format
	if _, err := parseEasysearchMajorVersion(easysearchVersion); err != nil {
		h.WriteError(w, "invalid easysearch_version format", http.StatusBadRequest)
		return
	}

	response, err := FetchJdkAndPlugins(easysearchVersion)
	if err != nil {
		if strings.Contains(err.Error(), "fetch") || strings.Contains(err.Error(), "parse") {
			h.WriteError(w, err.Error(), http.StatusBadGateway)
		} else {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	h.WriteJSON(w, response, http.StatusOK)
}

// FetchJdkAndPlugins fetches JDK info and plugin list for a given Easysearch version
func FetchJdkAndPlugins(easysearchVersion string) (*JdkEasysearchPluginsVersionsResponse, error) {
	majorVersion, err := parseEasysearchMajorVersion(easysearchVersion)
	if err != nil {
		return nil, fmt.Errorf("parse easysearch version: %w", err)
	}

	response := &JdkEasysearchPluginsVersionsResponse{}

	// Build JDK list
	recommendedJdk := getRecommendedJdkVersion(majorVersion)
	response.Jdk = []JdkInfo{
		{Source: "remote", Version: strconv.Itoa(recommendedJdk)},
	}

	localJdks, err := getLocalJdks(majorVersion)
	if err != nil {
		return nil, fmt.Errorf("search local jdk: %w", err)
	}
	response.Jdk = append(response.Jdk, localJdks...)

	// Fetch plugins
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(pluginsVersionsURL)
	if err != nil {
		return nil, fmt.Errorf("fetch plugins versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch plugins versions: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read plugins versions response: %w", err)
	}

	var pluginsResp pluginsVersionsResponse
	if err := json.Unmarshal(body, &pluginsResp); err != nil {
		return nil, fmt.Errorf("parse plugins versions: %w", err)
	}

	// Filter plugins
	currentPlatform := getCurrentPlatform()
	for name, info := range pluginsResp {
		compatible, err := isPluginCompatible(info, easysearchVersion, currentPlatform)
		if err != nil {
			return nil, fmt.Errorf("check plugin %s compatibility: %w", name, err)
		}
		if compatible {
			response.EasysearchPlugins = append(response.EasysearchPlugins, PluginInfo{
				Name:        name,
				Description: info.Description,
			})
		}
	}

	// Sort plugins alphabetically
	sort.Slice(response.EasysearchPlugins, func(i, j int) bool {
		return response.EasysearchPlugins[i].Name < response.EasysearchPlugins[j].Name
	})

	return response, nil
}

// parseEasysearchMajorVersion extracts major version from version string (e.g., "2.0.2" -> 2)
func parseEasysearchMajorVersion(version string) (int, error) {
	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid version format: %s", version)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("parse major version: %w", err)
	}
	return major, nil
}

// getRecommendedJdkVersion returns the recommended JDK version for an Easysearch major version
func getRecommendedJdkVersion(easysearchMajorVersion int) int {
	switch easysearchMajorVersion {
	case 1:
		return 17
	case 2:
		return 21
	default:
		return 21 // Default to latest
	}
}

// isJdkVersionCompatible checks if a JDK major version is compatible with an Easysearch major version
func isJdkVersionCompatible(jdkMajorVersion, easysearchMajorVersion int) bool {
	switch easysearchMajorVersion {
	case 1:
		return jdkMajorVersion >= 11 && jdkMajorVersion <= 17
	case 2:
		return jdkMajorVersion == 21
	default:
		return jdkMajorVersion == 21
	}
}

// getLocalJdks searches for locally installed JDKs compatible with the Easysearch version
func getLocalJdks(easysearchMajorVersion int) ([]JdkInfo, error) {
	allJdks := findAllLocalJdks()
	var jdks []JdkInfo
	for _, jdk := range allJdks {
		if isJdkVersionCompatible(jdk.Major, easysearchMajorVersion) {
			jdks = append(jdks, JdkInfo{
				Source:  "local",
				Version: strconv.Itoa(jdk.Major),
				Path:    jdk.HomePath,
			})
		}
	}
	return jdks, nil
}

// isPluginCompatible checks if a plugin is compatible with the given Easysearch version and platform

func isPluginCompatible(plugin pluginVersionInfo, easysearchVersion string, currentPlatform string) (bool, error) {
	// Check version range
	if plugin.MinVersion != "" {
		if cmp, err := compareVersions(easysearchVersion, plugin.MinVersion); err == nil && cmp < 0 {
			return false, nil
		}
	}

	if plugin.MaxVersion != "" && plugin.MaxVersion != "-" {
		if cmp, err := compareVersions(easysearchVersion, plugin.MaxVersion); err == nil && cmp > 0 {
			return false, nil
		}
	}

	// Check platform compatibility
	if len(plugin.Platforms) > 0 {
		found := false
		for _, platform := range plugin.Platforms {
			normalizedPlatform, err := normalizePlatform(platform)
			if err != nil {
				return false, fmt.Errorf("normalize plugin platform %q: %w", platform, err)
			}
			if normalizedPlatform == currentPlatform {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}

	return true, nil
}

// compareVersions compares two semantic version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) (int, error) {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		var err error

		if i < len(parts1) {
			n1, err = strconv.Atoi(parts1[i])
			if err != nil {
				return 0, fmt.Errorf("parse version %s: %w", v1, err)
			}
		}

		if i < len(parts2) {
			n2, err = strconv.Atoi(parts2[i])
			if err != nil {
				return 0, fmt.Errorf("parse version %s: %w", v2, err)
			}
		}

		if n1 < n2 {
			return -1, nil
		} else if n1 > n2 {
			return 1, nil
		}
	}

	return 0, nil
}
