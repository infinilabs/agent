---
weight: 20
title: 参数配置
bookCollapseSection: true
---

# 参数配置

- [env](http://infinilabs.cn/docs/latest/gateway/references/config/#使用环境变量)
- [path](http://infinilabs.cn/docs/latest/gateway/references/config/#path)
- [log](http://infinilabs.cn/docs/latest/gateway/references/config/#log)
- [configs](http://infinilabs.cn/docs/latest/gateway/references/config/#configs)
- [api](http://infinilabs.cn/docs/latest/gateway/references/config/#api)
- [badger](http://infinilabs.cn/docs/latest/gateway/references/config/#badger)
- [disk_queue](http://infinilabs.cn/docs/latest/gateway/references/config/#本地磁盘队列)
- [elasticsearch](https://infinilabs.cn/docs/latest/gateway/references/elasticsearch/)
- [resource_limit](http://infinilabs.cn/docs/latest/gateway/references/config/#资源限制)
- [metrics](http://infinilabs.cn/docs/latest/gateway/references/config/#metrics)
- [node](http://infinilabs.cn/docs/latest/gateway/references/config/#node)
- [processors](./processors/_index.md)
- [其他参数](http://infinilabs.cn/docs/latest/gateway/references/config/#其它配置-1)

### Agent 特有

- `configs.allow_generated_metrics_tasks`：是否开启自动生成监控任务（默认 `false`，在需要自动生成任务的 Kubernetes 部署中打开）。
