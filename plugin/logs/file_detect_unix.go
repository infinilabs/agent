//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris

/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
	"fmt"
	"os"
	"syscall"
)

// IsSameFile whether preState's file info and current file info describe the same file
func (w *FileDetector) IsSameFile(preState FileState, currentInfo os.FileInfo, path string) bool {
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

func LoadFileID(fi os.FileInfo, path string) (map[string]interface{}, error) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("failed to cast file info sys to stat_t")
	}
	id := map[string]interface{}{
		"Dev": st.Dev,
		"Ino": st.Ino,
	}
	return id, nil
}
