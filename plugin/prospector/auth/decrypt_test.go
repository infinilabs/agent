/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"log"
	"testing"
)

func TestDecrypt(t *testing.T) {
	encKey := "xxx"
	encIV := "xxx"
	encType := "xxx"
	encValue := "xxx"
	ret := OpensslAesDecrypt(encValue, encKey, encIV, encType)
	log.Println(ret)
}