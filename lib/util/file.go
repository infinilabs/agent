/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package util

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
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

// UnpackGzipFile extracts a .gz file and writes the output to a regular file.
func UnpackGzipFile(gzFile, outputFile string) error {
	f, err := os.Open(gzFile)
	if err != nil {
		return fmt.Errorf("failed to open .gz file %s: %w", gzFile, err)
	}
	defer f.Close()

	// Create a gzip reader
	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader with file %s: %w", gzFile, err)
	}
	defer gzReader.Close()

	// Create the output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFile, err)
	}
	defer outFile.Close()

	// Copy decompressed content to output file
	_, err = io.Copy(outFile, gzReader)
	if err != nil {
		return fmt.Errorf("failed to write to output file %s: %w", outputFile, err)
	}
	return nil
}
