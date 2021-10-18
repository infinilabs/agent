/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package network

import (
	"github.com/shirou/gopsutil/net"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
	"strings"
)

type MetricSet struct {
	interfaces   map[string]struct{}
	prevCounters networkCounter
}

type networkCounter struct {
	prevNetworkInBytes    uint64
	prevNetworkInPackets  uint64
	prevNetworkOutBytes   uint64
	prevNetworkOutPackets uint64
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		Interfaces []string `config:"interfaces"`
	}{}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	var interfaceSet map[string]struct{}
	if len(config.Interfaces) > 0 {
		interfaceSet = make(map[string]struct{}, len(config.Interfaces))
		for _, ifc := range config.Interfaces {
			interfaceSet[strings.ToLower(ifc)] = struct{}{}
		}
		debugf("network io stats will be included for %v", interfaceSet)
	}

	return &MetricSet{
		interfaces:    interfaceSet,
		prevCounters:  networkCounter{},
	}, nil
}

// Fetch fetches network IO metrics from the OS.
func (m *MetricSet) Fetch() error {
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

		isOpen := r.Event(mb.Event{
			MetricSetFields: ioCountersToMapStr(counters),
		})

		// accumulate values from all interfaces
		networkInBytes += counters.BytesRecv
		networkOutBytes += counters.BytesSent
		networkInPackets += counters.PacketsRecv
		networkOutPackets += counters.PacketsSent

		if !isOpen {
			return nil
		}
	}

	if m.prevCounters != (networkCounter{}) {
		// convert network metrics from counters to gauges
		r.Event(mb.Event{
			RootFields: util.MapStr{
				"host": util.MapStr{
					"network": util.MapStr{
						"ingress": util.MapStr{
							"bytes":   networkInBytes - m.prevCounters.prevNetworkInBytes,
							"packets": networkInPackets - m.prevCounters.prevNetworkInPackets,
						},
						"egress": util.MapStr{
							"bytes":   networkOutBytes - m.prevCounters.prevNetworkOutBytes,
							"packets": networkOutPackets - m.prevCounters.prevNetworkOutPackets,
						},
					},
				},
			},
		})
	}

	// update prevCounters
	m.prevCounters.prevNetworkInBytes = networkInBytes
	m.prevCounters.prevNetworkInPackets = networkInPackets
	m.prevCounters.prevNetworkOutBytes = networkOutBytes
	m.prevCounters.prevNetworkOutPackets = networkOutPackets

	return nil
}

func ioCountersToMapStr(counters net.IOCountersStat) util.MapStr {
	return util.MapStr{
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
	}
}

