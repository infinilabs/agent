/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
	"fmt"
	"time"

	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/shipper"
)

const (
	defaultShipBatchSize     = 500
	defaultShipFlushInterval = time.Second
)

// emitter routes log-event envelopes to either the local queue (durable
// buffering; the default) or a direct shipper (ship_direct mode).
//
// In ship mode the file itself plus offset checkpoints provide the
// durability: envelopes are batched in memory (bounded by batchSize),
// and the committed offset only advances after a batch is delivered.
// On delivery failure the pending batch is dropped and the file is
// re-read from the committed offset on the next scan -- the local disk
// queue is not needed.
//
// Harvesting is strictly sequential (one file at a time in the Process
// loop), so the emitter keeps a single in-flight batch without locks.
type emitter struct {
	shipMode   bool
	queueName  string
	queueType  string // 空=缺省 disk; "kafka"=直写 Kafka 总线
	batchSize  int
	flushEvery time.Duration

	shipper shipper.Shipper
	ticker  *time.Ticker

	batch     [][]byte // in-flight envelopes (ship mode)
	offsets   []int64  // end-offset of each envelope in the batch
	committed int64    // end-offset of the last delivered event
}

// newEmitter builds the emitter described by cfg; queue mode needs no
// external resources, ship mode resolves the named shipper.
func newEmitter(cfg Config) (*emitter, error) {
	e := &emitter{
		shipMode:  cfg.ShipDirect,
		queueName: cfg.QueueName,
		queueType: cfg.QueueType,
		batchSize: cfg.ShipBatchSize,
	}
	if !e.shipMode {
		return e, nil
	}

	if e.batchSize <= 0 {
		e.batchSize = defaultShipBatchSize
	}
	if cfg.ShipFlushInterval != "" {
		d, err := time.ParseDuration(cfg.ShipFlushInterval)
		if err != nil || d <= 0 {
			return nil, fmt.Errorf("%s: invalid ship_flush_interval %q", name, cfg.ShipFlushInterval)
		}
		e.flushEvery = d
	} else {
		e.flushEvery = defaultShipFlushInterval
	}

	shipperName := cfg.Shipper
	if shipperName == "" {
		shipperName = "otlp"
	}
	s, err := shipper.Get(shipperName, cfg.ShipConfig)
	if err != nil {
		return nil, fmt.Errorf("%s: ship_direct requires the %q shipper: %v", name, shipperName, err)
	}
	e.shipper = s
	e.ticker = time.NewTicker(e.flushEvery)
	return e, nil
}

// beginFile resets the per-file batch state; committed starts at the
// file's current offset.
func (e *emitter) beginFile(startOffset int64) {
	e.batch = e.batch[:0]
	e.offsets = e.offsets[:0]
	e.committed = startOffset
}

// emit delivers one envelope. It returns an error when delivery failed;
// the caller must stop reading the file and persist the committed
// offset so the event is re-delivered on the next scan (at-least-once).
func (e *emitter) emit(data []byte, endOffset int64) error {
	if !e.shipMode {
		// EnsureTypedConfig: queue_type 显式指定后端时强制注册 (kafka 总线
		// 模式), 否则按缺省 disk 动态创建。Push 失败即返回错误, 由上层停读
		// 文件、保留位点 —— 天然背压传导到采集端。
		qcfg := queue.EnsureTypedConfig(e.queueType, e.queueName)
		if err := queue.Push(qcfg, data); err != nil {
			return err
		}
		e.committed = endOffset
		return nil
	}

	e.batch = append(e.batch, data)
	e.offsets = append(e.offsets, endOffset)
	if len(e.batch) >= e.batchSize {
		return e.flush()
	}
	return nil
}

// maybeFlush ships the in-flight batch when the flush interval has
// elapsed (non-blocking; call between reads).
func (e *emitter) maybeFlush() error {
	if !e.shipMode || e.ticker == nil || len(e.batch) == 0 {
		return nil
	}
	select {
	case <-e.ticker.C:
		return e.flush()
	default:
		return nil
	}
}

// flush ships the in-flight batch; on success the committed offset
// advances to the last delivered event's end offset. On failure the
// batch is kept so the caller can see the error; the events will be
// re-read from the file (the durable buffer) on the next scan.
func (e *emitter) flush() error {
	if !e.shipMode || len(e.batch) == 0 {
		return nil
	}
	if err := e.shipper.Ship(e.batch); err != nil {
		return err
	}
	e.committed = e.offsets[len(e.offsets)-1]
	e.batch = e.batch[:0]
	e.offsets = e.offsets[:0]
	return nil
}
