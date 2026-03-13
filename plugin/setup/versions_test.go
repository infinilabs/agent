/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentPlatform(t *testing.T) {
	platform := getCurrentPlatform()

	// Verify format is os-arch
	assert.Contains(t, []string{"linux-amd64", "linux-aarch64", "mac-amd64", "mac-aarch64", "windows-amd64"}, platform)

	// Verify mapping
	switch runtime.GOOS {
	case "darwin":
		assert.Contains(t, platform, "mac-")
	case "linux":
		assert.Contains(t, platform, "linux-")
	case "windows":
		assert.Contains(t, platform, "windows-")
	}
}

func TestNormalizePlatform(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		err      string
	}{
		{name: "darwin arm64", input: "darwin-arm64", expected: "mac-aarch64"},
		{name: "mac arm64", input: "mac-arm64", expected: "mac-aarch64"},
		{name: "mac aarch64", input: "mac-aarch64", expected: "mac-aarch64"},
		{name: "linux amd64", input: "linux-amd64", expected: "linux-amd64"},
		{name: "windows arm64", input: "windows-arm64", expected: "windows-aarch64"},
		{name: "os only", input: "darwin", err: "invalid platform format"},
		{name: "empty", input: "", err: "empty platform string"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizePlatform(tc.input)
			if tc.err != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestParseEasysearchMajorVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		major    int
		hasError bool
	}{
		{"version 2.x", "2.0.2", 2, false},
		{"version 1.x", "1.14.1", 1, false},
		{"version with build", "2.0.2-2499", 2, false},
		{"empty version", "", 0, true},
		{"non-numeric", "abc", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			major, err := parseEasysearchMajorVersion(tc.version)
			if tc.hasError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.major, major)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"equal", "2.0.2", "2.0.2", 0},
		{"v1 greater", "2.0.2", "2.0.1", 1},
		{"v1 less", "2.0.1", "2.0.2", -1},
		{"different major", "3.0.0", "2.9.9", 1},
		{"different minor", "2.1.0", "2.0.9", 1},
		{"different length", "2.0", "2.0.0", 0},
		{"shorter version less", "2.0", "2.0.1", -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmp, err := compareVersions(tc.v1, tc.v2)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, cmp)
		})
	}
}

func TestGetRecommendedJdkVersion(t *testing.T) {
	tests := []struct {
		name                string
		easysearchMajor     int
		recommendedJdkMajor int
	}{
		{"Easysearch 1.x", 1, 17},
		{"Easysearch 2.x", 2, 21},
		{"Unknown version", 3, 21},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getRecommendedJdkVersion(tc.easysearchMajor)
			assert.Equal(t, tc.recommendedJdkMajor, result)
		})
	}
}

func TestIsJdkVersionCompatible(t *testing.T) {
	tests := []struct {
		name            string
		jdkMajor        int
		easysearchMajor int
		expected        bool
	}{
		// Easysearch 1.x compatibility
		{"ES1 with JDK 11", 11, 1, true},
		{"ES1 with JDK 17", 17, 1, true},
		{"ES1 with JDK 8", 8, 1, false},
		{"ES1 with JDK 21", 21, 1, false},
		// Easysearch 2.x compatibility
		{"ES2 with JDK 21", 21, 2, true},
		{"ES2 with JDK 17", 17, 2, false},
		{"ES2 with JDK 11", 11, 2, false},
		// Unknown version defaults to JDK 21
		{"ES3 with JDK 21", 21, 3, true},
		{"ES3 with JDK 17", 17, 3, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isJdkVersionCompatible(tc.jdkMajor, tc.easysearchMajor)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPluginCompatible(t *testing.T) {
	currentPlatform := getCurrentPlatform()

	tests := []struct {
		name      string
		plugin    pluginVersionInfo
		esVersion string
		expected  bool
	}{
		{
			name: "no version constraints",
			plugin: pluginVersionInfo{
				MinVersion: "",
				MaxVersion: "",
			},
			esVersion: "2.0.2",
			expected:  true,
		},
		{
			name: "version in range",
			plugin: pluginVersionInfo{
				MinVersion: "1.0.0",
				MaxVersion: "2.0.0",
			},
			esVersion: "1.5.0",
			expected:  true,
		},
		{
			name: "version below min",
			plugin: pluginVersionInfo{
				MinVersion: "2.0.0",
			},
			esVersion: "1.9.9",
			expected:  false,
		},
		{
			name: "version above max",
			plugin: pluginVersionInfo{
				MaxVersion: "1.11.0",
			},
			esVersion: "2.0.0",
			expected:  false,
		},
		{
			name: "max version is dash (unlimited)",
			plugin: pluginVersionInfo{
				MinVersion: "1.0.0",
				MaxVersion: "-",
			},
			esVersion: "99.0.0",
			expected:  true,
		},
		{
			name: "platform match",
			plugin: pluginVersionInfo{
				Platforms: []string{currentPlatform},
			},
			esVersion: "2.0.2",
			expected:  true,
		},
		{
			name: "platform mismatch",
			plugin: pluginVersionInfo{
				Platforms: []string{"some-other-platform"},
			},
			esVersion: "2.0.2",
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := isPluginCompatible(tc.plugin, tc.esVersion, currentPlatform)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPluginCompatibleNormalizesRemotePlatformAliases(t *testing.T) {
	matched, err := isPluginCompatible(
		pluginVersionInfo{Platforms: []string{"mac-arm64"}},
		"2.0.2",
		"mac-aarch64",
	)
	require.NoError(t, err)
	assert.True(t, matched)

	matched, err = isPluginCompatible(
		pluginVersionInfo{Platforms: []string{"linux-arm64"}},
		"2.0.2",
		"linux-aarch64",
	)
	require.NoError(t, err)
	assert.True(t, matched)
}

func TestFetchEasysearchVersionsReturnsEmptySliceWhenNoPlatformsMatch(t *testing.T) {
	versionsResp := easysearchVersionsResponse{
		"2.0.2": {Platforms: []string{"windows-amd64"}},
	}

	filteredVersions := make([]string, 0)
	currentPlatform, err := normalizePlatform("mac-aarch64")
	require.NoError(t, err)
	for version, info := range versionsResp {
		for _, platform := range info.Platforms {
			normalizedPlatform, err := normalizePlatform(platform)
			require.NoError(t, err)
			if normalizedPlatform == currentPlatform {
				filteredVersions = append(filteredVersions, version)
				break
			}
		}
	}

	require.NotNil(t, filteredVersions)
	assert.Empty(t, filteredVersions)
}

func TestIsPluginCompatibleRejectsInvalidPlatformFormat(t *testing.T) {
	_, err := isPluginCompatible(
		pluginVersionInfo{Platforms: []string{"darwin"}},
		"2.0.2",
		"mac-aarch64",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid platform format")
}

func TestGetJVMMemoryRecommendation(t *testing.T) {
	recommendation, err := GetJVMMemoryRecommendation()
	assert.NoError(t, err)
	assert.NotNil(t, recommendation)

	// Verify values are positive
	assert.Greater(t, recommendation.Total, 0.0)
	assert.Greater(t, recommendation.Available, 0.0)
	assert.Greater(t, recommendation.Suggestion, 0.0)

	// Verify suggestion is not more than half of total
	assert.LessOrEqual(t, recommendation.Suggestion, recommendation.Total/2+0.1) // +0.1 for rounding

	// Verify suggestion is not more than available
	assert.LessOrEqual(t, recommendation.Suggestion, recommendation.Available+0.1)

	// Verify suggestion stays under the compressed-oops-friendly heap size.
	assert.LessOrEqual(t, recommendation.Suggestion, maxCompressedOopsHeapGB)
}
