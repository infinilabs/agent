package instance

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/errors"
	host2 "infini.sh/framework/core/host"
	gonet "net"
	"runtime"
	"time"
)

var CollectNetIOLastTime time.Time
var NetIOUsageLast *host2.NetIOUsageInfo
var CollectDiskIOLastTime time.Time
var DiskIOUsageLast *host2.DiskIOUsageInfo

func collectHostInfo() (*agent.HostInfo, error) {
	hostInfo := &agent.HostInfo{
		OS: agent.OSInfo{},
	}
	var err error
	hostInfo.Name, _, hostInfo.OS.Name, _, hostInfo.OS.Version, hostInfo.OS.Arch, err = GetOSInfo()
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

func GetCPUInfo() (physicalCnt int, logicalCnt int, totalPercent float64, modelName string, err error) {
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

func GetDiskInfo() (total uint64, free uint64, used uint64, usedPercent float64, err error) {
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

func GetOSInfo() (hostName string, bootTime uint64, platform string, platformVersion string, kernelVersion string, kernelArch string, err error) {
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

func GetMemoryInfo() (total uint64, available uint64, used uint64, usedPercent float64, err error) {
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

func GetSwapInfo() (total uint64, used uint64, free uint64, usedPercent float64, err error) {
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

func GetMacAddress() ([]string, error) {
	interfaces, err := gonet.Interfaces()
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

func GetAllUsageInfo() (*host2.Usage, error) {
	usage := &host2.Usage{}
	var err error
	usage.NetIOUsage, err = GetNetIOUsage()
	if err != nil {
		return nil, err
	}
	usage.DiskUsage, err = GetDiskUsage()
	if err != nil {
		return nil, err
	}
	usage.DiskIOUsage, err = GetDiskIOUsageInfo()
	if err != nil {
		return nil, err
	}
	usage.MemoryUsage, usage.SwapMemoryUsage, err = GetMemoryUsage()
	if err != nil {
		return nil, err
	}
	usage.CPUPercent = GetCPUUsageInfo()
	return usage, nil
}

func GetCPUUsageInfo() float64 {
	_, _, cupPercent, _, err := GetCPUInfo()
	if err != nil {
		log.Error(err)
		return 0
	}
	return cupPercent
}

func GetDiskUsage() (*host2.DiskUsageInfo, error) {
	diskUsage := &host2.DiskUsageInfo{}
	var err error
	diskUsage.Total, diskUsage.Free, diskUsage.Used, diskUsage.UsedPercent, err = GetDiskInfo()
	return diskUsage, err
}

func GetDiskIOUsageInfo() (*host2.DiskIOUsageInfo, error) {
	ret, err := disk.IOCounters()
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, errors.New("instance.GetDiskIOUsageInfo: failed, result is empty")
	}
	empty := disk.IOCountersStat{}
	diskIOUsage := &host2.DiskIOUsageInfo{}
	for _, io := range ret {
		if io != empty {
			diskIOUsage.ReadBytes += io.ReadBytes
			diskIOUsage.WriteBytes += io.WriteBytes
			diskIOUsage.ReadTimeCost += io.ReadTime
			diskIOUsage.WriteTimeCost += io.WriteTime
		}
	}
	if DiskIOUsageLast == nil {
		return diskIOUsage, nil
	}
	var timePeriod uint64 = 10
	timePassed := uint64(time.Now().Unix() - CollectDiskIOLastTime.Unix())
	diskIOUsage.ReadBytes = (diskIOUsage.ReadBytes - DiskIOUsageLast.ReadBytes) / 1000 * timePeriod / timePassed
	diskIOUsage.WriteBytes = (diskIOUsage.WriteBytes - DiskIOUsageLast.WriteBytes) / 1000 * timePeriod / timePassed
	diskIOUsage.WriteTimeCost = (diskIOUsage.WriteTimeCost - DiskIOUsageLast.WriteTimeCost) * timePeriod / timePassed
	diskIOUsage.ReadTimeCost = (diskIOUsage.ReadTimeCost - DiskIOUsageLast.ReadTimeCost) * timePeriod / timePassed
	CollectDiskIOLastTime = time.Now()
	DiskIOUsageLast = diskIOUsage
	return diskIOUsage, nil
}

func GetNetIOUsage() (*host2.NetIOUsageInfo, error) {
	stats, err := net.IOCounters(false)
	if err != nil {
		log.Errorf("Could not get GetNetIOUsage: %v", err)
	}
	if len(stats) < 1 {
		return nil, errors.New("instance.GetNetIOUsage: failed")
	}
	stat := stats[0]
	usage := &host2.NetIOUsageInfo{}
	if NetIOUsageLast == nil {
		usage.BytesSent = stat.BytesSent
		usage.BytesRecv = stat.BytesRecv
		usage.PacketsRecv = stat.PacketsRecv
		usage.PacketsSent = stat.PacketsSent
		return usage, nil
	}
	var timePeriod uint64 = 10
	timePassed := uint64(time.Now().Unix() - CollectNetIOLastTime.Unix())

	usage.BytesRecv = (stat.BytesRecv - NetIOUsageLast.BytesRecv) / 1000 * timePeriod / timePassed
	usage.BytesSent = (stat.BytesSent - NetIOUsageLast.BytesSent) / 1000 * timePeriod / timePassed
	usage.PacketsRecv = (stat.PacketsRecv - NetIOUsageLast.PacketsRecv) * timePeriod / timePassed
	usage.PacketsSent = (stat.PacketsSent - NetIOUsageLast.PacketsSent) * timePeriod / timePassed
	CollectNetIOLastTime = time.Now()
	NetIOUsageLast = usage
	return usage, nil
}

func GetMemoryUsage() (*host2.MemoryUsageInfo, *host2.SwapMemoryUsageInfo, error) {

	memoryUsage := &host2.MemoryUsageInfo{}
	swapUsage := &host2.SwapMemoryUsageInfo{}
	var err error
	memoryUsage.Total, memoryUsage.Available, memoryUsage.Used, memoryUsage.UsedPercent, err = GetMemoryInfo()
	if err != nil {
		return nil, nil, err
	}
	swapUsage.Total, swapUsage.Used, swapUsage.Free, swapUsage.UsedPercent, err = GetSwapInfo()
	if err != nil {
		return nil, nil, err
	}
	return memoryUsage, swapUsage, nil
}
