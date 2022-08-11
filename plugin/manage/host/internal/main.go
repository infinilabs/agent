package main

import (
	"fmt"
	"infini.sh/agent/plugin/manage/host"
	"time"
)

func main() {

	defer func() {
		if data := recover(); data != nil {
			fmt.Println(data)
		}
		time.Sleep(time.Second * 60)
	}()

	host, err := host.WindowsTest()
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, cluster := range host.Clusters {
		for _, node := range cluster.Nodes {
			fmt.Println(node)
		}
	}
	time.Sleep(time.Second * 60)
}
