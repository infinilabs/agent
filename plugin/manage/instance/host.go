/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package instance

import (
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
	"io/ioutil"
	"net/http"
	"strings"
)

func GetInstanceInfo() (*model.Instance, error) {

	instanceInfo := &model.Instance{}
	instanceInfo.IPs = util.GetLocalIPs()
	_, majorIp, _, err := util.GetPublishNetworkDeviceInfo(config.EnvConfig.MajorIpPattern)
	if err != nil {
		return nil, err
	}
	instanceInfo.MajorIP = majorIp
	pathPorts, err := GetNodeInfoFromProcess()
	log.Debugf("host.GetInstanceInfo get pathPorts from process: %v", pathPorts)
	if err != nil {
		return nil, errors.Wrap(err, "host.GetInstanceInfo: get path & port info failed")
	}
	hostInfo, err := collectHostInfo()
	instanceInfo.Host = *hostInfo
	return instanceInfo, nil
}

func RegisterInstance() (*model.Instance, error) {

	host, err := GetInstanceInfo()
	if err != nil {
		return nil, err
	}
	host.TLS = config.IsHTTPS()
	host.AgentPort = config.GetListenPort()

	schema := "http"
	if host.TLS {
		schema = "https"
	}
	agInfo := util.MapStr{
		"schema": schema,
		"port": host.AgentPort,
		"ips": host.IPs,
		"major_ip": host.MajorIP,
		"host": host.Host,
	}
	body, err := json.Marshal(agInfo)
	if err != nil {
		return nil, errors.Wrap(err, "host.RegisterInstance: get hostinfo failed")
	}
	log.Debugf("host.RegisterInstance: request to: %s , body: %v\n", config.UrlUploadInstanceInfo, string(body))
	url := fmt.Sprintf("%s%s", config.GetManagerEndpoint(), config.UrlUploadInstanceInfo)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.Wrap(err, "host.RegisterInstance: register host failed")
	}
	defer resp.Body.Close()
	bodyC, _ := ioutil.ReadAll(resp.Body)
	if strings.Contains(string(bodyC), "already exists") {
		return nil, errors.New(fmt.Sprintf("\ncurrent cluster registered\nplease delete first in console\n"))
	}
	log.Debugf("host.RegisterInstance, resp: %s\n", string(bodyC))
	var registerResp model.RegisterResponse
	util.MustFromJSONBytes(bodyC, &registerResp)
	host.AgentID = registerResp.AgentId
	//if result is "acknowledged" => console receive register info, but need user review this request. if passed, console will callback from api
	host.IsRunning = true
	return host, nil
}

func UpdateProcessInfo(){
	instanceInfo := config.GetInstanceInfo()
	if instanceInfo == nil {
		log.Error("host.UpdateProcessInfo: host info in kv lost")
	}
	//pathPorts,err := GetNodeInfoFromProcess()
	//if err != nil {
	//	log.Errorf("host.UpdateProcessInfo:  %v", err)
	//	return
	//}
	config.SetInstanceInfoNoNotify(instanceInfo)
}

func IsRegistered() bool {
	if info := config.GetInstanceInfo(); info != nil {
		if info.AgentID == "" {
			return false
		}
		return true
	}
	hostInfo := config.GetInstanceInfo()
	if hostInfo == nil {
		return false
	}
	log.Info(hostInfo.AgentID)
	if hostInfo.AgentID == "" {
		return false
	}
	return true
}