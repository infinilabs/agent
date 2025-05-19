/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package process

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/shirou/gopsutil/v3/process"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type FilterFunc func(cmdline string) bool

var searchEngineRegx = regexp.MustCompile("(?i)org.(easy|elastic|open)search.bootstrap.(Easy|Elastic|Open)Search")

func ElasticFilter(cmdline string) bool {
	return searchEngineRegx.MatchString(cmdline)
}

func DiscoverESProcessors(filter FilterFunc) (map[int]model.ProcessInfo, error) {
	if filter == nil {
		return nil, fmt.Errorf("process filter func must not be empty")
	}
	processes, _ := process.Processes()
	var resultProcesses = map[int]model.ProcessInfo{}
	for _, p := range processes {
		cmdline, err := p.Cmdline()
		if p == nil || err != nil {
			if global.Env().IsDebug {
				log.Errorf("get process cmdline error: %v", err)
			}
			continue
		}
		if filter(cmdline) {
			processName, _ := p.Name()
			// Handle K8S container when easysearch specific process (pid 1)
			allowGenernated := global.Env().SystemConfig.Configs.AllowGeneratedMetricsTasks
			if p.Pid == 1 && allowGenernated {
				processInfo, err := handleK8sProcess(p, processName, cmdline)
				if err != nil {
					log.Errorf("Error handling k8s process: %v", err)
				} else {
					resultProcesses[processInfo.PID] = processInfo
					break
				}
			}

			connections, err := p.Connections()
			if err != nil {
				return nil, fmt.Errorf("get process connections error: %w", err)
			}

			var addresses []model.ListenAddr
			for _, connection := range connections {
				if connection.Status == "LISTEN" {
					addresses = append(addresses, model.ListenAddr{
						IP:   connection.Laddr.IP,
						Port: int(connection.Laddr.Port),
					})
				}
			}

			// If no listen addresses found, try to read from /proc/net/tcp and /proc/net/tcp6 files on linux
			if len(addresses) == 0 && runtime.GOOS == "linux" {
				log.Debugf("Try to read /proc/net/tcp and /proc/net/tcp6 files for process %d", p.Pid)
				procAddresses := readProcTcpServicePorts(p)
				addresses = append(addresses, procAddresses...)
			}

			if len(addresses) > 0 {
				processInfo := model.ProcessInfo{
					PID:             int(p.Pid),
					Name:            processName,
					Cmdline:         cmdline,
					ListenAddresses: addresses,
					Status:          "N/A",
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

// handleK8sProcess handles the specific case of processes with pid 1 in kubernetes environment
func handleK8sProcess(p *process.Process, processName string, cmdline string) (model.ProcessInfo, error) {
	envPort := os.Getenv("http.port")
	port, err := strconv.Atoi(envPort)
	if err != nil {
		return model.ProcessInfo{}, fmt.Errorf("get env http.port error: %w", err)
	}

	processInfo := model.ProcessInfo{
		PID:     int(p.Pid),
		Name:    processName,
		Cmdline: cmdline,
		ListenAddresses: []model.ListenAddr{
			{
				IP:   util.GetLocalIPs()[0],
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
	return processInfo, nil
}

// readProcNetFile reads the /proc/net/tcp and /proc/net/tcp6 files and returns the listening addresses
func readProcTcpServicePorts(p *process.Process) []model.ListenAddr {
	var listenAddrs []model.ListenAddr
	pid := p.Pid
	tcpFile := filepath.Join("/proc", strconv.Itoa(int(pid)), "net/tcp")
	if _, err := os.Stat(tcpFile); os.IsNotExist(err) {
		log.Errorf("skip: tcp file does not exists: %s", tcpFile)
	} else if err == nil {
		tcpPorts, err := readProcNetFile(tcpFile, p)
		if err != nil {
			log.Errorf("Error reading tcp file: %s, error: %v", tcpFile, err)
		} else {
			listenAddrs = append(listenAddrs, tcpPorts...)
		}
	}

	tcp6File := filepath.Join("/proc", strconv.Itoa(int(pid)), "net/tcp6")
	if _, err := os.Stat(tcp6File); os.IsNotExist(err) {
		log.Errorf("skip: tcp6 file does not exists: %s", tcp6File)
	} else if err == nil {
		tcp6Ports, err := readProcNetFile(tcp6File, p)
		if err != nil {
			log.Errorf("Error reading tcp6 file: %s, error: %v", tcp6File, err)
		} else {
			listenAddrs = append(listenAddrs, tcp6Ports...)
		}
	}
	return listenAddrs
}

func readProcNetFile(file string, p *process.Process) ([]model.ListenAddr, error) {
	var listenAddrs []model.ListenAddr
	b, err := os.ReadFile(file)
	if err != nil {
		return listenAddrs, fmt.Errorf("open file error: %w", err)
	}

	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		// The format contains whitespace padding (%4d, %5u), so we use
		// fmt.Sscanf instead of splitting on whitespace.
		var (
			sl                            int
			readLocalAddr, readRemoteAddr string
			state                         int
			queue, timer                  string
			retransmit                    int
			remoteUID                     uint
		)
		// Note that we must use %d where the kernel format uses %4d or %5u:
		// - %4d fails to parse for large number of entries (len(sl) > 4)
		// - %u is not understood by the fmt package (%U is something else)
		// - %5d cuts off longer uids (e.g. 149098 on gLinux)
		n, err := fmt.Sscanf(line, "%d: %s %s %02X %s %s %08X %d",
			&sl, &readLocalAddr, &readRemoteAddr, &state, &queue, &timer, &retransmit, &remoteUID)
		if n != 8 || err != nil {
			continue // invalid line (e.g. header line)
		}
		if state != 0x0A && state != 0x06 { // Only keep listening status, 0A: TCP_LISTEN, 06: TCP6_LISTEN
			continue
		}
		// Parse local address and port	with same uid as the process
		uids, _ := p.Uids()
		if len(uids) > 0 {
			uid := int(uids[0])
			same := uid == int(remoteUID)
			if !same {
				continue
			}
			parseLine(&listenAddrs, readLocalAddr)
		}
	}
	return listenAddrs, nil
}

func parseLine(listenAddr *[]model.ListenAddr, addr string) {
	// Parse address and port
	addrParts := strings.Split(addr, ":")
	if len(addrParts) != 2 {
		return
	}
	hexIP := addrParts[0]
	hexPort := addrParts[1]
	// Handle IPv6 address
	if len(hexIP) > 24 {
		hexIP = hexIP[24:]
	}
	localIP, _ := hexToDecimal(hexIP)
	ip := ipv4Ntoa(uint32(localIP))
	port, _ := hexToDecimal(hexPort)
	// IPv6 address is not supported
	if ip == util.LocalIpv6Address {
		ip = util.LocalAddress
	}

	for _, existingAddr := range *listenAddr {
		if existingAddr.IP == ip && existingAddr.Port == port {
			return
		}
	}
	*listenAddr = append(*listenAddr, model.ListenAddr{
		IP:   ip,
		Port: port,
	})
}

// ipv4Ntoa converts uint32 to IPv4 string representation
func ipv4Ntoa(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		ip>>0&0xFF,
		ip>>8&0xFF,
		ip>>16&0xFF,
		ip>>24&0xFF,
	)
}

// hexToDecimal converts hex string to decimal int
func hexToDecimal(hex string) (int, error) {
	value, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return 0, err
	}
	return int(value), nil
}
