---
weight: 80
title: "版本历史"
---

# 版本发布日志

这里是 INFINI Agent 历史版本发布的相关说明。

## 1.28.2 (2025-02-15)

### Improvements

- Add logs and optimize some settings (#17)
- Enroll failed on docker with diffent user process (#11)
- 同步更新 [Framework v1.1.2](https://docs.infinilabs.com/framework/v1.1.2/docs/references/http_client/) 修复的一些已知问题

## 1.28.1 (2025-01-24)

### Improvements

- 修复一个空指针判断的问题
- 同步更新 Framework 修复的一些已知问题

## 1.28.0 (2025-01-11)

### Improvements

- 同步更新 Framework 修复的一些已知问题

## 1.27.0 (2024-12-13)

### Improvements

- 代码开源，统一采用 [Github 仓库](https://github.com/infinilabs/agent) 进行开发
- 与 INFINI Console 统一版本号
- 同步更新 Framework 修复的已知问题
- 支持 K8S 环境指标采集

## 1.26.1 (2024-08-13)

### Improvements

- 与 INFINI Console 统一版本号
- 同步更新 Framework 修复的已知问题

## 1.26.0 (2024-06-07)

### Improvements

- 与 INFINI Console 统一版本号
- 同步更新 Framework 修复的已知问题

## 1.25.0 (2024-04-30)

### Improvements

- 保持与 Console 相同版本号
- 同步更新 Framework 修复的已知问题

## 1.24.0 (2024-04-15)

### Improvements

- 保持与 Console 相同版本号

## 1.23.0 (2024-03-01)

### Bug fix

- 修复删除实例队列后消费的 Offset 未重置问题

## 1.22.0 (2024-01-26)

### Improvements

- 与 INFINI Console 统一版本号

## 0.7.1 (2023-12-01)

### Features

- 添加 http processor

### Bug fix

- 修复由 Framework Bug 造成连接数不释放、内存异常增长的问题

### Improvements

- 进一步优化内存占用，降到 50M 以下

## 0.7.0 (2023-11-03)

### Breaking changes

### Features

- 限制探针资源消耗，限制 CPU 的使用
- 优化探针内存使用，10 倍降低
- 支持集中配置管理，支持动态下发
- 支持探针一键安装和自动注册
- 优化节点指标采集，仅采集本节点指标

### Bug fix

### Improvements

- 节点统计的重构
- 删除无用的文件
- 修复节点发现问题
- 避免远程配置问题导致的 panic
- 添加发现未知节点的 API
- 重构节点发现逻辑
- 根据新 API 进行重构

## 0.6.1 (2023-08-03)

### Bug fix

- 修复发现节点进程信息时获取 ES 节点端口不对的问题

## 0.6.0 (2023-07-21)

### Improvements

- 采集监控指标添加 cluster_uuid 信息

### Bug fix

- 修复发现节点进程信息时获取不到最新集群配置的问题

## 0.5.1 (2023-06-30)

### Improvements

- 优化查看节点日志文件性能

## 0.5.0 (2023-06-08)

### Features

- 支持将 Agent 注册到 Console
- 添加保存配置到动态加载目录接口

### Improvements

- 优化自动发现 Easysearch 实例进程
- 优化查看 Easysearch 实例日志相关 API

## 0.4.0 (2023-05-10)

### Features

- 新增 `logs_processor` ，配置采集本地日志文件

### Breaking changes

- `es_logs_processor` 调整日志字段
  - `created`重命名为`timestamp`
  - 自动提取`payload.timestamp` `payload.@timestmap`字段到`timestamp`
- `es_logs_processor` 删除 `enable`选项

## 0.3.0 (2023-04-14)

### Features

- 新增 `es_cluster_health` 采集 Easysearch 集群健康信息
- 新增 `es_cluster_stats` 采集 Easysearch 集群 stats
- 新增 `es_index_stats` 采集 Easysearch 索引 stats
- 新增 `es_node_stats` 采集 Easysearch 节点 stats
- 新增 `es_logs_processor` 采集 Easysearch 日志
