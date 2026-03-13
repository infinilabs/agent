/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveRootDirName(t *testing.T) {
	assert.Equal(t, "easysearch-2.0.2-2499-mac-arm64", archiveRootDirName("/tmp/easysearch-2.0.2-2499-mac-arm64.zip"))
	assert.Equal(t, "easysearch-2.0.2-linux-amd64", archiveRootDirName("/tmp/easysearch-2.0.2-linux-amd64.tar.gz"))
	assert.Equal(t, "jdk-21", archiveRootDirName("/tmp/jdk-21.tgz"))
}

func TestExtractZipWrapsFlatArchiveIntoArchiveNamedDirectory(t *testing.T) {
	workspace := t.TempDir()
	archivePath := filepath.Join(workspace, "easysearch-2.0.2-2499-mac-arm64.zip")
	require.NoError(t, createTestZip(archivePath, map[string]string{
		"bin/easysearch":              "#!/bin/sh",
		"config/easysearch.yml":       "cluster.name: test",
		"modules/analysis/readme.txt": "ok",
	}))

	rootDir, err := extractArchive(context.Background(), archivePath, workspace, archiveLayoutWrapInArchiveDir)
	require.NoError(t, err)
	assert.Equal(t, "easysearch-2.0.2-2499-mac-arm64", rootDir)

	assert.FileExists(t, filepath.Join(workspace, rootDir, "bin", "easysearch"))
	assert.FileExists(t, filepath.Join(workspace, rootDir, "config", "easysearch.yml"))
	assert.FileExists(t, filepath.Join(workspace, rootDir, "modules", "analysis", "readme.txt"))
}

func TestExtractZipPreservesExistingSingleTopLevelDirectory(t *testing.T) {
	workspace := t.TempDir()
	archivePath := filepath.Join(workspace, "easysearch-2.0.2-2499-mac-arm64.zip")
	require.NoError(t, createTestZip(archivePath, map[string]string{
		"easysearch-2.0.2-2499-mac-arm64/bin/easysearch":        "#!/bin/sh",
		"easysearch-2.0.2-2499-mac-arm64/config/easysearch.yml": "cluster.name: test",
	}))

	rootDir, err := extractArchive(context.Background(), archivePath, workspace, archiveLayoutExpectTopLevelDir)
	require.NoError(t, err)
	assert.Equal(t, "easysearch-2.0.2-2499-mac-arm64", rootDir)

	assert.FileExists(t, filepath.Join(workspace, rootDir, "bin", "easysearch"))
	assert.FileExists(t, filepath.Join(workspace, rootDir, "config", "easysearch.yml"))
}

func TestExtractZipWithTopLevelDirStillWrapsForEasysearchLayout(t *testing.T) {
	workspace := t.TempDir()
	archivePath := filepath.Join(workspace, "easysearch-2.0.2-2499-mac-arm64.zip")
	require.NoError(t, createTestZip(archivePath, map[string]string{
		"easysearch-2.0.2-2499-mac-arm64/bin/easysearch":        "#!/bin/sh",
		"easysearch-2.0.2-2499-mac-arm64/config/easysearch.yml": "cluster.name: test",
	}))

	rootDir, err := extractArchive(context.Background(), archivePath, workspace, archiveLayoutWrapInArchiveDir)
	require.NoError(t, err)
	assert.Equal(t, "easysearch-2.0.2-2499-mac-arm64", rootDir)

	assert.FileExists(t, filepath.Join(workspace, rootDir, "easysearch-2.0.2-2499-mac-arm64", "bin", "easysearch"))
}

func createTestZip(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			_ = zw.Close()
			return err
		}
	}
	return zw.Close()
}
