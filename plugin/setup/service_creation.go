/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import "context"

type creationStep interface {
	NameI18nKey() string

	// IsAssetStep reports whether this step prepares on-disk assets whose
	// progress should be persisted to assets_ready on success. Steps that
	// start or wait for the Easysearch process should return false.
	IsAssetStep() bool

	// Execute runs the step. Implementations must honour ctx cancellation so
	// that Pause() can stop execution promptly. Concretely:
	//
	//   - Before any blocking call (network I/O, process wait, file write,
	//     channel receive, etc.) prefer the ctx-aware variant of the API, e.g.
	//     http.NewRequestWithContext, exec.CommandContext, os.File with a
	//     deadline, or a select that includes ctx.Done().
	//
	//   - Inside CPU-intensive loops insert periodic checks:
	//
	//       if ctx.Err() != nil { return ctx.Err() }
	//
	//     Go's async preemption (Go 1.14+) will schedule other goroutines even
	//     during pure computation, but it does NOT inject a ctx cancellation —
	//     the goroutine simply resumes after the preemption and continues
	//     computing. Without an explicit check the step will run to completion
	//     before Pause() is able to proceed.
	//
	// Return ctx.Err() (or an error wrapping it) when cancellation is detected
	// so that executeLoop can distinguish a pause from a real failure.
	Execute(ctx context.Context, service *service) error

	// Rollback undoes or cleans up any partial side-effects that Execute may
	// have written to disk so that a subsequent Execute call starts from a
	// clean state. Rollback is called when:
	//   - Execute returned an error (step failed), or
	//   - the context was cancelled while Execute was running (task paused).
	// Rollback is best-effort: a non-nil error is logged but does not change
	// the task's status.
	Rollback(service *service) error
}
