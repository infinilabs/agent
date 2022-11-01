/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
	"os/exec"
	"src/github.com/openssl"
)

func decryptWithShell(encValue, encKey, encIV, encType string) string {
	absPath := util.TryGetFileAbsPath("decrypt.sh",false)
	if absPath == "" {
		log.Error("decrypt auth info failed, could not find decrypt.sh")
		return ""
	}
	command := absPath + " " + encValue + " " + encKey + " " + encIV + " " + encType
	cmd := exec.Command("/bin/bash", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Errorf("decrypt auth info failed: %s",err)
	}
	return out.String()
}

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
