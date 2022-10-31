/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"log"
	"testing"
)

func TestDecrypt(t *testing.T) {
	encKey := "3765323063323065613735363432333161373664643833616331636637303133"
	encIV := "66382f4e654c734a2a732a7679675640"
	encType := "-aes-256-cbc"
	encValue := "X4B1gCYhiYuUt49hBeVPbAauxponJW5t64qP50aVuQY="
	ret := decrypt(encValue, encKey, encIV, encType)
	log.Println(ret)
}