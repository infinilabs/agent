/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

var jdkVersionPattern = regexp.MustCompile(`version "([^"]+)"`)

// LocalJdk represents a locally discovered JDK installation.
type LocalJdk struct {
	// Major version number (e.g. 8, 11, 17, 21).
	Major int
	// Full version string as reported by "java -version" (e.g. "21.0.2").
	Version string
	// Resolved JDK home directory, with ~ prefix for paths under $HOME.
	HomePath string
}

// findAllLocalJdks searches common locations for locally installed JDKs and
// returns every unique installation found, regardless of compatibility with
// any particular Easysearch version.
func findAllLocalJdks() []LocalJdk {
	var jdks []LocalJdk
	seen := make(map[string]bool)

	for _, basePath := range jdkSearchPaths() {
		if basePath == "" {
			continue
		}
		for _, jdk := range searchJdksInPath(basePath, seen) {
			jdks = append(jdks, jdk)
		}
	}

	// Also check system PATH for a java that may not reside under any
	// of the well-known directories above.
	if javaPath, err := exec.LookPath("java"); err == nil {
		if jdk, ok := probeJdk(javaPath); ok {
			if !seen[jdk.HomePath] {
				jdks = append(jdks, jdk)
				seen[jdk.HomePath] = true
			}
		}
	}

	return jdks
}

// jdkSearchPaths returns the list of directories to scan for JDK installations.
func jdkSearchPaths() []string {
	var paths []string

	// Environment variables
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		paths = append(paths, javaHome)
	}
	if jdkHome := os.Getenv("JDK_HOME"); jdkHome != "" {
		paths = append(paths, jdkHome)
	}

	// Per-user directories
	if homeDir, _ := os.UserHomeDir(); homeDir != "" {
		paths = append(paths,
			filepath.Join(homeDir, ".jdks"),
			filepath.Join(homeDir, ".sdkman", "candidates", "java"),
		)
	}

	// Platform-specific directories
	switch runtime.GOOS {
	case "darwin":
		paths = append(paths,
			"/Library/Java/JavaVirtualMachines",
			"/opt/homebrew/Cellar/openjdk",
		)
	case "linux":
		paths = append(paths,
			"/usr/lib/jvm",
			"/usr/java",
			"/opt/java",
		)
	}

	return paths
}

// searchJdksInPath enumerates entries in dir, probes each as a possible JDK
// installation, and appends new discoveries to the result slice.  seen is
// updated in place to deduplicate across calls.
func searchJdksInPath(dir string, seen map[string]bool) []LocalJdk {
	// When dir itself is a JDK root (e.g. JAVA_HOME points directly to a
	// JDK), try probing it before descending into children.
	if jdk, ok := probeJdkHome(dir); ok {
		if !seen[jdk.HomePath] {
			seen[jdk.HomePath] = true
			return []LocalJdk{jdk}
		}
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	javaExe := "java"
	if runtime.GOOS == "windows" {
		javaExe = "java.exe"
	}

	var jdks []LocalJdk
	for _, entry := range entries {
		jdkPath := filepath.Join(dir, entry.Name())

		// Standard layout: <root>/bin/java
		javaPath := filepath.Join(jdkPath, "bin", javaExe)
		if _, err := os.Stat(javaPath); os.IsNotExist(err) {
			// macOS bundle layout: <root>/Contents/Home/bin/java
			javaPath = filepath.Join(jdkPath, "Contents", "Home", "bin", javaExe)
			if _, err := os.Stat(javaPath); os.IsNotExist(err) {
				continue
			}
		}

		if jdk, ok := probeJdk(javaPath); ok {
			if !seen[jdk.HomePath] {
				jdks = append(jdks, jdk)
				seen[jdk.HomePath] = true
			}
		}
	}

	return jdks
}

// probeJdkHome checks whether dir itself looks like a JDK root (i.e. has
// bin/java) and probes it.
func probeJdkHome(dir string) (LocalJdk, bool) {
	javaExe := "java"
	if runtime.GOOS == "windows" {
		javaExe = "java.exe"
	}

	javaPath := filepath.Join(dir, "bin", javaExe)
	if _, err := os.Stat(javaPath); err != nil {
		// Also try macOS bundle layout
		javaPath = filepath.Join(dir, "Contents", "Home", "bin", javaExe)
		if _, err := os.Stat(javaPath); err != nil {
			return LocalJdk{}, false
		}
	}

	return probeJdk(javaPath)
}

// probeJdk runs "java -version" at javaPath and returns parsed info.
func probeJdk(javaPath string) (LocalJdk, bool) {
	cmd := exec.Command(javaPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return LocalJdk{}, false
	}

	version, major, err := parseJavaVersion(string(output))
	if err != nil {
		return LocalJdk{}, false
	}

	// Resolve symlinks and derive JDK home (parent of bin/).
	realPath := filepath.Dir(filepath.Dir(javaPath))
	if resolved, err := filepath.EvalSymlinks(realPath); err == nil {
		realPath = resolved
	}

	displayPath := realPath
	if home, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(realPath, home) {
			displayPath = "~" + strings.TrimPrefix(realPath, home)
		}
	}

	return LocalJdk{
		Major:    major,
		Version:  version,
		HomePath: displayPath,
	}, true
}

// parseJavaVersion extracts the full version string and major version from
// "java -version" output.  Handles both the legacy "1.x" format (Java ≤8)
// and the modern major-first format (Java 9+).
func parseJavaVersion(versionOutput string) (string, int, error) {
	matches := jdkVersionPattern.FindStringSubmatch(versionOutput)
	if len(matches) < 2 {
		return "", 0, errors.New("unable to parse java version")
	}

	version := strings.TrimSpace(matches[1])
	if version == "" {
		return "", 0, errors.New("empty java version")
	}

	var major int
	if strings.HasPrefix(version, "1.") {
		parts := strings.Split(version, ".")
		if len(parts) < 2 {
			return "", 0, fmt.Errorf("invalid java version: %s", version)
		}
		v, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, fmt.Errorf("parse java major version: %w", err)
		}
		major = v
	} else {
		parts := strings.Split(version, ".")
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return "", 0, fmt.Errorf("parse java major version: %w", err)
		}
		major = v
	}

	return version, major, nil
}
