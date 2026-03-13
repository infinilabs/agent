/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
)

// ---------------------------------------------------------------------------
// Convention: all paths are derived from the service workspace.
//
//   workspace/                                   ← s.AbsoluteWorkspacePath()
//   workspace/assets/                            ← s.AbsoluteAssetsDirPath()
//   workspace/assets/easysearch                  ← symlink → extracted dir
//   workspace/assets/easysearch/jdk              ← symlink → JDK dir
//   workspace/assets/easysearch/config/easysearch.yml
//   workspace/assets/easysearch/config/jvm.options.d/heap.options
//   workspace/assets/graalvm-jdk-*/              ← extracted JDK
//   workspace/easysearch.pid
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Config-field extraction helpers using switch + type cast.
// ---------------------------------------------------------------------------

// resolveEasysearchVersion returns the Easysearch version string from the task
// config, regardless of whether this is a new-cluster or join-cluster task.
func resolveEasysearchVersion(s *service) (string, error) {
	switch c := s.config.(type) {
	case *NewClusterConfig:
		return c.Easysearch, nil
	case *joinClusterConfig:
		if c.FetchedEnrollInfo == nil {
			return "", fmt.Errorf("missing FetchedEnrollInfo")
		}
		return c.FetchedEnrollInfo.Version, nil
	default:
		return "", fmt.Errorf("unsupported config type: %T", s.config)
	}
}

// resolveJDKVersion returns the JDK version string from the task config.
func resolveJDKVersion(s *service) (string, error) {
	switch c := s.config.(type) {
	case *NewClusterConfig:
		return c.JDK.Version, nil
	case *joinClusterConfig:
		if c.FetchedEnrollInfo == nil {
			return "", fmt.Errorf("missing FetchedEnrollInfo")
		}
		return c.FetchedEnrollInfo.JDK.Version, nil
	default:
		return "", fmt.Errorf("unsupported config type: %T", s.config)
	}
}

// resolveJDKSource returns the JDK source and local path. For join-cluster
// tasks, JDK is always downloaded remotely.
func resolveJDKSource(s *service) (JDKSource, string) {
	switch c := s.config.(type) {
	case *NewClusterConfig:
		return c.JDK.Source, c.JDK.Path
	default:
		return JDKSourceRemote, ""
	}
}

// ========================== Step: PrepareWorkspace ==========================

type stepPrepareWorkspace struct{}

func (step *stepPrepareWorkspace) IsAssetStep() bool { return true }

func (step *stepPrepareWorkspace) NameI18nKey() string { return "stepPrepareWorkspace" }

func (step *stepPrepareWorkspace) Execute(_ context.Context, s *service) error {
	metadata, err := s.AbsoluteMetadataDirPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(metadata, 0755); err != nil {
		return err
	}
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return err
	}
	return os.MkdirAll(assets, 0755)
}

func (step *stepPrepareWorkspace) Rollback(s *service) error {
	ws, err := s.AbsoluteWorkspacePath()
	if err != nil {
		return err
	}
	return os.RemoveAll(ws)
}

// ======================== Step: DownloadEasysearch ==========================

type stepDownloadEasysearch struct{}

func (step *stepDownloadEasysearch) IsAssetStep() bool { return true }

func (step *stepDownloadEasysearch) NameI18nKey() string { return "stepDownloadEasysearch" }

func (step *stepDownloadEasysearch) Execute(ctx context.Context, s *service) error {
	version, err := resolveEasysearchVersion(s)
	if err != nil {
		return err
	}

	buildNum, err := fetchEasysearchBuildNumber(ctx, version)
	if err != nil {
		return fmt.Errorf("resolve build number: %w", err)
	}

	url := easysearchDownloadURL(version, buildNum)
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return err
	}

	dest := filepath.Join(assets, easysearchArchiveFilename(version, buildNum))

	log.Infof("[setup] downloading easysearch from %s", url)
	return httpDownload(ctx, url, dest)
}

func (step *stepDownloadEasysearch) Rollback(s *service) error {
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return err
	}

	pattern := filepath.Join(assets, "easysearch-*")
	matches, _ := filepath.Glob(pattern)
	for _, m := range matches {
		info, err := os.Lstat(m)
		if err != nil || info.IsDir() {
			continue
		}
		os.Remove(m)
	}
	return nil
}

// ====================== Step: UnpackEasysearch =============================

type stepUnpackEasysearch struct{}

func (step *stepUnpackEasysearch) IsAssetStep() bool { return true }

func (step *stepUnpackEasysearch) NameI18nKey() string { return "stepUnpackEasysearch" }

func (step *stepUnpackEasysearch) Execute(ctx context.Context, s *service) error {
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return err
	}

	archivePath, err := findEasysearchArchive(assets)
	if err != nil {
		return err
	}

	topDir, err := extractArchive(ctx, archivePath, assets, archiveLayoutWrapInArchiveDir)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	link, err := s.absoluteEasysearchHome()
	if err != nil {
		return err
	}
	os.Remove(link)
	if err := os.Symlink(topDir, link); err != nil {
		return fmt.Errorf("create easysearch symlink: %w", err)
	}

	os.Remove(archivePath)
	return nil
}

func (step *stepUnpackEasysearch) Rollback(s *service) error {
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return err
	}
	link, err := s.absoluteEasysearchHome()
	if err != nil {
		return err
	}

	if target, err := os.Readlink(link); err == nil {
		// target is relative to its parent directory
		os.RemoveAll(filepath.Join(assets, target))
	}
	os.Remove(link)
	return nil
}

// ========================== Step: DownloadJDK ==============================

type stepDownloadJDK struct{}

func (step *stepDownloadJDK) IsAssetStep() bool { return true }

func (step *stepDownloadJDK) NameI18nKey() string { return "stepDownloadJDK" }

func (step *stepDownloadJDK) Execute(ctx context.Context, s *service) error {
	jdkVersion, err := resolveJDKVersion(s)
	if err != nil {
		return err
	}

	url, err := jdkDownloadURL(jdkVersion)
	if err != nil {
		return err
	}

	filename := filepath.Base(url)
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return err
	}
	dest := filepath.Join(assets, filename)

	log.Infof("[setup] downloading JDK from %s", url)
	return httpDownload(ctx, url, dest)
}

func (step *stepDownloadJDK) Rollback(s *service) error {
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return err
	}

	// Remove JDK archives
	patterns := []string{"zulu*.tar.gz", "zulu*.zip", "graalvm-*.tar.gz", "graalvm-*.zip"}
	for _, p := range patterns {
		matches, _ := filepath.Glob(filepath.Join(assets, p))
		for _, m := range matches {
			os.Remove(m)
		}
	}
	return nil
}

// =========================== Step: UnpackJDK ===============================

type stepUnpackJDK struct{}

func (step *stepUnpackJDK) IsAssetStep() bool { return true }

func (step *stepUnpackJDK) NameI18nKey() string { return "stepUnpackJDK" }

func (step *stepUnpackJDK) Execute(ctx context.Context, s *service) error {
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return err
	}
	archivePath, err := findJDKArchive(assets)
	if err != nil {
		return err
	}

	_, err = extractArchive(ctx, archivePath, assets, archiveLayoutExpectTopLevelDir)
	if err != nil {
		return fmt.Errorf("extract JDK: %w", err)
	}

	os.Remove(archivePath)
	return nil
}

func (step *stepUnpackJDK) Rollback(s *service) error {
	assets, err := s.AbsoluteAssetsDirPath()
	if err != nil {
		return err
	}

	removeJDKDirs(assets)
	return nil
}

// ============================= Step: LinkJDK ===============================

type stepLinkJDK struct{}

func (step *stepLinkJDK) IsAssetStep() bool { return true }

func (step *stepLinkJDK) NameI18nKey() string { return "stepLinkJDK" }

func (step *stepLinkJDK) Execute(_ context.Context, s *service) error {
	esHomeDir, err := s.absoluteEasysearchHome()
	if err != nil {
		return fmt.Errorf("resolve easysearch home: %w", err)
	}
	link := filepath.Join(esHomeDir, "jdk")
	source, localPath := resolveJDKSource(s)

	var target string
	switch source {
	case JDKSourceLocal:
		expanded := localPath
		if strings.HasPrefix(expanded, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				expanded = filepath.Join(home, expanded[1:])
			}
		}
		target = expanded
	case JDKSourceRemote:
		assets, err := s.AbsoluteAssetsDirPath()
		if err != nil {
			return err
		}

		dir, err := findJDKDir(assets)
		if err != nil {
			return fmt.Errorf("locate extracted JDK: %w", err)
		}
		target = filepath.Join("..", filepath.Base(dir))
	default:
		return fmt.Errorf("unknown JDK source: %s", source)
	}

	os.Remove(link)
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("create jdk symlink: %w", err)
	}
	log.Infof("[setup] linked jdk → %s", target)
	return nil
}

func (step *stepLinkJDK) Rollback(s *service) error {
	if home, err := s.absoluteEasysearchHome(); err == nil {
		os.Remove(filepath.Join(home, "jdk"))
	}
	return nil
}

// ======================= Step: StartEasysearch ============================

type stepStartEasysearch struct{}

func (step *stepStartEasysearch) IsAssetStep() bool { return false }

func (step *stepStartEasysearch) NameI18nKey() string { return "stepStartEasysearch" }

func (step *stepStartEasysearch) Execute(ctx context.Context, s *service) error {
	return launchEasysearch(ctx, s)
}

// launchEasysearch spawns the Easysearch daemon process. It is used by both
// stepStartEasysearch (during creation) and startService (manual restart).
func launchEasysearch(ctx context.Context, s *service) error {
	// Resolve Easysearch home before spawning the child process. esHome already
	// returns an absolute path (derived from AbsoluteAssetsDirPath), so no
	// further filepath.Abs wrapping is needed.
	home, err := s.absoluteEasysearchHome()
	if err != nil {
		return fmt.Errorf("resolve easysearch home: %w", err)
	}
	// Resolve the workspace path as well so startup artifacts like --pidfile and
	// redirected stdout are written to the task workspace itself instead of being
	// re-resolved relative to Easysearch's working directory.
	ws, err := s.AbsoluteWorkspacePath()
	if err != nil {
		return err
	}
	pidFile := filepath.Join(ws, "easysearch.pid")
	stdoutLog := filepath.Join(ws, "easysearch_stdout.log")

	binary := filepath.Join(home, "bin", "easysearch")
	// Let Easysearch launch itself in daemon mode so it detaches from the
	// agent/terminal session and owns the pid file lifecycle itself.
	cmd := exec.CommandContext(ctx, binary, "-d", "--pidfile", pidFile)
	cmd.Dir = home

	logFile, err := os.OpenFile(stdoutLog,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("create stdout log: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start easysearch: %w", err)
	}
	logFile.Close()
	_ = cmd.Process.Release()

	log.Infof("[setup] easysearch daemon launched, pid file: %s, stdout log: %s", pidFile, stdoutLog)
	return nil
}

func (step *stepStartEasysearch) Rollback(s *service) error {
	return killEasysearch(s)
}

// ======================= Step: MarkAssetsReady ============================

type stepMarkAssetsReady struct{}

func (step *stepMarkAssetsReady) IsAssetStep() bool { return false }

func (step *stepMarkAssetsReady) NameI18nKey() string { return "stepMarkAssetsReady" }

func (step *stepMarkAssetsReady) Execute(_ context.Context, s *service) error {
	path, err := s.AbsoluteAssetsReadyPath()
	if err != nil {
		return fmt.Errorf("resolve assets_ready path: %w", err)
	}
	return atomicWriteFile(path, []byte("yes"))
}

func (step *stepMarkAssetsReady) Rollback(_ *service) error {
	return nil
}

// ---------------------------------------------------------------------------
// Helper functions (shared across new-cluster and join-cluster)
// ---------------------------------------------------------------------------

// fetchEasysearchBuildNumber fetches the build_number for a given Easysearch
// version from the remote versions API.
func fetchEasysearchBuildNumber(ctx context.Context, version string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, easysearchVersionsURL, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("fetch versions: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read body: %w", err)
	}

	var versions easysearchVersionsResponse
	if err := json.Unmarshal(body, &versions); err != nil {
		return 0, fmt.Errorf("parse: %w", err)
	}

	info, ok := versions[version]
	if !ok {
		return 0, fmt.Errorf("version %q not found in remote versions", version)
	}
	return info.BuildNumber, nil
}

// easysearchDownloadURL constructs the full download URL for an Easysearch
// release archive.
func easysearchDownloadURL(version string, buildNumber int) string {
	platform := getCurrentPlatform()
	parts := strings.SplitN(platform, "-", 2)
	osName, arch := parts[0], parts[1]

	ext := "tar.gz"
	if osName == "mac" || osName == "windows" {
		ext = "zip"
	}

	// Our package name uses arm64 instead of aarch64
	if arch == "aarch64" {
		arch = "arm64"
	}

	filename := fmt.Sprintf("easysearch-%s-%d-%s-%s.%s", version, buildNumber, osName, arch, ext)
	return "https://release.infinilabs.com/easysearch/stable/" + filename
}

// easysearchArchiveFilename returns just the filename part.
func easysearchArchiveFilename(version string, buildNumber int) string {
	platform := getCurrentPlatform()
	parts := strings.SplitN(platform, "-", 2)
	osName, arch := parts[0], parts[1]

	ext := "tar.gz"
	if osName == "mac" || osName == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("easysearch-%s-%d-%s-%s.%s", version, buildNumber, osName, arch, ext)
}

// findEasysearchArchive locates the Easysearch archive in the workspace.
func findEasysearchArchive(ws string) (string, error) {
	return findFileByGlob(ws, "easysearch-*.tar.gz", "easysearch-*.zip")
}

// jdkDownloadURL returns the download URL for the given JDK major version
// and current platform.
func jdkDownloadURL(jdkVersion string) (string, error) {
	const baseURL = "https://release.infinilabs.com/easysearch/jdk/"

	platform := getCurrentPlatform()

	switch jdkVersion {
	case "17":
		f, err := jdk17Filename(platform)
		if err != nil {
			return "", err
		}
		return baseURL + f, nil
	case "21":
		f, err := jdk21Filename(platform)
		if err != nil {
			return "", err
		}
		return baseURL + "21/" + f, nil
	default:
		return "", fmt.Errorf("unsupported JDK version: %s", jdkVersion)
	}
}

// jdk17Filename returns the filename for JDK 17 distribution.
func jdk17Filename(platform string) (string, error) {
	switch platform {
	case "mac-amd64":
		return "zulu17.46.19-ca-jdk17.0.9-macosx_x64.tar.gz", nil
	case "mac-aarch64":
		return "zulu17.46.19-ca-jdk17.0.9-macosx_aarch64.tar.gz", nil
	case "windows-amd64":
		return "zulu17.42.19-ca-jdk17.0.7-win_x64.zip", nil
	case "linux-amd64":
		return "zulu17.40.19-ca-jdk17.0.6-linux_x64.tar.gz", nil
	case "linux-aarch64":
		return "zulu17.40.19-ca-jdk17.0.6-linux_aarch64.tar.gz", nil
	default:
		return "", fmt.Errorf("unsupported platform for JDK 17: %s", platform)
	}
}

// jdk21Filename returns the filename for JDK 21 (GraalVM) distribution.
func jdk21Filename(platform string) (string, error) {
	switch platform {
	case "mac-amd64":
		return "graalvm-jdk-21_macos-x64_bin.tar.gz", nil
	case "mac-aarch64":
		return "graalvm-jdk-21_macos-aarch64_bin.tar.gz", nil
	case "windows-amd64":
		return "graalvm-jdk-21_windows-x64_bin.zip", nil
	case "linux-amd64":
		return "graalvm-jdk-21_linux-x64_bin.tar.gz", nil
	case "linux-aarch64":
		return "graalvm-jdk-21_linux-aarch64_bin.tar.gz", nil
	default:
		return "", fmt.Errorf("unsupported platform for JDK 21: %s", platform)
	}
}

// findJDKArchive locates a JDK archive file in the workspace dir.
func findJDKArchive(ws string) (string, error) {
	return findFileByGlob(ws, "zulu*.tar.gz", "zulu*.zip", "graalvm-*.tar.gz", "graalvm-*.zip")
}

// removeJDKArchive removes any JDK archive file from the workspace.
func removeJDKArchive(ws string) {
}

// findJDKDir finds the extracted JDK directory in the workspace.
func findJDKDir(ws string) (string, error) {
	entries, err := os.ReadDir(ws)
	if err != nil {
		return "", fmt.Errorf("read workspace: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "graalvm-") || strings.HasPrefix(name, "zulu") {
			return filepath.Join(ws, name), nil
		}
	}

	return "", fmt.Errorf("no JDK directory found in %s", ws)
}

// removeJDKDirs removes extracted JDK directories from workspace.
func removeJDKDirs(ws string) {
	entries, _ := os.ReadDir(ws)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "graalvm-") || strings.HasPrefix(name, "zulu") {
			os.RemoveAll(filepath.Join(ws, name))
		}
	}
}

// killEasysearch reads the PID file in the workspace and terminates the
// Easysearch process if it is still running.
func killEasysearch(s *service) error {
	ws, err := s.AbsoluteWorkspacePath()
	if err != nil {
		return fmt.Errorf("resolve workspace path: %w", err)
	}
	pidFile := filepath.Join(ws, "easysearch.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return nil
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID %q: %w", pidStr, err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}

	_ = proc.Signal(os.Interrupt)
	time.Sleep(5 * time.Second)
	_ = proc.Kill()

	os.Remove(pidFile)
	log.Infof("[setup] killed easysearch process %d", pid)
	return nil
}
