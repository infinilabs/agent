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
)

type FileState struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Path    string    `json:"path"`
	Offset  int64     `json:"offset"`
	Sys     any       `json:"sys"`
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

type LogEvent struct {
	Created   time.Time 	  `json:"created,omitempty" elastic_mapping:"created: { type: date }"`
	AgentMeta event.AgentMeta `json:"agent" elastic_mapping:"agent: { type: object }"`
	Meta      LogMeta         `json:"metadata" elastic_mapping:"metadata: { type: object }"`
	Fields    util.MapStr     `json:"payload" elastic_mapping:"payload: { type: object }"`
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
