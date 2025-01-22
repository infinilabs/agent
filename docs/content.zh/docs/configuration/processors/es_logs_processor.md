---
title: es_logs_processor
---

# es_logs_processor

## 描述

采集集群日志。

## 配置示例

```yaml
pipeline
  - name: collect_default_es_logs
    auto_start: true
    keep_running: true
    retry_delay_in_ms: 3000
    processor:
    - es_logs_processor:
        elasticsearch: default
        logs_path: "/opt/dev-environment/ecloud/logs"
        queue_name: logs
```

## 参数说明

| 名称 | 类型 | 说明 |
| --- | --- | --- |
| queue_name | string | 日志采集队列名称 |
| elasticsearch | string | 集群实例名称（请参考 [elasticsearch](https://infinilabs.cn/docs/latest/gateway/references/elasticsearch/) 的 `name` 参数） |
| logs_path | string | 集群日志存储路径 |
| labels | map | 自定义标签 |