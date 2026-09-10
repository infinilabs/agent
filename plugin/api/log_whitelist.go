/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"infini.sh/agent/lib/process"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	log "infini.sh/framework/core/log"
	"infini.sh/framework/core/util"
)

// Whitelist for the Elasticsearch/Easysearch log viewing API
// (/elasticsearch/logs/_list and /elasticsearch/logs/_read). These
// endpoints receive logs_path from the caller, which used to allow
// reading any directory visible to the agent process. Reads are now
// confined to a whitelist resolved from two sources:
//
//  1. elasticsearch_logs.allowed_paths in the agent config — the escape
//     hatch for layouts where discovery cannot see the log directory
//  2. log directories the local search nodes report themselves
//     (settings path.logs, falling back to path.home/logs), discovered
//     with the same process scan the console gets its paths from
//
// System paths (util.IsSystemReadPath) are never readable, even when
// whitelisted. The whitelist is cached for esLogDirsCacheTTL; config
// changes take effect within that window or after restart.

const esLogDirsCacheTTL = time.Minute

// ESLogsConfig is the optional elasticsearch_logs config section.
type ESLogsConfig struct {
	AllowedPaths []string `config:"allowed_paths" json:"allowed_paths"`
}

var (
	esLogWhitelistMu      sync.Mutex
	esLogWhitelistGuard   *util.ReadGuard
	esLogWhitelistFetched time.Time

	// esLogWhitelistLoader resolves the allowed roots; injectable in tests.
	esLogWhitelistLoader = defaultESLogWhitelist
)

// esLogsReadGuard returns the cached whitelist guard, refreshing it when
// stale. A failed refresh keeps the previous whitelist serving; when no
// whitelist can be established at all, access is denied (secure default).
func esLogsReadGuard() (*util.ReadGuard, error) {
	esLogWhitelistMu.Lock()
	defer esLogWhitelistMu.Unlock()
	if esLogWhitelistGuard != nil && time.Since(esLogWhitelistFetched) < esLogDirsCacheTTL {
		return esLogWhitelistGuard, nil
	}
	guard, err := buildESLogsReadGuard()
	if err != nil {
		if esLogWhitelistGuard != nil {
			log.Warnf("failed to refresh elasticsearch logs whitelist, keeping the previous one: %v", err)
			return esLogWhitelistGuard, nil
		}
		return nil, err
	}
	esLogWhitelistGuard = guard
	esLogWhitelistFetched = time.Now()
	return guard, nil
}

func resetESLogWhitelistCache() {
	esLogWhitelistMu.Lock()
	defer esLogWhitelistMu.Unlock()
	esLogWhitelistGuard = nil
}

func buildESLogsReadGuard() (*util.ReadGuard, error) {
	roots, err := esLogWhitelistLoader()
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("no allowed elasticsearch log directories: configure elasticsearch_logs.allowed_paths or make sure the local search node is discoverable")
	}
	// validate roots one by one: a stale or misconfigured entry must not
	// take the whole whitelist down
	var valid []string
	for _, root := range roots {
		if _, err := util.NewReadGuard(root); err != nil {
			log.Warnf("ignoring invalid elasticsearch logs path [%s]: %v", root, err)
			continue
		}
		valid = append(valid, root)
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("no valid elasticsearch log directories in %v", roots)
	}
	return util.NewReadGuard(valid...)
}

// defaultESLogWhitelist combines the static config section with the log
// directories discovered from the local search nodes.
func defaultESLogWhitelist() ([]string, error) {
	return append(loadStaticESLogPaths(), discoverESLogDirs()...), nil
}

// loadStaticESLogPaths reads elasticsearch_logs.allowed_paths from the
// app config file (main file + config dir), falling back to the global
// env config like the discovery endpoints do.
func loadStaticESLogPaths() []string {
	cfg := ESLogsConfig{}
	var err error
	appCfg, cfgErr := getAppConfig()
	if cfgErr != nil {
		_, err = env.ParseConfig("elasticsearch_logs", &cfg)
	} else {
		_, err = env.ParseConfigSection(appCfg, "elasticsearch_logs", &cfg)
	}
	if err != nil {
		log.Debugf("no elasticsearch_logs config: %v", err)
		return nil
	}
	return cfg.AllowedPaths
}

// discoverESLogDirs runs the same local process scan the console-facing
// discovery endpoint uses, and extracts each node's reported log dir.
func discoverESLogDirs() []string {
	result, err := process.DiscoverESNode(nil)
	if err != nil {
		log.Warnf("failed to discover local search nodes for logs whitelist: %v", err)
		return nil
	}
	var dirs []string
	for _, node := range result.Nodes {
		dirs = append(dirs, nodeLogDirs(node.NodeInfo)...)
	}
	return dirs
}

// nodeLogDirs extracts settings path.logs (string or array), falling back
// to path.home/logs when the node does not report a logs path.
func nodeLogDirs(info *elastic.NodesInfo) []string {
	if info == nil || len(info.Settings) == 0 {
		return nil
	}
	settings := util.MapStr(info.Settings)
	var dirs []string
	if v, err := settings.GetValue("path.logs"); err == nil {
		dirs = append(dirs, pathList(v)...)
	}
	if len(dirs) == 0 {
		if v, err := settings.GetValue("path.home"); err == nil {
			if home, err := util.ExtractString(v); err == nil && home != "" {
				dirs = append(dirs, filepath.Join(home, "logs"))
			}
		}
	}
	return dirs
}

func pathList(raw interface{}) []string {
	switch v := raw.(type) {
	case string:
		if v != "" {
			return []string{v}
		}
	case []string:
		return v
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			if s := util.ToString(item); s != "" {
				items = append(items, s)
			}
		}
		return items
	}
	return nil
}
