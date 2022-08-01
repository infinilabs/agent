/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package model

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestHost(t *testing.T) {
	ips := []string{"192.168.1.1", "192.168.2.1"}
	clusters := []Cluster{
		{
			Name:     "cluster01",
			ID:       "cluster01",
			UserName: "ck",
			Password: "ck1234",
		},
		{
			Name:     "cluster02",
			ID:       "cluster02",
			UserName: "ck02",
			Password: "ck1234",
		},
	}
	host := Host{
		IPs:      ips,
		Clusters: clusters,
		TLS:      false,
	}
	ret, err := json.Marshal(host)
	if err != nil {
		return
	}
	fmt.Println(string(ret))
}
