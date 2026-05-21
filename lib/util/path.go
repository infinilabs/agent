/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ExpandHomeDir(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("path is empty")
	}

	homePrefix := "~" + string(filepath.Separator)
	if trimmed == "~" || strings.HasPrefix(trimmed, homePrefix) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		if trimmed == "~" {
			return home, nil
		}
		return filepath.Join(home, trimmed[len(homePrefix):]), nil
	}

	return trimmed, nil
}
