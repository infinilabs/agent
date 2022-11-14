package logs

import (
	"context"
	"encoding/json"
	"fmt"
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
	"src/github.com/buger/jsonparser"
	"strings"
	"sync"
	"time"
)

type LogsProcessor struct {
	cfg       Config `config:"cfg"`
	watcher   *FileWatcher
	agentMeta *event2.AgentMeta
	lock      sync.RWMutex
}

type Config struct {
	Enable bool `config:"enable"`
}

func init() {
	pipeline.RegisterProcessorPlugin("logs_processor", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of echo processor: %s", err)
	}

	return &LogsProcessor{
		cfg:     cfg,
		watcher: NewFileWatcher(),
	}, nil
}

func (p *LogsProcessor) Name() string {
	return "logs_processor"
}

func (p *LogsProcessor) Process(c *pipeline.Context) error {
	task.RunWithinGroup("logs_processor", func(ctx context.Context) error {
		p.watcher.Watch(p.GetAllMeta(), ctx)
		return nil
	})

	var fsEvent FSEvent
	for !c.IsCanceled() {
		fsEvent = p.watcher.Event()
		if fsEvent.Op == OpDone {
			return nil
		}
		p.onFSEvent(fsEvent)
	}
	return nil
}

func (p *LogsProcessor) onFSEvent(event FSEvent) {
	switch event.Op {
	case OpCreate, OpWrite:
		if event.Op == OpCreate {
			log.Debugf("A new file %s has been found", event.Path)
			event.State.OffSet = 0
		} else if event.Op == OpWrite {
			log.Debugf("File %s has been updated", event.Path)
		}
		p.ReadLogs(event)
	case OpTruncate:
		log.Debugf("File %s has been truncated", event.Path)
		event.State.OffSet = 0
		p.ReadLogs(event)
	default:
		log.Error("Unknown return value %v", event.Op)
	}
}

func (p *LogsProcessor) ReadLogs(event FSEvent) {
	log.Debugf("logs process, start read logs: %s", util.MustToJSON(event))
	if strings.HasSuffix(event.Path, ".log") {
		p.ReadPlainTextLogs(event)
	} else {
		p.ReadJsonLogs(event)
	}
}

func (p *LogsProcessor) ReadJsonLogs(event FSEvent) {
	h, _ := harvester.NewHarvester(event.Path, event.State.OffSet)
	r, err := h.NewLogRead(false)
	var msg reader.Message
	offset := event.State.OffSet
	var logMapStr util.MapStr
	for {
		msg, err = r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			break
		}
		offset += int64(len(msg.Content))
		err = json.Unmarshal(deleteDuplicateFieldsInLog(msg.Content), &logMapStr)
		if err != nil {
			log.Error(err)
			continue
		}
		event.LogMeta.File.Offset = offset
		p.Save(event, logMapStr)
	}
	event.State = FileState{
		Name:    event.Info.Name(),
		Size:    event.Info.Size(),
		ModTime: event.Info.ModTime(),
		IsDir:   event.Info.IsDir(),
		Path:    event.Path,
		OffSet:  offset,
	}
	SaveFileState(event.Path, event.State)
}

func (p *LogsProcessor) ReadPlainTextLogs(event FSEvent) {
	h, _ := harvester.NewHarvester(event.Path, event.State.OffSet)
	r, err := h.NewPlainTextRead(false)
	var msg reader.Message
	offset := event.State.OffSet
	var metaMapStr util.MapStr
	var logTime string
	var logContent string
	for {
		msg, err = r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			break
		}
		offset += int64(len(msg.Content))
		err = json.Unmarshal(util.MustToJSONBytes(event), &metaMapStr)
		if err != nil {
			log.Error(err)
			continue
		}
		logTime = parseGCLogTime(string(msg.Content))
		event.LogMeta.File.Offset = offset
		if logTime != "" {
			logContent = strings.ReplaceAll(string(msg.Content), logTime, "")
			logTime = strings.ReplaceAll(logTime, "[", "")
			logTime = strings.ReplaceAll(logTime, "]", "")
			p.Save(event, util.MapStr{"timestamp": logTime, "message": logContent})
		} else {
			logContent = string(msg.Content)
			p.Save(event, util.MapStr{"message": logContent})
		}
	}
	event.State = FileState{
		Name:    event.Info.Name(),
		Size:    event.Info.Size(),
		ModTime: event.Info.ModTime(),
		IsDir:   event.Info.IsDir(),
		Path:    event.Path,
		OffSet:  offset,
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
		Timestamp: time.Now(),
		AgentMeta: *p.GetAgentMeta(),
		Meta:      event.LogMeta,
		Fields:    logContent,
	}
	var eventMapStr util.MapStr
	err := json.Unmarshal(util.MustToJSONBytes(logEvent), &eventMapStr)
	if err != nil {
		log.Error(err)
		return
	}
	queue.Push(queue.GetOrInitConfig(logEvent.AgentMeta.QueueName), util.MustToJSONBytes(eventMapStr))
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
			QueueName: "es_logs",
			AgentID:   instanceInfo.AgentID,
			HostID:    instanceInfo.HostID,
			Hostname:  util.GetHostName(),
			MajorIP:   publicIP,
			IP:        util.GetLocalIPs(),
			Tags:      nil,
			Labels:    nil,
		}
	}
	return p.agentMeta
}

func (p *LogsProcessor) judgeType(path string) string {
	if strings.HasSuffix(path, "server.json") {
		return "server"
	}
	if strings.HasSuffix(path, "index_search_slowlog.json") {
		return "index_search_slowlog"
	}
	if strings.HasSuffix(path, "index_indexing_slowlog.json") {
		return "index_indexing_slowlog"
	}
	if strings.HasSuffix(path, "deprecation.json") {
		return "deprecation"
	}
	if strings.HasSuffix(path, "audit.json") {
		return "audit"
	}
	if strings.HasSuffix(path, "gc.log") {
		return "gc"
	}
	return ""
}

func parseGCLogTime(content string) string {
	reg := regexp.MustCompile("^\\[.*?\\]")
	if reg == nil {
		fmt.Println("regexp err")
		return ""
	}
	result := reg.FindStringSubmatch(content)
	if len(result) == 0 {
		return ""
	}
	return result[0]
}

func deleteDuplicateFieldsInLog(logs []byte) []byte {
	keys := []string{"type", "cluster.name", "cluster.uuid", "node.name", "node.id"}
	for _, key := range keys {
		logs = jsonparser.Delete(logs, key)
	}
	return logs
}
