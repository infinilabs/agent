package manage

import (
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/agent/config"
	"infini.sh/agent/model"
	"infini.sh/agent/plugin/manage/instance"
	"infini.sh/framework/core/util"
	"strings"
	"time"
)

func Init() {
	_, err := isAgentAliveInConsole()
	if err != nil {
		log.Errorf("manage.Init: %v", err)
		return
	}

	if instance.IsRegistered() {
		//checkInstanceUpdate()
		config.UpdateAgentBootTime()
	} else {
		registerChan := make(chan bool)
		go Register(registerChan)
		select {
		case ok := <-registerChan:
			log.Debugf("manage.Init: register host %t", ok)
			if ok {
				//checkInstanceUpdate()
				config.UpdateAgentBootTime()
			}
		case <-time.After(time.Second * 60):
			log.Error("manage.Init: register timeout.")
		}
	}
}

//get agent info from console. if nil => delete kv. if not => update task info.
func isAgentAliveInConsole() (bool, error) {
	hostInfo := config.GetInstanceInfo()
	if hostInfo == nil || hostInfo.AgentID == ""{
		return false, nil
	}

	resp, err := GetHostInfoFromConsole(hostInfo.AgentID)
	if err != nil {
		return false, err
	}
	if !resp.Found {
		config.DeleteInstanceInfo()
		return false, nil
	}
	if !hostInfo.IsRunning {
		log.Debug("agent not running, wait console confirm")
		return false, nil
	}
	hostInfo.AgentPort = config.GetListenPort()
	hostInfo.TLS = config.IsHTTPS()
	config.SetInstanceInfo(hostInfo)
	return true, nil
}

func GetHostInfoFromConsole(agentID string) (*model.GetAgentInfoResponse, error) {
	reqPath := strings.ReplaceAll(config.UrlGetInstanceInfo, ":instance_id", agentID)
	url := fmt.Sprintf("%s%s", config.GetManagerEndpoint(), reqPath)
	var req = util.NewGetRequest(url, []byte(""))
	result, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}
	var resp model.GetAgentInfoResponse
	err = json.Unmarshal(result.Body, &resp)
	return &resp, err
}

//func checkInstanceUpdate() {
//	hostUpdateTask := task.ScheduleTask{
//		Description: "update agent host info",
//		Type:        "interval",
//		Interval:    "10s",
//		Task: func(ctx context.Context) {
//			if config.GetInstanceInfo() == nil || !config.GetInstanceInfo().IsRunning {
//				return
//			}
//			instance.UpdateProcessInfo()
//			isChanged, err := instance.IsHostInfoChanged()
//			if err != nil {
//				log.Errorf("manage.checkInstanceUpdate: update host info failed : %v", err)
//				return
//			}
//			if !isChanged {
//				return
//			}
//			log.Debugf("manage.checkInstanceUpdate: host info change")
//			updateChan := make(chan bool)
//			go UpdateInstanceInfo(updateChan)
//
//			select {
//			case ok := <-updateChan:
//				log.Debugf("manage.checkInstanceUpdate: update host info %t", ok)
//			case <-time.After(time.Second * 60):
//			}
//		},
//	}
//	task.RegisterScheduleTask(hostUpdateTask)
//}

func UpdateInstanceInfo(isSuccess chan bool) {

	hostKV := config.GetInstanceInfo()
	hostPid, err := instance.GetInstanceInfo()
	if err != nil {
		log.Errorf("get host info failed: %v", err)
		isSuccess <- false
		return
	}
	hostPid.IsRunning = hostKV.IsRunning
	hostPid.AgentID = hostKV.AgentID
	hostPid.AgentPort = hostKV.AgentPort
	hostPid.HostID = hostKV.HostID
	hostPid.BootTime = hostKV.BootTime
	hostPid.TLS = hostKV.TLS
	hostPid.MajorIP = hostKV.MajorIP
	isSuccess <- true
}

func Register(success chan bool) {
	log.Info("register agent to console")
	instanceInfo, err := instance.RegisterInstance()
	if err != nil {
		log.Errorf("manage.Register: %v\n", err)
		success <- false
		return
	}
	if instanceInfo == nil {
		log.Errorf("manage.Register: register agent Failed. all passwords are wrong?? es crashed?? cluster not register in console??\n")
		success <- false
		return
	}
	if instanceInfo != nil {
		if instanceInfo.IsRunning {
			log.Debugf("manage.Register: %v\n", instanceInfo)
			config.SetInstanceInfo(instanceInfo)
			success <- true
			return
		}
	} else {
		success <- false
	}
}

//func RegisterCallback(resp *model.RegisterResponse) (bool, error) {
//	log.Debugf("manage.RegisterCallback: %v\n", util.MustToJSON(resp))
//	instanceInfo, err := instance.UpdateClusterInfoFromResp(config.GetInstanceInfo(), resp)
//	if err != nil {
//		return false, err
//	}
//	instanceInfo.IsRunning = true
//	config.SetInstanceInfo(instanceInfo)
//	return true, nil
//}
//
