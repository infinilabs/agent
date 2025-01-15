---
weight: 10
title: Installing the Agent
asciinema: true
---

# Installing the Agent

INFINI Agent supports mainstream operating systems and platforms. The program package is small, with no extra external dependency. So, the loadgen can be installed very rapidly

## Downloading

**Automatic install**

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p agent
```

> The above script can automatically download the latest version of the corresponding platform's agent and extract it to /opt/agent

> The optional parameters for the script are as follows:

> &nbsp;&nbsp;&nbsp;&nbsp;_-v [version number]（Default to use the latest version number）_

> &nbsp;&nbsp;&nbsp;&nbsp;_-d [installation directory] (default installation to /opt/agent)_

**Manual install**

Select a package for downloading in the following URL based on your operating system and platform:

[https://release.infinilabs.com/agent/](https://release.infinilabs.com/agent/)

## System Service

To run the data platform of INFINI Agent as a background task, run the following commands:

```
➜ ./agent -service install
Success
➜ ./agent -service start
Success
```

Unloading the service is simple. To unload the service, run the following commands:

```
➜ ./agent -service stop
Success
➜ ./agent -service uninstall
Success
```
