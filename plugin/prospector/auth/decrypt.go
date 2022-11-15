/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"encoding/base64"
	"encoding/hex"
	log "github.com/cihub/seelog"
	"github.com/forgoer/openssl"
)

// TODO 如何兼容多种加密方式?
func opensslAesDecrypt(encValue, encKey, encIV, encType string) string {
	if encValue == "" || encKey == "" || encIV == "" || encType == "" {
		return ""
	}
	key := hexDecode(encKey)
	iv := hexDecode(encIV)
	if key == nil || iv == nil {
		return ""
	}
	enc, err := base64.StdEncoding.DecodeString(encValue)
	if err != nil {
		log.Error(err)
		return ""
	}
	dst, err := openssl.AesCBCDecrypt(enc, key, iv, openssl.PKCS7_PADDING)
	if err != nil {
		log.Error(err)
		return ""
	}
	return string(dst)
}

func hexDecode(src string) []byte {
	maxDeLen := hex.DecodedLen(len(src))
	dst := make([]byte, maxDeLen)
	_, err := hex.Decode(dst, []byte(src))
	if err != nil {
		log.Error(err)
		return nil
	}
	return dst
}
