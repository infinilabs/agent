---
title: es_cluster_stats
---

# es_cluster_stats

## Description

Collect the cluster stats metrics.

## Configuration Example

```yaml
pipeline
  - name: collect_default_es_cluster_stats
    auto_start: true
    keep_running: true
    retry_delay_in_ms: 3000
    processor:
    - es_cluster_stats:
        elasticsearch: default
```

## Parameter Description

| Name | Type | Description |
| --- | --- | --- |
| elasticsearch | string | Cluster instance name (Please see [elasticsearch](https://docs.infinilabs.com/gateway/main/docs/references/elasticsearch/) `name` parameter) |
| labels | map | Custom labels |