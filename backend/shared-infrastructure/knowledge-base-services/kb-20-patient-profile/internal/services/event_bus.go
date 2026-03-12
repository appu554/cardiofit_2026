package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/metrics"
	"kb-patient-profile/internal/models"
)

// EventHandler is a callback for event subscribers.
type EventHandler func(event models.Event)

// EventBus provides durable event publishing via a transactional outbox.
// Events are persisted to the event_outbox table atomically with the data
// mutation, then delivered to in-memory subscribers by a background poller.
type EventBus struct {
	db          *gorm.DB
	logger      *zap.Logger
	metrics     *metrics.Collector
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
	cancel      context.CancelFunc
	done        chan struct{}
}

// NewEventBus creates an event bus backed by a transactional outbox table.
func NewEventBus(db *gorm.DB, logger *zap.Logger, metricsCollector *metrics.Collector) *EventBus {
	return &EventBus{
		db:          db,
		logger:      logger,
		metrics:     metricsCollector,
		subscribers: make(map[string][]EventHandler),
		done:        make(chan struct{}),
	}
}

// Subscribe registers a handler for a specific event type.
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

// Publish writes an event to the outbox table using the default DB connection.
// The event is NOT delivered to subscribers immediately — the background
// poller handles delivery. This preserves the same call signature as before.
func (eb *EventBus) Publish(eventType string, patientID string, payload interface{}) {
	eb.PublishTx(eb.db, eventType, patientID, payload)
}

// PublishTx writes an event to the outbox table within the given transaction.
// Use this when you need the event to be atomic with other DB operations.
func (eb *EventBus) PublishTx(tx *gorm.DB, eventType string, patientID string, payload interface{}) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		eb.logger.Error("Failed to marshal event payload",
			zap.String("type", eventType),
			zap.Error(err))
		return
	}

	entry := models.EventOutboxEntry{
		ID:        uuid.New(),
		EventType: eventType,
		PatientID: patientID,
		Payload:   payloadJSON,
		CreatedAt: time.Now().UTC(),
	}

	if err := tx.Create(&entry).Error; err != nil {
		eb.logger.Error("Failed to write event to outbox",
			zap.String("type", eventType),
			zap.String("patient_id", patientID),
			zap.Error(err))
		return
	}

	eb.metrics.EventsPublished.WithLabelValues(eventType).Inc()
	eb.logger.Info("Event written to outbox",
		zap.String("type", eventType),
		zap.String("patient_id", patientID))
}

// StartPoller launches the background goroutine that polls for unpublished
// events and delivers them to subscribers. Call this once during startup.
func (eb *EventBus) StartPoller(ctx context.Context) {
	pollCtx, cancel := context.WithCancel(ctx)
	eb.cancel = cancel

	go func() {
		defer close(eb.done)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-pollCtx.Done():
				eb.logger.Info("Event outbox poller shutting down")
				return
			case <-ticker.C:
				eb.pollAndDeliver()
			}
		}
	}()

	eb.logger.Info("Event outbox poller started (1s interval)")
}

// Stop gracefully shuts down the outbox poller.
func (eb *EventBus) Stop() {
	if eb.cancel != nil {
		eb.cancel()
		<-eb.done
	}
}

// pollAndDeliver reads unpublished events and delivers to subscribers.
func (eb *EventBus) pollAndDeliver() {
	var entries []models.EventOutboxEntry
	err := eb.db.Where("published_at IS NULL").
		Order("created_at ASC").
		Limit(100).
		Find(&entries).Error
	if err != nil {
		eb.logger.Error("Failed to poll outbox", zap.Error(err))
		return
	}

	if len(entries) == 0 {
		return
	}

	for _, entry := range entries {
		event := models.Event{
			EventType: entry.EventType,
			PatientID: entry.PatientID,
			Timestamp: entry.CreatedAt,
			Payload:   entry.Payload,
		}

		// Deliver to subscribers
		eb.mu.RLock()
		handlers := eb.subscribers[entry.EventType]
		eb.mu.RUnlock()

		for _, handler := range handlers {
			handler(event)
		}

		// Mark as published
		now := time.Now().UTC()
		if err := eb.db.Model(&entry).Update("published_at", now).Error; err != nil {
			eb.logger.Error("Failed to mark event as published",
				zap.String("id", entry.ID.String()),
				zap.Error(err))
		}
	}

	eb.logger.Debug("Outbox poller delivered events", zap.Int("count", len(entries)))
}
