---
title: es_cluster_health
---

# es_cluster_health

## Description

Collect the cluster health metrics.

## Configuration Example

```yaml
pipeline
  - name: collect_default_es_cluster_health
    auto_start: true
    keep_running: true
    retry_delay_in_ms: 3000
    processor:
    - es_cluster_health:
        elasticsearch: default
```

## Parameter Description

| Name | Type | Description |
| --- | --- | --- |
| elasticsearch | string | Cluster instance name (Please see [elasticsearch](../../../gateway/references/elasticsearch.md) `name` parameter) |
| labels | map | Custom labels |
