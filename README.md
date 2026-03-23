<div align="center">

```
   _      ___   __    __  _____
  /_\    / _ \ /__\/\ \ \/__   \
 //_\\  / /_\// \ /  \/ /  / \/
/  _  \/ /_\\//__/ /\  /  / /
\_/ \_/\____/\__/\_\ \/   \/
```

# INFINI Agent

**A lightweight yet powerful cloud agent for managing services and configurations.**

[![License](https://img.shields.io/badge/license-INFINI-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8.svg)](https://golang.org/)
[![Version](https://img.shields.io/badge/version-1.0.0--SNAPSHOT-orange.svg)](config/generated.go)
[![Website](https://img.shields.io/badge/website-infinilabs.com-green.svg)](https://infinilabs.com)

[**Website**](https://infinilabs.com) · [**Documentation**](docs/) · [**Report a Bug**](https://github.com/infinilabs/agent/issues/new) · [**Request a Feature**](https://github.com/infinilabs/agent/issues/new)

</div>

---

## 📖 Overview

INFINI Agent is a lightweight, high-performance cloud agent designed to manage services, configurations, and infrastructure components. It seamlessly integrates with INFINI's ecosystem to provide centralized configuration management, service lifecycle control, real-time metrics collection, and Elasticsearch cluster observability — all from a single binary.

---

## ✨ Features

| Feature | Description |
|---|---|
| 🔧 **Configuration Management** | Automatically pulls and applies configuration changes from remote config servers every second |
| 🔌 **Service Lifecycle Management** | Discovers, starts, and restarts managed services and plugins automatically |
| 📊 **Metrics Collection** | Collects and reports system and application metrics at configurable intervals |
| 🔍 **Elasticsearch Discovery** | Detects and reports local Elasticsearch nodes and their runtime information |
| 📋 **Log Access** | Provides remote access to Elasticsearch log files, including `.gz` archives |
| 🔑 **Keystore Support** | Secures sensitive configuration values using an integrated keystore module |
| 🚀 **Pipeline Engine** | Runs data pipelines for indexing, merging, and processing workloads |
| 📬 **Disk Queue** | Durable message queue for reliable data delivery even during outages |
| 📡 **Status Reporting** | Continuously reports agent and service health back to the management server |
| 🔒 **TLS / mTLS** | Supports encrypted connections for both the API server and config server communication |

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      INFINI Agent                        │
│                                                         │
│  ┌───────────────┐   ┌───────────────┐  ┌────────────┐ │
│  │  Config Sync  │   │   Metrics     │  │  Keystore  │ │
│  │  (every 1s)   │   │  Collection   │  │  Module    │ │
│  └───────┬───────┘   └───────┬───────┘  └────────────┘ │
│          │                   │                          │
│  ┌───────▼───────────────────▼──────────────────────┐  │
│  │               REST API  (:2900)                   │  │
│  │  GET  /elasticsearch/node/_discovery              │  │
│  │  POST /elasticsearch/node/_info                   │  │
│  │  POST /elasticsearch/logs/_list                   │  │
│  │  POST /elasticsearch/logs/_read                   │  │
│  └───────────────────────────────────────────────────┘  │
│                                                         │
│  ┌───────────────┐   ┌───────────────┐  ┌────────────┐ │
│  │ Disk Queue    │   │  Pipeline     │  │  Stats     │ │
│  │ (durable)     │   │  Engine       │  │  Module    │ │
│  └───────────────┘   └───────────────┘  └────────────┘ │
└──────────────────────────┬──────────────────────────────┘
                           │
          ┌────────────────┴─────────────────┐
          │                                  │
  ┌───────▼──────┐                  ┌────────▼──────┐
  │ Config Server│                  │ Elasticsearch │
  │  (managed)   │                  │  Cluster      │
  └──────────────┘                  └───────────────┘
```

---

## 🚀 Getting Started

### Prerequisites

- **Go** 1.23 or higher
- **Git**
- Access to an **INFINI Console / Config Server** (optional, for managed mode)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/infinilabs/agent.git
cd agent

# Build the binary
make build

# (Optional) Build for all platforms
make cross-build
```

### Running the Agent

```bash
# Run with the default configuration
./bin/agent

# Run with a custom configuration file
./bin/agent -config /path/to/agent.yml

# Run in debug mode
./bin/agent -debug
```

---

## ⚙️ Configuration

The agent is configured via `agent.yml`. Below is an annotated example:

```yaml
# ── Paths ─────────────────────────────────────────────────────────────────────
path.data: data          # Directory for persistent data
path.logs: log           # Directory for log files
path.configs: "config"   # Directory for additional config files
configs.auto_reload: true

# ── Resource Limits ───────────────────────────────────────────────────────────
resource_limit.cpu.max_num_of_cpus: 1
resource_limit:
  memory:
    max_in_bytes: 533708800   # ~512 MB

# ── Task Scheduler ────────────────────────────────────────────────────────────
task:
  max_concurrent_tasks: 3

# ── REST API ──────────────────────────────────────────────────────────────────
api:
  enabled: true
  network:
    binding: "0.0.0.0:2900"
  # tls:
  #   enabled: true
  #   cert_file: /etc/ssl.crt
  #   key_file:  /etc/ssl.key
  #   skip_insecure_verify: false

# ── Elasticsearch Integration ─────────────────────────────────────────────────
elastic:
  skip_init_metadata_on_start: true
  health_check:
    enabled: true
    interval: 60s

# ── Managed Configuration (pull from Config Server) ───────────────────────────
configs:
  managed: false             # Set to true to enable managed mode
  interval: "1s"
  servers:
    - "http://localhost:9000"
  max_backup_files: 5
  soft_delete: false
  # tls:                     # mTLS to config server
  #   enabled: true
  #   cert_file: /etc/ssl.crt
  #   key_file:  /etc/ssl.key
  #   skip_insecure_verify: false

# ── Disk Queue ────────────────────────────────────────────────────────────────
disk_queue:
  max_msg_size: 20485760           # 20 MB per message
  max_bytes_per_file: 20485760
  max_used_bytes: 4147483648       # ~4 GB total
  retention.max_num_of_local_files: 1
  compress:
    idle_threshold: 1
    segment:
      enabled: true

# ── Metrics ───────────────────────────────────────────────────────────────────
metrics:
  enabled: true
```

---

## 🔌 API Reference

The agent exposes a REST API on port **2900** by default.

### Elasticsearch Node Discovery

**`GET /elasticsearch/node/_discovery`**

Discover Elasticsearch nodes running on the local host.

```bash
curl http://localhost:2900/elasticsearch/node/_discovery
```

---

### Get Node Info

**`POST /elasticsearch/node/_info`**

Retrieve detailed information about a specific Elasticsearch node by connecting to its endpoint.

```bash
curl -X POST http://localhost:2900/elasticsearch/node/_info \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "http://localhost:9200",
    "basic_auth": {
      "username": "elastic",
      "password": "changeme"
    }
  }'
```

---

### List Log Files

**`POST /elasticsearch/logs/_list`**

List all log files in the specified Elasticsearch logs directory.

```bash
curl -X POST http://localhost:2900/elasticsearch/logs/_list \
  -H "Content-Type: application/json" \
  -d '{ "logs_path": "/var/log/elasticsearch" }'
```

**Response:**
```json
{
  "success": true,
  "result": [
    {
      "name": "elasticsearch.log",
      "size_in_bytes": 102400,
      "modify_time": "2024-01-15T10:30:00Z",
      "total_rows": 1500
    }
  ]
}
```

---

### Read Log File

**`POST /elasticsearch/logs/_read`**

Read lines from an Elasticsearch log file (supports `.gz` compressed files).

```bash
curl -X POST http://localhost:2900/elasticsearch/logs/_read \
  -H "Content-Type: application/json" \
  -d '{
    "logs_path": "/var/log/elasticsearch",
    "file_name": "elasticsearch.log",
    "start_line_number": 0,
    "lines": 100
  }'
```

**Response:**
```json
{
  "success": true,
  "EOF": false,
  "result": [
    {
      "line_number": 1,
      "content": "[2024-01-15T10:00:00,000] [INFO ] ...",
      "bytes": 80,
      "offset": 0
    }
  ]
}
```

---

## 🔄 How It Works

The agent runs a continuous background loop that:

1. **Checks for configuration changes** — polls the config server every second for updated templates, configurations, and keystore entries.
2. **Detects new services** — discovers and registers newly available plugins and scripts.
3. **Pulls and applies changes** — downloads updated configurations and services from the server.
4. **Restarts affected services** — gracefully restarts any services that received configuration updates.
5. **Reports status** — pushes health and status information back to the management server.
6. **Collects metrics** — gathers system and application metrics at regular intervals and forwards them for storage.

---

## 🗂️ Project Structure

```
agent/
├── main.go                     # Application entry point
├── agent.yml                   # Default configuration file
├── config/
│   ├── config.go               # Configuration structures
│   └── generated.go            # Build metadata (version, build date)
├── plugin/
│   ├── api/                    # REST API handlers
│   │   ├── init.go             # API route registration
│   │   ├── discover.go         # Elasticsearch node discovery
│   │   ├── log.go              # Log file access
│   │   └── model.go            # Request/response models
│   └── elastic/                # Elasticsearch integration
├── lib/
│   ├── process/                # OS process utilities
│   ├── reader/                 # Log file readers (plain text, gzip)
│   └── util/                   # Shared utility functions
└── docs/                       # Project documentation
```

---

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. **Fork** the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Commit your changes using [Conventional Commits](https://www.conventionalcommits.org/):
   ```
   feat: add new metric collector
   fix: resolve config reload race condition
   docs: update API reference
   ```
4. Push to your fork and open a **Pull Request**

Please make sure your PR:
- [ ] Has a descriptive title
- [ ] Includes necessary tests
- [ ] Does not introduce obvious performance regressions
- [ ] Updates relevant documentation

---

## 📄 License

Copyright © [INFINI Ltd](https://infinilabs.com). All rights reserved.

See the [LICENSE](LICENSE) file for details.

---

## 📬 Contact

- **Website:** [https://infinilabs.com](https://infinilabs.com)
- **Email:** [hello@infini.ltd](mailto:hello@infini.ltd)
- **Issues:** [GitHub Issues](https://github.com/infinilabs/agent/issues)

<div align="center">
  <sub>Built with ❤️ by the <a href="https://infinilabs.com">INFINI Labs</a> team.</sub>
</div>
