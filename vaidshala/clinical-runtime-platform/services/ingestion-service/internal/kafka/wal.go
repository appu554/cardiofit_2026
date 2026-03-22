package kafka

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// WALConfig holds configuration for the Write-Ahead Log.
type WALConfig struct {
	Dir          string        // Directory for WAL files
	MaxSizeBytes int64         // Maximum total WAL size (default 10GB)
	RetryInterval time.Duration // Retry interval for replaying pending messages
}

// DefaultWALConfig returns production defaults from spec section 7.1.
func DefaultWALConfig() WALConfig {
	return WALConfig{
		Dir:           "/tmp/ingestion-wal",
		MaxSizeBytes:  10 * 1024 * 1024 * 1024, // 10 GB
		RetryInterval: 30 * time.Second,
	}
}

// walEntry is a single message buffered in the WAL when Kafka is unreachable.
type walEntry struct {
	Topic        string                 `json:"topic"`
	PartitionKey string                 `json:"partition_key"`
	Envelope     json.RawMessage        `json:"envelope"`
	CreatedAt    time.Time              `json:"created_at"`
	Attempts     int                    `json:"attempts"`
}

// WAL implements a file-backed Write-Ahead Log for Kafka publish failover.
// When Kafka is temporarily unreachable, messages are buffered to local disk
// (capped at MaxSizeBytes) and retried every RetryInterval.
type WAL struct {
	config   WALConfig
	mu       sync.Mutex
	entries  []walEntry
	sizeBytes int64
	logger   *zap.Logger
	stopCh   chan struct{}
}

// NewWAL creates a WAL instance and ensures the directory exists.
func NewWAL(config WALConfig, logger *zap.Logger) (*WAL, error) {
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, err
	}
	w := &WAL{
		config: config,
		logger: logger,
		stopCh: make(chan struct{}),
	}
	// Load any persisted entries from a previous crash
	w.loadFromDisk()
	return w, nil
}

// Append buffers a failed Kafka message to the WAL.
// Returns false if the WAL is full (MaxSizeBytes exceeded).
func (w *WAL) Append(topic, partitionKey string, envelope []byte) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	entrySize := int64(len(envelope) + len(topic) + len(partitionKey) + 128) // overhead estimate
	if w.sizeBytes+entrySize > w.config.MaxSizeBytes {
		w.logger.Warn("WAL full, dropping message",
			zap.String("topic", topic),
			zap.Int64("wal_size_bytes", w.sizeBytes),
		)
		return false
	}

	w.entries = append(w.entries, walEntry{
		Topic:        topic,
		PartitionKey: partitionKey,
		Envelope:     envelope,
		CreatedAt:    time.Now().UTC(),
	})
	w.sizeBytes += entrySize
	w.persistToDisk()
	return true
}

// Pending returns the number of messages waiting in the WAL.
func (w *WAL) Pending() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.entries)
}

// StartReplay begins the background retry loop. It calls publishFn for each
// pending entry and removes it on success.
func (w *WAL) StartReplay(ctx context.Context, publishFn func(ctx context.Context, topic, key string, data []byte) error) {
	go func() {
		ticker := time.NewTicker(w.config.RetryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				w.replayOnce(ctx, publishFn)
			}
		}
	}()
}

// replayOnce attempts to publish all pending WAL entries.
func (w *WAL) replayOnce(ctx context.Context, publishFn func(ctx context.Context, topic, key string, data []byte) error) {
	w.mu.Lock()
	if len(w.entries) == 0 {
		w.mu.Unlock()
		return
	}
	pending := make([]walEntry, len(w.entries))
	copy(pending, w.entries)
	w.mu.Unlock()

	var remaining []walEntry
	var remainingSize int64

	for _, entry := range pending {
		entry.Attempts++
		if err := publishFn(ctx, entry.Topic, entry.PartitionKey, entry.Envelope); err != nil {
			w.logger.Warn("WAL replay failed, will retry",
				zap.String("topic", entry.Topic),
				zap.Int("attempt", entry.Attempts),
				zap.Error(err),
			)
			remaining = append(remaining, entry)
			remainingSize += int64(len(entry.Envelope) + len(entry.Topic) + len(entry.PartitionKey) + 128)
		} else {
			w.logger.Info("WAL replay succeeded",
				zap.String("topic", entry.Topic),
				zap.Int("attempt", entry.Attempts),
			)
		}
	}

	w.mu.Lock()
	w.entries = remaining
	w.sizeBytes = remainingSize
	w.persistToDisk()
	w.mu.Unlock()
}

// Stop signals the replay goroutine to terminate.
func (w *WAL) Stop() {
	close(w.stopCh)
}

// walFilePath returns the path to the WAL persistence file.
func (w *WAL) walFilePath() string {
	return filepath.Join(w.config.Dir, "wal.json")
}

// persistToDisk writes the current entries to disk. Must be called with mu held.
func (w *WAL) persistToDisk() {
	data, err := json.Marshal(w.entries)
	if err != nil {
		w.logger.Error("failed to marshal WAL entries", zap.Error(err))
		return
	}
	if err := os.WriteFile(w.walFilePath(), data, 0644); err != nil {
		w.logger.Error("failed to persist WAL to disk", zap.Error(err))
	}
}

// loadFromDisk recovers WAL entries from a previous process. Must be called before StartReplay.
func (w *WAL) loadFromDisk() {
	data, err := os.ReadFile(w.walFilePath())
	if err != nil {
		return // no WAL file — clean start
	}
	var entries []walEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		w.logger.Warn("corrupt WAL file, starting fresh", zap.Error(err))
		return
	}
	w.entries = entries
	for _, e := range entries {
		w.sizeBytes += int64(len(e.Envelope) + len(e.Topic) + len(e.PartitionKey) + 128)
	}
	w.logger.Info("recovered WAL entries from disk", zap.Int("count", len(entries)))
}
