# Gap 21: Closed-Loop Outcome Learning — Implementation Plan (Sprint 1: Attribution Foundation)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the attribution foundation the Gap 19 lifecycle and Gap 20 risk predictor have been feeding into — ingest T4 outcomes, consolidate each T0→T4 alert into a causally-annotated record, produce a rule-based attribution verdict with the 5-label clinician vocabulary, and write every attribution run to an append-only HMAC-chained governance ledger.

**Architecture:** Outcome ingestion and the consolidated alert record live in `kb-23-decision-cards` next to the Gap 19 `DetectionLifecycle` model they extend. The rule-based attribution engine and the governance ledger live in `kb-26-metabolic-digital-twin` next to the Gap 20 `PredictedRisk` model they score against. The engine produces a `ClinicianLabel` enum (`prevented`, `no_effect_detected`, `outcome_despite_intervention`, `fragile_estimate`, `inconclusive`) via rule-based comparison against the patient's own pre-alert baseline — no propensity scores, no doubly-robust estimation, no counterfactual outcome head. When KB-28 Python ships (Sprint 2+) with propensity/TMLE/counterfactual, the `AttributionVerdict` contract stays stable and the engine behind it is replaced in-place.

**Tech Stack:** Go 1.21, existing Gap 19 `DetectionLifecycle` + Gap 20 `PredictedRisk` infrastructure as inputs, YAML market config for attribution thresholds and outcome definitions, HMAC-SHA256 chain for ledger signing (Ed25519 deferred to Sprint 2). Gin for HTTP handlers, GORM for persistence.

---

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| T0→T4 lifecycle tracker | [kb-23 detection_lifecycle.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/detection_lifecycle.go) | Source of T0 (`DetectedAt`), T1 (`DeliveredAt`), T2 (`AcknowledgedAt`), T3 (`ActionedAt`), T4 (`ResolvedAt`) timestamps + action type/detail fields |
| PAI + PredictedRisk | [kb-26 predicted_risk.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/predicted_risk.go) | Pre-alert risk snapshot (`RiskScore`, `PrimaryDrivers`, `ModelType`) — the prediction being attributed |
| Per-patient baselines | Gap 14–19 folded-service pattern | Attribution anchors counterfactual to patient's own pre-alert trajectory, not cohort mean |
| Market config YAML pattern | `market-configs/shared/` | Outcome horizons and attribution thresholds per pilot (HCF CHF 30d, Aged Care 90d) |
| Cohort scoping | `DetectionLifecycle.CohortID` | Attribution aggregates filter by cohort (`hcf_catalyst_chf`, `aged_care_au`) |

## File Inventory

### KB-23 — Outcome ingestion + consolidation
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/outcome_record.go` | `OutcomeRecord`, `OutcomeSource` enum, `ReconciliationStatus` enum |
| Create | `internal/services/outcome_ingestion.go` | `IngestOutcome`, `ReconcileOutcomes` — cross-source dedup and conflict queue |
| Create | `internal/services/outcome_ingestion_test.go` | 3 tests |
| Create | `internal/models/consolidated_alert_record.go` | `ConsolidatedAlertRecord`, `TreatmentStrategy` enum |
| Create | `internal/services/alert_consolidation.go` | `BuildConsolidatedRecord` — join lifecycle + outcome + treatment strategy + time-zero |
| Create | `internal/services/alert_consolidation_test.go` | 3 tests |
| Modify | `main.go` | AutoMigrate `OutcomeRecord`, `ConsolidatedAlertRecord` |

### KB-26 — Attribution engine + governance ledger
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/attribution_verdict.go` | `AttributionVerdict`, `ClinicianLabel` enum, `LedgerEntry` |
| Create | `internal/services/rule_attribution.go` | `ComputeAttribution` — rule-based verdict with 5-label mapping |
| Create | `internal/services/rule_attribution_test.go` | 4 tests |
| Create | `internal/services/append_only_ledger.go` | `AppendEntry`, `VerifyChain` — HMAC-SHA256 chain |
| Create | `internal/services/append_only_ledger_test.go` | 2 tests |
| Create | `internal/api/attribution_handlers.go` | `POST /attribution/run`, `GET /attribution/:recordId`, `GET /governance/ledger` |
| Modify | `internal/api/routes.go` | Add attribution + governance route groups |
| Modify | `main.go` | AutoMigrate `AttributionVerdict`, `LedgerEntry` |

### Market configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/outcome_ingestion_parameters.yaml` | Outcome definitions, source priorities, reconciliation rules per pilot |
| Create | `market-configs/shared/attribution_parameters.yaml` | Rule-based thresholds per `ClinicianLabel`, horizon windows, fragile-estimate bound |

**Total: 14 create, 3 modify, ~12 tests**

---

### Task 1: Outcome ingestion models + YAML config

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/outcome_record.go`
- Create: `backend/shared-infrastructure/market-configs/shared/outcome_ingestion_parameters.yaml`

- [ ] **Step 1:** Create `outcome_record.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

// OutcomeSource identifies where an outcome record came from.
type OutcomeSource string

const (
	OutcomeSourceHospitalDischarge OutcomeSource = "HOSPITAL_DISCHARGE"
	OutcomeSourceClaimsFeed        OutcomeSource = "CLAIMS_FEED"
	OutcomeSourceMortalityRegistry OutcomeSource = "MORTALITY_REGISTRY"
	OutcomeSourceClinicianConfirm  OutcomeSource = "CLINICIAN_CONFIRMATION"
	OutcomeSourceFacilityReport    OutcomeSource = "FACILITY_REPORT"
)

// ReconciliationStatus tracks whether an outcome has been resolved across sources.
type ReconciliationStatus string

const (
	ReconciliationPending    ReconciliationStatus = "PENDING"
	ReconciliationResolved   ReconciliationStatus = "RESOLVED"
	ReconciliationConflicted ReconciliationStatus = "CONFLICTED"
	ReconciliationHorizonExp ReconciliationStatus = "HORIZON_EXPIRED"
)

// OutcomeRecord is a single outcome observation for one patient from one source.
// Multiple OutcomeRecords for the same (patient, outcome_type) are reconciled into
// a single authoritative record by OutcomeIngestionService.
type OutcomeRecord struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID      string    `gorm:"size:100;index;not null" json:"patient_id"`
	LifecycleID    *uuid.UUID `gorm:"type:uuid;index" json:"lifecycle_id,omitempty"` // links to DetectionLifecycle
	CohortID       string    `gorm:"size:60;index" json:"cohort_id,omitempty"`
	OutcomeType    string    `gorm:"size:60;index;not null" json:"outcome_type"` // READMISSION_30D, ADMISSION_90D, MORTALITY_30D, etc.
	OutcomeOccurred bool     `gorm:"not null" json:"outcome_occurred"`
	OccurredAt     *time.Time `json:"occurred_at,omitempty"`
	Source         string    `gorm:"size:40;index;not null" json:"source"`
	SourceRecordID string    `gorm:"size:200" json:"source_record_id,omitempty"`
	Reconciliation string    `gorm:"size:20;index;not null;default:'PENDING'" json:"reconciliation"`
	ReconciledID   *uuid.UUID `gorm:"type:uuid" json:"reconciled_id,omitempty"` // points to authoritative record after reconciliation
	IngestedAt     time.Time `gorm:"autoCreateTime" json:"ingested_at"`
	Notes          string    `gorm:"type:text" json:"notes,omitempty"`
}

func (OutcomeRecord) TableName() string { return "outcome_records" }
```

- [ ] **Step 2:** Create `outcome_ingestion_parameters.yaml`:

```yaml
# Outcome ingestion configuration — Sprint 1.
# Per-market outcome definitions, source priorities for reconciliation,
# and horizon windows for T4 closure.

markets:
  hcf_catalyst_chf:
    outcomes:
      - type: READMISSION_30D
        horizon_days: 30
        sources_priority: [HOSPITAL_DISCHARGE, CLAIMS_FEED, CLINICIAN_CONFIRMATION]
        is_primary: true
      - type: READMISSION_90D
        horizon_days: 90
        sources_priority: [HOSPITAL_DISCHARGE, CLAIMS_FEED]
        is_primary: false
      - type: MORTALITY_30D
        horizon_days: 30
        sources_priority: [MORTALITY_REGISTRY, HOSPITAL_DISCHARGE, CLINICIAN_CONFIRMATION]
        is_primary: false

  aged_care_au:
    outcomes:
      - type: ADMISSION_90D
        horizon_days: 90
        sources_priority: [HOSPITAL_DISCHARGE, CLAIMS_FEED, FACILITY_REPORT]
        is_primary: true
      - type: MORTALITY_90D
        horizon_days: 90
        sources_priority: [MORTALITY_REGISTRY, FACILITY_REPORT]
        is_primary: false

reconciliation:
  # When two sources disagree on occurred vs not occurred, queue for adjudication.
  # When they agree but differ in occurred_at by more than this, flag.
  timestamp_tolerance_hours: 48
  # Minimum sources for auto-resolve; below this stays PENDING until horizon expires.
  min_sources_for_auto_resolve: 1

horizon_closure:
  # If no outcome received by horizon expiry, emit a HORIZON_EXPIRED record
  # with outcome_occurred=false so downstream attribution can still run.
  emit_on_expiry: true
```

- [ ] **Step 3:** Verify Go compile and YAML parse:

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./...`
Expected: success.

Run: `python3 -c "import yaml; yaml.safe_load(open('backend/shared-infrastructure/market-configs/shared/outcome_ingestion_parameters.yaml'))"`
Expected: no error.

- [ ] **Step 4:** Commit:

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/outcome_record.go \
        backend/shared-infrastructure/market-configs/shared/outcome_ingestion_parameters.yaml
git commit -m "feat(kb23): outcome ingestion models + config (Gap 21 Task 1)"
```

---

### Task 2: Outcome ingestion + reconciliation service

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/outcome_ingestion.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/outcome_ingestion_test.go`

- [ ] **Step 1:** Write 3 failing tests in `outcome_ingestion_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"kb-23-decision-cards/internal/models"
)

func TestOutcomeIngest_SingleSource_AutoResolves(t *testing.T) {
	lifecycleID := uuid.New()
	records := []models.OutcomeRecord{
		{
			PatientID:       "P001",
			LifecycleID:     &lifecycleID,
			CohortID:        "hcf_catalyst_chf",
			OutcomeType:     "READMISSION_30D",
			OutcomeOccurred: true,
			Source:          string(models.OutcomeSourceHospitalDischarge),
			IngestedAt:      time.Now(),
		},
	}
	result, err := ReconcileOutcomes(records, 48*time.Hour, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reconciliation != string(models.ReconciliationResolved) {
		t.Fatalf("expected RESOLVED, got %s", result.Reconciliation)
	}
	if !result.OutcomeOccurred {
		t.Fatalf("expected outcome_occurred=true")
	}
}

func TestOutcomeIngest_MultipleSourcesAgree_Resolves(t *testing.T) {
	lifecycleID := uuid.New()
	occurredAt := time.Now().Add(-10 * 24 * time.Hour)
	records := []models.OutcomeRecord{
		{
			PatientID: "P002", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
			OutcomeOccurred: true, OccurredAt: &occurredAt,
			Source: string(models.OutcomeSourceHospitalDischarge),
		},
		{
			PatientID: "P002", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
			OutcomeOccurred: true, OccurredAt: &occurredAt,
			Source: string(models.OutcomeSourceClaimsFeed),
		},
	}
	result, err := ReconcileOutcomes(records, 48*time.Hour, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reconciliation != string(models.ReconciliationResolved) {
		t.Fatalf("expected RESOLVED, got %s", result.Reconciliation)
	}
}

func TestOutcomeIngest_MultipleSourcesDisagree_Conflicts(t *testing.T) {
	lifecycleID := uuid.New()
	records := []models.OutcomeRecord{
		{
			PatientID: "P003", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
			OutcomeOccurred: true,
			Source: string(models.OutcomeSourceHospitalDischarge),
		},
		{
			PatientID: "P003", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
			OutcomeOccurred: false,
			Source: string(models.OutcomeSourceClaimsFeed),
		},
	}
	result, err := ReconcileOutcomes(records, 48*time.Hour, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reconciliation != string(models.ReconciliationConflicted) {
		t.Fatalf("expected CONFLICTED, got %s", result.Reconciliation)
	}
}
```

- [ ] **Step 2:** Run tests — expect FAIL ("undefined: ReconcileOutcomes"):

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run TestOutcomeIngest -v`
Expected: build failure — `ReconcileOutcomes` undefined.

- [ ] **Step 3:** Implement `outcome_ingestion.go`:

```go
package services

import (
	"fmt"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ReconcileOutcomes takes one or more OutcomeRecords for the same (patient, outcome_type)
// and returns a single authoritative record with Reconciliation set to RESOLVED,
// CONFLICTED, or PENDING. Source agreement on OutcomeOccurred resolves; disagreement
// conflicts; insufficient sources remain pending.
func ReconcileOutcomes(records []models.OutcomeRecord, tolerance time.Duration, minSources int) (models.OutcomeRecord, error) {
	if len(records) == 0 {
		return models.OutcomeRecord{}, fmt.Errorf("no records to reconcile")
	}
	if len(records) < minSources {
		r := records[0]
		r.Reconciliation = string(models.ReconciliationPending)
		return r, nil
	}

	firstOccurred := records[0].OutcomeOccurred
	allAgree := true
	for _, r := range records[1:] {
		if r.OutcomeOccurred != firstOccurred {
			allAgree = false
			break
		}
	}

	result := records[0]
	if !allAgree {
		result.Reconciliation = string(models.ReconciliationConflicted)
		result.Notes = fmt.Sprintf("conflict across %d sources", len(records))
		return result, nil
	}

	// All agree on occurrence — check timestamp agreement within tolerance.
	if firstOccurred && len(records) > 1 {
		for _, r := range records[1:] {
			if r.OccurredAt != nil && result.OccurredAt != nil {
				diff := r.OccurredAt.Sub(*result.OccurredAt)
				if diff < -tolerance || diff > tolerance {
					result.Reconciliation = string(models.ReconciliationConflicted)
					result.Notes = fmt.Sprintf("timestamp disagreement beyond %s", tolerance)
					return result, nil
				}
			}
		}
	}

	result.Reconciliation = string(models.ReconciliationResolved)
	return result, nil
}
```

- [ ] **Step 4:** Run tests — expect PASS:

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run TestOutcomeIngest -v`
Expected: 3/3 pass.

- [ ] **Step 5:** Commit:

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/outcome_ingestion.go \
        backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/outcome_ingestion_test.go
git commit -m "feat(kb23): outcome reconciliation service (Gap 21 Task 2)"
```

---

### Task 3: Consolidated alert record + causal annotator

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/consolidated_alert_record.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/alert_consolidation.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/alert_consolidation_test.go`

- [ ] **Step 1:** Create `consolidated_alert_record.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

// TreatmentStrategy is the TTE-protocol treatment label inferred from the clinician response.
// Derived from DetectionLifecycle.ActionType (Gap 19) and the override/intervention choice.
type TreatmentStrategy string

const (
	TreatmentInterventionTaken TreatmentStrategy = "INTERVENTION_TAKEN"
	TreatmentOverrideReason    TreatmentStrategy = "OVERRIDE_WITH_REASON"
	TreatmentNoResponse        TreatmentStrategy = "NO_RESPONSE"
	TreatmentAlreadyAddressed  TreatmentStrategy = "ALREADY_ADDRESSED"
)

// ConsolidatedAlertRecord is the TTE-ready per-alert record combining:
// - pre-alert prediction snapshot (Gap 20 PredictedRisk)
// - lifecycle timestamps (Gap 19 DetectionLifecycle)
// - treatment strategy (clinician response)
// - outcome (Task 1 OutcomeRecord)
// - time-zero (TTE protocol anchor)
type ConsolidatedAlertRecord struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	LifecycleID       uuid.UUID  `gorm:"type:uuid;index;not null" json:"lifecycle_id"`
	PatientID         string     `gorm:"size:100;index;not null" json:"patient_id"`
	CohortID          string     `gorm:"size:60;index" json:"cohort_id,omitempty"`

	// Pre-alert snapshot
	PreAlertRiskScore float64    `json:"pre_alert_risk_score"`
	PreAlertRiskTier  string     `gorm:"size:10" json:"pre_alert_risk_tier"`
	PredictionModelID string     `gorm:"size:60" json:"prediction_model_id,omitempty"`

	// Lifecycle anchors
	DetectedAt        time.Time  `gorm:"not null" json:"detected_at"`         // T0
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`              // T1
	AcknowledgedAt    *time.Time `json:"acknowledged_at,omitempty"`           // T2
	ActionedAt        *time.Time `json:"actioned_at,omitempty"`               // T3
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`               // T4

	// Causal annotation
	TimeZero          time.Time  `gorm:"not null" json:"time_zero"`           // TTE anchor
	TreatmentStrategy string     `gorm:"size:40;index;not null" json:"treatment_strategy"`
	ActionType        string     `gorm:"size:60" json:"action_type,omitempty"`
	OverrideReason    string     `gorm:"size:60" json:"override_reason,omitempty"`

	// Outcome
	OutcomeRecordID   *uuid.UUID `gorm:"type:uuid;index" json:"outcome_record_id,omitempty"`
	OutcomeOccurred   *bool      `json:"outcome_occurred,omitempty"`
	OutcomeType       string     `gorm:"size:60" json:"outcome_type,omitempty"`
	HorizonDays       int        `json:"horizon_days"`

	BuiltAt           time.Time  `gorm:"autoCreateTime" json:"built_at"`
}

func (ConsolidatedAlertRecord) TableName() string { return "consolidated_alert_records" }
```

- [ ] **Step 2:** Write 3 failing tests in `alert_consolidation_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"kb-23-decision-cards/internal/models"
)

func TestConsolidate_InterventionTaken_ActionTypePresent(t *testing.T) {
	lc := models.DetectionLifecycle{
		ID: uuid.New(), PatientID: "P100", CohortID: "hcf_catalyst_chf",
		DetectedAt: time.Now().Add(-40 * 24 * time.Hour),
		ActionType: "nurse_phone_followup",
	}
	t1 := lc.DetectedAt.Add(5 * time.Minute); lc.DeliveredAt = &t1
	t2 := lc.DetectedAt.Add(30 * time.Minute); lc.AcknowledgedAt = &t2
	t3 := lc.DetectedAt.Add(2 * time.Hour); lc.ActionedAt = &t3

	outcomeID := uuid.New()
	occurred := false
	outcome := models.OutcomeRecord{
		ID: outcomeID, PatientID: "P100", OutcomeType: "READMISSION_30D",
		OutcomeOccurred: occurred,
		Reconciliation: string(models.ReconciliationResolved),
	}
	riskScore := 62.0
	record, err := BuildConsolidatedRecord(lc, &outcome, riskScore, "HIGH", "gap20-heuristic-v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.TreatmentStrategy != string(models.TreatmentInterventionTaken) {
		t.Fatalf("expected INTERVENTION_TAKEN, got %s", record.TreatmentStrategy)
	}
	if record.ActionType != "nurse_phone_followup" {
		t.Fatalf("expected action_type preserved")
	}
	if record.TimeZero != lc.DetectedAt {
		t.Fatalf("expected time_zero = T0 (DetectedAt)")
	}
}

func TestConsolidate_Override_OverrideReasonPreserved(t *testing.T) {
	lc := models.DetectionLifecycle{
		ID: uuid.New(), PatientID: "P101",
		DetectedAt: time.Now().Add(-40 * 24 * time.Hour),
		ActionType: "OVERRIDE",
		ActionDetail: "reason=already_addressed",
	}
	record, err := BuildConsolidatedRecord(lc, nil, 55.0, "HIGH", "gap20-heuristic-v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.TreatmentStrategy != string(models.TreatmentOverrideReason) {
		t.Fatalf("expected OVERRIDE_WITH_REASON, got %s", record.TreatmentStrategy)
	}
	if record.OverrideReason != "already_addressed" {
		t.Fatalf("expected override_reason=already_addressed, got %s", record.OverrideReason)
	}
}

func TestConsolidate_NoResponse_HorizonClosure(t *testing.T) {
	lc := models.DetectionLifecycle{
		ID: uuid.New(), PatientID: "P102",
		DetectedAt: time.Now().Add(-45 * 24 * time.Hour),
		// no DeliveredAt / AcknowledgedAt / ActionedAt
	}
	record, err := BuildConsolidatedRecord(lc, nil, 48.0, "MODERATE", "gap20-heuristic-v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.TreatmentStrategy != string(models.TreatmentNoResponse) {
		t.Fatalf("expected NO_RESPONSE, got %s", record.TreatmentStrategy)
	}
}
```

- [ ] **Step 3:** Run tests — expect FAIL (`BuildConsolidatedRecord` undefined):

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run TestConsolidate -v`
Expected: build failure.

- [ ] **Step 4:** Implement `alert_consolidation.go`:

```go
package services

import (
	"fmt"
	"strings"

	"kb-23-decision-cards/internal/models"
)

// BuildConsolidatedRecord joins a DetectionLifecycle (Gap 19), an OutcomeRecord (Gap 21 Task 1),
// and the pre-alert PredictedRisk snapshot (Gap 20) into a single TTE-ready record.
// Time-zero is anchored at DetectedAt (T0) — the moment the alert fired is the
// earliest plausible point where the alert could have changed clinician behavior.
func BuildConsolidatedRecord(
	lc models.DetectionLifecycle,
	outcome *models.OutcomeRecord,
	preAlertRiskScore float64,
	preAlertRiskTier string,
	predictionModelID string,
) (models.ConsolidatedAlertRecord, error) {
	strategy := classifyTreatmentStrategy(lc)
	overrideReason := ""
	if strategy == models.TreatmentOverrideReason {
		overrideReason = extractOverrideReason(lc.ActionDetail)
		if overrideReason == "already_addressed" {
			strategy = models.TreatmentAlreadyAddressed
		}
	}

	record := models.ConsolidatedAlertRecord{
		LifecycleID:       lc.ID,
		PatientID:         lc.PatientID,
		CohortID:          lc.CohortID,
		PreAlertRiskScore: preAlertRiskScore,
		PreAlertRiskTier:  preAlertRiskTier,
		PredictionModelID: predictionModelID,
		DetectedAt:        lc.DetectedAt,
		DeliveredAt:       lc.DeliveredAt,
		AcknowledgedAt:    lc.AcknowledgedAt,
		ActionedAt:        lc.ActionedAt,
		ResolvedAt:        lc.ResolvedAt,
		TimeZero:          lc.DetectedAt,
		TreatmentStrategy: string(strategy),
		ActionType:        lc.ActionType,
		OverrideReason:    overrideReason,
	}

	if outcome != nil {
		if outcome.Reconciliation != string(models.ReconciliationResolved) &&
			outcome.Reconciliation != string(models.ReconciliationHorizonExp) {
			return record, fmt.Errorf("outcome record not resolved: %s", outcome.Reconciliation)
		}
		record.OutcomeRecordID = &outcome.ID
		occurred := outcome.OutcomeOccurred
		record.OutcomeOccurred = &occurred
		record.OutcomeType = outcome.OutcomeType
	}

	return record, nil
}

func classifyTreatmentStrategy(lc models.DetectionLifecycle) models.TreatmentStrategy {
	if lc.ActionType == "OVERRIDE" {
		return models.TreatmentOverrideReason
	}
	if lc.ActionedAt != nil && lc.ActionType != "" {
		return models.TreatmentInterventionTaken
	}
	return models.TreatmentNoResponse
}

func extractOverrideReason(detail string) string {
	// ActionDetail format (from Gap 18 worklist): "reason=<override_code>[;...]"
	for _, part := range strings.Split(detail, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "reason=") {
			return strings.TrimPrefix(part, "reason=")
		}
	}
	return ""
}
```

- [ ] **Step 5:** Run tests — expect PASS:

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run TestConsolidate -v`
Expected: 3/3 pass.

- [ ] **Step 6:** Commit:

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/consolidated_alert_record.go \
        backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/alert_consolidation.go \
        backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/alert_consolidation_test.go
git commit -m "feat(kb23): consolidated alert record + causal annotator (Gap 21 Task 3)"
```

---

### Task 4: Rule-based attribution engine

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/attribution_verdict.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/rule_attribution.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/rule_attribution_test.go`
- Create: `backend/shared-infrastructure/market-configs/shared/attribution_parameters.yaml`

- [ ] **Step 1:** Create `attribution_verdict.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

// ClinicianLabel is the discrete verdict shown to a clinician for one alert attribution.
// Vocabulary matches Gap 21 spec §6.1 and is the stable contract Sprint 2 ML attribution
// will produce with the same values.
type ClinicianLabel string

const (
	LabelPrevented                ClinicianLabel = "prevented"
	LabelNoEffectDetected         ClinicianLabel = "no_effect_detected"
	LabelOutcomeDespiteIntervention ClinicianLabel = "outcome_despite_intervention"
	LabelFragileEstimate          ClinicianLabel = "fragile_estimate"
	LabelInconclusive             ClinicianLabel = "inconclusive"
)

// AttributionVerdict is the output of the attribution engine for a single consolidated
// alert record. Sprint 1 fills in RiskDifference/RiskReductionPct via rule-based
// comparison against the patient's own pre-alert baseline. Sprint 2 (KB-28 Python)
// will replace the math but keep this struct.
type AttributionVerdict struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ConsolidatedRecordID uuid.UUID `gorm:"type:uuid;index;not null" json:"consolidated_record_id"`
	PatientID          string    `gorm:"size:100;index;not null" json:"patient_id"`
	CohortID           string    `gorm:"size:60;index" json:"cohort_id,omitempty"`

	// Attribution outputs
	ClinicianLabel     string    `gorm:"size:40;index;not null" json:"clinician_label"`
	TechnicalLabel     string    `gorm:"size:60" json:"technical_label"`
	RiskDifference     float64   `json:"risk_difference"`     // percentage points (patient baseline − observed)
	RiskReductionPct   float64   `json:"risk_reduction_pct"`  // 0-100
	CounterfactualRisk float64   `json:"counterfactual_risk"` // patient pre-alert baseline
	ObservedOutcome    bool      `json:"observed_outcome"`
	PredictionWindowDays int     `json:"prediction_window_days"`

	// Provenance
	AttributionMethod  string    `gorm:"size:20;not null;default:'RULE_BASED'" json:"attribution_method"` // RULE_BASED | IPW | DOUBLY_ROBUST | TMLE
	MethodVersion      string    `gorm:"size:40" json:"method_version"`
	Rationale          string    `gorm:"type:text" json:"rationale"`
	LedgerEntryID      *uuid.UUID `gorm:"type:uuid" json:"ledger_entry_id,omitempty"`

	ComputedAt         time.Time `gorm:"autoCreateTime" json:"computed_at"`
}

func (AttributionVerdict) TableName() string { return "attribution_verdicts" }

// LedgerEntry is one append-only governance ledger entry. Sprint 1 uses HMAC-SHA256
// chain (each entry includes prior entry's hash); Sprint 2 layers Ed25519 signatures on top.
type LedgerEntry struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Sequence          int64     `gorm:"uniqueIndex;not null" json:"sequence"`
	EntryType         string    `gorm:"size:40;not null" json:"entry_type"` // ATTRIBUTION_RUN | MODEL_PROMOTION | ROLLBACK | RECALIBRATION
	SubjectID         string    `gorm:"size:100;index" json:"subject_id"`    // verdict ID, model ID, etc.
	PayloadJSON       string    `gorm:"type:text;not null" json:"payload_json"`
	PriorHash         string    `gorm:"size:64;not null" json:"prior_hash"`  // hex sha256
	EntryHash         string    `gorm:"size:64;not null" json:"entry_hash"`  // hex sha256
	CreatedAt         time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (LedgerEntry) TableName() string { return "governance_ledger_entries" }
```

- [ ] **Step 2:** Create `attribution_parameters.yaml`:

```yaml
# Rule-based attribution thresholds — Sprint 1.
# These thresholds convert a patient's own pre-alert risk vs observed outcome into
# one of the five ClinicianLabel values. Sprint 2 replaces this with propensity-weighted
# and doubly-robust estimators; the ClinicianLabel vocabulary stays identical.

method:
  name: RULE_BASED
  version: sprint1-v1

# Per-patient-baseline rules — no cohort mean, no propensity model.
# The "counterfactual" is the patient's own pre-alert risk score (from Gap 20).
# If the outcome does NOT occur and baseline risk was HIGH, we attribute prevention.
# If the outcome DOES occur and baseline risk was HIGH, we attribute "outcome despite".
# If baseline risk was LOW, we return "no_effect_detected" — rule-based attribution
# cannot distinguish a genuine effect from chance on low-risk cases.

labels:
  prevented:
    conditions:
      treatment_strategy_in: [INTERVENTION_TAKEN]
      outcome_occurred: false
      pre_alert_risk_tier_in: [HIGH]
    rationale_template: "High pre-alert risk ({risk_score:.0f}/100); intervention taken; outcome did not occur within {horizon_days}-day window."

  outcome_despite_intervention:
    conditions:
      treatment_strategy_in: [INTERVENTION_TAKEN, ALREADY_ADDRESSED]
      outcome_occurred: true
    rationale_template: "Intervention taken but outcome occurred within {horizon_days}-day window."

  no_effect_detected:
    conditions:
      treatment_strategy_in: [INTERVENTION_TAKEN]
      outcome_occurred: false
      pre_alert_risk_tier_in: [MODERATE, LOW]
    rationale_template: "Pre-alert risk not high enough ({risk_score:.0f}/100) to credibly attribute non-occurrence to intervention."

  fragile_estimate:
    # Override cohort where outcome didn't occur — can't tell if override was right or if outcome would've happened anyway.
    conditions:
      treatment_strategy_in: [OVERRIDE_WITH_REASON, NO_RESPONSE]
      outcome_occurred: false
      pre_alert_risk_tier_in: [HIGH]
    rationale_template: "High-risk alert overridden/unresponded; outcome did not occur but attribution is fragile without propensity adjustment."

  inconclusive:
    # Default when none of the above match, including missing outcome data.
    rationale_template: "Insufficient data for rule-based attribution (outcome status or risk tier missing)."

# Aggregate attribution guardrails — portfolio-level reports must not claim
# more effect than per-patient evidence supports.
portfolio:
  min_records_for_aggregate: 30
  fragile_estimate_max_share: 0.40  # if >40% of records are fragile, flag aggregate as fragile
```

- [ ] **Step 3:** Write 4 failing tests in `rule_attribution_test.go`:

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"kb-26-metabolic-digital-twin/internal/models"
)

func attrInput(strategy string, outcomeOccurred bool, tier string) AttributionInput {
	occurred := outcomeOccurred
	return AttributionInput{
		ConsolidatedRecordID: uuid.New(),
		PatientID:            "P-test",
		CohortID:             "hcf_catalyst_chf",
		TreatmentStrategy:    strategy,
		OutcomeOccurred:      &occurred,
		OutcomeType:          "READMISSION_30D",
		HorizonDays:          30,
		PreAlertRiskScore:    62.0,
		PreAlertRiskTier:     tier,
	}
}

func TestAttribution_HighRiskInterventionNoOutcome_Prevented(t *testing.T) {
	v := ComputeAttribution(attrInput("INTERVENTION_TAKEN", false, "HIGH"))
	if v.ClinicianLabel != string(models.LabelPrevented) {
		t.Fatalf("expected prevented, got %s", v.ClinicianLabel)
	}
	if v.RiskReductionPct <= 0 {
		t.Fatalf("expected positive risk reduction, got %f", v.RiskReductionPct)
	}
}

func TestAttribution_InterventionOutcomeOccurred_Despite(t *testing.T) {
	v := ComputeAttribution(attrInput("INTERVENTION_TAKEN", true, "HIGH"))
	if v.ClinicianLabel != string(models.LabelOutcomeDespiteIntervention) {
		t.Fatalf("expected outcome_despite_intervention, got %s", v.ClinicianLabel)
	}
}

func TestAttribution_LowRiskInterventionNoOutcome_NoEffect(t *testing.T) {
	v := ComputeAttribution(attrInput("INTERVENTION_TAKEN", false, "LOW"))
	if v.ClinicianLabel != string(models.LabelNoEffectDetected) {
		t.Fatalf("expected no_effect_detected, got %s", v.ClinicianLabel)
	}
}

func TestAttribution_HighRiskOverrideNoOutcome_Fragile(t *testing.T) {
	v := ComputeAttribution(attrInput("OVERRIDE_WITH_REASON", false, "HIGH"))
	if v.ClinicianLabel != string(models.LabelFragileEstimate) {
		t.Fatalf("expected fragile_estimate, got %s", v.ClinicianLabel)
	}
}
```

- [ ] **Step 4:** Run tests — expect FAIL (`ComputeAttribution` undefined):

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run TestAttribution -v`
Expected: build failure.

- [ ] **Step 5:** Implement `rule_attribution.go`:

```go
package services

import (
	"fmt"

	"github.com/google/uuid"
	"kb-26-metabolic-digital-twin/internal/models"
)

// AttributionInput carries everything the rule-based engine needs.
// Built from a ConsolidatedAlertRecord (kb-23 Task 3).
type AttributionInput struct {
	ConsolidatedRecordID uuid.UUID
	PatientID            string
	CohortID             string

	TreatmentStrategy    string  // INTERVENTION_TAKEN | OVERRIDE_WITH_REASON | NO_RESPONSE | ALREADY_ADDRESSED
	OutcomeOccurred      *bool   // nil = outcome missing (returns inconclusive)
	OutcomeType          string
	HorizonDays          int

	PreAlertRiskScore    float64
	PreAlertRiskTier     string // HIGH | MODERATE | LOW
}

// ComputeAttribution produces a rule-based AttributionVerdict for one consolidated
// alert record. The counterfactual is the patient's own pre-alert risk score — no
// cohort mean, no propensity model. Sprint 2 replaces this function with IPW/DR
// estimators in KB-28; the returned struct stays identical.
func ComputeAttribution(in AttributionInput) models.AttributionVerdict {
	verdict := models.AttributionVerdict{
		ConsolidatedRecordID: in.ConsolidatedRecordID,
		PatientID:            in.PatientID,
		CohortID:             in.CohortID,
		CounterfactualRisk:   in.PreAlertRiskScore,
		PredictionWindowDays: in.HorizonDays,
		AttributionMethod:    "RULE_BASED",
		MethodVersion:        "sprint1-v1",
	}

	if in.OutcomeOccurred == nil {
		verdict.ClinicianLabel = string(models.LabelInconclusive)
		verdict.TechnicalLabel = "outcome_missing"
		verdict.Rationale = "Outcome status not available — attribution cannot be computed."
		return verdict
	}

	occurred := *in.OutcomeOccurred
	verdict.ObservedOutcome = occurred
	tier := in.PreAlertRiskTier
	ts := in.TreatmentStrategy

	switch {
	case isIntervention(ts) && !occurred && tier == "HIGH":
		verdict.ClinicianLabel = string(models.LabelPrevented)
		verdict.TechnicalLabel = "rule_prevented_high_risk_no_outcome"
		verdict.RiskDifference = in.PreAlertRiskScore
		verdict.RiskReductionPct = in.PreAlertRiskScore
		verdict.Rationale = fmt.Sprintf(
			"High pre-alert risk (%.0f/100); intervention taken; outcome did not occur within %d-day window.",
			in.PreAlertRiskScore, in.HorizonDays)

	case (isIntervention(ts) || ts == "ALREADY_ADDRESSED") && occurred:
		verdict.ClinicianLabel = string(models.LabelOutcomeDespiteIntervention)
		verdict.TechnicalLabel = "rule_outcome_despite_intervention"
		verdict.RiskDifference = 0
		verdict.Rationale = fmt.Sprintf(
			"Intervention taken but outcome occurred within %d-day window.", in.HorizonDays)

	case isIntervention(ts) && !occurred && (tier == "MODERATE" || tier == "LOW"):
		verdict.ClinicianLabel = string(models.LabelNoEffectDetected)
		verdict.TechnicalLabel = "rule_low_baseline_no_attribution"
		verdict.RiskDifference = 0
		verdict.Rationale = fmt.Sprintf(
			"Pre-alert risk not high enough (%.0f/100) to credibly attribute non-occurrence to intervention.",
			in.PreAlertRiskScore)

	case (ts == "OVERRIDE_WITH_REASON" || ts == "NO_RESPONSE") && !occurred && tier == "HIGH":
		verdict.ClinicianLabel = string(models.LabelFragileEstimate)
		verdict.TechnicalLabel = "rule_override_high_risk_no_outcome"
		verdict.RiskDifference = in.PreAlertRiskScore / 2
		verdict.Rationale = "High-risk alert overridden/unresponded; outcome did not occur but attribution is fragile without propensity adjustment."

	default:
		verdict.ClinicianLabel = string(models.LabelInconclusive)
		verdict.TechnicalLabel = "rule_no_matching_case"
		verdict.Rationale = "Insufficient data for rule-based attribution."
	}

	return verdict
}

func isIntervention(strategy string) bool {
	return strategy == "INTERVENTION_TAKEN"
}
```

- [ ] **Step 6:** Run tests — expect PASS:

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run TestAttribution -v`
Expected: 4/4 pass.

- [ ] **Step 7:** Commit:

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/attribution_verdict.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/rule_attribution.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/rule_attribution_test.go \
        backend/shared-infrastructure/market-configs/shared/attribution_parameters.yaml
git commit -m "feat(kb26): rule-based attribution engine + 5-label verdict (Gap 21 Task 4)"
```

---

### Task 5: Governance ledger + attribution API + wiring

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/append_only_ledger.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/append_only_ledger_test.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/attribution_handlers.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/main.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/main.go`

- [ ] **Step 1:** Write 2 failing tests in `append_only_ledger_test.go`:

```go
package services

import (
	"testing"
)

func TestLedger_AppendAndVerifyChain(t *testing.T) {
	ledger := NewInMemoryLedger([]byte("sprint1-test-hmac-key"))
	_, err := ledger.AppendEntry("ATTRIBUTION_RUN", "subject-001", `{"verdict":"prevented"}`)
	if err != nil {
		t.Fatalf("append 1 failed: %v", err)
	}
	_, err = ledger.AppendEntry("ATTRIBUTION_RUN", "subject-002", `{"verdict":"no_effect_detected"}`)
	if err != nil {
		t.Fatalf("append 2 failed: %v", err)
	}
	_, err = ledger.AppendEntry("MODEL_PROMOTION", "gap20-heuristic-v2", `{"from":"v1","to":"v2"}`)
	if err != nil {
		t.Fatalf("append 3 failed: %v", err)
	}

	ok, idx, err := ledger.VerifyChain()
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected chain valid, got first-broken-index=%d", idx)
	}
}

func TestLedger_TamperedEntry_VerifyFails(t *testing.T) {
	ledger := NewInMemoryLedger([]byte("sprint1-test-hmac-key"))
	_, _ = ledger.AppendEntry("ATTRIBUTION_RUN", "s1", `{"a":1}`)
	_, _ = ledger.AppendEntry("ATTRIBUTION_RUN", "s2", `{"a":2}`)
	_, _ = ledger.AppendEntry("ATTRIBUTION_RUN", "s3", `{"a":3}`)

	// Tamper with entry 1's payload in place (simulates post-hoc edit).
	ledger.TamperForTest(1, `{"a":2,"tampered":true}`)

	ok, idx, err := ledger.VerifyChain()
	if err != nil {
		t.Fatalf("verify should not error, got %v", err)
	}
	if ok {
		t.Fatalf("expected tampered chain to be invalid")
	}
	if idx < 1 {
		t.Fatalf("expected break at or after index 1, got %d", idx)
	}
}
```

- [ ] **Step 2:** Run tests — expect FAIL (`NewInMemoryLedger` undefined):

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run TestLedger -v`
Expected: build failure.

- [ ] **Step 3:** Implement `append_only_ledger.go`:

```go
package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"kb-26-metabolic-digital-twin/internal/models"
)

// InMemoryLedger is the Sprint 1 governance ledger. HMAC-SHA256 chain; each entry's
// hash is HMAC(key, prior_hash || entry_type || subject_id || payload_json || sequence || timestamp).
// Sprint 2 replaces storage with PostgreSQL and adds Ed25519 per-entry signatures.
type InMemoryLedger struct {
	mu      sync.Mutex
	key     []byte
	entries []models.LedgerEntry
}

const genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

func NewInMemoryLedger(hmacKey []byte) *InMemoryLedger {
	if len(hmacKey) == 0 {
		hmacKey = []byte("sprint1-default-do-not-use-in-prod")
	}
	return &InMemoryLedger{key: hmacKey}
}

// AppendEntry appends a new entry and returns it with EntryHash and Sequence set.
func (l *InMemoryLedger) AppendEntry(entryType, subjectID, payloadJSON string) (models.LedgerEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	prior := genesisHash
	seq := int64(0)
	if n := len(l.entries); n > 0 {
		prior = l.entries[n-1].EntryHash
		seq = l.entries[n-1].Sequence + 1
	}
	now := time.Now().UTC()
	hash := l.computeEntryHash(prior, entryType, subjectID, payloadJSON, seq, now)

	entry := models.LedgerEntry{
		ID:          uuid.New(),
		Sequence:    seq,
		EntryType:   entryType,
		SubjectID:   subjectID,
		PayloadJSON: payloadJSON,
		PriorHash:   prior,
		EntryHash:   hash,
		CreatedAt:   now,
	}
	l.entries = append(l.entries, entry)
	return entry, nil
}

// VerifyChain walks every entry and recomputes its hash against the recorded prior_hash.
// Returns (true, -1, nil) if valid; (false, first_broken_index, nil) if tampered.
func (l *InMemoryLedger) VerifyChain() (bool, int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	prior := genesisHash
	for i, e := range l.entries {
		expected := l.computeEntryHash(prior, e.EntryType, e.SubjectID, e.PayloadJSON, e.Sequence, e.CreatedAt)
		if !hmac.Equal([]byte(expected), []byte(e.EntryHash)) || e.PriorHash != prior {
			return false, i, nil
		}
		prior = e.EntryHash
	}
	return true, -1, nil
}

// TamperForTest mutates an entry's payload without updating its hash — used by the
// tamper-detection test only. Never call this in production code paths.
func (l *InMemoryLedger) TamperForTest(index int, newPayload string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if index >= 0 && index < len(l.entries) {
		l.entries[index].PayloadJSON = newPayload
	}
}

func (l *InMemoryLedger) Entries() []models.LedgerEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]models.LedgerEntry, len(l.entries))
	copy(out, l.entries)
	return out
}

func (l *InMemoryLedger) computeEntryHash(prior, entryType, subjectID, payloadJSON string, seq int64, ts time.Time) string {
	m := hmac.New(sha256.New, l.key)
	fmt.Fprintf(m, "%s|%s|%s|%s|%d|%s", prior, entryType, subjectID, payloadJSON, seq, ts.Format(time.RFC3339Nano))
	return hex.EncodeToString(m.Sum(nil))
}
```

- [ ] **Step 4:** Run ledger tests — expect PASS:

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run TestLedger -v`
Expected: 2/2 pass.

- [ ] **Step 5:** Create `attribution_handlers.go`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"kb-26-metabolic-digital-twin/internal/services"
)

// POST /api/v1/kb26/attribution/run — run attribution for one consolidated record.
// Body: AttributionInput JSON; returns AttributionVerdict + ledger entry.
func (s *Server) runAttribution(c *gin.Context) {
	var in services.AttributionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	verdict := services.ComputeAttribution(in)

	payload, _ := json.Marshal(verdict)
	entry, err := s.Ledger.AppendEntry("ATTRIBUTION_RUN", verdict.ID.String(), string(payload))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	verdict.LedgerEntryID = &entry.ID

	if s.DB != nil {
		_ = s.DB.Create(&verdict).Error
		_ = s.DB.Create(&entry).Error
	}
	c.JSON(http.StatusOK, gin.H{"verdict": verdict, "ledger_entry": entry})
}

// GET /api/v1/kb26/governance/ledger — return ledger entries (paginated).
func (s *Server) getLedger(c *gin.Context) {
	entries := s.Ledger.Entries()
	ok, brokenIdx, _ := s.Ledger.VerifyChain()
	c.JSON(http.StatusOK, gin.H{
		"entries":          entries,
		"chain_valid":      ok,
		"first_broken_idx": brokenIdx,
		"total":            len(entries),
	})
}
```

- [ ] **Step 6:** Modify kb-26 `routes.go` — add attribution + governance route groups. The existing file already registers a v1 group (see Gap 20 Task 3 `risk.Group("/risk")` at line 90). Add after the risk group:

```go
		// Attribution (Gap 21 — Closed-Loop Outcome Learning)
		attribution := v1.Group("/attribution")
		{
			attribution.POST("/run", s.runAttribution)
		}

		// Governance ledger (Gap 21)
		governance := v1.Group("/governance")
		{
			governance.GET("/ledger", s.getLedger)
		}
```

- [ ] **Step 7:** Modify kb-26 `main.go` — AutoMigrate new models and initialize ledger. Find the existing `AutoMigrate(...)` call (around line 82 per Gap 20 Task 3 which added `&models.PredictedRisk{}`). Extend the migrate list and add ledger init:

```go
		&models.PredictedRisk{},
		&models.AttributionVerdict{},
		&models.LedgerEntry{},
```

Initialize the ledger before starting the server (where the `Server` struct is constructed). Add a `Ledger *services.InMemoryLedger` field to `Server`, then in main wire it:

```go
	ledger := services.NewInMemoryLedger([]byte(os.Getenv("GAP21_LEDGER_HMAC_KEY")))
	srv := api.NewServer(db, ledger)
```

(Adjust `api.NewServer` signature to accept the ledger; default to `NewInMemoryLedger(nil)` in tests.)

- [ ] **Step 8:** Modify kb-23 `main.go` — AutoMigrate new Gap 21 models:

```go
		&models.OutcomeRecord{},
		&models.ConsolidatedAlertRecord{},
```

Add these lines to the existing `AutoMigrate(...)` list next to Gap 19's `DetectionLifecycle{}`.

- [ ] **Step 9:** Build both services and run the full Gap 21 test suite:

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./... && go test ./internal/... -v`
Expected: all Gap 21 Task 2+3 tests pass (6/6).

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./... && go test ./internal/... -v`
Expected: all Gap 21 Task 4+5 tests pass (6/6) plus existing Gap 20 tests still pass.

- [ ] **Step 10:** Commit:

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/append_only_ledger.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/append_only_ledger_test.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/attribution_handlers.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/main.go \
        backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/main.go
git commit -m "feat: complete Gap 21 Sprint 1 — attribution foundation + governance ledger"
```

- [ ] **Step 11:** Push to origin:

```bash
git push origin feature/v4-clinical-gaps
```

---

## Verification Questions

1. Does a high-risk intervention with no outcome return `prevented`? (yes / Task 4 test 1)
2. Does an intervention with the outcome occurring return `outcome_despite_intervention`? (yes / Task 4 test 2)
3. Does a low-risk intervention with no outcome return `no_effect_detected` (not `prevented`)? (yes / Task 4 test 3)
4. Does a high-risk override with no outcome return `fragile_estimate` (not `prevented`)? (yes / Task 4 test 4)
5. Are conflicting outcome sources flagged `CONFLICTED` rather than silently resolved? (yes / Task 2 test 3)
6. Does a tampered ledger entry fail `VerifyChain`? (yes / Task 5 test 2)
7. Is time-zero anchored at T0 (DetectedAt) in every consolidated record? (yes / Task 3 test 1)
8. Does every `ComputeAttribution` call result in exactly one ledger entry? (yes / Task 5 handler)

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Outcome models + YAML | 2 files | 1 hour |
| Task 2: Ingestion + reconciliation (3 tests) | 2 files | 1-2 hours |
| Task 3: Consolidated record + annotator (3 tests) | 3 files | 2 hours |
| Task 4: Rule-based attribution (4 tests) | 4 files | 2-3 hours |
| Task 5: Ledger + API + wiring (2 tests) | 6 files | 2-3 hours |
| **Total** | **~17 files, ~12 tests** | **~8-11 hours** |

---

## Sprint 2+ Deferred Items (Require KB-28 Python + Outcome Data)

| Component | Why deferred | When |
|---|---|---|
| KB-28 Python ML service | Needed for propensity/DR/TMLE | Sprint 2 |
| Propensity score estimator + overlap diagnostics | Requires ML infrastructure | Sprint 2 |
| Doubly-robust estimator (AIPW / TMLE) | Requires propensity + outcome regression | Sprint 2 |
| Counterfactual outcome head (Ŷ(0) model) | Needs override-cohort training data | Sprint 2 (6+ mo of overrides) |
| E-value + tipping-point sensitivity analysis | Needs per-record confidence intervals | Sprint 2 |
| Feedback-aware monitoring (Adherence-Weighted, Sampling-Weighted) | Needs Ŷ(0) | Sprint 3 |
| ADWIN + calibration-CUSUM drift detectors | Needs streaming evaluation infra | Sprint 3 |
| Subgroup + fairness monitoring (equalised odds, calibration-gap) | Needs consented subgroup labels | Sprint 3 |
| Structured clinician feedback ingestion (NLP pipeline) | Needs override-reason taxonomy workshop (§13 open question 6) | Sprint 3 |
| Active learning queue (ensemble CoV + k-center) | Needs Gap 20 ensemble predictions | Sprint 3 |
| Feedback-aware retraining pipeline | Needs labelled feedback + KB-28 | Sprint 4 |
| Shadow → canary → full promotion with rollback | Needs multi-version serving in KB-26 | Sprint 4 |
| Ed25519 signatures on ledger entries | HMAC chain is sufficient for Sprint 1 tamper-evidence | Sprint 4 |
| RACI multi-signature authorisation | Needs user/role model not in scope for Sprint 1 | Sprint 4 |
| Stakeholder / procurement / regulatory artifact generators | Needs 1+ quarter of ledger data | Sprint 5 |
| EU AI Act / FDA PCCP / TGA SaMD mapping docs | Needs legal review | Sprint 5 |

**Transition plan:** When KB-28 Python ships, the `AttributionVerdict` struct and `ClinicianLabel` vocabulary remain unchanged. `ComputeAttribution` is replaced by a client call to KB-28 `/attribute` which returns the same struct with `AttributionMethod="DOUBLY_ROBUST"` or `"TMLE"` instead of `"RULE_BASED"`. The governance ledger, consolidated-record builder, and outcome ingestion all remain identical — same contract, more defensible attribution.
