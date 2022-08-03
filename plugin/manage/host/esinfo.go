package host

import (
	"bufio"
	"fmt"
	"infini.sh/agent/model"
	"infini.sh/framework/core/util"
	"log"
	"src/gopkg.in/yaml.v2"
	"strings"
)

type PathPort struct {
	path  string
	ports []int
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
		ports := parseESPort(infos)
		pathPort := PathPort{path: path, ports: ports}
		if path != "" {
			pathPorts = append(pathPorts, pathPort)
		}
	}
	return &pathPorts
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

func getClusterConfigs(pathPorts *[]PathPort) []*model.Cluster {
	var clusters []*model.Cluster
	clusterMap := make(map[string]*model.Cluster)
	for _, pathPort := range *pathPorts {
		fileName := fmt.Sprintf("%s/elasticsearch.yml", pathPort.path)
		fmt.Println(fileName)
		content, err := util.FileGetContent(fileName)
		if err != nil {
			log.Panic("read es config file failed", err)
			return nil
		}
		//fmt.Println(string(content))
		clusTmp := model.Cluster{}
		if yaml.Unmarshal(content, &clusTmp) == nil {
			if clusTmp.Name == "" {
				clusTmp.Name = "elasticsearch"
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
			node.Ports = pathPort.ports
			cluster.Nodes = append(cluster.Nodes, node)
		}
	}
	for _, v := range clusterMap {
		clusters = append(clusters, v)
	}
	return clusters
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
