/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package util

import (
	"net"
	"sort"
	"strings"
)

func GetClientIp(filter string) string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	nameIPS := map[string]string{}
	names := []string{}
	for _, i := range interfaces {
		name := i.Name
		addrs, err := i.Addrs()
		if err != nil {
			panic(err)
		}
		// handle err
		for _, addr := range addrs {
			var (
				ip net.IP
			)
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if filter == "" || strings.Contains(name, filter) {
				names = append(names, name)
				nameIPS[name] = ip.String()
			}
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return nameIPS[names[0]]
}