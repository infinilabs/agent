package api

import (
	"os"
	"path/filepath"
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
