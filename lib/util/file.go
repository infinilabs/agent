/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package util

import (
	"bufio"
	"os"
	"path/filepath"
)

func CountFileRows(filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var count int64 = 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}
	return count, nil
}

// ResolveSymlink Parse the target file path of the soft link
func ResolveSymlink(link string) (string, error) {
	realPath, err := filepath.EvalSymlinks(link)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(realPath)
	if err != nil {
		return "", err
	}
	return absPath, nil
}
