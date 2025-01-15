---
title: es_node_stats
---

# es_node_stats

## Description

Collect the cluster node stats metrics.

## Configuration Example

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

## Parameter Description

| Name | Type | Description |
| --- | --- | --- |
| elasticsearch | string | Cluster instance name (Please see [elasticsearch](../../../gateway/references/elasticsearch.md) `name` parameter) |
| level | string | Metric level, Optional `cluster`, `indices`, `shards`, default is `shards`。 |
| labels | map | Custom labels |