//go:build linux

/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"fmt"
	"os"
	"strings"
)

// isProcessAlive reports whether the given process is genuinely running.
//
// On Linux we read /proc/<pid>/status rather than relying on signal 0. Signal 0
// succeeds even for zombie processes (state Z) because the kernel still has a
// PID table entry for them. Reading the status file lets us correctly treat a
// zombie as "not running" — it has already exited and cannot do any work.
//
// If the file is missing (ENOENT), the process has been fully reaped and no
// longer exists.
func isProcessAlive(proc *os.Process) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", proc.Pid))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "State:") {
			// Example: "State:\tZ (zombie)"
			return !strings.ContainsRune(line, 'Z')
		}
	}
	return false
}
