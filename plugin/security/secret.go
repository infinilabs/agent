/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"errors"

	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
)

const kvBucket = "Agent"
const kvKey = "agent"

var secretKey string

// GetSecret returns the HMAC signing key for JWT tokens.
// On first call it generates a random UUID and persists it in the KV store;
// subsequent calls return the cached value.
func GetSecret() (string, error) {
	if secretKey != "" {
		return secretKey, nil
	}

	exists, err := kv.ExistsKey(kvBucket, []byte(kvKey))
	if err != nil {
		return "", err
	}
	if !exists {
		key := util.GetUUID()
		err = kv.AddValue(kvBucket, []byte(kvKey), []byte(key))
		if err != nil {
			return "", err
		}
		secretKey = key
	} else {
		v, err := kv.GetValue(kvBucket, []byte(kvKey))
		if err != nil {
			return "", err
		}
		if len(v) > 0 {
			secretKey = string(v)
		}
	}

	if secretKey == "" {
		return "", errors.New("invalid secret: unable to create or retrieve secret key")
	}

	return secretKey, nil
}
