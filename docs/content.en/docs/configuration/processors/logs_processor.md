---
title: logs_processor
---

# logs_processor

## Description

Collect the local log files.

## Configuration Example

The elasticsearch log files as an example.

```yaml
pipeline:
  - name: log_collect
    auto_start: true
    keep_running: true
    retry_delay_in_ms: 3000
    processor:
      - logs_processor:
          queue_name: "logs"
          logs_path: "/opt/es/elasticsearch-7.7.1/logs"
          # metadata for all log items
          metadata:
            category: elasticsearch
          patterns:
            - pattern: ".*_server.json$"
              type: json
              metadata:
                name: server
              timestamp_fields: ["timestamp", "@timestamp"]
              remove_fields:
                [
                  "type",
                  "cluster.name",
                  "cluster.uuid",
                  "node.name",
                  "node.id",
                  "timestamp",
                  "@timestamp",
                ]
            - pattern: "gc.log$"
              type: text
              metadata:
                name: gc
              timestamp_patterns:
                - "\\d{4}-\\d{1,2}-\\d{1,2}T\\d{1,2}:\\d{1,2}:\\d{1,2}.\\d{3}\\+\\d{4}"
                - "\\d{4}-\\d{1,2}-\\d{1,2} \\d{1,2}:\\d{1,2}:\\d{1,2},\\d{3}"
                - "\\d{4}-\\d{1,2}-\\d{1,2}T\\d{1,2}:\\d{1,2}:\\d{1,2},\\d{3}"
            - pattern: ".*.log$"
              type: multiline
              line_pattern: '^\['
              metadata:
                name: server
              timestamp_patterns:
                - "\\d{4}-\\d{1,2}-\\d{1,2}T\\d{1,2}:\\d{1,2}:\\d{1,2}.\\d{3}\\+\\d{4}"
                - "\\d{4}-\\d{1,2}-\\d{1,2} \\d{1,2}:\\d{1,2}:\\d{1,2},\\d{3}"
                - "\\d{4}-\\d{1,2}-\\d{1,2}T\\d{1,2}:\\d{1,2}:\\d{1,2},\\d{3}"
```

## Parameter Description

| Name | Type | Description |
| --- | --- | --- |
| queue_name | string | Log files collection queue name |
| logs_path | string | Log files path |
| metadata | map | Configure the metadata for the log files |
| patterns | object | Patterns configuration for log files |
| patterns.pattern | string | Pattern for log files |
| patterns.metadata | map | Configure the metadata for the log files which matched |
| patterns.type | string | Log type, support `json`，`text`，`multiline` |
| patterns.line_pattern | string | When the log type is multiline, the pattern for a new line |
| patterns.remove_fields | []string | Fields that need to be removed (available when the log type is `json`) |
| patterns.timestamp_fields | []string | Timestamp field (available when the log type is `json`) |
| patterns.timestamp_patterns | []string | Timestamp pattern (available when the log type is `text` and `multiline` ) |
