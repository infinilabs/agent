/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package auth

import (
	"bytes"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/util"
	"os/exec"
)

func decrypt(encValue, encKey, encIV, encType string) string {
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
