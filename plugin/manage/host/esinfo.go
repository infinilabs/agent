package host

import (
	"bufio"
	"fmt"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
	"io"
	"log"
	"os"
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
		pidInfo := sc.Text()
		infos := strings.Split(pidInfo, " ")
		path := parseESConfigPath(infos)
		if path == "" {
			continue
		}
		esHomePath := parseESHomePath(infos)
		ports := parseESPort(infos)
		pathPort := PathPort{Path: path, Ports: ports, ESHomePath: esHomePath}
		pathPorts = append(pathPorts, pathPort)
	}
	return &pathPorts
}

func parseESHomePath(infos []string) string {
	for _, str := range infos {
		if strings.HasPrefix(str, "-Des.path.home") {
			paths := strings.Split(str, "=")
			if len(paths) > 1 {
				return paths[1]
			}
		}
	}
	return ""
}

func parseESPort(infos []string) []int {
	if len(infos) < 4 {
		return nil
	}
	pid := infos[3]
	return getPortByPid(pid)
}

func parseESConfigPath(infos []string) string {
	for _, str := range infos {
		if strings.HasPrefix(str, "-Des.path.conf") {
			paths := strings.Split(str, "=")
			if len(paths) > 1 {
				return paths[1]
			}
		}
	}
	return ""
}

func getClusterConfigs(pathPorts *[]PathPort) ([]*model.Cluster, error) {
	var clusters []*model.Cluster
	clusterMap := make(map[string]*model.Cluster)
	for _, pathPort := range *pathPorts {
		fileName := fmt.Sprintf("%s/%s", pathPort.Path, config.ESConfigFileName)
		content, err := util.FileGetContent(fileName)
		if err != nil {
			return nil, errors.Wrap(err, "read es config file failed")
		}
		clusTmp := model.Cluster{}
		if yaml.Unmarshal(content, &clusTmp) == nil {
			if clusTmp.Name == "" {
				clusTmp.Name = config.ESClusterDefaultName
			}
			if clusTmp.LogPath == "" {
				clusTmp.LogPath = fmt.Sprintf("%s/%s", pathPort.ESHomePath, "logs")
			}
			clusTmp.UUID, err = parseClusterUUID(clusTmp.LogPath)
			if err != nil {
				log.Printf("parse cluster uuid failed, path.log : %s\n %v \n", clusTmp.LogPath, err)
				continue
			}
			cluster := clusterMap[clusTmp.Name]
			if cluster == nil {
				cluster = &model.Cluster{}
				cluster.Name = clusTmp.Name
				cluster.Nodes = []*model.Node{}
				clusterMap[cluster.Name] = cluster
			}
			node := &model.Node{}
			node.ConfigPath = fileName
			node.Ports = pathPort.Ports
			cluster.Nodes = append(cluster.Nodes, node)
		}
	}
	for _, v := range clusterMap {
		clusters = append(clusters, v)
	}
	return clusters, nil
}

func parseClusterUUID(logPath string) (string, error) {
	files, err := os.ReadDir(logPath)
	var filePath string
	if err != nil {
		return "", errors.Wrap(err, "parseClusterUUID failed")
	}
	for _, file := range files {
		if strings.Contains(file.Name(), "server.json") {
			filePath = fmt.Sprintf("%s/%s", logPath, file.Name())
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
func ParseNodeID(nodeInfo string) string {

	nodesInfo := map[string]interface{}{}
	util.MustFromJSONBytes([]byte(nodeInfo), &nodesInfo)
	if nodes, ok := nodesInfo["nodes"]; ok {
		if nodesMap, ok := nodes.(map[string]interface{}); ok {
			for nodeID, _ := range nodesMap {
				return nodeID
			}
		}
	}
	return ""
}
