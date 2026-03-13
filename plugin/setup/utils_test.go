/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOSReleasePrettyName(t *testing.T) {
	content := `NAME="AlmaLinux"
VERSION="8.10 (Cerulean Leopard)"
PRETTY_NAME="AlmaLinux 8.10 (Cerulean Leopard)"
ID="almalinux"`

	assert.Equal(t, "AlmaLinux 8.10 (Cerulean Leopard)", parseOSReleasePrettyName(content))
}

func TestParseOSReleasePrettyNameMissing(t *testing.T) {
	assert.Equal(t, "", parseOSReleasePrettyName("NAME=Linux\nVERSION=1"))
}

func TestGetPlatformDisplayNameFormatsArch(t *testing.T) {
	assert.Equal(t, "", getPlatformDisplayName(""))
	assert.Equal(t, "freebsd/aarch64", getPlatformDisplayName("freebsd-arm64"))

	displayName := getPlatformDisplayName(normalizeOs(runtime.GOOS) + "-" + normalizeArch(runtime.GOARCH))
	assert.NotEmpty(t, displayName)
	assert.True(t, strings.HasSuffix(displayName, "/"+normalizeArch(runtime.GOARCH)))
}

func TestFormatVersionedOSDisplayName(t *testing.T) {
	assert.Equal(t, "macOS 26.0.1", formatVersionedOSDisplayName("macOS", "Darwin", "26.0.1"))
	assert.Equal(t, "Windows 10", formatVersionedOSDisplayName("Windows", "Microsoft Windows 10 Pro", "10.0.19045"))
	assert.Equal(t, "Windows 11", formatVersionedOSDisplayName("Windows", "Microsoft Windows 11 Pro", "10.0.26100"))
	assert.Equal(t, "macOS", formatVersionedOSDisplayName("macOS", "", ""))
}

func TestVersionToken(t *testing.T) {
	assert.Equal(t, "26.1.0", versionToken("26.1.0"))
	assert.Equal(t, "10.0.26100", versionToken("10.0.26100"))
	assert.Equal(t, "", versionToken("v26"))
	assert.Equal(t, "", versionToken(""))
}
