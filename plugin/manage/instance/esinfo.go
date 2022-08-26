package instance

import (
	"bufio"
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
	"io"
	"os"
	"regexp"
	"runtime"
	"src/gopkg.in/yaml.v2"
	"strings"
)

type PathPort struct {
	Path       string
	ESHomePath string
	Ports      []int
}

/**
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
		//pathPort := PathPort{Path: path, Ports: ports, ESHomePath: esHomePath}
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

func getClusterConfigs(pathPorts *[]PathPort) ([]*model.Cluster, error) {

	var clusters []*model.Cluster
	clusterMap := make(map[string]*model.Cluster)
	for _, pathPort := range *pathPorts {
		var fileName string
		switch runtime.GOOS {
		case "windows":
			fileName = fmt.Sprintf("%s\\%s", pathPort.Path, config.ESConfigFileName)
		default:
			fileName = fmt.Sprintf("%s/%s", pathPort.Path, config.ESConfigFileName)
		}
		content, err := util.FileGetContent(fileName)
		if err != nil {
			log.Errorf("read es config file failed, path: %s\n path2: %s", fileName, pathPort.Path)
			continue
			//return nil, errors.Wrap(err, fmt.Sprintf("read es config file failed, path: %s\n path2: %s", fileName, pathPort.Path))
		}
		var nodeYml *model.Node
		err = yaml.Unmarshal(content, &nodeYml)
		if err != nil {
			return nil, err
		}
		if nodeYml == nil {
			nodeYml = &model.Node{}
		}
		nodeYml.ConfigFileContent = []byte(RemoveCommentInFile(string(content)))
		if nodeYml.ClusterName == "" {
			nodeYml.ClusterName = config.ESClusterDefaultName
		}
		nodeYml.ClusterName = strings.ToLower(nodeYml.ClusterName)
		if nodeYml.LogPath == "" {
			switch runtime.GOOS {
			case "windows":
				nodeYml.LogPath = fmt.Sprintf("%s\\%s", pathPort.ESHomePath, "logs")
			default:
				nodeYml.LogPath = fmt.Sprintf("%s/%s", pathPort.ESHomePath, "logs")
			}
		}
		clusterUUID, err := parseClusterUUID(nodeYml.LogPath)
		if err != nil {
			log.Debugf("host.getClusterConfigs: parse cluster uuid failed, path.homePath: %s \npath.log : %s\n %v \n", pathPort.ESHomePath, nodeYml.LogPath, err)
			//continue
		}
		cluster := clusterMap[nodeYml.ClusterName]
		if cluster == nil {
			cluster = &model.Cluster{}
			cluster.Task = &model.Task{
				ClusterMetric: model.ClusterMetricTask{},
				NodeMetric:    &model.NodeMetricTask{},
			}
			cluster.Name = nodeYml.ClusterName
			cluster.UUID = clusterUUID
			cluster.Nodes = []*model.Node{}
			clusterMap[nodeYml.ClusterName] = cluster
		}
		nodeYml.ConfigPath = fileName
		if nodeYml.HttpPort == 0 {
			nodeYml.Ports = pathPort.Ports //yml里没有配置http.port，则把进程里解析到的多个端口都保存下来，拿到用户名密码之后再确认具体端口
		}
		cluster.Nodes = append(cluster.Nodes, nodeYml)
	}
	for _, v := range clusterMap {
		clusters = append(clusters, v)
	}
	return clusters, nil
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

func parseClusterUUID(logPath string) (string, error) {
	files, err := os.ReadDir(logPath)
	var filePath string
	if err != nil {
		return "", errors.Wrap(err, "parseClusterUUID failed")
	}
	for _, file := range files {
		if strings.Contains(file.Name(), "server.json") {
			//filePath = fmt.Sprintf("%s/%s", logPath, file.Name())
			switch runtime.GOOS {
			case "windows":
				filePath = fmt.Sprintf("%s\\%s", logPath, file.Name())
			default:
				filePath = fmt.Sprintf("%s/%s", logPath, file.Name())
			}
		}
	}
	if filePath == "" {
		return "", errors.New(fmt.Sprintf("cannot find server.json in the path: %s", logPath))
	}
	jsonFile, err := os.Open(filePath)
	if err != nil {
		return "", errors.Wrap(err, "read server.json failed")
	}
	defer jsonFile.Close()
	buf := bufio.NewReader(jsonFile)
	for {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		content := string(line)
		if strings.Contains(content, "cluster.uuid") {
			retMap := make(map[string]string)
			var jsonObject interface{}
			err := json.Unmarshal(line, &jsonObject)
			if err != nil {
				continue
			}
			util.MustFromJSONBytes(line, &retMap)
			if ret, ok := retMap["cluster.uuid"]; ok {
				return ret, nil
			}
		}
	}
	return "", errors.New(fmt.Sprintf("read %s success, but cannot find uuid", filePath))
}

//
//func SetESConfig(config *model.ESConfig) error {
//	var origin model.ESConfig
//	content, err := util.FileGetContent(config.ConfigPath)
//	if err != nil {
//		log.Printf("read es config file failed")
//		return err
//	}
//	yaml.Unmarshal(content, origin)
//	origin.HttpPort = config.HttpPort
//	origin.ClusterName = config.ClusterName
//	ret, _ := yaml.Marshal(origin)
//	_, err = util.FilePutContent(config.ConfigPath, string(ret))
//	return err
//}

//nodeInfo : 通过GET /_nodes/_local 获得的信息
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
