/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"time"
)

const (
	KVLogfileStateBucket = "log_state_bucket"
	KVLogfilePathsBucket = "log_paths_bucket"
	KVLogfilePaths       = "log_paths_info"
)

type FileState struct {
	Name    string    `json:"name"`     // base name of the file
	Size    int64     `json:"size"`     // length in bytes for regular files; system-dependent for others
	ModTime time.Time `json:"mod_time"` // modification time
	IsDir   bool      `json:"is_dir"`   // abbreviation for Mode().IsDir()
	Path    string    `json:"path"`
	OffSet  int64     `json:"off_set"`
}

func SavePaths(paths []string) {
	kv.AddValue(KVLogfilePathsBucket, []byte(KVLogfilePaths), util.MustToJSONBytes(paths))
}

func GetPaths() []string {
	ret, err := kv.GetValue(KVLogfilePathsBucket, []byte(KVLogfilePaths))
	if err != nil {
		log.Error(err)
		return nil
	}
	var paths []string
	err = json.Unmarshal(ret, &paths)
	if err != nil {
		log.Error(err)
		return nil
	}
	return paths
}

func SaveFileState(path string, source FileState) {
	err := kv.AddValue(KVLogfileStateBucket, []byte(path), util.MustToJSONBytes(source))
	if err != nil {
		log.Error(err)
	}
}

func GetFileState(path string) (FileState, error) {
	ret, err := kv.GetValue(KVLogfileStateBucket, []byte(path))
	if err != nil {
		return FileState{}, err
	}
	var state FileState
	err = json.Unmarshal(ret, &state)
	if err != nil {
		return FileState{}, err
	}
	return state, nil
}

func RemoveFileState(path string) {
	kv.DeleteKey(KVLogfileStateBucket, []byte(path))
}

type LogEvent struct {
	Timestamp time.Time       `json:"timestamp,omitempty" elastic_mapping:"timestamp: { type: date }"`
	AgentMeta event.AgentMeta `json:"agent"`
	Meta      LogMeta         `json:"metadata"`
	Fields    util.MapStr     `json:"payload"`
}

type LogMeta struct {
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Cluster  Cluster `json:"cluster"`
	Node     Node    `json:"node"`
	File     File    `json:"file"`
}

type Cluster struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	UUID string `json:"uuid"`
}

type Node struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Port int    `json:"port"`
}

type File struct {
	Path   string `json:"path"`
	Offset int64  `json:"offset"`
}
