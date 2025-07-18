/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package main

import (
	"bytes"
	"fmt"
	"infini.sh/agent/lib/util"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/keystore"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/model"
	util2 "infini.sh/framework/core/util"
	"infini.sh/framework/lib/go-ucfg"
	"infini.sh/framework/modules/configs/config"
	"os"
	"text/template"
)

func generatedMetricsTasksConfig() error {
	alreadyGenerated, err := kv.GetValue("app", []byte("auto_generated_metrics_tasks"))
	if err != nil {
		return fmt.Errorf("get kv auto_generated_metrics_tasks error: %w", err)
	}
	if string(alreadyGenerated) == "true" {
		return nil
	}
	nodeLabels := global.Env().SystemConfig.NodeConfig.Labels
	var clusterID string
	if len(nodeLabels) > 0 {
		clusterID = nodeLabels["cluster_id"]
	}

	schema := os.Getenv("schema")
	port := os.Getenv("http.port")

	// 如果从环境变量中获取不到 则使用默认值
	if schema == "" {
		schema = "https" //k8s easysearch is always be https protocol
	}
	if port == "" {
		port = "9200" //k8s easysearch port is always 9200
	}
	endpoint := fmt.Sprintf("%s://%s:%s", schema, util2.LocalAddress, port)
	v, err := keystore.GetValue("agent_user")
	if err != nil {
		return fmt.Errorf("get agent_user error: %w", err)
	}
	username := string(v)
	v, err = keystore.GetValue("agent_passwd")
	if err != nil {
		return fmt.Errorf("get agent_passwd error: %w", err)
	}
	password := string(v)
	auth := &model.BasicAuth{
		Username: username,
		Password: ucfg.SecretString(password),
	}
	clusterInfo, err := util.GetClusterVersion(endpoint, auth)
	if err != nil {
		return fmt.Errorf("get cluster info error: %w", err)
	}
	nodeUUID, nodeInfo, err := util.GetLocalNodeInfo(endpoint, auth)
	if err != nil {
		return fmt.Errorf("get local node info error: %w", err)
	}
	nodeLogsPath := nodeInfo.GetPathLogs()
	taskTpl := `configs.template:
  - name: "{{.cluster_id}}_{{.node_uuid}}"
    path: "./config/task_config.tpl"
    variable:
      TASK_ID: "{{.cluster_id}}_{{.node_uuid}}"
      CLUSTER_ID: "{{.cluster_id}}"
      CLUSTER_UUID: "{{.cluster_uuid}}"
      NODE_UUID: "{{.node_uuid}}"
      CLUSTER_VERSION: "{{.cluster_version}}"
      CLUSTER_DISTRIBUTION: "{{.cluster_distribution}}"
      CLUSTER_ENDPOINT: ["{{.cluster_endpoint}}"]
      CLUSTER_USERNAME: "{{.username}}"
      CLUSTER_PASSWORD: "{{.password}}"
      CLUSTER_LEVEL_TASKS_ENABLED: false
      NODE_LEVEL_TASKS_ENABLED: true
      LOG_TASKS_ENABLED: true
      NODE_LOGS_PATH: "{{.node_logs_path}}"
#MANAGED: false`
	tpl, err := template.New("metrics_tasks").Parse(taskTpl)
	if err != nil {
		return fmt.Errorf("parse template error: %w", err)
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, map[string]interface{}{
		"cluster_id":           clusterID,
		"node_uuid":            nodeUUID,
		"cluster_version":      clusterInfo.Version.Number,
		"cluster_distribution": clusterInfo.Version.Distribution,
		"cluster_uuid":         clusterInfo.ClusterUUID,
		"cluster_endpoint":     endpoint,
		"username":             username,
		"password":             password,
		"node_logs_path":       nodeLogsPath,
	})
	if err != nil {
		return fmt.Errorf("execute template error: %w", err)
	}
	err = config.SaveConfigStr("generated_metrics_tasks.yml", buf.String())
	if err != nil {
		return fmt.Errorf("save config error: %w", err)
	}
	err = kv.AddValue("app", []byte("auto_generated_metrics_tasks"), []byte("true"))
	if err != nil {
		return fmt.Errorf("add kv auto_generated_metrics_tasks error: %w", err)
	}
	return nil
}
