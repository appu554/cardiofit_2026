// Package governance tests for Tier-7 governance event emission
package governance

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-16-lab-interpretation/pkg/types"
)

// =============================================================================
// EVENT CREATION TESTS
// =============================================================================

func TestNewGovernanceEvent(t *testing.T) {
	event := NewGovernanceEvent(EventCriticalLabValue, "patient-123")

	assert.NotEqual(t, uuid.Nil, event.ID, "Event should have valid UUID")
	assert.Equal(t, EventCriticalLabValue, event.EventType)
	assert.Equal(t, "patient-123", event.PatientID)
	assert.Equal(t, "KB-16", event.Source)
	assert.Equal(t, StatusPending, event.Status)
	assert.NotNil(t, event.Payload)
	assert.NotZero(t, event.Timestamp)
}

func TestCriticalLabEvent(t *testing.T) {
	event := CriticalLabEvent(
		"patient-123",
		"result-456",
		"2823-3",
		"Potassium",
		6.8,
		"mmol/L",
		"CRITICAL_HIGH",
	)

	assert.Equal(t, EventCriticalLabValue, event.EventType)
	assert.Equal(t, "patient-123", event.PatientID)
	assert.Equal(t, SeverityCritical, event.Severity)
	assert.Equal(t, 1, event.Priority)
	assert.True(t, event.RequiresAcknowledgment)
	assert.True(t, event.RequiresReview)
	assert.Equal(t, 30, event.AcknowledgmentSLAMin)
	assert.Equal(t, 60, event.ReviewSLAMin)
	assert.Equal(t, "attending_physician", event.EscalationPath)

	// Check payload
	assert.Equal(t, "2823-3", event.Payload["lab_code"])
	assert.Equal(t, "Potassium", event.Payload["lab_name"])
	assert.Equal(t, 6.8, event.Payload["value"])
	assert.Equal(t, "mmol/L", event.Payload["unit"])
}

func TestPanicLabEvent(t *testing.T) {
	event := PanicLabEvent(
		"patient-123",
		"result-456",
		"2823-3",
		"Potassium",
		7.2,
		"mmol/L",
		"PANIC_HIGH",
	)

	assert.Equal(t, EventPanicLabValue, event.EventType)
	assert.Equal(t, SeverityCritical, event.Severity)
	assert.Equal(t, 1, event.Priority)
	assert.Equal(t, 15, event.AcknowledgmentSLAMin, "Panic should have tighter SLA")
	assert.Equal(t, 30, event.ReviewSLAMin)
	assert.Equal(t, "rapid_response", event.EscalationPath)
	assert.Equal(t, true, event.Payload["immediate_action_required"])
}

func TestSignificantDeltaEvent(t *testing.T) {
	event := SignificantDeltaEvent(
		"patient-123",
		"result-456",
		"718-7",
		"Hemoglobin",
		8.5,
		12.0,
		-29.2,
		"g/dL",
	)

	assert.Equal(t, EventSignificantDeltaLab, event.EventType)
	assert.Equal(t, SeverityHigh, event.Severity)
	assert.Equal(t, 2, event.Priority)
	assert.Equal(t, 60, event.AcknowledgmentSLAMin)
	assert.Equal(t, "decreased", event.Payload["direction"])
	assert.Equal(t, -29.2, event.Payload["percent_change"])
}

func TestClinicalPatternEvent(t *testing.T) {
	event := ClinicalPatternEvent(
		"patient-123",
		"SEPSIS_INDICATOR",
		"Sepsis Pattern",
		0.92,
		SeverityCritical,
	)

	assert.Equal(t, EventClinicalPattern, event.EventType)
	assert.Equal(t, SeverityCritical, event.Severity)
	assert.Equal(t, 1, event.Priority)
	assert.True(t, event.RequiresAcknowledgment)
	assert.Equal(t, 0.92, event.Payload["confidence"])
}

func TestWorseningTrendEvent(t *testing.T) {
	event := WorseningTrendEvent(
		"patient-123",
		"2160-0",
		"Creatinine",
		"worsening",
		0.15,
		"mg/dL",
	)

	assert.Equal(t, EventWorseningTrend, event.EventType)
	assert.Equal(t, SeverityMedium, event.Severity)
	assert.Equal(t, 3, event.Priority)
	assert.Equal(t, 240, event.ReviewSLAMin)
	assert.Equal(t, 0.15, event.Payload["rate_of_change"])
}

func TestCareGapEvent(t *testing.T) {
	// Test with moderate overdue
	event := CareGapEvent(
		"patient-123",
		"HBA1C_MONITORING",
		"HbA1c Monitoring",
		45,
	)

	assert.Equal(t, EventCareGapIdentified, event.EventType)
	assert.Equal(t, SeverityMedium, event.Severity)
	assert.Equal(t, 45, event.Payload["days_overdue"])

	// Test with severely overdue (>90 days)
	eventSevere := CareGapEvent(
		"patient-123",
		"HBA1C_MONITORING",
		"HbA1c Monitoring",
		120,
	)

	assert.Equal(t, SeverityHigh, eventSevere.Severity, "Should be high severity when >90 days overdue")
}

// =============================================================================
// SERIALIZATION TESTS
// =============================================================================

func TestEventSerialization(t *testing.T) {
	event := CriticalLabEvent(
		"patient-123",
		"result-456",
		"2823-3",
		"Potassium",
		6.8,
		"mmol/L",
		"CRITICAL_HIGH",
	)

	// Add provenance
	event.Provenance = EventProvenance{
		InterpretationVersion: "1.0.0",
		Timestamp:             time.Now().UTC(),
		KB8Calculations: []KB8Calculation{
			{
				Calculator: "anion_gap",
				Input:      map[string]interface{}{"na": 140.0, "cl": 100.0, "hco3": 24.0},
				Output:     map[string]interface{}{"value": 16.0},
				Formula:    "Na - (Cl + HCO3)",
				Version:    "1.0.0",
			},
		},
	}

	// Serialize
	data, err := event.ToJSON()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Deserialize
	restored, err := FromJSON(data)
	require.NoError(t, err)

	assert.Equal(t, event.ID, restored.ID)
	assert.Equal(t, event.EventType, restored.EventType)
	assert.Equal(t, event.PatientID, restored.PatientID)
	assert.Equal(t, event.Severity, restored.Severity)
	assert.Len(t, restored.Provenance.KB8Calculations, 1)
}

func TestEventProvenanceTracking(t *testing.T) {
	builder := NewProvenanceBuilder("1.0.0")

	builder.AddKB8Calculation(
		"egfr_ckd_epi_2021",
		map[string]interface{}{"creatinine": 1.5, "age": 65, "sex": "female"},
		map[string]interface{}{"egfr": 38.5, "stage": "3b"},
		"CKD-EPI 2021",
		"2021",
	)

	low := 0.6
	high := 1.2
	builder.AddReferenceRange(
		"2160-0",
		"CAP",
		"2023",
		&low,
		&high,
		nil,
		nil,
		true,
		true,
	)

	builder.AddRule(
		"ckd_progression",
		"CKD Progression Alert",
		"1.0",
		true,
		0.95,
	)

	provenance := builder.Build()

	assert.Equal(t, "1.0.0", provenance.InterpretationVersion)
	assert.Len(t, provenance.KB8Calculations, 1)
	assert.Equal(t, "egfr_ckd_epi_2021", provenance.KB8Calculations[0].Calculator)
	assert.Len(t, provenance.ReferenceRanges, 1)
	assert.Equal(t, "CAP", provenance.ReferenceRanges[0].Source)
	assert.Len(t, provenance.RulesApplied, 1)
	assert.True(t, provenance.RulesApplied[0].Triggered)
}

// =============================================================================
// PUBLISHER TESTS
// =============================================================================

func TestPublisherConfig(t *testing.T) {
	config := DefaultPublisherConfig()

	assert.True(t, config.RedisEnabled)
	assert.Equal(t, "kb16:governance:critical", config.CriticalChannel)
	assert.Equal(t, "kb16:governance:events", config.StandardChannel)
	assert.Equal(t, "kb16:governance:audit", config.AuditChannel)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100, config.BufferSize)
	assert.True(t, config.AsyncPublish)
	assert.True(t, config.AuditEnabled)
}

func TestPublisherWithoutRedis(t *testing.T) {
	log := logrus.New().WithField("test", "governance")
	config := DefaultPublisherConfig()
	config.RedisEnabled = false

	publisher := NewPublisher(config, nil, log)
	require.NotNil(t, publisher)

	ctx := context.Background()
	event := CriticalLabEvent("patient-1", "result-1", "2823-3", "Potassium", 6.8, "mmol/L", "CRITICAL_HIGH")

	// Should not error when Redis is disabled
	err := publisher.Publish(ctx, event)
	assert.NoError(t, err)

	metrics := publisher.GetMetrics()
	assert.Equal(t, int64(0), metrics.EventsPublished, "Events not published to Redis when disabled")
}

func TestEventIsCriticalOrPanic(t *testing.T) {
	tests := []struct {
		name     string
		event    *GovernanceEvent
		expected bool
	}{
		{
			name:     "Panic event is critical",
			event:    PanicLabEvent("p1", "r1", "code", "name", 1.0, "unit", "PANIC"),
			expected: true,
		},
		{
			name:     "Critical event is critical",
			event:    CriticalLabEvent("p1", "r1", "code", "name", 1.0, "unit", "CRITICAL"),
			expected: true,
		},
		{
			name:     "Delta event is not critical",
			event:    SignificantDeltaEvent("p1", "r1", "code", "name", 1.0, 2.0, 50.0, "unit"),
			expected: false,
		},
		{
			name:     "Trend event is not critical",
			event:    WorseningTrendEvent("p1", "code", "name", "worsening", 0.1, "unit"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.event.IsCriticalOrPanic())
		})
	}
}

// =============================================================================
// OBSERVER TESTS
// =============================================================================

func TestObserverConfig(t *testing.T) {
	config := DefaultObserverConfig()

	assert.True(t, config.CriticalLabEvents)
	assert.True(t, config.PanicLabEvents)
	assert.True(t, config.DeltaCheckEvents)
	assert.True(t, config.PatternDetection)
	assert.True(t, config.TrendingEvents)
	assert.True(t, config.CareGapEvents)

	assert.Equal(t, 15, config.PanicAckSLAMin)
	assert.Equal(t, 30, config.CriticalAckSLAMin)
	assert.Equal(t, "rapid_response", config.PanicEscalation)
}

func TestObserverOnInterpretation(t *testing.T) {
	log := logrus.New().WithField("test", "observer")
	config := DefaultPublisherConfig()
	config.RedisEnabled = false

	publisher := NewPublisher(config, nil, log)
	observer := NewInterpretationObserver(publisher, log)

	ctx := context.Background()
	value := 7.5 // Panic high potassium

	result := &types.InterpretedResult{
		Result: types.LabResult{
			ID:           uuid.New(),
			PatientID:    "patient-123",
			Code:         "2823-3",
			Name:         "Potassium",
			ValueNumeric: &value,
			Unit:         "mmol/L",
			CollectedAt:  time.Now(),
		},
		Interpretation: types.Interpretation{
			Flag:            types.FlagPanicHigh,
			Severity:        types.SeverityCritical,
			IsCritical:      true,
			IsPanic:         true,
			RequiresAction:  true,
			ClinicalComment: "PANIC HIGH: Potassium is critically high",
			Recommendations: []types.Recommendation{
				{Type: "action", Priority: "HIGH", Description: "Order ECG"},
				{Type: "action", Priority: "MEDIUM", Description: "Review medications"},
			},
		},
	}

	err := observer.OnInterpretation(ctx, result, nil)
	assert.NoError(t, err)
}

func TestObserverOnTrendDetected(t *testing.T) {
	log := logrus.New().WithField("test", "observer")
	config := DefaultPublisherConfig()
	config.RedisEnabled = false

	publisher := NewPublisher(config, nil, log)
	observer := NewInterpretationObserver(publisher, log)

	ctx := context.Background()

	// Test worsening trend
	trend := &types.TrendAnalysis{
		Trajectory:   types.TrajectoryWorsening,
		RateOfChange: 0.15,
		Windows: map[string]types.TrendWindow{
			"7d": {
				Name:     "7 days",
				Days:     7,
				Slope:    0.02,
				RSquared: 0.85,
				DataPoints: []types.DataPoint{
					{Timestamp: time.Now().Add(-6 * 24 * time.Hour), Value: 1.2},
					{Timestamp: time.Now().Add(-4 * 24 * time.Hour), Value: 1.4},
					{Timestamp: time.Now().Add(-2 * 24 * time.Hour), Value: 1.5},
					{Timestamp: time.Now().Add(-1 * 24 * time.Hour), Value: 1.6},
					{Timestamp: time.Now(), Value: 1.8},
				},
			},
		},
	}

	err := observer.OnTrendDetected(ctx, "patient-123", "2160-0", "Creatinine", trend, nil)
	assert.NoError(t, err)

	// Test stable trend (should not emit)
	stableTrend := &types.TrendAnalysis{
		Trajectory:   types.TrajectoryStable,
		RateOfChange: 0.01,
	}

	err = observer.OnTrendDetected(ctx, "patient-123", "2160-0", "Creatinine", stableTrend, nil)
	assert.NoError(t, err)
}

func TestObserverOnCareGapIdentified(t *testing.T) {
	log := logrus.New().WithField("test", "observer")
	config := DefaultPublisherConfig()
	config.RedisEnabled = false

	publisher := NewPublisher(config, nil, log)
	observer := NewInterpretationObserver(publisher, log)

	ctx := context.Background()

	err := observer.OnCareGapIdentified(ctx, "patient-123", "HBA1C_MONITORING", "HbA1c Test", 45, nil)
	assert.NoError(t, err)
}

// =============================================================================
// SEVERITY MAPPING TESTS
// =============================================================================

func TestSeverityToPriority(t *testing.T) {
	tests := []struct {
		severity Severity
		expected int
	}{
		{SeverityCritical, 1},
		{SeverityHigh, 2},
		{SeverityMedium, 3},
		{SeverityLow, 4},
		{SeverityInfo, 5},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			assert.Equal(t, tt.expected, severityToPriority(tt.severity))
		})
	}
}

// =============================================================================
// STATUS TRANSITION TESTS
// =============================================================================

func TestEventStatusTransitions(t *testing.T) {
	event := NewGovernanceEvent(EventCriticalLabValue, "patient-123")

	assert.Equal(t, StatusPending, event.Status)

	// Simulate acknowledgment
	event.Status = StatusAcknowledged
	event.StatusHistory = append(event.StatusHistory, StatusChange{
		From:      StatusPending,
		To:        StatusAcknowledged,
		ChangedAt: time.Now().UTC(),
		ChangedBy: "nurse_jane",
		Reason:    "Acknowledged via phone call",
	})

	assert.Equal(t, StatusAcknowledged, event.Status)
	assert.Len(t, event.StatusHistory, 1)
	assert.Equal(t, "nurse_jane", event.StatusHistory[0].ChangedBy)
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

func BenchmarkEventCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CriticalLabEvent(
			"patient-123",
			"result-456",
			"2823-3",
			"Potassium",
			6.8,
			"mmol/L",
			"CRITICAL_HIGH",
		)
	}
}

func BenchmarkEventSerialization(b *testing.B) {
	event := CriticalLabEvent(
		"patient-123",
		"result-456",
		"2823-3",
		"Potassium",
		6.8,
		"mmol/L",
		"CRITICAL_HIGH",
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = event.ToJSON()
	}
}

func BenchmarkEventDeserialization(b *testing.B) {
	event := CriticalLabEvent(
		"patient-123",
		"result-456",
		"2823-3",
		"Potassium",
		6.8,
		"mmol/L",
		"CRITICAL_HIGH",
	)
	data, _ := json.Marshal(event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FromJSON(data)
	}
}
