/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/host"
)

// Normalize arch names, convert aliases to standard names.
func normalizeArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "x86_64", "x64":
		return "amd64"
	case "arm64":
		return "aarch64"
	case "386", "i386", "i686":
		return "x86"
	default:
		return arch
	}
}

// Normalize OS names, convert aliases to standard names.
func normalizeOs(os string) string {
	switch strings.ToLower(strings.TrimSpace(os)) {
	case "darwin", "macos", "macosx", "osx":
		return "mac"
	default:
		return os
	}
}

func normalizePlatform(platform string) (string, error) {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return "", fmt.Errorf("empty platform string")
	}

	osPart, archPart, found := strings.Cut(platform, "-")
	if !found {
		return "", fmt.Errorf("invalid platform format: %s", platform)
	}

	osPart = normalizeOs(osPart)
	archPart = normalizeArch(archPart)
	if archPart == "" || osPart == "" {
		return osPart, fmt.Errorf("invalid platform format: [%s]", platform)
	}

	return osPart + "-" + archPart, nil
}

// getCurrentPlatform returns the normalized platform string.
func getCurrentPlatform() string {
	return normalizeOs(runtime.GOOS) + "-" + normalizeArch(runtime.GOARCH)
}

// Only the OS part changes
//
// mac -> macOS Xx
// linux -> os-release PRETTY_NAME="AlmaLinux 8.10 (Cerulean Leopard)"
// windows -> Windows 10 or Windows 11
func getPlatformDisplayName(platform string) string {
	trimmed := strings.TrimSpace(platform)
	if trimmed == "" {
		return ""
	}

	osPart, archPart, _ := strings.Cut(trimmed, "-")
	osPart = normalizeOs(osPart)
	archPart = normalizeArch(archPart)

	osDisplayName := getOSDisplayName(osPart)
	if archPart == "" {
		return osDisplayName
	}
	return osDisplayName + "/" + archPart
}

func getOSDisplayName(osPart string) string {
	switch normalizeOs(osPart) {
	case "mac":
		if normalizeOs(runtime.GOOS) == "mac" {
			if name := getMacOSDisplayName(); name != "" {
				return name
			}
		}
		return "macOS"
	case "linux":
		if normalizeOs(runtime.GOOS) == "linux" {
			if name := getLinuxDisplayName(); name != "" {
				return name
			}
		}
		return "Linux"
	case "windows":
		if normalizeOs(runtime.GOOS) == "windows" {
			if name := getWindowsDisplayName(); name != "" {
				return name
			}
		}
		return "Windows"
	default:
		return osPart
	}
}

func getMacOSDisplayName() string {
	info, err := host.Info()
	if err != nil {
		return ""
	}

	return formatVersionedOSDisplayName("macOS", info.Platform, info.PlatformVersion)
}

func getLinuxDisplayName() string {
	content, err := os.ReadFile("/etc/os-release")
	if err == nil {
		if prettyName := parseOSReleasePrettyName(string(content)); prettyName != "" {
			return prettyName
		}
	}

	info, err := host.Info()
	if err != nil {
		return ""
	}

	platform := strings.TrimSpace(info.Platform)
	version := strings.TrimSpace(info.PlatformVersion)
	if platform == "" {
		platform = "Linux"
	}
	if version == "" {
		return platform
	}
	if strings.Contains(platform, version) {
		return platform
	}
	return platform + " " + version
}

func getWindowsDisplayName() string {
	info, err := host.Info()
	if err != nil {
		return ""
	}

	return formatVersionedOSDisplayName("Windows", info.Platform, info.PlatformVersion)
}

func formatVersionedOSDisplayName(baseName, platformName, version string) string {
	baseName = strings.TrimSpace(baseName)
	platformName = strings.TrimSpace(platformName)
	version = strings.TrimSpace(version)

	if named := extractNamedOSVersion(baseName, platformName); named != "" {
		return named
	}

	if version != "" {
		return baseName + " " + version
	}
	return baseName
}

func extractNamedOSVersion(baseName, platformName string) string {
	baseLower := strings.ToLower(strings.TrimSpace(baseName))
	platformName = strings.TrimSpace(platformName)
	if platformName == "" {
		return ""
	}

	fields := strings.Fields(platformName)
	for i, field := range fields {
		lower := strings.ToLower(field)
		if lower != baseLower {
			continue
		}
		if i+1 >= len(fields) {
			return baseName
		}
		next := fields[i+1]
		if version := versionToken(next); version != "" {
			return baseName + " " + version
		}
		return baseName
	}

	return ""
}

func versionToken(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range version {
		if (r < '0' || r > '9') && r != '.' {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func parseOSReleasePrettyName(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "PRETTY_NAME=") {
			continue
		}

		value := strings.TrimSpace(strings.TrimPrefix(line, "PRETTY_NAME="))
		value = strings.Trim(value, `"'`)
		return value
	}
	return ""
}
