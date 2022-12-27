/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
	"os"
	"syscall"
)

// IsSameFile whether preState's file info and current file info describe the same file
func (w *FileDetector) IsSameFile(preState FileState, currentInfo os.FileInfo) bool {
	if preState == (FileState{}) {
		return false
	}
	preStateMap, ok := preState.Sys.(map[string]interface{})
	if !ok {
		return false
	}
	ctime, ok := preStateMap["CreationTime"]
	if !ok {
		return false
	}

	current := currentInfo.Sys().(*syscall.Win32FileAttributeData)
	if current == nil {
		return false
	}
	return ctime == current.CreationTime
}
