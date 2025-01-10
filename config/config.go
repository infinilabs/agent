package config

//
//import (
//	"encoding/json"
//	"errors"
//	log "github.com/cihub/seelog"
//	"infini.sh/agent/model"
//	"infini.sh/framework/core/env"
//	"infini.sh/framework/core/event"
//	"infini.sh/framework/core/global"
//	"infini.sh/framework/core/kv"
//	"infini.sh/framework/core/model"
//	"strconv"
//	"strings"
//	"sync"
//	"time"
//)
//
//type ManagedConfig struct {
//	Manager *Manager `config:"manager"`
//}
//
//type Manager struct {
//	Endpoint string `config:"endpoint"`
//}
//
//var agentConfig *ManagedConfig
//var hostInfo *model.Instance
//var hostInfoObserver []func(newHostInfo *model.Instance)
//var instanceLock sync.RWMutex
//
//const (
//	UrlUploadInstanceInfo string = "/agent/instance"
//	UrlUpdateInstanceInfo        = "/agent/instance/:instance_id"
//	UrlHearBeat                  = "/agent/instance/:instance_id/_heartbeat"
//	UrlGetInstanceInfo           = "/agent/instance/:instance_id"
//)
//
//func InitConfig() {
//	appConfig := &ManagedConfig{}
//	ok, err := env.ParseConfig("configs", appConfig)
//	if err != nil {
//		panic(err)
//	}
//	if !ok {
//		panic("config.InitConfig: can not find agent config")
//	}
//	agentConfig = appConfig
//	hostInfoObserver = make([]func(newHostInfo *model.Instance), 1)
//}
//
//func RegisterHostInfoObserver(fn func(newHostInfo *model.Instance)) {
//	hostInfoObserver = append(hostInfoObserver, fn)
//}
//
//func NotifyHostInfoObserver(newHostInfo *model.Instance) {
//	for i := 0; i < len(hostInfoObserver); i++ {
//		if hostInfoObserver[i] != nil {
//			hostInfoObserver[i](newHostInfo)
//		}
//	}
//}
//
//func GetManagerEndpoint() string {
//	if agentConfig == nil || agentConfig.Manager == nil {
//		return ""
//	}
//	endPoint := agentConfig.Manager.Endpoint
//	if strings.HasSuffix(endPoint, "/") {
//		endPoint = endPoint[:len(endPoint)-1]
//	}
//	return agentConfig.Manager.Endpoint
//}
//
//func GetListenPort() uint {
//	bindAddress := global.Env().SystemConfig.APIConfig.NetworkConfig.Binding
//	if strings.Contains(bindAddress, ":") {
//		temps := strings.Split(bindAddress, ":")
//		port, _ := strconv.Atoi(temps[1])
//		return uint(port)
//	}
//	return 0
//}
//
//func IsHTTPS() bool {
//	return global.Env().SystemConfig.APIConfig.TLSConfig.TLSEnabled
//}
//
//func IsAgentActivated() bool {
//	instanceInfo := GetInstanceInfo()
//	if instanceInfo == nil || !instanceInfo.IsRunning {
//		return false
//	}
//	return true
//}
//
//func GetInstanceInfo() *model.Instance {
//	if hostInfo != nil {
//		return hostInfo
//	}
//	hostInfo = getInstanceInfoFromKV()
//	return hostInfo
//}
//
////func GetOrInitInstanceInfo() *model.Instance {
////	if hostInfo != nil {
////		return hostInfo
////	}
////	hostInfo = getInstanceInfoFromKV()
////	if hostInfo == nil {
////		hostInfo = &model.Instance{
////			IPs:       util.GetLocalIPs(),
////			Host:      agent.HostInfo{},
////		}
////		_, majorIp, _, err := util.GetPublishNetworkDeviceInfo(agentConfig.MajorIpPattern)
////		if err != nil {
////			log.Error(err)
////		}
////		hostInfo.MajorIP = majorIp
////	}
////	SetInstanceInfo(hostInfo)
////	return hostInfo
////}
//
//func UpdateAgentBootTime() {
//	instanceInfo := GetInstanceInfo()
//	instanceInfo.BootTime = time.Now().UnixMilli()
//	SetInstanceInfo(instanceInfo)
//}
//
//func SetInstanceInfo(host *model.Instance) error {
//	if host == nil {
//		return errors.New("host info can not be nil")
//	}
//	instanceLock.Lock()
//	defer instanceLock.Unlock()
//	hostInfo = host
//	event.UpdateAgentID(hostInfo.AgentID)
//	hostByte, _ := json.Marshal(host)
//	if host.IsRunning {
//		NotifyHostInfoObserver(hostInfo)
//	}
//	return kv.AddValue(model.KVInstanceBucket, []byte(model.KVInstanceInfo), hostByte)
//}
//
//func SetInstanceInfoNoNotify(host *model.Instance) error {
//	if host == nil {
//		return errors.New("host info can not be nil")
//	}
//
//	instanceLock.Lock()
//	defer instanceLock.Unlock()
//	hostInfo = host
//	event.UpdateAgentID(hostInfo.AgentID)
//	hostByte, _ := json.Marshal(host)
//	return kv.AddValue(model.KVInstanceBucket, []byte(model.KVInstanceInfo), hostByte)
//}
//
//func DeleteInstanceInfo() error {
//	hostInfo = nil
//	return kv.DeleteKey(model.KVInstanceBucket, []byte(model.KVInstanceInfo))
//}
//
//func ReloadHostInfo() {
//	hostInf := getInstanceInfoFromKV()
//	if hostInf == nil {
//		return
//	}
//	ret, _ := json.Marshal(hostInf)
//	log.Debugf(string(ret))
//}
//
//var host *model.Instance
//
//func getInstanceInfoFromKV() *model.Instance {
//	hs, err := kv.GetValue(model.KVInstanceBucket, []byte(model.KVInstanceInfo))
//	if err != nil {
//		log.Error(err)
//		return nil
//	}
//	if hs == nil {
//		return nil
//	}
//	err = json.Unmarshal(hs, &host)
//	if err != nil {
//		log.Errorf("config.getInstanceInfoFromKV: %v\n", err)
//		return nil
//	}
//	return host
//}
