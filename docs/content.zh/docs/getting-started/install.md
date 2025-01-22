---
weight: 10
title: 下载安装
asciinema: true
---

# 安装 INFINI Agent

INFINI Agent 支持主流的操作系统和平台，程序包很小，没有任何额外的外部依赖，安装起来应该是很快的 ：）

## 下载安装

**自动安装**

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p agent
```

> 通过以上脚本可自动下载相应平台的 agent 最新版本并解压到/opt/agent

> 脚本的可选参数如下：

> &nbsp;&nbsp;&nbsp;&nbsp;_-v [版本号]（默认采用最新版本号）_

> &nbsp;&nbsp;&nbsp;&nbsp;_-d [安装目录]（默认安装到/opt/agent）_

**手动安装**

根据您所在的操作系统和平台选择下面相应的下载地址：

[https://release.infinilabs.com/agent/](https://release.infinilabs.com/agent/)

## 配置服务后台运行

如果希望将 INFINI Agent 以后台服务任务的方式运行，如下：

```
➜ ./agent -service install
Success
➜ ./agent -service start
Success
```

卸载服务也很简单，如下：

```
➜ ./agent -service stop
Success
➜ ./agent -service uninstall
Success
```
