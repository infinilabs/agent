/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package linenumber

import "os"

type Config struct {
	Offset int64
	file   *os.File
	whence int //io.SeekStart / io.SeekEnd
}

func NewConfig(offset int64, f *os.File, whence int) *Config {
	return &Config{
		Offset: offset,
		file:   f,
		whence: whence,
	}
}
