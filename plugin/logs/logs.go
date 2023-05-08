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

	log "github.com/cihub/seelog"
	config2 "infini.sh/agent/config"
	"infini.sh/agent/lib/reader/harvester"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	event2 "infini.sh/framework/core/event"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

type LogsProcessor struct {
	cfg       Config
	watcher   *FileDetector
	agentMeta *event2.AgentMeta
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
	QueueName string      `config:"queue_name"`
	LogsPath  string      `config:"logs_path"`
	Metadata  util.MapStr `config:"metadata"`
	Patterns  []*Pattern  `config:"patterns"`
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
	p := &LogsProcessor{
		cfg:     cfg,
		watcher: NewFileDetector(cfg.LogsPath, cfg.Patterns),
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
	task.RunWithinGroup(name, func(ctx context.Context) error {
		p.watcher.Detect(ctx)
		return nil
	})
	var fsEvent FSEvent
	for !c.IsCanceled() {
		fsEvent = p.watcher.Event()
		if fsEvent.Op == OpDone {
			return nil
		}
		p.onFSEvent(fsEvent, c)
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
		p.Save(event, logContent, timestamp)
	}

	event.State = FileState{
		Name:    event.Info.Name(),
		Size:    event.Info.Size(),
		ModTime: event.Info.ModTime(),
		Path:    event.Path,
		Offset:  offset,
		Sys:     event.Info.Sys(),
	}
	SaveFileState(event.Path, event.State)
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
	var logMessage string
	for !c.IsCanceled() {
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
		p.Save(event, logContent, timestamp)
	}

	event.State = FileState{
		Name:    event.Info.Name(),
		Size:    event.Info.Size(),
		ModTime: event.Info.ModTime(),
		Path:    event.Path,
		Offset:  offset,
		Sys:     event.Info.Sys(),
	}
	SaveFileState(event.Path, event.State)
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
	var logMessage string
	for !c.IsCanceled() {
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
		p.Save(event, logContent, timestamp)
	}

	event.State = FileState{
		Name:    event.Info.Name(),
		Size:    event.Info.Size(),
		ModTime: event.Info.ModTime(),
		Path:    event.Path,
		Offset:  offset,
		Sys:     event.Info.Sys(),
	}
	SaveFileState(event.Path, event.State)
}

func (p *LogsProcessor) Save(event FSEvent, logContent util.MapStr, timestamp string) {
	logEvent := LogEvent{
		AgentMeta: p.GetAgentMeta(),
		Fields:    logContent,
	}
	logEvent.Meta = util.MapStr{
		"log_type": event.Pattern.Type,
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
	queue.Push(queue.GetOrInitConfig(logEvent.AgentMeta.LoggingQueueName), util.MustToJSONBytes(logEvent))
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
		env.ParseConfig("agent.major_ip_pattern", &majorIPPattern)
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
			Tags:             config2.EnvConfig.Tags,
			Labels:           config2.EnvConfig.Labels,
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
