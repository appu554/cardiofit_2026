package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Intake-Onboarding Prometheus metrics -- 10 metrics from spec section 7.4.

var (
	// EnrollmentsTotal counts enrollment attempts by tenant, channel, and status.
	EnrollmentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "intake_enrollments_total",
			Help: "Total enrollment attempts by tenant, channel type, and outcome status",
		},
		[]string{"tenant_id", "channel_type", "status"},
	)

	// SlotFillsTotal counts slot fill operations by slot name, extraction mode, and confidence tier.
	SlotFillsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "intake_slot_fills_total",
			Help: "Total slot fill operations by slot, extraction mode, and confidence tier",
		},
		[]string{"slot_name", "extraction_mode", "confidence_tier"},
	)

	// SafetyTriggersTotal counts safety rule triggers by rule ID and type.
	SafetyTriggersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "intake_safety_triggers_total",
			Help: "Total safety rule triggers (HARD_STOP and SOFT_FLAG)",
		},
		[]string{"rule_id", "rule_type", "tenant_id"},
	)

	// SessionDuration tracks the duration of intake sessions.
	SessionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "intake_session_duration_seconds",
			Help:    "Duration of intake sessions from start to completion or abandonment",
			Buckets: []float64{60, 300, 600, 1800, 3600, 7200, 14400},
		},
		[]string{"channel_type", "flow_type"},
	)

	// NLULatency tracks NLU extraction latency by mode and confidence tier.
	NLULatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "intake_nlu_latency_seconds",
			Help:    "Latency of NLU extraction operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0},
		},
		[]string{"extraction_mode", "confidence_tier"},
	)

	// PharmacistReviewQueueDepth tracks the current review queue depth.
	PharmacistReviewQueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "intake_pharmacist_review_queue_depth",
			Help: "Current depth of the pharmacist review queue",
		},
		[]string{"tenant_id", "risk_stratum"},
	)

	// WhatsAppDeliveryRate tracks WhatsApp message delivery outcomes.
	WhatsAppDeliveryRate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "intake_whatsapp_delivery_rate",
			Help: "WhatsApp message delivery outcomes by message type",
		},
		[]string{"message_type"},
	)

	// OfflineQueueDepth tracks ASHA offline queue depth.
	OfflineQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "intake_offline_queue_depth",
			Help: "Number of messages pending in the ASHA offline sync queue",
		},
	)

	// SessionLockContention counts Redis lock contention events.
	SessionLockContention = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "intake_session_lock_contention",
			Help: "Number of Redis session lock contention events",
		},
	)

	// CheckinTrajectoryTotal counts check-in trajectory signals.
	CheckinTrajectoryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "intake_checkin_trajectory_total",
			Help: "Total check-in trajectory signals by outcome and tenant",
		},
		[]string{"trajectory", "tenant_id"},
	)
)

// ConfidenceTier maps a confidence score to a tier label for metric cardinality control.
func ConfidenceTier(confidence float64) string {
	switch {
	case confidence >= 0.90:
		return "high"
	case confidence >= 0.70:
		return "medium"
	default:
		return "low"
	}
}
