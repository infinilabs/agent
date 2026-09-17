/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"infini.sh/agent/lib/process"
	readguard "infini.sh/agent/lib/util"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/env"
	log "infini.sh/framework/core/log"
	"infini.sh/framework/core/util"
)

// Whitelist for the Elasticsearch/Easysearch log viewing API
// (/elasticsearch/logs/_list and /elasticsearch/logs/_read). These
// endpoints receive logs_path from the caller, which used to allow
// reading any directory visible to the agent process. Reads are now
// confined to a whitelist resolved from three sources:
//
//  1. elasticsearch_logs.allowed_paths in the agent config — the escape
//     hatch for layouts where discovery cannot see the log directory
//  2. log directories the local search nodes report themselves
//     (settings path.logs, plus path.home/logs), discovered with the
//     same process scan the console gets its paths from
//  3. directories derived from the search processes' command lines
//     (-Des.path.logs, path.home/logs, and the -Xlog gc file location),
//     mirroring how the console derives the paths it sends back
//
// System paths (readguard.IsSystemReadPath) are never readable, even when
// whitelisted. The whitelist is cached for esLogDirsCacheTTL; a stale
// whitelist keeps serving while a refresh runs in the background.

const esLogDirsCacheTTL = time.Minute

// ESLogsConfig is the optional elasticsearch_logs config section.
type ESLogsConfig struct {
	AllowedPaths []string `config:"allowed_paths" json:"allowed_paths"`
}

var (
	esLogWhitelistMu         sync.Mutex
	esLogWhitelistGuard      *readguard.ReadGuard
	esLogWhitelistFetched    time.Time
	esLogWhitelistRefreshing atomic.Bool

	// esLogWhitelistLoader resolves the allowed roots; injectable in tests.
	esLogWhitelistLoader = defaultESLogWhitelist
)

// esLogsReadGuard returns the cached whitelist guard. Once a guard exists
// it is served even when stale while a single background refresh runs, so
// a slow or failed discovery never blocks log requests; the first caller
// (no cache yet) builds synchronously. When no whitelist can be
// established at all, access is denied (secure default).
func esLogsReadGuard() (*readguard.ReadGuard, error) {
	esLogWhitelistMu.Lock()
	guard := esLogWhitelistGuard
	if guard != nil {
		stale := time.Since(esLogWhitelistFetched) >= esLogDirsCacheTTL
		esLogWhitelistMu.Unlock()
		if !stale {
			return guard, nil
		}
		if esLogWhitelistRefreshing.CompareAndSwap(false, true) {
			go func() {
				defer esLogWhitelistRefreshing.Store(false)
				if _, err := refreshESLogWhitelist(); err != nil {
					log.Warnf("failed to refresh elasticsearch logs whitelist, keeping the previous one: %v", err)
				}
			}()
		}
		return guard, nil
	}
	esLogWhitelistMu.Unlock()
	return refreshESLogWhitelist()
}

func refreshESLogWhitelist() (*readguard.ReadGuard, error) {
	guard, err := buildESLogsReadGuard()
	esLogWhitelistMu.Lock()
	defer esLogWhitelistMu.Unlock()
	if err != nil {
		if esLogWhitelistGuard != nil {
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

func buildESLogsReadGuard() (*readguard.ReadGuard, error) {
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
	seen := map[string]bool{}
	for _, root := range roots {
		if seen[root] {
			continue
		}
		seen[root] = true
		if _, err := readguard.NewReadGuard(root); err != nil {
			log.Warnf("ignoring invalid elasticsearch logs path [%s]: %v", root, err)
			continue
		}
		valid = append(valid, root)
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("no valid elasticsearch log directories in %v", roots)
	}
	return readguard.NewReadGuard(valid...)
}

// defaultESLogWhitelist combines the static config section with the log
// directories discovered from the local search nodes and their command
// lines.
func defaultESLogWhitelist() ([]string, error) {
	return append(append(loadStaticESLogPaths(), discoverESLogDirs()...), cmdlineESLogDirs()...), nil
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

// nodeLogDirs extracts settings path.logs (string or array) plus
// path.home/logs, so the whitelist covers every location the console can
// derive from the node's settings.
func nodeLogDirs(info *elastic.NodesInfo) []string {
	if info == nil || len(info.Settings) == 0 {
		return nil
	}
	settings := util.MapStr(info.Settings)
	var dirs []string
	if v, err := settings.GetValue("path.logs"); err == nil {
		dirs = append(dirs, pathList(v)...)
	}
	if v, err := settings.GetValue("path.home"); err == nil {
		if home, err := util.ExtractString(v); err == nil && home != "" {
			dirs = append(dirs, filepath.Join(home, "logs"))
		}
	}
	return dirs
}

// cmdlineESLogDirs mirrors the console's deriveLogsPathsFromCmdline so
// that whatever directory the console computes from a process command
// line (-Des.path.logs, path.home/logs, -Xlog gc file dir) is in the
// whitelist before it can be requested back.
func cmdlineESLogDirs() []string {
	procs, err := process.DiscoverESProcessors(process.ElasticFilter)
	if err != nil {
		log.Warnf("failed to scan search processes for logs whitelist: %v", err)
		return nil
	}
	var dirs []string
	for _, p := range procs {
		dirs = append(dirs, cmdlineLogDirs(p.Cmdline)...)
	}
	return dirs
}

var (
	cmdlinePathHomeRe = regexp.MustCompile(`(?:^|\s)-D(?:es|opensearch)\.path\.home=([^\s]+)`)
	cmdlinePathLogsRe = regexp.MustCompile(`(?:^|\s)-D(?:es|opensearch)\.path\.logs=([^\s]+)`)
	cmdlineGCFileRe   = regexp.MustCompile(`(?:^|\s)-Xlog:[^\s]*?file=([^\s]+)`)
)

func cmdlineLogDirs(cmdline string) []string {
	pathHome := cmdlineValue(cmdlinePathHomeRe, cmdline)
	var dirs []string
	if v := cmdlineValue(cmdlinePathLogsRe, cmdline); v != "" {
		if r := resolveCmdlinePath(v, pathHome); r != "" {
			dirs = append(dirs, r)
		}
	} else if pathHome != "" {
		dirs = append(dirs, filepath.Join(pathHome, "logs"))
	}
	if v := trimGCLogFileValue(cmdlineValue(cmdlineGCFileRe, cmdline)); v != "" {
		if r := resolveCmdlinePath(v, pathHome); r != "" {
			dirs = append(dirs, filepath.Dir(r))
		}
	}
	return dirs
}

func cmdlineValue(re *regexp.Regexp, cmdline string) string {
	matches := re.FindStringSubmatch(cmdline)
	if len(matches) > 1 {
		return strings.Trim(strings.TrimSpace(matches[1]), `"'`)
	}
	return ""
}

// trimGCLogFileValue drops the rotation tags after the file name, e.g.
// /var/log/gc.log:uptime,tags -> /var/log/gc.log (drive letters kept).
func trimGCLogFileValue(value string) string {
	searchFrom := 0
	if len(value) > 1 && value[1] == ':' {
		searchFrom = 2
	}
	if idx := strings.Index(value[searchFrom:], ":"); idx >= 0 {
		value = value[:searchFrom+idx]
	}
	return value
}

func resolveCmdlinePath(value, base string) string {
	if value == "" {
		return ""
	}
	if !filepath.IsAbs(value) {
		if base == "" {
			return ""
		}
		value = filepath.Join(base, value)
	}
	return filepath.Clean(value)
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
