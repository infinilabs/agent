---
title: es_node_stats
---

# es_node_stats

## 描述

采集集群节点 stats 指标。

## 配置示例

```yaml
pipeline
  - name: collect_default_es_node_stats
    auto_start: true
    keep_running: true
    retry_delay_in_ms: 3000
    processor:
    - es_node_stats:
        elasticsearch: default
```

## 参数说明

| 名称 | 类型 | 说明 |
| --- | --- | --- |
| elasticsearch | string | 集群实例名称（请参考 [elasticsearch](../../../gateway/references/elasticsearch.md) 的 `name` 参数） |
| level | string | 指标级别，可选 `cluster`，`indices`，`shards`，默认 `shards`。 |
| labels | map | 自定义标签 |