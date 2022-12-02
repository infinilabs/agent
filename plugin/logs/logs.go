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
	"infini.sh/agent/model"
	"infini.sh/framework/core/config"
	event2 "infini.sh/framework/core/event"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"io"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

type LogsProcessor struct {
	cfg       Config `config:"cfg"`
	watcher   *FileDetector
	agentMeta *event2.AgentMeta
	lock      sync.RWMutex
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

const name = "logs_processor"

type Config struct {
	Enable bool `config:"enable"`
	QueueName string `json:"queue_name"`
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

	if !cfg.Enable {
		return nil, nil
	}

	return &LogsProcessor{
		cfg:     cfg,
		watcher: NewFileDetector(),
	}, nil
}

func (p *LogsProcessor) Name() string {
	return name
}

func (p *LogsProcessor) Process(c *pipeline.Context) error {
	task.RunWithinGroup(name, func(ctx context.Context) error {
		p.watcher.Detect(p.GetAllMeta(), ctx)
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
		Name:    event.Info.Name(),
		Size:    event.Info.Size(),
		ModTime: event.Info.ModTime(),
		Path:    event.Path,
		Offset:  offset,
		CreateTime: time.Unix(event.Info.Sys().(*syscall.Stat_t).Birthtimespec.Sec, event.Info.Sys().(*syscall.Stat_t).Birthtimespec.Nsec),
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
		Name:    event.Info.Name(),
		Size:    event.Info.Size(),
		ModTime: event.Info.ModTime(),
		Path:    event.Path,
		Offset:  offset,
		CreateTime: time.Unix(event.Info.Sys().(*syscall.Stat_t).Birthtimespec.Sec, event.Info.Sys().(*syscall.Stat_t).Birthtimespec.Nsec),
	}
	SaveFileState(event.Path, event.State)
}

func (p *LogsProcessor) GetAllMeta() []*LogMeta {

	instanceInfo := config2.GetInstanceInfo()
	if instanceInfo == nil || len(instanceInfo.Clusters) == 0 {
		return nil
	}
	var metas []*LogMeta
	for _, cluster := range instanceInfo.Clusters {
		for _, node := range cluster.Nodes {
			if node.Status == model.NodeStatusOnline && node.ID != "" {
				metas = append(metas, &LogMeta{
					Cluster: Cluster{
						Name: cluster.Name,
						ID:   cluster.ID,
						UUID: cluster.UUID,
					},
					Node: Node{
						Name: node.Name,
						ID:   node.ID,
						Port: node.HttpPort,
					},
					File: File{
						Path:   node.LogPath,
						Offset: 0,
					},
				})
			}
		}
	}
	log.Debugf("logs process, get metas: %s", util.MustToJSON(metas))
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
		instanceInfo := config2.GetInstanceInfo()
		_, publicIP, _, _ := util.GetPublishNetworkDeviceInfo(instanceInfo.MajorIP)
		p.agentMeta = &event2.AgentMeta{
			QueueName: p.cfg.QueueName,
			AgentID:   instanceInfo.AgentID,
			HostID:    instanceInfo.HostID,
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
	if strings.HasSuffix(path, LogTypeServer) {
		return logTypes[LogTypeServer]
	}
	if strings.HasSuffix(path, LogTypeDeprecation) {
		return logTypes[LogTypeDeprecation]
	}
	if strings.HasSuffix(path, LogTypeAudit) {
		return logTypes[LogTypeAudit]
	}
	if strings.HasSuffix(path, LogTypeIndexingSlow) {
		return logTypes[LogTypeIndexingSlow]
	}
	if strings.HasSuffix(path, LogTypeSearchSlow) {
		return logTypes[LogTypeSearchSlow]
	}
	if strings.HasSuffix(path, LogTypeGC) {
		return logTypes[LogTypeGC]
	}
	return "unknown"
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
