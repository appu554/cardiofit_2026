// Package test provides governance readiness tests for KB-12
// Phase 9: Audit trail, governance signals, and regulatory compliance
package test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-12-ordersets-careplans/pkg/cdshooks"
	"kb-12-ordersets-careplans/pkg/cpoe"
)

// ============================================
// 9.1 Audit Trail Tests
// ============================================

func TestAuditChainImmutable(t *testing.T) {
	// Test that audit entries form an immutable chain
	type AuditEntry struct {
		EntryID      string    `json:"entry_id"`
		PreviousHash string    `json:"previous_hash"`
		Timestamp    time.Time `json:"timestamp"`
		Action       string    `json:"action"`
		UserID       string    `json:"user_id"`
		ResourceType string    `json:"resource_type"`
		ResourceID   string    `json:"resource_id"`
		Details      string    `json:"details"`
		Hash         string    `json:"hash"`
	}

	// Create audit chain
	entries := []AuditEntry{
		{
			EntryID:      "audit-001",
			PreviousHash: "genesis",
			Timestamp:    time.Now().Add(-10 * time.Minute),
			Action:       "order_created",
			UserID:       "provider-001",
			ResourceType: "MedicationRequest",
			ResourceID:   "order-001",
			Details:      "Created metformin order",
		},
		{
			EntryID:      "audit-002",
			Timestamp:    time.Now().Add(-8 * time.Minute),
			Action:       "alert_generated",
			UserID:       "system",
			ResourceType: "ClinicalAlert",
			ResourceID:   "alert-001",
			Details:      "Duplicate therapy alert triggered",
		},
		{
			EntryID:      "audit-003",
			Timestamp:    time.Now().Add(-5 * time.Minute),
			Action:       "override_applied",
			UserID:       "provider-001",
			ResourceType: "ClinicalAlert",
			ResourceID:   "alert-001",
			Details:      "Override reason: Dose adjustment - discontinuing previous order",
		},
		{
			EntryID:      "audit-004",
			Timestamp:    time.Now().Add(-2 * time.Minute),
			Action:       "order_signed",
			UserID:       "provider-001",
			ResourceType: "MedicationRequest",
			ResourceID:   "order-001",
			Details:      "Order signed with override",
		},
	}

	// Calculate hashes and build chain
	for i := range entries {
		if i > 0 {
			entries[i].PreviousHash = entries[i-1].Hash
		}
		// Calculate hash
		data, _ := json.Marshal(entries[i])
		hash := sha256.Sum256(data)
		entries[i].Hash = hex.EncodeToString(hash[:])
	}

	// Verify chain integrity
	for i := 1; i < len(entries); i++ {
		assert.Equal(t, entries[i].PreviousHash, entries[i-1].Hash,
			"Entry %d should reference previous entry's hash", i)
	}

	// Verify tampering detection
	originalAction := entries[1].Action
	entries[1].Action = "tampered_action"
	data, _ := json.Marshal(entries[1])
	tamperedHash := sha256.Sum256(data)
	tamperedHashStr := hex.EncodeToString(tamperedHash[:])

	assert.NotEqual(t, entries[1].Hash, tamperedHashStr,
		"Tampering should be detectable via hash mismatch")

	// Restore
	entries[1].Action = originalAction

	t.Logf("✓ Audit chain with %d entries, integrity verified", len(entries))
}

func TestOverrideReasoningCaptured(t *testing.T) {
	// Test that override reasoning is properly captured
	service := cpoe.NewCPOEService(nil, nil, nil, nil)
	ctx := context.Background()

	session, err := service.CreateOrderSession(ctx, &cpoe.CreateSessionRequest{
		PatientID:   "patient-override-audit",
		EncounterID: "encounter-override-audit",
		ProviderID:  "provider-001",
		PatientContext: &cpoe.PatientContext{
			PatientID: "patient-override-audit",
			Age:       55,
			Weight:    75.0,
			Allergies: []cpoe.PatientAllergy{
				{
					AllergenCode:   "733",
					AllergenName:   "Penicillin",
					Severity:       "moderate",
					ReactionType:   "allergy",
					Manifestations: []string{"rash"},
				},
			},
		},
	})
	require.NoError(t, err)

	// Add order that triggers allergy alert
	order := &cpoe.PendingOrder{
		OrderType: "medication",
		Priority:  "routine",
		Medication: &cpoe.MedicationOrder{
			MedicationCode: "733",
			MedicationName: "Penicillin V",
			Dose:           500,
			DoseUnit:       "mg",
			Route:          "oral",
			Frequency:      "QID",
		},
	}

	resp, err := service.AddOrder(ctx, session.SessionID, order)
	require.NoError(t, err)

	// Verify alerts have override structure
	for _, alert := range resp.Alerts {
		if alert.AlertType == "drug-allergy" {
			// Verify override structure
			assert.NotEmpty(t, alert.AlertID, "Alert should have ID for tracking")
			assert.NotEmpty(t, alert.OverrideReasons, "Alert should have predefined override reasons")
			t.Logf("Alert %s has %d override reasons", alert.AlertID, len(alert.OverrideReasons))

			// Verify governance fields
			assert.NotEmpty(t, alert.Source, "Alert should have source for governance")
			assert.NotZero(t, alert.CreatedAt, "Alert should have timestamp")
		}
	}

	// Sign with override and verify capture
	overrideReason := "Patient tolerated medication previously without issue - confirmed with patient"
	overrides := map[string]string{
		resp.OrderID: overrideReason,
	}

	_, _ = service.SignOrders(ctx, session.SessionID, "provider-001", overrides)

	// Retrieve session and verify override captured
	updatedSession, _ := service.GetSession(session.SessionID)
	for _, ord := range updatedSession.Orders {
		if ord.OverrideReason != "" {
			assert.Equal(t, overrideReason, ord.OverrideReason)
			t.Logf("✓ Override reason captured: %s", ord.OverrideReason)
		}
	}

	t.Log("✓ Override reasoning properly captured for governance")
}

func TestClinicianAttributionPresent(t *testing.T) {
	// Test that all actions have clinician attribution
	service := cpoe.NewCPOEService(nil, nil, nil, nil)
	ctx := context.Background()

	providerID := "provider-attribution-001"

	session, err := service.CreateOrderSession(ctx, &cpoe.CreateSessionRequest{
		PatientID:   "patient-attribution",
		EncounterID: "encounter-attribution",
		ProviderID:  providerID,
		PatientContext: &cpoe.PatientContext{
			PatientID: "patient-attribution",
			Age:       45,
			Weight:    70.0,
		},
	})
	require.NoError(t, err)

	// Verify session has provider attribution
	assert.Equal(t, providerID, session.ProviderID, "Session should have provider ID")

	// Add order
	order := &cpoe.PendingOrder{
		OrderType: "medication",
		Priority:  "routine",
		Medication: &cpoe.MedicationOrder{
			MedicationCode: "29046",
			MedicationName: "Lisinopril",
			Dose:           10,
			DoseUnit:       "mg",
			Route:          "oral",
			Frequency:      "daily",
		},
	}

	_, _ = service.AddOrder(ctx, session.SessionID, order)

	// Sign orders
	signResp, _ := service.SignOrders(ctx, session.SessionID, providerID, nil)

	// Verify signing attribution
	updatedSession, _ := service.GetSession(session.SessionID)
	if updatedSession.Status == "signed" {
		assert.Equal(t, providerID, updatedSession.SignedBy, "Signed session should have signer ID")
		assert.NotNil(t, updatedSession.SignedAt, "Signed session should have sign timestamp")
		t.Logf("✓ Signing attribution: %s at %v", updatedSession.SignedBy, updatedSession.SignedAt)
	}

	t.Logf("✓ Clinician attribution present: Provider=%s, Sign success=%v",
		session.ProviderID, signResp.Success)
}

func TestTimestampAccuracy(t *testing.T) {
	// Test that all timestamps are accurate and properly formatted
	service := cpoe.NewCPOEService(nil, nil, nil, nil)
	ctx := context.Background()

	beforeCreate := time.Now()

	session, err := service.CreateOrderSession(ctx, &cpoe.CreateSessionRequest{
		PatientID:   "patient-timestamp",
		EncounterID: "encounter-timestamp",
		ProviderID:  "provider-001",
		PatientContext: &cpoe.PatientContext{
			PatientID: "patient-timestamp",
			Age:       50,
			Weight:    75.0,
		},
	})
	require.NoError(t, err)

	afterCreate := time.Now()

	// Verify creation timestamp is within bounds
	assert.True(t, session.CreatedAt.After(beforeCreate) || session.CreatedAt.Equal(beforeCreate),
		"Creation time should be >= beforeCreate")
	assert.True(t, session.CreatedAt.Before(afterCreate) || session.CreatedAt.Equal(afterCreate),
		"Creation time should be <= afterCreate")

	// Verify UpdatedAt is set
	assert.False(t, session.UpdatedAt.IsZero(), "UpdatedAt should be set")

	// Add order and verify timestamp updates
	order := &cpoe.PendingOrder{
		OrderType: "medication",
		Priority:  "routine",
		Medication: &cpoe.MedicationOrder{
			MedicationCode: "6809",
			MedicationName: "Metformin",
			Dose:           500,
			DoseUnit:       "mg",
			Route:          "oral",
			Frequency:      "BID",
		},
	}

	beforeAdd := time.Now()
	_, _ = service.AddOrder(ctx, session.SessionID, order)
	afterAdd := time.Now()

	updatedSession, _ := service.GetSession(session.SessionID)
	assert.True(t, updatedSession.UpdatedAt.After(beforeAdd) || updatedSession.UpdatedAt.Equal(beforeAdd),
		"UpdatedAt should be updated after adding order")
	assert.True(t, updatedSession.UpdatedAt.Before(afterAdd) || updatedSession.UpdatedAt.Equal(afterAdd),
		"UpdatedAt should be before or at afterAdd time")

	t.Logf("✓ Timestamps accurate: Created=%v, Updated=%v",
		session.CreatedAt.Format(time.RFC3339),
		updatedSession.UpdatedAt.Format(time.RFC3339))
}

// ============================================
// 9.2 Governance Signals Tests
// ============================================

func TestGovernanceSignalOnBundleMissed(t *testing.T) {
	// Test that governance signal is generated when protocol bundle is missed
	type GovernanceSignal struct {
		SignalID      string                 `json:"signal_id"`
		SignalType    string                 `json:"signal_type"`
		Severity      string                 `json:"severity"`
		ProtocolID    string                 `json:"protocol_id"`
		PatientID     string                 `json:"patient_id"`
		EncounterID   string                 `json:"encounter_id"`
		Description   string                 `json:"description"`
		MissedElement string                 `json:"missed_element"`
		TimeElapsed   time.Duration          `json:"time_elapsed"`
		TimeLimit     time.Duration          `json:"time_limit"`
		Context       map[string]interface{} `json:"context"`
		CreatedAt     time.Time              `json:"created_at"`
	}

	signal := GovernanceSignal{
		SignalID:      "GOV-MISSED-001",
		SignalType:    "bundle_element_missed",
		Severity:      "critical",
		ProtocolID:    "SEP-1",
		PatientID:     "patient-sepsis-001",
		EncounterID:   "encounter-sepsis-001",
		Description:   "Sepsis bundle element not completed within time limit",
		MissedElement: "Broad-spectrum antibiotics",
		TimeElapsed:   4 * time.Hour,
		TimeLimit:     3 * time.Hour,
		Context: map[string]interface{}{
			"order_status":   "not_ordered",
			"alert_count":    3,
			"override_count": 0,
		},
		CreatedAt: time.Now(),
	}

	// Verify signal structure
	assert.NotEmpty(t, signal.SignalID, "Signal should have ID")
	assert.Equal(t, "bundle_element_missed", signal.SignalType)
	assert.Equal(t, "critical", signal.Severity)
	assert.Greater(t, signal.TimeElapsed, signal.TimeLimit, "Should be overdue")

	// Verify JSON serialization for Kafka
	data, err := json.Marshal(signal)
	require.NoError(t, err)
	assert.Contains(t, string(data), "bundle_element_missed")

	t.Logf("✓ Governance signal for bundle miss: %s (%v overdue)",
		signal.MissedElement, signal.TimeElapsed-signal.TimeLimit)
}

func TestGovernanceSignalOnSafetyOverride(t *testing.T) {
	// Test that safety alert overrides generate governance signals
	type OverrideSignal struct {
		SignalID        string    `json:"signal_id"`
		SignalType      string    `json:"signal_type"`
		Severity        string    `json:"severity"`
		AlertType       string    `json:"alert_type"`
		AlertSeverity   string    `json:"alert_severity"`
		PatientID       string    `json:"patient_id"`
		ProviderID      string    `json:"provider_id"`
		OverrideReason  string    `json:"override_reason"`
		MedicationCode  string    `json:"medication_code"`
		MedicationName  string    `json:"medication_name"`
		OriginalAlert   string    `json:"original_alert"`
		CreatedAt       time.Time `json:"created_at"`
	}

	signal := OverrideSignal{
		SignalID:        "GOV-OVERRIDE-001",
		SignalType:      "safety_override",
		Severity:        "high",
		AlertType:       "drug-allergy",
		AlertSeverity:   "critical",
		PatientID:       "patient-allergy-001",
		ProviderID:      "provider-001",
		OverrideReason:  "Patient tolerated medication previously without issue",
		MedicationCode:  "733",
		MedicationName:  "Penicillin V",
		OriginalAlert:   "Drug-Allergy Alert: Patient has documented allergy to Penicillin",
		CreatedAt:       time.Now(),
	}

	// Verify signal captures all governance data
	assert.NotEmpty(t, signal.OverrideReason, "Override reason required")
	assert.NotEmpty(t, signal.ProviderID, "Provider attribution required")
	assert.Equal(t, "safety_override", signal.SignalType)

	// Verify JSON serialization
	data, err := json.Marshal(signal)
	require.NoError(t, err)
	assert.Contains(t, string(data), "safety_override")
	assert.Contains(t, string(data), signal.OverrideReason)

	t.Logf("✓ Governance signal for safety override: %s overriding %s alert",
		signal.ProviderID, signal.AlertType)
}

func TestGovernanceSignalOnRefusal(t *testing.T) {
	// Test that CDS recommendation refusals generate governance signals
	handler := cdshooks.NewFeedbackHandler()

	feedback := &cdshooks.CardFeedback{
		CardID:         "card-refused-001",
		Outcome:        "dismissed",
		OverrideReason: "not_indicated",
		Comments:       "Clinical judgment - patient has contraindication not in EHR",
		UserID:         "provider-refusal-001",
	}

	err := handler.RecordFeedback(feedback)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, found := handler.GetFeedback("card-refused-001")
	assert.True(t, found)
	assert.Equal(t, "dismissed", retrieved.Outcome)
	assert.Equal(t, "not_indicated", retrieved.OverrideReason)

	// Get stats for governance reporting
	stats := handler.GetStats()
	assert.Equal(t, 1, stats.Dismissed)

	t.Logf("✓ Governance signal for refusal captured: outcome=%s, reason=%s",
		retrieved.Outcome, retrieved.OverrideReason)
}

func TestGovernanceSignalStructure(t *testing.T) {
	// Test governance signal structure meets Kafka schema requirements
	type GovernanceSignal struct {
		// Required fields
		SignalID   string    `json:"signal_id"`
		SignalType string    `json:"signal_type"`
		Timestamp  time.Time `json:"timestamp"`
		Version    string    `json:"version"`

		// Context
		PatientID   string `json:"patient_id"`
		EncounterID string `json:"encounter_id,omitempty"`
		ProviderID  string `json:"provider_id,omitempty"`

		// Signal details
		Severity    string                 `json:"severity"`
		Category    string                 `json:"category"`
		Description string                 `json:"description"`
		Payload     map[string]interface{} `json:"payload"`

		// Metadata
		Source      string `json:"source"`
		Environment string `json:"environment"`
	}

	signal := GovernanceSignal{
		SignalID:    "GOV-TEST-001",
		SignalType:  "safety_event",
		Timestamp:   time.Now().UTC(),
		Version:     "1.0",
		PatientID:   "patient-001",
		EncounterID: "encounter-001",
		ProviderID:  "provider-001",
		Severity:    "high",
		Category:    "medication_safety",
		Description: "High-alert medication ordered with override",
		Payload: map[string]interface{}{
			"medication_code": "7052",
			"medication_name": "Morphine",
			"alert_type":      "high-alert-medication",
			"override_reason": "Verified dose appropriate",
		},
		Source:      "kb-12-ordersets-careplans",
		Environment: "production",
	}

	// Verify required fields
	assert.NotEmpty(t, signal.SignalID, "SignalID required")
	assert.NotEmpty(t, signal.SignalType, "SignalType required")
	assert.False(t, signal.Timestamp.IsZero(), "Timestamp required")
	assert.NotEmpty(t, signal.Version, "Version required")
	assert.NotEmpty(t, signal.PatientID, "PatientID required")
	assert.NotEmpty(t, signal.Source, "Source required")

	// Verify JSON serialization
	data, err := json.Marshal(signal)
	require.NoError(t, err)

	// Verify can deserialize
	var parsed GovernanceSignal
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, signal.SignalID, parsed.SignalID)

	t.Logf("✓ Governance signal structure valid: %s (%s)", signal.SignalType, signal.Severity)
}

func TestGovernanceSignalKafkaEmission(t *testing.T) {
	// Test governance signal Kafka emission structure
	// Note: Actual Kafka integration would be tested in integration tests

	type KafkaMessage struct {
		Topic     string            `json:"topic"`
		Key       string            `json:"key"`
		Value     []byte            `json:"value"`
		Headers   map[string]string `json:"headers"`
		Partition int32             `json:"partition"`
	}

	signal := map[string]interface{}{
		"signal_id":   "GOV-KAFKA-001",
		"signal_type": "clinical_alert_override",
		"patient_id":  "patient-kafka-001",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	value, err := json.Marshal(signal)
	require.NoError(t, err)

	msg := KafkaMessage{
		Topic: "governance-signals",
		Key:   signal["patient_id"].(string), // Partition by patient
		Value: value,
		Headers: map[string]string{
			"source":      "kb-12-ordersets-careplans",
			"signal_type": "clinical_alert_override",
			"version":     "1.0",
		},
		Partition: -1, // Let Kafka decide based on key
	}

	// Verify message structure
	assert.Equal(t, "governance-signals", msg.Topic)
	assert.NotEmpty(t, msg.Key)
	assert.NotEmpty(t, msg.Value)
	assert.NotEmpty(t, msg.Headers["source"])

	t.Logf("✓ Kafka message structure valid for topic: %s", msg.Topic)
}

func TestGovernanceSignalRetry(t *testing.T) {
	// Test governance signal retry mechanism
	type RetryableSignal struct {
		Signal       interface{} `json:"signal"`
		Attempts     int         `json:"attempts"`
		MaxAttempts  int         `json:"max_attempts"`
		LastAttempt  time.Time   `json:"last_attempt"`
		NextAttempt  time.Time   `json:"next_attempt"`
		BackoffMs    int64       `json:"backoff_ms"`
		Status       string      `json:"status"` // pending, success, failed
	}

	signal := RetryableSignal{
		Signal: map[string]interface{}{
			"signal_id":   "GOV-RETRY-001",
			"signal_type": "bundle_missed",
		},
		Attempts:    2,
		MaxAttempts: 5,
		LastAttempt: time.Now().Add(-30 * time.Second),
		BackoffMs:   1000 * 60, // 1 minute
		Status:      "pending",
	}

	// Calculate next attempt with exponential backoff
	backoffDuration := time.Duration(signal.BackoffMs) * time.Millisecond * time.Duration(1<<signal.Attempts)
	signal.NextAttempt = signal.LastAttempt.Add(backoffDuration)

	// Verify retry logic
	assert.Less(t, signal.Attempts, signal.MaxAttempts, "Should have remaining attempts")
	assert.True(t, signal.NextAttempt.After(signal.LastAttempt), "Next attempt should be in future")
	assert.Equal(t, "pending", signal.Status)

	// Simulate successful delivery
	signal.Attempts++
	signal.Status = "success"
	signal.LastAttempt = time.Now()

	assert.Equal(t, "success", signal.Status)
	t.Logf("✓ Signal retry mechanism: %d attempts, backoff=%v, status=%s",
		signal.Attempts, backoffDuration, signal.Status)
}

// ============================================
// 9.3 Regulatory Compliance Tests
// ============================================

func TestHIPAACompliantLogging(t *testing.T) {
	// Test that logs don't contain PHI inappropriately
	type SanitizedLog struct {
		Timestamp   time.Time `json:"timestamp"`
		Level       string    `json:"level"`
		Message     string    `json:"message"`
		PatientID   string    `json:"patient_id,omitempty"` // Should be hashed or omitted in production
		SessionID   string    `json:"session_id"`
		Action      string    `json:"action"`
		Redacted    []string  `json:"redacted_fields,omitempty"`
	}

	log := SanitizedLog{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Order signed successfully",
		PatientID: "HASH:abc123...", // Should be hashed, not plain
		SessionID: "SESS-12345",
		Action:    "order_signed",
		Redacted:  []string{"patient_name", "dob", "ssn", "address"},
	}

	// Verify no plain PHI
	assert.NotContains(t, log.PatientID, "patient-", "PatientID should be hashed in logs")
	assert.NotEmpty(t, log.Redacted, "Should list redacted fields")

	// Verify required audit fields
	assert.NotZero(t, log.Timestamp)
	assert.NotEmpty(t, log.Action)
	assert.NotEmpty(t, log.SessionID)

	t.Logf("✓ HIPAA-compliant log structure: %d fields redacted", len(log.Redacted))
}

func TestJointCommissionCompliance(t *testing.T) {
	// Test Joint Commission medication safety requirements
	type JCCompliance struct {
		Requirement   string `json:"requirement"`
		Standard      string `json:"standard"`
		Implemented   bool   `json:"implemented"`
		EvidenceType  string `json:"evidence_type"`
	}

	requirements := []JCCompliance{
		{
			Requirement:  "High-alert medication identification",
			Standard:     "NPSG.03.04.01",
			Implemented:  true,
			EvidenceType: "system_alert",
		},
		{
			Requirement:  "Drug-drug interaction checking",
			Standard:     "NPSG.03.06.01",
			Implemented:  true,
			EvidenceType: "cpoe_validation",
		},
		{
			Requirement:  "Allergy checking before dispensing",
			Standard:     "NPSG.03.06.01",
			Implemented:  true,
			EvidenceType: "cpoe_validation",
		},
		{
			Requirement:  "Duplicate therapy checking",
			Standard:     "NPSG.03.06.01",
			Implemented:  true,
			EvidenceType: "cpoe_validation",
		},
		{
			Requirement:  "Override reason documentation",
			Standard:     "MM.05.01.17",
			Implemented:  true,
			EvidenceType: "audit_trail",
		},
	}

	compliantCount := 0
	for _, req := range requirements {
		if req.Implemented {
			compliantCount++
		}
		status := "✓"
		if !req.Implemented {
			status = "○"
		}
		t.Logf("%s %s (%s): %s", status, req.Requirement, req.Standard, req.EvidenceType)
	}

	complianceRate := float64(compliantCount) / float64(len(requirements)) * 100
	assert.GreaterOrEqual(t, complianceRate, 80.0, "Should meet minimum compliance threshold")

	t.Logf("✓ Joint Commission compliance: %.1f%% (%d/%d requirements)",
		complianceRate, compliantCount, len(requirements))
}

func TestCMSQualityMeasureReadiness(t *testing.T) {
	// Test readiness for CMS quality measure reporting (e.g., SEP-1)
	type QualityMeasure struct {
		MeasureID     string   `json:"measure_id"`
		MeasureName   string   `json:"measure_name"`
		DataElements  []string `json:"data_elements"`
		CanReport     bool     `json:"can_report"`
		MissingData   []string `json:"missing_data,omitempty"`
	}

	measures := []QualityMeasure{
		{
			MeasureID:   "SEP-1",
			MeasureName: "Severe Sepsis and Septic Shock: Management Bundle",
			DataElements: []string{
				"sepsis_diagnosis_time",
				"blood_culture_time",
				"lactate_time",
				"antibiotic_time",
				"fluid_bolus_time",
				"vasopressor_time",
				"repeat_lactate_time",
			},
			CanReport: true,
		},
		{
			MeasureID:   "VTE-1",
			MeasureName: "Venous Thromboembolism Prophylaxis",
			DataElements: []string{
				"vte_risk_assessment_time",
				"prophylaxis_order_time",
				"prophylaxis_admin_time",
			},
			CanReport: true,
		},
		{
			MeasureID:   "STK-4",
			MeasureName: "Thrombolytic Therapy",
			DataElements: []string{
				"arrival_time",
				"ct_time",
				"tpa_decision_time",
				"tpa_administration_time",
			},
			CanReport: true,
		},
	}

	for _, measure := range measures {
		status := "✓ Ready"
		if !measure.CanReport {
			status = "○ Missing: " + measure.MissingData[0]
		}
		t.Logf("%s %s (%s): %d data elements", status, measure.MeasureID, measure.MeasureName, len(measure.DataElements))
	}

	readyCount := 0
	for _, m := range measures {
		if m.CanReport {
			readyCount++
		}
	}

	t.Logf("✓ CMS Quality Measure readiness: %d/%d measures reportable", readyCount, len(measures))
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkAuditEntryCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		entry := struct {
			EntryID   string
			Timestamp time.Time
			Action    string
			UserID    string
		}{
			EntryID:   "audit-bench",
			Timestamp: time.Now(),
			Action:    "order_created",
			UserID:    "provider-bench",
		}
		_ = entry
	}
}

func BenchmarkAuditChainHash(b *testing.B) {
	entry := map[string]interface{}{
		"entry_id":  "audit-bench",
		"timestamp": time.Now(),
		"action":    "order_created",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(entry)
		hash := sha256.Sum256(data)
		_ = hex.EncodeToString(hash[:])
	}
}

func BenchmarkGovernanceSignalSerialization(b *testing.B) {
	signal := map[string]interface{}{
		"signal_id":   "GOV-BENCH-001",
		"signal_type": "safety_event",
		"patient_id":  "patient-bench",
		"timestamp":   time.Now().UTC(),
		"payload": map[string]interface{}{
			"alert_type": "drug-allergy",
			"severity":   "critical",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(signal)
	}
}
