/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/lib/reader/harvester"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
		if strings.HasSuffix(info.Name(), ".log") || strings.HasSuffix(info.Name(), ".json") {
			if info.IsDir() {
				continue
			}
			fInfo, err := info.Info()
			if err != nil {
				log.Error(err)
				continue
			}
			files = append(files, util.MapStr{
				"name":          fInfo.Name(),
				"size_in_bytes": fInfo.Size(),
				"modify_time":   fInfo.ModTime(),
			})
		}
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
	h, err := harvester.NewHarvester(logFilePath, reqBody.Offset)
	if err != nil {
		log.Error(err)
		handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}
	r, err := h.NewPlainTextRead(true)
	if err != nil {
		log.Error(err)
		handler.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
			"content": string(msg.Content),
			"bytes": msg.Bytes,
			"offset": msg.Offset,
			"line_number": coverLineNumbers(msg.LineNumbers),
		})
	}
	if h.Close() != nil {
		log.Error(err)
		handler.WriteError(w, fmt.Sprintf("close reader error: %v", err), http.StatusInternalServerError)
		return
	}
	handler.WriteJSON(w, util.MapStr{
		"result":  msgs,
		"success": true,
		"EOF": isEOF,
	}, http.StatusOK)
}

func coverLineNumbers(numbers []int) interface{}{
	if len(numbers) == 1 {
		return numbers[0]
	} else {
		return numbers
	}
}
