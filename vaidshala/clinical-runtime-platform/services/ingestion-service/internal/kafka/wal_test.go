package kafka

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func testWALLogger() *zap.Logger {
	return zap.NewNop()
}

func newTestWAL(t *testing.T, publishFn func(ctx context.Context, topic, key string, data []byte) error) *WAL {
	t.Helper()
	cfg := WALConfig{
		Dir:           t.TempDir(),
		MaxSizeBytes:  1024 * 1024, // 1 MB for tests
		RetryInterval: 50 * time.Millisecond,
	}
	w, err := NewWAL(cfg, testWALLogger())
	if err != nil {
		t.Fatalf("NewWAL: %v", err)
	}
	return w
}

func TestWAL_AppendAndReplay(t *testing.T) {
	var replayed int
	publishFn := func(ctx context.Context, topic, key string, data []byte) error {
		replayed++
		return nil
	}

	w := newTestWAL(t, publishFn)
	defer w.Stop()

	for i := 0; i < 3; i++ {
		ok := w.Append("test-topic", fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf(`{"i":%d}`, i)))
		if !ok {
			t.Fatalf("Append %d returned false", i)
		}
	}

	if pending := w.Pending(); pending != 3 {
		t.Fatalf("expected 3 pending, got %d", pending)
	}

	// Trigger a single replay pass.
	w.replayOnce(context.Background(), publishFn)

	if replayed != 3 {
		t.Fatalf("expected 3 replayed, got %d", replayed)
	}
	if pending := w.Pending(); pending != 0 {
		t.Fatalf("expected 0 pending after replay, got %d", pending)
	}
}

func TestWAL_ReplayPartialFailure(t *testing.T) {
	var callCount int
	publishFn := func(ctx context.Context, topic, key string, data []byte) error {
		callCount++
		if callCount == 3 {
			return fmt.Errorf("simulated failure at message 3")
		}
		return nil
	}

	w := newTestWAL(t, publishFn)
	defer w.Stop()

	for i := 0; i < 5; i++ {
		w.Append("topic", fmt.Sprintf("k%d", i), []byte(fmt.Sprintf(`{"v":%d}`, i)))
	}

	w.replayOnce(context.Background(), publishFn)

	// Message 3 failed; messages 4 and 5 also tried. Only message 3 remains
	// if it failed, plus any after it that also failed. The WAL replays all
	// sequentially: 1 ok, 2 ok, 3 fail, 4 ok, 5 ok => 1 remaining.
	pending := w.Pending()
	if pending != 1 {
		t.Fatalf("expected 1 pending after partial failure, got %d", pending)
	}
}

func TestWAL_SizeCap(t *testing.T) {
	w := newTestWAL(t, nil)
	defer w.Stop()

	w.mu.Lock()
	sizeBefore := w.sizeBytes
	w.mu.Unlock()

	w.Append("topic", "key", []byte(`{"data":"payload"}`))

	w.mu.Lock()
	sizeAfter := w.sizeBytes
	w.mu.Unlock()

	if sizeAfter <= sizeBefore {
		t.Fatalf("expected size to increase after append: before=%d after=%d", sizeBefore, sizeAfter)
	}
}

func TestWAL_EmptyReplay(t *testing.T) {
	var replayed int
	publishFn := func(ctx context.Context, topic, key string, data []byte) error {
		replayed++
		return nil
	}

	w := newTestWAL(t, publishFn)
	defer w.Stop()

	w.replayOnce(context.Background(), publishFn)

	if replayed != 0 {
		t.Fatalf("expected 0 replayed on empty WAL, got %d", replayed)
	}
}

func TestWAL_PersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	cfg := WALConfig{
		Dir:           dir,
		MaxSizeBytes:  1024 * 1024,
		RetryInterval: time.Minute,
	}

	// First instance: append 2 entries.
	w1, err := NewWAL(cfg, testWALLogger())
	if err != nil {
		t.Fatalf("NewWAL (1st): %v", err)
	}
	w1.Append("t1", "k1", []byte(`{"a":1}`))
	w1.Append("t2", "k2", []byte(`{"a":2}`))
	w1.Stop()

	// Second instance: should recover entries from disk.
	w2, err := NewWAL(cfg, testWALLogger())
	if err != nil {
		t.Fatalf("NewWAL (2nd): %v", err)
	}
	defer w2.Stop()

	if pending := w2.Pending(); pending != 2 {
		t.Fatalf("expected 2 pending after reopen, got %d", pending)
	}
}

func TestWAL_FileCreation(t *testing.T) {
	dir := t.TempDir()
	cfg := WALConfig{
		Dir:           dir,
		MaxSizeBytes:  1024 * 1024,
		RetryInterval: time.Minute,
	}

	w, err := NewWAL(cfg, testWALLogger())
	if err != nil {
		t.Fatalf("NewWAL: %v", err)
	}
	defer w.Stop()

	// Append to trigger file persistence.
	w.Append("topic", "key", []byte(`{"x":1}`))

	walFile := filepath.Join(dir, "wal.json")
	info, err := os.Stat(walFile)
	if err != nil {
		t.Fatalf("WAL file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("WAL file is empty after append")
	}
}
