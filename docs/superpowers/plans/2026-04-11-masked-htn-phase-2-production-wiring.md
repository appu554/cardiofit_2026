# Masked Hypertension Phase 2 — Production Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Phase 1 BP context classifier and card system actually reachable, callable, persistable, observable, and configurable in a running CardioFit deployment.

**Architecture:** Phase 1 built `ClassifyBPContext()` and `EvaluateMaskedHTNCards()` as pure library functions with passing tests but zero platform integration. Phase 2 wires them into the platform: KB-26 exposes an HTTP endpoint, KB-23 calls it via a new client, the migration runs at startup, history is persisted, market YAML configs are loaded at boot, upstream BP/morning-surge/engagement values are fetched from KB-20 and KB-21, and observability + event publication follow existing patterns.

**Tech Stack:** Go 1.22+ (Gin, GORM, Zap, Prometheus), PostgreSQL 15, YAML (gopkg.in/yaml.v3 — already in KB-23 go.mod)

**Pre-requisite:** Phase 1 commits `fa1d5608` through `544b410a` are present on the working branch. KB-26 service compiles. KB-23 service compiles.

---

## Architectural Decisions Locked Before Tasks

These decisions came out of the gap analysis and exploration. They are NOT open questions in this plan — they are fixed constraints the tasks below assume.

### Decision 1: BP reading storage

**Decision:** Phase 2 does NOT introduce a new `bp_readings` table. Individual readings with `Source`/`TimeContext`/`Timestamp` annotations do not exist anywhere in the Go services today, and creating that storage layer is a multi-week data-architecture project (FHIR Observation ingestion, Flink sink changes, retention policy).

**Instead:** Phase 2 reads from `patient_profiles` aggregate fields that already exist (`SBP14dMean`, `MorningSurge7dAvg`, etc.) and KB-26's `twin_state.SBP14dMean`. The classifier is wrapped in an adapter that constructs synthetic `[]BPReading` slices from these aggregates so Phase 1's library code is not changed. The clinic-vs-home discordance is computed by reading TWO aggregates: home mean (from `patient_profiles`) and a new `clinic_sbp_mean` field that Phase 2 adds via a new ingestion path.

**Trade-off accepted:** Without per-reading source data, the classifier operates on means rather than reading lists. Day-1 discard, separate-visit checks, and morning/evening differential medication-timing detection cannot work in Phase 2 — they degrade to "always returns empty." Phase 3 (out of scope here) would add raw reading storage to enable them. This is documented in each affected task.

### Decision 2: Engagement phenotype mapping

**Decision:** KB-21 exposes `BehavioralPhenotype` enum (`CHAMPION/STEADY/SPORADIC/DECLINING/DORMANT/CHURNED`) — `MEASUREMENT_AVOIDANT` and `CRISIS_ONLY_MEASURER` do not exist there.

**Instead:** Phase 2 introduces a mapping function in the new KB-26-side KB-21 client. `DORMANT` and `CHURNED` map to `MEASUREMENT_AVOIDANT`; `SPORADIC` with `EngagementComposite < 0.5` maps to `CRISIS_ONLY_MEASURER`; everything else maps to empty string (no bias). The Phase 1 classifier already handles empty string correctly (no flag fires).

### Decision 3: YAML market config loader

**Decision:** No YAML market config loader exists anywhere in `knowledge-base-services`. Building a generic one is out of scope. Phase 2 ships a minimal, BP-context-specific loader that reads three YAML files and exposes a `BPContextThresholds` struct. It is intentionally NOT a general-purpose framework — it is one struct, one loader, and one resolver, callable from `main.go` at startup.

### Decision 4: Migration mechanism

**Decision:** KB-26 uses GORM `AutoMigrate` exclusively (no golang-migrate, no goose). The Phase 1 SQL file `migrations/006_bp_context.sql` is dead code — nothing runs it. Phase 2 fixes this by adding `&models.BPContextHistory{}` to the existing `db.DB.AutoMigrate(...)` call in `main.go`. The SQL file stays as documentation but is NOT the active mechanism.

### Decision 5: Event publication

**Decision:** KB-23 does not publish to Kafka. All outbound events go via HTTP POST to `{KB19_URL}/api/v1/events`. Phase 2 follows this pattern — masked HTN phenotype changes publish via KB-23's existing `kb19_publisher.go` with two new event types.

### Decision 6: Card text source

**Decision:** Phase 1 hardcoded all 8 card rationales as Go string literals in `masked_htn_cards.go`. Migrating them to YAML fragment templates is a non-trivial rewrite touching `template_loader.go`, `fragment_loader.go`, `card_builder.go`, and the SLA scanner. **Phase 2 leaves this for Phase 3.** Hardcoded strings stay. This is documented as a known gap.

---

## File Structure

### KB-26 (Metabolic Digital Twin)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-26-metabolic-digital-twin/internal/api/bp_context_handlers.go` | HTTP handler `POST /api/v1/kb26/bp-context/:patientId` |
| Create | `kb-26-metabolic-digital-twin/internal/services/bp_context_repository.go` | GORM repository for `BPContextHistory` snapshots |
| Create | `kb-26-metabolic-digital-twin/internal/services/bp_context_repository_test.go` | Repository tests using sqlite in-memory |
| Create | `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go` | Orchestrator: fetches inputs from KB-20/KB-21, calls classifier, persists snapshot |
| Create | `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_test.go` | Orchestrator tests with stubbed clients |
| Create | `kb-26-metabolic-digital-twin/internal/clients/kb20_client.go` | HTTP client to fetch KB-20 patient profile (BP aggregates, morning surge) |
| Create | `kb-26-metabolic-digital-twin/internal/clients/kb21_client.go` | HTTP client to fetch KB-21 engagement profile (with phenotype mapping) |
| Create | `kb-26-metabolic-digital-twin/internal/config/bp_context_thresholds.go` | YAML loader for shared + market overrides |
| Create | `kb-26-metabolic-digital-twin/internal/config/bp_context_thresholds_test.go` | YAML loader tests |
| Create | `kb-26-metabolic-digital-twin/internal/services/bp_context_metrics.go` | Prometheus counters and histograms |
| Modify | `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier.go` | Replace hardcoded constants with `ThresholdsConfig` parameter |
| Modify | `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier_test.go` | Update tests for new signature |
| Modify | `kb-26-metabolic-digital-twin/internal/api/routes.go` | Register BP context route |
| Modify | `kb-26-metabolic-digital-twin/internal/api/server.go` | Wire orchestrator into Server struct |
| Modify | `kb-26-metabolic-digital-twin/main.go` | AutoMigrate BPContextHistory, load YAML config, wire clients |
| Modify | `kb-26-metabolic-digital-twin/internal/config/config.go` | Add KB20_URL, KB21_URL, MARKET_CONFIG_DIR, MARKET_CODE env vars |
| Modify | `kb-26-metabolic-digital-twin/internal/metrics/collector.go` | Add BP context metric fields |

### KB-23 (Decision Cards)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-23-decision-cards/internal/services/kb26_bp_context_client.go` | HTTP client to call KB-26 `POST /bp-context/:id` |
| Create | `kb-23-decision-cards/internal/services/kb26_bp_context_client_test.go` | Client tests with `httptest` |
| Modify | `kb-23-decision-cards/internal/services/kb19_publisher.go` | Add `PublishMaskedHTNDetected` and `PublishPhenotypeChanged` |
| Modify | `kb-23-decision-cards/internal/api/server.go` | Instantiate `KB26BPContextClient` in `InitServices()` |
| Modify | `kb-23-decision-cards/internal/config/config.go` | Add `KB26_URL`, `KB26Timeout()` |

**Total: 17 create, 9 modify = 26 files**

---

## Task 1: Add `BPContextHistory` to AutoMigrate

This is the smallest possible step that makes the Phase 1 migration actually create a table. No new code — just one struct added to an existing call.

**Files:**
- Modify: `kb-26-metabolic-digital-twin/main.go`

- [ ] **Step 1: Read the current AutoMigrate call**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && grep -n "AutoMigrate" main.go`
Expected output: shows the existing `db.DB.AutoMigrate(...)` call around line 55-64.

- [ ] **Step 2: Add `BPContextHistory` to the AutoMigrate list**

In `main.go`, locate the existing AutoMigrate call:

```go
if err := db.DB.AutoMigrate(
    &models.TwinState{},
    &models.CalibratedEffect{},
    &models.SimulationRun{},
    &models.MRIScore{},
    &models.MRINadir{},
    &models.RelapseEvent{},
    &models.QuarterlySummary{},
    &models.PREVENTScore{},
); err != nil {
    logger.Fatal("Failed to auto-migrate models", zap.Error(err))
}
```

Add `&models.BPContextHistory{},` to the list (placement: after `&models.PREVENTScore{},`).

- [ ] **Step 3: Build to verify compilation**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/main.go
git commit -m "fix(kb26): register BPContextHistory in AutoMigrate

Phase 1 created migration 006_bp_context.sql but KB-26 uses GORM
AutoMigrate exclusively. Without this registration the bp_context_history
table is never created on service startup, and any code persisting to it
would fail with relation does not exist."
```

---

## Task 2: BP Context Repository (KB-26)

GORM repository wrapping `BPContextHistory` for persistence. Uses sqlite in-memory in tests, matching the existing pattern in `egfr_trajectory_test.go`.

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_repository.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_repository_test.go`

- [ ] **Step 1: Write the failing repository test**

Create `bp_context_repository_test.go`:

```go
package services

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

func setupBPContextTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.BPContextHistory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestBPContextRepository_SaveAndFetchLatest(t *testing.T) {
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)

	snapshot := &models.BPContextHistory{
		PatientID:     "p1",
		SnapshotDate:  time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC),
		Phenotype:     models.PhenotypeMaskedHTN,
		ClinicSBPMean: 128,
		HomeSBPMean:   148,
		GapSBP:        -20,
		Confidence:    "HIGH",
	}

	if err := repo.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	latest, err := repo.FetchLatest("p1")
	if err != nil {
		t.Fatalf("FetchLatest failed: %v", err)
	}
	if latest == nil {
		t.Fatal("expected non-nil latest snapshot")
	}
	if latest.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", latest.Phenotype)
	}
}

func TestBPContextRepository_SaveSnapshot_UpsertOnSameDay(t *testing.T) {
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)

	day := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)

	first := &models.BPContextHistory{
		PatientID:    "p1",
		SnapshotDate: day,
		Phenotype:    models.PhenotypeMaskedHTN,
		Confidence:   "HIGH",
	}
	if err := repo.SaveSnapshot(first); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Reclassification on the same day should upsert, not duplicate.
	second := &models.BPContextHistory{
		PatientID:    "p1",
		SnapshotDate: day,
		Phenotype:    models.PhenotypeMaskedUncontrolled,
		Confidence:   "HIGH",
	}
	if err := repo.SaveSnapshot(second); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	all, err := repo.FetchHistory("p1", 10)
	if err != nil {
		t.Fatalf("FetchHistory failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 snapshot after upsert, got %d", len(all))
	}
	if all[0].Phenotype != models.PhenotypeMaskedUncontrolled {
		t.Errorf("expected upserted phenotype MASKED_UNCONTROLLED, got %s", all[0].Phenotype)
	}
}

func TestBPContextRepository_FetchLatest_NotFound(t *testing.T) {
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)

	latest, err := repo.FetchLatest("unknown")
	if err != nil {
		t.Fatalf("FetchLatest should return nil snapshot, no error; got err=%v", err)
	}
	if latest != nil {
		t.Errorf("expected nil for unknown patient, got %+v", latest)
	}
}

func TestBPContextRepository_FetchHistory_OrderedDesc(t *testing.T) {
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)

	dates := []time.Time{
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC),
	}
	for _, d := range dates {
		if err := repo.SaveSnapshot(&models.BPContextHistory{
			PatientID:    "p1",
			SnapshotDate: d,
			Phenotype:    models.PhenotypeSustainedHTN,
			Confidence:   "HIGH",
		}); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	history, err := repo.FetchHistory("p1", 10)
	if err != nil {
		t.Fatalf("FetchHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 history rows, got %d", len(history))
	}
	if !history[0].SnapshotDate.Equal(dates[2]) {
		t.Errorf("expected newest first; got %v", history[0].SnapshotDate)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBPContextRepository" -v 2>&1 | head -8`
Expected: compilation error — `NewBPContextRepository` undefined.

- [ ] **Step 3: Create the repository implementation**

Create `bp_context_repository.go`:

```go
package services

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kb-26-metabolic-digital-twin/internal/models"
)

// BPContextRepository persists BP context classification snapshots for
// progression tracking (e.g. WCH -> SH conversion over months).
type BPContextRepository struct {
	db *gorm.DB
}

// NewBPContextRepository constructs a repository bound to the given DB handle.
func NewBPContextRepository(db *gorm.DB) *BPContextRepository {
	return &BPContextRepository{db: db}
}

// SaveSnapshot inserts a new snapshot, or upserts if one already exists for
// the same (patient_id, snapshot_date) — reclassification on the same day
// replaces the prior row rather than creating a duplicate.
func (r *BPContextRepository) SaveSnapshot(snapshot *models.BPContextHistory) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "patient_id"},
			{Name: "snapshot_date"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"phenotype",
			"clinic_sbp_mean",
			"home_sbp_mean",
			"gap_sbp",
			"confidence",
		}),
	}).Create(snapshot).Error
}

// FetchLatest returns the most recent snapshot for a patient, or nil if none.
func (r *BPContextRepository) FetchLatest(patientID string) (*models.BPContextHistory, error) {
	var snapshot models.BPContextHistory
	err := r.db.
		Where("patient_id = ?", patientID).
		Order("snapshot_date DESC").
		First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// FetchHistory returns up to `limit` snapshots for a patient, newest first.
func (r *BPContextRepository) FetchHistory(patientID string, limit int) ([]models.BPContextHistory, error) {
	var snapshots []models.BPContextHistory
	err := r.db.
		Where("patient_id = ?", patientID).
		Order("snapshot_date DESC").
		Limit(limit).
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}
```

- [ ] **Step 4: Run tests — all 4 must pass**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBPContextRepository" -v`
Expected: 4/4 PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/services/bp_context_repository.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_repository_test.go
git commit -m "feat(kb26): BP context history repository with upsert semantics

GORM repository wrapping BPContextHistory. Same-day reclassification
upserts rather than duplicating. FetchLatest returns nil (not error) for
unknown patients. FetchHistory returns newest-first."
```

---

## Task 3: BP Context Thresholds Config Loader

Loads `market-configs/shared/bp_context_thresholds.yaml`, applies a market-specific override file, and exposes a typed `BPContextThresholds` struct.

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/config/bp_context_thresholds.go`
- Create: `kb-26-metabolic-digital-twin/internal/config/bp_context_thresholds_test.go`

- [ ] **Step 1: Write the failing test**

Create `bp_context_thresholds_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sharedYAMLContent = `
thresholds:
  clinic:
    sbp_elevated: 140
    dbp_elevated: 90
    sbp_elevated_dm: 130
    dbp_elevated_dm: 80
  home:
    sbp_elevated: 135
    dbp_elevated: 85

data_requirements:
  clinic:
    min_readings: 2
    max_age_days: 90
  home:
    min_readings: 12
    min_days: 4
    max_age_days: 14

white_coat_effect:
  clinically_significant: 15
  severe: 30

selection_bias:
  min_home_readings_for_confidence: 20
  flag_if_readings_below: 12
`

const indiaOverrideContent = `
white_coat_effect_override:
  clinically_significant: 20
`

func writeTempYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoadBPContextThresholds_SharedOnly(t *testing.T) {
	dir := t.TempDir()
	sharedDir := filepath.Join(dir, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTempYAML(t, sharedDir, "bp_context_thresholds.yaml", sharedYAMLContent)

	thresholds, err := LoadBPContextThresholds(dir, "us")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if thresholds.ClinicSBPElevated != 140 {
		t.Errorf("expected ClinicSBPElevated=140, got %f", thresholds.ClinicSBPElevated)
	}
	if thresholds.ClinicSBPElevatedDM != 130 {
		t.Errorf("expected ClinicSBPElevatedDM=130, got %f", thresholds.ClinicSBPElevatedDM)
	}
	if thresholds.WCEClinicallySignificant != 15 {
		t.Errorf("expected WCE=15 (no override), got %f", thresholds.WCEClinicallySignificant)
	}
}

func TestLoadBPContextThresholds_IndiaOverride(t *testing.T) {
	dir := t.TempDir()
	sharedDir := filepath.Join(dir, "shared")
	indiaDir := filepath.Join(dir, "india")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("mkdir shared: %v", err)
	}
	if err := os.MkdirAll(indiaDir, 0o755); err != nil {
		t.Fatalf("mkdir india: %v", err)
	}
	writeTempYAML(t, sharedDir, "bp_context_thresholds.yaml", sharedYAMLContent)
	writeTempYAML(t, indiaDir, "bp_context_overrides.yaml", indiaOverrideContent)

	thresholds, err := LoadBPContextThresholds(dir, "india")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if thresholds.WCEClinicallySignificant != 20 {
		t.Errorf("expected WCE=20 (India override), got %f", thresholds.WCEClinicallySignificant)
	}
	// Non-overridden values should still come from shared.
	if thresholds.ClinicSBPElevated != 140 {
		t.Errorf("expected ClinicSBPElevated=140 (shared), got %f", thresholds.ClinicSBPElevated)
	}
}

func TestLoadBPContextThresholds_MissingShared(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadBPContextThresholds(dir, "us")
	if err == nil {
		t.Fatal("expected error when shared YAML missing")
	}
}

func TestLoadBPContextThresholds_UnknownMarketUsesShared(t *testing.T) {
	dir := t.TempDir()
	sharedDir := filepath.Join(dir, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTempYAML(t, sharedDir, "bp_context_thresholds.yaml", sharedYAMLContent)

	thresholds, err := LoadBPContextThresholds(dir, "mars")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	// Unknown market: no override file present, shared values used.
	if thresholds.WCEClinicallySignificant != 15 {
		t.Errorf("expected shared WCE=15, got %f", thresholds.WCEClinicallySignificant)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/config/ -run "TestLoadBPContextThresholds" -v 2>&1 | head -8`
Expected: compilation error — `LoadBPContextThresholds` undefined.

- [ ] **Step 3: Create the loader**

KB-26 does NOT currently have `gopkg.in/yaml.v3` in its go.mod. First add it:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go get gopkg.in/yaml.v3@v3.0.1`

Then create `bp_context_thresholds.go`:

```go
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// BPContextThresholds is the loaded configuration consumed by the BP
// context classifier. Fields are flattened from the YAML hierarchy for
// ergonomic access at call sites.
type BPContextThresholds struct {
	// Clinic thresholds
	ClinicSBPElevated   float64
	ClinicDBPElevated   float64
	ClinicSBPElevatedDM float64
	ClinicDBPElevatedDM float64

	// Home thresholds
	HomeSBPElevated float64
	HomeDBPElevated float64

	// Data requirements
	MinClinicReadings    int
	ClinicMaxAgeDays     int
	MinHomeReadings      int
	MinHomeDays          int
	HomeMaxAgeDays       int

	// White-coat effect
	WCEClinicallySignificant float64
	WCESevere                float64

	// Selection bias
	MinHomeForConfidence int
	FlagIfReadingsBelow  int
}

// rawSharedConfig matches the YAML structure of bp_context_thresholds.yaml.
type rawSharedConfig struct {
	Thresholds struct {
		Clinic struct {
			SBPElevated   float64 `yaml:"sbp_elevated"`
			DBPElevated   float64 `yaml:"dbp_elevated"`
			SBPElevatedDM float64 `yaml:"sbp_elevated_dm"`
			DBPElevatedDM float64 `yaml:"dbp_elevated_dm"`
		} `yaml:"clinic"`
		Home struct {
			SBPElevated float64 `yaml:"sbp_elevated"`
			DBPElevated float64 `yaml:"dbp_elevated"`
		} `yaml:"home"`
	} `yaml:"thresholds"`
	DataRequirements struct {
		Clinic struct {
			MinReadings int `yaml:"min_readings"`
			MaxAgeDays  int `yaml:"max_age_days"`
		} `yaml:"clinic"`
		Home struct {
			MinReadings int `yaml:"min_readings"`
			MinDays     int `yaml:"min_days"`
			MaxAgeDays  int `yaml:"max_age_days"`
		} `yaml:"home"`
	} `yaml:"data_requirements"`
	WhiteCoatEffect struct {
		ClinicallySignificant float64 `yaml:"clinically_significant"`
		Severe                float64 `yaml:"severe"`
	} `yaml:"white_coat_effect"`
	SelectionBias struct {
		MinHomeForConfidence int `yaml:"min_home_readings_for_confidence"`
		FlagIfReadingsBelow  int `yaml:"flag_if_readings_below"`
	} `yaml:"selection_bias"`
}

// rawOverrideConfig matches the YAML structure of *_overrides.yaml.
type rawOverrideConfig struct {
	ThresholdsOverride struct {
		Clinic *struct {
			SBPElevated   *float64 `yaml:"sbp_elevated"`
			DBPElevated   *float64 `yaml:"dbp_elevated"`
			SBPElevatedDM *float64 `yaml:"sbp_elevated_dm"`
			DBPElevatedDM *float64 `yaml:"dbp_elevated_dm"`
		} `yaml:"clinic"`
	} `yaml:"thresholds_override"`
	WhiteCoatEffectOverride *struct {
		ClinicallySignificant *float64 `yaml:"clinically_significant"`
		Severe                *float64 `yaml:"severe"`
	} `yaml:"white_coat_effect_override"`
}

// LoadBPContextThresholds reads shared thresholds from
// {configDir}/shared/bp_context_thresholds.yaml and applies the market
// override from {configDir}/{market}/bp_context_overrides.yaml if present.
// Unknown markets are NOT errors — only shared values are used.
func LoadBPContextThresholds(configDir, market string) (*BPContextThresholds, error) {
	sharedPath := filepath.Join(configDir, "shared", "bp_context_thresholds.yaml")
	sharedBytes, err := os.ReadFile(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("read shared BP context thresholds: %w", err)
	}

	var shared rawSharedConfig
	if err := yaml.Unmarshal(sharedBytes, &shared); err != nil {
		return nil, fmt.Errorf("parse shared BP context thresholds: %w", err)
	}

	t := &BPContextThresholds{
		ClinicSBPElevated:        shared.Thresholds.Clinic.SBPElevated,
		ClinicDBPElevated:        shared.Thresholds.Clinic.DBPElevated,
		ClinicSBPElevatedDM:      shared.Thresholds.Clinic.SBPElevatedDM,
		ClinicDBPElevatedDM:      shared.Thresholds.Clinic.DBPElevatedDM,
		HomeSBPElevated:          shared.Thresholds.Home.SBPElevated,
		HomeDBPElevated:          shared.Thresholds.Home.DBPElevated,
		MinClinicReadings:        shared.DataRequirements.Clinic.MinReadings,
		ClinicMaxAgeDays:         shared.DataRequirements.Clinic.MaxAgeDays,
		MinHomeReadings:          shared.DataRequirements.Home.MinReadings,
		MinHomeDays:              shared.DataRequirements.Home.MinDays,
		HomeMaxAgeDays:           shared.DataRequirements.Home.MaxAgeDays,
		WCEClinicallySignificant: shared.WhiteCoatEffect.ClinicallySignificant,
		WCESevere:                shared.WhiteCoatEffect.Severe,
		MinHomeForConfidence:     shared.SelectionBias.MinHomeForConfidence,
		FlagIfReadingsBelow:      shared.SelectionBias.FlagIfReadingsBelow,
	}

	overridePath := filepath.Join(configDir, market, "bp_context_overrides.yaml")
	overrideBytes, err := os.ReadFile(overridePath)
	if errors.Is(err, os.ErrNotExist) {
		// Unknown market or no override file — return shared-only thresholds.
		return t, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s override: %w", market, err)
	}

	var override rawOverrideConfig
	if err := yaml.Unmarshal(overrideBytes, &override); err != nil {
		return nil, fmt.Errorf("parse %s override: %w", market, err)
	}

	if override.ThresholdsOverride.Clinic != nil {
		c := override.ThresholdsOverride.Clinic
		if c.SBPElevated != nil {
			t.ClinicSBPElevated = *c.SBPElevated
		}
		if c.DBPElevated != nil {
			t.ClinicDBPElevated = *c.DBPElevated
		}
		if c.SBPElevatedDM != nil {
			t.ClinicSBPElevatedDM = *c.SBPElevatedDM
		}
		if c.DBPElevatedDM != nil {
			t.ClinicDBPElevatedDM = *c.DBPElevatedDM
		}
	}
	if override.WhiteCoatEffectOverride != nil {
		w := override.WhiteCoatEffectOverride
		if w.ClinicallySignificant != nil {
			t.WCEClinicallySignificant = *w.ClinicallySignificant
		}
		if w.Severe != nil {
			t.WCESevere = *w.Severe
		}
	}

	return t, nil
}
```

- [ ] **Step 4: Run tests — all 4 must pass**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/config/ -run "TestLoadBPContextThresholds" -v`
Expected: 4/4 PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/config/bp_context_thresholds.go
git add kb-26-metabolic-digital-twin/internal/config/bp_context_thresholds_test.go
git add kb-26-metabolic-digital-twin/go.mod kb-26-metabolic-digital-twin/go.sum
git commit -m "feat(kb26): BP context thresholds YAML loader with market overrides

Loads market-configs/shared/bp_context_thresholds.yaml and applies
market-specific override file if present. Unknown markets gracefully
fall back to shared. First YAML loader in KB-26 — adds yaml.v3 to go.mod."
```

---

## Task 4: Refactor Classifier to Accept Thresholds Parameter

Phase 1's classifier hardcodes thresholds as `const`. Replace with a `*BPContextThresholds` parameter so it can be configured per market. Existing tests must still pass after threading a default.

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier_test.go`

- [ ] **Step 1: Add `defaultBPContextThresholds()` helper to the classifier file**

In `bp_context_classifier.go`, ADD a new helper function near the top (after imports):

```go
// defaultBPContextThresholds returns the ESH 2023 / ISH 2020 reference values
// used when no market config is loaded. Tests use these via the same path
// production code does (call sites pass nil to mean "use defaults").
func defaultBPContextThresholds() *config.BPContextThresholds {
	return &config.BPContextThresholds{
		ClinicSBPElevated:        140.0,
		ClinicDBPElevated:        90.0,
		ClinicSBPElevatedDM:      130.0,
		ClinicDBPElevatedDM:      80.0,
		HomeSBPElevated:          135.0,
		HomeDBPElevated:          85.0,
		MinClinicReadings:        2,
		MinHomeReadings:          12,
		MinHomeDays:              4,
		WCEClinicallySignificant: 15.0,
		MinHomeForConfidence:     20,
		FlagIfReadingsBelow:      12,
	}
}
```

Add `"kb-26-metabolic-digital-twin/internal/config"` to the imports.

- [ ] **Step 2: Change `ClassifyBPContext` signature**

Change the function signature from:

```go
func ClassifyBPContext(input BPContextInput) models.BPContextClassification {
```

to:

```go
func ClassifyBPContext(input BPContextInput, thresholds *config.BPContextThresholds) models.BPContextClassification {
	if thresholds == nil {
		thresholds = defaultBPContextThresholds()
	}
```

- [ ] **Step 3: Replace all hardcoded constant references with `thresholds.X`**

Inside `ClassifyBPContext`, replace:
- `defaultClinicSBP` → `thresholds.ClinicSBPElevated`
- `defaultClinicDBP` → `thresholds.ClinicDBPElevated`
- `defaultClinicSBP_DM` → `thresholds.ClinicSBPElevatedDM`
- `defaultClinicDBP_DM` → `thresholds.ClinicDBPElevatedDM`
- `defaultHomeSBP` → `thresholds.HomeSBPElevated`
- `defaultHomeDBP` → `thresholds.HomeDBPElevated`
- `minClinicReadings` → `thresholds.MinClinicReadings`
- `minHomeReadings` → `thresholds.MinHomeReadings`
- `minHomeDays` → `thresholds.MinHomeDays`
- `minHomeForConfidence` → `thresholds.MinHomeForConfidence`

`assessBPConfidence` also needs to take thresholds. Change its signature to:

```go
func assessBPConfidence(result models.BPContextClassification, input BPContextInput, thresholds *config.BPContextThresholds) string {
```

And replace `minHomeForConfidence` inside it with `thresholds.MinHomeForConfidence`. Update the call site inside `ClassifyBPContext`:

```go
result.Confidence = assessBPConfidence(result, input, thresholds)
```

- [ ] **Step 4: Remove the now-unused constants**

Delete the `const ( ... )` block at the top of the file. Keep only `morningSurgeCompoundLimit = 20.0` (this is not a market-tunable threshold; it's clinical literature value).

The `significantWCE = 15.0` constant flagged as unused in the prior code review can also be removed.

- [ ] **Step 5: Update test call sites**

Every call to `ClassifyBPContext(input)` in `bp_context_classifier_test.go` must become `ClassifyBPContext(input, nil)`. There are 14 test functions — update each one.

- [ ] **Step 6: Run all classifier tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestClassifyBPContext" -v`
Expected: 14/14 PASS (passing nil threads through `defaultBPContextThresholds()`, which has identical values to the deleted constants).

- [ ] **Step 7: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/services/bp_context_classifier.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_classifier_test.go
git commit -m "refactor(kb26): BP context classifier accepts thresholds parameter

Phase 1 hardcoded clinic/home thresholds as package constants. Phase 2
makes them per-call so a YAML-loaded BPContextThresholds struct (with
market overrides) can be passed in. Tests pass nil to use the default
values, preserving Phase 1 behaviour."
```

---

## Task 5: KB-26 Side KB-20 Client (fetch patient profile aggregates)

KB-26 currently has no client to KB-20. Phase 2 adds one — narrow scope: just fetch the fields the BP context classifier needs.

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/clients/kb20_client.go`

Note: KB-26 does not currently have `internal/clients/`. Create the directory.

- [ ] **Step 1: Create the directory and client file**

Run: `mkdir -p /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/clients`

Create `kb20_client.go`:

```go
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB20PatientProfile is the subset of KB-20's patient profile that KB-26
// needs for BP context classification. Field names match KB-20's JSON output.
type KB20PatientProfile struct {
	PatientID         string   `json:"patient_id"`
	SBP14dMean        *float64 `json:"sbp_14d_mean,omitempty"`
	DBP14dMean        *float64 `json:"dbp_14d_mean,omitempty"`
	ClinicSBPMean     *float64 `json:"clinic_sbp_mean,omitempty"`
	ClinicDBPMean     *float64 `json:"clinic_dbp_mean,omitempty"`
	ClinicReadings    int      `json:"clinic_readings_count,omitempty"`
	HomeReadings      int      `json:"home_readings_count,omitempty"`
	HomeDaysWithData  int      `json:"home_days_with_data,omitempty"`
	MorningSurge7dAvg *float64 `json:"morning_surge_7d_avg,omitempty"`
	IsDiabetic        bool     `json:"is_diabetic,omitempty"`
	HasCKD            bool     `json:"has_ckd,omitempty"`
	OnHTNMeds         bool     `json:"on_htn_meds,omitempty"`
}

// KB20Client fetches patient profile data from KB-20 for BP context analysis.
type KB20Client struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewKB20Client constructs a client. Timeout is short — KB-20 is on the
// classification hot path.
func NewKB20Client(baseURL string, timeout time.Duration, log *zap.Logger) *KB20Client {
	return &KB20Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
	}
}

// FetchProfile retrieves a patient's profile from KB-20.
func (c *KB20Client) FetchProfile(ctx context.Context, patientID string) (*KB20PatientProfile, error) {
	url := fmt.Sprintf("%s/api/v1/patient/%s/profile", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build KB-20 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-20 unreachable", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("KB-20 fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-20 returned status %d: %s", resp.StatusCode, string(body))
	}

	var profile KB20PatientProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("decode KB-20 response: %w", err)
	}
	return &profile, nil
}
```

- [ ] **Step 2: Build to verify compilation**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./internal/clients/`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/clients/kb20_client.go
git commit -m "feat(kb26): KB-20 client for BP context patient profile fetch

First client in KB-26's internal/clients/. Fetches the subset of patient
profile fields needed for BP context classification: SBP14dMean (home BP
proxy), morning_surge_7d_avg, diabetes/CKD flags, treatment status. Returns
nil on 404 to let callers handle 'patient not found' explicitly."
```

---

## Task 6: KB-26 Side KB-21 Client (fetch + map engagement phenotype)

The classifier wants `MEASUREMENT_AVOIDANT` strings. KB-21 provides `BehavioralPhenotype` (CHAMPION/STEADY/SPORADIC/DECLINING/DORMANT/CHURNED). The mapping logic lives in this client.

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/clients/kb21_client.go`
- Create: `kb-26-metabolic-digital-twin/internal/clients/kb21_client_test.go`

- [ ] **Step 1: Write the failing mapping test**

Create `kb21_client_test.go`:

```go
package clients

import "testing"

func TestMapEngagementToBPPhenotype_DormantToAvoidant(t *testing.T) {
	if got := MapEngagementToBPPhenotype("DORMANT", 0.4); got != "MEASUREMENT_AVOIDANT" {
		t.Errorf("DORMANT should map to MEASUREMENT_AVOIDANT, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_ChurnedToAvoidant(t *testing.T) {
	if got := MapEngagementToBPPhenotype("CHURNED", 0.2); got != "MEASUREMENT_AVOIDANT" {
		t.Errorf("CHURNED should map to MEASUREMENT_AVOIDANT, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_SporadicLowEngagementToCrisis(t *testing.T) {
	if got := MapEngagementToBPPhenotype("SPORADIC", 0.45); got != "CRISIS_ONLY_MEASURER" {
		t.Errorf("SPORADIC + low engagement should map to CRISIS_ONLY_MEASURER, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_SporadicNormalEngagementToEmpty(t *testing.T) {
	if got := MapEngagementToBPPhenotype("SPORADIC", 0.6); got != "" {
		t.Errorf("SPORADIC + normal engagement should map to empty, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_ChampionToEmpty(t *testing.T) {
	if got := MapEngagementToBPPhenotype("CHAMPION", 0.95); got != "" {
		t.Errorf("CHAMPION should map to empty, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_UnknownToEmpty(t *testing.T) {
	if got := MapEngagementToBPPhenotype("MYSTERY", 0.5); got != "" {
		t.Errorf("unknown phenotype should map to empty, got %s", got)
	}
}
```

- [ ] **Step 2: Verify test fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/clients/ -run "TestMapEngagementToBPPhenotype" -v 2>&1 | head -8`
Expected: compilation error — `MapEngagementToBPPhenotype` undefined.

- [ ] **Step 3: Create the KB-21 client**

Create `kb21_client.go`:

```go
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB21EngagementProfile is the subset of KB-21's engagement profile that
// KB-26 uses to derive BP-context engagement phenotype.
type KB21EngagementProfile struct {
	PatientID           string   `json:"patient_id"`
	Phenotype           string   `json:"phenotype"`
	EngagementComposite *float64 `json:"engagement_composite,omitempty"`
}

// KB21Client fetches engagement profile data from KB-21.
type KB21Client struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewKB21Client constructs a client.
func NewKB21Client(baseURL string, timeout time.Duration, log *zap.Logger) *KB21Client {
	return &KB21Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
	}
}

// FetchEngagement retrieves a patient's engagement profile from KB-21.
func (c *KB21Client) FetchEngagement(ctx context.Context, patientID string) (*KB21EngagementProfile, error) {
	url := fmt.Sprintf("%s/api/v1/patient/%s/engagement", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build KB-21 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-21 unreachable", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("KB-21 fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-21 returned status %d: %s", resp.StatusCode, string(body))
	}

	var profile KB21EngagementProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("decode KB-21 response: %w", err)
	}
	return &profile, nil
}

// MapEngagementToBPPhenotype translates KB-21's BehavioralPhenotype enum
// into the BP-context engagement strings the classifier understands.
//
//   DORMANT, CHURNED       -> MEASUREMENT_AVOIDANT
//   SPORADIC + composite < 0.5 -> CRISIS_ONLY_MEASURER
//   anything else          -> "" (no flag, classifier treats as no bias)
//
// engagementComposite may be 0 if KB-21 has not computed it yet.
func MapEngagementToBPPhenotype(kb21Phenotype string, engagementComposite float64) string {
	switch kb21Phenotype {
	case "DORMANT", "CHURNED":
		return "MEASUREMENT_AVOIDANT"
	case "SPORADIC":
		if engagementComposite < 0.5 {
			return "CRISIS_ONLY_MEASURER"
		}
		return ""
	default:
		return ""
	}
}
```

- [ ] **Step 4: Run mapping tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/clients/ -v`
Expected: 6/6 PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/clients/kb21_client.go
git add kb-26-metabolic-digital-twin/internal/clients/kb21_client_test.go
git commit -m "feat(kb26): KB-21 client + engagement phenotype mapping

KB-21 exposes BehavioralPhenotype (CHAMPION/STEADY/SPORADIC/DECLINING/
DORMANT/CHURNED). The BP context classifier expects MEASUREMENT_AVOIDANT
or CRISIS_ONLY_MEASURER. This client fetches KB-21 data and provides
MapEngagementToBPPhenotype() to bridge the two enums:

  DORMANT, CHURNED            -> MEASUREMENT_AVOIDANT
  SPORADIC + composite <0.5   -> CRISIS_ONLY_MEASURER
  everything else             -> empty (no bias flag)"
```

---

## Task 7: BP Context Orchestrator

The orchestrator wires everything together: fetch from KB-20, fetch from KB-21, build `BPContextInput`, call classifier, persist snapshot, return result.

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_test.go`

- [ ] **Step 1: Write the orchestrator test with stubbed clients**

Create `bp_context_orchestrator_test.go`:

```go
package services

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/models"
)

// stubKB20Client implements the KB20Fetcher interface for tests.
type stubKB20Client struct {
	profile *clients.KB20PatientProfile
	err     error
}

func (s *stubKB20Client) FetchProfile(ctx context.Context, patientID string) (*clients.KB20PatientProfile, error) {
	return s.profile, s.err
}

// stubKB21Client implements the KB21Fetcher interface for tests.
type stubKB21Client struct {
	profile *clients.KB21EngagementProfile
	err     error
}

func (s *stubKB21Client) FetchEngagement(ctx context.Context, patientID string) (*clients.KB21EngagementProfile, error) {
	return s.profile, s.err
}

func ptrFloat(v float64) *float64 { return &v }

func newOrchestrator(t *testing.T, kb20 KB20Fetcher, kb21 KB21Fetcher) *BPContextOrchestrator {
	t.Helper()
	db := setupBPContextTestDB(t) // from bp_context_repository_test.go
	repo := NewBPContextRepository(db)
	thresholds := defaultBPContextThresholds()
	return NewBPContextOrchestrator(kb20, kb21, repo, thresholds, zap.NewNop())
}

func TestBPContextOrchestrator_MaskedHTN_Persists(t *testing.T) {
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p1",
			SBP14dMean:       ptrFloat(148),
			DBP14dMean:       ptrFloat(92),
			ClinicSBPMean:    ptrFloat(128),
			ClinicDBPMean:    ptrFloat(78),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
			IsDiabetic:       false,
			HasCKD:           false,
			OnHTNMeds:        false,
		},
	}
	kb21 := &stubKB21Client{
		profile: &clients.KB21EngagementProfile{
			PatientID: "p1",
			Phenotype: "STEADY",
		},
	}
	orch := newOrchestrator(t, kb20, kb21)

	result, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", result.Phenotype)
	}

	// Verify snapshot was persisted.
	latest, err := orch.repo.FetchLatest("p1")
	if err != nil {
		t.Fatalf("FetchLatest failed: %v", err)
	}
	if latest == nil {
		t.Fatal("expected snapshot to be persisted")
	}
	if latest.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected persisted MASKED_HTN, got %s", latest.Phenotype)
	}
}

func TestBPContextOrchestrator_KB20Unavailable_ReturnsError(t *testing.T) {
	kb20 := &stubKB20Client{err: errSimulated()}
	kb21 := &stubKB21Client{}
	orch := newOrchestrator(t, kb20, kb21)

	_, err := orch.Classify(context.Background(), "p1")
	if err == nil {
		t.Error("expected error when KB-20 unavailable")
	}
}

func TestBPContextOrchestrator_KB21Unavailable_ContinuesWithoutEngagement(t *testing.T) {
	// KB-21 is non-critical: the classifier still works without engagement
	// data, just without selection bias detection.
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p1",
			SBP14dMean:       ptrFloat(120),
			DBP14dMean:       ptrFloat(75),
			ClinicSBPMean:    ptrFloat(118),
			ClinicDBPMean:    ptrFloat(74),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
		},
	}
	kb21 := &stubKB21Client{err: errSimulated()}
	orch := newOrchestrator(t, kb20, kb21)

	result, err := orch.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify should tolerate KB-21 outage, got %v", err)
	}
	if result.Phenotype != models.PhenotypeSustainedNormotension {
		t.Errorf("expected SUSTAINED_NORMOTENSION, got %s", result.Phenotype)
	}
	if result.SelectionBiasRisk {
		t.Error("expected no selection bias when KB-21 unavailable")
	}
}

func TestBPContextOrchestrator_PatientNotFound(t *testing.T) {
	kb20 := &stubKB20Client{profile: nil}
	kb21 := &stubKB21Client{}
	orch := newOrchestrator(t, kb20, kb21)

	_, err := orch.Classify(context.Background(), "ghost")
	if err == nil {
		t.Error("expected error for unknown patient")
	}
}

// Local error helper to avoid pulling errors package into test scope.
func errSimulated() error {
	return &simulatedErr{msg: "simulated outage"}
}

type simulatedErr struct{ msg string }

func (e *simulatedErr) Error() string { return e.msg }

// silence unused import warnings in case of partial test selection
var _ = config.BPContextThresholds{}
```

- [ ] **Step 2: Verify test fails (orchestrator not implemented)**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBPContextOrchestrator" -v 2>&1 | head -10`
Expected: compilation error — `BPContextOrchestrator`, `KB20Fetcher`, `KB21Fetcher`, `NewBPContextOrchestrator` all undefined.

- [ ] **Step 3: Create the orchestrator**

Create `bp_context_orchestrator.go`:

```go
package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/models"
)

// KB20Fetcher is the narrow interface the orchestrator needs from KB-20.
// Defined here (not in the clients package) so tests can stub it without
// importing the real client.
type KB20Fetcher interface {
	FetchProfile(ctx context.Context, patientID string) (*clients.KB20PatientProfile, error)
}

// KB21Fetcher is the narrow interface the orchestrator needs from KB-21.
type KB21Fetcher interface {
	FetchEngagement(ctx context.Context, patientID string) (*clients.KB21EngagementProfile, error)
}

// BPContextOrchestrator coordinates upstream fetches, classification, and
// persistence for a single patient's BP context analysis.
type BPContextOrchestrator struct {
	kb20       KB20Fetcher
	kb21       KB21Fetcher
	repo       *BPContextRepository
	thresholds *config.BPContextThresholds
	log        *zap.Logger
}

// NewBPContextOrchestrator wires the orchestrator dependencies.
func NewBPContextOrchestrator(
	kb20 KB20Fetcher,
	kb21 KB21Fetcher,
	repo *BPContextRepository,
	thresholds *config.BPContextThresholds,
	log *zap.Logger,
) *BPContextOrchestrator {
	return &BPContextOrchestrator{
		kb20:       kb20,
		kb21:       kb21,
		repo:       repo,
		thresholds: thresholds,
		log:        log,
	}
}

// Classify is the entry point for BP context analysis. It fetches inputs
// from KB-20 (required) and KB-21 (best-effort), runs the Phase 1
// classifier, and persists the result.
func (o *BPContextOrchestrator) Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error) {
	profile, err := o.kb20.FetchProfile(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("fetch KB-20 profile: %w", err)
	}
	if profile == nil {
		return nil, fmt.Errorf("patient %s not found in KB-20", patientID)
	}

	// KB-21 is best-effort. Outage degrades to "no engagement phenotype",
	// which the classifier handles cleanly (no bias flag fires).
	var engagementPhenotype string
	engagement, kb21Err := o.kb21.FetchEngagement(ctx, patientID)
	if kb21Err != nil {
		o.log.Warn("KB-21 fetch failed; continuing without engagement",
			zap.String("patient_id", patientID),
			zap.Error(kb21Err))
	} else if engagement != nil {
		composite := 0.0
		if engagement.EngagementComposite != nil {
			composite = *engagement.EngagementComposite
		}
		engagementPhenotype = clients.MapEngagementToBPPhenotype(engagement.Phenotype, composite)
	}

	input := buildBPContextInputFromProfile(profile, engagementPhenotype)
	result := ClassifyBPContext(input, o.thresholds)
	result.PatientID = patientID

	// Persist snapshot. If persistence fails, log but do not block the
	// classification result — the caller still gets the analysis.
	snapshot := &models.BPContextHistory{
		PatientID:     patientID,
		SnapshotDate:  time.Now().UTC().Truncate(24 * time.Hour),
		Phenotype:     result.Phenotype,
		ClinicSBPMean: result.ClinicSBPMean,
		HomeSBPMean:   result.HomeSBPMean,
		GapSBP:        result.ClinicHomeGapSBP,
		Confidence:    result.Confidence,
	}
	if err := o.repo.SaveSnapshot(snapshot); err != nil {
		o.log.Error("BP context snapshot persistence failed",
			zap.String("patient_id", patientID),
			zap.Error(err))
	}

	return &result, nil
}

// buildBPContextInputFromProfile constructs synthetic BPReading slices
// from the aggregate values on KB-20's patient profile. This is a Phase 2
// limitation: per-reading data does not exist anywhere in the Go services,
// so we manufacture the minimum number of "readings" needed to satisfy
// the classifier's data sufficiency gates, all carrying the means as values.
//
// Phase 3 would replace this with a real per-reading store and remove
// this synthetic-reading hack.
func buildBPContextInputFromProfile(profile *clients.KB20PatientProfile, engagementPhenotype string) BPContextInput {
	input := BPContextInput{
		PatientID:           profile.PatientID,
		IsDiabetic:          profile.IsDiabetic,
		HasCKD:              profile.HasCKD,
		OnAntihypertensives: profile.OnHTNMeds,
		EngagementPhenotype: engagementPhenotype,
	}
	if profile.MorningSurge7dAvg != nil {
		input.MorningSurge7dAvg = *profile.MorningSurge7dAvg
	}

	// Synthesize clinic readings from the clinic mean.
	if profile.ClinicSBPMean != nil && profile.ClinicDBPMean != nil && profile.ClinicReadings >= 2 {
		count := profile.ClinicReadings
		input.ClinicReadings = make([]BPReading, count)
		now := time.Now()
		for i := 0; i < count; i++ {
			input.ClinicReadings[i] = BPReading{
				SBP:       *profile.ClinicSBPMean,
				DBP:       *profile.ClinicDBPMean,
				Source:    "CLINIC",
				Timestamp: now.AddDate(0, 0, -i*30),
			}
		}
	}

	// Synthesize home readings from the home mean (SBP14dMean stand-in).
	// Spread across distinct days so the classifier's "min distinct days"
	// gate passes when the count is sufficient.
	if profile.SBP14dMean != nil && profile.DBP14dMean != nil && profile.HomeReadings >= 12 {
		count := profile.HomeReadings
		input.HomeReadings = make([]BPReading, count)
		now := time.Now()
		for i := 0; i < count; i++ {
			input.HomeReadings[i] = BPReading{
				SBP:       *profile.SBP14dMean,
				DBP:       *profile.DBP14dMean,
				Source:    "HOME_CUFF",
				Timestamp: now.Add(time.Duration(-i*12) * time.Hour),
			}
		}
	}

	return input
}
```

- [ ] **Step 4: Run orchestrator tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBPContextOrchestrator" -v`
Expected: 4/4 PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_test.go
git commit -m "feat(kb26): BP context orchestrator wiring KB-20/KB-21 to classifier

Fetches patient profile (required) and engagement (best-effort) from
upstream KB services, synthesises the BPContextInput from aggregate
values (Phase 2 limitation: no raw reading store), runs the Phase 1
classifier, persists a daily snapshot. KB-21 outage degrades cleanly:
classification still produced without engagement phenotype.

Phase 2 limitation acknowledged in code comment: ClinicReadings and
HomeReadings are synthesised from KB-20 means rather than fetched as
per-reading data. Phase 3 will replace with real reading storage."
```

---

## Task 8: HTTP Handler + Route Registration (KB-26)

Expose the orchestrator over HTTP at `POST /api/v1/kb26/bp-context/:patientId`.

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/api/bp_context_handlers.go`
- Modify: `kb-26-metabolic-digital-twin/internal/api/routes.go`
- Modify: `kb-26-metabolic-digital-twin/internal/api/server.go`

- [ ] **Step 1: Add orchestrator field to Server struct**

In `internal/api/server.go`, locate the `Server` struct definition. Add a new field:

```go
bpContextOrchestrator *services.BPContextOrchestrator
```

Add a setter or include it in the `NewServer` constructor signature — match whatever pattern is used for `twinUpdater`, `mriScorer`, etc. (the exploration showed these are passed via `NewServer`).

- [ ] **Step 2: Create the handler file**

Create `bp_context_handlers.go`:

```go
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// classifyBPContext handles POST /api/v1/kb26/bp-context/:patientId.
// Body is empty — the patient ID in the path is sufficient; all other
// inputs are fetched from KB-20 and KB-21 by the orchestrator.
func (s *Server) classifyBPContext(c *gin.Context) {
	patientID := c.Param("patientId")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "patientId is required", "MISSING_PATIENT_ID", nil)
		return
	}

	result, err := s.bpContextOrchestrator.Classify(c.Request.Context(), patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "BP context classification failed", "BP_CONTEXT_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, result, map[string]interface{}{
		"patient_id": patientID,
	})
}
```

- [ ] **Step 3: Register the route**

In `routes.go`, inside the `setupRoutes()` function, add to the `v1 := s.Router.Group("/api/v1/kb26")` block:

```go
v1.POST("/bp-context/:patientId", s.classifyBPContext)
```

- [ ] **Step 4: Build to verify compilation**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./...`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/api/bp_context_handlers.go
git add kb-26-metabolic-digital-twin/internal/api/routes.go
git add kb-26-metabolic-digital-twin/internal/api/server.go
git commit -m "feat(kb26): HTTP endpoint POST /api/v1/kb26/bp-context/:patientId

Exposes the BP context orchestrator over HTTP. Body is empty; all upstream
data fetched via the orchestrator's KB-20 and KB-21 clients. Wraps the
Server struct with a bpContextOrchestrator field following the existing
twinUpdater/mriScorer pattern."
```

---

## Task 9: Wire Everything in main.go (KB-26)

Load YAML config at startup, instantiate clients, repository, orchestrator, and pass them into `NewServer`.

**Files:**
- Modify: `kb-26-metabolic-digital-twin/main.go`
- Modify: `kb-26-metabolic-digital-twin/internal/config/config.go`

- [ ] **Step 1: Add new config env vars**

In `internal/config/config.go`, add fields to the `Config` struct:

```go
KB20URL          string
KB21URL          string
KB20Timeout      time.Duration
KB21Timeout      time.Duration
MarketConfigDir  string
MarketCode       string
```

In the `Load()` function, add:

```go
cfg.KB20URL = getEnv("KB20_URL", "http://localhost:8131")
cfg.KB21URL = getEnv("KB21_URL", "http://localhost:8133")
cfg.KB20Timeout = time.Duration(getEnvAsInt("KB20_TIMEOUT_MS", 2000)) * time.Millisecond
cfg.KB21Timeout = time.Duration(getEnvAsInt("KB21_TIMEOUT_MS", 2000)) * time.Millisecond
cfg.MarketConfigDir = getEnv("MARKET_CONFIG_DIR", "../../market-configs")
cfg.MarketCode = getEnv("MARKET_CODE", "shared")
```

Add `"time"` to imports if not already present.

- [ ] **Step 2: Wire startup in main.go**

In `main.go`, after the `db.DB.AutoMigrate(...)` call (which now includes `BPContextHistory` from Task 1), add:

```go
// Load BP context thresholds (Phase 2)
bpThresholds, err := config.LoadBPContextThresholds(cfg.MarketConfigDir, cfg.MarketCode)
if err != nil {
    logger.Warn("BP context thresholds load failed; using defaults",
        zap.String("market", cfg.MarketCode), zap.Error(err))
    // bpThresholds is nil; orchestrator will fall back to defaultBPContextThresholds()
}

// BP context dependencies
kb20Client := clients.NewKB20Client(cfg.KB20URL, cfg.KB20Timeout, logger)
kb21Client := clients.NewKB21Client(cfg.KB21URL, cfg.KB21Timeout, logger)
bpContextRepo := services.NewBPContextRepository(db.DB)
bpContextOrch := services.NewBPContextOrchestrator(kb20Client, kb21Client, bpContextRepo, bpThresholds, logger)
```

Then update the `api.NewServer(...)` call to pass `bpContextOrch` in. Match the exact constructor signature pattern (the exploration showed services are injected via the constructor).

Add imports:

```go
"kb-26-metabolic-digital-twin/internal/clients"
```

- [ ] **Step 3: Build and run quick smoke test**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build -o /tmp/kb26-test ./...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/main.go
git add kb-26-metabolic-digital-twin/internal/config/config.go
git commit -m "feat(kb26): wire BP context orchestrator at startup

main.go now: loads BP context thresholds from market YAML, instantiates
KB-20 and KB-21 clients, creates the repository and orchestrator, and
injects the orchestrator into the HTTP server. Config gets KB20_URL,
KB21_URL, MARKET_CONFIG_DIR, MARKET_CODE env vars with sensible defaults.
Threshold load failure logs a warning and falls back to in-code defaults
rather than refusing to start (graceful degradation)."
```

---

## Task 10: Prometheus Metrics for BP Context

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_metrics.go`
- Modify: `kb-26-metabolic-digital-twin/internal/metrics/collector.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go`

- [ ] **Step 1: Add metric fields to Collector**

In `internal/metrics/collector.go`, locate the `Collector` struct. Add fields:

```go
BPPhenotypeTotal       *prometheus.CounterVec   // labels: phenotype
BPClassifyLatency      prometheus.Histogram
BPClassifyErrors       prometheus.Counter
```

In the constructor (`NewCollector` or equivalent), register them:

```go
c.BPPhenotypeTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "kb26_bp_phenotype_total",
        Help: "Total number of BP context classifications by phenotype",
    },
    []string{"phenotype"},
)
c.BPClassifyLatency = promauto.NewHistogram(
    prometheus.HistogramOpts{
        Name:    "kb26_bp_classify_latency_seconds",
        Help:    "Latency of BP context classification end-to-end",
        Buckets: prometheus.DefBuckets,
    },
)
c.BPClassifyErrors = promauto.NewCounter(
    prometheus.CounterOpts{
        Name: "kb26_bp_classify_errors_total",
        Help: "Total number of BP context classification failures",
    },
)
```

Match the exact registration pattern other metrics use (the existing collector should show whether it uses `promauto` or manual `prometheus.NewCounterVec` + `Register`).

- [ ] **Step 2: Instrument the orchestrator**

In `bp_context_orchestrator.go`, add a `metrics *metrics.Collector` field to `BPContextOrchestrator` and pass it through `NewBPContextOrchestrator`.

In `Classify()`, wrap the body:

```go
func (o *BPContextOrchestrator) Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error) {
    start := time.Now()
    defer func() {
        if o.metrics != nil {
            o.metrics.BPClassifyLatency.Observe(time.Since(start).Seconds())
        }
    }()

    // ... existing body ...

    if err != nil {
        if o.metrics != nil {
            o.metrics.BPClassifyErrors.Inc()
        }
        return nil, err
    }

    // After successful classification:
    if o.metrics != nil {
        o.metrics.BPPhenotypeTotal.WithLabelValues(string(result.Phenotype)).Inc()
    }
    return &result, nil
}
```

Update existing tests in `bp_context_orchestrator_test.go` to pass `nil` for the metrics collector (the nil-check inside the orchestrator handles it).

Update `main.go` to pass `metricsCollector` into `NewBPContextOrchestrator`.

- [ ] **Step 3: Run tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -v -count=1 2>&1 | tail -20`
Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/metrics/collector.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_test.go
git add kb-26-metabolic-digital-twin/main.go
git commit -m "feat(kb26): Prometheus metrics for BP context classification

kb26_bp_phenotype_total{phenotype} — counter per classification outcome
kb26_bp_classify_latency_seconds — end-to-end histogram
kb26_bp_classify_errors_total — failure counter

Nil-safe: tests can pass nil collector. Orchestrator sums on each call;
main.go injects the real collector at startup."
```

---

## Task 11: KB-23 Side BP Context Client

KB-23 needs a client to call KB-26's new endpoint. Pattern matches the existing `kb20_client.go` exactly.

**Files:**
- Create: `kb-23-decision-cards/internal/services/kb26_bp_context_client.go`
- Create: `kb-23-decision-cards/internal/services/kb26_bp_context_client_test.go`
- Modify: `kb-23-decision-cards/internal/config/config.go`

- [ ] **Step 1: Add KB-26 URL to KB-23 config**

In `kb-23-decision-cards/internal/config/config.go`, add:

```go
KB26URL     string
KB26Timeout time.Duration
```

And in the load function:

```go
cfg.KB26URL = getEnv("KB26_URL", "http://localhost:8137")
cfg.KB26Timeout = time.Duration(getEnvAsInt("KB26_TIMEOUT_MS", 3000)) * time.Millisecond
```

Add a method:

```go
func (c *Config) KB26Timeout_() time.Duration { return c.KB26Timeout }
```

(Match the existing `KB20Timeout()` accessor pattern.)

- [ ] **Step 2: Write the failing client test**

Create `kb26_bp_context_client_test.go`:

```go
package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

func TestKB26BPContextClient_FetchClassification_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/kb26/bp-context/p1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		resp := map[string]interface{}{
			"success": true,
			"data": models.BPContextClassification{
				PatientID:     "p1",
				Phenotype:     models.PhenotypeMaskedHTN,
				ClinicSBPMean: 128,
				HomeSBPMean:   148,
				Confidence:    "HIGH",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewKB26BPContextClient(server.URL, 1*time.Second, zap.NewNop())
	result, err := client.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", result.Phenotype)
	}
}

func TestKB26BPContextClient_FetchClassification_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewKB26BPContextClient(server.URL, 1*time.Second, zap.NewNop())
	result, err := client.Classify(context.Background(), "ghost")
	if err != nil {
		t.Errorf("404 should be nil result, no error; got err=%v", err)
	}
	if result != nil {
		t.Errorf("expected nil for 404, got %+v", result)
	}
}

func TestKB26BPContextClient_FetchClassification_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewKB26BPContextClient(server.URL, 1*time.Second, zap.NewNop())
	_, err := client.Classify(context.Background(), "p1")
	if err == nil {
		t.Error("expected error on 500")
	}
}
```

- [ ] **Step 3: Verify test fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestKB26BPContextClient" -v 2>&1 | head -8`
Expected: compilation error — `NewKB26BPContextClient` undefined.

- [ ] **Step 4: Create the client**

Create `kb26_bp_context_client.go`:

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// KB26BPContextClient calls KB-26's BP context classification endpoint.
type KB26BPContextClient struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewKB26BPContextClient constructs a client.
func NewKB26BPContextClient(baseURL string, timeout time.Duration, log *zap.Logger) *KB26BPContextClient {
	return &KB26BPContextClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
	}
}

// kb26Envelope mirrors KB-26's standard sendSuccess wrapper:
//   {"success": true, "data": {...}, "metadata": {...}}
type kb26Envelope struct {
	Success bool                              `json:"success"`
	Data    *models.BPContextClassification   `json:"data"`
}

// Classify requests a fresh BP context classification for the patient.
// Returns nil (no error) on 404 — caller decides how to handle missing data.
func (c *KB26BPContextClient) Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error) {
	url := fmt.Sprintf("%s/api/v1/kb26/bp-context/%s", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build KB-26 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-26 unreachable", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("KB-26 fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-26 returned status %d: %s", resp.StatusCode, string(body))
	}

	var envelope kb26Envelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode KB-26 response: %w", err)
	}
	return envelope.Data, nil
}
```

- [ ] **Step 5: Run client tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestKB26BPContextClient" -v`
Expected: 3/3 PASS.

- [ ] **Step 6: Wire into KB-23 server initialization**

In `kb-23-decision-cards/internal/api/server.go`, locate `InitServices()` and add (matching the existing `kb20Client` pattern):

```go
s.kb26BPContextClient = services.NewKB26BPContextClient(
    s.cfg.KB26URL,
    s.cfg.KB26Timeout,
    s.log,
)
```

Add `kb26BPContextClient *services.KB26BPContextClient` to the `Server` struct.

- [ ] **Step 7: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-23-decision-cards/internal/services/kb26_bp_context_client.go
git add kb-23-decision-cards/internal/services/kb26_bp_context_client_test.go
git add kb-23-decision-cards/internal/config/config.go
git add kb-23-decision-cards/internal/api/server.go
git commit -m "feat(kb23): KB-26 BP context client with httptest coverage

POST /api/v1/kb26/bp-context/:patientId -> *BPContextClassification.
404 returns nil (not error) for graceful 'patient unknown' handling.
Wired into Server.InitServices() following the existing KB20Client
pattern. KB26_URL and KB26_TIMEOUT_MS env vars added to config."
```

---

## Task 12: KB-23 Event Publisher Methods

Add two methods to `kb19_publisher.go` to publish phenotype events.

**Files:**
- Modify: `kb-23-decision-cards/internal/services/kb19_publisher.go`
- Modify: `kb-23-decision-cards/internal/services/kb19_publisher_test.go` (if it exists; otherwise create)

- [ ] **Step 1: Read existing publisher to understand event struct**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && grep -n "KB19Event" internal/services/kb19_publisher.go internal/models/*.go`
Expected: shows the `models.KB19Event` struct definition. Note its fields exactly — the new methods must use the same envelope.

- [ ] **Step 2: Add `PublishMaskedHTNDetected` and `PublishPhenotypeChanged`**

Append to `kb19_publisher.go`:

```go
// PublishMaskedHTNDetected publishes when a patient is newly classified as
// masked HTN or masked uncontrolled — the highest clinical priority because
// these phenotypes are invisible to clinic-only measurement.
func (p *KB19Publisher) PublishMaskedHTNDetected(patientID string, phenotype string, urgency string) error {
	event := models.KB19Event{
		EventType: "MASKED_HTN_DETECTED",
		PatientID: patientID,
		Payload: map[string]interface{}{
			"phenotype": phenotype,
			"urgency":   urgency,
		},
		Timestamp: time.Now().UTC(),
	}
	return p.publishEvent(event)
}

// PublishPhenotypeChanged publishes when a patient's BP context phenotype
// changes from one classification to another (e.g. WCH -> SH after 6 months).
func (p *KB19Publisher) PublishPhenotypeChanged(patientID string, oldPhenotype, newPhenotype string) error {
	event := models.KB19Event{
		EventType: "BP_PHENOTYPE_CHANGED",
		PatientID: patientID,
		Payload: map[string]interface{}{
			"old_phenotype": oldPhenotype,
			"new_phenotype": newPhenotype,
		},
		Timestamp: time.Now().UTC(),
	}
	return p.publishEvent(event)
}
```

The exact field names of `models.KB19Event` may differ — adjust to match what `grep` showed in Step 1. If the existing struct uses different field names (e.g., `Type` instead of `EventType`), update accordingly.

- [ ] **Step 3: Build to verify**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-23-decision-cards/internal/services/kb19_publisher.go
git commit -m "feat(kb23): publish MASKED_HTN_DETECTED and BP_PHENOTYPE_CHANGED events

Two new methods on KB19Publisher: PublishMaskedHTNDetected for first
detection of masked HTN/MUCH (highest clinical urgency), and
PublishPhenotypeChanged for any phenotype transition (e.g. WCH -> SH
after 6 months). Both POST to KB-19 via the existing publishEvent
helper — no Kafka involved."
```

---

## Task 13: End-to-End Smoke Test

Verify the entire chain compiles, KB-26 starts cleanly, and the new endpoint responds.

**Files:** none — this is a verification task.

- [ ] **Step 1: Full build of both services**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./...
cd ../kb-23-decision-cards && go build ./...
```
Expected: both clean.

- [ ] **Step 2: Full test suite both services**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./... -count=1 2>&1 | tail -15
cd ../kb-23-decision-cards && go test ./... -count=1 2>&1 | tail -15
```
Expected: all tests pass, no regressions in either service.

- [ ] **Step 3: Verify route is registered**

Run: `grep -n "bp-context" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go`
Expected: shows `v1.POST("/bp-context/:patientId", s.classifyBPContext)`.

- [ ] **Step 4: Verify migration is wired**

Run: `grep -n "BPContextHistory" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/main.go`
Expected: shows `&models.BPContextHistory{},` inside the AutoMigrate call.

- [ ] **Step 5: Verify YAML loader is wired**

Run: `grep -n "LoadBPContextThresholds" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/main.go`
Expected: shows the call in startup.

- [ ] **Step 6: Final commit (if any verification surfaced fixes)**

If all 5 grep checks pass and all tests are green, no commit is needed — the verification is complete. If any check failed, dispatch a follow-up fix.

---

## Plan Summary

| Task | Files | Test Cases | Outcome |
|------|-------|------------|---------|
| 1 | 1 modify | 0 (build only) | `BPContextHistory` table created on startup |
| 2 | 2 create | 4 | Repository with upsert semantics |
| 3 | 2 create | 4 | YAML loader with market overrides |
| 4 | 2 modify | 14 (refactored) | Classifier accepts thresholds parameter |
| 5 | 1 create | 0 (compile only) | KB-26-side KB-20 client |
| 6 | 2 create | 6 | KB-26-side KB-21 client + phenotype mapping |
| 7 | 2 create | 4 | Orchestrator wiring everything |
| 8 | 1 create + 2 modify | 0 (build only) | HTTP endpoint registered |
| 9 | 2 modify | 0 (build only) | Startup wiring + env vars |
| 10 | 1 create + 3 modify | 0 (build only) | Prometheus metrics |
| 11 | 2 create + 2 modify | 3 | KB-23-side KB-26 client |
| 12 | 1 modify | 0 (build only) | KB-19 event publishers |
| 13 | 0 | 0 | End-to-end verification |

**Total:** 17 create, 9 modify = 26 files. 35 new test cases across 6 test files. 13 commits expected.

## What This Plan Delivers

After all 13 tasks pass:

- KB-26 has a working HTTP endpoint `POST /api/v1/kb26/bp-context/:patientId`
- KB-26 fetches patient profile from KB-20 and engagement from KB-21 at classification time
- KB-26 persists daily snapshots to `bp_context_history`
- KB-26 emits Prometheus metrics on every classification
- KB-26 reads market-specific thresholds from YAML at startup
- KB-23 has a client to call the KB-26 endpoint
- KB-23 can publish `MASKED_HTN_DETECTED` and `BP_PHENOTYPE_CHANGED` events to KB-19
- The `BPContextHistory` table actually exists in the database
- Engagement phenotype is correctly mapped from KB-21's enum to BP-context strings

## What This Plan Does NOT Deliver (Phase 3 Scope)

The gap analysis identified 31 gaps. Phase 2 closes ~22 of them. The following gaps are deliberately deferred:

1. **Per-reading BP storage** — Phase 2 synthesises readings from aggregates. Gaps G1, G2, G4, G5, G6, G7, P6 (partial), and the day-1 discard / 14-day age filter all wait for real per-reading storage.
2. **Card text fragment templates** — P13. Card text remains hardcoded in `masked_htn_cards.go`. Migrating to YAML fragments touches `template_loader.go`, `fragment_loader.go`, and the rendering pipeline — too large for Phase 2.
3. **Hysteresis engine integration** — P14. Phenotype flapping protection deferred.
4. **Composite card aggregation** — P15. Multiple BP cards firing for the same patient remain separate.
5. **Caller wiring on KB-23 side** — Phase 2 creates `KB26BPContextClient` but does NOT change which existing card-generation flows call it. Wiring the client into `card_builder.go` or a signal consumer is a Phase 3 decision because it depends on triggering policy (every signal? Daily batch? On-demand only?).
6. **WCH progression tracking job** — G8. The `bp_context_history` table will accumulate snapshots, but no scheduled job reads them to detect WCH → SH transitions yet. The publisher exists; the job that calls it does not.
7. **Confidence tier integration** — P11. The `Confidence string` field stays as plain HIGH/MODERATE/LOW. Mapping to KB-23's `ConfidenceTier` enum (Firm/Probable/Possible/Uncertain) is a separate alignment task.

These are genuine work items, but each is independently scoped and can be planned later as their own focused mini-projects.
