/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/lib/reader/linenumber"
	util2 "infini.sh/agent/lib/util"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func (handler *AgentAPI) getElasticLogFiles(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	reqBody := GetElasticLogFilesReq{}
	handler.DecodeJSON(req, &reqBody)
	if reqBody.LogsPath == "" {
		handler.WriteError(w, "miss param logs_path", http.StatusInternalServerError)
		return
	}

	fileInfos, err := os.ReadDir(reqBody.LogsPath)
	if err != nil {
		log.Error(err)
		handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var files []util.MapStr
	for _, info := range fileInfos {
		if info.IsDir() {
			continue
		}
		fInfo, err := info.Info()
		if err != nil {
			log.Error(err)
			continue
		}
		filePath := path.Join(reqBody.LogsPath, info.Name())
		totalRows, err := util2.CountFileRows(filePath)
		if err != nil {
			log.Error(err)
			continue
		}
		files = append(files, util.MapStr{
			"name":          fInfo.Name(),
			"size_in_bytes": fInfo.Size(),
			"modify_time":   fInfo.ModTime(),
			"total_rows":    totalRows,
		})
	}

	handler.WriteJSON(w, util.MapStr{
		"result":  files,
		"success": true,
	}, http.StatusOK)
}

func (handler *AgentAPI) readElasticLogFile(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	reqBody := ReadElasticLogFileReq{}
	err := handler.DecodeJSON(req, &reqBody)
	if err != nil {
		log.Error(err)
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logFilePath := filepath.Join(reqBody.LogsPath, reqBody.FileName)
	if reqBody.StartLineNumber < 0 {
		reqBody.StartLineNumber = 0
	}
	if strings.HasSuffix(reqBody.FileName, ".gz") {
		// read gzip log file, and then unpack it to tmp file
		tmpFilePath := filepath.Join(os.TempDir(), "agent", strings.TrimSuffix(reqBody.FileName, ".gz"))
		if !util.FileExists(tmpFilePath) {
			fileDir := filepath.Dir(tmpFilePath)
			if !util.FileExists(fileDir) {
				err = os.MkdirAll(fileDir, os.ModePerm)
				if err != nil {
					log.Error(err)
					handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			err = util2.UnpackGzipFile(logFilePath, tmpFilePath)
			if err != nil {
				log.Error(err)
				handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		logFilePath = tmpFilePath
	}
	r, err := linenumber.NewLinePlainTextReader(logFilePath, reqBody.StartLineNumber, io.SeekStart)
	if err != nil {
		log.Error(err)
		handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				log.Error("error on exit disk_queue,", v)
			}
		}
		if r != nil {
			r.Close()
		}
	}()

	var msgs []util.MapStr
	isEOF := false
	for i := 0; i < reqBody.Lines; i++ {
		msg, err := r.Next()
		if err != nil {
			if err == io.EOF {
				isEOF = true
				break
			} else {
				log.Error(err)
				handler.WriteError(w, fmt.Sprintf("read logs error: %v", err), http.StatusInternalServerError)
				return
			}
		}
		msgs = append(msgs, util.MapStr{
			"content":     string(msg.Content),
			"bytes":       msg.Bytes,
			"offset":      msg.Offset,
			"line_number": coverLineNumbers(msg.LineNumbers),
		})
	}
	handler.WriteJSON(w, util.MapStr{
		"result":  msgs,
		"success": true,
		"EOF":     isEOF,
	}, http.StatusOK)
}

func coverLineNumbers(numbers []int64) interface{} {
	if len(numbers) == 1 {
		return numbers[0]
	} else {
		return numbers
	}
}
