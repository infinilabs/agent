/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type archiveLayout int

const (
	archiveLayoutExpectTopLevelDir archiveLayout = iota
	archiveLayoutWrapInArchiveDir
)

// httpDownload downloads a file from url to destPath, honouring ctx
// cancellation. It writes to a temporary file first, then renames
// atomically on success.
func httpDownload(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	dir := filepath.Dir(destPath)
	tmpFile, err := os.CreateTemp(dir, ".download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Stream with periodic context checks.
	buf := make([]byte, 32*1024)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("write: %w", writeErr)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read body: %w", readErr)
		}
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	success = true
	return nil
}

// extractArchive extracts a tar.gz or zip archive into destDir.
//
// For `archiveLayoutExpectTopLevelDir`, the archive itself must contain a
// single top-level directory and that directory name is returned.
//
// For `archiveLayoutWrapInArchiveDir`, the extracted contents are always
// placed under a synthetic root directory named after the archive file, and
// that synthetic root directory name is returned.
func extractArchive(ctx context.Context, archivePath, destDir string, layout archiveLayout) (string, error) {
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(ctx, archivePath, destDir, layout)
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(ctx, archivePath, destDir, layout)
	default:
		return "", fmt.Errorf("unsupported archive format: %s", filepath.Base(archivePath))
	}
}

func archiveRootDirName(archivePath string) string {
	name := filepath.Base(archivePath)
	lower := strings.ToLower(name)

	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		return name[:len(name)-len(".tar.gz")]
	case strings.HasSuffix(lower, ".tgz"):
		return name[:len(name)-len(".tgz")]
	case strings.HasSuffix(lower, ".zip"):
		return name[:len(name)-len(".zip")]
	default:
		return strings.TrimSuffix(name, filepath.Ext(name))
	}
}

// extractTarGz extracts a .tar.gz archive into destDir.
func extractTarGz(ctx context.Context, archivePath, destDir string, layout archiveLayout) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	topDirs := make(map[string]struct{})
	cleanDest := filepath.Clean(destDir)
	rootDir := archiveRootDirName(archivePath)

	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read entry: %w", err)
		}

		// Track top-level directory.
		parts := strings.SplitN(filepath.ToSlash(hdr.Name), "/", 2)
		if len(parts) > 0 && parts[0] != "." && parts[0] != "" {
			topDirs[parts[0]] = struct{}{}
		}

		targetName := hdr.Name
		if layout == archiveLayoutWrapInArchiveDir {
			targetName = filepath.Join(rootDir, hdr.Name)
		}

		target := filepath.Join(destDir, targetName)
		// Prevent path traversal (zip-slip).
		if !isInsideDir(target, cleanDest) {
			return "", fmt.Errorf("illegal path in archive: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)|0755); err != nil {
				return "", fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", fmt.Errorf("mkdir parent: %w", err)
			}
			if err := writeFile(tr, target, os.FileMode(hdr.Mode)); err != nil {
				return "", err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", fmt.Errorf("mkdir parent: %w", err)
			}
			os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return "", fmt.Errorf("symlink %s: %w", target, err)
			}
		case tar.TypeLink:
			linkTarget := filepath.Join(destDir, hdr.Linkname)
			if err := os.Link(linkTarget, target); err != nil {
				return "", fmt.Errorf("link %s: %w", target, err)
			}
		}
	}

	if layout == archiveLayoutWrapInArchiveDir {
		return rootDir, nil
	}

	return singleTopDir(topDirs)
}

// extractZip extracts a .zip archive into destDir.
func extractZip(ctx context.Context, archivePath, destDir string, layout archiveLayout) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	topDirs := make(map[string]struct{})
	rootDir := archiveRootDirName(archivePath)
	cleanDest := filepath.Clean(destDir)

	for _, f := range r.File {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		parts := strings.SplitN(filepath.ToSlash(f.Name), "/", 2)
		if len(parts) > 0 && parts[0] != "." && parts[0] != "" {
			topDirs[parts[0]] = struct{}{}
		}

		targetName := f.Name
		if layout == archiveLayoutWrapInArchiveDir {
			targetName = filepath.Join(rootDir, f.Name)
		}

		target := filepath.Join(destDir, targetName)
		if !isInsideDir(target, cleanDest) {
			return "", fmt.Errorf("illegal path in zip: %s", f.Name)
		}

		// Symlink (Unix mode bits in external attributes).
		if f.Mode()&os.ModeSymlink != 0 {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("open symlink entry %s: %w", f.Name, err)
			}
			linkTarget, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("read symlink %s: %w", f.Name, err)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", fmt.Errorf("mkdir parent: %w", err)
			}
			os.Remove(target)
			if err := os.Symlink(string(linkTarget), target); err != nil {
				return "", fmt.Errorf("symlink %s: %w", target, err)
			}
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()|0755); err != nil {
				return "", fmt.Errorf("mkdir %s: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return "", fmt.Errorf("mkdir parent: %w", err)
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}
		err = writeFile(rc, target, f.Mode())
		rc.Close()
		if err != nil {
			return "", err
		}
	}

	if layout == archiveLayoutWrapInArchiveDir {
		return rootDir, nil
	}

	return singleTopDir(topDirs)
}

// writeFile writes the contents of r into a new file at path.
func writeFile(r io.Reader, path string, mode os.FileMode) error {
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	return out.Close()
}

// isInsideDir checks whether target is under (or equal to) dir.
func isInsideDir(target, dir string) bool {
	ct := filepath.Clean(target)
	cd := filepath.Clean(dir)
	return ct == cd || strings.HasPrefix(ct, cd+string(os.PathSeparator))
}

// singleTopDir extracts the single entry from a set of top-level directory
// names. Returns an error if the set is empty or has more than one entry.
func singleTopDir(dirs map[string]struct{}) (string, error) {
	if len(dirs) != 1 {
		return "", fmt.Errorf("expected 1 top-level directory, found %d", len(dirs))
	}
	for d := range dirs {
		return d, nil
	}
	return "", nil // unreachable
}

// findFileByGlob finds a single file matching any of the given glob patterns
// in a directory. Returns an error if there is no match.
func findFileByGlob(dir string, patterns ...string) (string, error) {
	var matches []string
	for _, p := range patterns {
		m, _ := filepath.Glob(filepath.Join(dir, p))
		matches = append(matches, m...)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no file matching %v in %s", patterns, dir)
	}
	return matches[0], nil
}
