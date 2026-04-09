package services

import (
	"encoding/json"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// RenalEvent — event payload published to Kafka for downstream consumers
// ---------------------------------------------------------------------------

// RenalEvent represents a renal-related clinical event published to Kafka
// for consumption by KB-23 (decision cards) and V-MCU (safety channel).
type RenalEvent struct {
	PatientID   string    `json:"patient_id"`
	EventType   string    `json:"event_type"`   // RENAL_RAPID_DECLINE, RENAL_THRESHOLD_APPROACHING
	Severity    string    `json:"severity"`      // CRITICAL, WARNING
	EGFR        float64   `json:"egfr"`
	Slope       float64   `json:"slope"`
	Details     string    `json:"details"`
	PublishedAt time.Time `json:"published_at"`
}

// ---------------------------------------------------------------------------
// EventPublisher — interface for Kafka (or any message broker) publishing
// ---------------------------------------------------------------------------

// EventPublisher abstracts message publishing so tests can use a mock.
type EventPublisher interface {
	Publish(topic string, key string, payload []byte) error
}

// ---------------------------------------------------------------------------
// RenalEventPublisher — evaluates trajectory and publishes renal events
// ---------------------------------------------------------------------------

// RenalEventPublisher evaluates eGFR trajectory results and publishes
// clinically significant renal events to Kafka.
type RenalEventPublisher struct {
	publisher EventPublisher
	topic     string
}

// NewRenalEventPublisher creates a publisher bound to the given topic.
func NewRenalEventPublisher(publisher EventPublisher, topic string) *RenalEventPublisher {
	return &RenalEventPublisher{
		publisher: publisher,
		topic:     topic,
	}
}

// ---------------------------------------------------------------------------
// Drug-class eGFR thresholds for approaching-threshold alerts
// ---------------------------------------------------------------------------

type drugThreshold struct {
	DrugClass string
	Threshold float64
}

var renalDrugThresholds = []drugThreshold{
	{DrugClass: "METFORMIN", Threshold: 30},
	{DrugClass: "SULFONYLUREA", Threshold: 30},
	{DrugClass: "MRA", Threshold: 30},
	{DrugClass: "FINERENONE", Threshold: 25},
	{DrugClass: "SGLT2i", Threshold: 20},
}

// ---------------------------------------------------------------------------
// EvaluateAndPublish — core evaluation logic
// ---------------------------------------------------------------------------

// EvaluateAndPublish examines the trajectory result and publishes events when:
//  1. IsRapidDecliner is true → RENAL_RAPID_DECLINE (CRITICAL)
//  2. For each drug threshold, if current eGFR > threshold and projected time
//     to reach threshold is ≤ 12 months → RENAL_THRESHOLD_APPROACHING
//     (CRITICAL if ≤ 3 months, WARNING otherwise)
func (r *RenalEventPublisher) EvaluateAndPublish(patientID string, trajectory *EGFRTrajectoryResult) error {
	now := time.Now().UTC()

	// 1. Rapid decline event
	if trajectory.IsRapidDecliner {
		evt := RenalEvent{
			PatientID:   patientID,
			EventType:   "RENAL_RAPID_DECLINE",
			Severity:    "CRITICAL",
			EGFR:        trajectory.LatestEGFR,
			Slope:       trajectory.Slope,
			Details:     fmt.Sprintf("eGFR slope %.1f mL/min/1.73m²/year (rapid decline threshold: -5.0)", trajectory.Slope),
			PublishedAt: now,
		}
		if err := r.publishEvent(patientID, evt); err != nil {
			return fmt.Errorf("publish rapid decline event: %w", err)
		}
	}

	// 2. Threshold-approaching events for each drug class
	for _, dt := range renalDrugThresholds {
		if trajectory.LatestEGFR <= dt.Threshold {
			continue // already at or below threshold — not "approaching"
		}

		months := ProjectTimeToThreshold(trajectory.LatestEGFR, trajectory.Slope, dt.Threshold)
		if months == nil {
			continue // stable or improving — won't cross
		}

		if *months > 12.0 {
			continue // too far out to alert
		}

		severity := "WARNING"
		if *months <= 3.0 {
			severity = "CRITICAL"
		}

		evt := RenalEvent{
			PatientID:   patientID,
			EventType:   "RENAL_THRESHOLD_APPROACHING",
			Severity:    severity,
			EGFR:        trajectory.LatestEGFR,
			Slope:       trajectory.Slope,
			Details:     fmt.Sprintf("%s threshold %.0f: projected crossing in %.1f months", dt.DrugClass, dt.Threshold, *months),
			PublishedAt: now,
		}
		if err := r.publishEvent(patientID, evt); err != nil {
			return fmt.Errorf("publish threshold event for %s: %w", dt.DrugClass, err)
		}
	}

	return nil
}

// publishEvent serialises and sends a single event.
func (r *RenalEventPublisher) publishEvent(key string, evt RenalEvent) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal renal event: %w", err)
	}
	return r.publisher.Publish(r.topic, key, payload)
}
