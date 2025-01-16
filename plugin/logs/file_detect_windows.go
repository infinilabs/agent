/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
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
	vol, ok := preStateMap["VolumeSerialNumber"].(float64)
	if !ok {
		return false
	}
	idxhi, ok := preStateMap["FileIndexHigh"].(float64)
	if !ok {
		return false
	}
	idxlo, ok := preStateMap["FileIndexLow"].(float64)
	if !ok {
		return false
	}
	fstate, err := LoadFileID(currentInfo, path)
	if err != nil {
		return false
	}

	return fstate["VolumeSerialNumber"] == uint32(vol) && fstate["FileIndexHigh"] == uint32(idxhi) && fstate["FileIndexLow"] == uint32(idxlo)
}

func LoadFileID(fi os.FileInfo, path string) (map[string]interface{}, error) {
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	attrs := uint32(syscall.FILE_FLAG_BACKUP_SEMANTICS | syscall.FILE_FLAG_OPEN_REPARSE_POINT)

	h, err := syscall.CreateFile(pathp, 0, 0, nil, syscall.OPEN_EXISTING, attrs, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(h)
	var i syscall.ByHandleFileInformation
	err = syscall.GetFileInformationByHandle(h, &i)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"VolumeSerialNumber": i.VolumeSerialNumber,
		"FileIndexHigh":      i.FileIndexHigh,
		"FileIndexLow":       i.FileIndexLow,
	}, nil
}
