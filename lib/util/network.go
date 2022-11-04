/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package util

import (
	"net"
	"strings"
)

func GetClientIp(filter string) (string, error) {
	ret, err := net.InterfaceByName(filter)
	if err != nil {
		return "", err
	}
	address, err := ret.Addrs()
	if err != nil {
		return "", err
	}
	var ipStr string
	for _, addr := range address {
		var (
			ip net.IP
		)
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		ipStr = ip.String()
		if strings.Contains(ipStr, "::") {
			ipStr = strings.Split(ipStr, "::")[1]
		}
	}
	return ipStr, nil
}