---
title: es_cluster_stats
---

# es_cluster_stats

## 描述

采集集群 stats 指标。

## 配置示例

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

## 参数说明

| 名称 | 类型 | 说明 |
| --- | --- | --- |
| elasticsearch | string | 集群实例名称（请参考 [elasticsearch](https://infinilabs.cn/docs/latest/gateway/references/elasticsearch/) 的 `name` 参数） |
| labels | map | 自定义标签 |