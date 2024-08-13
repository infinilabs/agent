/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
	"context"
	"os"
	"path/filepath"

	log "github.com/cihub/seelog"
)

type Operation uint8

const (
	OpDone Operation = iota
	OpCreate
	OpWrite
	OpTruncate
)

type FSEvent struct {
	Pattern *Pattern // matched pattern
	Path    string
	Offset  int64
	Op      Operation
	Info    os.FileInfo
	State   FileState
}

func NewFileDetector(rootPath string, patterns []*Pattern) *FileDetector {
	return &FileDetector{
		root:     rootPath,
		patterns: patterns,
		events:   make(chan FSEvent),
	}
}

type FileDetector struct {
	root     string
	patterns []*Pattern
	prev     map[string]os.FileInfo
	events   chan FSEvent
}

func (w *FileDetector) Detect(ctx context.Context) {
	defer func() {
		w.events <- doneEvent()
	}()

	if len(w.patterns) == 0 {
		return
	}
	err := filepath.Walk(w.root, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if info == nil {
			log.Warnf("missing file info for path [%s]", path)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		for _, pattern := range w.patterns {
			if !pattern.patternRegex.MatchString(info.Name()) {
				continue
			}
			w.judgeEvent(ctx, path, info, pattern)
			break
		}
		return nil
	})
	if err != nil {
		log.Errorf("failed to walk logs under [%s], err: %v", w.root, err)
	}
}

func (w *FileDetector) judgeEvent(ctx context.Context, path string, info os.FileInfo, pattern *Pattern) {
	preState, err := GetFileState(path)
	isSameFile := w.IsSameFile(preState, info, path)
	if err != nil || preState == (FileState{}) || !isSameFile {
		select {
		case <-ctx.Done():
			return
		case w.events <- createEvent(path, info, pattern, preState):
		}
		return
	}

	if preState.ModTime.UnixNano() != info.ModTime().UnixNano() {
		//mod time changed, if pre info has same size or bigger => truncate
		if preState.Size >= info.Size() {
			select {
			case <-ctx.Done():
				return
			case w.events <- truncateEvent(path, info, pattern, preState):
			}
		} else {
			select {
			case <-ctx.Done():
				return
			case w.events <- writeEvent(path, info, pattern, preState):
			}
		}
	}
}

func (w *FileDetector) Event() FSEvent {
	return <-w.events
}

func createEvent(path string, fi os.FileInfo, pattern *Pattern, state FileState) FSEvent {
	return FSEvent{pattern, path, -1, OpCreate, fi, state}
}

func writeEvent(path string, fi os.FileInfo, pattern *Pattern, state FileState) FSEvent {
	return FSEvent{pattern, path, -1, OpWrite, fi, state}
}

func truncateEvent(path string, fi os.FileInfo, pattern *Pattern, state FileState) FSEvent {
	return FSEvent{pattern, path, -1, OpTruncate, fi, state}
}

func doneEvent() FSEvent {
	return FSEvent{nil, "", -1, OpDone, nil, FileState{}}
}
