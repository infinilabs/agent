/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package logs

import (
	"fmt"
	"sync"
	"testing"

	"infini.sh/framework/core/shipper"
)

// mockShipper records the batches it was asked to deliver; failN makes
// the first N Ship calls fail.
type mockShipper struct {
	mu       sync.Mutex
	batches  []int // size of each delivered batch
	failN    int
	calls    int
	closeCnt int
}

func (m *mockShipper) Ship(batch [][]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.calls <= m.failN {
		return fmt.Errorf("mock ship failure %d", m.calls)
	}
	m.batches = append(m.batches, len(batch))
	return nil
}

func (m *mockShipper) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCnt++
	return nil
}

var (
	registerOnce sync.Once
	currentMock  *mockShipper
)

// useMockShipper registers the "mock_ship_test" factory once (Go test
// binaries run tests in one process; duplicate registry entries panic)
// and swaps in a fresh mock for the next emitter construction.
func useMockShipper(failN int) *mockShipper {
	registerOnce.Do(func() {
		shipper.Register("mock_ship_test", func(cfg map[string]interface{}) (shipper.Shipper, error) {
			return currentMock, nil
		})
	})
	m := &mockShipper{failN: failN}
	currentMock = m
	return m
}

func shipModeConfig(batchSize int) Config {
	return Config{
		QueueName:     "logs",
		ShipDirect:    true,
		Shipper:       "mock_ship_test",
		ShipBatchSize: batchSize,
	}
}

func data(i int) []byte { return []byte(fmt.Sprintf("event-%d", i)) }

// TestEmitterFlushOnBatchSize verifies the in-flight batch is shipped
// as soon as it reaches batchSize and the committed offset advances to
// the last delivered event.
func TestEmitterFlushOnBatchSize(t *testing.T) {
	m := useMockShipper(0)
	e, err := newEmitter(shipModeConfig(2))
	if err != nil {
		t.Fatalf("newEmitter: %v", err)
	}

	e.beginFile(100)
	if err := e.emit(data(1), 110); err != nil {
		t.Fatalf("emit 1: %v", err)
	}
	if err := e.emit(data(2), 120); err != nil {
		t.Fatalf("emit 2: %v", err) // batch full -> auto flush
	}
	if got := e.committed; got != 120 {
		t.Fatalf("committed = %d, want 120", got)
	}
	if len(m.batches) != 1 || m.batches[0] != 2 {
		t.Fatalf("batches = %v, want [2]", m.batches)
	}

	// remainder ships on the final flush
	if err := e.emit(data(3), 130); err != nil {
		t.Fatalf("emit 3: %v", err)
	}
	if err := e.flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if got := e.committed; got != 130 {
		t.Fatalf("committed after final flush = %d, want 130", got)
	}
	if len(m.batches) != 2 || m.batches[1] != 1 {
		t.Fatalf("batches = %v, want [2 1]", m.batches)
	}
}

// TestEmitterFailureKeepsOffset verifies that when delivery fails the
// committed offset does NOT advance -- the caller re-reads from the
// committed offset on the next scan (the file is the durable buffer).
func TestEmitterFailureKeepsOffset(t *testing.T) {
	useMockShipper(1) // first Ship call fails
	e, err := newEmitter(shipModeConfig(1))
	if err != nil {
		t.Fatalf("newEmitter: %v", err)
	}

	e.beginFile(50)
	if err := e.emit(data(1), 60); err == nil {
		t.Fatal("expected delivery error, got nil")
	}
	if got := e.committed; got != 50 {
		t.Fatalf("committed = %d, want 50 (unchanged after failure)", got)
	}
}

// TestEmitterQueueModeRequiresNoShipper verifies queue mode builds
// without any registered shipper (no external dependencies).
func TestEmitterQueueModeRequiresNoShipper(t *testing.T) {
	e, err := newEmitter(Config{QueueName: "logs"})
	if err != nil {
		t.Fatalf("newEmitter (queue mode): %v", err)
	}
	if e.shipMode {
		t.Fatal("default mode must be queue mode")
	}
}

// TestEmitterBeginFileResetsBatch verifies per-file batch state resets
// and the pending (undelivered) batch of the previous file is dropped:
// those bytes will be re-read from the file.
func TestEmitterBeginFileResetsBatch(t *testing.T) {
	useMockShipper(0)
	e, err := newEmitter(shipModeConfig(100)) // never auto-flushes
	if err != nil {
		t.Fatalf("newEmitter: %v", err)
	}

	e.beginFile(0)
	_ = e.emit(data(1), 10) // stays in-flight
	e.beginFile(500)
	if len(e.batch) != 0 {
		t.Fatalf("batch not reset: %d events", len(e.batch))
	}
	if got := e.committed; got != 500 {
		t.Fatalf("committed = %d, want 500", got)
	}
}

// TestEmitterInvalidFlushInterval verifies a bad ship_flush_interval is
// rejected at construction time.
func TestEmitterInvalidFlushInterval(t *testing.T) {
	useMockShipper(0)
	_, err := newEmitter(Config{
		ShipDirect:        true,
		Shipper:           "mock_ship_test",
		ShipFlushInterval: "nope",
	})
	if err == nil {
		t.Fatal("expected error for invalid ship_flush_interval, got nil")
	}
}
