/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package util

import (
	"bufio"
	"os"
)

func CountFileRows(filePath string) (int64, error){
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
