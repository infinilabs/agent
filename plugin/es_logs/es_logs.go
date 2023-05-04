package es_logs

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	log "github.com/cihub/seelog"

	"infini.sh/agent/plugin/logs"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

const name = "es_logs_processor"

type EsLogsProcessor struct {
	cfg           Config
	logProcessors []pipeline.Processor
	configs       []*logs.Config
	patterns      []*logs.Pattern
}

type CollectConfig struct {
	Server       CollectConfigItem `config:"server"`
	SearchSlow   CollectConfigItem `config:"search_slow"`
	IndexingSlow CollectConfigItem `config:"indexing_slow"`
	Deprecation  CollectConfigItem `config:"deprecation"`
	Audit        CollectConfigItem `config:"audit"`
	GC           CollectConfigItem `config:"gc"`
}

type CollectConfigItem struct {
	JSON bool `config:"json"`
	Text bool `config:"text"`
}

type Config struct {
	QueueName     string        `config:"queue_name"`
	Elasticsearch string        `config:"elasticsearch"`
	Metadata      util.MapStr   `config:"metadata,omitempty"`
	Collect       CollectConfig `config:"collect"`
}

func init() {
	pipeline.RegisterProcessorPlugin(name, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		QueueName: "es_logs",
		Collect: CollectConfig{
			Server: CollectConfigItem{
				JSON: true,
			},
			SearchSlow: CollectConfigItem{
				JSON: true,
			},
			IndexingSlow: CollectConfigItem{
				JSON: true,
			},
			Deprecation: CollectConfigItem{
				JSON: true,
			},
			Audit: CollectConfigItem{
				JSON: true,
			},
			GC: CollectConfigItem{
				Text: true,
			},
		},
	}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of echo processor: %s", err)
	}

	p := &EsLogsProcessor{
		cfg: cfg,
	}

	return p, nil
}

func (p *EsLogsProcessor) Name() string {
	return name
}

func (p *EsLogsProcessor) Process(c *pipeline.Context) error {
	if len(p.logProcessors) == 0 {
		p.configs = p.GetLocalConfigs()
		if len(p.configs) == 0 {
			return nil
		}
		for _, config := range p.configs {
			logProcessor, err := logs.NewFromConfig(*config)
			if err != nil {
				return fmt.Errorf("failed to generate sub processor, err: %v", err)
			}
			if logProcessor != nil {
				p.logProcessors = append(p.logProcessors, logProcessor)
			}
		}
	}

	for _, processor := range p.logProcessors {
		err := processor.Process(c)
		if err != nil {
			log.Errorf("failed to run sub processor, err: %v", err)
		}
	}

	return nil
}

// TODO: refresh regularly
func (p *EsLogsProcessor) GetLocalConfigs() []*logs.Config {
	if p.configs != nil {
		return p.configs
	}
	var configs []*logs.Config
	client := elastic.GetClientNoPanic(p.cfg.Elasticsearch)
	if client == nil {
		log.Errorf("failed to client for [%s]", p.cfg.Elasticsearch)
		return nil
	}
	localIPs := util.GetLocalIPs()
	log.Debugf("local ips: %v", localIPs)
	nodes, err := client.GetNodes()
	if err != nil {
		log.Errorf("failed to get nodes info from elasticsearch, err: %v", err)
		return nil
	}
	if nodes == nil {
		log.Error("elasticsearch cluster has no node")
		return nil
	}
	localNodes := []elastic.NodesInfo{}
	localNodeIds := []string{}
	for nodeId, node := range *nodes {
		if util.StringInArray(localIPs, node.Host) {
			localNodeIds = append(localNodeIds, nodeId)
			localNodes = append(localNodes, node)
			continue
		}
		// handle the special case that ES is running on localhost
		ipAddr := net.ParseIP(node.Host)
		if ipAddr != nil && ipAddr.IsLoopback() {
			localNodeIds = append(localNodeIds, nodeId)
			localNodes = append(localNodes, node)
			continue
		}
	}

	meta := elastic.GetMetadata(p.cfg.Elasticsearch)
	if meta.ClusterState == nil {
		log.Infof("elasticsearch [%s] metadata not available yet, skip reading logs. make sure elastic.metadata_refresh.enabled is true", meta.Config.Name)
		return nil
	}

	for i := range localNodeIds {
		nodeId := localNodeIds[i]
		nodeInfo := localNodes[i]

		if err != nil {
			log.Error(err)
			return configs
		}

		tempUrl, err := url.Parse("http://" + nodeInfo.Http.PublishAddress)
		if err != nil {
			log.Error(err)
			return configs
		}
		labels := map[string]interface{}{
			"cluster_config_name": meta.Config.Name,
			"cluster_config_id":   meta.Config.ID,
			"cluster_config_uuid": meta.Config.ClusterUUID,
			"node_uuid":           nodeId,
			"node_name":           nodeInfo.Name,
			"port":                tempUrl.Port(),
		}
		labels["cluster_uuid"] = meta.ClusterState.ClusterUUID
		labels["cluster_name"] = meta.ClusterState.ClusterName

		var logsPath string
		settings := util.MapStr(nodeInfo.Settings)
		logsPathVar, err := settings.GetValue("path.logs")
		if err == nil {
			logsPath, _ = util.ExtractString(logsPathVar)
			logsPath = fixLogPath(logsPath)
		}
		if logsPath == "" {
			log.Error("failed to get logs path for node [%s] of cluster [%s]", nodeInfo.Name, meta.Config.Name)
			continue
		}

		metadata := util.MapStr{
			"category": "elasticsearch",
			"labels":   labels,
		}
		metadata.Update(p.cfg.Metadata)
		nodeConfig := &logs.Config{
			QueueName: p.cfg.QueueName,
			LogsPath:  logsPath,
			Metadata:  metadata,
			Patterns:  p.generatePatterns(),
		}
		log.Debugf("collecting logs at path [%s] for node [%s] from cluster [%s]", logsPath, nodeInfo.Name, meta.Config.Name)
		configs = append(configs, nodeConfig)
	}

	log.Debugf("local node configs: %s", util.MustToJSON(configs))
	p.configs = configs
	return configs
}

var duplicateKeys = []string{"type", "cluster.name", "cluster.uuid", "node.name", "node.id", "timestamp", "@timestamp", "elasticsearch.cluster.name", "elasticsearch.cluster.uuid", "elasticsearch.node.id", "elasticsearch.node.name"}
var timestampKeys = []string{"timestamp", "@timestamp"}
var timestampPatterns = []string{
	"\\d{4}-\\d{1,2}-\\d{1,2}T\\d{1,2}:\\d{1,2}:\\d{1,2}.\\d{3}\\+\\d{4}",
	"\\d{4}-\\d{1,2}-\\d{1,2} \\d{1,2}:\\d{1,2}:\\d{1,2},\\d{3}",
	"\\d{4}-\\d{1,2}-\\d{1,2}T\\d{1,2}:\\d{1,2}:\\d{1,2},\\d{3}",
}

func (p *EsLogsProcessor) generatePatterns() []*logs.Pattern {
	if len(p.patterns) > 0 {
		return p.patterns
	}
	if p.cfg.Collect.Server.JSON {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*_server.json$",
			Metadata: util.MapStr{
				"name": "server",
			},
			RemoveFields:    duplicateKeys,
			TimestampFields: timestampKeys,
			Type:            logs.FileTypeJSON,
		})
	}
	if p.cfg.Collect.SearchSlow.JSON {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*_index_search_slowlog.json$",
			Metadata: util.MapStr{
				"name": "index_search_slowlog",
			},
			RemoveFields:    duplicateKeys,
			TimestampFields: timestampKeys,
			Type:            logs.FileTypeJSON,
		})
	}
	if p.cfg.Collect.IndexingSlow.JSON {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*_index_indexing_slowlog.json$",
			Metadata: util.MapStr{
				"name": "index_indexing_slowlog",
			},
			RemoveFields:    duplicateKeys,
			TimestampFields: timestampKeys,
			Type:            logs.FileTypeJSON,
		})
	}
	if p.cfg.Collect.Deprecation.JSON {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*_deprecation.json$",
			Metadata: util.MapStr{
				"name": "deprecation",
			},
			RemoveFields:    duplicateKeys,
			TimestampFields: timestampKeys,
			Type:            logs.FileTypeJSON,
		})
	}
	if p.cfg.Collect.Audit.JSON {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*_audit.json$",
			Metadata: util.MapStr{
				"name": "audit",
			},
			RemoveFields:    duplicateKeys,
			TimestampFields: timestampKeys,
			Type:            logs.FileTypeJSON,
		})
	}
	if p.cfg.Collect.GC.Text {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: "gc.log$",
			Metadata: util.MapStr{
				"name": "gc",
			},
			Type:              logs.FileTypeMultiline,
			LinePattern:       "^\\[",
			TimestampPatterns: timestampPatterns,
		})
	}
	if p.cfg.Collect.SearchSlow.Text {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*_index_search_slowlog.log$",
			Metadata: util.MapStr{
				"name": "index_search_slowlog",
			},
			Type:              logs.FileTypeMultiline,
			LinePattern:       "^\\[",
			TimestampPatterns: timestampPatterns,
		})
	}
	if p.cfg.Collect.IndexingSlow.Text {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*_index_indexing_slowlog.log$",
			Metadata: util.MapStr{
				"name": "index_indexing_slowlog",
			},
			Type:              logs.FileTypeMultiline,
			LinePattern:       "^\\[",
			TimestampPatterns: timestampPatterns,
		})
	}
	if p.cfg.Collect.Deprecation.Text {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*_deprecation.log$",
			Metadata: util.MapStr{
				"name": "deprecation",
			},
			Type:              logs.FileTypeMultiline,
			LinePattern:       "^\\[",
			TimestampPatterns: timestampPatterns,
		})
	}
	if p.cfg.Collect.Audit.Text {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*_audit.log$",
			Metadata: util.MapStr{
				"name": "audit",
			},
			Type:              logs.FileTypeMultiline,
			LinePattern:       "^\\[",
			TimestampPatterns: timestampPatterns,
		})
	}
	if p.cfg.Collect.Server.Text {
		p.patterns = append(p.patterns, &logs.Pattern{
			Pattern: ".*.log$",
			Metadata: util.MapStr{
				"name": "server",
			},
			Type:              logs.FileTypeMultiline,
			LinePattern:       "^\\[",
			TimestampPatterns: timestampPatterns,
		})
	}
	return p.patterns
}

func fixLogPath(path string) string {
	if !strings.HasPrefix(path, "/") {
		return strings.ReplaceAll(path, `/`, `\`)
	}
	return path
}
