/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"errors"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckPortAvailable(t *testing.T) {
	t.Run("port occupied", func(t *testing.T) {
		listener, err := net.Listen("tcp", ":0")
		require.NoError(t, err)
		defer listener.Close()

		port := listener.Addr().(*net.TCPAddr).Port
		available, err := checkPortAvailable(port)
		require.NoError(t, err)
		assert.False(t, available)
	})

	t.Run("port available", func(t *testing.T) {
		itWorks := false
		maxTries := 10

		// It is possible that port will be acquired by other processes
		// between line a and b, but it is rare. We test this 10 times,
		// as long as it works once, the test passes.
		for _i := 0; _i < maxTries; _i += 1 {
			port := reserveFreeTCPPort(t) // line a

			available, err := checkPortAvailable(port) // line b
			require.NoError(t, err)
			if available {
				itWorks = true
				break
			}
		}

		if !itWorks {
			// fail the test, either there is a bug or we are not lucky.
			// If you think it failed due to bad luck, increase maxTries.
			assert.Fail(t, "")
		}
	})
}

func TestCheckDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip the test on Windows as it is tested using POSIX stuff")
	}
	if currentUser, err := user.Current(); err == nil && currentUser.Uid == "0" {
		t.Skip("permission checks are unreliable when running as root")
	}

	baseDir := t.TempDir()

	existingWritableDir := filepath.Join(baseDir, "existing-dir")
	require.NoError(t, os.Mkdir(existingWritableDir, 0755))

	nonDirectoryFile := filepath.Join(baseDir, "not-a-dir")
	require.NoError(t, os.WriteFile(nonDirectoryFile, []byte("test"), 0644))

	writableMissingDir := filepath.Join(baseDir, "missing", "child")

	readOnlyParent := filepath.Join(baseDir, "readonly-parent")
	require.NoError(t, os.Mkdir(readOnlyParent, 0755))
	makeDirReadOnlyForTest(t, readOnlyParent)
	nonWritableMissingDir := filepath.Join(readOnlyParent, "child")

	tests := []struct {
		name            string
		path            string
		expectAvailable bool
		expectErr       error
		assertResponse  func(t *testing.T, result CheckDirectoryResponse)
	}{
		{
			name:            "missing writable directory",
			path:            writableMissingDir,
			expectAvailable: true,
			assertResponse: func(t *testing.T, result CheckDirectoryResponse) {
				assert.NotEmpty(t, result.MountPoint)
			},
		},
		{
			name:            "missing non writable directory",
			path:            nonWritableMissingDir,
			expectAvailable: false,
		},
		{
			name:            "non directory path",
			path:            nonDirectoryFile,
			expectAvailable: false,
		},
		{
			name:            "existing writable directory",
			path:            existingWritableDir,
			expectAvailable: true,
			assertResponse: func(t *testing.T, result CheckDirectoryResponse) {
				assert.NotEmpty(t, result.MountPoint)
			},
		},
		{
			name:            "relative path rejected",
			path:            filepath.Join("relative", "path"),
			expectErr:       errPathNotAbsolute,
			expectAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckDirectory(tt.path)
			if tt.expectErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectErr))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectAvailable, result.Available)
			if tt.assertResponse != nil {
				tt.assertResponse(t, result)
			}
		})
	}
}

func TestCheckDirectorySymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip the test on Windows as symlink behavior differs on Windows")
	}
	if currentUser, err := user.Current(); err == nil && currentUser.Uid == "0" {
		t.Skip("permission checks are unreliable when running as root")
	}

	baseDir := t.TempDir()

	existingWritableDir := filepath.Join(baseDir, "existing-dir")
	require.NoError(t, os.Mkdir(existingWritableDir, 0755))
	existingDirSymlink := filepath.Join(baseDir, "dir-link")
	createSymlink(t, existingWritableDir, existingDirSymlink)

	nonDirectoryFile := filepath.Join(baseDir, "not-a-dir")
	require.NoError(t, os.WriteFile(nonDirectoryFile, []byte("test"), 0644))
	nonDirectorySymlink := filepath.Join(baseDir, "file-link")
	createSymlink(t, nonDirectoryFile, nonDirectorySymlink)

	writableMissingDir := filepath.Join(baseDir, "missing", "child")
	writableMissingDirSymlink := filepath.Join(baseDir, "writable_missing_dir")
	createSymlink(t, writableMissingDir, writableMissingDirSymlink)

	readOnlyParent := filepath.Join(baseDir, "readonly-parent")
	require.NoError(t, os.Mkdir(readOnlyParent, 0755))
	nonWritableMissingTarget := filepath.Join(readOnlyParent, "child")
	nonWritableMissingSymlink := filepath.Join(readOnlyParent, "missing-link")
	createSymlink(t, nonWritableMissingTarget, nonWritableMissingSymlink)
	makeDirReadOnlyForTest(t, readOnlyParent)

	tests := []struct {
		name            string
		path            string
		expectAvailable bool
	}{
		{name: "symlink to existing writable directory", path: existingDirSymlink, expectAvailable: true},
		{name: "symlink to file", path: nonDirectorySymlink, expectAvailable: false},
		{name: "symlink to missing writable directory", path: writableMissingDirSymlink, expectAvailable: true},
		{name: "symlink to missing non writable directory", path: nonWritableMissingSymlink, expectAvailable: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckDirectory(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.expectAvailable, result.Available)
		})
	}
}

func makeDirReadOnlyForTest(t *testing.T, path string) {
	t.Helper()

	require.NoError(t, os.Chmod(path, 0555))
	t.Cleanup(func() {
		_ = os.Chmod(path, 0755)
	})
}

func createSymlink(t *testing.T, target, link string) {
	t.Helper()

	require.NoError(t, os.Symlink(target, link))
}

func reserveFreeTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	return port
}
