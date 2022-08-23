package instance

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"net"

	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/errors"
	"runtime"
	"time"
)

func collectHostInfo() (*agent.HostInfo, error) {
	//TODO 数据已经拿到了，等实体确认后，继续
	hostInfo := &agent.HostInfo{
		OS: agent.OSInfo{},
	}
	var err error
	hostInfo.Name, _, hostInfo.OS.Name, _, hostInfo.OS.Version, hostInfo.OS.Arch, err = getHostInfo()
	if err != nil {
		return nil, err
	}
	//log.Info(getCPUInfo())
	//log.Info(getDiskInfo())
	//log.Info(getHostInfo())
	//log.Info(getMemoryInfo())
	//log.Info(getSwapInfo())
	//log.Info(getMacAddress())
	return hostInfo, nil
}

func getCPUInfo() (physicalCnt int, logicalCnt int, totalPercent float64, modelName string, err error) {
	physicalCnt, err = cpu.Counts(false) //物理内核数
	if err != nil {
		return 0, 0, 0, "", err
	}
	logicalCnt, err = cpu.Counts(true) //逻辑内核数
	if err != nil {
		return 0, 0, 0, "", err
	}
	totalPercents, err := cpu.Percent(3*time.Second, false) //过去3秒cpu使用率
	if len(totalPercents) > 0 {
		totalPercent = totalPercents[0] //这个使用率
	}
	if err != nil {
		return 0, 0, 0, "", err
	}
	cpuInfos, _ := cpu.Info()
	for _, info := range cpuInfos {
		modelName = info.ModelName
	}
	return physicalCnt, logicalCnt, totalPercent, modelName, nil
}

func getDiskInfo() (total uint64, free uint64, used uint64, usedPercent float64, err error) {
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:"
	}
	v, err := disk.Usage(path)
	if err != nil {
		log.Errorf("error %v", err)
		return 0, 0, 0, 0, err
	}
	if v.Path != path {
		log.Errorf("error %v", err)
		return 0, 0, 0, 0, nil
	}
	total = v.Total
	free = v.Free
	used = v.Used
	usedPercent = v.UsedPercent
	return total, free, used, usedPercent, nil
}

func getHostInfo() (hostName string, bootTime uint64, platform string, platformVersion string, kernelVersion string, kernelArch string, err error) {
	v, err := host.Info()
	if err != nil {
		return "", 0, "", "", "", "", err
	}
	empty := &host.InfoStat{}
	if v == empty {
		return "", 0, "", "", "", "", errors.New(fmt.Sprintf("Could not get hostinfo %v", v))
	}
	if v.Procs == 0 {
		return "", 0, "", "", "", "", errors.New("Could not determine the number of host processes")
	}
	hostName = v.Hostname
	bootTime = v.BootTime
	platform = v.Platform
	platformVersion = v.PlatformVersion
	kernelVersion = v.KernelVersion
	kernelArch = v.KernelArch
	return hostName, bootTime, platform, platformVersion, kernelVersion, kernelArch, nil
}

func getMemoryInfo() (total uint64, available uint64, used uint64, usedPercent float64, err error) {
	if runtime.GOOS == "solaris" {
		return 0, 0, 0, 0, errors.New("Only .Total is supported on Solaris")
	}

	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	empty := &mem.VirtualMemoryStat{}
	if v == empty {
		return 0, 0, 0, 0, errors.New("computer.memoryInfo: mem.VirtualMemoryStat is empty")
	}

	total = v.Used + v.Free + v.Buffers + v.Cached
	available = v.Available
	used = v.Used
	usedPercent = v.UsedPercent
	return total, available, used, usedPercent, nil
}

func getSwapInfo() (total uint64, used uint64, free uint64, usedPercent float64, err error) {
	v, err := mem.SwapMemory()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	empty := &mem.SwapMemoryStat{}
	if v == empty {
		return 0, 0, 0, 0, errors.New("computer.swapInfo: mem.SwapMemoryStat is empty")
	}
	total = v.Total
	used = v.Used
	free = v.Free
	usedPercent = v.UsedPercent
	return total, used, free, usedPercent, nil
}

func getMacAddress() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var macs []string
	for _, inter := range interfaces {
		mac := inter.HardwareAddr
		if mac.String() != "" {
			macs = append(macs, mac.String())
		}
	}
	return macs, nil
}
