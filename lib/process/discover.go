/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package process

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"infini.sh/agent/model"
	"strings"
)

type FilterFunc func(cmdline string) bool

func ElasticFilter(cmdline string) bool {
	return strings.Contains(cmdline, "elasticsearch")
}

func Discover(filter FilterFunc)(map[int]model.ProcessInfo, error){
	if filter == nil {
		return nil, fmt.Errorf("process filter func must not be empty")
	}
	processes, _ := process.Processes()
	var resultProcesses = map[int]model.ProcessInfo{}
	for _, p := range processes {
		cmdline, err := p.Cmdline()
		if err != nil {
			continue
		}
		if filter(cmdline) {
			connections, err := p.Connections()
			if err != nil {
				return nil, fmt.Errorf("get process connections error: %w", err)
			}
			processName, _ := p.Name()
			var addresses []model.ListenAddr
			for _, connection := range connections {
				if connection.Status == "LISTEN" {
					addresses = append(addresses, model.ListenAddr{
						IP: connection.Laddr.IP,
						Port: int(connection.Laddr.Port),
					})
				}
			}
			if len(addresses) > 0 {
				processInfo := model.ProcessInfo{
					PID: int(p.Pid),
					Name: processName,
					Cmdline: cmdline,
					ListenAddresses: addresses,
					Status: "N/A",
				}
				status, _ := p.Status()
				if len(status) > 0 {
					processInfo.Status = status[0]
				}
				processInfo.CreateTime, _ = p.CreateTime()

				resultProcesses[processInfo.PID] = processInfo
			}
		}
	}
	return resultProcesses, nil
}