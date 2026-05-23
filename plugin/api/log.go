/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	log "github.com/cihub/seelog"
	agent_util "infini.sh/agent/lib/util"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

const (
	logReadBufferSize = 64 * 1024
	maxReadLines      = 200
	maxCountRowsBytes = 8 * 1024 * 1024
)

func (handler *AgentAPI) getElasticLogFiles(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	handler.getSearchLogFiles(w, req, params)
}

func (handler *AgentAPI) getSearchLogFiles(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	reqBody := GetSearchLogFilesReq{}
	err := handler.DecodeJSON(req, &reqBody)
	if err != nil {
		log.Errorf("failed to decode search log files request: %v", err)
		handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logsPaths := normalizeJSONLogsPaths(reqBody.LogsPath)
	if len(logsPaths) == 0 {
		handler.WriteError(w, "miss param logs_path", http.StatusInternalServerError)
		return
	}

	var files []util.MapStr
	var errors []string
	appendError := func(format string, args ...interface{}) {
		errMsg := fmt.Sprintf(format, args...)
		log.Error(errMsg)
		errors = append(errors, errMsg)
	}

	for _, logsPath := range logsPaths {
		resolvedLogsPath, err := resolveLogsDirectory(logsPath)
		if err != nil {
			appendError("failed to resolve search logs directory [%s]: %v", logsPath, err)
			continue
		}

		fileInfos, err := os.ReadDir(resolvedLogsPath)
		if err != nil {
			appendError("failed to read search logs directory [%s]: %v", resolvedLogsPath, err)
			continue
		}

		for _, info := range fileInfos {
			if info.IsDir() {
				continue
			}
			fInfo, err := info.Info()
			if err != nil {
				appendError("failed to read file info in logs directory [%s], file=[%s]: %v", resolvedLogsPath, info.Name(), err)
				continue
			}
			filePath := path.Join(resolvedLogsPath, info.Name())
			fileItem := util.MapStr{
				"name":             fInfo.Name(),
				"logs_path":        resolvedLogsPath,
				"size_in_bytes":    fInfo.Size(),
				"modify_time":      fInfo.ModTime(),
				"total_rows_known": false,
			}
			if shouldCountLogRows(fInfo.Size()) {
				totalRows, err := agent_util.CountFileRows(filePath)
				if err != nil {
					appendError("failed to count rows for log file [%s]: %v", filePath, err)
					continue
				}
				fileItem["total_rows"] = totalRows
				fileItem["total_rows_known"] = true
			}
			files = append(files, fileItem)
		}
	}

	if len(errors) > 0 {
		handler.WriteJSON(w, util.MapStr{
			"result":  files,
			"errors":  errors,
			"success": false,
		}, http.StatusOK)
		return
	}

	handler.WriteJSON(w, util.MapStr{
		"result":  files,
		"success": true,
	}, http.StatusOK)
}

func (handler *AgentAPI) readElasticLogFile(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	handler.readSearchLogFile(w, req, params)
}

func (handler *AgentAPI) readSearchLogFile(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
	reqBody := ReadSearchLogFileReq{}
	err := handler.DecodeJSON(req, &reqBody)
	if err != nil {
		log.Errorf("failed to decode search log read request: %v", err)
		handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reqBody.LogsPath = strings.TrimSpace(reqBody.LogsPath)
	if reqBody.LogsPath == "" {
		handler.WriteError(w, "miss param logs_path", http.StatusInternalServerError)
		return
	}

	reqBody.LogsPath, err = resolveLogsDirectory(reqBody.LogsPath)
	if err != nil {
		log.Error(err)
		handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logFilePath, err := resolveLogFilePath(reqBody.LogsPath, reqBody.FileName)
	if err != nil {
		log.Errorf("invalid search log file request, logs_path=[%s], file_name=[%s]: %v", reqBody.LogsPath, reqBody.FileName, err)
		handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if reqBody.StartLineNumber < 0 {
		reqBody.StartLineNumber = 0
	}
	if strings.HasSuffix(reqBody.FileName, ".gz") {
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
			err = agent_util.UnpackGzipFile(logFilePath, tmpFilePath)
			if err != nil {
				log.Errorf("failed to unpack gzip log file from [%s] to [%s]: %v", logFilePath, tmpFilePath, err)
				handler.WriteJSON(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		logFilePath = tmpFilePath
	}
	var msgs []util.MapStr
	var isEOF bool
	if reqBody.TailLines > 0 {
		msgs, err = readSearchLogTailLines(logFilePath, reqBody.TailLines)
		isEOF = true
	} else {
		msgs, isEOF, err = readSearchLogLines(logFilePath, reqBody.StartLineNumber, reqBody.Offset, reqBody.Lines)
	}
	if err != nil {
		log.Errorf("failed to read search log file [%s] at start_line_number=[%d], offset=[%d], tail_lines=[%d]: %v", logFilePath, reqBody.StartLineNumber, reqBody.Offset, reqBody.TailLines, err)
		handler.WriteError(w, fmt.Sprintf("read logs error: %v", err), http.StatusInternalServerError)
		return
	}
	handler.WriteJSON(w, util.MapStr{
		"result":  msgs,
		"success": true,
		"EOF":     isEOF,
	}, http.StatusOK)
}

func readSearchLogLines(filePath string, startLineNumber int64, offset int64, lines int) ([]util.MapStr, bool, error) {
	if lines <= 0 {
		lines = 50
	}
	if lines > maxReadLines {
		lines = maxReadLines
	}
	if offset < 0 {
		offset = 0
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, false, err
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
		file.Close()
	}()

	currentOffset := int64(0)
	currentLineNumber := int64(1)
	lineNumbersKnown := true

	if offset == 0 && startLineNumber < 1 {
		startLineNumber = 1
	}

	if offset > 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return nil, false, err
		}
		currentOffset = offset
		if startLineNumber > 0 {
			currentLineNumber = startLineNumber
		} else {
			currentLineNumber = 0
			lineNumbersKnown = false
		}
	} else {
		currentLineNumber = startLineNumber
	}

	reader := bufio.NewReaderSize(file, logReadBufferSize)
	result := make([]util.MapStr, 0, lines)
	isEOF := false

	for len(result) < lines {
		rawLine, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return result, false, err
		}
		if len(rawLine) == 0 {
			if err == io.EOF {
				isEOF = true
			}
			break
		}

		currentOffset += int64(len(rawLine))
		lineContent := trimLogLineEnding(rawLine)

		if offset > 0 {
			line := util.MapStr{
				"content": string(lineContent),
				"bytes":   len(rawLine),
				"offset":  currentOffset,
			}
			if lineNumbersKnown {
				line["line_number"] = currentLineNumber
				currentLineNumber++
			}
			result = append(result, line)
		} else {
			if currentLineNumber >= startLineNumber {
				line := util.MapStr{
					"content": string(lineContent),
					"bytes":   len(rawLine),
					"offset":  currentOffset,
				}
				if lineNumbersKnown {
					line["line_number"] = currentLineNumber
				}
				result = append(result, line)
			}
			currentLineNumber++
		}

		if err == io.EOF {
			isEOF = true
			break
		}
	}

	return result, isEOF, nil
}

func readSearchLogTailLines(filePath string, lines int) ([]util.MapStr, error) {
	if lines <= 0 {
		lines = 50
	}
	if lines > maxReadLines {
		lines = maxReadLines
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() == 0 {
		return []util.MapStr{}, nil
	}

	windowStart, windowBytes, err := locateTailWindow(file, stat.Size(), lines)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReaderSize(bytes.NewReader(windowBytes), logReadBufferSize)
	currentOffset := windowStart
	result := make([]util.MapStr, 0, lines)

	for len(result) < lines {
		rawLine, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		if len(rawLine) == 0 {
			break
		}

		currentOffset += int64(len(rawLine))
		result = append(result, util.MapStr{
			"content": string(trimLogLineEnding(rawLine)),
			"bytes":   len(rawLine),
			"offset":  currentOffset,
		})

		if err == io.EOF {
			break
		}
	}

	return result, nil
}

func locateTailWindow(file *os.File, fileSize int64, lines int) (int64, []byte, error) {
	position := fileSize
	window := make([]byte, 0)
	newlineCount := 0

	for position > 0 && newlineCount <= lines {
		chunkSize := int64(logReadBufferSize)
		if position < chunkSize {
			chunkSize = position
		}
		position -= chunkSize

		chunk := make([]byte, chunkSize)
		if _, err := file.ReadAt(chunk, position); err != nil && err != io.EOF {
			return 0, nil, err
		}
		window = append(chunk, window...)
		newlineCount += bytes.Count(chunk, []byte{'\n'})
	}

	startIndex := findTailStartIndex(window, lines)
	return position + int64(startIndex), window[startIndex:], nil
}

func findTailStartIndex(window []byte, lines int) int {
	if len(window) == 0 || lines <= 0 {
		return 0
	}

	idx := len(window) - 1
	if window[idx] == '\n' {
		idx--
	}

	newlineCount := 0
	for ; idx >= 0; idx-- {
		if window[idx] != '\n' {
			continue
		}
		newlineCount++
		if newlineCount == lines {
			return idx + 1
		}
	}
	return 0
}

func shouldCountLogRows(fileSize int64) bool {
	return fileSize >= 0 && fileSize <= maxCountRowsBytes
}

func trimLogLineEnding(rawLine []byte) []byte {
	return bytes.TrimRight(rawLine, "\r\n")
}

func resolveLogsDirectory(logsPath string) (string, error) {
	expanded, err := agent_util.ExpandHomeDir(logsPath)
	if err != nil {
		return "", err
	}
	resolved := filepath.Clean(expanded)
	if resolved == "" {
		return "", fmt.Errorf("invalid logs_path")
	}
	if realPath, err := filepath.EvalSymlinks(resolved); err == nil {
		resolved = realPath
	}
	stat, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("logs_path is not a directory: %s", logsPath)
	}
	return resolved, nil
}

func resolveLogFilePath(logsPath, fileName string) (string, error) {
	logFilePath, err := safeJoinLogsFile(logsPath, fileName)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(logFilePath)
	if err != nil {
		return "", err
	}
	relativePath, err := filepath.Rel(logsPath, resolved)
	if err != nil {
		return "", err
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid file_name: %s", fileName)
	}
	return resolved, nil
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
		expanded, err := agent_util.ExpandHomeDir(item)
		if err == nil {
			item = expanded
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

	expanded, err := agent_util.ExpandHomeDir(logsPath)
	if err != nil {
		return "", err
	}
	basePath := filepath.Clean(expanded)
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
