package api

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

func TestNormalizeJSONLogsPathsSupportsStringAndArray(t *testing.T) {
	paths := normalizeJSONLogsPaths("/var/log/elasticsearch")
	if len(paths) != 1 || paths[0] != "/var/log/elasticsearch" {
		t.Fatalf("expected single path, got %#v", paths)
	}

	paths = normalizeJSONLogsPaths([]interface{}{"/var/log/elasticsearch", "/var/log/elasticsearch/gc", "/var/log/elasticsearch"})
	if len(paths) != 2 || paths[0] != "/var/log/elasticsearch" || paths[1] != "/var/log/elasticsearch/gc" {
		t.Fatalf("expected deduplicated paths, got %#v", paths)
	}
}

func withESLogWhitelist(t *testing.T, dirs ...string) {
	t.Helper()
	orig := esLogWhitelistLoader
	esLogWhitelistLoader = func() ([]string, error) { return dirs, nil }
	resetESLogWhitelistCache()
	t.Cleanup(func() {
		esLogWhitelistLoader = orig
		resetESLogWhitelistCache()
	})
}

func TestGetSearchLogFilesRejectsPathOutsideWhitelist(t *testing.T) {
	dir := t.TempDir()
	withESLogWhitelist(t, dir)

	handler := AgentAPI{}
	req := httptest.NewRequest("POST", "/elasticsearch/logs/_list", strings.NewReader(`{"logs_path":"/etc"}`))
	w := httptest.NewRecorder()
	handler.getSearchLogFiles(w, req, httprouter.Params{})
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetSearchLogFilesEmptyWhitelistDenied(t *testing.T) {
	withESLogWhitelist(t)

	handler := AgentAPI{}
	req := httptest.NewRequest("POST", "/elasticsearch/logs/_list", strings.NewReader(`{"logs_path":"/var/log/elasticsearch"}`))
	w := httptest.NewRecorder()
	handler.getSearchLogFiles(w, req, httprouter.Params{})
	if w.Code != 403 {
		t.Fatalf("expected 403 when no whitelist can be established, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetSearchLogFilesListsAllowedPath(t *testing.T) {
	dir := t.TempDir()
	withESLogWhitelist(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "server.log"), []byte("line1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	handler := AgentAPI{}
	body := `{"logs_path":` + quoteJSON(dir) + `}`
	req := httptest.NewRequest("POST", "/elasticsearch/logs/_list", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.getSearchLogFiles(w, req, httprouter.Params{})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := w.Body.String()
	if !strings.Contains(resp, `"success":true`) || !strings.Contains(resp, "server.log") {
		t.Errorf("expected server.log in successful listing: %s", resp)
	}
	if !strings.Contains(resp, "notes.txt") {
		t.Errorf("expected listing to keep every regular file: %s", resp)
	}
}

func TestReadSearchLogFileWhitelistAndTraversal(t *testing.T) {
	dir := t.TempDir()
	withESLogWhitelist(t, dir)

	logFile := filepath.Join(dir, "server.log")
	if err := os.WriteFile(logFile, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	handler := AgentAPI{}

	// base outside the whitelist
	req := httptest.NewRequest("POST", "/elasticsearch/logs/_read",
		strings.NewReader(`{"logs_path":"/etc","file_name":"passwd","lines":10}`))
	w := httptest.NewRecorder()
	handler.readSearchLogFile(w, req, httprouter.Params{})
	if w.Code != 403 {
		t.Fatalf("expected 403 for logs_path outside whitelist, got %d: %s", w.Code, w.Body.String())
	}

	// traversal within an allowed base
	req = httptest.NewRequest("POST", "/elasticsearch/logs/_read",
		strings.NewReader(`{"logs_path":`+quoteJSON(dir)+`,"file_name":"../secret.log","lines":10}`))
	w = httptest.NewRecorder()
	handler.readSearchLogFile(w, req, httprouter.Params{})
	if w.Code != 400 {
		t.Fatalf("expected 400 for traversal, got %d: %s", w.Code, w.Body.String())
	}

	// symlink escape out of the allowed base
	link := filepath.Join(dir, "escape.log")
	if err := os.Symlink("/etc/passwd", link); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest("POST", "/elasticsearch/logs/_read",
		strings.NewReader(`{"logs_path":`+quoteJSON(dir)+`,"file_name":"escape.log","lines":10}`))
	w = httptest.NewRecorder()
	handler.readSearchLogFile(w, req, httprouter.Params{})
	if w.Code != 400 {
		t.Fatalf("expected 400 for symlink escape, got %d: %s", w.Code, w.Body.String())
	}

	// normal read inside the whitelist
	req = httptest.NewRequest("POST", "/elasticsearch/logs/_read",
		strings.NewReader(`{"logs_path":`+quoteJSON(dir)+`,"file_name":"server.log","lines":10,"start_line_number":1}`))
	w = httptest.NewRecorder()
	handler.readSearchLogFile(w, req, httprouter.Params{})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "line2") {
		t.Errorf("expected file content in response: %s", w.Body.String())
	}
}

func TestNodeLogDirs(t *testing.T) {
	// path.logs as string (home/logs also kept as a candidate)
	info := &elastic.NodesInfo{Settings: map[string]interface{}{
		"path": map[string]interface{}{
			"logs": "/var/log/easysearch",
			"home": "/usr/share/easysearch",
		},
	}}
	if dirs := nodeLogDirs(info); len(dirs) != 2 || dirs[0] != "/var/log/easysearch" || dirs[1] != filepath.Join("/usr/share/easysearch", "logs") {
		t.Fatalf("expected logs and home/logs dirs, got %#v", dirs)
	}

	// path.logs as array (multi-path)
	info.Settings = map[string]interface{}{
		"path": map[string]interface{}{"logs": []interface{}{"/data1/logs", "/data2/logs"}},
	}
	if dirs := nodeLogDirs(info); len(dirs) != 2 {
		t.Fatalf("expected multi-path logs dirs, got %#v", dirs)
	}

	// fallback to path.home/logs
	info.Settings = map[string]interface{}{
		"path": map[string]interface{}{"home": "/usr/share/easysearch"},
	}
	if dirs := nodeLogDirs(info); len(dirs) != 1 || dirs[0] != filepath.Join("/usr/share/easysearch", "logs") {
		t.Fatalf("expected home/logs fallback, got %#v", dirs)
	}

	if dirs := nodeLogDirs(nil); dirs != nil {
		t.Fatalf("expected no dirs for nil node info, got %#v", dirs)
	}
}

func TestGetSearchLogFilesAllowsSubdirOfWhitelistedRoot(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "gc")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	withESLogWhitelist(t, dir)

	if err := os.WriteFile(filepath.Join(sub, "gc.log"), []byte("gc\n"), 0644); err != nil {
		t.Fatal(err)
	}

	handler := AgentAPI{}
	req := httptest.NewRequest("POST", "/elasticsearch/logs/_list", strings.NewReader(`{"logs_path":`+quoteJSON(sub)+`}`))
	w := httptest.NewRecorder()
	handler.getSearchLogFiles(w, req, httprouter.Params{})
	if w.Code != 200 {
		t.Fatalf("expected 200 for subdirectory of whitelisted root, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "gc.log") {
		t.Errorf("expected gc.log in listing: %s", w.Body.String())
	}
}

func TestCmdlineLogDirs(t *testing.T) {
	// -Des.path.logs override
	dirs := cmdlineLogDirs(`java -Des.path.home=/usr/share/easysearch -Des.path.logs=/var/log/easysearch -Xlog:gc*:file=/var/log/easysearch/gc.log:uptime,tags`)
	if len(dirs) != 2 || dirs[0] != "/var/log/easysearch" || dirs[1] != "/var/log/easysearch" {
		t.Fatalf("expected cmdline logs dir and gc dir, got %#v", dirs)
	}

	// gc file outside path.logs must be whitelisted too
	dirs = cmdlineLogDirs(`java -Des.path.home=/usr/share/easysearch -Des.path.logs=/var/log/easysearch -Xlog:gc*:file=/var/log/gc/es_gc.log:uptime`)
	if len(dirs) != 2 || dirs[1] != "/var/log/gc" {
		t.Fatalf("expected gc dir outside path.logs, got %#v", dirs)
	}

	// no explicit path.logs: derive home/logs
	dirs = cmdlineLogDirs(`java -Des.path.home=/opt/easysearch`)
	if len(dirs) != 1 || dirs[0] != filepath.Join("/opt/easysearch", "logs") {
		t.Fatalf("expected home/logs dir, got %#v", dirs)
	}

	// quoted relative gc file resolved against home
	dirs = cmdlineLogDirs(`java -Des.path.home=/opt/es -Xlog:file="logs/gc.log"`)
	if len(dirs) != 2 || dirs[1] != filepath.Join("/opt/es", "logs") {
		t.Fatalf("expected relative gc file joined on home, got %#v", dirs)
	}
}

func TestBuildESLogsReadGuardSkipsInvalidRoots(t *testing.T) {
	dir := t.TempDir()
	orig := esLogWhitelistLoader
	esLogWhitelistLoader = func() ([]string, error) {
		return []string{"/etc", filepath.Join(dir, "missing"), dir}, nil
	}
	resetESLogWhitelistCache()
	defer func() {
		esLogWhitelistLoader = orig
		resetESLogWhitelistCache()
	}()

	guard, err := esLogsReadGuard()
	if err != nil {
		t.Fatalf("expected valid roots to survive: %v", err)
	}
	if !guard.Contains(dir) {
		t.Error("expected the valid root to be whitelisted")
	}
	if guard.Contains("/etc") {
		t.Error("expected system path to be excluded from the whitelist")
	}
}

func TestReadSearchLogFileResponseShape(t *testing.T) {
	dir := t.TempDir()
	withESLogWhitelist(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "server.log"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	handler := AgentAPI{}
	var buf bytes.Buffer
	buf.WriteString(`{"logs_path":"`)
	buf.WriteString(strings.ReplaceAll(dir, `\`, `\\`))
	buf.WriteString(`","file_name":"server.log","lines":5,"start_line_number":0}`)
	req := httptest.NewRequest("POST", "/elasticsearch/logs/_read", &buf)
	w := httptest.NewRecorder()
	handler.readSearchLogFile(w, req, httprouter.Params{})
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := util.MapStr{}
	if err := util.FromJSONBytes(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["success"] != true {
		t.Errorf("expected success=true: %s", w.Body.String())
	}
}

func quoteJSON(dir string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(dir, `\`, `\\`), `"`, `\"`) + `"`
}
