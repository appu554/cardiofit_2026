package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// CGMAnalyticsEventPayload mirrors the JSON wire format emitted by the
// Flink Module3_CGMStreamJob.serializeAnalyticsEvent helper. Keeping
// this struct narrow on the Go side so a Flink-side schema addition
// (new AGP percentile field, etc.) doesn't break KB-26 deserialization
// — json.Unmarshal tolerates unknown fields by default.
//
// Phase 7 P7-E Milestone 1: this struct is the read-only projection
// used by the log-only consumer. Milestone 2 extends it with the
// persistence fields needed by cgm_period_reports.
type CGMAnalyticsEventPayload struct {
	EventType      string  `json:"event_type"`
	EventVersion   string  `json:"event_version"`
	PatientID      string  `json:"patient_id"`
	ComputedAtMs   int64   `json:"computed_at_ms"`
	WindowEndMs    int64   `json:"window_end_ms"`
	WindowDays     int     `json:"window_days"`
	TotalReadings  int     `json:"total_readings"`
	CoveragePct    float64 `json:"coverage_pct"`
	SufficientData bool    `json:"sufficient_data"`
	ConfidenceLvl  string  `json:"confidence_level"`
	MeanGlucose    float64 `json:"mean_glucose"`
	SDGlucose      float64 `json:"sd_glucose"`
	CVPct          float64 `json:"cv_pct"`
	GlucoseStable  bool    `json:"glucose_stable"`
	TIRPct         float64 `json:"tir_pct"`
	TBRL1Pct       float64 `json:"tbr_l1_pct"`
	TBRL2Pct       float64 `json:"tbr_l2_pct"`
	TARL1Pct       float64 `json:"tar_l1_pct"`
	TARL2Pct       float64 `json:"tar_l2_pct"`
	GMI            float64 `json:"gmi"`
	GRI            float64 `json:"gri"`
	GRIZone        string  `json:"gri_zone"`

	SustainedHypoDetected       bool `json:"sustained_hypo_detected"`
	SustainedSevereHypoDetected bool `json:"sustained_severe_hypo_detected"`
	SustainedHyperDetected      bool `json:"sustained_hyper_detected"`
	NocturnalHypoDetected       bool `json:"nocturnal_hypo_detected"`
	RapidRiseDetected           bool `json:"rapid_rise_detected"`
	RapidFallDetected           bool `json:"rapid_fall_detected"`
}

// CGMAnalyticsHandler is the per-event callback the consumer invokes
// for each parsed CGMAnalyticsEventPayload. Phase 7 P7-E Milestone 1
// uses a log-only handler; Milestone 2 wires this to a repository
// that persists the event into cgm_period_reports.
type CGMAnalyticsHandler func(ctx context.Context, evt CGMAnalyticsEventPayload) error

// CGMAnalyticsConsumer is the Kafka consumer for the
// clinical.cgm-analytics.v1 topic emitted by Flink's Module3_CGMStreamJob.
//
// Follows the same shape as SignalConsumer — kafka-go Reader with a
// single consume loop, FetchMessage + CommitMessages for at-least-once
// delivery, per-message handler callback. The consumer is feature-
// flagged behind KB26_KAFKA_ENABLED (same flag as the P7-F trajectory
// publisher and the existing SignalConsumer) so enabling Kafka
// activates or disables all KB-26 Kafka surface at once.
type CGMAnalyticsConsumer struct {
	reader  *kafka.Reader
	log     *zap.Logger
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewCGMAnalyticsConsumer creates a Kafka consumer for
// clinical.cgm-analytics.v1. The consumer group ID kb26-cgm-analytics
// matches the ops-script convention documented in
// runtime-layer/config/kafka-topics.yaml under kb_services.
func NewCGMAnalyticsConsumer(brokers []string, log *zap.Logger) *CGMAnalyticsConsumer {
	return &CGMAnalyticsConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          "clinical.cgm-analytics.v1",
			GroupID:        "kb26-cgm-analytics",
			MinBytes:       1,
			MaxBytes:       10485760,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		}),
		log:  log,
		done: make(chan struct{}),
	}
}

// Start launches the consume goroutine. The handler callback is
// invoked once per successfully-parsed event; a handler error is
// logged but does not abort the loop (per-message isolation — one
// bad record can't poison the whole stream).
func (c *CGMAnalyticsConsumer) Start(ctx context.Context, handler CGMAnalyticsHandler) {
	consumerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.consumeLoop(consumerCtx, handler)

	c.log.Info("KB-26 CGM analytics consumer started",
		zap.String("topic", "clinical.cgm-analytics.v1"),
		zap.String("group", "kb26-cgm-analytics"))
}

func (c *CGMAnalyticsConsumer) consumeLoop(ctx context.Context, handler CGMAnalyticsHandler) {
	defer close(c.done)
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.log.Error("CGM analytics Kafka fetch error", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		evt, parseErr := ParseCGMAnalyticsEvent(msg.Value)
		if parseErr != nil {
			c.log.Warn("CGM analytics event parse failed",
				zap.String("raw", string(msg.Value)),
				zap.Error(parseErr))
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				c.log.Warn("Failed to commit malformed CGM event",
					zap.Error(commitErr))
			}
			continue
		}

		if handler != nil {
			if handlerErr := handler(ctx, evt); handlerErr != nil {
				c.log.Error("CGM analytics handler failed",
					zap.String("patient_id", evt.PatientID),
					zap.Error(handlerErr))
			}
		}

		if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
			c.log.Warn("Failed to commit CGM analytics message", zap.Error(commitErr))
		}
	}
}

// Stop gracefully shuts down the consumer. Safe to call multiple times.
func (c *CGMAnalyticsConsumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.reader != nil {
		_ = c.reader.Close()
	}
}

// ParseCGMAnalyticsEvent is the exported pure helper that deserialises
// a Kafka record value into a CGMAnalyticsEventPayload. Exported so
// tests can verify the wire contract without spinning up a real Kafka
// reader.
func ParseCGMAnalyticsEvent(data []byte) (CGMAnalyticsEventPayload, error) {
	var evt CGMAnalyticsEventPayload
	if err := json.Unmarshal(data, &evt); err != nil {
		return CGMAnalyticsEventPayload{}, err
	}
	return evt, nil
}

// LogOnlyCGMAnalyticsHandler is the Milestone 1 default handler — it
// logs the parsed event at INFO level and returns nil. Milestone 2
// replaces this with a repository-backed handler that persists the
// event into cgm_period_reports.
func LogOnlyCGMAnalyticsHandler(log *zap.Logger) CGMAnalyticsHandler {
	return func(ctx context.Context, evt CGMAnalyticsEventPayload) error {
		log.Info("CGM analytics event received (log-only mode)",
			zap.String("patient_id", evt.PatientID),
			zap.String("event_version", evt.EventVersion),
			zap.Int("window_days", evt.WindowDays),
			zap.Int("total_readings", evt.TotalReadings),
			zap.Float64("tir_pct", evt.TIRPct),
			zap.Float64("mean_glucose", evt.MeanGlucose),
			zap.String("gri_zone", evt.GRIZone),
			zap.Int64("window_end_ms", evt.WindowEndMs))
		return nil
	}
}
