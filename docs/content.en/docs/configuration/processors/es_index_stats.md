---
title: es_index_stats
---

# es_index_stats

## Description

Collect the cluster index stats metrics.

## Configuration Example

```yaml
pipeline
  - name: collect_default_es_index_stats
    auto_start: true
    keep_running: true
    retry_delay_in_ms: 3000
    processor:
    - es_index_stats:
        elasticsearch: default
```

## Parameter Description

| Name | Type | Description |
| --- | --- | --- |
| elasticsearch | string | Cluster instance name (Please see [elasticsearch](../../../gateway/references/elasticsearch.md) `name` parameter) |
| all_index_stats | bool | Whether to enable the metric collection of all indexes, default is `true`. |
| index_primary_stats | bool | Whether to enable the metric collection of index primary shards, default is `true`. |
| labels | map | Custom labels |