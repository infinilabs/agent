package instance

import (
	"fmt"
	"infini.sh/framework/core/agent"
	log "src/github.com/cihub/seelog"
	"src/github.com/shirou/gopsutil/cpu"
	"time"
)

func CollectHostInfo() *agent.HostInfo {

	return nil
}

func HardwareInfo() {
	physicalCnt, err := cpu.Counts(false) //物理内核数
	if err != nil {
		log.Errorf("error %v", err)
	}
	logicalCnt, err := cpu.Counts(true) //逻辑内核数
	if err != nil {
		log.Errorf("error %v", err)
	}
	totalPercent, err := cpu.Percent(3*time.Second, false) //过去3秒cpu使用率
	if err != nil {
		log.Errorf("error %v", err)
	}
	log.Infof("physicalCnt: %d, logicalCnt: %d, totalPercent: %.2f\n", physicalCnt, logicalCnt, totalPercent)
	cpuInfos, _ := cpu.Info()
	for _, info := range cpuInfos {
		log.Infof("cpuInfo: %v\n", info)
		fmt.Printf("cpuInfo: %v\n", info)
	}
	fmt.Printf("physicalCnt: %d, logicalCnt: %d, totalPercent: %.2f\n", physicalCnt, logicalCnt, totalPercent)

}
