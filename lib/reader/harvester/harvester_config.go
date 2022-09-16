/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package harvester

import (
	"github.com/dustin/go-humanize"
	"infini.sh/agent/lib/reader/linenumber"
	"infini.sh/agent/lib/reader/multiline"
	"infini.sh/agent/lib/reader/readfile"
	"infini.sh/agent/lib/reader/readjson"
)

type Config struct {
	Encoding       string                  `config:"encoding"`
	BufferSize     int                     `config:"harvester_buffer_size"`
	MaxBytes       int                     `config:"max_bytes" validate:"min=0,nonzero"`
	LineTerminator readfile.LineTerminator `config:"line_terminator"`
	JSON           *readjson.Config        `config:"json"`
	Multiline      *multiline.Config       `config:"multiline"`
	LineNumber     *linenumber.Config
}

func defaultConfig() Config {
	return Config{
		BufferSize:     16 * humanize.KiByte,
		MaxBytes:       10 * humanize.MiByte,
		LineTerminator: readfile.AutoLineTerminator,
	}
}
