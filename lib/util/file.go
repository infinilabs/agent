// CountFileRows counts the number of rows in the specified file.
// It takes a file path as an argument and returns the row count and an error if any.
// If the file cannot be opened, it returns an error.
//
// Parameters:
//   - filePath: The path to the file to be read.
//
// Returns:
//   - int64: The number of rows in the file.
//   - error: An error if the file cannot be opened or read.
//
// Example usage:
//   count, err := CountFileRows("/path/to/file")
//   if err != nil {
//       log.Fatal(err)
//   }
//   fmt.Println("Number of rows:", count)

// ResolveSymlink resolves the target file path of a symbolic link.
// It takes a symbolic link path as an argument and returns the absolute path of the target file and an error if any.
// If the symbolic link cannot be resolved, it returns an error.
//
// Parameters:
//   - link: The path to the symbolic link.
//
// Returns:
//   - string: The absolute path of the target file.
//   - error: An error if the symbolic link cannot be resolved.
//
// Example usage:
//   realPath, err := ResolveSymlink("/path/to/symlink")
//   if err != nil {
//       log.Fatal(err)
//   }
//   fmt.Println("Resolved path:", realPath)
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
