// Package governance provides Tier-7 governance event emission for KB-16
// Publisher component handles event emission to message queues and audit stores.
package governance

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// =============================================================================
// PUBLISHER CONFIGURATION
// =============================================================================

// PublisherConfig contains configuration for the governance publisher
type PublisherConfig struct {
	// Redis configuration
	RedisURL     string
	RedisEnabled bool

	// Channel configuration
	CriticalChannel string // For panic/critical values - immediate processing
	StandardChannel string // For normal governance events
	AuditChannel    string // For audit trail logging

	// Retry configuration
	MaxRetries    int
	RetryInterval time.Duration

	// Buffer configuration
	BufferSize    int
	FlushInterval time.Duration

	// Feature flags
	AsyncPublish     bool
	AuditEnabled     bool
	MetricsEnabled   bool
}

// DefaultPublisherConfig returns sensible defaults for governance publishing
func DefaultPublisherConfig() PublisherConfig {
	return PublisherConfig{
		RedisEnabled:    true,
		CriticalChannel: "kb16:governance:critical",
		StandardChannel: "kb16:governance:events",
		AuditChannel:    "kb16:governance:audit",
		MaxRetries:      3,
		RetryInterval:   100 * time.Millisecond,
		BufferSize:      100,
		FlushInterval:   5 * time.Second,
		AsyncPublish:    true,
		AuditEnabled:    true,
		MetricsEnabled:  true,
	}
}

// =============================================================================
// PUBLISHER
// =============================================================================

// Publisher handles governance event emission with guaranteed delivery
type Publisher struct {
	config      PublisherConfig
	redis       *redis.Client
	log         *logrus.Entry
	buffer      chan *GovernanceEvent
	metrics     *PublisherMetrics
	mu          sync.RWMutex
	isRunning   bool
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// PublisherMetrics tracks publishing statistics
type PublisherMetrics struct {
	mu                sync.RWMutex
	EventsPublished   int64
	EventsFailed      int64
	CriticalEvents    int64
	AuditEvents       int64
	RetryCount        int64
	LastPublishTime   time.Time
	LastError         error
	LastErrorTime     time.Time
}

// NewPublisher creates a new governance event publisher
func NewPublisher(config PublisherConfig, redisClient *redis.Client, log *logrus.Entry) *Publisher {
	p := &Publisher{
		config:  config,
		redis:   redisClient,
		log:     log.WithField("component", "governance-publisher"),
		buffer:  make(chan *GovernanceEvent, config.BufferSize),
		metrics: &PublisherMetrics{},
		stopCh:  make(chan struct{}),
	}

	return p
}

// Start begins the async publishing goroutine
func (p *Publisher) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()
		return nil
	}
	p.isRunning = true
	p.mu.Unlock()

	p.log.Info("Starting governance event publisher")

	// Start async publisher worker
	if p.config.AsyncPublish {
		p.wg.Add(1)
		go p.publishWorker(ctx)
	}

	return nil
}

// Stop gracefully stops the publisher
func (p *Publisher) Stop() error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil
	}
	p.isRunning = false
	p.mu.Unlock()

	close(p.stopCh)
	p.wg.Wait()

	p.log.Info("Governance event publisher stopped")
	return nil
}

// =============================================================================
// EVENT PUBLISHING
// =============================================================================

// Publish emits a governance event
func (p *Publisher) Publish(ctx context.Context, event *GovernanceEvent) error {
	if event == nil {
		return fmt.Errorf("cannot publish nil event")
	}

	// Determine channel based on severity
	channel := p.config.StandardChannel
	if event.IsCriticalOrPanic() {
		channel = p.config.CriticalChannel
	}

	// Async or sync publishing
	if p.config.AsyncPublish {
		return p.publishAsync(event)
	}
	return p.publishSync(ctx, event, channel)
}

// PublishCritical immediately publishes a critical event (bypasses buffer)
func (p *Publisher) PublishCritical(ctx context.Context, event *GovernanceEvent) error {
	if event == nil {
		return fmt.Errorf("cannot publish nil event")
	}

	// Critical events always go sync and to critical channel
	event.Priority = 1
	return p.publishSync(ctx, event, p.config.CriticalChannel)
}

// PublishBatch publishes multiple events
func (p *Publisher) PublishBatch(ctx context.Context, events []*GovernanceEvent) error {
	var errs []error
	for _, event := range events {
		if err := p.Publish(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to publish %d/%d events", len(errs), len(events))
	}
	return nil
}

// =============================================================================
// INTERNAL PUBLISHING METHODS
// =============================================================================

func (p *Publisher) publishAsync(event *GovernanceEvent) error {
	select {
	case p.buffer <- event:
		return nil
	default:
		// Buffer full - fall back to sync
		p.log.Warn("Event buffer full, falling back to sync publish")
		return p.publishSync(context.Background(), event, p.config.StandardChannel)
	}
}

func (p *Publisher) publishSync(ctx context.Context, event *GovernanceEvent, channel string) error {
	if !p.config.RedisEnabled || p.redis == nil {
		p.log.Debug("Redis disabled, event logged locally")
		p.logEvent(event)
		return nil
	}

	// Serialize event
	data, err := event.ToJSON()
	if err != nil {
		p.recordError(err)
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Publish with retries
	var lastErr error
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(p.config.RetryInterval * time.Duration(attempt))
			p.metrics.mu.Lock()
			p.metrics.RetryCount++
			p.metrics.mu.Unlock()
		}

		err = p.redis.Publish(ctx, channel, data).Err()
		if err == nil {
			p.recordSuccess(event)

			// Also publish to audit channel if enabled
			if p.config.AuditEnabled {
				go p.publishAudit(ctx, event, data)
			}

			return nil
		}
		lastErr = err
	}

	p.recordError(lastErr)
	return fmt.Errorf("failed to publish after %d retries: %w", p.config.MaxRetries, lastErr)
}

func (p *Publisher) publishAudit(ctx context.Context, event *GovernanceEvent, data []byte) {
	auditRecord := AuditRecord{
		EventID:     event.ID.String(),
		EventType:   string(event.EventType),
		PatientID:   event.PatientID,
		Severity:    string(event.Severity),
		PublishedAt: time.Now().UTC(),
		Channel:     p.config.StandardChannel,
	}

	auditData, err := json.Marshal(auditRecord)
	if err != nil {
		p.log.WithError(err).Warn("Failed to serialize audit record")
		return
	}

	if err := p.redis.Publish(ctx, p.config.AuditChannel, auditData).Err(); err != nil {
		p.log.WithError(err).Warn("Failed to publish audit record")
	}

	p.metrics.mu.Lock()
	p.metrics.AuditEvents++
	p.metrics.mu.Unlock()
}

func (p *Publisher) publishWorker(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			// Drain remaining events
			p.drainBuffer(ctx)
			return

		case <-ctx.Done():
			p.drainBuffer(ctx)
			return

		case event := <-p.buffer:
			channel := p.config.StandardChannel
			if event.IsCriticalOrPanic() {
				channel = p.config.CriticalChannel
			}
			if err := p.publishSync(ctx, event, channel); err != nil {
				p.log.WithError(err).Error("Failed to publish event from buffer")
			}

		case <-ticker.C:
			// Periodic flush - currently just logs metrics
			p.logMetrics()
		}
	}
}

func (p *Publisher) drainBuffer(ctx context.Context) {
	p.log.Info("Draining event buffer...")
	timeout := time.After(10 * time.Second)

	for {
		select {
		case event := <-p.buffer:
			channel := p.config.StandardChannel
			if event.IsCriticalOrPanic() {
				channel = p.config.CriticalChannel
			}
			_ = p.publishSync(ctx, event, channel)
		case <-timeout:
			p.log.Warn("Buffer drain timeout")
			return
		default:
			p.log.Info("Event buffer drained")
			return
		}
	}
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// IsCriticalOrPanic checks if event requires immediate handling
func (e *GovernanceEvent) IsCriticalOrPanic() bool {
	return e.Severity == SeverityCritical ||
		e.EventType == EventPanicLabValue ||
		e.EventType == EventCriticalLabValue
}

func (p *Publisher) logEvent(event *GovernanceEvent) {
	p.log.WithFields(logrus.Fields{
		"event_id":    event.ID,
		"event_type":  event.EventType,
		"patient_id":  event.PatientID,
		"severity":    event.Severity,
		"title":       event.Title,
	}).Info("Governance event (local log)")
}

func (p *Publisher) recordSuccess(event *GovernanceEvent) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.EventsPublished++
	p.metrics.LastPublishTime = time.Now()

	if event.IsCriticalOrPanic() {
		p.metrics.CriticalEvents++
	}
}

func (p *Publisher) recordError(err error) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.EventsFailed++
	p.metrics.LastError = err
	p.metrics.LastErrorTime = time.Now()
}

func (p *Publisher) logMetrics() {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	p.log.WithFields(logrus.Fields{
		"published":      p.metrics.EventsPublished,
		"failed":         p.metrics.EventsFailed,
		"critical":       p.metrics.CriticalEvents,
		"audit":          p.metrics.AuditEvents,
		"retries":        p.metrics.RetryCount,
		"buffer_size":    len(p.buffer),
	}).Debug("Governance publisher metrics")
}

// GetMetrics returns current publisher metrics
func (p *Publisher) GetMetrics() PublisherMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// =============================================================================
// AUDIT RECORD
// =============================================================================

// AuditRecord represents an immutable audit trail entry
type AuditRecord struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	PatientID   string    `json:"patient_id"`
	Severity    string    `json:"severity"`
	PublishedAt time.Time `json:"published_at"`
	Channel     string    `json:"channel"`
}

// =============================================================================
// CONVENIENCE CONSTRUCTORS
// =============================================================================

// NewPublisherFromRedisURL creates a publisher with Redis connection
func NewPublisherFromRedisURL(redisURL string, log *logrus.Entry) (*Publisher, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	config := DefaultPublisherConfig()
	config.RedisURL = redisURL

	return NewPublisher(config, client, log), nil
}
