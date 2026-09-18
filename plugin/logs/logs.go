/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	"infini.sh/agent/lib/reader/harvester"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	event2 "infini.sh/framework/core/event"
	"infini.sh/framework/core/global"
	log "infini.sh/framework/core/log"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

type LogsProcessor struct {
	cfg       Config
	watcher   *FileDetector
	agentMeta *event2.AgentMeta
	emit      *emitter
	lock      sync.RWMutex
}

const name = "logs_processor"

const (
	FileTypeJSON      = "json"
	FileTypeText      = "text"
	FileTypeMultiline = "multiline"
)

type Pattern struct {
	Pattern           string      `config:"pattern"`
	Metadata          util.MapStr `config:"metadata"`
	Type              string      `config:"type"`
	LinePattern       string      `config:"line_pattern"`
	RemoveFields      []string    `config:"remove_fields"`
	TimestampFields   []string    `config:"timestamp_fields"`
	TimestampPatterns []string    `config:"timestamp_patterns"`

	patternRegex   *regexp.Regexp
	timestampRegex []*regexp.Regexp
}

type Config struct {
	QueueName string `config:"queue_name"`
	// QueueType selects the queue backend (empty = default disk). Set
	// to "kafka" to write logs straight to the Kafka bus (brokers come
	// from the instance-level kafka_queue section), consumed by Gateway.
	QueueType string      `config:"queue_type"`
	LogsPath  string      `config:"logs_path"`
	Metadata  util.MapStr `config:"metadata"`
	Patterns  []*Pattern  `config:"patterns"`

	// ScanInterval controls how often the logs path is re-scanned while
	// the processor runs (default 10s); lower it for closer-to-live
	// tailing. Empty means a single scan per pipeline run (legacy
	// behavior).
	ScanInterval string `config:"scan_interval"`

	// ShipDirect bypasses the local queue: envelopes are shipped
	// straight to the configured shipper (default: OTLP/gRPC to the
	// gateway tier). The file itself plus offset checkpoints provide
	// the durability -- offsets only advance after successful delivery
	// -- so the double disk I/O of a local queue copy is avoided.
	ShipDirect bool `config:"ship_direct"`

	// Shipper names the direct-ship transport (default "otlp").
	Shipper string `config:"shipper"`

	// ShipBatchSize flushes the in-flight batch at this many events
	// (default 500). Memory bound of ship mode.
	ShipBatchSize int `config:"ship_batch_size"`

	// ShipFlushInterval flushes the in-flight batch at least this often
	// while a file is being read (default 1s).
	ShipFlushInterval string `config:"ship_flush_interval"`

	// ShipConfig is passed through to the shipper factory (for "otlp":
	// the same keys as the otlp_export processor).
	ShipConfig map[string]interface{} `config:"ship_config"`
}

func init() {
	pipeline.RegisterProcessorPlugin(name, New)
}

func NewFromConfig(cfg Config) (pipeline.Processor, error) {
	if len(cfg.Patterns) == 0 {
		return nil, nil
	}
	var err error
	var patterns []*Pattern
	for _, pattern := range cfg.Patterns {
		if pattern.Pattern == "" {
			log.Warn("empty pattern, skip")
			continue
		}
		pattern.patternRegex, err = regexp.Compile(pattern.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern regexp: %s", pattern.Pattern)
		}
		for _, timestampPattern := range pattern.TimestampPatterns {
			if timestampPattern != "" {
				timestampRegex, err := regexp.Compile(timestampPattern)
				if err != nil {
					return nil, fmt.Errorf("invalid timestamp_pattern regexp: %s", timestampPattern)
				}
				pattern.timestampRegex = append(pattern.timestampRegex, timestampRegex)
			}
		}
		patterns = append(patterns, pattern)
	}
	emit, err := newEmitter(cfg)
	if err != nil {
		return nil, err
	}
	p := &LogsProcessor{
		cfg:     cfg,
		watcher: NewFileDetector(cfg.LogsPath, cfg.Patterns),
		emit:    emit,
	}

	return p, nil
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{QueueName: "logs"}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of echo processor: %s", err)
	}

	return NewFromConfig(cfg)
}

func (p *LogsProcessor) Name() string {
	return name
}

func (p *LogsProcessor) Process(c *pipeline.Context) error {
	interval := time.Duration(0)
	if p.cfg.ScanInterval != "" {
		d, err := time.ParseDuration(p.cfg.ScanInterval)
		if err != nil || d <= 0 {
			return fmt.Errorf("%s: invalid scan_interval %q", name, p.cfg.ScanInterval)
		}
		interval = d
	}

	// derive from the pipeline context so an in-flight walk aborts when
	// the pipeline stops; no extra goroutine is spawned (the embedded
	// stdlib cancelCtx is linked via the parentCancelCtx fast path)
	scanCtx, cancel := context.WithCancel(c)
	defer cancel()

	first := true
	for !c.IsCanceled() {
		task.RunWithinGroup(name, func(ctx context.Context) error {
			// the detector signals completion with a done event; use a
			// context tied to this scan run
			p.watcher.Detect(scanCtx)
			return nil
		})
		if first {
			first = false
		}
		// drain all pending events of this scan before the next walk
		for !c.IsCanceled() {
			fsEvent := p.watcher.Event()
			if fsEvent.Op == OpDone {
				break
			}
			p.onFSEvent(fsEvent, c)
		}
		if interval <= 0 {
			return nil // legacy single-scan behavior
		}
		select {
		case <-c.Done():
			return nil
		case <-time.After(interval):
		}
	}
	return nil
}

func (p *LogsProcessor) onFSEvent(event FSEvent, c *pipeline.Context) {
	switch event.Op {
	case OpCreate, OpWrite:
		if event.Op == OpCreate {
			log.Debugf("new file %s has been found", event.Path)
			event.State.Offset = 0
		} else if event.Op == OpWrite {
			log.Debugf("file %s has been updated", event.Path)
		}
		p.ReadLogs(event, c)
	case OpTruncate:
		log.Debugf("file %s has been truncated", event.Path)
		event.State.Offset = 0
		p.ReadLogs(event, c)
	default:
		log.Error("unknown return value %v", event.Op)
	}
}

func (p *LogsProcessor) ReadLogs(event FSEvent, c *pipeline.Context) {
	switch event.Pattern.Type {
	case FileTypeJSON:
		p.ReadJsonLogs(event, c)
	case FileTypeText:
		p.ReadPlainTextLogs(event, c)
	case FileTypeMultiline:
		p.ReadMultilineLogs(event, c)
	default:
		log.Errorf("invalid file type [%s] for pattern [%s]", event.Pattern.Type, event.Pattern.Pattern)
	}
}

func (p *LogsProcessor) ReadJsonLogs(event FSEvent, c *pipeline.Context) {
	log.Debugf("reading json logs from [%s], offset: [%d]", event.Path, event.State.Offset)
	offset := event.State.Offset
	p.emit.beginFile(offset)
	h, err := harvester.NewHarvester(event.Path, offset)
	if err != nil {
		log.Errorf("failed to initialize harvester, err: %v", err)
		return
	}
	r, err := h.NewJsonFileReader("^{", false)
	if err != nil {
		log.Errorf("failed to initialize json file reader, err: %v", err)
		return
	}
	for !c.IsCanceled() {
		if err := p.emit.maybeFlush(); err != nil {
			log.Errorf("failed to flush batch for file [%s], err: %v", event.Path, err)
			break
		}
		msg, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("failed to read next message, err: %v", err)
			break
		}
		offset += int64(len(msg.Content))
		event.Offset = offset
		logContent := util.MapStr{}
		err = json.Unmarshal(msg.Content, &logContent)
		if err != nil {
			log.Errorf("failed to extract json from file [%s], err: %v", event.Path, err)
			continue
		}
		logContent, timestamp := processJSON(event.Pattern, logContent)
		if !p.emitEvent(event, logContent, timestamp, offset) {
			break
		}
	}
	p.finishFile(event)
}

func (p *LogsProcessor) ReadPlainTextLogs(event FSEvent, c *pipeline.Context) {
	log.Debugf("reading text logs from [%s], offset: [%d]", event.Path, event.State.Offset)
	h, err := harvester.NewHarvester(event.Path, event.State.Offset)
	if err != nil {
		log.Errorf("failed to initialize harvester, err: %v", err)
		return
	}
	r, err := h.NewLogFileReader("", false)
	if err != nil {
		log.Errorf("failed to initialize log file reader, err: %v", err)
		return
	}
	offset := event.State.Offset
	p.emit.beginFile(offset)
	var logMessage string
	for !c.IsCanceled() {
		if err := p.emit.maybeFlush(); err != nil {
			log.Errorf("failed to flush batch for file [%s], err: %v", event.Path, err)
			break
		}
		msg, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("failed to read next message, err: %v", err)
			break
		}

		offset += int64(len(msg.Content))
		event.Offset = offset

		logMessage = util.UnsafeBytesToString(msg.Content)
		logContent, timestamp := processText(event.Pattern, logMessage)
		logContent["message"] = logMessage
		if !p.emitEvent(event, logContent, timestamp, offset) {
			break
		}
	}
	p.finishFile(event)
}

func (p *LogsProcessor) ReadMultilineLogs(event FSEvent, c *pipeline.Context) {
	log.Debugf("reading text logs from [%s], offset: [%d]", event.Path, event.State.Offset)
	h, err := harvester.NewHarvester(event.Path, event.State.Offset)
	if err != nil {
		log.Errorf("failed to initialize harvester, err: %v", err)
		return
	}
	r, err := h.NewLogFileReader(event.Pattern.LinePattern, false)
	if err != nil {
		log.Errorf("failed to initialize log file reader, err: %v", err)
		return
	}
	offset := event.State.Offset
	p.emit.beginFile(offset)
	var logMessage string
	for !c.IsCanceled() {
		if err := p.emit.maybeFlush(); err != nil {
			log.Errorf("failed to flush batch for file [%s], err: %v", event.Path, err)
			break
		}
		msg, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("failed to read next message, err: %v", err)
			break
		}

		offset += int64(len(msg.Content))
		event.Offset = offset

		logMessage = util.UnsafeBytesToString(msg.Content)
		logContent, timestamp := processText(event.Pattern, logMessage)
		logContent["message"] = logMessage
		if !p.emitEvent(event, logContent, timestamp, offset) {
			break
		}
	}
	p.finishFile(event)
}

// buildEnvelope renders one collected event as the LogEvent envelope
// JSON (the format shared with the queue boundary and the gateway).
func (p *LogsProcessor) buildEnvelope(event FSEvent, logContent util.MapStr, timestamp string) []byte {
	logEvent := LogEvent{
		AgentMeta: p.GetAgentMeta(),
		Fields:    logContent,
	}
	logEvent.Meta = util.MapStr{
		"log_type": event.Pattern.Type,
	}
	// adapt log level to ECSJsonLayout format
	if level, exists := logContent["log.level"]; exists {
		logContent["level"] = level
		delete(logContent, "log.level")
	}
	logEvent.Meta.Update(p.cfg.Metadata)
	logEvent.Meta.Update(event.Pattern.Metadata)
	logEvent.Meta["file"] = File{
		Path:   event.Path,
		Offset: event.Offset,
	}
	if timestamp != "" {
		logEvent.Timestamp = timestamp
	} else {
		logEvent.Timestamp = time.Now().Format(time.RFC3339)
	}
	return util.MustToJSONBytes(logEvent)
}

// emitEvent delivers one event via the emitter (queue or direct
// shipper). It returns false when delivery failed and the read loop
// must stop; the file state stays at the committed offset so the event
// is re-delivered on the next scan (at-least-once).
func (p *LogsProcessor) emitEvent(event FSEvent, logContent util.MapStr, timestamp string, endOffset int64) bool {
	data := p.buildEnvelope(event, logContent, timestamp)
	if err := p.emit.emit(data, endOffset); err != nil {
		log.Errorf("failed to deliver log event from file [%s] at offset %d, will retry on next scan: %v",
			event.Path, endOffset, err)
		return false
	}
	return true
}

// finishFile flushes the in-flight ship batch (if any) and persists the
// file state at the committed offset.
func (p *LogsProcessor) finishFile(event FSEvent) {
	if err := p.emit.flush(); err != nil {
		log.Errorf("failed to flush batch for file [%s], state stays at offset %d: %v",
			event.Path, p.emit.committed, err)
	}
	sysInfo, err := LoadFileID(event.Info, event.Path)
	if err != nil {
		log.Errorf("failed to get file info, err: %v", err)
		return
	}
	event.State = FileState{
		Name:    event.Info.Name(),
		Size:    event.Info.Size(),
		ModTime: event.Info.ModTime(),
		Path:    event.Path,
		Offset:  p.emit.committed,
		Sys:     sysInfo,
	}
	SaveFileState(event.Path, event.State)
}

func (p *LogsProcessor) GetAgentMeta() *event2.AgentMeta {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.agentMeta != nil {
		return p.agentMeta
	}
	if p.agentMeta == nil {
		//instanceInfo := config2.GetInstanceInfo()
		var majorIPPattern string
		env.ParseConfig("node.major_ip_pattern", &majorIPPattern)
		if majorIPPattern == "" {
			majorIPPattern = ".*"
		}
		_, publicIP, _, _ := util.GetPublishNetworkDeviceInfo(majorIPPattern)
		p.agentMeta = &event2.AgentMeta{
			LoggingQueueName: p.cfg.QueueName,
			AgentID:          global.Env().SystemConfig.NodeConfig.ID,
			Hostname:         util.GetHostName(),
			MajorIP:          publicIP,
			IP:               util.GetLocalIPs(),
			Tags:             global.Env().SystemConfig.NodeConfig.Tags,
			Labels:           global.Env().SystemConfig.NodeConfig.Labels,
		}
	}
	return p.agentMeta
}

func processJSON(pattern *Pattern, logContent util.MapStr) (ret util.MapStr, timestamp string) {
	ret = logContent
	for _, key := range pattern.TimestampFields {
		if v, err := logContent.GetValue(key); err == nil {
			if vv, err := util.ExtractString(v); err == nil && vv != "" {
				timestamp = vv
				break
			}
		}
	}
	for _, key := range pattern.RemoveFields {
		// NOTE: logs could contains both "a.b.c" or "a": { "b" : { "c" ...
		// we only support these two cases (not "a.b": { "c" ...)
		delete(ret, key)
		ret.Delete(key)
	}
	return
}

func processText(pattern *Pattern, logMessage string) (ret util.MapStr, timestamp string) {
	ret = util.MapStr{}
	for _, timestampRegex := range pattern.timestampRegex {
		result := timestampRegex.FindStringSubmatch(logMessage)
		if len(result) > 0 {
			timestamp = result[0]
			break
		}
	}
	return
}
