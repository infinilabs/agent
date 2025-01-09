/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package harvester

import (
	"fmt"
	"io"
	"os"

	"infini.sh/agent/lib/reader"
	"infini.sh/agent/lib/reader/linenumber"
	"infini.sh/agent/lib/reader/multiline"
	"infini.sh/agent/lib/reader/readfile"
	"infini.sh/agent/lib/reader/readfile/encoding"
	"infini.sh/agent/lib/reader/readjson"
)

type Harvester struct {
	reader reader.Reader
	file   *os.File
	config Config
	offset int64

	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding
}

func NewHarvester(path string, offset int64) (*Harvester, error) {
	f, err := readOpen(path)
	if f == nil || err != nil {
		return nil, err
	}
	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	h := &Harvester{
		file:   f,
		config: defaultConfig(),
		offset: offset,
	}
	encodingFactory, ok := encoding.FindEncoding(h.config.Encoding)
	if !ok || encodingFactory == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", h.config.Encoding)
	}
	h.encodingFactory = encodingFactory
	h.encoding, err = h.encodingFactory(f)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func readOpen(path string) (*os.File, error) {
	flag := os.O_RDONLY
	perm := os.FileMode(0)
	return os.OpenFile(path, flag, perm)
}

func (h *Harvester) NewJsonFileReader(pattern string, showLineNumber bool) (reader.Reader, error) {
	var r reader.Reader
	var err error
	if h.file == nil {
		return nil, fmt.Errorf("file is nil")
	}

	encReaderMaxBytes := h.config.MaxBytes * 4
	r, err = readfile.NewEncodeReader(h.file, readfile.Config{
		Codec:      h.encoding,
		BufferSize: h.config.BufferSize,
		MaxBytes:   encReaderMaxBytes,
		Terminator: h.config.LineTerminator,
	})
	if err != nil {
		return nil, err
	}

	if h.config.JSON != nil {
		r = readjson.NewJSONReader(r, h.config.JSON)
	}

	//r = readfile.NewStripNewline(r, h.config.LineTerminator)

	h.config.Multiline = multiline.DefaultConfig(pattern)
	r, err = multiline.New(r, "", h.config.MaxBytes, h.config.Multiline)
	if err != nil {
		return nil, err
	}
	r = readfile.NewLimitReader(r, h.config.MaxBytes)
	if showLineNumber {
		h.config.LineNumber = linenumber.NewConfig(h.offset, h.file, io.SeekStart)
		h.reader = linenumber.NewLineNumberReader(r, h.config.LineNumber)
	} else {
		h.reader = r
	}
	return h.reader, nil
}

func (h *Harvester) NewLogFileReader(pattern string, showLineNumber bool) (reader.Reader, error) {
	var r reader.Reader
	var err error

	if h.file == nil {
		return nil, fmt.Errorf("file is nil")
	}
	encReaderMaxBytes := h.config.MaxBytes * 4
	r, err = readfile.NewEncodeReader(h.file, readfile.Config{
		Codec:      h.encoding,
		BufferSize: h.config.BufferSize,
		MaxBytes:   encReaderMaxBytes,
		Terminator: h.config.LineTerminator,
	})
	if err != nil {
		return nil, err
	}

	//r = readfile.NewStripNewline(r, h.config.LineTerminator)

	h.config.Multiline = multiline.DefaultConfig(pattern)
	r, err = multiline.New(r, "", h.config.MaxBytes, h.config.Multiline)
	if err != nil {
		return nil, err
	}
	r = readfile.NewLimitReader(r, h.config.MaxBytes)
	if showLineNumber {
		h.config.LineNumber = linenumber.NewConfig(h.offset, h.file, io.SeekStart)
		h.reader = linenumber.NewLineNumberReader(r, h.config.LineNumber)
	} else {
		h.reader = r
	}
	return h.reader, nil
}

func (h *Harvester) Close() error {
	err := h.reader.Close()
	if err != nil {
		return err
	}
	return nil
}
