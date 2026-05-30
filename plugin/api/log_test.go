package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestSafeJoinLogsFileRejectsTraversal(t *testing.T) {
	path, err := safeJoinLogsFile("/var/log/elasticsearch", "server.log")
	if err != nil || path != "/var/log/elasticsearch/server.log" {
		t.Fatalf("expected normal path, got path=%q err=%v", path, err)
	}

	if _, err := safeJoinLogsFile("/var/log/elasticsearch", "../server.log"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestReadSearchLogLinesUsesRealOffset(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "server.log")
	content := "first\r\nsecond\r\nthird"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp log file: %v", err)
	}

	lines, isEOF, err := readSearchLogLines(filePath, 1, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isEOF {
		t.Fatal("expected more content after first page")
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %#v", lines)
	}
	if lines[0]["offset"] != int64(7) || lines[1]["offset"] != int64(15) {
		t.Fatalf("unexpected offsets: %#v", lines)
	}

	lines, isEOF, err = readSearchLogLines(filePath, 3, lines[1]["offset"].(int64), 2)
	if err != nil {
		t.Fatalf("unexpected error on second page: %v", err)
	}
	if !isEOF {
		t.Fatal("expected EOF on second page")
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %#v", lines)
	}
	if lines[0]["line_number"] != int64(3) || lines[0]["content"] != "third" {
		t.Fatalf("unexpected second page data: %#v", lines[0])
	}
}

func TestReadSearchLogLinesKeepsLineNumbersUnknownAfterTailMode(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "server.log")
	content := "first\nsecond\nthird\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp log file: %v", err)
	}

	lines, _, err := readSearchLogLines(filePath, 0, 6, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %#v", lines)
	}
	if _, ok := lines[0]["line_number"]; ok {
		t.Fatalf("line number should stay unknown after offset-only reads: %#v", lines[0])
	}
}

func TestReadSearchLogTailLinesReturnsLatestOffsets(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "server.log")
	content := "first\nsecond\nthird\nfourth\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp log file: %v", err)
	}

	lines, err := readSearchLogTailLines(filePath, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %#v", lines)
	}
	if lines[0]["content"] != "third" || lines[1]["content"] != "fourth" {
		t.Fatalf("unexpected tail lines: %#v", lines)
	}
	if lines[0]["offset"] != int64(19) || lines[1]["offset"] != int64(26) {
		t.Fatalf("unexpected tail offsets: %#v", lines)
	}
	if _, ok := lines[0]["line_number"]; ok {
		t.Fatalf("tail reads should not claim absolute line numbers: %#v", lines[0])
	}
}

func TestShouldCountLogRows(t *testing.T) {
	if !shouldCountLogRows(maxCountRowsBytes) {
		t.Fatal("expected row counts for small files")
	}
	if shouldCountLogRows(maxCountRowsBytes + 1) {
		t.Fatal("expected row counts to be skipped for large files")
	}
}

func TestGetSearchLogFilesPreservesRequestedLogsPath(t *testing.T) {
	root := t.TempDir()
	realLogsDir := filepath.Join(root, "real", "logs")
	if err := os.MkdirAll(realLogsDir, 0o755); err != nil {
		t.Fatalf("failed to create real logs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(realLogsDir, "server.log"), []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatalf("failed to create temp log file: %v", err)
	}

	linkParent := filepath.Join(root, "linked")
	if err := os.MkdirAll(linkParent, 0o755); err != nil {
		t.Fatalf("failed to create link parent dir: %v", err)
	}
	linkLogsDir := filepath.Join(linkParent, "logs")
	if err := os.Symlink(realLogsDir, linkLogsDir); err != nil {
		t.Skipf("symlink unsupported in current environment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/elasticsearch/logs/_list", strings.NewReader(`{"logs_path":"`+linkLogsDir+`"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	(&AgentAPI{}).getSearchLogFiles(recorder, req, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Success bool                     `json:"success"`
		Result  []map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %s", recorder.Body.String())
	}
	if len(resp.Result) != 1 {
		t.Fatalf("expected 1 log file, got %#v", resp.Result)
	}
	if got := resp.Result[0]["logs_path"]; got != linkLogsDir {
		t.Fatalf("expected requested logs path %q, got %#v", linkLogsDir, got)
	}
}
