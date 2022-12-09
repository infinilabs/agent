/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
	"context"
	log "github.com/cihub/seelog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type Operation uint8

const (
	OpDone Operation = iota
	OpCreate
	OpWrite
	OpTruncate
)

type FSEvent struct {
	Path    string `json:"path"`
	Op      Operation
	Info    os.FileInfo
	LogMeta LogMeta `json:"log_meta"`
	State   FileState
}

func NewFileDetector() *FileDetector {
	return &FileDetector{
		events: make(chan FSEvent),
	}
}

type FileDetector struct {
	prev   map[string]os.FileInfo
	events chan FSEvent
}

func (w *FileDetector) Detect(metas []*LogMeta, ctx context.Context) {
	defer func() {
		w.events <- doneEvent()
	}()

	if len(metas) == 0 {
		return
	}
	for _, meta := range metas {
		if ctx.Err() != nil {
			return
		}
		root := meta.File.Path
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if info.IsDir() {
				return nil
			}
			if !strings.EqualFold(".json", filepath.Ext(path)) && !strings.EqualFold(info.Name(), "gc.log") {
				return nil
			}
			w.judgeEvent(path, info, *meta, ctx)
			return nil
		})
		if err != nil {
			log.Error(err)
		}
	}
}

func (w *FileDetector) judgeEvent(path string, info os.FileInfo, meta LogMeta, ctx context.Context) {
	preState, err := GetFileState(path)
	if err != nil || preState == (FileState{}) || !w.IsSameFile(preState, info){
		select {
		case <-ctx.Done():
			return
		case w.events <- createEvent(path, info, meta, preState):
		}
		return
	}

	if preState.ModTime != info.ModTime() {
		//mod time changed, if pre info has same size or bigger => truncate
		if preState.Size >= info.Size() {
			select {
			case <-ctx.Done():
				return
			case w.events <- truncateEvent(path, info, meta, preState):
			}
		} else {
			select {
			case <-ctx.Done():
				return
			case w.events <- writeEvent(path, info, meta, preState):
			}
		}
	}
}

// IsSameFile whether preState's file info and current file info describe the same file
func (w *FileDetector) IsSameFile(preState FileState, currentInfo os.FileInfo) bool {
	if preState == (FileState{}) {
		return false
	}
	preStateMap, ok := preState.Sys.(map[string]interface{})
	if !ok {
		return false
	}
	DevF64, ok := preStateMap["Dev"].(float64)
	if !ok {
		return false
	}
	InoF64, ok := preStateMap["Ino"].(float64)
	if !ok {
		return false
	}
	Dev := int32(DevF64)
	Ino := uint64(InoF64)
	current := currentInfo.Sys().(*syscall.Stat_t)
	if current == nil {
		return false
	}
	return Dev == current.Dev && Ino == current.Ino
}

func (w *FileDetector) Event() FSEvent {
	return <-w.events
}

func createEvent(path string, fi os.FileInfo, meta LogMeta, state FileState) FSEvent {
	return FSEvent{path, OpCreate, fi, meta, state}
}

func writeEvent(path string, fi os.FileInfo, meta LogMeta, state FileState) FSEvent {
	return FSEvent{path, OpWrite, fi, meta, state}
}

func truncateEvent(path string, fi os.FileInfo, meta LogMeta, state FileState) FSEvent {
	return FSEvent{path, OpTruncate, fi, meta, state}
}

func doneEvent() FSEvent {
	return FSEvent{"", OpDone, nil, LogMeta{}, FileState{}}
}
