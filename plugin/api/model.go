/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

type GetSearchLogFilesReq struct {
	LogsPath interface{} `json:"logs_path"`
}

type ReadSearchLogFileReq struct {
	LogsPath        string `json:"logs_path"`
	FileName        string `json:"file_name"`
	Offset          int64  `json:"offset"`
	Lines           int    `json:"lines"`
	StartLineNumber int64  `json:"start_line_number"`
}
