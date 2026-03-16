# KB-24 L3 Context Modifier Pipeline — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the extraction pipeline to route contextual modifier and ADR facts through KB-20's existing ADR service, add L3 validation, extend onset categories, and populate 11 FULL-grade drug class records for KB-22's CMApplicator.

**Architecture:** KB-24 is a logical routing label — the physical store is KB-20's `adverse_reaction_profiles` table. The pipeline already extracts `contextual` facts to `KB20ExtractionResult` but the push client (`kb20_push_client.py`) is not wired into the pipeline runner. This plan connects the existing push client, adds `IDIOSYNCRATIC` to onset enums, adds L3 intake validation on the Go side, and authors 11 FULL-grade records (9 drug classes + SGLT2i DKA + PDE5i) as JSON seed data that the push client can ingest.

**Tech Stack:** Python 3.11 (pipeline), Go 1.22 (KB-20 service), PostgreSQL (storage), Pydantic (schemas), GORM (ORM), Gin (HTTP)

**Spec document:** `KB24_L3_Extraction_Template_Spec.md` (repo root)

---

## What Already Exists (Do NOT Rebuild)

| Component | File | Status |
|-----------|------|--------|
| `AdverseReactionProfile` GORM model | `kb-20/.../models/adr_profile.go` | EXISTS — has all 4 elements, completeness grading |
| `ContextModifier` GORM model | `kb-20/.../models/context_modifier.go` | EXISTS — has EffectiveMagnitude() with grade scaling |
| `ADRService.Upsert()` with merge | `kb-20/.../services/adr_service.go` | EXISTS — MANUAL > SPL > PIPELINE priority |
| ADR handlers + routes | `kb-20/.../api/adr_handlers.go`, `routes.go` | EXISTS — GET `/adr/profiles/:drug_class`, POST `/pipeline/adr-profiles` |
| Modifier handlers + routes | `kb-20/.../api/modifier_handlers.go`, `routes.go` | EXISTS — GET `/modifiers/registry/:node_id`, POST `/pipeline/modifiers` |
| `KB20PushClient` | `shared/extraction/v4/kb20_push_client.py` | EXISTS — pushes to `/api/v1/pipeline/adr-profiles` and `/pipeline/modifiers` |
| `FourElementChainAssembler` | `shared/extraction/v4/kb20_push_client.py` | EXISTS — grades chains as FULL/PARTIAL/STUB |
| `KB20ExtractionResult` Pydantic schema | `shared/extraction/schemas/kb20_contextual.py` | EXISTS — `adr_profiles` + `standalone_modifiers` |
| `CMApplicator` in KB-22 | `kb-22/.../services/cm_applicator.go` | EXISTS — supports all 5 effect types |
| Pipeline `target_kbs` mapping | `run_pipeline_targeted.py:1178-1188` | EXISTS — maps `contextual` → `KB-20` |

## File Structure

### Files to Modify

| File | Responsibility | Change |
|------|---------------|--------|
| `shared/tools/guideline-atomiser/data/run_pipeline_targeted.py` | Pipeline runner | Wire `KB20PushClient` after L3 extraction for `contextual` target |
| `shared/extraction/schemas/kb20_contextual.py` | Pydantic schema | Add `IDIOSYNCRATIC` to `onset_category` Literal |
| `kb-20/.../models/adr_profile.go` | Go GORM model | Add `IDIOSYNCRATIC` to CHECK constraint |
| `kb-20/.../migrations/001_initial_schema.sql` | DB migration | Add `IDIOSYNCRATIC` to CHECK constraint |
| `kb-20/.../api/adr_handlers.go` | Go API handlers | Add L3 validation middleware for batch write endpoint |
| `kb-20/.../services/adr_service.go` | Go ADR service | Add `mergePartialCMRule()` for PIPELINE→existing CM rule merge |

### Files to Create

| File | Responsibility |
|------|---------------|
| `shared/extraction/v4/l3_seed_data/arb_oh.json` | ARB → Orthostatic Hypotension (FULL) |
| `shared/extraction/v4/l3_seed_data/sglt2i_volume.json` | SGLT2i → Volume Depletion (FULL) |
| `shared/extraction/v4/l3_seed_data/sglt2i_dka.json` | SGLT2i → Euglycemic DKA (FULL) |
| `shared/extraction/v4/l3_seed_data/bb_hypo_mask.json` | Beta-blocker → Hypo Masking (FULL) |
| `shared/extraction/v4/l3_seed_data/su_hypo.json` | Sulfonylurea → Hypoglycemia (FULL) |
| `shared/extraction/v4/l3_seed_data/acei_cough.json` | ACEi → Cough/Dyspnea (FULL) |
| `shared/extraction/v4/l3_seed_data/ccb_oedema.json` | CCB → Pedal Oedema (FULL) |
| `shared/extraction/v4/l3_seed_data/loop_diuretic_adhf.json` | Loop Diuretic → ADHF Decompensation (FULL) |
| `shared/extraction/v4/l3_seed_data/metformin_lactic.json` | Metformin → Lactic Acidosis (FULL) |
| `shared/extraction/v4/l3_seed_data/thiazide_electrolyte.json` | Thiazide → Electrolyte Derangement (FULL) |
| `shared/extraction/v4/l3_seed_data/pde5i_nitrate.json` | PDE5i → Nitrate HARD_BLOCK (FULL) |
| `shared/extraction/v4/l3_seed_data/insulin_su_combo.json` | Insulin+SU → Severe Hypoglycemia (FULL) — addresses spec Issue 4 |
| `shared/extraction/v4/seed_loader.py` | CLI script to load seed data via `KB20PushClient` |
| `kb-20/.../services/adr_service_test.go` | Tests for `mergePartialCMRule()` |
| `shared/extraction/v4/l3_seed_data/test_seed_schema.py` | Validates all seed JSON against Pydantic schema |
| `shared/pipeline/templates/l3_context_modifier.json` | L3 template definition (spec Section 4.2) |

---

## Chunk 1: Pipeline Wiring + Onset Enum Extension

**Why:** The push client exists but is not called from the pipeline runner. The `IDIOSYNCRATIC` onset category is required by ACEi and is missing from both Python and Go.

### Task 1: Add `IDIOSYNCRATIC` to Onset Category Enums

**Problem:** The spec's ACEi cough record uses `onset_category: IDIOSYNCRATIC`. This value is rejected by both the Python Pydantic Literal and the Go/PostgreSQL CHECK constraint.

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb20_contextual.py:232-239`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/adr_profile.go:30`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/001_initial_schema.sql:155`

- [ ] **Step 1: Update Python schema onset_category Literal**

In `kb20_contextual.py`, line 232-239, change the Literal to include IDIOSYNCRATIC:

```python
    onset_category: Optional[Literal[
        "IMMEDIATE", "ACUTE", "SUBACUTE", "CHRONIC", "DELAYED", "IDIOSYNCRATIC"
    ]] = Field(
        None,
        alias="onsetCategory",
        description="Categorized onset: IMMEDIATE (<1h), ACUTE (1h-7d), "
        "SUBACUTE (1-6wk), CHRONIC (>6wk), DELAYED (variable), "
        "IDIOSYNCRATIC (unpredictable, not PK-determinable)",
    )
```

- [ ] **Step 2: Update Go model CHECK constraint**

In `adr_profile.go`, line 30, change:

```go
	OnsetCategory string `gorm:"size:20;check:onset_category IN ('IMMEDIATE','ACUTE','SUBACUTE','CHRONIC','DELAYED','IDIOSYNCRATIC','')" json:"onset_category,omitempty"`
```

- [ ] **Step 3: Update SQL migration CHECK constraint**

In `001_initial_schema.sql`, line 155, change:

```sql
        CHECK (onset_category IN ('IMMEDIATE','ACUTE','SUBACUTE','CHRONIC','DELAYED','IDIOSYNCRATIC') OR onset_category IS NULL),
```

- [ ] **Step 4: Verify Go builds**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && go build ./...`
Expected: BUILD SUCCESS (no test changes needed — this is a constraint relaxation)

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb20_contextual.py
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/adr_profile.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/001_initial_schema.sql
git commit -m "feat(kb-20): add IDIOSYNCRATIC onset category to ADR profile enums

Required for ACEi cough and other drug reactions where onset is not
PK-determinable. Updates Python Pydantic Literal, Go GORM CHECK
constraint, and SQL migration."
```

---

### Task 2: Wire KB20PushClient into Pipeline Runner

**Problem:** The pipeline extracts `contextual` facts into `KB20ExtractionResult` and saves JSON to disk, but never calls `KB20PushClient` to push the results to KB-20's batch endpoints. The push client exists at `shared/extraction/v4/kb20_push_client.py` but is dead code from the pipeline runner's perspective.

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py:1178-1215`

- [ ] **Step 1: Add push client import**

Near the top imports of `run_pipeline_targeted.py` (around line 30-50), add:

```python
from shared.extraction.v4.kb20_push_client import KB20PushClient
```

- [ ] **Step 2: Add `--push-kb20` CLI flag**

In the `argparse` section (search for `add_argument`), add:

```python
    parser.add_argument(
        "--push-kb20",
        action="store_true",
        default=False,
        help="Push contextual/ADR extraction results to KB-20 via batch API (requires KB-20 running)",
    )
```

> **Note:** The spec also references an `adverse_effects` fact type routed to KB-24. In the current pipeline, only `contextual` exists as a target type. Adding `adverse_effects` as a target alias is out of scope for this plan — it can be added when SPLGuard integration lands. The push wiring below only fires for `kb == "contextual"`.

- [ ] **Step 3: Wire push after L3 extraction for contextual target**

After the L3 extraction loop (around line 1210-1215, after `all_l3_results[dossier.drug_name][kb] = result`), add the push logic inside the `for kb in target_kbs` loop, gated by the `--push-kb20` flag:

```python
                # Push to KB-20 if flag is set and target is contextual
                if args.push_kb20 and kb == "contextual" and result is not None:
                    push_client = KB20PushClient()
                    if push_client.health_check():
                        push_result = push_client.push_extraction(
                            result.model_dump(by_alias=True),
                            profile,
                            source="PIPELINE",
                        )
                        print(f"      📤 KB-20 push: {push_result.total_succeeded} OK, "
                              f"{push_result.total_failed} failed")
                        if push_result.errors:
                            for err in push_result.errors[:3]:
                                print(f"         ⚠️  {err}")
                    else:
                        print(f"      ⚠️  KB-20 not reachable — skipping push (results saved to JSON)")
```

- [ ] **Step 4: Verify pipeline still runs without --push-kb20**

Run: `cd backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data && python run_pipeline_targeted.py --help`
Expected: Shows `--push-kb20` flag in help output. Pipeline runs without error when flag is omitted.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py
git commit -m "feat(pipeline): wire KB20PushClient into pipeline runner with --push-kb20 flag

Contextual/ADR extraction results are pushed to KB-20's batch write
endpoints when --push-kb20 is passed. Push is gated by health check
to avoid failures when KB-20 is offline. Default behavior unchanged."
```

---

### Task 3: Create L3 Template Definition

**Problem:** The spec defines an L3 template JSON schema (Section 4.2) that documents the four-element chain structure, source coverage, and completeness grading. This serves as the contract between pipeline writers and KB-20's ADR service.

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/shared/pipeline/templates/l3_context_modifier.json`

- [ ] **Step 1: Create the template directory and file**

```bash
mkdir -p backend/shared-infrastructure/knowledge-base-services/shared/pipeline/templates
```

- [ ] **Step 2: Write the L3 template**

Create `l3_context_modifier.json` with the content from spec Section 4.2. Key fields:

```json
{
  "template_id": "L3_CONTEXT_MODIFIER_V1",
  "version": "1.0.0",
  "description": "Four-element contextual modifier chain for HPI Bayesian engine",
  "target_kb": "KB-24",
  "physical_store": "KB-20.adverse_reaction_profiles",
  "intake_endpoint": "/api/v1/pipeline/adr-profiles",
  "completeness_grades": {
    "FULL":    { "multiplier": 1.0, "requires": ["E1", "E2", "E3", "E4"], "min_confidence": 0.70 },
    "PARTIAL": { "multiplier": 0.7, "requires": ["E1", "E2_OR_E3"] },
    "STUB":    { "multiplier": 0.0, "requires": ["E1"] }
  },
  "elements": {
    "E1_drug_symptom": {
      "kb20_fields": ["drug_class", "reaction", "reaction_snomed", "frequency", "severity"],
      "required": true,
      "sources": {
        "SPL": { "confidence": 0.95 },
        "PIPELINE": { "confidence": 0.60 },
        "MANUAL_CURATED": { "confidence": 0.90 }
      },
      "merge_rule": "SPL_WINS_OVER_PIPELINE; MANUAL_CURATED_WINS_ALL"
    },
    "E2_mechanism": {
      "kb20_fields": ["mechanism"],
      "required": false,
      "sources": { "MANUAL_CURATED": { "confidence": 0.85 } },
      "merge_rule": "MANUAL_CURATED_ONLY"
    },
    "E3_onset_window": {
      "kb20_fields": ["onset_window", "onset_category"],
      "required": false,
      "sources": {
        "SPL": { "confidence": 0.75 },
        "MANUAL_CURATED": { "confidence": 0.90 }
      },
      "merge_rule": "SPL_PK_DERIVED_AS_BASELINE; MANUAL_CURATED_OVERRIDES"
    },
    "E4_cm_rule": {
      "kb20_fields": ["context_modifier_rule"],
      "required": false,
      "field_type": "JSONB",
      "sources": {
        "PIPELINE": { "confidence": 0.80 },
        "MANUAL_CURATED": { "confidence": 0.85 }
      },
      "merge_rule": "PIPELINE_PROVIDES_CONDITION; MANUAL_CURATED_PROVIDES_DELTA_P"
    }
  },
  "onset_categories": ["IMMEDIATE", "ACUTE", "SUBACUTE", "CHRONIC", "DELAYED", "IDIOSYNCRATIC"],
  "effect_types": ["INCREASE_PRIOR", "DECREASE_PRIOR", "HARD_BLOCK", "OVERRIDE", "SYMPTOM_MODIFICATION"],
  "drug_classes": [
    "ARB", "SGLT2i", "BETA_BLOCKER", "SULFONYLUREA", "INSULIN",
    "ACEi", "CCB", "LOOP_DIURETIC", "THIAZIDE", "METFORMIN", "PDE5i"
  ]
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/pipeline/templates/l3_context_modifier.json
git commit -m "feat(pipeline): add L3 context modifier template definition

Documents the four-element chain schema (drug→symptom, mechanism,
onset_window, CM rule), source coverage matrix, completeness grading,
and valid enums. Contract between pipeline writers and KB-20 ADR service."
```

---

## Chunk 2: KB-20 Go Service Enhancements

**Why:** The Go side needs L3 intake validation (delta_p bounds, source tri-state) and partial CM rule merging so pipeline-sourced conditions can be merged into manual-curated delta_p values without overwriting.

### Task 4: Add L3 Intake Validation to Batch Write Handler

**Problem:** The batch write endpoint at `POST /api/v1/pipeline/adr-profiles` currently accepts any payload. The spec requires validation: E1 fields required, delta_p bounds `(-0.49, 0.49)`, source must be `SPL|PIPELINE|MANUAL_CURATED`, and null delta_p accepted for `HARD_BLOCK|OVERRIDE|SYMPTOM_MODIFICATION` effect types.

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/pipeline_handlers.go` (where `batchWriteADRProfiles` lives)
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/l3_validation.go` (validation logic in its own file)
- Test: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/l3_validation_test.go`

- [ ] **Step 1: Write tests for L3 validation**

Create `l3_validation_test.go`:

```go
package api

import (
	"testing"
)

func TestValidateL3_RejectsMissingE1(t *testing.T) {
	profiles := []map[string]interface{}{
		{"drug_class": "", "reaction": "", "source": "PIPELINE"},
	}
	errors := validateL3Payload(profiles)
	if len(errors) == 0 {
		t.Fatal("expected E1_MISSING error for empty drug_class+reaction")
	}
	if errors[0].Code != "E1_MISSING" {
		t.Errorf("got code %s, want E1_MISSING", errors[0].Code)
	}
}

func TestValidateL3_RejectsDeltaPOutOfRange(t *testing.T) {
	profiles := []map[string]interface{}{
		{
			"drug_class": "ARB",
			"reaction":   "Dizziness",
			"source":     "MANUAL_CURATED",
			"context_modifier_rule": `{"delta_p": 0.55, "target_differential": "OH"}`,
		},
	}
	errors := validateL3Payload(profiles)
	if len(errors) == 0 {
		t.Fatal("expected DELTA_P_OUT_OF_RANGE error for delta_p=0.55")
	}
	if errors[0].Code != "DELTA_P_OUT_OF_RANGE" {
		t.Errorf("got code %s, want DELTA_P_OUT_OF_RANGE", errors[0].Code)
	}
}

func TestValidateL3_AcceptsNullDeltaPForHardBlock(t *testing.T) {
	profiles := []map[string]interface{}{
		{
			"drug_class": "PDE5i",
			"reaction":   "Severe hypotension",
			"source":     "MANUAL_CURATED",
			"context_modifier_rule": `{"effect_type": "HARD_BLOCK", "condition": "med_class==PDE5i"}`,
		},
	}
	errors := validateL3Payload(profiles)
	if len(errors) != 0 {
		t.Errorf("expected no errors for HARD_BLOCK without delta_p, got %d: %v", len(errors), errors)
	}
}

func TestValidateL3_RejectsInvalidSource(t *testing.T) {
	profiles := []map[string]interface{}{
		{
			"drug_class": "ARB",
			"reaction":   "Dizziness",
			"source":     "UNKNOWN_SOURCE",
		},
	}
	errors := validateL3Payload(profiles)
	if len(errors) == 0 {
		t.Fatal("expected INVALID_SOURCE error")
	}
	if errors[0].Code != "INVALID_SOURCE" {
		t.Errorf("got code %s, want INVALID_SOURCE", errors[0].Code)
	}
}
```

- [ ] **Step 2: Implement L3 validation in l3_validation.go**

Create `l3_validation.go` with the `validateL3Payload` function:

```go
package api

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

var validSources = map[string]bool{
	"SPL": true, "PIPELINE": true, "MANUAL_CURATED": true,
}

// nullDeltaPEffectTypes are effect types that accept null/missing delta_p.
var nullDeltaPEffectTypes = map[string]bool{
	"HARD_BLOCK": true, "OVERRIDE": true, "SYMPTOM_MODIFICATION": true,
}

type l3ValidationError struct {
	Index  int    `json:"index"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// validateL3Payload validates a batch of ADR profile payloads against L3 template rules.
func validateL3Payload(profiles []map[string]interface{}) []l3ValidationError {
	var errors []l3ValidationError
	for i, p := range profiles {
		// E1: drug_class and reaction required
		dc, _ := p["drug_class"].(string)
		rx, _ := p["reaction"].(string)
		if dc == "" || rx == "" {
			errors = append(errors, l3ValidationError{i, "E1_MISSING",
				"drug_class and reaction are required for all L3 records"})
			continue
		}

		// Source tri-state
		src, _ := p["source"].(string)
		if !validSources[src] {
			errors = append(errors, l3ValidationError{i, "INVALID_SOURCE",
				"source must be SPL|PIPELINE|MANUAL_CURATED, got: " + src})
		}

		// delta_p bounds check on context_modifier_rule
		cmRuleStr, _ := p["context_modifier_rule"].(string)
		if cmRuleStr != "" && cmRuleStr != "{}" {
			var cmRule map[string]interface{}
			if json.Unmarshal([]byte(cmRuleStr), &cmRule) == nil {
				if dp, ok := cmRule["delta_p"].(float64); ok {
					if math.Abs(dp) >= 0.49 {
						errors = append(errors, l3ValidationError{i, "DELTA_P_OUT_OF_RANGE",
							"delta_p must be in (-0.49, 0.49)"})
					}
				}
				// null delta_p is OK for HARD_BLOCK/OVERRIDE/SYMPTOM_MODIFICATION
				// No validation needed for those cases — absence is the valid state
			}
		}
	}
	return errors
}
```

Then wire validation into `batchWriteADRProfiles` in `pipeline_handlers.go`. The current handler binds directly to `[]models.AdverseReactionProfile`. Change it to bind raw JSON first, validate, then proceed:

```go
func (s *Server) batchWriteADRProfiles(c *gin.Context) {
	// Bind raw JSON for validation before model binding
	var rawPayload []map[string]interface{}
	if err := c.ShouldBindJSON(&rawPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// L3 intake validation
	validationErrors := validateL3Payload(rawPayload)
	if len(validationErrors) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":             "L3 validation failed",
			"validation_errors": validationErrors,
		})
		return
	}

	// Re-marshal and bind to typed models for upsert
	jsonBytes, _ := json.Marshal(rawPayload)
	var profiles []models.AdverseReactionProfile
	if err := json.Unmarshal(jsonBytes, &profiles); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.pipelineService.BatchWriteADRProfiles(profiles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
```

- [ ] **Step 3: Run tests, verify build**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && go build ./... && go test ./internal/api/... -v -run TestValidateL3`
Expected: BUILD SUCCESS. All 4 validation tests PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/l3_validation.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/l3_validation_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/pipeline_handlers.go
git commit -m "feat(kb-20): add L3 intake validation for ADR batch write endpoint

Validates: E1 fields required (drug_class + reaction), source tri-state
(SPL|PIPELINE|MANUAL_CURATED), delta_p bounds (-0.49, 0.49). Null delta_p
accepted for HARD_BLOCK/OVERRIDE/SYMPTOM_MODIFICATION effect types."
```

---

### Task 5: Add mergePartialCMRule to ADR Service

**Problem:** When the pipeline provides a `condition` (e.g., `egfr < 45` from KDIGO extraction) but not a `delta_p`, and a manual record already has `delta_p` without the condition, the merge should combine both without overwriting either. The existing `mergeProfiles()` replaces the entire CM rule — it doesn't do field-level merge within the JSONB.

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/adr_service.go:98-114`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/adr_service_test.go`

- [ ] **Step 1: Write failing test for mergePartialCMRule**

Create `adr_service_test.go`:

```go
package services

import (
	"encoding/json"
	"testing"
)

func TestMergePartialCMRule_PipelineAddsConditionToExistingDeltaP(t *testing.T) {
	existing := `{"target_differential":"OH","delta_p":0.20,"effect_type":"INCREASE_PRIOR"}`
	incoming := `{"condition":"egfr < 45","safety_flag_text_en":"eGFR threshold from KDIGO"}`

	result := mergePartialCMRule(existing, incoming)

	var merged map[string]interface{}
	if err := json.Unmarshal([]byte(result), &merged); err != nil {
		t.Fatalf("failed to parse merged result: %v", err)
	}

	// existing fields preserved
	if merged["delta_p"] != 0.20 {
		t.Errorf("delta_p = %v, want 0.20", merged["delta_p"])
	}
	if merged["target_differential"] != "OH" {
		t.Errorf("target_differential = %v, want OH", merged["target_differential"])
	}

	// incoming field added
	if merged["condition"] != "egfr < 45" {
		t.Errorf("condition = %v, want 'egfr < 45'", merged["condition"])
	}
	if merged["safety_flag_text_en"] != "eGFR threshold from KDIGO" {
		t.Errorf("safety_flag_text_en missing")
	}
}

func TestMergePartialCMRule_NeverOverwritesExistingKeys(t *testing.T) {
	existing := `{"delta_p":0.20,"condition":"med_class==ARB"}`
	incoming := `{"delta_p":0.35,"condition":"egfr < 45"}`

	result := mergePartialCMRule(existing, incoming)

	var merged map[string]interface{}
	json.Unmarshal([]byte(result), &merged)

	// Existing keys must NOT be overwritten
	if merged["delta_p"] != 0.20 {
		t.Errorf("delta_p was overwritten: got %v, want 0.20", merged["delta_p"])
	}
	if merged["condition"] != "med_class==ARB" {
		t.Errorf("condition was overwritten: got %v, want 'med_class==ARB'", merged["condition"])
	}
}

func TestMergePartialCMRule_EmptyExistingGetsIncoming(t *testing.T) {
	existing := `{}`
	incoming := `{"delta_p":0.25,"condition":"med_class==SGLT2i"}`

	result := mergePartialCMRule(existing, incoming)

	var merged map[string]interface{}
	json.Unmarshal([]byte(result), &merged)

	if merged["delta_p"] != 0.25 {
		t.Errorf("delta_p = %v, want 0.25", merged["delta_p"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && go test ./internal/services/ -run TestMergePartialCMRule -v`
Expected: FAIL — `mergePartialCMRule` undefined.

- [ ] **Step 3: Implement mergePartialCMRule**

Add to `adr_service.go` after the existing `mergeProfiles` function:

```go
// mergePartialCMRule merges two JSONB CM rule strings at the field level.
// Incoming (PIPELINE) fields fill gaps in existing (MANUAL) without overwriting.
// This handles the case where pipeline adds a "condition" but the manual record
// already has "delta_p" — both fields are preserved in the merged result.
func mergePartialCMRule(existingJSON, incomingJSON string) string {
	var existing, incoming map[string]interface{}
	if err := json.Unmarshal([]byte(existingJSON), &existing); err != nil {
		return incomingJSON // existing is malformed — use incoming
	}
	if err := json.Unmarshal([]byte(incomingJSON), &incoming); err != nil {
		return existingJSON // incoming is malformed — keep existing
	}

	// Add incoming keys only if they don't exist in existing
	for k, v := range incoming {
		if _, exists := existing[k]; !exists {
			existing[k] = v
		}
	}

	result, err := json.Marshal(existing)
	if err != nil {
		return existingJSON
	}
	return string(result)
}
```

- [ ] **Step 4: Wire mergePartialCMRule into mergeProfiles**

In `mergeProfiles()`, replace lines 99-114 (the SPL↔PIPELINE merge block) to include partial CM rule merging:

```go
	// SPL provides better mechanism and onset data
	if existing.Source == "SPL" && incoming.Source == "PIPELINE" {
		if incoming.ContextModifierRule != "" && incoming.ContextModifierRule != "{}" {
			if existing.ContextModifierRule != "" && existing.ContextModifierRule != "{}" {
				// Partial merge: pipeline adds condition to existing CM rule
				updates["context_modifier_rule"] = mergePartialCMRule(
					existing.ContextModifierRule, incoming.ContextModifierRule)
			} else {
				updates["context_modifier_rule"] = incoming.ContextModifierRule
			}
		}
		if incoming.Confidence.GreaterThan(existing.Confidence) {
			updates["confidence"] = incoming.Confidence
		}
	} else if existing.Source == "PIPELINE" && incoming.Source == "SPL" {
		if incoming.Mechanism != "" {
			updates["mechanism"] = incoming.Mechanism
		}
		if incoming.OnsetWindow != "" {
			updates["onset_window"] = incoming.OnsetWindow
			updates["onset_category"] = incoming.OnsetCategory
		}
	}
```

**IMPORTANT:** Also fix the fallthrough block at lines 124-125. The current code:
```go
	if incoming.ContextModifierRule != "" && existing.ContextModifierRule == "" {
		updates["context_modifier_rule"] = incoming.ContextModifierRule
	}
```
This only fires when existing is empty, so it does NOT violate the no-overwrite invariant — it's safe as-is. However, for consistency, route it through `mergePartialCMRule` (which handles empty existing correctly):
```go
	if incoming.ContextModifierRule != "" && incoming.ContextModifierRule != "{}" {
		if existing.ContextModifierRule == "" || existing.ContextModifierRule == "{}" {
			updates["context_modifier_rule"] = incoming.ContextModifierRule
		} else if _, alreadySet := updates["context_modifier_rule"]; !alreadySet {
			// Fallthrough: both have data but neither SPL nor PIPELINE block handled it
			updates["context_modifier_rule"] = mergePartialCMRule(
				existing.ContextModifierRule, incoming.ContextModifierRule)
		}
	}
```

Also add `"encoding/json"` to the import block if not already present.

- [ ] **Step 5: Run tests, verify pass**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && go test ./internal/services/ -run TestMergePartialCMRule -v`
Expected: PASS — all 3 tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/adr_service.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/adr_service_test.go
git commit -m "feat(kb-20): add mergePartialCMRule for field-level JSONB CM rule merge

When pipeline provides a condition but not delta_p, and existing record
has delta_p, the merge preserves both fields. Never overwrites existing
keys — only fills gaps. Used in SPL→PIPELINE merge path."
```

---

## Chunk 3: Seed Data — 12 Drug Class Records

**Why:** KB-22's CMApplicator needs FULL-grade records to apply drug-aware Bayesian prior shifts. No single automated source produces FULL records — these are clinician-calibrated `MANUAL_CURATED` records matching the spec's Section 8 examples. Includes the insulin+SU combo record (spec Issue 4) and SGLT2i DKA festival fasting condition (spec Issue 3).

### Task 6: Create Seed Data JSON Files

**Problem:** The spec provides 9 drug class reference records (Section 8) plus PDE5i (Section 9). The review identified two missing records: insulin+SU combo and festival fasting for SGLT2i DKA.

**Files:**
- Create: 12 JSON files in `backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/l3_seed_data/`

- [ ] **Step 1: Create seed data directory**

```bash
mkdir -p backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/l3_seed_data
```

- [ ] **Step 2: Create all 12 seed JSON files**

Each file follows the `AdverseReactionProfile` Pydantic schema. The `contextual_modifiers` list contains the four-element chain data, and `context_modifier_rule` is the E4 JSONB. Source is `MANUAL_CURATED` for all seed records.

Create the following files with content from spec Section 8 (drug records 1-9), Section 9 (PDE5i), plus the two additional records identified in the review:

1. `arb_oh.json` — ARB → OH (delta_p +0.20), secondary VOLUME_DEPLETION +0.10
2. `sglt2i_volume.json` — SGLT2i → Volume Depletion (delta_p +0.25), TEMPORAL modifier for fasting risk
3. `sglt2i_dka.json` — SGLT2i → Euglycemic DKA (OVERRIDE), TEMPORAL modifier with `modifier_value: "active fasting period (festival or religious)"` connecting KB-21 Festival Calendar signal
4. `bb_hypo_mask.json` — Beta-blocker → SYMPTOM_MODIFICATION (null delta_p)
5. `su_hypo.json` — Sulfonylurea → Hypoglycemia (delta_p +0.30)
6. `acei_cough.json` — ACEi → Cough/Dyspnea (OVERRIDE, onset_category IDIOSYNCRATIC)
7. `ccb_oedema.json` — CCB → Pedal Oedema (delta_p +0.30)
8. `loop_diuretic_adhf.json` — Loop Diuretic → ADHF (delta_p +0.20)
9. `metformin_lactic.json` — Metformin → Lactic Acidosis (OVERRIDE, onset_category DELAYED)
10. `thiazide_electrolyte.json` — Thiazide → Electrolyte (delta_p +0.10), secondary OH +0.08
11. `pde5i_nitrate.json` — PDE5i → Nitrate contraindication (CONCOMITANT_DRUG modifier, `effect: "absolute contraindication — risk of fatal hypotension"`, `effect_magnitude: MAJOR`). Note: HARD_BLOCK lives in the `context_modifier_rule` JSONB (consumed by KB-22 CMApplicator), NOT in the `ContextModifier.effect` column which only accepts INCREASE_PRIOR/DECREASE_PRIOR. KB-5 DDI also owns this interaction — this seed provides the HPI differential overlay.
12. `insulin_su_combo.json` — Insulin+SU → Severe Hypoglycemia (delta_p +0.35, condition `med_class==INSULIN AND med_class==SULFONYLUREA`)

**Example file — `arb_oh.json`:**

```json
{
  "rxnormCode": "",
  "drugName": "telmisartan",
  "drugClass": "ARB",
  "reaction": "Dizziness, postural hypotension",
  "reactionSnomed": "404640003",
  "mechanism": "AT1 receptor blockade reduces angiotensin II-mediated vasoconstriction → peripheral vasodilation → reduced SVR → postural BP drop on standing. First-dose phenomenon maximal Days 1-7. Telmisartan half-life ~24h; steady-state by Day 4-6.",
  "symptom": "dizziness",
  "onsetWindow": "Days 1-14",
  "onsetCategory": "ACUTE",
  "frequency": "COMMON",
  "severity": "MODERATE",
  "riskFactors": ["elderly", "volume depleted", "concurrent diuretic"],
  "contextualModifiers": [
    {
      "modifierType": "CONCOMITANT_DRUG",
      "modifierValue": "concurrent diuretic increases risk",
      "effect": "increases volume depletion risk",
      "effectMagnitude": "MAJOR",
      "concomitantDrug": "hydrochlorothiazide"
    }
  ],
  "sourceSnippet": "StatPearls ARB; India OH PMC 2022",
  "governance": {
    "sourceAuthority": "StatPearls",
    "sourceDocument": "Angiotensin Receptor Blockers",
    "sourceSection": "Adverse Effects",
    "evidenceLevel": "1A",
    "effectiveDate": "2024-01-01"
  }
}
```

**Example file — `insulin_su_combo.json` (spec Issue 4 — missing record):**

```json
{
  "rxnormCode": "",
  "drugName": "insulin + sulfonylurea combination",
  "drugClass": "INSULIN",
  "reaction": "Severe hypoglycemia from dual insulin secretagogue + exogenous insulin",
  "reactionSnomed": "302866003",
  "mechanism": "Exogenous insulin + sulfonylurea-stimulated endogenous insulin = dual hypoglycemia driver. Combined rate 3-4× higher than either alone. Sulfonylurea blocks ATP-sensitive K+ channels → insulin secretion independent of glucose. Added exogenous insulin overwhelms counter-regulatory capacity.",
  "symptom": "hypoglycemia",
  "onsetWindow": "Hours 2-8 (SU Tmax overlap with basal insulin nadir)",
  "onsetCategory": "IMMEDIATE",
  "frequency": "COMMON",
  "severity": "CRITICAL",
  "riskFactors": ["elderly", "renal impairment", "missed meals", "fasting"],
  "contextualModifiers": [
    {
      "modifierType": "CONCOMITANT_DRUG",
      "modifierValue": "insulin + sulfonylurea combination",
      "effect": "severe hypoglycemia risk 3-4x higher than either alone",
      "effectMagnitude": "MAJOR",
      "concomitantDrug": "sulfonylurea"
    }
  ],
  "sourceSnippet": "ADA Standards 2024; ACCORD Trial; StatPearls Insulin-SU interaction",
  "governance": {
    "sourceAuthority": "ADA",
    "sourceDocument": "Standards of Care in Diabetes 2024",
    "sourceSection": "Pharmacologic Approaches to Glycemic Treatment",
    "evidenceLevel": "1A",
    "effectiveDate": "2024-01-01"
  }
}
```

Write all 12 files following this pattern. Each must have all four elements (drug→symptom, mechanism, onset_window, CM rule in `contextualModifiers`) to grade as FULL.

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/l3_seed_data/
git commit -m "feat(kb-24): add 12 FULL-grade drug class seed records for CMApplicator

9 spec drug classes (ARB, SGLT2i, BB, SU, ACEi, CCB, Loop Diuretic,
Metformin, Thiazide) + PDE5i HARD_BLOCK + SGLT2i DKA + Insulin+SU combo.
SGLT2i DKA condition includes festival_fasting_active for Indian population.
Source: MANUAL_CURATED. All records have complete four-element chains."
```

---

### Task 7: Create Seed Loader Script + Schema Validation Test

**Problem:** The seed data JSON files need a CLI script to load them into KB-20 via the push client, and a test that validates all files against the Pydantic schema before loading.

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/seed_loader.py`
- Create: `backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/l3_seed_data/test_seed_schema.py`

- [ ] **Step 1: Create schema validation test**

```python
"""Validate all L3 seed data JSON files against the Pydantic schema."""

import json
import pathlib
import pytest
from shared.extraction.schemas.kb20_contextual import AdverseReactionProfile

SEED_DIR = pathlib.Path(__file__).parent


def _load_seed_files():
    return sorted(SEED_DIR.glob("*.json"))


@pytest.fixture(params=_load_seed_files(), ids=lambda p: p.stem)
def seed_file(request):
    return request.param


def test_seed_validates_against_schema(seed_file):
    with open(seed_file) as f:
        data = json.load(f)

    # Must parse without error
    profile = AdverseReactionProfile(**data)

    # Must be FULL grade (all four elements present)
    assert profile.completeness_grade == "FULL", (
        f"{seed_file.name}: grade is {profile.completeness_grade}, expected FULL"
    )

    # Must have mechanism (E2) for FULL grade
    assert profile.mechanism, f"{seed_file.name}: missing mechanism (E2)"

    # Must have onset_window (E3) for FULL grade
    assert profile.onset_window, f"{seed_file.name}: missing onset_window (E3)"

    # Must have at least one contextual modifier (E4)
    assert len(profile.contextual_modifiers) > 0, (
        f"{seed_file.name}: missing contextual_modifiers (E4)"
    )

    # Source must be MANUAL_CURATED for seed data
    gov = data.get("governance", {})
    assert gov.get("sourceAuthority"), f"{seed_file.name}: missing sourceAuthority"


def test_all_12_seed_files_present():
    files = _load_seed_files()
    assert len(files) >= 12, f"Expected ≥12 seed files, found {len(files)}"
```

- [ ] **Step 2: Create seed loader CLI script**

```python
#!/usr/bin/env python3
"""
Load L3 seed data into KB-20 via KB20PushClient.

Usage:
    python seed_loader.py                    # dry run (validate only)
    python seed_loader.py --push             # push to KB-20
    python seed_loader.py --push --kb20-url http://localhost:8131
"""

import argparse
import json
import pathlib
import sys

# Add parent packages to path
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[2]))

from shared.extraction.schemas.kb20_contextual import AdverseReactionProfile
from shared.extraction.v4.kb20_push_client import KB20PushClient

SEED_DIR = pathlib.Path(__file__).parent / "l3_seed_data"


def main():
    parser = argparse.ArgumentParser(description="Load L3 seed data into KB-20")
    parser.add_argument("--push", action="store_true", help="Push to KB-20 (default: dry run)")
    parser.add_argument("--kb20-url", default="http://localhost:8131", help="KB-20 base URL")
    args = parser.parse_args()

    seed_files = sorted(SEED_DIR.glob("*.json"))
    if not seed_files:
        print(f"❌ No seed files found in {SEED_DIR}")
        sys.exit(1)

    print(f"📂 Found {len(seed_files)} seed files in {SEED_DIR}")

    # Validate all files first
    valid_profiles = []
    for f in seed_files:
        try:
            with open(f) as fh:
                data = json.load(fh)
            profile = AdverseReactionProfile(**data)
            grade = profile.completeness_grade
            print(f"  ✅ {f.name}: {grade} ({profile.drug_class} → {profile.reaction[:40]})")
            valid_profiles.append((f.name, data))
        except Exception as e:
            print(f"  ❌ {f.name}: {e}")

    print(f"\n📊 Validated: {len(valid_profiles)}/{len(seed_files)} files")

    if not args.push:
        print("\n🔍 Dry run complete. Use --push to write to KB-20.")
        return

    # Push to KB-20
    client = KB20PushClient(base_url=args.kb20_url)
    if not client.health_check():
        print(f"\n❌ KB-20 not reachable at {args.kb20_url}")
        sys.exit(1)

    succeeded = 0
    failed = 0
    for name, data in valid_profiles:
        try:
            resp = client.session.post(
                f"{client.base_url}/api/v1/pipeline/adr-profiles",
                json=[data],
                timeout=client.timeout,
            )
            if resp.status_code < 300:
                succeeded += 1
                print(f"  📤 {name}: OK")
            else:
                failed += 1
                print(f"  ❌ {name}: HTTP {resp.status_code}")
        except Exception as e:
            failed += 1
            print(f"  ❌ {name}: {e}")

    print(f"\n📊 Push complete: {succeeded} OK, {failed} failed")


if __name__ == "__main__":
    main()
```

- [ ] **Step 3: Run schema validation test**

Run: `cd backend/shared-infrastructure/knowledge-base-services && python -m pytest shared/extraction/v4/l3_seed_data/test_seed_schema.py -v`
Expected: All 12 files PASS, all graded FULL.

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/seed_loader.py
git add backend/shared-infrastructure/knowledge-base-services/shared/extraction/v4/l3_seed_data/test_seed_schema.py
git commit -m "feat(kb-24): add seed loader CLI and schema validation tests

seed_loader.py: validates all L3 seed JSON against Pydantic schema,
optionally pushes to KB-20 via batch API with --push flag.
test_seed_schema.py: pytest suite ensuring all 12 files parse as FULL
grade with complete four-element chains."
```

---

## Dependency Graph

```
Task 1 (IDIOSYNCRATIC enum) ──→ Task 6 (seed data — ACEi uses IDIOSYNCRATIC)
Task 2 (pipeline push wiring) ── Independent
Task 3 (L3 template JSON)    ── Independent
Task 4 (L3 validation)       ── Independent of Tasks 1-3
Task 5 (mergePartialCMRule)   ── Independent of Tasks 1-4
Task 6 (seed data files)     ──→ depends on Task 1 (onset enum)
Task 7 (seed loader + tests) ──→ depends on Task 6 (seed files exist)
```

**Parallelization opportunities:**
- Tasks 1, 2, 3, 4, 5 can all run in parallel (independent changes)
- Task 6 must wait for Task 1 (IDIOSYNCRATIC enum needed)
- Task 7 must wait for Task 6 (validates seed files)

---

## Spec Issues Addressed in This Plan

| Issue # | From Review | How Addressed |
|---------|-------------|---------------|
| 1 | Port 8124 conflict | NOT assigned — KB-24 uses KB-20's endpoint via `KB24_ENDPOINT` env var. No phantom port entry. |
| 2 | `mergePartialCMRule()` silent data loss | Task 5 — implemented with never-overwrite semantics. MANUAL_CURATED full-replace handled by existing `Upsert()` E4 branch. |
| 3 | SGLT2i DKA condition too narrow | Task 6 — `sglt2i_dka.json` uses `modifier_type: TEMPORAL` with `modifier_value: "active fasting period (festival or religious)"` connecting KB-21 Festival Calendar signal. |
| 4 | Missing insulin+SU combo record | Task 6 — `insulin_su_combo.json` with delta_p +0.35 for dual secretagogue risk. |
| 5 | Completeness tracking inconsistency | Seed data creates 12 records (not 11) — clarified in Task 6 step text. |
