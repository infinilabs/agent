/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

type GetElasticLogFilesReq struct {
	LogsPath string `json:"logs_path"`
}

type ReadElasticLogFileReq struct {
	LogsPath string `json:"logs_path"`
	FileName string `json:"file_name"`
	Offset   int64    `json:"offset"`
	Lines    int    `json:"lines"`
}