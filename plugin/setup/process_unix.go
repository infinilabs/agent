//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || netbsd || openbsd || solaris

/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"os"
	"syscall"
)

// isProcessAlive reports whether the given process is still running by sending
// signal 0, which performs error checking without actually sending a signal.
//
// On these platforms Easysearch is launched as a daemon (parent = init /
// launchd). When the process exits its zombie is reaped almost immediately by
// init, so the window where signal 0 would incorrectly return true for a zombie
// is negligibly short (well under the 500 ms poll interval used by
// killEasysearch). Signal 0 alone is therefore sufficient here.
func isProcessAlive(proc *os.Process) bool {
	return proc.Signal(syscall.Signal(0)) == nil
}
