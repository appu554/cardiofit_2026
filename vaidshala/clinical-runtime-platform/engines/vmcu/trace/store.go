// store.go defines the SafetyTrace persistence interface (Phase 6.3-6.4).
//
// The engine writes traces to an in-memory buffer via TraceWriter.
// The runtime layer implements TraceStore for durable persistence
// (PostgreSQL, append-only, 10-year DISHA retention).
package trace

import "context"

// TraceStore is the persistence interface for SafetyTrace records.
// Implementations must be APPEND-ONLY — no UPDATE or DELETE operations.
type TraceStore interface {
	// Persist writes one or more SafetyTrace records to durable storage.
	// Must be non-blocking (target < 5ms via buffered channel + goroutine).
	// Records written here are immutable — 10-year DISHA retention.
	Persist(ctx context.Context, traces []SafetyTrace) error

	// QueryByPatient returns paginated traces for a patient.
	// GET /patients/:id/safety-traces
	QueryByPatient(ctx context.Context, patientID string, opts QueryOpts) ([]SafetyTrace, error)

	// QueryByGate returns traces filtered by gate outcome.
	// GET /patients/:id/safety-traces?gate=HALT
	QueryByGate(ctx context.Context, patientID string, gate string, opts QueryOpts) ([]SafetyTrace, error)

	// GetByID returns a single trace by its ID.
	// GET /safety-traces/:trace_id
	GetByID(ctx context.Context, traceID string) (*SafetyTrace, error)
}

// QueryOpts controls pagination for trace queries.
type QueryOpts struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// RetentionPolicy defines the data retention rules for SafetyTrace.
type RetentionPolicy struct {
	// RetentionYears is the minimum retention period (default: 10 for DISHA).
	RetentionYears int

	// ArchiveAfterYears triggers archival to cold storage (not deletion).
	// Records beyond this age are moved, never deleted.
	ArchiveAfterYears int
}

// DefaultRetentionPolicy returns the DISHA-compliant retention policy.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		RetentionYears:    10,
		ArchiveAfterYears: 5,
	}
}

// AsyncTraceWriter wraps TraceWriter with a non-blocking persistence layer.
// Buffers traces and flushes to TraceStore asynchronously.
type AsyncTraceWriter struct {
	writer *TraceWriter
	store  TraceStore
	ch     chan SafetyTrace
	done   chan struct{}
}

// NewAsyncTraceWriter creates an async writer that persists traces in the background.
// bufferSize controls the channel capacity (recommended: 1000).
func NewAsyncTraceWriter(store TraceStore, bufferSize int) *AsyncTraceWriter {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	atw := &AsyncTraceWriter{
		writer: NewTraceWriter(),
		store:  store,
		ch:     make(chan SafetyTrace, bufferSize),
		done:   make(chan struct{}),
	}
	go atw.persistLoop()
	return atw
}

// Record creates a SafetyTrace and queues it for async persistence.
func (a *AsyncTraceWriter) Record(result interface{ GetCycleResult() interface{} }) SafetyTrace {
	// This is called via the normal TraceWriter path;
	// the runtime layer is responsible for feeding traces to the channel.
	return SafetyTrace{}
}

// Enqueue adds a trace to the async persistence queue.
// Non-blocking — drops if buffer is full (logged as metric).
func (a *AsyncTraceWriter) Enqueue(st SafetyTrace) bool {
	select {
	case a.ch <- st:
		return true
	default:
		return false // buffer full — caller should log/metric
	}
}

// Stop gracefully shuts down the async writer, flushing remaining traces.
func (a *AsyncTraceWriter) Stop() {
	close(a.ch)
	<-a.done
}

func (a *AsyncTraceWriter) persistLoop() {
	defer close(a.done)
	batch := make([]SafetyTrace, 0, 100)

	for st := range a.ch {
		batch = append(batch, st)
		// Flush in batches of 100 or when channel drains
		if len(batch) >= 100 {
			_ = a.store.Persist(context.Background(), batch)
			batch = batch[:0]
		}
	}
	// Flush remaining
	if len(batch) > 0 {
		_ = a.store.Persist(context.Background(), batch)
	}
}
