package logging

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	config2 "infini.sh/framework/core/config"
)

func TestFixLogPath(t *testing.T) {
	v := fixLogPath(`C:/logs/246`)
	assert.Equal(t, v, `C:\logs\246`)
}

func TestNormalizeLogsPathsSupportsStringAndArray(t *testing.T) {
	assert.Equal(t,
		[]string{filepath.Clean("/var/log/elasticsearch")},
		normalizeLogsPaths("/var/log/elasticsearch", ""),
	)

	assert.Equal(t,
		[]string{filepath.Clean("/var/log/elasticsearch"), filepath.Clean("/var/log/elasticsearch/gc")},
		normalizeLogsPaths([]interface{}{"/var/log/elasticsearch", "/var/log/elasticsearch/gc", "/var/log/elasticsearch"}, ""),
	)
}

func TestEsLogsConfigUnpackSupportsScalarAndArrayLogsPath(t *testing.T) {
	cfg, err := config2.NewConfigWithYAML([]byte("queue_name: logs\nelasticsearch: cluster-1\nlogs_path: /var/log/elasticsearch\n"), "scalar.yml")
	assert.NoError(t, err)

	processor, err := New(cfg)
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{filepath.Clean("/var/log/elasticsearch")},
		normalizeLogsPaths(processor.(*EsLogsProcessor).cfg.LogsPath, ""),
	)

	cfg, err = config2.NewConfigWithYAML([]byte("queue_name: logs\nelasticsearch: cluster-1\nlogs_path: [\"/var/log/elasticsearch\", \"/var/log/elasticsearch/gc\"]\n"), "array.yml")
	assert.NoError(t, err)

	processor, err = New(cfg)
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{filepath.Clean("/var/log/elasticsearch"), filepath.Clean("/var/log/elasticsearch/gc")},
		normalizeLogsPaths(processor.(*EsLogsProcessor).cfg.LogsPath, ""),
	)
}
