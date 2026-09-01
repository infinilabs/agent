/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package harvester

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"infini.sh/agent/lib/reader"
	"infini.sh/agent/lib/reader/linenumber"
	"infini.sh/agent/lib/reader/multiline"
	"infini.sh/agent/lib/reader/readfile"
	"infini.sh/agent/lib/reader/readfile/encoding"
	"infini.sh/agent/lib/reader/readjson"
	"infini.sh/framework/core/errors"
)

type Harvester struct {
	reader reader.Reader
	file   *os.File
	config Config
	offset int64
	// gzipWrapped: the file is a rotated .gz archive — immutable, always
	// read from the start of the DECOMPRESSED stream (offsets into it are
	// not comparable across runs, so state tracking is mtime-based only).
	gzipWrapped bool
	// src is the reading source (raw file or gzip-decompressed stream).
	src io.ReadCloser

	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding
}

// IsGzipFile reports whether the path looks like a gzip archive — the
// detector routes rotated *.gz logs here so compressed history is ingested.
func IsGzipFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".gz")
}

func NewHarvester(path string, offset int64) (*Harvester, error) {
	f, err := readOpen(path)
	if f == nil || err != nil {
		return nil, errors.Errorf("failed to open file(%s),%v", path, err)
	}
	h := &Harvester{
		file:   f,
		config: defaultConfig(),
		offset: offset,
	}
	// Reading source: the raw file, or its decompressed stream for rotated
	// .gz archives (immutable — always from the start, offsets in the
	// decompressed stream are not comparable across runs).
	src := io.ReadCloser(f)
	if IsGzipFile(path) {
		zr, zerr := gzip.NewReader(f)
		if zerr != nil {
			_ = f.Close()
			return nil, errors.Errorf("failed to open gzip file(%s),%v", path, zerr)
		}
		src = zr
		h.gzipWrapped = true
		h.offset = 0
	} else if _, err := f.Seek(offset, io.SeekStart); err != nil {
		_ = f.Close()
		return nil, err
	}
	encodingFactory, ok := encoding.FindEncoding(h.config.Encoding)
	if !ok || encodingFactory == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", h.config.Encoding)
	}
	h.encodingFactory = encodingFactory
	h.encoding, err = h.encodingFactory(src)
	if err != nil {
		return nil, err
	}
	h.src = src
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
	if h == nil || h.file == nil {
		return nil, fmt.Errorf("file is nil")
	}

	encReaderMaxBytes := h.config.MaxBytes * 4
	r, err = readfile.NewEncodeReader(h.src, readfile.Config{
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

	if h == nil || h.file == nil {
		return nil, fmt.Errorf("file is nil")
	}
	encReaderMaxBytes := h.config.MaxBytes * 4
	r, err = readfile.NewEncodeReader(h.src, readfile.Config{
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
