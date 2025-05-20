package elastic

import (
	"bufio"
	"net"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/shirou/gopsutil/v3/process"
	"infini.sh/framework/core/util"
)

type PathPort struct {
	Path       string
	ESHomePath string
	Ports      []int
	PID        int32
}

func GetNodeInfoFromProcess() ([]*PathPort, error) {
	var pathPorts []*PathPort
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}
	for _, process := range processes {
		cmdLine, err := process.Cmdline()
		if err != nil {
			continue
		}
		if !strings.Contains(cmdLine, "Des.path.home") || !strings.Contains(cmdLine, "elasticsearch") {
			continue
		}
		pathPort := &PathPort{}
		pathPort.PID = process.Pid
		conns, _ := process.Connections()
		var ports = make(map[uint32]uint32)
		for _, conn := range conns {
			ports[conn.Laddr.Port] = conn.Laddr.Port
		}

		for _, port := range ports {
			pathPort.Ports = append(pathPort.Ports, int(port))
		}
		pidInfo := cmdLine
		infos := strings.Split(pidInfo, " ")
		switch runtime.GOOS {
		case "windows":
			re := regexp.MustCompile(`\-Des\.path\.conf="([^\"]+)"`)
			result := re.FindAllStringSubmatch(pidInfo, -1)
			if result == nil {
				continue
			}
			pathPort.Path = result[0][1]

			re = regexp.MustCompile(`\-Des\.path\.home="([^\"]+)"`)
			result = re.FindAllStringSubmatch(pidInfo, -1)
			if result == nil {
				continue
			}
			pathPort.ESHomePath = result[0][1]
		default:
			path := parseESConfigPath(infos)
			if path == "" {
				continue
			}
			pathPort.Path = path
			pathPort.ESHomePath = parseESHomePath(infos)
		}
		pathPorts = append(pathPorts, pathPort)
	}
	return pathPorts, nil
}

/*
*
从进程信息里，解析es配置文件路径
通过getProcessInfo()获取进程信息
*/
func getNodeConfigPaths(processInfos string) *[]PathPort {
	if processInfos == "" {
		return nil
	}
	var pathPorts []PathPort
	sc := bufio.NewScanner(strings.NewReader(processInfos))
	for sc.Scan() {
		pPort := PathPort{}
		pidInfo := sc.Text()
		infos := strings.Split(pidInfo, " ")
		switch runtime.GOOS {
		case "windows":
			re := regexp.MustCompile(`\-Des\.path\.conf="([^\"]+)"`)
			result := re.FindAllStringSubmatch(pidInfo, -1)
			if result == nil {
				continue
			}
			pPort.Path = result[0][1]

			re = regexp.MustCompile(`\-Des\.path\.home="([^\"]+)"`)
			result = re.FindAllStringSubmatch(pidInfo, -1)
			if result == nil {
				continue
			}
			pPort.ESHomePath = result[0][1]
		default:
			path := parseESConfigPath(infos)
			if path == "" {
				continue
			}
			pPort.Path = path
			pPort.ESHomePath = parseESHomePath(infos)
		}

		ports := parseESPort(infos)
		pPort.Ports = ports
		pathPorts = append(pathPorts, pPort)
	}
	return &pathPorts
}

func parseESHomePath(infos []string) string {
	for _, str := range infos {
		if strings.HasPrefix(str, "-Des.path.home") {
			paths := strings.Split(str, "=")
			if len(paths) > 1 {
				return strings.ReplaceAll(paths[1], "\"", "") //这里是针对windows，linux的无需这样处理
			}
		}
	}
	return ""
}

func parseESPort(infos []string) []int {
	//TODO 总感觉这里的逻辑很呆板...
	var pid string
	switch runtime.GOOS {
	case "windows":
		for i := len(infos) - 1; i >= 0; i-- {
			if infos[i] != "" && infos[i] != " " {
				pid = infos[i] //倒序，取第一个不为空字符串的，作为进程ID
				break
			}
		}
	default:
		count := 0
		for _, info := range infos {
			if info != "" && info != " " {
				count++
				if count == 2 {
					pid = info //顺序，取第2个不为空字符串的，作为进程ID
					break
				}
			}
		}
	}
	return getPortByPid(pid)
}

func getPortByPid(pid string) []int {
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		return []int{}
	}
	p, err := process.NewProcess(int32(pidInt)) // Create a new process instance by PID
	if err != nil {
		return []int{}
	}

	// Get all network connections for this process (both listening and connected)
	conns, err := p.Connections()
	if err != nil {
		return []int{}
	}

	listeningPorts := make([]int, 0) // Initialize a slice to hold the listening ports
	for _, conn := range conns {     // Iterate through all connection stats
		// Check if the connection is in "LISTEN" status
		if conn.Status == "LISTEN" {
			// Get the local IP and port
			localAddr := conn.Laddr.IP
			localPort := conn.Laddr.Port

			if localAddr == util.AnyAddress {
				// If listening on 0.0.0.0, it's listening on all interfaces, so include port
				listeningPorts = append(listeningPorts, int(localPort))
			} else {
				// If listening on a specific IP, we need to check if it's a local IP of this machine
				addrs, err := net.InterfaceAddrs()
				if err != nil {
					log.Error("Error getting network interface addresses: %v", err)
					continue // If an error occur get next connection info
				}

				for _, addr := range addrs {
					ipnet, ok := addr.(*net.IPNet)
					// Check if this is a valid non-loopback local IP
					if ok && !ipnet.IP.IsLoopback() && ipnet.IP.Equal(net.ParseIP(localAddr)) {
						listeningPorts = append(listeningPorts, int(localPort))
						break // If local IP matches connection IP , add to the listening ports
					}
				}
			}

		}
	}
	return listeningPorts
}

func parseESConfigPath(infos []string) string {
	for _, str := range infos {
		if strings.HasPrefix(str, "-Des.path.conf") {
			paths := strings.Split(str, "=")
			if len(paths) > 1 {
				return strings.ReplaceAll(paths[1], "\"", "") //这里是针对windows，linux的无需这样处理
			}
		}
	}
	return ""
}

func RemoveCommentInFile(content string) string {
	var builder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "#") {
			builder.WriteString(scanner.Text())
		}
	}
	return builder.String()
}

// nodeInfo : 通过GET /_nodes/_local 获得的信息
func ParseNodeInfo(nodeInfo string) map[string]string {

	result := make(map[string]string)
	nodesInfo := map[string]interface{}{}
	util.MustFromJSONBytes([]byte(nodeInfo), &nodesInfo)
	if nodes, ok := nodesInfo["nodes"]; ok {
		if nodesMap, ok := nodes.(map[string]interface{}); ok {
			for id, v := range nodesMap {
				result["node_id"] = id
				if nodeInfo, ok := v.(map[string]interface{}); ok {
					result["node_name"] = nodeInfo["name"].(string)
					result["version"] = nodeInfo["version"].(string)
				}
			}
		}
	}
	return result
}
