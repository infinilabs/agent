/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris

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
	devF64, ok := preStateMap["Dev"].(float64)
	if !ok {
		return false
	}
	inoF64, ok := preStateMap["Ino"].(float64)
	if !ok {
		return false
	}
	dev := int32(devF64)
	ino := uint64(inoF64)
	current := currentInfo.Sys().(*syscall.Stat_t)
	if current == nil {
		return false
	}
	return uint64(dev) == uint64(current.Dev) && ino == current.Ino
}