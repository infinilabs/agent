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
	logsPaths := normalizeJSONLogsPaths(reqBody.LogsPath)
	if len(logsPaths) == 0 {
		handler.WriteError(w, "miss param logs_path", http.StatusInternalServerError)
		return
	}

	var files []util.MapStr
	var lastErr error
	for _, logsPath := range logsPaths {
		fileInfos, err := os.ReadDir(logsPath)
		if err != nil {
			log.Errorf("failed to read elasticsearch logs directory [%s]: %v", logsPath, err)
			lastErr = err
			continue
		}
		for _, info := range fileInfos {
			if info.IsDir() {
				continue
			}
			fInfo, err := info.Info()
			if err != nil {
				log.Errorf("failed to read file info in logs directory [%s], file=[%s]: %v", logsPath, info.Name(), err)
				continue
			}
			filePath := path.Join(logsPath, info.Name())
			totalRows, err := util2.CountFileRows(filePath)
			if err != nil {
				log.Errorf("failed to count rows for log file [%s]: %v", filePath, err)
				continue
			}
			files = append(files, util.MapStr{
				"name":          fInfo.Name(),
				"logs_path":     logsPath,
				"size_in_bytes": fInfo.Size(),
				"modify_time":   fInfo.ModTime(),
				"total_rows":    totalRows,
			})
		}
	}
	if len(files) == 0 && lastErr != nil {
		handler.WriteJSON(w, lastErr.Error(), http.StatusInternalServerError)
		return
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
		log.Errorf("failed to decode elastic log read request: %v", err)
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logFilePath, err := safeJoinLogsFile(reqBody.LogsPath, reqBody.FileName)
	if err != nil {
		log.Errorf("invalid elastic log file request, logs_path=[%s], file_name=[%s]: %v", reqBody.LogsPath, reqBody.FileName, err)
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
					log.Errorf("failed to create temporary log directory [%s] for source [%s]: %v", fileDir, logFilePath, err)
					handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			err = util2.UnpackGzipFile(logFilePath, tmpFilePath)
			if err != nil {
				log.Errorf("failed to unpack gzip log file from [%s] to [%s]: %v", logFilePath, tmpFilePath, err)
				handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		logFilePath = tmpFilePath
	}
	r, err := linenumber.NewLinePlainTextReader(logFilePath, reqBody.StartLineNumber, io.SeekStart)
	if err != nil {
		log.Errorf("failed to open elastic log file [%s], start_line_number=[%d]: %v", logFilePath, reqBody.StartLineNumber, err)
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
				log.Errorf("failed to read elastic log file [%s] at start_line_number=[%d]: %v", logFilePath, reqBody.StartLineNumber, err)
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

func normalizeJSONLogsPaths(raw interface{}) []string {
	items := make([]string, 0)
	switch v := raw.(type) {
	case string:
		items = append(items, v)
	case []string:
		items = append(items, v...)
	case []interface{}:
		for _, item := range v {
			items = append(items, util.ToString(item))
		}
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		item = filepath.Clean(item)
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func safeJoinLogsFile(logsPath, fileName string) (string, error) {
	logsPath = strings.TrimSpace(logsPath)
	fileName = strings.TrimSpace(fileName)
	if logsPath == "" || fileName == "" {
		return "", fmt.Errorf("invalid log file request")
	}

	basePath := filepath.Clean(logsPath)
	fullPath := filepath.Clean(filepath.Join(basePath, fileName))
	rel, err := filepath.Rel(basePath, fullPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid log file path")
	}
	return fullPath, nil
}
