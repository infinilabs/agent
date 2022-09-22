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
		var content string
		//获取当前文件换行符的长度。因为在StripNewline这一步的时候，是去掉了每一行的换行符，所以差值为换行符的长度。这里就不再处理一遍换行符了
		terminatorLength := message.Bytes - len(message.Content)
		for scanner.Scan() {
			//scanner返回的内容，是已经去掉换行符的
			content = scanner.Text()
			contentLen = int64(len([]byte(content)))
			offset += contentLen + int64(terminatorLength)
			line++
			//当前读到的offset，已经超过了用户指定的offset + 当前内容的长度， 那说明offset已经超过范围
			if offset > (r.currentOffset + contentLen + int64(terminatorLength)) {
				offset -= contentLen + int64(terminatorLength)
				break
			}
			//当前读到的offset 大于了用户指定的offset，说明已经到了目标为止，记录行号
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
