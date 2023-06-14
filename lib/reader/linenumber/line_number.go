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
	scanner *bufio.Scanner
	currentLine int64
	innerFile *os.File
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
	if r.cfg.file == nil {
		return message, nil
	}
	if r.scanner == nil {
		fileName := r.cfg.file.Name()
		flag := os.O_RDONLY
		perm := os.FileMode(0)
		r.innerFile, err = os.OpenFile(fileName, flag, perm)
		if err != nil {
			return message, err
		}
		r.innerFile.Seek(0, io.SeekStart)
		r.scanner = bufio.NewScanner(r.innerFile)
	}

	var offset int64 = 0
	if r.currentLine > 0 {
		offset = r.currentOffset
	}
	line := r.currentLine
	var contentLen int64 = 0
	var content string
	//获取当前文件换行符的长度。因为在StripNewline这一步的时候，是去掉了每一行的换行符，所以差值为换行符的长度。这里就不再处理一遍换行符了
	for r.scanner.Scan() {
		//scanner返回的内容，是已经去掉换行符的
		content = r.scanner.Text()
		contentLen = int64(len([]byte(content)))
		offset += contentLen + 1
		line++
		//当前读到的offset，已经超过了用户指定的offset + 当前内容的长度， 那说明offset已经超过范围
		//add 1 for the newline character
		if offset > (r.currentOffset + contentLen + 1) {
			offset -= contentLen + 1
			//skip prefix lines
			r.currentOffset = offset
			break
		}
		//当前读到的offset 大于了用户指定的offset，说明已经到了目标为止，记录行号
		if offset > r.currentOffset {
			message.LineNumbers = append(message.LineNumbers, line)
			break
		}
	}
	r.currentOffset = offset
	r.currentLine = line
	message.Offset = offset
	return message, nil
}

func (r *LineNumberReader) Close() error {
	r.innerFile.Close()
	return r.reader.Close()
}
