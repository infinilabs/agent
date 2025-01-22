---
title: es_logs_processor
---

# es_logs_processor

## Description

Collect the cluster log files.

## Configuration Example

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

## Parameter Description

| Name | Type | Description |
| --- | --- | --- |
| queue_name | string | Log collection queue name |
| elasticsearch | string | Cluster instance name (Please see [elasticsearch](https://docs.infinilabs.com/gateway/main/docs/references/elasticsearch/) `name` parameter) |
| logs_path | string | Cluster log path |
| labels | map | Custom labels |