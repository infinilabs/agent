/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package ecloud

import (
	"fmt"
	"infini.sh/agent/plugin/prospector/auth"
	"infini.sh/framework/core/util"
	"os"
)

type Decryptor struct {
	cfg DecryptConfig
	encPassword string
}

func NewDecryptor(userName, authPath string) *Decryptor {
	dec := &Decryptor{
		cfg: DecryptConfig{
			ESUserName: userName,
			Path: authPath,
		},
	}
	return dec
}

type DecryptConfig struct {
	Enable  bool   `json:"enable" config:"enable"`
	Path    string `json:"path" config:"path"`
	EncKey  string `json:"enc_key" config:"enc_key"`
	EncIV   string `json:"enc_iv" config:"enc_iv"`
	EncType string `json:"enc_type" config:"enc_type"`
	ESUserName string `json:"es_username" config:"es_username"`
}

func (dec *Decryptor) Decrypt() (string, error) {
	if !dec.loadEnvConfig() {
		return "", fmt.Errorf("please config environment variables")
	}
	err := dec.loadAuthFile()
	if err != nil {
		return "", err
	}
	pwd := auth.OpensslAesDecrypt(dec.encPassword, dec.cfg.EncKey, dec.cfg.EncIV, dec.cfg.EncType)
	return pwd, nil
}

func (dec *Decryptor) loadAuthFile() error {
	content, err := os.ReadFile(dec.cfg.Path)
	if err != nil {
		return err
	}
	var authInfo map[string]string
	err = util.FromJSONBytes(content, &authInfo)
	if err != nil {
		return err
	}
	encPWD, ok := authInfo[dec.cfg.ESUserName]
	if !ok {
		return fmt.Errorf("can not find auth info from: %s", dec.cfg.Path)
	}
	dec.encPassword = encPWD
	return nil
}

func (dec *Decryptor) loadEnvConfig() bool {
	encKey := os.Getenv("AUTH_ENC_KEY")
	encIV := os.Getenv("AUTH_ENC_IV")
	encType := os.Getenv("AUTH_ENC_TYPE")
	if encKey == "" || encIV == "" || encType == "" {
		return false
	}
	dec.cfg.EncIV = encIV
	dec.cfg.EncKey = encKey
	dec.cfg.EncType = encType
	return true
}


