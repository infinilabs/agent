/* Copyright © INFINI LTD. All rights reserved.
* Web: https://infinilabs.com
* Email: hello#infini.ltd */

package linenumber

import (
	"bufio"
	"infini.sh/agent/lib/reader"
	"io"
	"os"
)

type LinePlainTextReader struct {
	currentOffset int64
	scanner       *bufio.Scanner
	currentLine   int64
	innerFile     *os.File
	startLine     int64
}

func NewLinePlainTextReader(filePath string, startLineNumber int64, whence int) (*LinePlainTextReader, error) {
	lineReader := &LinePlainTextReader{
		startLine: startLineNumber,
	}
	flag := os.O_RDONLY
	perm := os.FileMode(0)
	var err error
	lineReader.innerFile, err = os.OpenFile(filePath, flag, perm)
	if err != nil {
		return nil, err
	}
	lineReader.innerFile.Seek(0, whence)
	lineReader.scanner = bufio.NewScanner(lineReader.innerFile)
	return lineReader, nil
}

func (r *LinePlainTextReader) Next() (message reader.Message, err error) {
	var offset int64 = 0
	if r.currentLine > 0 {
		offset = r.currentOffset
	}
	line := r.currentLine
	var contentLen int64 = 0
	var content []byte
	for r.scanner.Scan() {
		content = r.scanner.Bytes()
		contentLen = int64(len(content))
		//add 1 for the newline character
		offset += contentLen + 1
		line++
		if line < r.startLine {
			//skip prefix lines
			r.currentOffset = offset
			continue
		}
		message.LineNumbers = append(message.LineNumbers, line)
		break
	}
	if len(content) == 0 || line < r.startLine {
		return message, io.EOF
	}
	r.currentOffset = offset
	r.currentLine = line
	message.Offset = offset
	message.Content = content
	return message, nil
}

func (r *LinePlainTextReader) Close() error {
	return r.innerFile.Close()
}
