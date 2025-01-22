---
title: es_index_stats
---

# es_index_stats

## 描述

采集集群索引 stats 指标。

## 配置示例

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

## 参数说明

| 名称 | 类型 | 说明 |
| --- | --- | --- |
| elasticsearch | string | 集群实例名称（请参考 [elasticsearch](https://infinilabs.cn/docs/latest/gateway/references/elasticsearch/) 的 `name` 参数） |
| all_index_stats | bool | 是否开启所有索引指标的采集，默认 `true` |
| index_primary_stats | bool | 是否开启索引主分片指标的采集，默认 `true` |
| labels | map | 自定义标签 |