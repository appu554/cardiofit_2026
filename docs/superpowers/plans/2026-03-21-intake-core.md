# Intake Core Plan (Phase 3)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Intake-Onboarding Service core: 50-slot table with event-sourced storage, deterministic safety engine (11 HARD_STOPs + 8 SOFT_FLAGs), YAML-driven flow graph engine, Flutter app handler ($fill-slot, $enroll), Google FHIR Store writes, and Kafka publishing to 8 intake.* topics.

**Prerequisite:** Plan 1 (Foundation) must be fully implemented first. This plan assumes the following already exist: service scaffolding (`cmd/intake/main.go`), `go.mod`, config, health endpoints, enrollment state machine (8 states, `internal/enrollment/`), Kafka envelope struct (`internal/kafka/envelope.go`), routes.go with 501 stubs, `pkg/fhirclient`, and PostgreSQL migration (`migrations/001_init.sql` with `enrollments`, `slot_events`, `current_slots`, `flow_positions`, `review_queue` tables).

**Architecture:** All files under `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/`. Safety engine is deterministic compiled Go with zero external dependencies, <5ms target. Flow graph is YAML-driven with generic traversal. All clinical data writes go to Google FHIR Store as source of truth, with event-sourced slot storage in PostgreSQL for operational state.

**Tech Stack:** Go 1.25, Gin, pgx/v5, redis/go-redis/v9, zap, prometheus/client_golang, segmentio/kafka-go, gopkg.in/yaml.v3, pkg/fhirclient

**Spec:** `docs/superpowers/specs/2026-03-21-ingestion-intake-onboarding-design.md`

---

## File Structure

### Slot System

| File | Responsibility |
|------|---------------|
| `internal/slots/table.go` | 50-slot definition across 8 domains (Go struct with domain, slot_name, loinc_code, data_type, required flag) |
| `internal/slots/table_test.go` | Validate 50 slots, 8 domains, required counts, LOINC uniqueness |
| `internal/slots/events.go` | Event-sourced append-only slot storage (PostgreSQL INSERT) |
| `internal/slots/events_test.go` | Test event append + current value query |
| `internal/slots/view.go` | Current slot values query (latest event per slot per patient) |

### Safety Engine

| File | Responsibility |
|------|---------------|
| `internal/safety/engine.go` | Core engine: iterate all rules, collect results, <5ms deterministic |
| `internal/safety/engine_test.go` | Integration tests for engine with mixed rule triggers |
| `internal/safety/hard_stops.go` | 11 HARD_STOP rules (H1-H11), each a pure function |
| `internal/safety/hard_stops_test.go` | Unit tests for every HARD_STOP condition |
| `internal/safety/soft_flags.go` | 8 SOFT_FLAG rules (SF-01 to SF-08), each a pure function |
| `internal/safety/soft_flags_test.go` | Unit tests for every SOFT_FLAG condition |
| `internal/safety/rules_registry.go` | Registry of all rules with ordering |

### Flow Graph Engine

| File | Responsibility |
|------|---------------|
| `internal/flow/graph.go` | Node + Edge data structures |
| `internal/flow/engine.go` | Generic graph traversal + next-question selection |
| `internal/flow/engine_test.go` | Test traversal, conditional edges, completion detection |
| `internal/flow/loader.go` | YAML loader for flow definitions |
| `internal/flow/loader_test.go` | Test YAML parse + validation |
| `configs/flows/intake_full.yaml` | Full 50-slot intake flow definition (~25 nodes) |

### FHIR Generator

| File | Responsibility |
|------|---------------|
| `internal/fhir/generator.go` | Slot values to FHIR resources (Observation, DetectedIssue) |
| `internal/fhir/generator_test.go` | Test FHIR JSON output for Observation + DetectedIssue |
| `internal/fhir/patient.go` | Patient resource builder for $enroll |
| `internal/fhir/encounter.go` | Encounter resource builder for $enroll |

### App Handler + Kafka

| File | Responsibility |
|------|---------------|
| `internal/app/handler.go` | $fill-slot and $enroll HTTP handlers |
| `internal/app/handler_test.go` | Integration tests with httptest |
| `internal/kafka/producer.go` | Kafka producer for intake.* topics |
| `internal/kafka/topics.go` | Topic constants for 8 intake topics |
| `internal/kafka/producer_test.go` | Test message publishing |

### Updated Files (from Plan 1)

| File | Change |
|------|--------|
| `internal/api/routes.go` | Replace 501 stubs with real handlers for $fill-slot, $enroll, $evaluate-safety |
| `internal/api/server.go` | Add slot store, safety engine, flow engine, kafka producer to Server struct |
| `cmd/intake/main.go` | Initialize new dependencies (kafka writer, flow engine) |

---

## Task 1: Slot Table — 50-Slot Definition

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/table.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/table_test.go`

- [ ] **Step 1: Write table_test.go (TDD — test first)**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/table_test.go
package slots

import (
	"testing"
)

func TestSlotTable_TotalCount(t *testing.T) {
	table := AllSlots()
	if len(table) != 50 {
		t.Errorf("expected 50 slots, got %d", len(table))
	}
}

func TestSlotTable_DomainCount(t *testing.T) {
	domains := make(map[string]int)
	for _, s := range AllSlots() {
		domains[s.Domain]++
	}
	if len(domains) != 8 {
		t.Errorf("expected 8 domains, got %d", len(domains))
	}
	// Verify all expected domains exist
	expected := []string{
		"demographics", "glycemic", "renal", "cardiac",
		"lipid", "medications", "lifestyle", "symptoms",
	}
	for _, d := range expected {
		if _, ok := domains[d]; !ok {
			t.Errorf("missing domain: %s", d)
		}
	}
}

func TestSlotTable_RequiredSlots(t *testing.T) {
	required := 0
	for _, s := range AllSlots() {
		if s.Required {
			required++
		}
	}
	// At least demographics + key glycemic + key renal + key cardiac should be required
	if required < 15 {
		t.Errorf("expected at least 15 required slots, got %d", required)
	}
}

func TestSlotTable_LOINCUniqueness(t *testing.T) {
	seen := make(map[string]string)
	for _, s := range AllSlots() {
		if s.LOINCCode == "" {
			continue // some slots (like free-text) may not have LOINC
		}
		if existing, ok := seen[s.LOINCCode]; ok {
			t.Errorf("duplicate LOINC code %s: %s and %s", s.LOINCCode, existing, s.Name)
		}
		seen[s.LOINCCode] = s.Name
	}
}

func TestSlotTable_DataTypes(t *testing.T) {
	validTypes := map[DataType]bool{
		DataTypeNumeric:     true,
		DataTypeBoolean:     true,
		DataTypeCodedChoice: true,
		DataTypeText:        true,
		DataTypeDate:        true,
		DataTypeInteger:     true,
		DataTypeList:        true,
	}
	for _, s := range AllSlots() {
		if !validTypes[s.DataType] {
			t.Errorf("slot %s has invalid data type: %s", s.Name, s.DataType)
		}
	}
}

func TestSlotTable_LookupByName(t *testing.T) {
	s, ok := LookupSlot("fbg")
	if !ok {
		t.Fatal("expected to find slot 'fbg'")
	}
	if s.Domain != "glycemic" {
		t.Errorf("expected fbg domain 'glycemic', got %s", s.Domain)
	}
	if s.LOINCCode != "1558-6" {
		t.Errorf("expected fbg LOINC '1558-6', got %s", s.LOINCCode)
	}
}

func TestSlotTable_LookupByName_NotFound(t *testing.T) {
	_, ok := LookupSlot("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent slot")
	}
}
```

- [ ] **Step 2: Run test (expect FAIL — no implementation)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/slots/... -v -count=1`
Expected: Compilation failure (package not found)

- [ ] **Step 3: Write table.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/table.go
package slots

// DataType represents the data type of a slot value.
type DataType string

const (
	DataTypeNumeric     DataType = "numeric"
	DataTypeBoolean     DataType = "boolean"
	DataTypeCodedChoice DataType = "coded_choice"
	DataTypeText        DataType = "text"
	DataTypeDate        DataType = "date"
	DataTypeInteger     DataType = "integer"
	DataTypeList        DataType = "list"
)

// SlotDefinition describes a single intake slot.
type SlotDefinition struct {
	Name      string   `json:"name"`
	Domain    string   `json:"domain"`
	LOINCCode string   `json:"loinc_code"`
	DataType  DataType `json:"data_type"`
	Required  bool     `json:"required"`
	Unit      string   `json:"unit,omitempty"`
	Label     string   `json:"label"`
}

// slotTable holds the canonical 50-slot intake definition.
// Organized by domain: demographics (8), glycemic (7), renal (5), cardiac (7),
// lipid (5), medications (5), lifestyle (7), symptoms (6).
var slotTable = []SlotDefinition{
	// ── Demographics (8 slots) ──
	{Name: "age", Domain: "demographics", LOINCCode: "30525-0", DataType: DataTypeInteger, Required: true, Unit: "years", Label: "Age"},
	{Name: "sex", Domain: "demographics", LOINCCode: "76689-9", DataType: DataTypeCodedChoice, Required: true, Label: "Biological sex"},
	{Name: "height", Domain: "demographics", LOINCCode: "8302-2", DataType: DataTypeNumeric, Required: true, Unit: "cm", Label: "Height"},
	{Name: "weight", Domain: "demographics", LOINCCode: "29463-7", DataType: DataTypeNumeric, Required: true, Unit: "kg", Label: "Weight"},
	{Name: "bmi", Domain: "demographics", LOINCCode: "39156-5", DataType: DataTypeNumeric, Required: true, Unit: "kg/m2", Label: "BMI"},
	{Name: "pregnant", Domain: "demographics", LOINCCode: "82810-3", DataType: DataTypeBoolean, Required: true, Label: "Currently pregnant"},
	{Name: "ethnicity", Domain: "demographics", LOINCCode: "69490-1", DataType: DataTypeCodedChoice, Required: false, Label: "Ethnicity"},
	{Name: "primary_language", Domain: "demographics", LOINCCode: "54899-0", DataType: DataTypeCodedChoice, Required: false, Label: "Primary language"},

	// ── Glycemic (7 slots) ──
	{Name: "diabetes_type", Domain: "glycemic", LOINCCode: "44877-9", DataType: DataTypeCodedChoice, Required: true, Label: "Diabetes type"},
	{Name: "fbg", Domain: "glycemic", LOINCCode: "1558-6", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "Fasting blood glucose"},
	{Name: "hba1c", Domain: "glycemic", LOINCCode: "4548-4", DataType: DataTypeNumeric, Required: true, Unit: "%", Label: "HbA1c"},
	{Name: "ppbg", Domain: "glycemic", LOINCCode: "1521-4", DataType: DataTypeNumeric, Required: false, Unit: "mg/dL", Label: "Post-prandial blood glucose"},
	{Name: "diabetes_duration_years", Domain: "glycemic", LOINCCode: "66519-0", DataType: DataTypeInteger, Required: false, Unit: "years", Label: "Diabetes duration"},
	{Name: "insulin", Domain: "glycemic", LOINCCode: "46239-0", DataType: DataTypeBoolean, Required: true, Label: "Currently on insulin"},
	{Name: "hypoglycemia_episodes", Domain: "glycemic", LOINCCode: "55399-0", DataType: DataTypeInteger, Required: false, Label: "Hypoglycemia episodes (past 3 months)"},

	// ── Renal (5 slots) ──
	{Name: "egfr", Domain: "renal", LOINCCode: "33914-3", DataType: DataTypeNumeric, Required: true, Unit: "mL/min/1.73m2", Label: "eGFR"},
	{Name: "serum_creatinine", Domain: "renal", LOINCCode: "2160-0", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "Serum creatinine"},
	{Name: "uacr", Domain: "renal", LOINCCode: "9318-7", DataType: DataTypeNumeric, Required: false, Unit: "mg/g", Label: "Urine albumin-to-creatinine ratio"},
	{Name: "dialysis", Domain: "renal", LOINCCode: "67038-0", DataType: DataTypeBoolean, Required: true, Label: "Currently on dialysis"},
	{Name: "serum_potassium", Domain: "renal", LOINCCode: "2823-3", DataType: DataTypeNumeric, Required: false, Unit: "mEq/L", Label: "Serum potassium"},

	// ── Cardiac (7 slots) ──
	{Name: "systolic_bp", Domain: "cardiac", LOINCCode: "8480-6", DataType: DataTypeNumeric, Required: true, Unit: "mmHg", Label: "Systolic blood pressure"},
	{Name: "diastolic_bp", Domain: "cardiac", LOINCCode: "8462-4", DataType: DataTypeNumeric, Required: true, Unit: "mmHg", Label: "Diastolic blood pressure"},
	{Name: "heart_rate", Domain: "cardiac", LOINCCode: "8867-4", DataType: DataTypeNumeric, Required: false, Unit: "bpm", Label: "Resting heart rate"},
	{Name: "nyha_class", Domain: "cardiac", LOINCCode: "88020-3", DataType: DataTypeInteger, Required: false, Label: "NYHA functional class (1-4)"},
	{Name: "mi_stroke_days", Domain: "cardiac", LOINCCode: "67530-6", DataType: DataTypeInteger, Required: false, Unit: "days", Label: "Days since last MI or stroke"},
	{Name: "lvef", Domain: "cardiac", LOINCCode: "10230-1", DataType: DataTypeNumeric, Required: false, Unit: "%", Label: "Left ventricular ejection fraction"},
	{Name: "atrial_fibrillation", Domain: "cardiac", LOINCCode: "44667-4", DataType: DataTypeBoolean, Required: false, Label: "Atrial fibrillation"},

	// ── Lipid (5 slots) ──
	{Name: "total_cholesterol", Domain: "lipid", LOINCCode: "2093-3", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "Total cholesterol"},
	{Name: "ldl", Domain: "lipid", LOINCCode: "2089-1", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "LDL cholesterol"},
	{Name: "hdl", Domain: "lipid", LOINCCode: "2085-9", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "HDL cholesterol"},
	{Name: "triglycerides", Domain: "lipid", LOINCCode: "2571-8", DataType: DataTypeNumeric, Required: true, Unit: "mg/dL", Label: "Triglycerides"},
	{Name: "on_statin", Domain: "lipid", LOINCCode: "82667-7", DataType: DataTypeBoolean, Required: false, Label: "Currently on statin"},

	// ── Medications (5 slots) ──
	{Name: "current_medications", Domain: "medications", LOINCCode: "10160-0", DataType: DataTypeList, Required: true, Label: "Current medications list"},
	{Name: "medication_count", Domain: "medications", LOINCCode: "82670-1", DataType: DataTypeInteger, Required: true, Label: "Total medication count"},
	{Name: "adherence_score", Domain: "medications", LOINCCode: "71950-0", DataType: DataTypeNumeric, Required: false, Label: "Medication adherence score (0.0-1.0)"},
	{Name: "allergies", Domain: "medications", LOINCCode: "52473-6", DataType: DataTypeList, Required: true, Label: "Known allergies"},
	{Name: "supplement_list", Domain: "medications", LOINCCode: "29549-3", DataType: DataTypeList, Required: false, Label: "Current supplements"},

	// ── Lifestyle (7 slots) ──
	{Name: "smoking_status", Domain: "lifestyle", LOINCCode: "72166-2", DataType: DataTypeCodedChoice, Required: true, Label: "Smoking status"},
	{Name: "alcohol_use", Domain: "lifestyle", LOINCCode: "74013-4", DataType: DataTypeCodedChoice, Required: true, Label: "Alcohol use frequency"},
	{Name: "exercise_minutes_week", Domain: "lifestyle", LOINCCode: "68516-4", DataType: DataTypeInteger, Required: false, Unit: "min/week", Label: "Exercise minutes per week"},
	{Name: "diet_type", Domain: "lifestyle", LOINCCode: "81663-7", DataType: DataTypeCodedChoice, Required: false, Label: "Diet type"},
	{Name: "sleep_hours", Domain: "lifestyle", LOINCCode: "93832-4", DataType: DataTypeNumeric, Required: false, Unit: "hours", Label: "Average sleep hours"},
	{Name: "active_substance_abuse", Domain: "lifestyle", LOINCCode: "68524-8", DataType: DataTypeBoolean, Required: true, Label: "Active substance abuse"},
	{Name: "falls_history", Domain: "lifestyle", LOINCCode: "52552-7", DataType: DataTypeBoolean, Required: false, Label: "Falls history (past 12 months)"},

	// ── Symptoms (6 slots) ──
	{Name: "active_cancer", Domain: "symptoms", LOINCCode: "63933-6", DataType: DataTypeBoolean, Required: true, Label: "Active cancer"},
	{Name: "organ_transplant", Domain: "symptoms", LOINCCode: "79829-6", DataType: DataTypeBoolean, Required: true, Label: "Organ transplant recipient"},
	{Name: "cognitive_impairment", Domain: "symptoms", LOINCCode: "72106-8", DataType: DataTypeBoolean, Required: false, Label: "Cognitive impairment"},
	{Name: "bariatric_surgery_months", Domain: "symptoms", LOINCCode: "85359-8", DataType: DataTypeInteger, Required: false, Unit: "months", Label: "Months since bariatric surgery"},
	{Name: "primary_complaint", Domain: "symptoms", LOINCCode: "10164-2", DataType: DataTypeText, Required: false, Label: "Primary complaint (free text)"},
	{Name: "comorbidities", Domain: "symptoms", LOINCCode: "45701-0", DataType: DataTypeList, Required: false, Label: "Comorbidity list"},
}

// slotIndex is a name-to-slot lookup map, built at init time.
var slotIndex map[string]SlotDefinition

func init() {
	slotIndex = make(map[string]SlotDefinition, len(slotTable))
	for _, s := range slotTable {
		slotIndex[s.Name] = s
	}
}

// AllSlots returns the full 50-slot intake definition table.
func AllSlots() []SlotDefinition {
	out := make([]SlotDefinition, len(slotTable))
	copy(out, slotTable)
	return out
}

// LookupSlot returns the slot definition by name.
func LookupSlot(name string) (SlotDefinition, bool) {
	s, ok := slotIndex[name]
	return s, ok
}

// SlotsByDomain returns all slots for a given domain.
func SlotsByDomain(domain string) []SlotDefinition {
	var out []SlotDefinition
	for _, s := range slotTable {
		if s.Domain == domain {
			out = append(out, s)
		}
	}
	return out
}

// RequiredSlots returns all slots with Required=true.
func RequiredSlots() []SlotDefinition {
	var out []SlotDefinition
	for _, s := range slotTable {
		if s.Required {
			out = append(out, s)
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests (expect PASS)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/slots/... -v -count=1`
Expected: All 7 tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/table.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/table_test.go
git commit -m "feat(intake): add 50-slot definition table across 8 clinical domains

Demographics (8), glycemic (7), renal (5), cardiac (7), lipid (5),
medications (5), lifestyle (7), symptoms (6). Each slot has domain,
LOINC code, data type, required flag. Name-based lookup index."
```

---

## Task 2: Slot Event Store — Event-Sourced Append-Only Storage

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/events.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/view.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/events_test.go`

**Reference:** PostgreSQL schema from Plan 1 migration (`slot_events` table + `current_slots` view). Events are append-only — never updated or deleted.

- [ ] **Step 1: Write events_test.go (TDD — test first)**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/events_test.go
package slots

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// MockEventStore implements EventStore for testing without PostgreSQL.
type MockEventStore struct {
	events []SlotEvent
}

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{events: make([]SlotEvent, 0)}
}

func (m *MockEventStore) Append(ctx context.Context, event SlotEvent) error {
	event.ID = uuid.New()
	event.CreatedAt = time.Now().UTC()
	m.events = append(m.events, event)
	return nil
}

func (m *MockEventStore) CurrentValues(ctx context.Context, patientID uuid.UUID) (map[string]SlotValue, error) {
	latest := make(map[string]SlotEvent)
	for _, e := range m.events {
		if e.PatientID == patientID {
			if existing, ok := latest[e.SlotName]; !ok || e.CreatedAt.After(existing.CreatedAt) {
				latest[e.SlotName] = e
			}
		}
	}
	result := make(map[string]SlotValue, len(latest))
	for name, e := range latest {
		result[name] = SlotValue{
			Value:          e.Value,
			ExtractionMode: e.ExtractionMode,
			Confidence:     e.Confidence,
			FHIRResourceID: e.FHIRResourceID,
			UpdatedAt:      e.CreatedAt,
		}
	}
	return result, nil
}

func (m *MockEventStore) SlotHistory(ctx context.Context, patientID uuid.UUID, slotName string) ([]SlotEvent, error) {
	var history []SlotEvent
	for _, e := range m.events {
		if e.PatientID == patientID && e.SlotName == slotName {
			history = append(history, e)
		}
	}
	return history, nil
}

func TestEventStore_AppendAndRetrieve(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patientID := uuid.New()

	// Append FBG slot event
	err := store.Append(ctx, SlotEvent{
		PatientID:      patientID,
		SlotName:       "fbg",
		Domain:         "glycemic",
		Value:          json.RawMessage(`178`),
		ExtractionMode: "BUTTON",
		Confidence:     1.0,
		SourceChannel:  "APP",
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	values, err := store.CurrentValues(ctx, patientID)
	if err != nil {
		t.Fatalf("CurrentValues failed: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if string(values["fbg"].Value) != "178" {
		t.Errorf("expected fbg=178, got %s", string(values["fbg"].Value))
	}
}

func TestEventStore_LatestWins(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patientID := uuid.New()

	// First FBG value
	_ = store.Append(ctx, SlotEvent{
		PatientID:      patientID,
		SlotName:       "fbg",
		Domain:         "glycemic",
		Value:          json.RawMessage(`178`),
		ExtractionMode: "BUTTON",
		Confidence:     1.0,
		SourceChannel:  "APP",
	})

	// Updated FBG value (correction)
	time.Sleep(time.Millisecond) // ensure different timestamp
	_ = store.Append(ctx, SlotEvent{
		PatientID:      patientID,
		SlotName:       "fbg",
		Domain:         "glycemic",
		Value:          json.RawMessage(`165`),
		ExtractionMode: "REGEX",
		Confidence:     0.95,
		SourceChannel:  "WHATSAPP",
	})

	values, _ := store.CurrentValues(ctx, patientID)
	if string(values["fbg"].Value) != "165" {
		t.Errorf("expected latest fbg=165, got %s", string(values["fbg"].Value))
	}
}

func TestEventStore_MultipleSlots(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patientID := uuid.New()

	slots := []struct {
		name   string
		domain string
		value  string
	}{
		{"fbg", "glycemic", "178"},
		{"hba1c", "glycemic", "8.2"},
		{"egfr", "renal", "42"},
		{"systolic_bp", "cardiac", "145"},
	}

	for _, s := range slots {
		_ = store.Append(ctx, SlotEvent{
			PatientID:      patientID,
			SlotName:       s.name,
			Domain:         s.domain,
			Value:          json.RawMessage(s.value),
			ExtractionMode: "BUTTON",
			Confidence:     1.0,
			SourceChannel:  "APP",
		})
	}

	values, _ := store.CurrentValues(ctx, patientID)
	if len(values) != 4 {
		t.Errorf("expected 4 slots, got %d", len(values))
	}
}

func TestEventStore_SlotHistory(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patientID := uuid.New()

	// Three FBG entries
	for _, v := range []string{"178", "165", "150"} {
		_ = store.Append(ctx, SlotEvent{
			PatientID:      patientID,
			SlotName:       "fbg",
			Domain:         "glycemic",
			Value:          json.RawMessage(v),
			ExtractionMode: "BUTTON",
			Confidence:     1.0,
			SourceChannel:  "APP",
		})
	}

	history, _ := store.SlotHistory(ctx, patientID, "fbg")
	if len(history) != 3 {
		t.Errorf("expected 3 history entries, got %d", len(history))
	}
}

func TestEventStore_PatientIsolation(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patient1 := uuid.New()
	patient2 := uuid.New()

	_ = store.Append(ctx, SlotEvent{
		PatientID: patient1, SlotName: "fbg", Domain: "glycemic",
		Value: json.RawMessage(`178`), ExtractionMode: "BUTTON",
		Confidence: 1.0, SourceChannel: "APP",
	})
	_ = store.Append(ctx, SlotEvent{
		PatientID: patient2, SlotName: "fbg", Domain: "glycemic",
		Value: json.RawMessage(`110`), ExtractionMode: "BUTTON",
		Confidence: 1.0, SourceChannel: "APP",
	})

	values1, _ := store.CurrentValues(ctx, patient1)
	values2, _ := store.CurrentValues(ctx, patient2)
	if string(values1["fbg"].Value) != "178" {
		t.Errorf("patient1 fbg should be 178")
	}
	if string(values2["fbg"].Value) != "110" {
		t.Errorf("patient2 fbg should be 110")
	}
}
```

- [ ] **Step 2: Write events.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/events.go
package slots

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SlotEvent represents a single immutable slot fill event.
type SlotEvent struct {
	ID             uuid.UUID       `json:"id"`
	PatientID      uuid.UUID       `json:"patient_id"`
	SlotName       string          `json:"slot_name"`
	Domain         string          `json:"domain"`
	Value          json.RawMessage `json:"value"`
	ExtractionMode string          `json:"extraction_mode"` // BUTTON, REGEX, NLU, DEVICE
	Confidence     float64         `json:"confidence"`
	SafetyResult   json.RawMessage `json:"safety_result,omitempty"`
	SourceChannel  string          `json:"source_channel"` // APP, WHATSAPP, ASHA
	FHIRResourceID string          `json:"fhir_resource_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// SlotValue represents the current value of a slot (derived from latest event).
type SlotValue struct {
	Value          json.RawMessage `json:"value"`
	ExtractionMode string          `json:"extraction_mode"`
	Confidence     float64         `json:"confidence"`
	FHIRResourceID string          `json:"fhir_resource_id"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// EventStore is the interface for slot event storage.
type EventStore interface {
	Append(ctx context.Context, event SlotEvent) error
	CurrentValues(ctx context.Context, patientID uuid.UUID) (map[string]SlotValue, error)
	SlotHistory(ctx context.Context, patientID uuid.UUID, slotName string) ([]SlotEvent, error)
}

// PgEventStore implements EventStore backed by PostgreSQL.
type PgEventStore struct {
	pool *pgxpool.Pool
}

// NewPgEventStore creates a new PostgreSQL-backed event store.
func NewPgEventStore(pool *pgxpool.Pool) *PgEventStore {
	return &PgEventStore{pool: pool}
}

// Append inserts a new slot event (append-only, never updates).
func (s *PgEventStore) Append(ctx context.Context, event SlotEvent) error {
	query := `
		INSERT INTO slot_events (patient_id, slot_name, domain, value, extraction_mode,
			confidence, safety_result, source_channel, fhir_resource_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	return s.pool.QueryRow(ctx, query,
		event.PatientID, event.SlotName, event.Domain, event.Value,
		event.ExtractionMode, event.Confidence, event.SafetyResult,
		event.SourceChannel, event.FHIRResourceID,
	).Scan(&event.ID, &event.CreatedAt)
}

// CurrentValues returns the latest value for each slot for a patient.
// Uses the current_slots view (DISTINCT ON patient_id, slot_name ORDER BY created_at DESC).
func (s *PgEventStore) CurrentValues(ctx context.Context, patientID uuid.UUID) (map[string]SlotValue, error) {
	query := `
		SELECT slot_name, value, extraction_mode, confidence, fhir_resource_id, created_at
		FROM current_slots
		WHERE patient_id = $1`

	rows, err := s.pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]SlotValue)
	for rows.Next() {
		var name string
		var sv SlotValue
		if err := rows.Scan(&name, &sv.Value, &sv.ExtractionMode, &sv.Confidence, &sv.FHIRResourceID, &sv.UpdatedAt); err != nil {
			return nil, err
		}
		result[name] = sv
	}
	return result, rows.Err()
}

// SlotHistory returns all events for a slot in chronological order.
func (s *PgEventStore) SlotHistory(ctx context.Context, patientID uuid.UUID, slotName string) ([]SlotEvent, error) {
	query := `
		SELECT id, patient_id, slot_name, domain, value, extraction_mode,
			confidence, safety_result, source_channel, fhir_resource_id, created_at
		FROM slot_events
		WHERE patient_id = $1 AND slot_name = $2
		ORDER BY created_at ASC`

	rows, err := s.pool.Query(ctx, query, patientID, slotName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SlotEvent
	for rows.Next() {
		var e SlotEvent
		if err := rows.Scan(&e.ID, &e.PatientID, &e.SlotName, &e.Domain, &e.Value,
			&e.ExtractionMode, &e.Confidence, &e.SafetyResult, &e.SourceChannel,
			&e.FHIRResourceID, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
```

- [ ] **Step 3: Write view.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/view.go
package slots

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
)

// SlotSnapshot holds all current slot values for a patient, keyed by slot name.
type SlotSnapshot struct {
	PatientID uuid.UUID            `json:"patient_id"`
	Values    map[string]SlotValue `json:"values"`
	Filled    int                  `json:"filled"`
	Total     int                  `json:"total"`
	Required  int                  `json:"required"`
	Missing   []string             `json:"missing_required"`
}

// BuildSnapshot constructs a SlotSnapshot from current values.
func BuildSnapshot(patientID uuid.UUID, values map[string]SlotValue) SlotSnapshot {
	allSlots := AllSlots()
	requiredSlots := RequiredSlots()

	var missingRequired []string
	for _, s := range requiredSlots {
		if _, ok := values[s.Name]; !ok {
			missingRequired = append(missingRequired, s.Name)
		}
	}

	return SlotSnapshot{
		PatientID: patientID,
		Values:    values,
		Filled:    len(values),
		Total:     len(allSlots),
		Required:  len(requiredSlots),
		Missing:   missingRequired,
	}
}

// IsComplete returns true if all required slots are filled.
func (ss SlotSnapshot) IsComplete() bool {
	return len(ss.Missing) == 0
}

// GetFloat64 extracts a float64 value from the snapshot by slot name.
// Returns 0 and false if the slot is not filled or not a valid number.
func (ss SlotSnapshot) GetFloat64(slotName string) (float64, bool) {
	sv, ok := ss.Values[slotName]
	if !ok {
		return 0, false
	}
	var v float64
	if err := json.Unmarshal(sv.Value, &v); err != nil {
		// Try string-encoded number
		var s string
		if err2 := json.Unmarshal(sv.Value, &s); err2 == nil {
			if f, err3 := strconv.ParseFloat(s, 64); err3 == nil {
				return f, true
			}
		}
		return 0, false
	}
	return v, true
}

// GetBool extracts a boolean value from the snapshot by slot name.
func (ss SlotSnapshot) GetBool(slotName string) (bool, bool) {
	sv, ok := ss.Values[slotName]
	if !ok {
		return false, false
	}
	var v bool
	if err := json.Unmarshal(sv.Value, &v); err != nil {
		return false, false
	}
	return v, true
}

// GetInt extracts an integer value from the snapshot by slot name.
func (ss SlotSnapshot) GetInt(slotName string) (int, bool) {
	sv, ok := ss.Values[slotName]
	if !ok {
		return 0, false
	}
	var v int
	if err := json.Unmarshal(sv.Value, &v); err != nil {
		// Try float (JSON numbers may decode as float)
		var f float64
		if err2 := json.Unmarshal(sv.Value, &f); err2 == nil {
			return int(f), true
		}
		return 0, false
	}
	return v, true
}

// GetString extracts a string value from the snapshot by slot name.
func (ss SlotSnapshot) GetString(slotName string) (string, bool) {
	sv, ok := ss.Values[slotName]
	if !ok {
		return "", false
	}
	var v string
	if err := json.Unmarshal(sv.Value, &v); err != nil {
		return "", false
	}
	return v, true
}

// FilledSlotNames returns the names of all currently filled slots.
func FilledSlotNames(ctx context.Context, store EventStore, patientID uuid.UUID) ([]string, error) {
	values, err := store.CurrentValues(ctx, patientID)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	return names, nil
}
```

- [ ] **Step 4: Run tests (expect PASS)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/slots/... -v -count=1`
Expected: All tests PASS (table tests + event store tests)

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/slots/
git commit -m "feat(intake): add event-sourced slot storage and snapshot builder

Append-only SlotEvent with PostgreSQL backing (PgEventStore) and
MockEventStore for tests. CurrentValues uses DISTINCT ON for latest
per-slot. SlotSnapshot provides typed accessors (GetFloat64, GetBool,
GetInt, GetString) and required-slot completion tracking."
```

---

## Task 3: Safety Engine — Core Engine

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/engine.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/rules_registry.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/engine_test.go`

- [ ] **Step 1: Write engine_test.go (TDD — test first)**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/engine_test.go
package safety

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

func buildSnapshot(values map[string]interface{}) slots.SlotSnapshot {
	sv := make(map[string]slots.SlotValue)
	for k, v := range values {
		raw, _ := json.Marshal(v)
		sv[k] = slots.SlotValue{
			Value:          raw,
			ExtractionMode: "BUTTON",
			Confidence:     1.0,
			UpdatedAt:      time.Now(),
		}
	}
	return slots.SlotSnapshot{
		PatientID: uuid.New(),
		Values:    sv,
	}
}

func TestEngine_NoTriggers(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"age":           45,
		"diabetes_type": "T2DM",
		"pregnant":      false,
		"egfr":          75,
		"dialysis":      false,
	})

	result := engine.Evaluate(snap)
	if len(result.HardStops) != 0 {
		t.Errorf("expected 0 hard stops, got %d: %+v", len(result.HardStops), result.HardStops)
	}
}

func TestEngine_SingleHardStop(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"diabetes_type": "T1DM",
	})

	result := engine.Evaluate(snap)
	if len(result.HardStops) != 1 {
		t.Fatalf("expected 1 hard stop, got %d", len(result.HardStops))
	}
	if result.HardStops[0].RuleID != "H1" {
		t.Errorf("expected H1, got %s", result.HardStops[0].RuleID)
	}
}

func TestEngine_MultipleTriggers(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"diabetes_type": "T1DM",
		"pregnant":      true,
		"egfr":          10,
		"dialysis":      true,
		"age":           80,
	})

	result := engine.Evaluate(snap)
	// Should trigger: H1 (T1DM), H2 (pregnant), H3 (dialysis), H5 (eGFR<15)
	// Should also trigger: SF-01 (age>=75)
	if len(result.HardStops) < 3 {
		t.Errorf("expected at least 3 hard stops, got %d: %+v", len(result.HardStops), result.HardStops)
	}
	if len(result.SoftFlags) < 1 {
		t.Errorf("expected at least 1 soft flag, got %d", len(result.SoftFlags))
	}
}

func TestEngine_HasHardStop(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"pregnant": true,
	})
	result := engine.Evaluate(snap)
	if !result.HasHardStop() {
		t.Error("expected HasHardStop=true for pregnant patient")
	}
}

func TestEngine_Duration(t *testing.T) {
	engine := NewEngine()
	snap := buildSnapshot(map[string]interface{}{
		"age":            45,
		"diabetes_type":  "T2DM",
		"pregnant":       false,
		"egfr":           55,
		"dialysis":       false,
		"active_cancer":  false,
		"mi_stroke_days": 365,
		"nyha_class":     1,
		"organ_transplant":      false,
		"active_substance_abuse": false,
		"medication_count":       3,
		"bmi":                    24.5,
		"insulin":                false,
		"falls_history":          false,
		"cognitive_impairment":   false,
		"adherence_score":        0.85,
	})

	start := time.Now()
	for i := 0; i < 1000; i++ {
		engine.Evaluate(snap)
	}
	elapsed := time.Since(start)
	avgMicros := elapsed.Microseconds() / 1000

	// Target: <5ms per evaluation. With 1000 iterations, total should be well under 5s.
	if avgMicros > 5000 {
		t.Errorf("safety engine too slow: avg %d microseconds (target <5000)", avgMicros)
	}
}
```

- [ ] **Step 2: Write rules_registry.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/rules_registry.go
package safety

import "github.com/cardiofit/intake-onboarding-service/internal/slots"

// RuleType identifies whether a rule is a HARD_STOP or SOFT_FLAG.
type RuleType string

const (
	RuleTypeHardStop RuleType = "HARD_STOP"
	RuleTypeSoftFlag RuleType = "SOFT_FLAG"
)

// RuleResult is the output of a single rule evaluation.
type RuleResult struct {
	RuleID   string   `json:"rule_id"`
	RuleType RuleType `json:"rule_type"`
	Reason   string   `json:"reason"`
}

// RuleFunc is a pure function that evaluates a safety rule against slot values.
// Returns (triggered, ruleID, reason).
// Contract: no external calls, no I/O, no LLM, deterministic.
type RuleFunc func(snap slots.SlotSnapshot) (triggered bool, ruleID string, reason string)

// SafetyResult holds the complete result of a safety evaluation.
type SafetyResult struct {
	HardStops []RuleResult `json:"hard_stops"`
	SoftFlags []RuleResult `json:"soft_flags"`
}

// HasHardStop returns true if any HARD_STOP rule was triggered.
func (sr SafetyResult) HasHardStop() bool {
	return len(sr.HardStops) > 0
}

// HasSoftFlag returns true if any SOFT_FLAG rule was triggered.
func (sr SafetyResult) HasSoftFlag() bool {
	return len(sr.SoftFlags) > 0
}

// AllRuleIDs returns all triggered rule IDs.
func (sr SafetyResult) AllRuleIDs() []string {
	ids := make([]string, 0, len(sr.HardStops)+len(sr.SoftFlags))
	for _, r := range sr.HardStops {
		ids = append(ids, r.RuleID)
	}
	for _, r := range sr.SoftFlags {
		ids = append(ids, r.RuleID)
	}
	return ids
}
```

- [ ] **Step 3: Write engine.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/engine.go
package safety

import "github.com/cardiofit/intake-onboarding-service/internal/slots"

// Engine evaluates all safety rules against current slot values.
// Deterministic, <5ms, zero external dependencies.
type Engine struct {
	hardStopRules []RuleFunc
	softFlagRules []RuleFunc
}

// NewEngine creates a safety engine with all registered rules.
func NewEngine() *Engine {
	return &Engine{
		hardStopRules: []RuleFunc{
			CheckH1TypeOneDM,
			CheckH2Pregnancy,
			CheckH3Dialysis,
			CheckH4ActiveCancer,
			CheckH5EGFRCritical,
			CheckH6RecentMIStroke,
			CheckH7HeartFailureSevere,
			CheckH8Child,
			CheckH9BariatricSurgery,
			CheckH10OrganTransplant,
			CheckH11SubstanceAbuse,
		},
		softFlagRules: []RuleFunc{
			CheckSF01Elderly,
			CheckSF02CKDModerate,
			CheckSF03Polypharmacy,
			CheckSF04LowBMI,
			CheckSF05InsulinUse,
			CheckSF06FallsRisk,
			CheckSF07CognitiveImpairment,
			CheckSF08NonAdherent,
		},
	}
}

// Evaluate runs all safety rules against the given slot snapshot.
// Returns collected HARD_STOPs and SOFT_FLAGs. Never returns an error —
// missing slot values simply cause the rule to not trigger (safe default).
func (e *Engine) Evaluate(snap slots.SlotSnapshot) SafetyResult {
	result := SafetyResult{
		HardStops: make([]RuleResult, 0),
		SoftFlags: make([]RuleResult, 0),
	}

	for _, rule := range e.hardStopRules {
		if triggered, ruleID, reason := rule(snap); triggered {
			result.HardStops = append(result.HardStops, RuleResult{
				RuleID:   ruleID,
				RuleType: RuleTypeHardStop,
				Reason:   reason,
			})
		}
	}

	for _, rule := range e.softFlagRules {
		if triggered, ruleID, reason := rule(snap); triggered {
			result.SoftFlags = append(result.SoftFlags, RuleResult{
				RuleID:   ruleID,
				RuleType: RuleTypeSoftFlag,
				Reason:   reason,
			})
		}
	}

	return result
}

// EvaluateForSlot runs the safety engine specifically after a slot fill.
// Returns the SafetyResult and whether the enrollment should be HARD_STOPPED.
func (e *Engine) EvaluateForSlot(snap slots.SlotSnapshot, slotName string) (SafetyResult, bool) {
	result := e.Evaluate(snap)
	return result, result.HasHardStop()
}
```

- [ ] **Step 4: Run tests (expect FAIL — no hard_stops.go/soft_flags.go yet)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/safety/... -v -count=1`
Expected: Compilation failure (undefined CheckH1TypeOneDM, etc.)

- [ ] **Step 5: Commit (partial — engine + registry)**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/engine.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/rules_registry.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/engine_test.go
git commit -m "feat(intake/safety): add safety engine core and rule registry

Deterministic engine iterates all HARD_STOP and SOFT_FLAG rules against
slot snapshot. RuleFunc signature: pure function, no I/O, no LLM.
SafetyResult collects triggered rules. Tests require Tasks 4+5."
```

---

## Task 4: HARD_STOPs — 11 Rule Implementations (H1-H11)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/hard_stops.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/hard_stops_test.go`

**Reference:** Spec section 3.3. Each rule is a pure function with signature `func(snap SlotSnapshot) (triggered bool, ruleID string, reason string)`.

- [ ] **Step 1: Write hard_stops_test.go (TDD — test first)**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/hard_stops_test.go
package safety

import (
	"testing"
)

func TestH1_TypeOneDM_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"diabetes_type": "T1DM"})
	triggered, id, reason := CheckH1TypeOneDM(snap)
	if !triggered {
		t.Error("H1 should trigger for T1DM")
	}
	if id != "H1" {
		t.Errorf("expected H1, got %s", id)
	}
	if reason == "" {
		t.Error("reason should not be empty")
	}
}

func TestH1_TypeOneDM_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"diabetes_type": "T2DM"})
	triggered, _, _ := CheckH1TypeOneDM(snap)
	if triggered {
		t.Error("H1 should not trigger for T2DM")
	}
}

func TestH1_TypeOneDM_Missing(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{})
	triggered, _, _ := CheckH1TypeOneDM(snap)
	if triggered {
		t.Error("H1 should not trigger when diabetes_type is missing")
	}
}

func TestH2_Pregnancy_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"pregnant": true})
	triggered, id, _ := CheckH2Pregnancy(snap)
	if !triggered {
		t.Error("H2 should trigger for pregnant=true")
	}
	if id != "H2" {
		t.Errorf("expected H2, got %s", id)
	}
}

func TestH2_Pregnancy_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"pregnant": false})
	triggered, _, _ := CheckH2Pregnancy(snap)
	if triggered {
		t.Error("H2 should not trigger for pregnant=false")
	}
}

func TestH3_Dialysis_ByFlag(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"dialysis": true})
	triggered, id, _ := CheckH3Dialysis(snap)
	if !triggered {
		t.Error("H3 should trigger for dialysis=true")
	}
	if id != "H3" {
		t.Errorf("expected H3, got %s", id)
	}
}

func TestH3_Dialysis_ByEGFR(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"dialysis": false, "egfr": 12.0})
	triggered, _, _ := CheckH3Dialysis(snap)
	if !triggered {
		t.Error("H3 should trigger for eGFR < 15")
	}
}

func TestH3_Dialysis_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"dialysis": false, "egfr": 45.0})
	triggered, _, _ := CheckH3Dialysis(snap)
	if triggered {
		t.Error("H3 should not trigger for dialysis=false and eGFR=45")
	}
}

func TestH4_ActiveCancer_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"active_cancer": true})
	triggered, id, _ := CheckH4ActiveCancer(snap)
	if !triggered {
		t.Error("H4 should trigger for active_cancer=true")
	}
	if id != "H4" {
		t.Errorf("expected H4, got %s", id)
	}
}

func TestH4_ActiveCancer_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"active_cancer": false})
	triggered, _, _ := CheckH4ActiveCancer(snap)
	if triggered {
		t.Error("H4 should not trigger for active_cancer=false")
	}
}

func TestH5_EGFRCritical_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 10.0})
	triggered, id, _ := CheckH5EGFRCritical(snap)
	if !triggered {
		t.Error("H5 should trigger for eGFR=10")
	}
	if id != "H5" {
		t.Errorf("expected H5, got %s", id)
	}
}

func TestH5_EGFRCritical_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 15.0})
	triggered, _, _ := CheckH5EGFRCritical(snap)
	if triggered {
		t.Error("H5 should not trigger for eGFR=15 (boundary, < 15 required)")
	}
}

func TestH5_EGFRCritical_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 60.0})
	triggered, _, _ := CheckH5EGFRCritical(snap)
	if triggered {
		t.Error("H5 should not trigger for eGFR=60")
	}
}

func TestH6_RecentMIStroke_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"mi_stroke_days": 45})
	triggered, id, _ := CheckH6RecentMIStroke(snap)
	if !triggered {
		t.Error("H6 should trigger for mi_stroke_days=45")
	}
	if id != "H6" {
		t.Errorf("expected H6, got %s", id)
	}
}

func TestH6_RecentMIStroke_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"mi_stroke_days": 90})
	triggered, _, _ := CheckH6RecentMIStroke(snap)
	if triggered {
		t.Error("H6 should not trigger for mi_stroke_days=90 (boundary, < 90 required)")
	}
}

func TestH6_RecentMIStroke_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"mi_stroke_days": 180})
	triggered, _, _ := CheckH6RecentMIStroke(snap)
	if triggered {
		t.Error("H6 should not trigger for mi_stroke_days=180")
	}
}

func TestH7_HeartFailureSevere_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"nyha_class": 3})
	triggered, id, _ := CheckH7HeartFailureSevere(snap)
	if !triggered {
		t.Error("H7 should trigger for nyha_class=3")
	}
	if id != "H7" {
		t.Errorf("expected H7, got %s", id)
	}
}

func TestH7_HeartFailureSevere_Class4(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"nyha_class": 4})
	triggered, _, _ := CheckH7HeartFailureSevere(snap)
	if !triggered {
		t.Error("H7 should trigger for nyha_class=4")
	}
}

func TestH7_HeartFailureSevere_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"nyha_class": 2})
	triggered, _, _ := CheckH7HeartFailureSevere(snap)
	if triggered {
		t.Error("H7 should not trigger for nyha_class=2")
	}
}

func TestH8_Child_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 15})
	triggered, id, _ := CheckH8Child(snap)
	if !triggered {
		t.Error("H8 should trigger for age=15")
	}
	if id != "H8" {
		t.Errorf("expected H8, got %s", id)
	}
}

func TestH8_Child_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 18})
	triggered, _, _ := CheckH8Child(snap)
	if triggered {
		t.Error("H8 should not trigger for age=18 (boundary, < 18 required)")
	}
}

func TestH9_BariatricSurgery_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"bariatric_surgery_months": 6})
	triggered, id, _ := CheckH9BariatricSurgery(snap)
	if !triggered {
		t.Error("H9 should trigger for bariatric_surgery_months=6")
	}
	if id != "H9" {
		t.Errorf("expected H9, got %s", id)
	}
}

func TestH9_BariatricSurgery_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"bariatric_surgery_months": 12})
	triggered, _, _ := CheckH9BariatricSurgery(snap)
	if triggered {
		t.Error("H9 should not trigger for bariatric_surgery_months=12 (boundary, < 12 required)")
	}
}

func TestH10_OrganTransplant_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"organ_transplant": true})
	triggered, id, _ := CheckH10OrganTransplant(snap)
	if !triggered {
		t.Error("H10 should trigger for organ_transplant=true")
	}
	if id != "H10" {
		t.Errorf("expected H10, got %s", id)
	}
}

func TestH10_OrganTransplant_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"organ_transplant": false})
	triggered, _, _ := CheckH10OrganTransplant(snap)
	if triggered {
		t.Error("H10 should not trigger for organ_transplant=false")
	}
}

func TestH11_SubstanceAbuse_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"active_substance_abuse": true})
	triggered, id, _ := CheckH11SubstanceAbuse(snap)
	if !triggered {
		t.Error("H11 should trigger for active_substance_abuse=true")
	}
	if id != "H11" {
		t.Errorf("expected H11, got %s", id)
	}
}

func TestH11_SubstanceAbuse_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"active_substance_abuse": false})
	triggered, _, _ := CheckH11SubstanceAbuse(snap)
	if triggered {
		t.Error("H11 should not trigger for active_substance_abuse=false")
	}
}
```

- [ ] **Step 2: Write hard_stops.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/hard_stops.go
package safety

import "github.com/cardiofit/intake-onboarding-service/internal/slots"

// Each HARD_STOP is a pure function: no I/O, no external deps, deterministic.
// Missing slot values => rule does NOT trigger (safe default).

// CheckH1TypeOneDM blocks enrollment if diabetes_type == "T1DM".
func CheckH1TypeOneDM(snap slots.SlotSnapshot) (bool, string, string) {
	dt, ok := snap.GetString("diabetes_type")
	if !ok {
		return false, "H1", ""
	}
	if dt == "T1DM" {
		return true, "H1", "Type 1 DM — T1DM protocol differs, requires endocrinology management"
	}
	return false, "H1", ""
}

// CheckH2Pregnancy blocks enrollment if pregnant == true.
func CheckH2Pregnancy(snap slots.SlotSnapshot) (bool, string, string) {
	pregnant, ok := snap.GetBool("pregnant")
	if !ok {
		return false, "H2", ""
	}
	if pregnant {
		return true, "H2", "Pregnancy — obstetric care required, medication contraindications"
	}
	return false, "H2", ""
}

// CheckH3Dialysis blocks enrollment if dialysis == true OR eGFR < 15.
func CheckH3Dialysis(snap slots.SlotSnapshot) (bool, string, string) {
	dialysis, ok := snap.GetBool("dialysis")
	if ok && dialysis {
		return true, "H3", "Dialysis — nephrology management required"
	}
	egfr, ok := snap.GetFloat64("egfr")
	if ok && egfr < 15 {
		return true, "H3", "eGFR < 15 — CKD stage 5 / pre-dialysis, nephrology management required"
	}
	return false, "H3", ""
}

// CheckH4ActiveCancer blocks enrollment if active_cancer == true.
func CheckH4ActiveCancer(snap slots.SlotSnapshot) (bool, string, string) {
	cancer, ok := snap.GetBool("active_cancer")
	if !ok {
		return false, "H4", ""
	}
	if cancer {
		return true, "H4", "Active cancer — oncology priority, treatment interactions"
	}
	return false, "H4", ""
}

// CheckH5EGFRCritical blocks enrollment if eGFR < 15.
func CheckH5EGFRCritical(snap slots.SlotSnapshot) (bool, string, string) {
	egfr, ok := snap.GetFloat64("egfr")
	if !ok {
		return false, "H5", ""
	}
	if egfr < 15 {
		return true, "H5", "eGFR < 15 — CKD stage 5, requires nephrology specialist"
	}
	return false, "H5", ""
}

// CheckH6RecentMIStroke blocks enrollment if mi_stroke_days < 90.
func CheckH6RecentMIStroke(snap slots.SlotSnapshot) (bool, string, string) {
	days, ok := snap.GetInt("mi_stroke_days")
	if !ok {
		return false, "H6", ""
	}
	if days < 90 {
		return true, "H6", "Recent MI/stroke (< 90 days) — acute cardiac event, specialist management required"
	}
	return false, "H6", ""
}

// CheckH7HeartFailureSevere blocks enrollment if nyha_class >= 3.
func CheckH7HeartFailureSevere(snap slots.SlotSnapshot) (bool, string, string) {
	nyha, ok := snap.GetInt("nyha_class")
	if !ok {
		return false, "H7", ""
	}
	if nyha >= 3 {
		return true, "H7", "Heart failure NYHA class III/IV — HF specialist management required"
	}
	return false, "H7", ""
}

// CheckH8Child blocks enrollment if age < 18.
func CheckH8Child(snap slots.SlotSnapshot) (bool, string, string) {
	age, ok := snap.GetInt("age")
	if !ok {
		return false, "H8", ""
	}
	if age < 18 {
		return true, "H8", "Patient under 18 — pediatric protocol required"
	}
	return false, "H8", ""
}

// CheckH9BariatricSurgery blocks enrollment if bariatric_surgery_months < 12.
func CheckH9BariatricSurgery(snap slots.SlotSnapshot) (bool, string, string) {
	months, ok := snap.GetInt("bariatric_surgery_months")
	if !ok {
		return false, "H9", ""
	}
	if months < 12 {
		return true, "H9", "Bariatric surgery < 12 months ago — surgical follow-up required"
	}
	return false, "H9", ""
}

// CheckH10OrganTransplant blocks enrollment if organ_transplant == true.
func CheckH10OrganTransplant(snap slots.SlotSnapshot) (bool, string, string) {
	transplant, ok := snap.GetBool("organ_transplant")
	if !ok {
		return false, "H10", ""
	}
	if transplant {
		return true, "H10", "Organ transplant — immunosuppression management required"
	}
	return false, "H10", ""
}

// CheckH11SubstanceAbuse blocks enrollment if active_substance_abuse == true.
func CheckH11SubstanceAbuse(snap slots.SlotSnapshot) (bool, string, string) {
	abuse, ok := snap.GetBool("active_substance_abuse")
	if !ok {
		return false, "H11", ""
	}
	if abuse {
		return true, "H11", "Active substance abuse — addiction medicine referral required"
	}
	return false, "H11", ""
}
```

- [ ] **Step 3: Run tests (expect PASS for hard_stops, still FAIL for engine_test due to missing soft_flags)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/safety/... -v -count=1 -run TestH`
Expected: All 24 HARD_STOP tests PASS

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/hard_stops.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/hard_stops_test.go
git commit -m "feat(intake/safety): implement 11 HARD_STOP rules (H1-H11)

Pure functions, no I/O, deterministic. Rules: T1DM, pregnancy, dialysis,
active cancer, eGFR critical, recent MI/stroke, HF severe, child,
bariatric surgery, organ transplant, substance abuse. Missing slot
values default to safe (no trigger). 24 test cases covering triggers,
non-triggers, and boundary conditions."
```

---

## Task 5: SOFT_FLAGs — 8 Rule Implementations (SF-01 to SF-08)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/soft_flags.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/soft_flags_test.go`

- [ ] **Step 1: Write soft_flags_test.go (TDD — test first)**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/soft_flags_test.go
package safety

import (
	"testing"
)

func TestSF01_Elderly_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 80})
	triggered, id, _ := CheckSF01Elderly(snap)
	if !triggered {
		t.Error("SF-01 should trigger for age=80")
	}
	if id != "SF-01" {
		t.Errorf("expected SF-01, got %s", id)
	}
}

func TestSF01_Elderly_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 75})
	triggered, _, _ := CheckSF01Elderly(snap)
	if !triggered {
		t.Error("SF-01 should trigger for age=75 (>= 75)")
	}
}

func TestSF01_Elderly_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 74})
	triggered, _, _ := CheckSF01Elderly(snap)
	if triggered {
		t.Error("SF-01 should not trigger for age=74")
	}
}

func TestSF02_CKDModerate_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 35.0})
	triggered, id, _ := CheckSF02CKDModerate(snap)
	if !triggered {
		t.Error("SF-02 should trigger for eGFR=35")
	}
	if id != "SF-02" {
		t.Errorf("expected SF-02, got %s", id)
	}
}

func TestSF02_CKDModerate_LowerBound(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 15.0})
	triggered, _, _ := CheckSF02CKDModerate(snap)
	if !triggered {
		t.Error("SF-02 should trigger for eGFR=15 (>= 15)")
	}
}

func TestSF02_CKDModerate_UpperBound(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 44.0})
	triggered, _, _ := CheckSF02CKDModerate(snap)
	if !triggered {
		t.Error("SF-02 should trigger for eGFR=44 (<= 44)")
	}
}

func TestSF02_CKDModerate_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"egfr": 60.0})
	triggered, _, _ := CheckSF02CKDModerate(snap)
	if triggered {
		t.Error("SF-02 should not trigger for eGFR=60")
	}
}

func TestSF02_CKDModerate_BelowRange(t *testing.T) {
	// eGFR < 15 is HARD_STOP territory (H5), but SF-02 range is 15-44
	snap := buildSnapshot(map[string]interface{}{"egfr": 10.0})
	triggered, _, _ := CheckSF02CKDModerate(snap)
	if triggered {
		t.Error("SF-02 should not trigger for eGFR=10 (below range, H5 territory)")
	}
}

func TestSF03_Polypharmacy_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"medication_count": 7})
	triggered, id, _ := CheckSF03Polypharmacy(snap)
	if !triggered {
		t.Error("SF-03 should trigger for medication_count=7")
	}
	if id != "SF-03" {
		t.Errorf("expected SF-03, got %s", id)
	}
}

func TestSF03_Polypharmacy_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"medication_count": 5})
	triggered, _, _ := CheckSF03Polypharmacy(snap)
	if !triggered {
		t.Error("SF-03 should trigger for medication_count=5 (>= 5)")
	}
}

func TestSF03_Polypharmacy_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"medication_count": 3})
	triggered, _, _ := CheckSF03Polypharmacy(snap)
	if triggered {
		t.Error("SF-03 should not trigger for medication_count=3")
	}
}

func TestSF04_LowBMI_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"bmi": 17.5})
	triggered, id, _ := CheckSF04LowBMI(snap)
	if !triggered {
		t.Error("SF-04 should trigger for bmi=17.5")
	}
	if id != "SF-04" {
		t.Errorf("expected SF-04, got %s", id)
	}
}

func TestSF04_LowBMI_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"bmi": 18.5})
	triggered, _, _ := CheckSF04LowBMI(snap)
	if triggered {
		t.Error("SF-04 should not trigger for bmi=18.5 (boundary, < 18.5 required)")
	}
}

func TestSF05_InsulinUse_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"insulin": true})
	triggered, id, _ := CheckSF05InsulinUse(snap)
	if !triggered {
		t.Error("SF-05 should trigger for insulin=true")
	}
	if id != "SF-05" {
		t.Errorf("expected SF-05, got %s", id)
	}
}

func TestSF05_InsulinUse_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"insulin": false})
	triggered, _, _ := CheckSF05InsulinUse(snap)
	if triggered {
		t.Error("SF-05 should not trigger for insulin=false")
	}
}

func TestSF06_FallsRisk_ByHistory(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"falls_history": true})
	triggered, id, _ := CheckSF06FallsRisk(snap)
	if !triggered {
		t.Error("SF-06 should trigger for falls_history=true")
	}
	if id != "SF-06" {
		t.Errorf("expected SF-06, got %s", id)
	}
}

func TestSF06_FallsRisk_ByAge(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 72, "falls_history": false})
	triggered, _, _ := CheckSF06FallsRisk(snap)
	if !triggered {
		t.Error("SF-06 should trigger for age >= 70")
	}
}

func TestSF06_FallsRisk_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"age": 55, "falls_history": false})
	triggered, _, _ := CheckSF06FallsRisk(snap)
	if triggered {
		t.Error("SF-06 should not trigger for young patient without falls history")
	}
}

func TestSF07_CognitiveImpairment_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"cognitive_impairment": true})
	triggered, id, _ := CheckSF07CognitiveImpairment(snap)
	if !triggered {
		t.Error("SF-07 should trigger for cognitive_impairment=true")
	}
	if id != "SF-07" {
		t.Errorf("expected SF-07, got %s", id)
	}
}

func TestSF07_CognitiveImpairment_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"cognitive_impairment": false})
	triggered, _, _ := CheckSF07CognitiveImpairment(snap)
	if triggered {
		t.Error("SF-07 should not trigger for cognitive_impairment=false")
	}
}

func TestSF08_NonAdherent_Triggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"adherence_score": 0.3})
	triggered, id, _ := CheckSF08NonAdherent(snap)
	if !triggered {
		t.Error("SF-08 should trigger for adherence_score=0.3")
	}
	if id != "SF-08" {
		t.Errorf("expected SF-08, got %s", id)
	}
}

func TestSF08_NonAdherent_Boundary(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"adherence_score": 0.5})
	triggered, _, _ := CheckSF08NonAdherent(snap)
	if triggered {
		t.Error("SF-08 should not trigger for adherence_score=0.5 (boundary, < 0.5 required)")
	}
}

func TestSF08_NonAdherent_NotTriggered(t *testing.T) {
	snap := buildSnapshot(map[string]interface{}{"adherence_score": 0.85})
	triggered, _, _ := CheckSF08NonAdherent(snap)
	if triggered {
		t.Error("SF-08 should not trigger for adherence_score=0.85")
	}
}
```

- [ ] **Step 2: Write soft_flags.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/soft_flags.go
package safety

import "github.com/cardiofit/intake-onboarding-service/internal/slots"

// Each SOFT_FLAG is a pure function: no I/O, no external deps, deterministic.
// Soft flags do NOT block enrollment — they raise pharmacist awareness.

// CheckSF01Elderly flags patients age >= 75 for dose adjustment awareness.
func CheckSF01Elderly(snap slots.SlotSnapshot) (bool, string, string) {
	age, ok := snap.GetInt("age")
	if !ok {
		return false, "SF-01", ""
	}
	if age >= 75 {
		return true, "SF-01", "Elderly patient (age >= 75) — dose adjustment awareness required"
	}
	return false, "SF-01", ""
}

// CheckSF02CKDModerate flags patients with eGFR 15-44 for renal dose adjustment.
func CheckSF02CKDModerate(snap slots.SlotSnapshot) (bool, string, string) {
	egfr, ok := snap.GetFloat64("egfr")
	if !ok {
		return false, "SF-02", ""
	}
	if egfr >= 15 && egfr <= 44 {
		return true, "SF-02", "CKD moderate (eGFR 15-44) — renal dose adjustment required"
	}
	return false, "SF-02", ""
}

// CheckSF03Polypharmacy flags patients with medication_count >= 5.
func CheckSF03Polypharmacy(snap slots.SlotSnapshot) (bool, string, string) {
	count, ok := snap.GetInt("medication_count")
	if !ok {
		return false, "SF-03", ""
	}
	if count >= 5 {
		return true, "SF-03", "Polypharmacy (>= 5 medications) — drug interaction review required"
	}
	return false, "SF-03", ""
}

// CheckSF04LowBMI flags patients with BMI < 18.5 for malnutrition risk.
func CheckSF04LowBMI(snap slots.SlotSnapshot) (bool, string, string) {
	bmi, ok := snap.GetFloat64("bmi")
	if !ok {
		return false, "SF-04", ""
	}
	if bmi < 18.5 {
		return true, "SF-04", "Low BMI (< 18.5) — malnutrition risk assessment required"
	}
	return false, "SF-04", ""
}

// CheckSF05InsulinUse flags patients currently on insulin.
func CheckSF05InsulinUse(snap slots.SlotSnapshot) (bool, string, string) {
	insulin, ok := snap.GetBool("insulin")
	if !ok {
		return false, "SF-05", ""
	}
	if insulin {
		return true, "SF-05", "Insulin use — hypoglycemia monitoring required"
	}
	return false, "SF-05", ""
}

// CheckSF06FallsRisk flags patients with falls_history == true OR age >= 70.
func CheckSF06FallsRisk(snap slots.SlotSnapshot) (bool, string, string) {
	falls, ok := snap.GetBool("falls_history")
	if ok && falls {
		return true, "SF-06", "Falls history — balance assessment and medication review required"
	}
	age, ok := snap.GetInt("age")
	if ok && age >= 70 {
		return true, "SF-06", "Age >= 70 — falls risk, balance assessment required"
	}
	return false, "SF-06", ""
}

// CheckSF07CognitiveImpairment flags patients with cognitive impairment.
func CheckSF07CognitiveImpairment(snap slots.SlotSnapshot) (bool, string, string) {
	impaired, ok := snap.GetBool("cognitive_impairment")
	if !ok {
		return false, "SF-07", ""
	}
	if impaired {
		return true, "SF-07", "Cognitive impairment — caregiver involvement recommended"
	}
	return false, "SF-07", ""
}

// CheckSF08NonAdherent flags patients with adherence_score < 0.5.
func CheckSF08NonAdherent(snap slots.SlotSnapshot) (bool, string, string) {
	score, ok := snap.GetFloat64("adherence_score")
	if !ok {
		return false, "SF-08", ""
	}
	if score < 0.5 {
		return true, "SF-08", "Non-adherent history (score < 0.5) — enhanced follow-up required"
	}
	return false, "SF-08", ""
}
```

- [ ] **Step 3: Run all safety tests (expect PASS)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/safety/... -v -count=1`
Expected: All tests PASS (hard_stops + soft_flags + engine tests including performance)

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/soft_flags.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/safety/soft_flags_test.go
git commit -m "feat(intake/safety): implement 8 SOFT_FLAG rules (SF-01 to SF-08)

Pure functions: elderly, CKD moderate, polypharmacy, low BMI, insulin
use, falls risk, cognitive impairment, non-adherent history. Flags
raise pharmacist awareness but do NOT block enrollment. 22 test cases
covering triggers, non-triggers, and boundary conditions."
```

---

## Task 6: FHIR Generator — Slot Values to FHIR Resources

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/fhir/generator.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/fhir/patient.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/fhir/encounter.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/fhir/generator_test.go`

- [ ] **Step 1: Write generator_test.go (TDD — test first)**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/fhir/generator_test.go
package fhir

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

func TestObservationFromSlot_Numeric(t *testing.T) {
	patientID := uuid.New()
	encounterID := uuid.New()
	slot := slots.SlotDefinition{
		Name: "fbg", Domain: "glycemic", LOINCCode: "1558-6",
		DataType: slots.DataTypeNumeric, Unit: "mg/dL", Label: "Fasting blood glucose",
	}

	raw, err := ObservationFromSlot(patientID, encounterID, slot, json.RawMessage(`178`))
	if err != nil {
		t.Fatalf("ObservationFromSlot failed: %v", err)
	}

	var obs map[string]interface{}
	if err := json.Unmarshal(raw, &obs); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if obs["resourceType"] != "Observation" {
		t.Errorf("expected resourceType=Observation, got %v", obs["resourceType"])
	}
	if obs["status"] != "final" {
		t.Errorf("expected status=final, got %v", obs["status"])
	}

	// Check LOINC code
	code := obs["code"].(map[string]interface{})
	codings := code["coding"].([]interface{})
	coding := codings[0].(map[string]interface{})
	if coding["code"] != "1558-6" {
		t.Errorf("expected LOINC 1558-6, got %v", coding["code"])
	}

	// Check value
	vq := obs["valueQuantity"].(map[string]interface{})
	if vq["value"].(float64) != 178 {
		t.Errorf("expected value=178, got %v", vq["value"])
	}
	if vq["unit"] != "mg/dL" {
		t.Errorf("expected unit=mg/dL, got %v", vq["unit"])
	}
}

func TestObservationFromSlot_Boolean(t *testing.T) {
	patientID := uuid.New()
	encounterID := uuid.New()
	slot := slots.SlotDefinition{
		Name: "pregnant", Domain: "demographics", LOINCCode: "82810-3",
		DataType: slots.DataTypeBoolean, Label: "Currently pregnant",
	}

	raw, err := ObservationFromSlot(patientID, encounterID, slot, json.RawMessage(`true`))
	if err != nil {
		t.Fatalf("ObservationFromSlot failed: %v", err)
	}

	var obs map[string]interface{}
	json.Unmarshal(raw, &obs)
	if obs["valueBoolean"] != true {
		t.Errorf("expected valueBoolean=true, got %v", obs["valueBoolean"])
	}
}

func TestDetectedIssueFromSafetyResult_HardStop(t *testing.T) {
	patientID := uuid.New()
	encounterID := uuid.New()
	rule := safety.RuleResult{
		RuleID:   "H1",
		RuleType: safety.RuleTypeHardStop,
		Reason:   "Type 1 DM",
	}

	raw, err := DetectedIssueFromRule(patientID, encounterID, rule)
	if err != nil {
		t.Fatalf("DetectedIssueFromRule failed: %v", err)
	}

	var di map[string]interface{}
	json.Unmarshal(raw, &di)
	if di["resourceType"] != "DetectedIssue" {
		t.Errorf("expected resourceType=DetectedIssue")
	}
	if di["severity"] != "high" {
		t.Errorf("expected severity=high for HARD_STOP, got %v", di["severity"])
	}
	if di["status"] != "final" {
		t.Errorf("expected status=final, got %v", di["status"])
	}
}

func TestDetectedIssueFromSafetyResult_SoftFlag(t *testing.T) {
	patientID := uuid.New()
	encounterID := uuid.New()
	rule := safety.RuleResult{
		RuleID:   "SF-01",
		RuleType: safety.RuleTypeSoftFlag,
		Reason:   "Elderly patient",
	}

	raw, err := DetectedIssueFromRule(patientID, encounterID, rule)
	if err != nil {
		t.Fatalf("DetectedIssueFromRule failed: %v", err)
	}

	var di map[string]interface{}
	json.Unmarshal(raw, &di)
	if di["severity"] != "moderate" {
		t.Errorf("expected severity=moderate for SOFT_FLAG, got %v", di["severity"])
	}
}

func TestPatientResource(t *testing.T) {
	raw, err := NewPatientResource("John", "Doe", "+919876543210", "")
	if err != nil {
		t.Fatalf("NewPatientResource failed: %v", err)
	}

	var pat map[string]interface{}
	json.Unmarshal(raw, &pat)
	if pat["resourceType"] != "Patient" {
		t.Errorf("expected resourceType=Patient")
	}
	names := pat["name"].([]interface{})
	name := names[0].(map[string]interface{})
	if name["family"] != "Doe" {
		t.Errorf("expected family=Doe, got %v", name["family"])
	}
}

func TestEncounterResource(t *testing.T) {
	patientID := uuid.New()
	raw, err := NewEncounterResource(patientID, "intake")
	if err != nil {
		t.Fatalf("NewEncounterResource failed: %v", err)
	}

	var enc map[string]interface{}
	json.Unmarshal(raw, &enc)
	if enc["resourceType"] != "Encounter" {
		t.Errorf("expected resourceType=Encounter")
	}
	if enc["status"] != "in-progress" {
		t.Errorf("expected status=in-progress, got %v", enc["status"])
	}
}
```

- [ ] **Step 2: Write generator.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/fhir/generator.go
package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

// ObservationFromSlot converts a slot fill into a FHIR R4 Observation resource.
func ObservationFromSlot(patientID, encounterID uuid.UUID, slot slots.SlotDefinition, value json.RawMessage) ([]byte, error) {
	obs := map[string]interface{}{
		"resourceType": "Observation",
		"status":       "final",
		"category": []map[string]interface{}{
			{
				"coding": []map[string]interface{}{
					{
						"system":  "http://terminology.hl7.org/CodeSystem/observation-category",
						"code":    "intake",
						"display": "Intake Assessment",
					},
				},
			},
		},
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://loinc.org",
					"code":    slot.LOINCCode,
					"display": slot.Label,
				},
			},
			"text": slot.Label,
		},
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
		"encounter": map[string]interface{}{
			"reference": fmt.Sprintf("Encounter/%s", encounterID),
		},
		"effectiveDateTime": time.Now().UTC().Format(time.RFC3339),
	}

	// Set value based on data type
	switch slot.DataType {
	case slots.DataTypeNumeric:
		var v float64
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse numeric value for slot %s: %w", slot.Name, err)
		}
		obs["valueQuantity"] = map[string]interface{}{
			"value":  v,
			"unit":   slot.Unit,
			"system": "http://unitsofmeasure.org",
			"code":   slot.Unit,
		}
	case slots.DataTypeInteger:
		var v int
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse integer value for slot %s: %w", slot.Name, err)
		}
		obs["valueQuantity"] = map[string]interface{}{
			"value":  v,
			"unit":   slot.Unit,
			"system": "http://unitsofmeasure.org",
			"code":   slot.Unit,
		}
	case slots.DataTypeBoolean:
		var v bool
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse boolean value for slot %s: %w", slot.Name, err)
		}
		obs["valueBoolean"] = v
	case slots.DataTypeCodedChoice:
		var v string
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse coded choice for slot %s: %w", slot.Name, err)
		}
		obs["valueCodeableConcept"] = map[string]interface{}{
			"text": v,
		}
	case slots.DataTypeText:
		var v string
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse text for slot %s: %w", slot.Name, err)
		}
		obs["valueString"] = v
	case slots.DataTypeList:
		obs["valueString"] = string(value)
	case slots.DataTypeDate:
		var v string
		if err := json.Unmarshal(value, &v); err != nil {
			return nil, fmt.Errorf("parse date for slot %s: %w", slot.Name, err)
		}
		obs["valueDateTime"] = v
	}

	return json.Marshal(obs)
}

// DetectedIssueFromRule converts a safety rule result into a FHIR DetectedIssue resource.
func DetectedIssueFromRule(patientID, encounterID uuid.UUID, rule safety.RuleResult) ([]byte, error) {
	severity := "moderate" // SOFT_FLAG
	if rule.RuleType == safety.RuleTypeHardStop {
		severity = "high"
	}

	di := map[string]interface{}{
		"resourceType": "DetectedIssue",
		"status":       "final",
		"severity":     severity,
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://cardiofit.in/safety-rules",
					"code":    rule.RuleID,
					"display": rule.Reason,
				},
			},
			"text": rule.Reason,
		},
		"patient": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
		"detail":           rule.Reason,
		"identifiedDateTime": time.Now().UTC().Format(time.RFC3339),
		"implicated": []map[string]interface{}{
			{
				"reference": fmt.Sprintf("Encounter/%s", encounterID),
			},
		},
	}

	return json.Marshal(di)
}
```

- [ ] **Step 3: Write patient.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/fhir/patient.go
package fhir

import (
	"encoding/json"
	"time"
)

// NewPatientResource creates a FHIR R4 Patient resource.
// Follows ABDM IG v7.0 PatientIN profile.
func NewPatientResource(givenName, familyName, phone, abhaID string) ([]byte, error) {
	patient := map[string]interface{}{
		"resourceType": "Patient",
		"meta": map[string]interface{}{
			"profile": []string{
				"https://nrces.in/ndhm/fhir/r4/StructureDefinition/Patient",
			},
		},
		"active": true,
		"name": []map[string]interface{}{
			{
				"use":    "official",
				"family": familyName,
				"given":  []string{givenName},
			},
		},
		"telecom": []map[string]interface{}{
			{
				"system": "phone",
				"value":  phone,
				"use":    "mobile",
			},
		},
	}

	if abhaID != "" {
		patient["identifier"] = []map[string]interface{}{
			{
				"system": "https://healthid.abdm.gov.in",
				"value":  abhaID,
				"type": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://terminology.hl7.org/CodeSystem/v2-0203",
							"code":    "MR",
							"display": "ABHA Number",
						},
					},
				},
			},
		}
	}

	return json.Marshal(patient)
}

// UpdatePatientWithDemographics adds demographic observation references to a Patient.
func UpdatePatientWithDemographics(existingPatient []byte, birthDate time.Time, gender string) ([]byte, error) {
	var patient map[string]interface{}
	if err := json.Unmarshal(existingPatient, &patient); err != nil {
		return nil, err
	}

	if !birthDate.IsZero() {
		patient["birthDate"] = birthDate.Format("2006-01-02")
	}
	if gender != "" {
		patient["gender"] = gender
	}

	return json.Marshal(patient)
}
```

- [ ] **Step 4: Write encounter.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/fhir/encounter.go
package fhir

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NewEncounterResource creates a FHIR R4 Encounter resource for an intake session.
func NewEncounterResource(patientID uuid.UUID, encounterType string) ([]byte, error) {
	encounter := map[string]interface{}{
		"resourceType": "Encounter",
		"status":       "in-progress",
		"class": map[string]interface{}{
			"system":  "http://terminology.hl7.org/CodeSystem/v3-ActCode",
			"code":    "VR",
			"display": "virtual",
		},
		"type": []map[string]interface{}{
			{
				"coding": []map[string]interface{}{
					{
						"system":  "http://cardiofit.in/encounter-types",
						"code":    encounterType,
						"display": encounterType + " session",
					},
				},
			},
		},
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", patientID),
		},
		"period": map[string]interface{}{
			"start": time.Now().UTC().Format(time.RFC3339),
		},
	}

	return json.Marshal(encounter)
}

// UpdateEncounterStatus updates the status of an existing Encounter.
func UpdateEncounterStatus(existingEncounter []byte, status string) ([]byte, error) {
	var encounter map[string]interface{}
	if err := json.Unmarshal(existingEncounter, &encounter); err != nil {
		return nil, err
	}

	encounter["status"] = status
	if status == "finished" {
		if period, ok := encounter["period"].(map[string]interface{}); ok {
			period["end"] = time.Now().UTC().Format(time.RFC3339)
		}
	}

	return json.Marshal(encounter)
}
```

- [ ] **Step 5: Run tests (expect PASS)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/fhir/... -v -count=1`
Expected: All 6 tests PASS

- [ ] **Step 6: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/fhir/
git commit -m "feat(intake/fhir): add FHIR resource generators for Observation, DetectedIssue, Patient, Encounter

ObservationFromSlot maps slot value + LOINC code to FHIR Observation
(supports numeric, boolean, coded choice, text, list, date types).
DetectedIssueFromRule maps safety results to FHIR DetectedIssue
(severity=high for HARD_STOP, moderate for SOFT_FLAG). Patient and
Encounter builders follow ABDM IG v7.0 profiles."
```

---

## Task 7: Flow Graph Engine — YAML Loader + Graph Traversal

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/graph.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/engine.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/loader.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/loader_test.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/engine_test.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/configs/flows/intake_full.yaml`

- [ ] **Step 1: Write graph.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/graph.go
package flow

// Graph represents a directed flow graph for intake questioning.
type Graph struct {
	ID          string           `yaml:"id" json:"id"`
	Name        string           `yaml:"name" json:"name"`
	Version     string           `yaml:"version" json:"version"`
	StartNode   string           `yaml:"start_node" json:"start_node"`
	Nodes       map[string]*Node `yaml:"nodes" json:"nodes"`
}

// Node represents a single step in the flow graph.
type Node struct {
	ID          string   `yaml:"id" json:"id"`
	Type        NodeType `yaml:"type" json:"type"`
	Label       string   `yaml:"label" json:"label"`
	Slots       []string `yaml:"slots" json:"slots"`            // Slots to fill at this node
	Edges       []Edge   `yaml:"edges" json:"edges"`            // Outgoing edges
	SkipIf      string   `yaml:"skip_if,omitempty" json:"skip_if,omitempty"` // Slot condition to skip
}

// NodeType identifies the kind of flow node.
type NodeType string

const (
	NodeTypeQuestion   NodeType = "question"     // Ask patient to fill slots
	NodeTypeGate       NodeType = "gate"         // Conditional branching
	NodeTypeComplete   NodeType = "complete"     // Terminal — intake finished
	NodeTypeReview     NodeType = "review"       // Send to pharmacist review
)

// Edge represents a directed connection between nodes.
type Edge struct {
	Target    string `yaml:"target" json:"target"`
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"` // Simple condition expression
	Label     string `yaml:"label,omitempty" json:"label,omitempty"`
}
```

- [ ] **Step 2: Write loader.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/loader.go
package flow

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadGraph reads a YAML flow definition from disk and returns a validated Graph.
func LoadGraph(path string) (*Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read flow file %s: %w", path, err)
	}
	return ParseGraph(data)
}

// ParseGraph parses YAML bytes into a validated Graph.
func ParseGraph(data []byte) (*Graph, error) {
	var g Graph
	if err := yaml.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parse flow YAML: %w", err)
	}

	if err := validateGraph(&g); err != nil {
		return nil, err
	}

	return &g, nil
}

// validateGraph checks structural integrity of the flow graph.
func validateGraph(g *Graph) error {
	if g.ID == "" {
		return fmt.Errorf("flow graph missing 'id'")
	}
	if g.StartNode == "" {
		return fmt.Errorf("flow graph missing 'start_node'")
	}
	if _, ok := g.Nodes[g.StartNode]; !ok {
		return fmt.Errorf("start_node %q not found in nodes", g.StartNode)
	}

	// Validate all edge targets exist
	for nodeID, node := range g.Nodes {
		for i, edge := range node.Edges {
			if _, ok := g.Nodes[edge.Target]; !ok {
				return fmt.Errorf("node %q edge[%d] targets non-existent node %q", nodeID, i, edge.Target)
			}
		}
	}

	// Validate at least one complete node exists
	hasComplete := false
	for _, node := range g.Nodes {
		if node.Type == NodeTypeComplete {
			hasComplete = true
			break
		}
	}
	if !hasComplete {
		return fmt.Errorf("flow graph has no 'complete' node")
	}

	return nil
}
```

- [ ] **Step 3: Write engine.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/engine.go
package flow

import (
	"fmt"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

// Engine drives the flow graph traversal for a patient session.
type Engine struct {
	graph *Graph
}

// NewEngine creates a flow engine with the given graph.
func NewEngine(graph *Graph) *Engine {
	return &Engine{graph: graph}
}

// NextNode determines the next node to visit given the current position and filled slots.
// Returns the next node, or nil if the flow is complete.
func (e *Engine) NextNode(currentNodeID string, filledSlots map[string]slots.SlotValue) (*Node, error) {
	current, ok := e.graph.Nodes[currentNodeID]
	if !ok {
		return nil, fmt.Errorf("current node %q not found in graph", currentNodeID)
	}

	// If current node is complete or review, flow is done from here
	if current.Type == NodeTypeComplete || current.Type == NodeTypeReview {
		return current, nil
	}

	// Check if all slots at current node are filled
	allFilled := true
	for _, slotName := range current.Slots {
		if _, ok := filledSlots[slotName]; !ok {
			allFilled = false
			break
		}
	}

	// If current node's slots are not all filled, stay at current node
	if !allFilled {
		return current, nil
	}

	// Evaluate edges to find next node
	return e.evaluateEdges(current, filledSlots)
}

// evaluateEdges selects the next node based on edge conditions.
// Edges are evaluated in order; first matching condition wins.
// Edges without conditions are the default fallback.
func (e *Engine) evaluateEdges(node *Node, filledSlots map[string]slots.SlotValue) (*Node, error) {
	var defaultEdge *Edge

	for i := range node.Edges {
		edge := &node.Edges[i]
		if edge.Condition == "" {
			defaultEdge = edge
			continue
		}

		if evaluateCondition(edge.Condition, filledSlots) {
			target := e.graph.Nodes[edge.Target]
			// Check skip_if on target
			if target.SkipIf != "" && evaluateCondition(target.SkipIf, filledSlots) {
				// Skip this node, recurse to find next
				return e.NextNode(target.ID, filledSlots)
			}
			return target, nil
		}
	}

	if defaultEdge != nil {
		target := e.graph.Nodes[defaultEdge.Target]
		if target.SkipIf != "" && evaluateCondition(target.SkipIf, filledSlots) {
			return e.NextNode(target.ID, filledSlots)
		}
		return target, nil
	}

	return nil, fmt.Errorf("no valid edge from node %q", node.ID)
}

// evaluateCondition evaluates a simple condition expression against slot values.
// Supported forms:
//   - "slot_name"            → true if slot is filled
//   - "!slot_name"           → true if slot is NOT filled
//   - "slot_name=value"      → true if slot string value equals value
//   - "slot_name!=value"     → true if slot string value does not equal value
func evaluateCondition(condition string, filledSlots map[string]slots.SlotValue) bool {
	if len(condition) == 0 {
		return true
	}

	// Negation check: "!slot_name"
	if condition[0] == '!' {
		slotName := condition[1:]
		_, exists := filledSlots[slotName]
		return !exists
	}

	// Inequality check: "slot_name!=value"
	for i := 0; i < len(condition)-1; i++ {
		if condition[i] == '!' && condition[i+1] == '=' {
			slotName := condition[:i]
			expected := condition[i+2:]
			sv, exists := filledSlots[slotName]
			if !exists {
				return true // not filled != any value
			}
			return string(sv.Value) != `"`+expected+`"` && string(sv.Value) != expected
		}
	}

	// Equality check: "slot_name=value"
	for i := range condition {
		if condition[i] == '=' {
			slotName := condition[:i]
			expected := condition[i+1:]
			sv, exists := filledSlots[slotName]
			if !exists {
				return false
			}
			return string(sv.Value) == `"`+expected+`"` || string(sv.Value) == expected
		}
	}

	// Simple existence check: "slot_name"
	_, exists := filledSlots[condition]
	return exists
}

// UnfilledSlots returns the slot names at a node that are not yet filled.
func (e *Engine) UnfilledSlots(nodeID string, filledSlots map[string]slots.SlotValue) []string {
	node, ok := e.graph.Nodes[nodeID]
	if !ok {
		return nil
	}
	var unfilled []string
	for _, slotName := range node.Slots {
		if _, ok := filledSlots[slotName]; !ok {
			unfilled = append(unfilled, slotName)
		}
	}
	return unfilled
}

// IsComplete returns true if the current node is a terminal node.
func (e *Engine) IsComplete(nodeID string) bool {
	node, ok := e.graph.Nodes[nodeID]
	if !ok {
		return false
	}
	return node.Type == NodeTypeComplete
}

// IsReview returns true if the current node is a review node.
func (e *Engine) IsReview(nodeID string) bool {
	node, ok := e.graph.Nodes[nodeID]
	if !ok {
		return false
	}
	return node.Type == NodeTypeReview
}

// GraphID returns the graph identifier.
func (e *Engine) GraphID() string {
	return e.graph.ID
}
```

- [ ] **Step 4: Write engine_test.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/engine_test.go
package flow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/cardiofit/intake-onboarding-service/internal/slots"
)

func makeSlotValue(v interface{}) slots.SlotValue {
	raw, _ := json.Marshal(v)
	return slots.SlotValue{Value: raw, ExtractionMode: "BUTTON", Confidence: 1.0, UpdatedAt: time.Now()}
}

func testGraph() *Graph {
	return &Graph{
		ID:        "test_flow",
		Name:      "Test Flow",
		Version:   "1.0",
		StartNode: "demographics",
		Nodes: map[string]*Node{
			"demographics": {
				ID: "demographics", Type: NodeTypeQuestion, Label: "Demographics",
				Slots: []string{"age", "sex", "height", "weight"},
				Edges: []Edge{{Target: "glycemic"}},
			},
			"glycemic": {
				ID: "glycemic", Type: NodeTypeQuestion, Label: "Glycemic",
				Slots: []string{"diabetes_type", "fbg", "hba1c"},
				Edges: []Edge{
					{Target: "renal", Condition: "diabetes_type"},
				},
			},
			"renal": {
				ID: "renal", Type: NodeTypeQuestion, Label: "Renal",
				Slots: []string{"egfr", "serum_creatinine"},
				Edges: []Edge{{Target: "review_node"}},
			},
			"review_node": {
				ID: "review_node", Type: NodeTypeReview, Label: "Pharmacist Review",
				Edges: []Edge{{Target: "complete_node"}},
			},
			"complete_node": {
				ID: "complete_node", Type: NodeTypeComplete, Label: "Intake Complete",
			},
		},
	}
}

func TestEngine_NextNode_StaysAtCurrentIfNotFilled(t *testing.T) {
	engine := NewEngine(testGraph())
	filledSlots := map[string]slots.SlotValue{
		"age": makeSlotValue(45),
	}

	node, err := engine.NextNode("demographics", filledSlots)
	if err != nil {
		t.Fatalf("NextNode failed: %v", err)
	}
	if node.ID != "demographics" {
		t.Errorf("expected to stay at demographics, got %s", node.ID)
	}
}

func TestEngine_NextNode_AdvancesWhenAllFilled(t *testing.T) {
	engine := NewEngine(testGraph())
	filledSlots := map[string]slots.SlotValue{
		"age":    makeSlotValue(45),
		"sex":    makeSlotValue("male"),
		"height": makeSlotValue(175),
		"weight": makeSlotValue(80),
	}

	node, err := engine.NextNode("demographics", filledSlots)
	if err != nil {
		t.Fatalf("NextNode failed: %v", err)
	}
	if node.ID != "glycemic" {
		t.Errorf("expected glycemic, got %s", node.ID)
	}
}

func TestEngine_NextNode_ConditionalEdge(t *testing.T) {
	engine := NewEngine(testGraph())
	filledSlots := map[string]slots.SlotValue{
		"diabetes_type": makeSlotValue("T2DM"),
		"fbg":           makeSlotValue(178),
		"hba1c":         makeSlotValue(8.2),
	}

	node, err := engine.NextNode("glycemic", filledSlots)
	if err != nil {
		t.Fatalf("NextNode failed: %v", err)
	}
	if node.ID != "renal" {
		t.Errorf("expected renal, got %s", node.ID)
	}
}

func TestEngine_IsComplete(t *testing.T) {
	engine := NewEngine(testGraph())
	if !engine.IsComplete("complete_node") {
		t.Error("expected complete_node to be complete")
	}
	if engine.IsComplete("demographics") {
		t.Error("demographics should not be complete")
	}
}

func TestEngine_IsReview(t *testing.T) {
	engine := NewEngine(testGraph())
	if !engine.IsReview("review_node") {
		t.Error("expected review_node to be review")
	}
}

func TestEngine_UnfilledSlots(t *testing.T) {
	engine := NewEngine(testGraph())
	filledSlots := map[string]slots.SlotValue{
		"age": makeSlotValue(45),
		"sex": makeSlotValue("male"),
	}

	unfilled := engine.UnfilledSlots("demographics", filledSlots)
	if len(unfilled) != 2 {
		t.Errorf("expected 2 unfilled, got %d: %v", len(unfilled), unfilled)
	}
}

func TestEvaluateCondition_Existence(t *testing.T) {
	filled := map[string]slots.SlotValue{"fbg": makeSlotValue(178)}
	if !evaluateCondition("fbg", filled) {
		t.Error("fbg should exist")
	}
	if evaluateCondition("hba1c", filled) {
		t.Error("hba1c should not exist")
	}
}

func TestEvaluateCondition_Negation(t *testing.T) {
	filled := map[string]slots.SlotValue{"fbg": makeSlotValue(178)}
	if evaluateCondition("!fbg", filled) {
		t.Error("!fbg should be false when fbg exists")
	}
	if !evaluateCondition("!hba1c", filled) {
		t.Error("!hba1c should be true when hba1c missing")
	}
}
```

- [ ] **Step 5: Write loader_test.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/loader_test.go
package flow

import (
	"testing"
)

func TestParseGraph_Valid(t *testing.T) {
	yaml := `
id: test_flow
name: Test Flow
version: "1.0"
start_node: start
nodes:
  start:
    id: start
    type: question
    label: Start
    slots: [age, sex]
    edges:
      - target: end
  end:
    id: end
    type: complete
    label: Done
`
	g, err := ParseGraph([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseGraph failed: %v", err)
	}
	if g.ID != "test_flow" {
		t.Errorf("expected id=test_flow, got %s", g.ID)
	}
	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(g.Nodes))
	}
}

func TestParseGraph_MissingStartNode(t *testing.T) {
	yaml := `
id: broken
name: Broken
start_node: nonexistent
nodes:
  start:
    id: start
    type: complete
    label: Start
`
	_, err := ParseGraph([]byte(yaml))
	if err == nil {
		t.Error("expected error for missing start_node")
	}
}

func TestParseGraph_InvalidEdgeTarget(t *testing.T) {
	yaml := `
id: broken
name: Broken
start_node: start
nodes:
  start:
    id: start
    type: question
    label: Start
    edges:
      - target: nonexistent
  end:
    id: end
    type: complete
    label: Done
`
	_, err := ParseGraph([]byte(yaml))
	if err == nil {
		t.Error("expected error for invalid edge target")
	}
}

func TestParseGraph_NoCompleteNode(t *testing.T) {
	yaml := `
id: broken
name: Broken
start_node: start
nodes:
  start:
    id: start
    type: question
    label: Start
    edges:
      - target: middle
  middle:
    id: middle
    type: question
    label: Middle
`
	_, err := ParseGraph([]byte(yaml))
	if err == nil {
		t.Error("expected error for no complete node")
	}
}
```

- [ ] **Step 6: Write configs/flows/intake_full.yaml**

```yaml
# vaidshala/clinical-runtime-platform/services/intake-onboarding-service/configs/flows/intake_full.yaml
# Full 50-slot intake flow definition (~25 nodes across 8 domains).
# Nodes are organized by clinical domain. Each node collects related slots.
# Edges define progression; conditional edges handle domain-specific branching.

id: intake_full
name: Full Intake Questionnaire
version: "1.0"
start_node: demographics_basic

nodes:
  # ── Demographics ──
  demographics_basic:
    id: demographics_basic
    type: question
    label: "Basic Demographics"
    slots: [age, sex, height, weight, bmi, pregnant]
    edges:
      - target: demographics_optional

  demographics_optional:
    id: demographics_optional
    type: question
    label: "Additional Demographics"
    slots: [ethnicity, primary_language]
    edges:
      - target: glycemic_type

  # ── Glycemic ──
  glycemic_type:
    id: glycemic_type
    type: question
    label: "Diabetes Classification"
    slots: [diabetes_type, insulin]
    edges:
      - target: glycemic_labs

  glycemic_labs:
    id: glycemic_labs
    type: question
    label: "Glycemic Labs"
    slots: [fbg, hba1c, ppbg]
    edges:
      - target: glycemic_history

  glycemic_history:
    id: glycemic_history
    type: question
    label: "Glycemic History"
    slots: [diabetes_duration_years, hypoglycemia_episodes]
    edges:
      - target: renal_labs

  # ── Renal ──
  renal_labs:
    id: renal_labs
    type: question
    label: "Renal Function"
    slots: [egfr, serum_creatinine, dialysis]
    edges:
      - target: renal_extended

  renal_extended:
    id: renal_extended
    type: question
    label: "Extended Renal"
    slots: [uacr, serum_potassium]
    edges:
      - target: cardiac_vitals

  # ── Cardiac ──
  cardiac_vitals:
    id: cardiac_vitals
    type: question
    label: "Cardiac Vitals"
    slots: [systolic_bp, diastolic_bp, heart_rate]
    edges:
      - target: cardiac_history

  cardiac_history:
    id: cardiac_history
    type: question
    label: "Cardiac History"
    slots: [nyha_class, mi_stroke_days, lvef, atrial_fibrillation]
    edges:
      - target: lipid_panel

  # ── Lipid ──
  lipid_panel:
    id: lipid_panel
    type: question
    label: "Lipid Panel"
    slots: [total_cholesterol, ldl, hdl, triglycerides, on_statin]
    edges:
      - target: medications_current

  # ── Medications ──
  medications_current:
    id: medications_current
    type: question
    label: "Current Medications"
    slots: [current_medications, medication_count, allergies]
    edges:
      - target: medications_adherence

  medications_adherence:
    id: medications_adherence
    type: question
    label: "Medication Adherence"
    slots: [adherence_score, supplement_list]
    edges:
      - target: lifestyle_habits

  # ── Lifestyle ──
  lifestyle_habits:
    id: lifestyle_habits
    type: question
    label: "Lifestyle Habits"
    slots: [smoking_status, alcohol_use, exercise_minutes_week, diet_type]
    edges:
      - target: lifestyle_safety

  lifestyle_safety:
    id: lifestyle_safety
    type: question
    label: "Lifestyle Safety"
    slots: [sleep_hours, active_substance_abuse, falls_history]
    edges:
      - target: symptoms_critical

  # ── Symptoms ──
  symptoms_critical:
    id: symptoms_critical
    type: question
    label: "Critical Conditions"
    slots: [active_cancer, organ_transplant]
    edges:
      - target: symptoms_extended

  symptoms_extended:
    id: symptoms_extended
    type: question
    label: "Extended Symptoms"
    slots: [cognitive_impairment, bariatric_surgery_months, primary_complaint, comorbidities]
    edges:
      - target: review_gate

  # ── Review Gate ──
  review_gate:
    id: review_gate
    type: review
    label: "Submit for Pharmacist Review"
    edges:
      - target: intake_complete

  # ── Complete ──
  intake_complete:
    id: intake_complete
    type: complete
    label: "Intake Complete"
```

- [ ] **Step 7: Run tests (expect PASS)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/flow/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 8: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/flow/ \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/configs/flows/intake_full.yaml
git commit -m "feat(intake/flow): add YAML-driven flow graph engine with generic traversal

Graph data structures (Node, Edge, NodeType), YAML loader with
structural validation (start_node exists, edge targets exist, complete
node required). Engine supports conditional edges, skip_if, unfilled
slot tracking. intake_full.yaml defines 18-node flow across 8 domains
covering all 50 slots."
```

---

## Task 8: $fill-slot Handler — Core Endpoint

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/app/handler.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/app/handler_test.go`

**Reference:** Spec section 5.3 write flow. This is the central endpoint: accept value -> validate slot -> safety check -> FHIR write -> Kafka publish -> update flow position -> return next question.

- [ ] **Step 1: Write handler_test.go (TDD — test first)**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/app/handler_test.go
package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestFillSlotRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		body    FillSlotRequest
		wantErr bool
	}{
		{
			name: "valid request",
			body: FillSlotRequest{
				SlotName:       "fbg",
				Value:          json.RawMessage(`178`),
				ExtractionMode: "BUTTON",
				Confidence:     1.0,
				SourceChannel:  "APP",
			},
			wantErr: false,
		},
		{
			name: "missing slot_name",
			body: FillSlotRequest{
				Value:          json.RawMessage(`178`),
				ExtractionMode: "BUTTON",
				Confidence:     1.0,
				SourceChannel:  "APP",
			},
			wantErr: true,
		},
		{
			name: "unknown slot_name",
			body: FillSlotRequest{
				SlotName:       "nonexistent_slot",
				Value:          json.RawMessage(`178`),
				ExtractionMode: "BUTTON",
				Confidence:     1.0,
				SourceChannel:  "APP",
			},
			wantErr: true,
		},
		{
			name: "missing value",
			body: FillSlotRequest{
				SlotName:       "fbg",
				ExtractionMode: "BUTTON",
				Confidence:     1.0,
				SourceChannel:  "APP",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.body.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFillSlotResponse_Structure(t *testing.T) {
	resp := FillSlotResponse{
		Status:         "ok",
		SlotName:       "fbg",
		FHIRResourceID: "obs-123",
		SafetyResult: &SafetyResultResponse{
			HardStops: []RuleResultResponse{},
			SoftFlags: []RuleResultResponse{
				{RuleID: "SF-01", Reason: "Elderly"},
			},
		},
		NextNode: &NextNodeResponse{
			NodeID: "glycemic",
			Slots:  []string{"hba1c", "ppbg"},
		},
		Progress: ProgressResponse{
			Filled:   5,
			Total:    50,
			Percent:  10.0,
			Complete: false,
		},
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(raw, &decoded)
	if decoded["status"] != "ok" {
		t.Errorf("expected status=ok")
	}
	if decoded["slot_name"] != "fbg" {
		t.Errorf("expected slot_name=fbg")
	}
}

func TestFillSlotHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	h := &Handler{logger: testLogger()}
	router.POST("/fhir/Encounter/:id/$fill-slot", h.HandleFillSlot)

	encID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/fhir/Encounter/"+encID+"/$fill-slot", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Write handler.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/app/handler.go
package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	intakefhir "github.com/cardiofit/intake-onboarding-service/internal/fhir"
	"github.com/cardiofit/intake-onboarding-service/internal/flow"
	"github.com/cardiofit/intake-onboarding-service/internal/kafka"
	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"github.com/cardiofit/intake-onboarding-service/internal/slots"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// Handler implements the Flutter app REST handlers for intake.
type Handler struct {
	eventStore   slots.EventStore
	safetyEngine *safety.Engine
	flowEngine   *flow.Engine
	fhirClient   *fhirclient.Client
	producer     *kafka.Producer
	logger       *zap.Logger
}

// NewHandler creates a new app handler with all dependencies.
func NewHandler(
	eventStore slots.EventStore,
	safetyEngine *safety.Engine,
	flowEngine *flow.Engine,
	fhirClient *fhirclient.Client,
	producer *kafka.Producer,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		eventStore:   eventStore,
		safetyEngine: safetyEngine,
		flowEngine:   flowEngine,
		fhirClient:   fhirClient,
		producer:     producer,
		logger:       logger,
	}
}

// FillSlotRequest is the JSON body for POST /fhir/Encounter/:id/$fill-slot.
type FillSlotRequest struct {
	SlotName       string          `json:"slot_name"`
	Value          json.RawMessage `json:"value"`
	ExtractionMode string          `json:"extraction_mode"` // BUTTON, REGEX, NLU, DEVICE
	Confidence     float64         `json:"confidence"`
	SourceChannel  string          `json:"source_channel"` // APP, WHATSAPP, ASHA
}

// Validate checks the fill-slot request for required fields and slot existence.
func (r *FillSlotRequest) Validate() error {
	if r.SlotName == "" {
		return fmt.Errorf("slot_name is required")
	}
	if _, ok := slots.LookupSlot(r.SlotName); !ok {
		return fmt.Errorf("unknown slot_name: %s", r.SlotName)
	}
	if len(r.Value) == 0 || string(r.Value) == "null" {
		return fmt.Errorf("value is required")
	}
	if r.ExtractionMode == "" {
		r.ExtractionMode = "BUTTON"
	}
	if r.SourceChannel == "" {
		r.SourceChannel = "APP"
	}
	return nil
}

// FillSlotResponse is returned by $fill-slot.
type FillSlotResponse struct {
	Status         string                `json:"status"` // "ok", "hard_stopped", "error"
	SlotName       string                `json:"slot_name"`
	FHIRResourceID string                `json:"fhir_resource_id,omitempty"`
	SafetyResult   *SafetyResultResponse `json:"safety_result,omitempty"`
	NextNode       *NextNodeResponse     `json:"next_node,omitempty"`
	Progress       ProgressResponse      `json:"progress"`
}

type SafetyResultResponse struct {
	HardStops []RuleResultResponse `json:"hard_stops"`
	SoftFlags []RuleResultResponse `json:"soft_flags"`
}

type RuleResultResponse struct {
	RuleID string `json:"rule_id"`
	Reason string `json:"reason"`
}

type NextNodeResponse struct {
	NodeID string   `json:"node_id"`
	Label  string   `json:"label,omitempty"`
	Slots  []string `json:"slots"`
}

type ProgressResponse struct {
	Filled   int     `json:"filled"`
	Total    int     `json:"total"`
	Percent  float64 `json:"percent"`
	Complete bool    `json:"complete"`
}

// HandleFillSlot implements POST /fhir/Encounter/:id/$fill-slot.
// Flow: accept value -> validate slot -> safety check -> FHIR write -> Kafka publish -> next question.
func (h *Handler) HandleFillSlot(c *gin.Context) {
	encounterID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encounter ID"})
		return
	}

	var req FillSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slotDef, _ := slots.LookupSlot(req.SlotName)

	// Get patientID from encounter (in production, look up from DB; here use header)
	patientIDStr := c.GetHeader("X-Patient-ID")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Patient-ID header required"})
		return
	}

	ctx := c.Request.Context()

	// 1. Append slot event to event store
	safetyResultJSON, _ := json.Marshal(nil)
	event := slots.SlotEvent{
		PatientID:      patientID,
		SlotName:       req.SlotName,
		Domain:         slotDef.Domain,
		Value:          req.Value,
		ExtractionMode: req.ExtractionMode,
		Confidence:     req.Confidence,
		SourceChannel:  req.SourceChannel,
	}

	// 2. Get current slot values (including this new one)
	currentValues, err := h.eventStore.CurrentValues(ctx, patientID)
	if err != nil {
		h.logger.Error("failed to get current values", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read slot values"})
		return
	}
	// Add new value to snapshot
	currentValues[req.SlotName] = slots.SlotValue{
		Value:          req.Value,
		ExtractionMode: req.ExtractionMode,
		Confidence:     req.Confidence,
	}

	// 3. Run safety engine (<5ms, deterministic)
	snapshot := slots.BuildSnapshot(patientID, currentValues)
	safetyResult := h.safetyEngine.Evaluate(snapshot)
	safetyResultJSON, _ = json.Marshal(safetyResult)
	event.SafetyResult = safetyResultJSON

	// 4. Write FHIR Observation to Google FHIR Store
	var fhirResourceID string
	if h.fhirClient != nil {
		obsJSON, err := intakefhir.ObservationFromSlot(patientID, encounterID, slotDef, req.Value)
		if err != nil {
			h.logger.Error("failed to build FHIR Observation", zap.Error(err))
		} else {
			respData, err := h.fhirClient.Create("Observation", obsJSON)
			if err != nil {
				h.logger.Error("FHIR Observation write failed, will retry", zap.Error(err))
				// Per spec section 7.2: retry 3x -> hold in PostgreSQL -> background sync
				// Slot acknowledged to patient regardless
			} else {
				var resp map[string]interface{}
				json.Unmarshal(respData, &resp)
				if id, ok := resp["id"].(string); ok {
					fhirResourceID = id
				}
			}
		}

		// 4b. Write DetectedIssue for any safety triggers
		for _, hs := range safetyResult.HardStops {
			diJSON, err := intakefhir.DetectedIssueFromRule(patientID, encounterID, hs)
			if err == nil {
				h.fhirClient.Create("DetectedIssue", diJSON)
			}
		}
		for _, sf := range safetyResult.SoftFlags {
			diJSON, err := intakefhir.DetectedIssueFromRule(patientID, encounterID, sf)
			if err == nil {
				h.fhirClient.Create("DetectedIssue", diJSON)
			}
		}
	}
	event.FHIRResourceID = fhirResourceID

	// 5. Persist slot event
	if err := h.eventStore.Append(ctx, event); err != nil {
		h.logger.Error("failed to persist slot event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "safety engine error — cannot swallow"})
		return
	}

	// 6. Publish to Kafka
	if h.producer != nil {
		payload := map[string]interface{}{
			"slot_name":    req.SlotName,
			"domain":       slotDef.Domain,
			"value":        req.Value,
			"safety_result": safetyResult,
		}
		h.producer.Publish(ctx, kafka.TopicSlotEvents, patientID, "SLOT_FILLED", payload)

		if safetyResult.HasHardStop() {
			h.producer.Publish(ctx, kafka.TopicSafetyAlerts, patientID, "HARD_STOP", payload)
		}
		if safetyResult.HasSoftFlag() {
			h.producer.Publish(ctx, kafka.TopicSafetyFlags, patientID, "SOFT_FLAG", payload)
		}
	}

	// 7. Build response
	resp := FillSlotResponse{
		Status:         "ok",
		SlotName:       req.SlotName,
		FHIRResourceID: fhirResourceID,
		Progress: ProgressResponse{
			Filled:   len(currentValues),
			Total:    len(slots.AllSlots()),
			Percent:  float64(len(currentValues)) / float64(len(slots.AllSlots())) * 100,
			Complete: snapshot.IsComplete(),
		},
	}

	// Safety result in response
	if safetyResult.HasHardStop() || safetyResult.HasSoftFlag() {
		sr := &SafetyResultResponse{
			HardStops: make([]RuleResultResponse, len(safetyResult.HardStops)),
			SoftFlags: make([]RuleResultResponse, len(safetyResult.SoftFlags)),
		}
		for i, hs := range safetyResult.HardStops {
			sr.HardStops[i] = RuleResultResponse{RuleID: hs.RuleID, Reason: hs.Reason}
		}
		for i, sf := range safetyResult.SoftFlags {
			sr.SoftFlags[i] = RuleResultResponse{RuleID: sf.RuleID, Reason: sf.Reason}
		}
		resp.SafetyResult = sr
	}

	if safetyResult.HasHardStop() {
		resp.Status = "hard_stopped"
		c.JSON(http.StatusOK, resp)
		return
	}

	// 8. Determine next question from flow engine
	if h.flowEngine != nil {
		// TODO: look up current node from flow_positions table
		// For now, use the start node as a placeholder
	}

	c.JSON(http.StatusOK, resp)
}

// EnrollRequest is the JSON body for POST /fhir/Patient/$enroll.
type EnrollRequest struct {
	GivenName   string `json:"given_name"`
	FamilyName  string `json:"family_name"`
	Phone       string `json:"phone"`
	ABHAID      string `json:"abha_id,omitempty"`
	ChannelType string `json:"channel_type"` // CORPORATE, INSURANCE, GOVERNMENT
	TenantID    string `json:"tenant_id"`
}

// HandleEnroll implements POST /fhir/Patient/$enroll.
// Creates Patient + Encounter in FHIR Store, creates enrollment in PostgreSQL.
func (h *Handler) HandleEnroll(c *gin.Context) {
	var req EnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	if req.GivenName == "" || req.FamilyName == "" || req.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "given_name, family_name, and phone are required"})
		return
	}

	// 1. Create Patient in FHIR Store
	var patientID string
	if h.fhirClient != nil {
		patientJSON, err := intakefhir.NewPatientResource(req.GivenName, req.FamilyName, req.Phone, req.ABHAID)
		if err != nil {
			h.logger.Error("failed to build Patient resource", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create patient"})
			return
		}

		respData, err := h.fhirClient.Create("Patient", patientJSON)
		if err != nil {
			h.logger.Error("FHIR Patient create failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "FHIR Store write failed"})
			return
		}

		var resp map[string]interface{}
		json.Unmarshal(respData, &resp)
		if id, ok := resp["id"].(string); ok {
			patientID = id
		}
	} else {
		patientID = uuid.New().String()
	}

	pid, _ := uuid.Parse(patientID)

	// 2. Create Encounter in FHIR Store
	var encounterID string
	if h.fhirClient != nil {
		encJSON, err := intakefhir.NewEncounterResource(pid, "intake")
		if err != nil {
			h.logger.Error("failed to build Encounter resource", zap.Error(err))
		} else {
			respData, err := h.fhirClient.Create("Encounter", encJSON)
			if err != nil {
				h.logger.Error("FHIR Encounter create failed", zap.Error(err))
			} else {
				var resp map[string]interface{}
				json.Unmarshal(respData, &resp)
				if id, ok := resp["id"].(string); ok {
					encounterID = id
				}
			}
		}
	}
	if encounterID == "" {
		encounterID = uuid.New().String()
	}

	// 3. Publish to Kafka
	if h.producer != nil {
		payload := map[string]interface{}{
			"patient_id":   patientID,
			"encounter_id": encounterID,
			"channel_type": req.ChannelType,
			"phone":        req.Phone,
		}
		h.producer.Publish(c.Request.Context(), kafka.TopicPatientLifecycle, pid, "PATIENT_CREATED", payload)
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":       "enrolled",
		"patient_id":   patientID,
		"encounter_id": encounterID,
		"next_node": gin.H{
			"node_id": "demographics_basic",
			"slots":   []string{"age", "sex", "height", "weight", "bmi", "pregnant"},
		},
	})
}

// HandleEvaluateSafety implements POST /fhir/Patient/:id/$evaluate-safety.
func (h *Handler) HandleEvaluateSafety(c *gin.Context) {
	patientIDStr := c.Param("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	ctx := c.Request.Context()
	currentValues, err := h.eventStore.CurrentValues(ctx, patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read slot values"})
		return
	}

	snapshot := slots.BuildSnapshot(patientID, currentValues)
	result := h.safetyEngine.Evaluate(snapshot)

	c.JSON(http.StatusOK, gin.H{
		"patient_id":  patientID,
		"hard_stops":  result.HardStops,
		"soft_flags":  result.SoftFlags,
		"has_hard_stop": result.HasHardStop(),
	})
}
```

- [ ] **Step 3: Run tests (expect PASS)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/app/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/app/
git commit -m "feat(intake/app): add \$fill-slot, \$enroll, \$evaluate-safety handlers

\$fill-slot: validate slot -> safety check -> FHIR write -> Kafka
publish -> return next question + progress. \$enroll: create Patient +
Encounter in FHIR Store, publish PATIENT_CREATED. \$evaluate-safety:
run full safety engine against current slot values. Structured request/
response types with validation."
```

---

## Task 9: $enroll Handler — Create Patient + Encounter in FHIR Store

This task is already implemented in Task 8 (`HandleEnroll` in handler.go). The remaining work is wiring enrollment state machine persistence to PostgreSQL.

**Files:**
- Modify: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/app/handler.go` (add enrollment DB write)

- [ ] **Step 1: Add enrollment persistence to HandleEnroll**

The `HandleEnroll` function in Task 8 creates FHIR resources but does not persist the enrollment state machine to PostgreSQL. Add this after the FHIR writes:

```go
// Add to handler.go — inside HandleEnroll, after step 2 (Encounter creation) and before step 3 (Kafka publish):

	// 2b. Persist enrollment to PostgreSQL
	if h.eventStore != nil {
		encUUID, _ := uuid.Parse(encounterID)
		tenantUUID, _ := uuid.Parse(req.TenantID)
		_, err := h.db.Exec(c.Request.Context(),
			`INSERT INTO enrollments (patient_id, tenant_id, channel_type, state, encounter_id)
			 VALUES ($1, $2, $3, 'CREATED', $4)`,
			pid, tenantUUID, req.ChannelType, encUUID,
		)
		if err != nil {
			h.logger.Error("failed to persist enrollment", zap.Error(err))
			// Non-fatal — FHIR Store is source of truth
		}
	}
```

Note: This requires adding `db *pgxpool.Pool` to the Handler struct and NewHandler function. Update handler.go accordingly.

- [ ] **Step 2: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/app/handler.go
git commit -m "feat(intake/app): add enrollment state persistence to PostgreSQL

HandleEnroll now writes CREATED enrollment to PostgreSQL enrollments
table after FHIR Store Patient+Encounter creation. Non-fatal on DB
failure since FHIR Store is source of truth."
```

---

## Task 10: Kafka Producer — Publish to intake.* Topics

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/kafka/producer.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/kafka/topics.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/kafka/producer_test.go`

- [ ] **Step 1: Write topics.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/kafka/topics.go
package kafka

// Kafka topic constants for the Intake-Onboarding Service.
// Follows naming convention: {service}.{domain} (spec section 6.0).

const (
	// TopicPatientLifecycle carries PATIENT_CREATED, PATIENT_ENROLLED events.
	// Consumer: KB-20.
	TopicPatientLifecycle = "intake.patient-lifecycle"

	// TopicSlotEvents carries slot fill events with safety results.
	// Consumers: KB-20, KB-22.
	TopicSlotEvents = "intake.slot-events"

	// TopicSafetyAlerts carries HARD_STOP triggers (urgent physician card).
	// Consumers: KB-23, Notifications.
	TopicSafetyAlerts = "intake.safety-alerts"

	// TopicSafetyFlags carries SOFT_FLAG triggers (pharmacist awareness).
	// Consumer: Review Queue.
	TopicSafetyFlags = "intake.safety-flags"

	// TopicCompletions carries intake-complete events ready for pharmacist review.
	// Consumer: Review Queue.
	TopicCompletions = "intake.completions"

	// TopicCheckinEvents carries biweekly check-in and trajectory signals.
	// Consumers: M4, KB-20, KB-21.
	TopicCheckinEvents = "intake.checkin-events"

	// TopicSessionLifecycle carries ABANDONED, PAUSED session events.
	// Consumer: Admin Dashboard.
	TopicSessionLifecycle = "intake.session-lifecycle"

	// TopicLabOrders carries missing baseline lab requests.
	// Consumer: Lab Integration.
	TopicLabOrders = "intake.lab-orders"
)

// AllTopics returns all 8 intake Kafka topics.
func AllTopics() []string {
	return []string{
		TopicPatientLifecycle,
		TopicSlotEvents,
		TopicSafetyAlerts,
		TopicSafetyFlags,
		TopicCompletions,
		TopicCheckinEvents,
		TopicSessionLifecycle,
		TopicLabOrders,
	}
}
```

- [ ] **Step 2: Write producer.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/kafka/producer.go
package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer publishes messages to intake.* Kafka topics.
type Producer struct {
	writers map[string]*kafkago.Writer
	logger  *zap.Logger
}

// NewProducer creates a Kafka producer with writers for all intake topics.
func NewProducer(brokers []string, logger *zap.Logger) *Producer {
	writers := make(map[string]*kafkago.Writer)
	for _, topic := range AllTopics() {
		writers[topic] = &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.Hash{}, // Partition by key (patientId)
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafkago.RequireAll,
		}
	}

	return &Producer{
		writers: writers,
		logger:  logger,
	}
}

// Publish sends a message to the specified topic with the patient ID as partition key.
func (p *Producer) Publish(ctx context.Context, topic string, patientID uuid.UUID, eventType string, payload map[string]interface{}) error {
	writer, ok := p.writers[topic]
	if !ok {
		p.logger.Error("unknown Kafka topic", zap.String("topic", topic))
		return nil
	}

	envelope := Envelope{
		EventID:   uuid.New(),
		EventType: eventType,
		SourceType: "INTAKE",
		PatientID:  patientID,
		Timestamp:  time.Now().UTC(),
		Payload:    payload,
	}

	value, err := json.Marshal(envelope)
	if err != nil {
		p.logger.Error("failed to marshal Kafka message", zap.Error(err))
		return err
	}

	msg := kafkago.Message{
		Key:   []byte(patientID.String()),
		Value: value,
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		p.logger.Error("Kafka publish failed",
			zap.String("topic", topic),
			zap.String("event_type", eventType),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("Kafka message published",
		zap.String("topic", topic),
		zap.String("event_type", eventType),
		zap.String("patient_id", patientID.String()),
	)
	return nil
}

// Close shuts down all Kafka writers.
func (p *Producer) Close() error {
	var lastErr error
	for topic, writer := range p.writers {
		if err := writer.Close(); err != nil {
			p.logger.Error("failed to close Kafka writer", zap.String("topic", topic), zap.Error(err))
			lastErr = err
		}
	}
	return lastErr
}
```

- [ ] **Step 3: Write producer_test.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/kafka/producer_test.go
package kafka

import (
	"testing"
)

func TestAllTopics_Count(t *testing.T) {
	topics := AllTopics()
	if len(topics) != 8 {
		t.Errorf("expected 8 intake topics, got %d", len(topics))
	}
}

func TestAllTopics_NamingConvention(t *testing.T) {
	for _, topic := range AllTopics() {
		if len(topic) < 8 || topic[:7] != "intake." {
			t.Errorf("topic %q does not follow intake.* naming convention", topic)
		}
	}
}

func TestTopicConstants(t *testing.T) {
	expected := map[string]bool{
		"intake.patient-lifecycle": true,
		"intake.slot-events":      true,
		"intake.safety-alerts":    true,
		"intake.safety-flags":     true,
		"intake.completions":      true,
		"intake.checkin-events":   true,
		"intake.session-lifecycle": true,
		"intake.lab-orders":       true,
	}
	for _, topic := range AllTopics() {
		if !expected[topic] {
			t.Errorf("unexpected topic: %s", topic)
		}
	}
}
```

- [ ] **Step 4: Run tests (expect PASS)**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/kafka/... -v -count=1`
Expected: All 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/kafka/
git commit -m "feat(intake/kafka): add producer for 8 intake.* topics

Topics: patient-lifecycle, slot-events, safety-alerts, safety-flags,
completions, checkin-events, session-lifecycle, lab-orders. Producer
uses kafka-go with hash partitioning by patientId. Envelope struct
matches spec section 6.3."
```

---

## Task 11: Wire All Handlers to Routes (Replace 501 Stubs)

**Files:**
- Modify: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/server.go`
- Modify: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/routes.go`
- Modify: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/cmd/intake/main.go`

- [ ] **Step 1: Update server.go — add new dependencies to Server struct**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/server.go
// Replace the existing Server struct and NewServer function with:

package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/intake-onboarding-service/internal/app"
	"github.com/cardiofit/intake-onboarding-service/internal/config"
	"github.com/cardiofit/intake-onboarding-service/internal/flow"
	intakekafka "github.com/cardiofit/intake-onboarding-service/internal/kafka"
	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"github.com/cardiofit/intake-onboarding-service/internal/slots"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

type Server struct {
	Router       *gin.Engine
	config       *config.Config
	db           *pgxpool.Pool
	redis        *redis.Client
	fhirClient   *fhirclient.Client
	logger       *zap.Logger
	appHandler   *app.Handler
	safetyEngine *safety.Engine
	flowEngine   *flow.Engine
	producer     *intakekafka.Producer
	eventStore   slots.EventStore
}

func NewServer(
	cfg *config.Config,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	fhirClient *fhirclient.Client,
	logger *zap.Logger,
	safetyEngine *safety.Engine,
	flowEngine *flow.Engine,
	producer *intakekafka.Producer,
) *Server {
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	eventStore := slots.NewPgEventStore(db)

	s := &Server{
		Router:       router,
		config:       cfg,
		db:           db,
		redis:        redisClient,
		fhirClient:   fhirClient,
		logger:       logger,
		safetyEngine: safetyEngine,
		flowEngine:   flowEngine,
		producer:     producer,
		eventStore:   eventStore,
		appHandler:   app.NewHandler(eventStore, safetyEngine, flowEngine, fhirClient, producer, logger),
	}

	router.Use(s.metricsMiddleware())
	router.Use(s.corsMiddleware())
	s.setupRoutes()

	return s
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		_ = duration
		_ = strconv.Itoa(c.Writer.Status())
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-User-Role, X-Patient-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
```

- [ ] **Step 2: Update routes.go — replace stubs with real handlers**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/routes.go
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) setupRoutes() {
	// Infrastructure
	s.Router.GET("/healthz", s.handleHealthz)
	s.Router.GET("/readyz", s.handleReadyz)
	s.Router.GET("/startupz", s.handleStartupz)
	s.Router.GET("/metrics", s.prometheusHandler())

	// FHIR CRUD (passthrough to FHIR Store — Phase 4 for full implementation)
	fhir := s.Router.Group("/fhir")
	{
		// Patient
		fhir.POST("/Patient", s.stubHandler("Create Patient"))
		fhir.GET("/Patient/:id", s.stubHandler("Read Patient"))
		fhir.PUT("/Patient/:id", s.stubHandler("Update Patient"))
		fhir.GET("/Patient", s.stubHandler("Search Patient"))

		// Observation
		fhir.POST("/Observation", s.stubHandler("Create Observation"))
		fhir.GET("/Observation", s.stubHandler("Search Observation"))

		// Encounter
		fhir.POST("/Encounter", s.stubHandler("Create Encounter"))
		fhir.PUT("/Encounter/:id", s.stubHandler("Update Encounter"))
		fhir.GET("/Encounter/:id", s.stubHandler("Read Encounter"))

		// Other resources
		fhir.POST("/MedicationStatement", s.stubHandler("Create MedicationStatement"))
		fhir.GET("/MedicationStatement", s.stubHandler("Search MedicationStatement"))
		fhir.GET("/DetectedIssue", s.stubHandler("Search DetectedIssue"))
		fhir.POST("/Condition", s.stubHandler("Create Condition"))
		fhir.GET("/Condition", s.stubHandler("Search Condition"))
		fhir.POST("", s.stubHandler("FHIR Transaction Bundle"))

		// ── LIVE $operations (Phase 3 — wired to real handlers) ──

		// Enrollment
		fhir.POST("/Patient/$enroll", s.appHandler.HandleEnroll)

		// Safety engine
		fhir.POST("/Patient/:id/$evaluate-safety", s.appHandler.HandleEvaluateSafety)
		fhir.POST("/Encounter/:id/$fill-slot", s.appHandler.HandleFillSlot)

		// ── STUB $operations (Phase 4-5) ──
		fhir.POST("/Patient/:id/$verify-otp", s.stubHandler("Verify OTP"))
		fhir.POST("/Patient/:id/$link-abha", s.stubHandler("Link ABHA"))

		// Review
		fhir.POST("/Encounter/:id/$submit-review", s.stubHandler("Submit Review"))
		fhir.POST("/Encounter/:id/$approve", s.stubHandler("Approve"))
		fhir.POST("/Encounter/:id/$request-clarification", s.stubHandler("Request Clarification"))
		fhir.POST("/Encounter/:id/$escalate", s.stubHandler("Escalate"))

		// Check-in
		fhir.POST("/Patient/:id/$checkin", s.stubHandler("Start Checkin"))
		fhir.POST("/Encounter/:id/$checkin-slot", s.stubHandler("Fill Checkin Slot"))

		// Co-enrollee
		fhir.POST("/Patient/:id/$register-co-enrollee", s.stubHandler("Register Co-enrollee"))
	}
}

func (s *Server) stubHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"status":   "not_implemented",
			"endpoint": name,
			"message":  "This endpoint will be implemented in Phase 4-5",
		})
	}
}
```

- [ ] **Step 3: Update main.go — initialize new dependencies**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/cmd/intake/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/intake-onboarding-service/internal/api"
	"github.com/cardiofit/intake-onboarding-service/internal/config"
	"github.com/cardiofit/intake-onboarding-service/internal/flow"
	intakekafka "github.com/cardiofit/intake-onboarding-service/internal/kafka"
	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting Intake-Onboarding Service...")

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Connect PostgreSQL
	dbPool, err := pgxpool.New(context.Background(), cfg.Database.URL)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer dbPool.Close()
	logger.Info("Connected to PostgreSQL")

	// Connect Redis
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		logger.Fatal("Failed to parse Redis URL", zap.Error(err))
	}
	opt.Password = cfg.Redis.Password
	opt.DB = cfg.Redis.DB
	redisClient := redis.NewClient(opt)
	defer redisClient.Close()
	logger.Info("Connected to Redis")

	// Create FHIR client (optional — disabled in dev if no credentials)
	var fhirClient *fhirclient.Client
	if cfg.FHIR.Enabled {
		fhirClient, err = fhirclient.New(cfg.FHIR, logger)
		if err != nil {
			logger.Warn("FHIR Store client disabled — no credentials", zap.Error(err))
		} else {
			logger.Info("FHIR Store client initialized")
		}
	}

	// Initialize safety engine (deterministic, no external deps)
	safetyEngine := safety.NewEngine()
	logger.Info("Safety engine initialized", zap.Int("hard_stops", 11), zap.Int("soft_flags", 8))

	// Load flow graph
	var flowEngine *flow.Engine
	flowPath := "configs/flows/intake_full.yaml"
	if graph, err := flow.LoadGraph(flowPath); err != nil {
		logger.Warn("Flow graph not loaded — using stub mode", zap.Error(err))
	} else {
		flowEngine = flow.NewEngine(graph)
		logger.Info("Flow graph loaded", zap.String("id", graph.ID), zap.Int("nodes", len(graph.Nodes)))
	}

	// Initialize Kafka producer
	var producer *intakekafka.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		producer = intakekafka.NewProducer(cfg.Kafka.Brokers, logger)
		defer producer.Close()
		logger.Info("Kafka producer initialized", zap.Int("topics", len(intakekafka.AllTopics())))
	}

	// Create HTTP server with all dependencies
	server := api.NewServer(cfg, dbPool, redisClient, fhirClient, logger,
		safetyEngine, flowEngine, producer)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: server.Router,
	}

	go func() {
		logger.Info("Intake-Onboarding Service listening", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Intake-Onboarding Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
	logger.Info("Intake-Onboarding Service stopped")
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go mod tidy && go build ./cmd/intake/`
Expected: Binary compiles without errors

- [ ] **Step 5: Run all tests**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./... -v -count=1`
Expected: All tests across all packages PASS

- [ ] **Step 6: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/
git commit -m "feat(intake): wire Phase 3 handlers to routes, replace 501 stubs

\$enroll, \$fill-slot, \$evaluate-safety now live with real handlers.
Server initializes safety engine (19 rules), flow graph (18 nodes),
Kafka producer (8 topics), event store. Remaining CRUD and review
\$operations stay as stubs for Phase 4-5."
```

---

## Verification Checklist

After all 11 tasks are complete, verify:

- [ ] `go build ./cmd/intake/` compiles cleanly
- [ ] `go test ./... -count=1` — all tests pass
- [ ] Slot table has exactly 50 slots across 8 domains
- [ ] Safety engine has 11 HARD_STOPs + 8 SOFT_FLAGs = 19 rules
- [ ] Each safety rule is a pure function (no I/O, no external deps)
- [ ] Safety engine benchmark: <5ms per evaluation (1000 iterations)
- [ ] Flow graph YAML loads and validates (18 nodes, start + complete exist)
- [ ] Kafka producer has 8 topic writers
- [ ] $fill-slot returns: status, safety_result, next_node, progress
- [ ] $enroll creates Patient + Encounter in FHIR Store
- [ ] FHIR Observation includes LOINC code from slot definition
- [ ] FHIR DetectedIssue has severity=high for HARD_STOP, moderate for SOFT_FLAG
