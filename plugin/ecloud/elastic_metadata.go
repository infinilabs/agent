/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package ecloud

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/fsnotify/fsnotify"
	"infini.sh/agent/lib/util"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/modules/elastic/common"
	"os"
)

type ElasticMetadataProcessor struct {
	Input struct{
		Schema string `config:"schema"`
		ESPort int `config:"es_port"`
		ESHostEnv string `config:"es_host_env"`
		ESUsername string `config:"es_username"`
		AuthFile string `config:"auth_file"`
	} `config:"input"`
	Output struct{
		Elasticsearch string `config:"elasticsearch"`
	} `config:"output"`
	dec *Decryptor
	host string
}

func init() {
	pipeline.RegisterProcessorPlugin("ecloud_es_metadata", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	processor := &ElasticMetadataProcessor{}
	if err := c.Unpack(processor); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of node prospector processor: %s", err)
	}
	if processor.Input.ESUsername == ""  {
		return processor, fmt.Errorf("miss input param es_username")
	}
	if processor.Input.AuthFile == ""  {
		return processor, fmt.Errorf("miss input param auth_file")
	}
	if processor.Input.ESHostEnv == ""  {
		return processor, fmt.Errorf("miss input param es_host_env")
	}
	if host, exists := os.LookupEnv(processor.Input.ESHostEnv); !exists {
		return processor, fmt.Errorf("es_host_env [%s] not set", processor.Input.ESHostEnv)
	}else{
		processor.host = host
	}
	if processor.Output.Elasticsearch == ""  {
		return processor, fmt.Errorf("miss ouput param elasticsearch")
	}
	dec := NewDecryptor(processor.Input.ESUsername, processor.Input.AuthFile)
	processor.dec = dec
	return processor, nil
}

func (p *ElasticMetadataProcessor) Name() string {
	return "ecloud_es_metadata"
}

func (p *ElasticMetadataProcessor) Process(c *pipeline.Context) error {
	p.watchAuthFile()
	return p.refreshMetadata()
}

func (p *ElasticMetadataProcessor) watchAuthFile(){
	config.AddPathToWatch(p.Input.AuthFile, func(file string, op fsnotify.Op) {
		log.Debug("auth file changed!!")
		err := p.refreshMetadata()
		if err != nil {
			log.Error(err)
		}
	})
}

func (p *ElasticMetadataProcessor) refreshMetadata() error{
	pwd, err := p.dec.Decrypt()
	if err != nil {
		return fmt.Errorf("decrypt password error: %w", err)
	}
	var cfg elastic.ElasticsearchConfig
	cfg = elastic.ElasticsearchConfig{
		Endpoint: fmt.Sprintf("%s://%s:%d", p.Input.Schema, p.host, p.Input.ESPort),
		Enabled: true,
		BasicAuth: &elastic.BasicAuth{
			Username: p.Input.ESUsername,
			Password: pwd,
		},
	}
	cfg.ID = p.Output.Elasticsearch
	clusterInfo, err := util.GetClusterVersion(cfg.Endpoint, cfg.BasicAuth)
	if err != nil {
		return fmt.Errorf("get cluster info error: %w", err)
	}
	cfg.ClusterUUID = clusterInfo.ClusterUUID
	cfg.Name = clusterInfo.ClusterName
	_, err = common.InitElasticInstance(cfg)
	if err != nil {
		return fmt.Errorf("init elastic client error: %w", err)
	}
	return nil
}