/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package reader

import "io"

type Reader interface {
	io.Closer
	Next() (Message, error)
}
