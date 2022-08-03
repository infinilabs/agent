package host

import (
	"bufio"
	"bytes"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

/**
从系统进程里获取es进程信息
*/
func getProcessInfo() string {
	cmdErr = nil
	sysType := runtime.GOOS
	if sysType == "windows" {
		log.Panic("windows unsupported")
		return ""
	}
	//ps -ef | grep -v grep | grep elastic
	cmds := []string{"ps", "-ef", "grep", "-v", "grep", "grep", "elastic"}
	var stdout bytes.Buffer
	c1 := exec.Command(cmds[0], cmds[1])
	c2 := exec.Command(cmds[2], cmds[3], cmds[4])
	c3 := exec.Command(cmds[5], cmds[6])
	c2.Stdin, _ = c1.StdoutPipe()
	c3.Stdin, _ = c2.StdoutPipe()
	c3.Stdout = &stdout
	cmdRun(c3.Start)
	cmdRun(c2.Start)
	cmdRun(c1.Run)
	cmdRun(c2.Wait)
	cmdRun(c3.Wait)
	if cmdErr != nil {
		log.Panic("get host process info failed")
		return ""
	}
	return stdout.String()
}

var cmdErr error

func cmdRun(f func() error) {
	if cmdErr != nil {
		return
	}
	cmdErr = f()
}

func getPortByPid(pid string) []int {
	if pid == "" {
		return nil
	}
	cmdErr = nil
	sysType := runtime.GOOS
	if sysType == "windows" {
		log.Panic("windows unsupported")
		return nil
	}
	//lsof -i -P | grep -i LISTEN | grep #port#
	cmds := []string{"lsof", "-i", "-P", "grep", "-i", "LISTEN", "grep", pid}
	var stdout bytes.Buffer
	c1 := exec.Command(cmds[0], cmds[1], cmds[2])
	c2 := exec.Command(cmds[3], cmds[4], cmds[5])
	c3 := exec.Command(cmds[6], cmds[7])
	c2.Stdin, _ = c1.StdoutPipe()
	c3.Stdin, _ = c2.StdoutPipe()
	c3.Stdout = &stdout
	cmdRun(c3.Start)
	cmdRun(c2.Start)
	cmdRun(c1.Run)
	cmdRun(c2.Wait)
	cmdRun(c3.Wait)
	if cmdErr != nil {
		log.Panic("get host process info failed")
		return nil
	}
	out := stdout.String()
	sc := bufio.NewScanner(strings.NewReader(out))
	retMap := make(map[int]int)
	for sc.Scan() {
		info := sc.Text()
		spls := strings.Split(info, " ")
		for _, str := range spls {
			if strings.Contains(str, ":") {
				port := strings.Split(str, ":")[1]
				retInt, err := strconv.Atoi(port)
				if err != nil {
					continue
				}
				retMap[retInt] = retInt
			}
		}
	}

	var ports []int
	for _, v := range retMap {
		ports = append(ports, v)
	}
	return ports
}
