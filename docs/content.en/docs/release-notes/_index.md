---
weight: 80
title: "Release Notes"
---

# Release Notes

Information about release notes of INFINI Agent is provided here.

## Latest (In development)  
### ❌ Breaking changes  
### 🚀 Features  
- feat: use 'http.port' env var to detect EasySearch HTTP port in Kubernetes
### 🐛 Bug fix  
### ✈️ Improvements  

## 1.29.7 (2025-06-29)
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
### ✈️ Improvements  
- This release includes updates from the underlying [Framework v1.1.9](https://docs.infinilabs.com/framework/v1.1.9), which resolves several common issues and enhances overall stability and performance. While there are no direct changes to AGENT itself, the improvements inherited from Framework benefit AGENT indirectly.

## 1.29.6 (2025-06-13)
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
### ✈️ Improvements  

## 1.29.4 (2025-05-16)
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
- fix: `k8s` env generated metrics tasks `endpoint`
### ✈️ Improvements  
- chore: misc change for node discover log (#31)
- chore: handle panic when module start with check Capcity error (#31)
- chore: update `agent.yml` disable `metadata_refresh` (#31)
- This release includes updates from the underlying [Framework v1.1.7](https://docs.infinilabs.com/framework/v1.1.7/docs/references/http_client/), which resolves several common issues and enhances overall stability and performance. While there are no direct changes to Agent itself, the improvements inherited from Framework benefit Agent indirectly.

## 1.29.3 (2025-04-27)

This release includes updates from the underlying [Framework v1.1.6](https://docs.infinilabs.com/framework/v1.1.6/docs/references/http_client/), which resolves several common issues and enhances overall stability and performance. While there are no direct changes to Agent itself, the improvements inherited from Framework benefit Agent indirectly.

## 1.29.2 (2025-03-31)

This release includes updates from the underlying [Framework v1.1.5](https://docs.infinilabs.com/framework/v1.1.5/docs/references/http_client/), which resolves several common issues and enhances overall stability and performance. While there are no direct changes to Agent itself, the improvements inherited from Framework benefit Agent indirectly.

## 1.29.1 (2025-03-14)

This release includes updates from the underlying [Framework v1.1.4](https://docs.infinilabs.com/framework/v1.1.4/docs/references/http_client/), which resolves several common issues and enhances overall stability and performance. While there are no direct changes to Agent itself, the improvements inherited from Framework benefit Agent indirectly.

## 1.29.0 (2025-02-27)

### Breaking changes

### Features
- Support real-time reading of gzip log files (#22)
- Adapt log level to ECSJsonLayout format (#24)

### Bug fix

### Improvements
- Add metrics enable to the main config (#23)
- This release includes updates from the underlying [Framework v1.1.3](https://docs.infinilabs.com/framework/v1.1.3/docs/references/http_client/), which resolves several common issues and enhances overall stability and performance. While there are no direct changes to Agent itself, the improvements inherited from Framework benefit Agent indirectly.

## 1.28.2 (2025-02-15)

This release includes updates from the underlying [Framework v1.1.2](https://docs.infinilabs.com/framework/v1.1.2/docs/references/http_client/), which resolves several common issues and enhances overall stability and performance. While there are no direct changes to Agent itself, the improvements inherited from Framework benefit Agent indirectly.

### Improvements

- Add logs and optimize some settings (#17)
- Enroll failed on docker with diffent user process  (#11)

## 1.28.1 (2025-01-24)

### Improvements

- Fix the issue with null pointer checks.
- Synchronize updates for known issues fixed in the Framework.

## 1.28.0 (2025-01-11)

### Improvements

- Synchronize updates for known issues fixed in the Framework.

## 1.27.0 (2024-12-13)

### Improvements

- The code is open source, and [Github repository](https://github.com/infinilabs/agent) is used for development.
- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.
- Support collect metrics on Kubernetes

## 1.26.1 (2024-08-13)

### Improvements

- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.

## 1.26.0 (2024-06-07)

### Improvements

- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.

## 1.25.0 (2024-04-30)

### Improvements

- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.

## 1.24.0 (2024-04-15)

### Improvements

- Keep the same version number as INFINI Console.

## 1.23.0 (2024-03-01)

### Bug fix

- Fix the problem that the offset of consumer is not reset after deleting the instance queue.

## 1.22.0 (2024-01-26)

### Improvements

- Unified version number with INFINI Console

## 0.7.1 (2023-12-01)

### Features

- Add `http` processor.

### Bug fix

- Fix the problem of connection numbers not being released and abnormal memory growth caused by the Framework Bug.

### Improvements

- Further optimize the memory usage and reduce it below 50M.

## 0.7.0 (2023-11-03)

### Breaking changes

### Features

- Limit the resource consumption of Agent which cpu etc.
- Optimize the memory used of the agent and reduce it by 10 times.
- Support centralized configuration management and dynamic issuance.
- Support agent one-click install and auto register.
- Optimize the node metrics collection which only collect metrics of this node.

### Bug fix

### Improvements

- Refactor the stats of the node metrics
- Delete useless files.
- Fix the node discovery problem.
- Avoid panic caused by remote configuration issues.
- Add the API for discovering unknown nodes.
- Refactor the node discovery.
- Refactor according to the new API.

## 0.6.1 (2023-08-03)

### Bug fix

- Fix the problem of incorrect ES node port acquisition when discovering node process information.

## 0.6.0 (2023-07-21)

### Improvements

- Add cluster_uuid information when collecting monitoring metrics.

### Bug fix

- Fix the problem of not being able to obtain the latest cluster configuration when discovering node process information.

## 0.5.1 (2023-06-30)

### Improvements

- Optimize the performance of viewing the node log files.

## 0.5.0 (2023-06-08)

### Features

- Support registering the Agent to the Console.
- Add the interface for saving configurations to the dynamic loading directory.

### Improvements

- Optimize the automatic discovery of Easysearch instance processes.
- Optimize the API of viewing Easysearch instance logs.

## 0.4.0 (2023-05-10)

### Features

- Add `logs_processor` and configure to collect local log files.

### Breaking changes

- `es_logs_processor` change log fields.
  - `created` rename to `timestamp`.
  - Automatically extract the `payload.timestamp` and `payload.@timestmap` fields to `timestamp`.
- `es_logs_processor` delete `enable` option.

## 0.3.0 (2023-04-14)

### Features

- Add `es_cluster_health` for collect the cluster health metrics.
- Add `es_cluster_stats` for collect the cluster stats metrics.
- Add `es_index_stats` for collect the cluster index stats metrics.
- Add `es_node_stats` for collect the cluster node stats metrics.
- Add `es_logs_processor` for collect the cluster log.
