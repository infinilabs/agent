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
)

type Operation uint8

const (
	OpDone Operation = iota
	OpCreate
	OpWrite
	OpDelete
	OpTruncate
	OpArchived
)

var operationNames = map[Operation]string{
	OpDone:     "done",
	OpCreate:   "create",
	OpWrite:    "write",
	OpDelete:   "delete",
	OpTruncate: "truncate",
	OpArchived: "archive",
}

type FSEvent struct {
	Path    string `json:"path"`
	Op      Operation
	Info    os.FileInfo
	LogMeta LogMeta `json:"log_meta"`
	State   FileState
}

func NewFileWatcher() *FileWatcher {
	return &FileWatcher{
		events: make(chan FSEvent),
	}
}

type FileWatcher struct {
	prev   map[string]os.FileInfo
	events chan FSEvent
}

func (w *FileWatcher) Watch(metas []*LogMeta, ctx context.Context) {
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

func (w *FileWatcher) judgeEvent(path string, info os.FileInfo, meta LogMeta, ctx context.Context) {
	preState, err := GetFileState(path)
	if err != nil || preState.Name == ""{
		select {
		case <-ctx.Done():
			return
		case w.events <- createEvent(path, info, meta, preState):
		}
		return
	}

	if preState.ModTime != info.ModTime() {
		if preState.Size > info.Size() {
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

func (w *FileWatcher) Event() FSEvent {
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

func deleteEvent(path string, fi os.FileInfo, meta LogMeta, state FileState) FSEvent {
	return FSEvent{path, OpDelete, fi, meta, state}
}

func doneEvent() FSEvent {
	return FSEvent{"", OpDone, nil, LogMeta{}, FileState{}}
}
