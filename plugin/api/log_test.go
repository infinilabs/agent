package api

import "testing"

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
