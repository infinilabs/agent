/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package linenumber

import (
	"bufio"
	"errors"
	"infini.sh/agent/lib/reader"
	"io"
	"os"
)

type LineNumberReader struct {
	reader        reader.Reader
	cfg           *Config
	currentOffset int64
}

func NewLineNumberReader(r reader.Reader, config *Config) *LineNumberReader {
	lineReader := &LineNumberReader{
		reader: r,
		cfg:    config,
	}
	lineReader.currentOffset = config.Offset
	return lineReader
}

func (r *LineNumberReader) Next() (reader.Message, error) {
	message, err := r.reader.Next()
	if err != nil {
		return message, err
	}
	if r.cfg == nil {
		return message, errors.New("config can not be nil")
	}
	if r.cfg.file != nil {
		fileName := r.cfg.file.Name()
		flag := os.O_RDONLY
		perm := os.FileMode(0)
		file, err := os.OpenFile(fileName, flag, perm)
		if err != nil {
			return message, err
		}
		file.Seek(0, io.SeekStart)
		scanner := bufio.NewScanner(file)
		var offset int64 = 0
		line := 0
		var contentLen int64 = 0
		for scanner.Scan() {
			contentLen = int64(len([]byte(scanner.Text())))
			offset += contentLen + 1
			line++
			if offset >= (r.currentOffset + int64(message.Bytes)) {
				message.LineNumbers = append(message.LineNumbers, line)
				break
			}
			if offset > r.currentOffset {
				message.LineNumbers = append(message.LineNumbers, line)
			}
		}
		r.currentOffset = offset
		message.Offset = offset
		file.Close()
	}
	return message, nil
}

func (r *LineNumberReader) Close() error {
	return r.reader.Close()
}
