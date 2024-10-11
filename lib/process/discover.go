/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package process

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
	"os"
	"regexp"
	"strconv"
)

type FilterFunc func(cmdline string) bool

var searchEngineRegx = regexp.MustCompile( "(?i)org.(easy|elastic|open)search.bootstrap.(Easy|Elastic|Open)Search")
func ElasticFilter(cmdline string) bool {
	return searchEngineRegx.MatchString(cmdline)
}

func DiscoverESProcessors(filter FilterFunc)(map[int]model.ProcessInfo, error){
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
			processName, _ := p.Name()
			//k8s easysearch process pid is 1
			if p.Pid == 1 {
				envPort := os.Getenv("http.port")
				port, _ := strconv.Atoi(envPort)
				processInfo := model.ProcessInfo{
					PID: int(p.Pid),
					Name: processName,
					Cmdline: cmdline,
					ListenAddresses: []model.ListenAddr{
						{
							IP: util.GetLocalIPs()[0],
							Port: port,
						},
					},
					Status: "N/A",
				}
				status, _ := p.Status()
				if len(status) > 0 {
					processInfo.Status = status[0]
				}
				processInfo.CreateTime, _ = p.CreateTime()

				resultProcesses[processInfo.PID] = processInfo
				break
			}
			connections, err := p.Connections()
			if err != nil {
				return nil, fmt.Errorf("get process connections error: %w", err)
			}

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