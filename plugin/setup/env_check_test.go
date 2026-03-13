/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListEnvCheckEntries(t *testing.T) {
	os := normalizeOs(runtime.GOOS)
	entries := listEnvCheckEntries(os)
	assert.NotEmpty(t, entries)

	var ids []string
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}

	assert.Contains(t, ids, "os_cpu")
	assert.NotContains(t, ids, "os")
	assert.NotContains(t, ids, "cpu")
	assert.Contains(t, ids, "kernel")
	assert.Contains(t, ids, "ram")
	assert.Contains(t, ids, "jdk")

	if os == "linux" || os == "mac" {
		assert.Contains(t, ids, "open_fd_limits")
	} else {
		assert.NotContains(t, ids, "open_fd_limits")
	}

	if os == "linux" {
		assert.Contains(t, ids, "max_map_count")
	} else {
		assert.NotContains(t, ids, "max_map_count")
	}
}

func TestParseJavaVersion(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		version  string
		major    int
		hasError bool
	}{
		{
			name:    "modern major version",
			output:  `openjdk version "21.0.2" 2024-01-16`,
			version: "21.0.2",
			major:   21,
		},
		{
			name:    "legacy java 8 style version",
			output:  `java version "1.8.0_431"`,
			version: "1.8.0_431",
			major:   8,
		},
		{
			name:     "invalid output",
			output:   `openjdk unknown`,
			hasError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			version, major, err := parseJavaVersion(tc.output)
			if tc.hasError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.version, version)
			assert.Equal(t, tc.major, major)
		})
	}
}
