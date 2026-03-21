package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/cardiofit/shared/signals"
	"kb-patient-profile/internal/metrics"
	"kb-patient-profile/internal/models"
)

// KafkaWriter is an interface for writing Kafka messages (mockable in tests).
type KafkaWriter interface {
	WriteMessage(ctx context.Context, topic, key string, value []byte) error
	Close() error
}

// KafkaGoWriter is the production KafkaWriter using segmentio/kafka-go.
type KafkaGoWriter struct {
	writers map[string]*kafka.Writer
}

// NewKafkaGoWriter creates per-topic writers for the clinical signal topics.
func NewKafkaGoWriter(brokers []string, clientID string) *KafkaGoWriter {
	transport := &kafka.Transport{
		DialTimeout: 10 * time.Second,
		ClientID:    clientID,
	}
	newWriter := func(topic string) *kafka.Writer {
		return &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: false,
			Async:                  false,
			Transport:              transport,
			RequiredAcks:           kafka.RequireAll,
			Compression:            kafka.Snappy,
			BatchSize:              100,
			BatchTimeout:           10 * time.Millisecond,
		}
	}
	return &KafkaGoWriter{
		writers: map[string]*kafka.Writer{
			"clinical.observations.v1":    newWriter("clinical.observations.v1"),
			"clinical.priority-events.v1": newWriter("clinical.priority-events.v1"),
			"clinical.state-changes.v1":   newWriter("clinical.state-changes.v1"),
		},
	}
}

func (w *KafkaGoWriter) WriteMessage(ctx context.Context, topic, key string, value []byte) error {
	writer, ok := w.writers[topic]
	if !ok {
		return fmt.Errorf("unknown topic %s — not in writer pool", topic)
	}
	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	})
}

func (w *KafkaGoWriter) Close() error {
	var errs []error
	for _, wr := range w.writers {
		if err := wr.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// KafkaOutboxRelay polls the event_outbox table for Kafka-unpublished rows,
// maps them to signal/state-change envelopes, and publishes to Kafka.
type KafkaOutboxRelay struct {
	db            *gorm.DB
	mapper        *EventSignalMapper
	writer        KafkaWriter
	metrics       *metrics.Collector
	signalMetrics *signals.SignalMetrics
	pollInterval  time.Duration
	batchSize     int
	log           *zap.Logger
	cancel        context.CancelFunc
	done          chan struct{}
}

// NewKafkaOutboxRelay creates a relay that bridges the KB-20 outbox to Kafka.
func NewKafkaOutboxRelay(
	db *gorm.DB,
	writer KafkaWriter,
	metricsCollector *metrics.Collector,
	log *zap.Logger,
) *KafkaOutboxRelay {
	return &KafkaOutboxRelay{
		db:           db,
		mapper:       NewEventSignalMapper(),
		writer:       writer,
		metrics:      metricsCollector,
		pollInterval: 1 * time.Second,
		batchSize:    50,
		log:          log,
		done:         make(chan struct{}),
	}
}

// SetSignalMetrics attaches optional shared signal pipeline metrics to the relay.
func (r *KafkaOutboxRelay) SetSignalMetrics(m *signals.SignalMetrics) {
	r.signalMetrics = m
}

// Start launches the background polling goroutine.
func (r *KafkaOutboxRelay) Start(ctx context.Context) {
	pollCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	go func() {
		defer close(r.done)
		ticker := time.NewTicker(r.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-pollCtx.Done():
				r.log.Info("Kafka outbox relay shutting down")
				return
			case <-ticker.C:
				r.pollAndPublish(pollCtx)
			}
		}
	}()

	r.log.Info("Kafka outbox relay started",
		zap.Duration("poll_interval", r.pollInterval),
		zap.Int("batch_size", r.batchSize),
	)
}

// Stop gracefully shuts down the relay.
func (r *KafkaOutboxRelay) Stop() {
	if r.cancel != nil {
		r.cancel()
		<-r.done
	}
}

func (r *KafkaOutboxRelay) pollAndPublish(ctx context.Context) {
	var entries []models.EventOutboxEntry
	err := r.db.Where("kafka_published_at IS NULL").
		Order("created_at ASC").
		Limit(r.batchSize).
		Find(&entries).Error
	if err != nil {
		r.log.Error("Failed to poll outbox for Kafka relay", zap.Error(err))
		return
	}
	if r.signalMetrics != nil {
		r.signalMetrics.OutboxRelayPendingCount.Set(float64(len(entries)))
	}
	if len(entries) == 0 {
		return
	}

	published := r.processEntries(ctx, entries)

	// Mark published entries
	now := time.Now().UTC()
	for _, id := range published {
		if err := r.db.Model(&models.EventOutboxEntry{}).
			Where("id = ?", id).
			Update("kafka_published_at", now).Error; err != nil {
			r.log.Error("Failed to mark kafka_published_at",
				zap.String("id", id.String()),
				zap.Error(err))
		}
	}

	r.log.Debug("Kafka outbox relay published events",
		zap.Int("total", len(entries)),
		zap.Int("published", len(published)),
	)
}

// processEntries maps and publishes entries. Returns IDs of successfully processed entries.
func (r *KafkaOutboxRelay) processEntries(ctx context.Context, entries []models.EventOutboxEntry) []uuid.UUID {
	var published []uuid.UUID

	for _, entry := range entries {
		mapped, err := r.mapper.Map(entry)
		if err != nil {
			r.log.Warn("Kafka relay: failed to map event",
				zap.String("event_type", entry.EventType),
				zap.String("id", entry.ID.String()),
				zap.Error(err))
			continue
		}

		if mapped == nil {
			// Unmapped event type — mark as published to clear backlog
			published = append(published, entry.ID)
			continue
		}

		if mapped.Signal != nil {
			if err := r.publishSignal(ctx, mapped.Signal); err != nil {
				r.log.Warn("Kafka relay: failed to publish signal",
					zap.String("signal_type", string(mapped.Signal.SignalType)),
					zap.Error(err))
				continue
			}
		}

		if mapped.StateChange != nil {
			if err := r.publishStateChange(ctx, mapped.StateChange); err != nil {
				r.log.Warn("Kafka relay: failed to publish state change",
					zap.String("change_type", mapped.StateChange.ChangeType),
					zap.Error(err))
				continue
			}
		}

		published = append(published, entry.ID)
	}

	return published
}

func (r *KafkaOutboxRelay) publishSignal(ctx context.Context, sig *MappedSignal) error {
	topic := "clinical.observations.v1"
	if sig.Priority {
		topic = "clinical.priority-events.v1"
	}

	envelope := map[string]interface{}{
		"event_id":    sig.EventID,
		"patient_id":  sig.PatientID,
		"signal_type": sig.SignalType,
		"priority":    sig.Priority,
		"measured_at": sig.Timestamp,
		"source":      "APP_MANUAL",
		"confidence":  1.0,
		"loinc_code":  sig.LOINCCode,
		"payload":     json.RawMessage(sig.Payload),
		"created_at":  time.Now().UTC(),
	}

	value, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	if err := r.writer.WriteMessage(ctx, topic, sig.PatientID, value); err != nil {
		return err
	}
	if r.signalMetrics != nil {
		r.signalMetrics.OutboxRelayPublishedTotal.WithLabelValues(topic).Inc()
	}
	return nil
}

func (r *KafkaOutboxRelay) publishStateChange(ctx context.Context, sc *MappedStateChange) error {
	envelope := map[string]interface{}{
		"event_id":    sc.EventID,
		"patient_id":  sc.PatientID,
		"change_type": sc.ChangeType,
		"timestamp":   sc.Timestamp,
		"payload":     json.RawMessage(sc.Payload),
		"created_at":  time.Now().UTC(),
	}

	value, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	if err := r.writer.WriteMessage(ctx, "clinical.state-changes.v1", sc.PatientID, value); err != nil {
		return err
	}
	if r.signalMetrics != nil {
		r.signalMetrics.OutboxRelayPublishedTotal.WithLabelValues("clinical.state-changes.v1").Inc()
	}
	return nil
}
