/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	log "github.com/cihub/seelog"
	config2 "infini.sh/agent/config"
	"infini.sh/agent/lib/reader"
	"infini.sh/agent/lib/reader/harvester"
	util2 "infini.sh/agent/lib/util"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	event2 "infini.sh/framework/core/event"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"io"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type LogsProcessor struct {
	cfg       Config `config:"cfg"`
	watcher   *FileDetector
	agentMeta *event2.AgentMeta
	lock      sync.RWMutex
	metas []*LogMeta
}

const (
	LogTypeServer       string = "server.json"
	LogTypeSearchSlow          = "index_search_slowlog.json"
	LogTypeIndexingSlow        = "index_indexing_slowlog.json"
	LogTypeDeprecation         = "deprecation.json"
	LogTypeAudit               = "audit.json"
	LogTypeGC                  = "gc.log"
)

var logTypes = map[string]string{
	LogTypeServer:       "server",
	LogTypeSearchSlow:   "index_search_slowlog",
	LogTypeIndexingSlow: "index_indexing_slowlog",
	LogTypeDeprecation:  "deprecation",
	LogTypeAudit:        "audit",
	LogTypeGC:           "gc",
}

const name = "es_logs_processor"

type Config struct {
	Enable bool `config:"enable"`
	QueueName string `json:"queue_name"`
	Elasticsearch string `config:"elasticsearch"`
	LogsPath string `config:"logs_path"`
	Labels map[string]interface{} `config:"labels,omitempty"`
}

var duplicateKeys = []string{"type", "cluster.name", "cluster.uuid", "node.name", "node.id"}
var gcTimeReg = regexp.MustCompile("\\d{4}-\\d{1,2}-\\d{1,2}T\\d{1,2}:\\d{1,2}:\\d{1,2}.\\d{3}\\+\\d{4}")

func init() {
	pipeline.RegisterProcessorPlugin(name, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{QueueName: "es_logs"}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of echo processor: %s", err)
	}
	if cfg.LogsPath == "" {
		return nil, fmt.Errorf("configuration logs_path required")
	}

	if !cfg.Enable {
		return nil, nil
	}
	p := &LogsProcessor{
		cfg:     cfg,
		watcher: NewFileDetector(),
	}

	return p, nil
}

func (p *LogsProcessor) Name() string {
	return name
}

func (p *LogsProcessor) Process(c *pipeline.Context) error {
	task.RunWithinGroup(name, func(ctx context.Context) error {
		p.watcher.Detect(p.GetMetas(), ctx)
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
	log.Debugf("logs process, start read logs: %s", util.MustToJSON(event))
	if strings.HasSuffix(event.Path, ".log") {
		p.ReadPlainTextLogs(event, c)
	} else {
		p.ReadJsonLogs(event, c)
	}
}

func (p *LogsProcessor) ReadJsonLogs(event FSEvent, c *pipeline.Context) {
	h, _ := harvester.NewHarvester(event.Path, event.State.Offset)
	r, err := h.NewLogRead(false)
	var msg reader.Message
	var logContent util.MapStr
	offset := event.State.Offset
	for !c.IsCanceled() {
		msg, err = r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			break
		}
		offset += int64(len(msg.Content))
		event.LogMeta.File.Offset = offset
		msg.Content = deleteDuplicateFieldsInLog(msg.Content)
		err = json.Unmarshal(msg.Content, &logContent)
		if err != nil {
			log.Error(err)
			continue
		}
		p.Save(event, logContent)
	}

	event.State = FileState{
		Name: event.Info.Name(),
		Size: event.Info.Size(),
		ModTime: event.Info.ModTime(),
		Path:    event.Path,
		Offset:  offset,
		Sys: event.Info.Sys(),
	}
	SaveFileState(event.Path, event.State)
}

func (p *LogsProcessor) ReadPlainTextLogs(event FSEvent, c *pipeline.Context) {
	h, _ := harvester.NewHarvester(event.Path, event.State.Offset)
	r, err := h.NewPlainTextRead(false)
	var msg reader.Message
	offset := event.State.Offset
	var logTime string
	var logContent string
	for !c.IsCanceled() {
		msg, err = r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			break
		}

		offset += int64(len(msg.Content))
		event.LogMeta.File.Offset = offset

		logContent = util.UnsafeBytesToString(msg.Content)
		logTime = parseGCLogTime(logContent)
		if logTime != "" {
			p.Save(event, util.MapStr{"timestamp": logTime, "message": logContent})
		} else {
			p.Save(event, util.MapStr{"message": logContent})
		}
	}

	event.State = FileState{
		Name: event.Info.Name(),
		Size: event.Info.Size(),
		ModTime: event.Info.ModTime(),
		Path:    event.Path,
		Offset:  offset,
		Sys: event.Info.Sys(),
	}
	SaveFileState(event.Path, event.State)
}

func (p *LogsProcessor) GetMetas() []*LogMeta {
	if p.metas != nil {
		return p.metas
	}
	var metas []*LogMeta
	client := elastic.GetClientNoPanic(p.cfg.Elasticsearch)
	if client == nil {
		return nil
	}
	meta := elastic.GetMetadata(p.cfg.Elasticsearch)
	nodeId, nodeInfo, err := util2.GetLocalNodeInfo(meta.Config.Endpoint, meta.Config.BasicAuth)
	if err != nil {
		log.Error(err)
		return metas
	}
	tempUrl, err := url.Parse(meta.Config.Endpoint)
	if err != nil {
		log.Error(err)
		return metas
	}
	labels := map[string]interface{}{
		"cluster_name": meta.Config.Name,
		"cluster_id": meta.Config.ID,
		"cluster_uuid": meta.Config.ClusterUUID,
		"node_uuid": nodeId,
		"node_name": nodeInfo.Name,
		"port": tempUrl.Port(),
	}
	if len(p.cfg.Labels) > 0 {
		for k, v := range p.cfg.Labels {
			labels[k] = v
		}
	}
	nodeMeta := &LogMeta{
		Category: "elasticsearch",
		Labels: labels,
		File: File{
			Offset: 0,
			Path: p.cfg.LogsPath,
		},
	}
	metas = append(metas, nodeMeta)

	log.Debugf("logs process, get metas: %s", util.MustToJSON(metas))
	p.metas = metas
	return metas
}

func (p *LogsProcessor) Save(event FSEvent, logContent util.MapStr) {
	event.LogMeta.Category = "elasticsearch"
	event.LogMeta.Name = p.judgeType(event.Path)
	event.LogMeta.File.Path = event.Path
	logEvent := LogEvent{
		AgentMeta: *p.GetAgentMeta(),
		Meta:      event.LogMeta,
		Fields:    logContent,
	}
	logEvent.Created = time.Now()
	queue.Push(queue.GetOrInitConfig(logEvent.AgentMeta.QueueName), util.MustToJSONBytes(logEvent))
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
			QueueName: p.cfg.QueueName,
			AgentID:  global.Env().SystemConfig.NodeConfig.ID,
			Hostname:  util.GetHostName(),
			MajorIP:   publicIP,
			IP:        util.GetLocalIPs(),
			Tags:      config2.EnvConfig.Tags,
			Labels:    config2.EnvConfig.Labels,
		}
	}
	return p.agentMeta
}

func (p *LogsProcessor) judgeType(path string) string {
	var logType string
	switch {
	case strings.HasSuffix(path, LogTypeGC):
		logType = logTypes[LogTypeGC]
	case strings.HasSuffix(path, LogTypeServer):
		logType = logTypes[LogTypeServer]
	case strings.HasSuffix(path, LogTypeDeprecation):
		logType = logTypes[LogTypeDeprecation]
	case strings.HasSuffix(path, LogTypeAudit):
		logType = logTypes[LogTypeAudit]
	case strings.HasSuffix(path, LogTypeIndexingSlow):
		logType = logTypes[LogTypeIndexingSlow]
	case strings.HasSuffix(path, LogTypeSearchSlow):
		logType = logTypes[LogTypeSearchSlow]
	default:
		logType = "unknown"
	}
	return logType
}

func parseGCLogTime(content string) string {
	result := gcTimeReg.FindStringSubmatch(content)
	if len(result) == 0 {
		return ""
	}
	return result[0]
}

func deleteDuplicateFieldsInLog(logs []byte) []byte {
	for _, key := range duplicateKeys {
		logs = jsonparser.Delete(logs, key)
	}
	return logs
}

