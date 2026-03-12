// Package evidence provides the Evidence Router for unified data ingestion.
package evidence

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// =============================================================================
// PROCESSING STREAM INTERFACE
// =============================================================================

// DraftFact represents a fact extracted but not yet approved
// This is a forward declaration - the actual type is in factstore package
type DraftFact interface{}

// ProcessingStream is the interface that all processing streams must implement.
// Each stream handles a specific type of evidence (SPL, API, CSV, etc.)
type ProcessingStream interface {
	// Name returns the unique identifier for this stream
	Name() string

	// CanProcess returns true if this stream can handle the evidence
	CanProcess(ev *EvidenceUnit) bool

	// Process extracts facts from the evidence unit
	Process(ctx context.Context, ev *EvidenceUnit) (*StreamResult, error)

	// SupportedSourceTypes returns the source types this stream handles
	SupportedSourceTypes() []SourceType

	// SupportedDomains returns the clinical domains this stream can extract
	SupportedDomains() []ClinicalDomain
}

// StreamResult contains the output of a processing stream
type StreamResult struct {
	// DraftFacts are the extracted facts (status=DRAFT)
	DraftFacts []DraftFact `json:"draftFacts"`

	// ProcessingTimeMs is how long extraction took
	ProcessingTimeMs int64 `json:"processingTimeMs"`

	// TokensUsed is the LLM token count (for LLM streams)
	TokensUsed int `json:"tokensUsed,omitempty"`

	// APICallsMade is the number of external API calls
	APICallsMade int `json:"apiCallsMade,omitempty"`

	// Warnings are non-fatal issues encountered
	Warnings []string `json:"warnings,omitempty"`

	// Metadata contains stream-specific output metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// =============================================================================
// ROUTER CONFIGURATION
// =============================================================================

// RouterConfig holds configuration for the Evidence Router
type RouterConfig struct {
	// MaxConcurrency limits parallel evidence processing
	MaxConcurrency int

	// DefaultConfidenceFloor is applied to evidence without explicit floor
	DefaultConfidenceFloor float64

	// EnableMetrics turns on Prometheus metrics
	EnableMetrics bool

	// RetryConfig for failed processing
	MaxRetries    int
	RetryDelayMs  int

	// DeduplicationEnabled prevents reprocessing identical evidence
	DeduplicationEnabled bool

	// DeduplicationTTL is how long to remember processed checksums
	DeduplicationTTL time.Duration
}

// DefaultRouterConfig returns sensible defaults
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		MaxConcurrency:         10,
		DefaultConfidenceFloor: 0.65,
		EnableMetrics:          true,
		MaxRetries:             3,
		RetryDelayMs:           1000,
		DeduplicationEnabled:   true,
		DeduplicationTTL:       24 * time.Hour,
	}
}

// =============================================================================
// EVIDENCE ROUTER
// =============================================================================

// Router directs evidence units to appropriate processing streams.
// It is the single entry point for all external data ingestion.
type Router struct {
	mu      sync.RWMutex
	config  RouterConfig
	log     *logrus.Entry
	streams map[string]ProcessingStream

	// Metrics
	totalProcessed   int64
	totalFailed      int64
	totalFactsOutput int64

	// Deduplication cache
	processedChecksums map[string]time.Time
	dedupeCleanupTicker *time.Ticker
}

// NewRouter creates a new Evidence Router with the given streams
func NewRouter(config RouterConfig, log *logrus.Entry, streams ...ProcessingStream) *Router {
	r := &Router{
		config:             config,
		log:                log.WithField("component", "evidence-router"),
		streams:            make(map[string]ProcessingStream),
		processedChecksums: make(map[string]time.Time),
	}

	// Register all provided streams
	for _, s := range streams {
		r.RegisterStream(s)
	}

	// Start deduplication cleanup if enabled
	if config.DeduplicationEnabled {
		r.startDedupeCleanup()
	}

	return r
}

// RegisterStream adds a processing stream to the router
func (r *Router) RegisterStream(stream ProcessingStream) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := stream.Name()
	if _, exists := r.streams[name]; exists {
		r.log.WithField("stream", name).Warn("Overwriting existing stream")
	}

	r.streams[name] = stream
	r.log.WithFields(logrus.Fields{
		"stream":      name,
		"sourceTypes": stream.SupportedSourceTypes(),
		"domains":     stream.SupportedDomains(),
	}).Info("Registered processing stream")
}

// Route processes a single evidence unit through the appropriate stream
func (r *Router) Route(ctx context.Context, ev *EvidenceUnit) (*ProcessingResult, error) {
	startTime := time.Now()

	// Check deduplication
	if r.config.DeduplicationEnabled && r.isDuplicate(ev) {
		return &ProcessingResult{
			EvidenceID:       ev.EvidenceID,
			Status:           StatusSkipped,
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			Warnings:         []string{"Duplicate evidence skipped"},
		}, nil
	}

	// Apply default confidence floor if not set
	if ev.ConfidenceFloor == 0 {
		ev.ConfidenceFloor = r.config.DefaultConfidenceFloor
	}

	// Find matching stream
	stream := r.findStream(ev)
	if stream == nil {
		return &ProcessingResult{
			EvidenceID:       ev.EvidenceID,
			Status:           StatusFailed,
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			Error:            fmt.Sprintf("no stream can process source type: %s", ev.SourceType),
		}, fmt.Errorf("no stream can process evidence type: %s", ev.SourceType)
	}

	r.log.WithFields(logrus.Fields{
		"evidenceId": ev.EvidenceID,
		"stream":     stream.Name(),
		"sourceType": ev.SourceType,
		"domains":    ev.ClinicalDomains,
	}).Info("Routing evidence to stream")

	// Process with retries
	var result *StreamResult
	var err error
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		result, err = stream.Process(ctx, ev)
		if err == nil {
			break
		}

		if attempt < r.config.MaxRetries {
			r.log.WithFields(logrus.Fields{
				"evidenceId": ev.EvidenceID,
				"attempt":    attempt + 1,
				"error":      err,
			}).Warn("Processing failed, retrying")
			time.Sleep(time.Duration(r.config.RetryDelayMs) * time.Millisecond)
		}
	}

	processingTime := time.Since(startTime).Milliseconds()

	if err != nil {
		r.mu.Lock()
		r.totalFailed++
		r.mu.Unlock()

		return &ProcessingResult{
			EvidenceID:       ev.EvidenceID,
			Status:           StatusFailed,
			ProcessingTimeMs: processingTime,
			StreamUsed:       stream.Name(),
			Error:            err.Error(),
		}, err
	}

	// Mark as processed for deduplication
	if r.config.DeduplicationEnabled && ev.Checksum != "" {
		r.markProcessed(ev.Checksum)
	}

	// Update metrics
	r.mu.Lock()
	r.totalProcessed++
	r.totalFactsOutput += int64(len(result.DraftFacts))
	r.mu.Unlock()

	// Build fact IDs list
	factIDs := make([]string, 0, len(result.DraftFacts))
	// In real implementation, we'd extract IDs from the facts

	status := StatusProcessed
	if len(result.DraftFacts) == 0 {
		status = StatusNoExtraction
	}

	return &ProcessingResult{
		EvidenceID:       ev.EvidenceID,
		Status:           status,
		FactsExtracted:   len(result.DraftFacts),
		FactIDs:          factIDs,
		ProcessingTimeMs: processingTime,
		StreamUsed:       stream.Name(),
		Warnings:         result.Warnings,
	}, nil
}

// RouteAll processes multiple evidence units, returning all results
func (r *Router) RouteAll(ctx context.Context, units []*EvidenceUnit) ([]*ProcessingResult, error) {
	results := make([]*ProcessingResult, 0, len(units))
	var totalErrors int

	// Process with bounded concurrency
	sem := make(chan struct{}, r.config.MaxConcurrency)
	resultsChan := make(chan *ProcessingResult, len(units))
	var wg sync.WaitGroup

	for _, ev := range units {
		wg.Add(1)
		go func(evidence *EvidenceUnit) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			result, err := r.Route(ctx, evidence)
			if err != nil {
				r.log.WithError(err).WithField("evidenceId", evidence.EvidenceID).Warn("Failed to process evidence")
			}
			resultsChan <- result
		}(ev)
	}

	// Wait for all to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results = append(results, result)
		if result.Status == StatusFailed {
			totalErrors++
		}
	}

	if totalErrors > 0 {
		r.log.WithFields(logrus.Fields{
			"total":   len(units),
			"failed":  totalErrors,
			"success": len(units) - totalErrors,
		}).Warn("Batch processing completed with errors")
	}

	return results, nil
}

// RouteBatch processes a batch with aggregated results
func (r *Router) RouteBatch(ctx context.Context, units []*EvidenceUnit) (*BatchResult, error) {
	startTime := time.Now()

	results, err := r.RouteAll(ctx, units)
	if err != nil {
		return nil, err
	}

	// Aggregate results
	batch := &BatchResult{
		TotalEvidence: len(units),
		Processed:     0,
		Skipped:       0,
		Failed:        0,
		TotalFacts:    0,
		ProcessingMs:  time.Since(startTime).Milliseconds(),
		Results:       results,
	}

	for _, r := range results {
		switch r.Status {
		case StatusProcessed:
			batch.Processed++
			batch.TotalFacts += r.FactsExtracted
		case StatusSkipped, StatusNoExtraction:
			batch.Skipped++
		case StatusFailed:
			batch.Failed++
		}
	}

	return batch, nil
}

// =============================================================================
// INTERNAL METHODS
// =============================================================================

// findStream locates the appropriate stream for the evidence
func (r *Router) findStream(ev *EvidenceUnit) ProcessingStream {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, stream := range r.streams {
		if stream.CanProcess(ev) {
			return stream
		}
	}
	return nil
}

// isDuplicate checks if evidence was recently processed
func (r *Router) isDuplicate(ev *EvidenceUnit) bool {
	if ev.Checksum == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.processedChecksums[ev.Checksum]
	return exists
}

// markProcessed records that evidence was processed
func (r *Router) markProcessed(checksum string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processedChecksums[checksum] = time.Now()
}

// startDedupeCleanup starts the periodic cleanup of old checksums
func (r *Router) startDedupeCleanup() {
	r.dedupeCleanupTicker = time.NewTicker(1 * time.Hour)
	go func() {
		for range r.dedupeCleanupTicker.C {
			r.cleanupOldChecksums()
		}
	}()
}

// cleanupOldChecksums removes expired checksum entries
func (r *Router) cleanupOldChecksums() {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-r.config.DeduplicationTTL)
	for checksum, processedAt := range r.processedChecksums {
		if processedAt.Before(cutoff) {
			delete(r.processedChecksums, checksum)
		}
	}
}

// Stop gracefully shuts down the router
func (r *Router) Stop() {
	if r.dedupeCleanupTicker != nil {
		r.dedupeCleanupTicker.Stop()
	}
}

// =============================================================================
// METRICS
// =============================================================================

// GetMetrics returns router metrics
func (r *Router) GetMetrics() RouterMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RouterMetrics{
		TotalProcessed:   r.totalProcessed,
		TotalFailed:      r.totalFailed,
		TotalFactsOutput: r.totalFactsOutput,
		RegisteredStreams: len(r.streams),
		CachedChecksums:   len(r.processedChecksums),
	}
}

// RouterMetrics contains router statistics
type RouterMetrics struct {
	TotalProcessed    int64 `json:"totalProcessed"`
	TotalFailed       int64 `json:"totalFailed"`
	TotalFactsOutput  int64 `json:"totalFactsOutput"`
	RegisteredStreams int   `json:"registeredStreams"`
	CachedChecksums   int   `json:"cachedChecksums"`
}

// BatchResult aggregates results from batch processing
type BatchResult struct {
	TotalEvidence int                 `json:"totalEvidence"`
	Processed     int                 `json:"processed"`
	Skipped       int                 `json:"skipped"`
	Failed        int                 `json:"failed"`
	TotalFacts    int                 `json:"totalFacts"`
	ProcessingMs  int64               `json:"processingMs"`
	Results       []*ProcessingResult `json:"results"`
}

// =============================================================================
// STREAM REGISTRY
// =============================================================================

// ListStreams returns information about registered streams
func (r *Router) ListStreams() []StreamInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]StreamInfo, 0, len(r.streams))
	for _, stream := range r.streams {
		infos = append(infos, StreamInfo{
			Name:         stream.Name(),
			SourceTypes:  stream.SupportedSourceTypes(),
			Domains:      stream.SupportedDomains(),
		})
	}
	return infos
}

// StreamInfo describes a registered processing stream
type StreamInfo struct {
	Name        string           `json:"name"`
	SourceTypes []SourceType     `json:"sourceTypes"`
	Domains     []ClinicalDomain `json:"domains"`
}
