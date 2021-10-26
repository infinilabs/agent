/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package network

import (
	"infini.sh/agent/metrics/common"
	"infini.sh/agent/store"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
	log "github.com/cihub/seelog"
	"github.com/shirou/gopsutil/net"
	"strings"
)

type Metric struct {
	interfaces   map[string]struct{}
	prevCounters networkCounter
	summary bool
	detail bool
}

type networkCounter struct {
	prevNetworkInBytes    uint64
	prevNetworkInPackets  uint64
	prevNetworkOutBytes   uint64
	prevNetworkOutPackets uint64
}

func New() (*Metric, error) {
	cfg := struct {
		Types []string `config:"types"`
		Interfaces []string `config:"interfaces"`
	}{}

	_,err := env.ParseConfig("network",&cfg)
	if err != nil {
		return nil, err
	}

	var interfaceSet map[string]struct{}
	if len(cfg.Interfaces) > 0 {
		interfaceSet = make(map[string]struct{}, len(cfg.Interfaces))
		for _, ifc := range cfg.Interfaces {
			interfaceSet[strings.ToLower(ifc)] = struct{}{}
		}
		log.Debugf("network io stats will be included for %v", interfaceSet)
	}
	me:=&Metric{
		interfaces:   interfaceSet,
		prevCounters: networkCounter{},
	}
	if len(cfg.Types)>0{
		if util.ContainsAnyInArray("summary",cfg.Types){
			 me.summary=true
		 }
		if util.ContainsAnyInArray("interface",cfg.Types){
			 me.detail=true
		 }
	}else {
		me.detail=true
		me.summary=true
	}

	return me, nil
}

func (m *Metric) Collect() error {
	stats, err := net.IOCounters(true)
	if err != nil {
		return errors.Wrap(err, "network io counters")
	}

	var networkInBytes, networkOutBytes, networkInPackets, networkOutPackets uint64

	for _, counters := range stats {
		if m.interfaces != nil {
			// Select stats by interface name.
			name := strings.ToLower(counters.Name)
			if _, include := m.interfaces[name]; !include {
				continue
			}
		}

		if m.detail{
			store.Save("accumulate",common.Event{
				MetricFields: ioCountersToMapStr(counters),
			})
		}

		// accumulate values from all interfaces
		networkInBytes += counters.BytesRecv
		networkOutBytes += counters.BytesSent
		networkInPackets += counters.PacketsRecv
		networkOutPackets += counters.PacketsSent
	}

	if m.summary{
		if m.prevCounters != (networkCounter{}) {
			// convert network metrics from counters to gauges
			store.Save("gauge",common.Event{
				MetricFields: util.MapStr{
					"network": util.MapStr{
						"total": util.MapStr{
							"in": util.MapStr{
								"bytes":   networkInBytes - m.prevCounters.prevNetworkInBytes,
								"packets": networkInPackets - m.prevCounters.prevNetworkInPackets,
							},
							"out": util.MapStr{
								"bytes":   networkOutBytes - m.prevCounters.prevNetworkOutBytes,
								"packets": networkOutPackets - m.prevCounters.prevNetworkOutPackets,
							},
						},
					},
				},
			})
		}
	}

	//total traffics of all interfaces on host
	// update prevCounters
	m.prevCounters.prevNetworkInBytes = networkInBytes
	m.prevCounters.prevNetworkInPackets = networkInPackets
	m.prevCounters.prevNetworkOutBytes = networkOutBytes
	m.prevCounters.prevNetworkOutPackets = networkOutPackets

	return nil
}

func ioCountersToMapStr(counters net.IOCountersStat) util.MapStr {
	return util.MapStr{
	"network": util.MapStr{
		"interface":util.MapStr{
			"name": counters.Name,
			"in": util.MapStr{
				"errors":  counters.Errin,
				"dropped": counters.Dropin,
				"bytes":   counters.BytesRecv,
				"packets": counters.PacketsRecv,
			},
			"out": util.MapStr{
				"errors":  counters.Errout,
				"dropped": counters.Dropout,
				"packets": counters.PacketsSent,
				"bytes":   counters.BytesSent,
			},
	}}}
}
