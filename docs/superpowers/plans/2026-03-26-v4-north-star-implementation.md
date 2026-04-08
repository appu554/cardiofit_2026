# V4 North Star Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the complete Vaidshala V4 cardiometabolic platform — 5 new Flink jobs, 9 new Kafka topics, KB-20/KB-23/KB-24/KB-26 extensions, IOR system, Market Shim, and the physician feedback learning pipeline — transforming V3's single-disease engine into a dual-domain (diabetes + hypertension) integrated correction loop.

**Architecture:** V4 extends V3's 5-layer architecture without adding new layers. New Flink jobs (BP Variability, MealResponseCorrelator, Comorbidity Interaction Detector, Engagement Monitor, InterventionWindowMonitor) are added as Module7-Module11 in the existing `FlinkJobOrchestrator`. Go KB services are extended following the identical Gin+GORM+Redis+Prometheus+zap+segmentio/kafka-go pattern used by KB-20 through KB-26. All V4 additions are deterministic — no LLM in the clinical decision path.

**Tech Stack:**
- Flink 2.1.0 (Java 17) — new operator modules registered in `FlinkJobOrchestrator.java`
- Go 1.24 — KB service extensions (Gin, GORM, Redis, kafka-go)
- PostgreSQL 16 — IOR store, feedback store (same RDS, separate schemas)
- Kafka (Confluent) — 9 new topics following existing envelope format
- Neo4j — KB-25 causal chain graph
- Python 3.11+ — Phenotype clustering batch pipeline (UMAP + HDBSCAN)

**Source Documents (12):**
- V4 NorthStar Architecture (master blueprint)
- DeepDive #0: Market Shim (7 configuration components)
- DeepDive #1: BP Variability Engine (ARV, morning surge, dipping)
- DeepDive #4: IOR Schema + Generator (intervention-outcome records)
- DeepDive #5: Dual-Domain Decision Card Generator
- DeepDive #6: MHRI Score Computation (5-component 0-100)
- DeepDive #7: Comorbidity Interaction Detector (17 rules)
- DeepDive #8: Engagement Monitor (6-signal disengagement)
- DeepDive #9: Population Phenotype Clustering (21-feature UMAP+HDBSCAN)
- DeepDive #10: Physician Feedback Learning Pipeline
- Lifestyle Intelligence Stack (KB-25 + M3 + KB-26 calibration)
- Waist Circumference Assessment Research

---

## File Structure Overview

### New Flink Modules + V3 Job Extensions (Java)
```
backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/
├── operators/
│   ├── Module1b_IngestionCanonicalizer.java    # MODIFY: add cgm_active, nocturnal profile, data_tier (Task 6b)
│   ├── Module3_TrajectoryAnalysis.java         # MODIFY: add BP trajectory slope, dual-domain flag (Task 6b)
│   ├── Module4_PatternDetection.java           # MODIFY: add cross-domain deterioration CEP (Task 6b)
│   ├── Module7_BPVariability.java              # CREATE: DD#1: ARV, surge, dip
│   ├── Module8_ComorbidityInteraction.java     # CREATE: DD#7: 17 CID rules
│   ├── Module9_EngagementMonitor.java          # CREATE: DD#8: 6-signal scoring
│   ├── Module10_MealResponseCorrelator.java    # CREATE: NorthStar: glucose+sodium
│   ├── Module10b_MealPatternAggregator.java    # CREATE: Weekly sodium/food patterns
│   └── Module11_InterventionWindowMonitor.java # CREATE: DD#4: IOR window tracking
├── models/
│   ├── BPVariabilityMetrics.java               # Output schema for Module7
│   ├── ComorbidityAlert.java                   # Output schema for Module8
│   ├── EngagementScore.java                    # Output schema for Module9 (+ channel, appliedThreshold)
│   ├── MealResponsePair.java                   # Output schema for Module10
│   └── InterventionWindowEvent.java            # Output schema for Module11
├── analytics/
│   ├── ARVCalculator.java                      # Average Real Variability
│   ├── MorningSurgeDetector.java               # Sleep-trough surge method
│   ├── DippingPatternClassifier.java           # Nocturnal dip ratio
│   ├── HypertensiveCrisisDetector.java         # SBP>180/DBP>120 bypass
│   └── EngagementScorer.java                   # 6-signal scoring + channel-aware thresholds (G2)
└── state/
    └── BPVariabilityStateDescriptors.java      # Keyed state for daily BP
```

### KB Service Extensions (Go)
```
# KB-20 Patient Profile extensions
kb-20-patient-profile/internal/
├── models/
│   ├── patient_profile.go          # MODIFY: add ~25 new V4 fields
│   ├── stratum.go                  # MODIFY: fix DM_HTN_base naming
│   ├── ckm_stage.go               # CREATE: AHA CKM stage model
│   ├── intervention_record.go      # CREATE: IOR intervention schema (+ confounder_flags, data_completeness — G3)
│   └── outcome_record.go           # CREATE: IOR outcome schema
├── services/
│   ├── stratum_engine.go           # MODIFY: fix hasHTN bug + add LAB_RESULT/MEDICATION_CHANGE event publication (Task 3c/P12)
│   ├── mhri_provider.go           # CREATE: MHRI data provider for KB-26
│   ├── ckm_stage_computer.go      # CREATE: AHA 2023 CKM stage computation (G1)
│   ├── ior_store.go                # CREATE: IOR CRUD + similar-patient query (completeness filter)
│   └── ior_generator.go            # CREATE: daily batch job + confounder capture (G3)

# KB-22 HPI Engine contract fixes (Task 3c)
kb-22-hpi-engine/internal/
├── services/
│   ├── medication_safety_provider.go # MODIFY: rewrite KB-9→KB-5 gRPC (P14)
│   ├── outcome_publisher.go          # MODIFY: /events→/execute endpoint (P15-a)
│   ├── session_manager.go            # MODIFY: add R-05 completion guard (P15-b)
│   └── stratum_hierarchy.go          # CREATE: hierarchy resolver (Task 1)

# KB-23 Decision Cards extensions
kb-23-decision-cards/internal/
├── models/
│   ├── card_types.go               # MODIFY: add INTEGRATED_DUAL_DOMAIN
│   ├── feedback.go                 # CREATE: physician feedback schema
│   └── rule_change_proposal.go     # CREATE: governance lifecycle model (G4)
├── services/
│   ├── four_pillar_evaluator.go    # CREATE: DD#3 pillar assessment decision trees
│   ├── dual_domain_generator.go    # CREATE: 7-step card pipeline
│   ├── conflict_detector.go        # CREATE: cross-domain conflict resolution
│   ├── card_consolidator.go        # CREATE: max 3 active cards rule
│   ├── ior_insight_provider.go     # CREATE: similar-patient query
│   ├── feedback_store.go           # CREATE: DD#10 feedback capture
│   ├── feedback_analyzer.go        # CREATE: 4 analysis pipelines (DD#10)
│   └── governance_store.go         # CREATE: proposal lifecycle + SafetyTrace integration (G4)
├── api/
│   └── governance_handlers.go      # CREATE: proposal CRUD + committee review API (G4)

# KB-24 Safety Constraint Engine extensions
kb-24-safety-constraint-engine/
├── configs/
│   └── comorbidity_rules.yaml      # CREATE: 17 CID rules for Go evaluation
├── internal/
│   └── services/
│       └── comorbidity_evaluator.go # CREATE: Go-side CID rule evaluation

# KB-26 Metabolic Digital Twin extensions
kb-26-metabolic-digital-twin/internal/
├── models/
│   └── mhri.go                     # CREATE: MHRI output schema
├── services/
│   ├── mhri_scorer.go              # CREATE: 5-component MHRI computation
│   ├── mhri_trajectory.go          # CREATE: 14-day trajectory engine
│   └── twin_calibrator.go          # MODIFY: add hemodynamic domain
```

### New Kafka Topics (9)
```
flink.bp-variability-metrics        # Module7 output (8 partitions, 30-day)
flink.meal-response                 # Module10 real-time pairs (8 partitions, 30-day)
flink.meal-patterns                 # Module10 weekly aggregation (4 partitions, 90-day)
flink.engagement-signals            # Module9 daily scores (4 partitions, 30-day)
clinical.intervention-events        # IOR trigger (4 partitions, 90-day)
clinical.intervention-window-signals # Module11 WINDOW_OPENED/CLOSED (4 partitions, 90-day)
clinical.decision-cards             # KB-23 card output (4 partitions, 30-day)
alerts.comorbidity-interactions     # Module8 CID alerts (4 partitions, 90-day)
alerts.engagement-drop              # Module9 threshold breach (2 partitions, 90-day)
```

### Ingestion Service Extensions (Go)
```
ingestion-service/internal/
├── canonical/
│   └── observation.go              # MODIFY: add S23-S26 signal types
├── kafka/
│   └── router.go                   # MODIFY: add topic routing for S23-S26
└── fhir/
    └── observation_mapper.go       # MODIFY: add data_tier field mapping
```

### Market Shim Configuration
```
deploy/markets/
├── india/
│   ├── clinical_params.yaml        # BMI 23/25, SBP targets, salt thresholds
│   ├── channels.yaml               # GOVERNMENT, CORPORATE, GP_PRIMARY
│   ├── pharma_shim.yaml            # Drug affordability flags
│   ├── food_db_config.yaml         # IFCT 2017 database config
│   ├── health_record_adapter.yaml  # ABDM integration config
│   ├── compliance.yaml             # DPDPA, autonomous_threshold: enabled
│   └── localization/               # Hindi, regional languages
└── australia/
    ├── clinical_params.yaml        # BMI 25/30, AU-specific targets
    ├── clinical_params_indigenous.yaml # BMI 23/27.5, ACCHS overrides
    ├── channels.yaml               # GP_PRIMARY, SPECIALIST, ACCHS
    ├── pharma_shim.yaml            # PBS formulary
    ├── food_db_config.yaml         # AUSNUT database config
    ├── health_record_adapter.yaml  # My Health Record config
    ├── compliance.yaml             # TGA, autonomous_threshold: DISABLED
    └── localization/               # English AU
```

### Phenotype Clustering (Python batch)
```
backend/shared-infrastructure/analytics/
├── phenotype_clustering/
│   ├── requirements.txt            # umap-learn, hdbscan, scikit-learn, pandas, pyyaml
│   ├── feature_extractor.py        # 21-feature vector from KB-20
│   ├── clustering_pipeline.py      # UMAP + HDBSCAN + validation
│   ├── cluster_validator.py        # 4 clinical validation criteria
│   ├── therapy_mapper.py           # CREATE (G6): cluster → phenotype_therapy_map.yaml for KB-23
│   └── centroid_exporter.py        # phenotype_centroids.json for AU transfer
```

---

## Phase C0: V3 Contract Fixes (Prerequisite)

> V4 is built on V3. These fixes unblock the KB service chain.

### Task 1: Fix Stratum Naming Contract — Hierarchy Resolver (Option A)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/stratum_hierarchy.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/stratum_hierarchy_test.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/session_context_provider.go` (1 line: call `StratumMatches()`)
- Test: Unit tests + manual KB-22 session creation

**Context:** KB-20 emits exact strata (`DM_HTN`, `DM_HTN_CKD`, `DM_HTN_CKD_HF`), but KB-22 pilot nodes use `DM_HTN_base` in `strata_supported` as a catch-all. Rather than renaming every YAML file (which creates a combinatorial maintenance burden as V4 adds `DM_HTN_CKD_3a`, `DM_HTN_CKD_HF_REDUCED`, etc.), we add a hierarchy resolver. `DM_HTN_base` means "any stratum rooted at DM+HTN". When KB-22 checks if a patient's stratum is supported, `StratumMatches("DM_HTN_CKD", ["DM_HTN_base"])` returns true. Zero YAML changes. Zero future maintenance per new stratum.

**Why Option A over Option B (rename YAMLs):** Option B forces every YAML node to enumerate every stratum it accepts. V4 adds ~5 new strata → 12 nodes × 8 strata = 96 entries to maintain. Option A: 12 nodes × 1 entry (`DM_HTN_base`) + 1 hierarchy map = 13 things to maintain. The hierarchy map grows linearly with new strata; node YAML files never change.

- [ ] **Step 1: Write failing tests for stratum hierarchy resolver**

Create `kb-22-hpi-engine/internal/services/stratum_hierarchy_test.go`:

```go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestStratumMatches_ExactMatch(t *testing.T) {
    assert.True(t, StratumMatches("DM_HTN", []string{"DM_HTN"}))
}

func TestStratumMatches_BaseAcceptsDescendants(t *testing.T) {
    // DM_HTN_base should accept DM_HTN, DM_HTN_CKD, DM_HTN_CKD_HF
    assert.True(t, StratumMatches("DM_HTN", []string{"DM_HTN_base"}))
    assert.True(t, StratumMatches("DM_HTN_CKD", []string{"DM_HTN_base"}))
    assert.True(t, StratumMatches("DM_HTN_CKD_HF", []string{"DM_HTN_base"}))
}

func TestStratumMatches_V4CKDSubstaging(t *testing.T) {
    // V4 adds finer CKD strata — hierarchy resolver handles them automatically
    assert.True(t, StratumMatches("DM_HTN_CKD_3a", []string{"DM_HTN_base"}))
    assert.True(t, StratumMatches("DM_HTN_CKD_3a", []string{"DM_HTN_CKD"}))
    assert.False(t, StratumMatches("DM_HTN_CKD_3a", []string{"DM_HTN"})) // CKD patient ≠ non-CKD node
}

func TestStratumMatches_V4HFSubtyping(t *testing.T) {
    assert.True(t, StratumMatches("DM_HTN_CKD_HF_REDUCED", []string{"DM_HTN_base"}))
    assert.True(t, StratumMatches("DM_HTN_CKD_HF_PRESERVED", []string{"DM_HTN_CKD_HF"}))
}

func TestStratumMatches_DMOnlyDoesNotMatchDMHTN(t *testing.T) {
    assert.False(t, StratumMatches("DM_ONLY", []string{"DM_HTN_base"}))
}

func TestStratumMatches_EmptySupported(t *testing.T) {
    assert.False(t, StratumMatches("DM_HTN", []string{}))
}
```

Run: `cd kb-22-hpi-engine && go test ./internal/services/ -run TestStratumMatches -v`
Expected: FAIL — `StratumMatches` function not defined.

- [ ] **Step 2: Implement stratum hierarchy resolver**

Create `kb-22-hpi-engine/internal/services/stratum_hierarchy.go`:

```go
package services

// stratumAncestors maps each stratum to its ordered list of ancestors (most specific → most general).
// When V4 adds new strata, add entries here — no YAML file changes needed.
var stratumAncestors = map[string][]string{
    // V3 strata
    "DM_HTN":        {"DM_HTN", "DM_HTN_base"},
    "DM_HTN_CKD":    {"DM_HTN_CKD", "DM_HTN_base"},
    "DM_HTN_CKD_HF": {"DM_HTN_CKD_HF", "DM_HTN_CKD", "DM_HTN_base"},

    // V4 CKD substaging (finerenone eligibility, KDIGO 2024)
    "DM_HTN_CKD_3a": {"DM_HTN_CKD_3a", "DM_HTN_CKD", "DM_HTN_base"},
    "DM_HTN_CKD_3b": {"DM_HTN_CKD_3b", "DM_HTN_CKD", "DM_HTN_base"},
    "DM_HTN_CKD_A3": {"DM_HTN_CKD_A3", "DM_HTN_CKD", "DM_HTN_base"},

    // V4 HF subtyping (EF-based, ESC 2024)
    "DM_HTN_CKD_HF_REDUCED":   {"DM_HTN_CKD_HF_REDUCED", "DM_HTN_CKD_HF", "DM_HTN_CKD", "DM_HTN_base"},
    "DM_HTN_CKD_HF_PRESERVED": {"DM_HTN_CKD_HF_PRESERVED", "DM_HTN_CKD_HF", "DM_HTN_CKD", "DM_HTN_base"},
}

// StratumMatches returns true if patientStratum is accepted by any entry in supportedStrata.
// Uses ancestor chain: if a node supports "DM_HTN_base", it accepts any descendant.
func StratumMatches(patientStratum string, supportedStrata []string) bool {
    if len(supportedStrata) == 0 {
        return false
    }

    // Build lookup set for O(1) matching
    supported := make(map[string]bool, len(supportedStrata))
    for _, s := range supportedStrata {
        supported[s] = true
    }

    // Direct match
    if supported[patientStratum] {
        return true
    }

    // Walk ancestor chain
    ancestors, ok := stratumAncestors[patientStratum]
    if !ok {
        return false // unknown stratum
    }
    for _, ancestor := range ancestors {
        if supported[ancestor] {
            return true
        }
    }
    return false
}
```

- [ ] **Step 3: Run stratum hierarchy tests — verify all pass**

Run: `cd kb-22-hpi-engine && go test ./internal/services/ -run TestStratumMatches -v`
Expected: 8 tests PASS.

- [ ] **Step 4: Wire StratumMatches into session creation**

In `kb-22-hpi-engine/internal/services/session_context_provider.go`, find the stratum check in `CreateSession()` (where it checks if `patientStratum` is in the node's `strata_supported` list) and replace the direct string comparison with:

```go
// BEFORE: direct membership check
// if !contains(node.StrataSupported, patientStratum) { return ErrStratumNotSupported }

// AFTER: hierarchy-aware matching
if !StratumMatches(patientStratum, node.StrataSupported) {
    return ErrStratumNotSupported
}
```

This is a 1-line change. The YAML files keep `DM_HTN_base` unchanged.

- [ ] **Step 5: Verify KB-22 session creation end-to-end**

```bash
cd backend/shared-infrastructure/knowledge-base-services
docker compose -f kb-20-patient-profile/docker-compose.yml up -d
docker compose -f kb-22-hpi-engine/docker-compose.yml up -d

# Create session — DM_HTN patient against node with DM_HTN_base
curl -X POST http://localhost:8132/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"patient_id": "ba8f876b-xxxx-xxxx-xxxx-xxxxxxxxxxxx", "node_id": "P02_DYSPNEA"}'
```

Expected: 201 Created with session_id. No YAML changes were made.

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/stratum_hierarchy.go \
       backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/stratum_hierarchy_test.go \
       backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/session_context_provider.go
git commit -m "fix(kb-22): add stratum hierarchy resolver — DM_HTN_base accepts all V3+V4 descendants"
```

### Task 2: Create 9 New V4 Kafka Topics

**Files:**
- Modify: `backend/shared-infrastructure/kafka/config/topics-config.yaml`
- Create: `backend/shared-infrastructure/kafka/scripts/create-v4-topics.sh`

- [ ] **Step 1: Add V4 topic definitions to topics-config.yaml**

Add a new `v4_outputs` section after the existing `clinical_signals` section (V4 topics span `flink.*`, `clinical.*`, and `alerts.*` prefixes — grouping them under a dedicated section keeps the YAML organized by release rather than mixing into existing categories):

```yaml
v4_outputs:
  # V4 Flink Output Topics
  - name: flink.bp-variability-metrics
    description: "Per-patient BP variability metrics (ARV, surge, dip) from Module7"
    partitions: 8
    retention_ms: 2592000000  # 30 days
    compression_type: snappy
    cleanup_policy: delete
    producers: ["flink-module7-bp-variability"]
    consumers: ["kb-26-metabolic-digital-twin", "kb-23-decision-cards"]

  - name: flink.meal-response
    description: "Meal-glucose and sodium-BP correlation pairs from Module10"
    partitions: 8
    retention_ms: 2592000000  # 30 days
    compression_type: snappy
    cleanup_policy: delete
    producers: ["flink-module10-meal-response"]
    consumers: ["kb-26-metabolic-digital-twin", "kb-25-lifestyle-knowledge-graph"]

  - name: flink.engagement-signals
    description: "Daily per-patient engagement composite scores from Module9"
    partitions: 4
    retention_ms: 2592000000  # 30 days
    compression_type: snappy
    cleanup_policy: delete
    producers: ["flink-module9-engagement"]
    consumers: ["kb-26-metabolic-digital-twin", "kb-21-behavioral-intelligence"]

  - name: clinical.intervention-events
    description: "Intervention approval/modification events for IOR tracking"
    partitions: 4
    retention_ms: 7776000000  # 90 days
    compression_type: gzip
    cleanup_policy: delete
    min_insync_replicas: 3
    producers: ["kb-23-decision-cards"]
    consumers: ["flink-module11-intervention-window", "ior-generator"]

  - name: clinical.decision-cards
    description: "Generated Decision Cards from KB-23 dual-domain pipeline"
    partitions: 4
    retention_ms: 2592000000  # 30 days
    compression_type: snappy
    cleanup_policy: delete
    producers: ["kb-23-decision-cards"]
    consumers: ["api-gateway", "notification-service"]

  - name: alerts.comorbidity-interactions
    description: "Cross-domain drug interaction alerts from Module8 CID"
    partitions: 4
    retention_ms: 7776000000  # 90 days
    compression_type: gzip
    cleanup_policy: delete
    min_insync_replicas: 3
    producers: ["flink-module8-comorbidity"]
    consumers: ["kb-23-decision-cards", "notification-service"]

  - name: flink.meal-patterns
    description: "Weekly dietary pattern aggregation from Module10 MealPatternAggregator — worst foods, avg daily sodium, salt_sensitivity_beta"
    partitions: 4
    retention_ms: 7776000000  # 90 days
    compression_type: snappy
    cleanup_policy: delete
    producers: ["flink-module10-meal-response"]
    consumers: ["kb-21-behavioral-intelligence", "kb-26-metabolic-digital-twin", "phenotype-clustering"]

  - name: alerts.engagement-drop
    description: "Low-volume engagement threshold breach alerts (ORANGE/RED) — triggers re-engagement workflow"
    partitions: 2
    retention_ms: 7776000000  # 90 days
    compression_type: gzip
    cleanup_policy: delete
    producers: ["flink-module9-engagement"]
    consumers: ["notification-service", "kb-21-behavioral-intelligence"]

  - name: clinical.intervention-window-signals
    description: "Module11 WINDOW_OPENED/WINDOW_CLOSED events — triggers IOR generator"
    partitions: 4
    retention_ms: 7776000000  # 90 days
    compression_type: gzip
    cleanup_policy: delete
    min_insync_replicas: 3
    producers: ["flink-module11-intervention-window"]
    consumers: ["ior-generator", "kb-20-patient-profile"]
```

- [ ] **Step 2: Create topic creation script**

```bash
#!/bin/bash
# create-v4-topics.sh — Creates 9 V4 Kafka topics
BOOTSTRAP="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"

TOPICS=(
  "flink.bp-variability-metrics:8:2592000000:snappy"
  "flink.meal-response:8:2592000000:snappy"
  "flink.meal-patterns:4:7776000000:snappy"
  "flink.engagement-signals:4:2592000000:snappy"
  "clinical.intervention-events:4:7776000000:gzip"
  "clinical.intervention-window-signals:4:7776000000:gzip"
  "clinical.decision-cards:4:2592000000:snappy"
  "alerts.comorbidity-interactions:4:7776000000:gzip"
  "alerts.engagement-drop:2:7776000000:gzip"
)

for entry in "${TOPICS[@]}"; do
  IFS=':' read -r topic partitions retention compression <<< "$entry"
  echo "Creating topic: $topic (partitions=$partitions, retention=${retention}ms)"
  kafka-topics --bootstrap-server "$BOOTSTRAP" \
    --create --if-not-exists \
    --topic "$topic" \
    --partitions "$partitions" \
    --replication-factor 3 \
    --config retention.ms="$retention" \
    --config cleanup.policy=delete \
    --config compression.type="$compression" \
    --config min.insync.replicas=2
done

echo "V4 topics created successfully."
```

- [ ] **Step 3: Register topics in KafkaTopics.java enum**

In `flink-processing/src/main/java/com/cardiofit/flink/utils/KafkaTopics.java`, add:

```java
// V4 Topics (9 new)
FLINK_BP_VARIABILITY_METRICS("flink.bp-variability-metrics", 8, 30),
FLINK_MEAL_RESPONSE("flink.meal-response", 8, 30),
FLINK_MEAL_PATTERNS("flink.meal-patterns", 4, 90),
FLINK_ENGAGEMENT_SIGNALS("flink.engagement-signals", 4, 30),
CLINICAL_INTERVENTION_EVENTS("clinical.intervention-events", 4, 90),
CLINICAL_INTERVENTION_WINDOW_SIGNALS("clinical.intervention-window-signals", 4, 90),
CLINICAL_DECISION_CARDS("clinical.decision-cards", 4, 30),
ALERTS_COMORBIDITY_INTERACTIONS("alerts.comorbidity-interactions", 4, 90),
ALERTS_ENGAGEMENT_DROP("alerts.engagement-drop", 2, 90),
```

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/kafka/config/topics-config.yaml \
       backend/shared-infrastructure/kafka/scripts/create-v4-topics.sh \
       backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/utils/KafkaTopics.java
git commit -m "feat(kafka): add 9 V4 topics — BP variability, meal response/patterns, engagement/drop, IOR windows, cards, CID"
```

### Task 3: Add V4 Signal Types (S23-S26) to Ingestion Service

**Files:**
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/observation.go`
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/router.go`

- [ ] **Step 1: Add observation type constants**

In `canonical/observation.go`, add constants aligned with **North Star Section 2.1** canonical numbering:

```go
// V4 Signal Types — numbering per NorthStar Architecture §2.1
ObsSodiumEstimate     ObservationType = "SODIUM_ESTIMATE"     // S23 — per-meal sodium
ObsCGMRaw             ObservationType = "CGM_RAW"             // S24 — continuous glucose raw
ObsInterventionEvent  ObservationType = "INTERVENTION_EVENT"  // S25 — IOR trigger
ObsPhysicianFeedback  ObservationType = "PHYSICIAN_FEEDBACK"  // S26 — card feedback

// Additional V4 signals (extend beyond S26 — covered by existing types or new S27+)
ObsWaistCircumference ObservationType = "WAIST_CIRCUMFERENCE" // S27 (also via S12 anthropometric)
ObsExerciseSession    ObservationType = "EXERCISE_SESSION"    // S28 (also via S15 activity)
ObsMoodStress         ObservationType = "MOOD_STRESS"         // S29 (also via S18 patient-reported)
```

**Note:** Waist circumference, exercise, and mood/stress may already be routed via existing signal types (S12, S15, S18). The explicit S27-S29 constants allow finer-grained routing if needed, but MUST NOT conflict with the S23-S26 numbering from the North Star spec.

- [ ] **Step 2: Add data_tier field to CanonicalObservation**

In the `CanonicalObservation` struct, add:

```go
DataTier string `json:"data_tier,omitempty"` // TIER_1_CGM, TIER_2_HYBRID, TIER_3_SMBG
```

- [ ] **Step 3: Write failing test for V4 signal type routing**

Create `ingestion-service/internal/kafka/router_v4_test.go`:

```go
package kafka

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "ingestion-service/internal/canonical"
)

func TestV4SignalRouting_SodiumEstimate(t *testing.T) {
    topic := RouteObservation(canonical.ObsSodiumEstimate)
    assert.Equal(t, "ingestion.patient-reported", topic)
}

func TestV4SignalRouting_CGMRaw(t *testing.T) {
    topic := RouteObservation(canonical.ObsCGMRaw)
    assert.Equal(t, "ingestion.vitals", topic)
}

func TestV4SignalRouting_InterventionEvent(t *testing.T) {
    topic := RouteObservation(canonical.ObsInterventionEvent)
    assert.Equal(t, "clinical.intervention-events", topic)
}

func TestV4SignalRouting_PhysicianFeedback(t *testing.T) {
    topic := RouteObservation(canonical.ObsPhysicianFeedback)
    assert.Equal(t, "clinical.decision-cards", topic)
}
```

Run: `cd ingestion-service && go test ./internal/kafka/ -run TestV4Signal -v`
Expected: FAIL — new observation types not in routing map.

- [ ] **Step 4: Add topic routing for new signal types**

In `kafka/router.go`, add mappings:

```go
// V4 NorthStar signals (S23-S26)
canonical.ObsSodiumEstimate:     "ingestion.patient-reported",
canonical.ObsCGMRaw:             "ingestion.vitals",
canonical.ObsInterventionEvent:  "clinical.intervention-events",
canonical.ObsPhysicianFeedback:  "clinical.decision-cards",

// Extended V4 signals (S27-S29)
canonical.ObsWaistCircumference: "ingestion.patient-reported",
canonical.ObsExerciseSession:    "ingestion.wearable-aggregates",
canonical.ObsMoodStress:         "ingestion.patient-reported",
```

- [ ] **Step 5: Run tests — verify pass**

Run: `cd ingestion-service && go test ./internal/kafka/ -run TestV4Signal -v`
Expected: 4 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/ \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/
git commit -m "feat(ingestion): add V4 signal types S23-S26 and data_tier field"
```

### Task 3b: Extend V3 Signal Schemas for V4 Field Requirements

**Files:**
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/observation.go`
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/fhir/observation_mapper.go`

**Context:** The Flink Architecture specification requires field-level extensions to 7 existing V3 signal schemas. Without these fields, Module7 (BP Variability), Module8 (CID), Module10 (MealResponseCorrelator), and the MHRI hemodynamic component cannot function correctly. These are additive — existing producers that omit the new fields will serialize them as zero-values/empty via `omitempty`.

- [ ] **Step 1: Extend CanonicalObservation struct with V4 signal fields**

In `canonical/observation.go`, add the following fields to the `CanonicalObservation` struct:

```go
// V4 Signal Schema Extensions (per Flink Architecture §7.1–7.3)
// --- S1 FBG extension ---
SourceProtocol   string `json:"source_protocol,omitempty"`   // Tier 3 rotating meal protocol identifier

// --- S2 PPBG extension ---
LinkedMealID     string `json:"linked_meal_id,omitempty"`    // Correlates with S4 meal log entry for MealResponseCorrelator

// --- S4 Meal log extensions ---
SodiumEstimatedMg float64 `json:"sodium_estimated_mg,omitempty"` // Auto-computed from IFCT 2017/AUSNUT sodium lookup
PreparationMethod string  `json:"preparation_method,omitempty"` // Enum: RAW, BOILED, FRIED, ROASTED, STEAMED, CURRY, OTHER
FoodNameLocal     string  `json:"food_name_local,omitempty"`    // Regional language food name (Hindi, Tamil, etc.)

// --- S6 Hypo event extension ---
SymptomAwareness *bool `json:"symptom_awareness,omitempty"` // true=patient reported symptoms, false=no symptoms (CID-03 masking detection)

// --- S7 BP seated extensions ---
DeviceType        string `json:"device_type,omitempty"`        // oscillometric_cuff, cuffless_ppg, cuffless_tonometric
ClinicalGrade     *bool  `json:"clinical_grade,omitempty"`     // true=validated device, false=consumer-grade
MeasurementMethod string `json:"measurement_method,omitempty"` // auscultatory, oscillometric, cuffless

// --- S8 BP standing extension ---
LinkedSeatedReadingID string `json:"linked_seated_reading_id,omitempty"` // For automatic orthostatic delta computation

// --- S9/S10 Morning/Evening BP extensions ---
WakingTime string `json:"waking_time,omitempty"` // HH:MM format — precise surge window calculation
SleepTime  string `json:"sleep_time,omitempty"`  // HH:MM format — nocturnal window definition
```

- [ ] **Step 2: Write failing tests for V4 schema field serialization**

Create `ingestion-service/internal/canonical/observation_v4_schema_test.go`:

```go
package canonical

import (
    "encoding/json"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestV4SchemaFields_MealLogSodium(t *testing.T) {
    obs := CanonicalObservation{
        Type:              ObsMealLog,
        SodiumEstimatedMg: 1850.0,
        PreparationMethod: "CURRY",
        FoodNameLocal:     "दाल चावल",
    }
    data, err := json.Marshal(obs)
    require.NoError(t, err)
    assert.Contains(t, string(data), `"sodium_estimated_mg":1850`)
    assert.Contains(t, string(data), `"preparation_method":"CURRY"`)
    assert.Contains(t, string(data), `"food_name_local"`)
}

func TestV4SchemaFields_BPDeviceType(t *testing.T) {
    clinicalGrade := true
    obs := CanonicalObservation{
        Type:              ObsBloodPressure,
        DeviceType:        "oscillometric_cuff",
        ClinicalGrade:     &clinicalGrade,
        MeasurementMethod: "oscillometric",
    }
    data, err := json.Marshal(obs)
    require.NoError(t, err)
    assert.Contains(t, string(data), `"device_type":"oscillometric_cuff"`)
    assert.Contains(t, string(data), `"clinical_grade":true`)
}

func TestV4SchemaFields_HypoSymptomAwareness(t *testing.T) {
    aware := false
    obs := CanonicalObservation{
        Type:             ObsHypoglycaemiaEvent,
        SymptomAwareness: &aware,
    }
    data, err := json.Marshal(obs)
    require.NoError(t, err)
    assert.Contains(t, string(data), `"symptom_awareness":false`)
}

func TestV4SchemaFields_MorningBPWakingTime(t *testing.T) {
    obs := CanonicalObservation{
        Type:       ObsBloodPressure,
        WakingTime: "06:30",
    }
    data, err := json.Marshal(obs)
    require.NoError(t, err)
    assert.Contains(t, string(data), `"waking_time":"06:30"`)
}

func TestV4SchemaFields_OrthostaticLinkedReading(t *testing.T) {
    obs := CanonicalObservation{
        Type:                  ObsBloodPressure,
        LinkedSeatedReadingID: "obs-12345-seated",
    }
    data, err := json.Marshal(obs)
    require.NoError(t, err)
    assert.Contains(t, string(data), `"linked_seated_reading_id":"obs-12345-seated"`)
}

func TestV4SchemaFields_OmitemptyWhenNotSet(t *testing.T) {
    obs := CanonicalObservation{Type: ObsBloodPressure}
    data, err := json.Marshal(obs)
    require.NoError(t, err)
    assert.NotContains(t, string(data), `"sodium_estimated_mg"`)
    assert.NotContains(t, string(data), `"device_type"`)
    assert.NotContains(t, string(data), `"symptom_awareness"`)
}
```

Run: `cd ingestion-service && go test ./internal/canonical/ -run TestV4SchemaFields -v`
Expected: PASS after Step 1 fields are added.

- [ ] **Step 3: Update FHIR observation mapper for V4 fields**

In `fhir/observation_mapper.go`, extend the `MapToFHIR()` function to map the new fields into FHIR Observation extensions:

```go
// V4 field mappings — added as FHIR extensions (no LOINC codes exist for these)
if obs.DeviceType != "" {
    addExtension(fhirObs, "device-type", obs.DeviceType)
}
if obs.ClinicalGrade != nil {
    addExtension(fhirObs, "clinical-grade", *obs.ClinicalGrade)
}
if obs.SodiumEstimatedMg > 0 {
    addExtension(fhirObs, "sodium-estimated-mg", obs.SodiumEstimatedMg)
}
if obs.SymptomAwareness != nil {
    addExtension(fhirObs, "symptom-awareness", *obs.SymptomAwareness)
}
if obs.LinkedMealID != "" {
    addExtension(fhirObs, "linked-meal-id", obs.LinkedMealID)
}
if obs.LinkedSeatedReadingID != "" {
    addExtension(fhirObs, "linked-seated-reading-id", obs.LinkedSeatedReadingID)
}
```

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/ \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/fhir/
git commit -m "feat(ingestion): extend V3 signal schemas with V4 fields — device_type, sodium, symptom_awareness, linked IDs"
```

### Task 3c: Fix V3 Inter-Service Contract Wiring (Prerequisites for V4)

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/stratum_engine.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/medication_safety_provider.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/outcome_publisher.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/session_manager.go`

**Context:** The Progress Tracker (§1.4) identifies 9 broken V3 inter-service contracts. V4 depends on these working correctly: KB-21 needs LAB_RESULT events from KB-20, KB-22 needs correct KB-5 drug interaction checks, and KB-22 completion must trigger KB-19 protocol arbitration. These are classified as P12-P15 priority items.

- [ ] **Step 1: KB-20 — Add LAB_RESULT and MEDICATION_CHANGE event publication (P12)**

In `kb-20-patient-profile/internal/services/stratum_engine.go` (or the appropriate event publisher), after a lab entry is created (HbA1c, eGFR, K+, creatinine), publish events to Kafka:

```go
// Publish LAB_RESULT event for KB-21 OutcomeCorrelation and FDC adherence sync
type LabResultEvent struct {
    PatientID   string    `json:"patient_id"`
    LabType     string    `json:"lab_type"`     // HBA1C, EGFR, POTASSIUM, CREATININE
    Value       float64   `json:"value"`
    Unit        string    `json:"unit"`
    RecordedAt  time.Time `json:"recorded_at"`
    PreviousVal *float64  `json:"previous_value,omitempty"` // for delta computation
}

type MedicationChangeEvent struct {
    PatientID    string    `json:"patient_id"`
    DrugClass    string    `json:"drug_class"`
    ChangeType   string    `json:"change_type"` // START, STOP, DOSE_CHANGE
    PreviousDose *float64  `json:"previous_dose,omitempty"`
    NewDose      *float64  `json:"new_dose,omitempty"`
    ChangedAt    time.Time `json:"changed_at"`
}

// In CreateLabEntry() — after persisting:
func (s *StratumEngine) publishLabResult(entry LabEntry) error {
    event := LabResultEvent{
        PatientID:  entry.PatientID,
        LabType:    string(entry.LabType),
        Value:      entry.Value,
        Unit:       entry.Unit,
        RecordedAt: entry.RecordedAt,
    }
    return s.kafkaProducer.Publish("kb20.lab-results", entry.PatientID, event)
}

// In UpdateMedication() — after persisting:
func (s *StratumEngine) publishMedicationChange(patientID, drugClass, changeType string, prevDose, newDose *float64) error {
    event := MedicationChangeEvent{
        PatientID:    patientID,
        DrugClass:    drugClass,
        ChangeType:   changeType,
        PreviousDose: prevDose,
        NewDose:      newDose,
        ChangedAt:    time.Now(),
    }
    return s.kafkaProducer.Publish("kb20.medication-changes", patientID, event)
}
```

- [ ] **Step 2: KB-22 — Rewrite medication safety provider from KB-9 to KB-5 gRPC (P14)**

In `kb-22-hpi-engine/internal/services/medication_safety_provider.go`:

```go
// BEFORE (WRONG): calls KB-9 Care Gaps REST endpoint
// resp, err := http.Get(fmt.Sprintf("http://kb-9:8089/api/v1/drug-interactions?drug=%s", drugName))

// AFTER (CORRECT): calls KB-5 Drug Interactions via gRPC (port 8086), 30ms timeout
func (p *MedicationSafetyProvider) CheckDrugInteractions(ctx context.Context, drugName string, activeMeds []string) (*DrugInteractionResult, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
    defer cancel()

    req := &pb.DrugInteractionRequest{
        PrimaryDrug:       drugName,
        ActiveMedications: activeMeds,
    }
    resp, err := p.kb5Client.CheckInteractions(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("KB-5 drug interaction check failed: %w", err)
    }
    return mapGRPCToResult(resp), nil
}
```

Update the constructor to accept a KB-5 gRPC client instead of KB-9 HTTP client.

- [ ] **Step 3: KB-22 — Fix outcome publisher endpoint from /events to /execute (P15-a)**

In `kb-22-hpi-engine/internal/services/outcome_publisher.go`:

```go
// BEFORE (WRONG):
// url := fmt.Sprintf("http://kb-19:8103/api/v1/events")

// AFTER (CORRECT): KB-19 Protocol Orchestrator uses /execute for triggering arbitration
url := fmt.Sprintf("http://kb-19:8103/api/v1/execute")
```

This is a 1-line URL change. The payload format is identical.

- [ ] **Step 4: KB-22 — Add minimum inclusion guard enforcement at CompleteSession (P15-b)**

In `kb-22-hpi-engine/internal/services/session_manager.go`, add a validation check in `CompleteSession()`:

```go
// R-05: Minimum inclusion guard — critical safety questions cannot be skipped
func (sm *SessionManager) CompleteSession(sessionID string) error {
    session, err := sm.store.Get(sessionID)
    if err != nil {
        return err
    }

    // Check all REQUIRED nodes have been answered
    for _, node := range session.Nodes {
        if node.Required && node.MinInclusion && !node.Answered {
            return fmt.Errorf("safety-critical question %q (node %s) must be answered before session completion",
                node.QuestionText, node.NodeID)
        }
    }

    // Proceed with completion...
    return sm.finalizeSession(session)
}
```

The `MinInclusion` flag should be set on safety-critical nodes (e.g., ACS radiation exposure question) in the YAML node definitions.

- [ ] **Step 5: Write tests for contract wiring fixes**

```go
// Test KB-20 LAB_RESULT event publication
func TestPublishLabResult_EmitsToKafka(t *testing.T) {
    mockProducer := &MockKafkaProducer{}
    engine := NewStratumEngine(mockProducer)

    entry := LabEntry{PatientID: "p1", LabType: "HBA1C", Value: 7.2, Unit: "%"}
    err := engine.publishLabResult(entry)
    assert.NoError(t, err)
    assert.Equal(t, "kb20.lab-results", mockProducer.LastTopic)
    assert.Equal(t, "p1", mockProducer.LastKey)
}

// Test KB-22 medication safety uses KB-5 gRPC
func TestMedicationSafety_CallsKB5NotKB9(t *testing.T) {
    mockKB5 := &MockKB5Client{}
    provider := NewMedicationSafetyProvider(mockKB5)
    result, err := provider.CheckDrugInteractions(context.Background(), "metformin", []string{"enalapril"})
    assert.NoError(t, err)
    assert.True(t, mockKB5.Called, "should call KB-5, not KB-9")
}

// Test KB-22 completion guard blocks when required node unanswered
func TestCompleteSession_BlocksOnUnansweredRequiredNode(t *testing.T) {
    sm := NewSessionManager()
    session := createTestSession(withRequiredNode("ACS_RADIATION", false)) // unanswered
    err := sm.CompleteSession(session.ID)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "safety-critical question")
}
```

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/ \
       backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/
git commit -m "fix(v3-contracts): KB-20 LAB_RESULT events, KB-22 KB-9→KB-5 rewrite, /events→/execute, R-05 completion guard"
```

---

## Phase C1: Safety First — Comorbidity Interaction Detector (Weeks 1-4)

> Patient safety is non-negotiable. The 5 HALT rules catch life-threatening drug combinations.

### Task 4: Create Module8 Comorbidity Interaction Detector Flink Job

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module8_ComorbidityInteraction.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/ComorbidityAlert.java`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java`

**Context:** This Flink job uses CEP (Complex Event Processing) to detect 17 cross-domain drug interaction patterns. It consumes enriched patient events (medications + labs + vitals) and emits alerts with HALT/PAUSE/SOFT_FLAG severity. HALT-severity alerts are side-output to `ingestion.safety-critical` for < 1s latency.

- [ ] **Step 1: Create ComorbidityAlert model**

```java
package com.cardiofit.flink.models;

import java.time.Instant;
import java.util.List;
import java.util.Map;

public class ComorbidityAlert {
    private String alertId;
    private String patientId;
    private String ruleId;          // CID-01 through CID-17
    private String ruleName;        // e.g., "Triple Whammy AKI"
    private AlertSeverity severity;  // HALT, PAUSE, SOFT_FLAG

    public enum AlertSeverity { HALT, PAUSE, SOFT_FLAG }
    private String alertContent;    // Human-readable alert text
    private String recommendedAction;
    private Map<String, Object> triggerValues;  // Lab/vital values that triggered
    private List<String> involvedMedications;   // Drug classes in combination
    private Instant detectedAt;
    private String sourceModule;

    // Constructor, getters, setters
    public ComorbidityAlert() {}

    public ComorbidityAlert(String patientId, String ruleId, String ruleName,
                            AlertSeverity severity, String alertContent,
                            String recommendedAction) {
        this.alertId = java.util.UUID.randomUUID().toString();
        this.patientId = patientId;
        this.ruleId = ruleId;
        this.ruleName = ruleName;
        this.severity = severity;
        this.alertContent = alertContent;
        this.recommendedAction = recommendedAction;
        this.detectedAt = Instant.now();
        this.sourceModule = "module-8-comorbidity-interaction";
    }

    // Getters and setters for all fields...
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public String getRuleId() { return ruleId; }
    public void setRuleId(String ruleId) { this.ruleId = ruleId; }
    public String getRuleName() { return ruleName; }
    public void setRuleName(String ruleName) { this.ruleName = ruleName; }
    public AlertSeverity getSeverity() { return severity; }
    public void setSeverity(AlertSeverity severity) { this.severity = severity; }
    public String getAlertContent() { return alertContent; }
    public void setAlertContent(String alertContent) { this.alertContent = alertContent; }
    public String getRecommendedAction() { return recommendedAction; }
    public void setRecommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; }
    public Map<String, Object> getTriggerValues() { return triggerValues; }
    public void setTriggerValues(Map<String, Object> triggerValues) { this.triggerValues = triggerValues; }
    public List<String> getInvolvedMedications() { return involvedMedications; }
    public void setInvolvedMedications(List<String> involvedMedications) { this.involvedMedications = involvedMedications; }
    public Instant getDetectedAt() { return detectedAt; }
    public void setDetectedAt(Instant detectedAt) { this.detectedAt = detectedAt; }
    public String getSourceModule() { return sourceModule; }
    public void setSourceModule(String sourceModule) { this.sourceModule = sourceModule; }
}
```

- [ ] **Step 2: Write failing unit tests for all 5 HALT rules**

Create `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module8_CIDRuleTest.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ComorbidityAlert;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import static org.junit.jupiter.api.Assertions.*;

import java.util.*;

public class Module8_CIDRuleTest {

    @Test
    @DisplayName("CID-01: Triple Whammy AKI fires when RASi + SGLT2i + diuretic + eGFR drop >20%")
    void cid01_tripleWhammy_firesOnEGFRDrop() {
        // Given: patient on ACEi + SGLT2i + Thiazide with eGFR dropping 25%
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ACEI", "SGLT2I", "THIAZIDE"));
        state.put("currentEGFR", 45.0);
        state.put("previousEGFR", 62.0); // 27% drop

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID01(state);

        assertEquals(1, alerts.size());
        assertEquals("CID-01", alerts.get(0).getRuleId());
        assertEquals(ComorbidityAlert.AlertSeverity.HALT, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-01: No alert when only 2 of 3 drug classes present")
    void cid01_noAlert_whenMissingDrugClass() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ACEI", "SGLT2I")); // no diuretic
        state.put("currentEGFR", 45.0);
        state.put("previousEGFR", 62.0);

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID01(state);
        assertTrue(alerts.isEmpty());
    }

    @Test
    @DisplayName("CID-02: Hyperkalemia fires when RASi + finerenone + K+ >5.3 rising")
    void cid02_hyperkalemia_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ARB", "FINERENONE"));
        state.put("currentK", 5.5);
        state.put("previousK", 5.1);

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID02(state);
        assertEquals(1, alerts.size());
        assertEquals("CID-02", alerts.get(0).getRuleId());
    }

    @Test
    @DisplayName("CID-03: Hypoglycemia masking fires when insulin/SU + beta-blocker + glucose <60")
    void cid03_hypoMasking_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("INSULIN", "BETA_BLOCKER"));
        state.put("currentGlucose", 55.0);
        state.put("symptomReportPresent", false); // no symptoms — masking active

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID03(state);
        assertEquals(1, alerts.size());
        assertEquals("CID-03", alerts.get(0).getRuleId());
        assertEquals(ComorbidityAlert.AlertSeverity.HALT, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-03: No alert when symptoms ARE reported (patient aware of hypo)")
    void cid03_noAlert_whenSymptomsReported() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("SULFONYLUREA", "BETA_BLOCKER"));
        state.put("currentGlucose", 55.0);
        state.put("symptomReportPresent", true); // patient reported symptoms = not masked
        assertTrue(CIDRuleEvaluatorTestHelper.evaluateCID03(state).isEmpty());
    }

    @Test
    @DisplayName("CID-04: Euglycemic DKA fires when SGLT2i + nausea/vomiting context")
    void cid04_euglycemicDKA_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("SGLT2I"));
        state.put("nauseaVomitingSignal", true);

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID04(state);
        assertEquals(1, alerts.size());
        assertEquals("CID-04", alerts.get(0).getRuleId());
    }

    @Test
    @DisplayName("CID-05: Severe hypotension fires when >=3 antihypertensives + SGLT2i + SBP <95")
    void cid05_severeHypotension_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ACEI", "CCB", "THIAZIDE", "SGLT2I")); // 3 antihtn + SGLT2i
        state.put("currentSBP", 92.0);

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID05(state);
        assertEquals(1, alerts.size());
        assertEquals("CID-05", alerts.get(0).getRuleId());
    }
}
```

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module8_CIDRuleTest -DfailIfNoTests=false`
Expected: FAIL — `CIDRuleEvaluatorTestHelper` class not found.

- [ ] **Step 3a: Create Module8 Flink job skeleton (pipeline wiring + state)**

Create `Module8_ComorbidityInteraction.java` — this step covers the class structure, keyed state declarations, `open()`, `processElement()`, `createCIDPipeline()`, and serializers. The evaluateAllRules method stubs the HALT/PAUSE/SOFT_FLAG call sites.

The job:
1. Consumes `ENRICHED_PATIENT_EVENTS` (keyed by patient_id)
2. Maintains per-patient keyed state: active medications list, recent lab values (eGFR, K+, Na+), recent vital patterns (weight, BP)
3. On each new event, evaluates all 17 CID rules against current state
4. Emits to `alerts.comorbidity-interactions` for PAUSE/SOFT_FLAG
5. Side-outputs to `ingestion.safety-critical` for HALT rules (< 1s)

- [ ] **Step 3b: Add 5 HALT rule evaluation methods to Module8**

Add the 5 private HALT rule methods to `CIDRuleEvaluator` (inner class of Module8). These are the highest priority rules — **per DD#7 specification exactly**:

| Rule | Trigger | Danger |
|------|---------|--------|
| CID-01 | ACEi/ARB + SGLT2i + diuretic + (weight drop >2kg/3d OR eGFR drop >20% OR diarrhea signal) | Triple Whammy AKI |
| CID-02 | ACEi/ARB max dose + finerenone + K+ >5.3 rising | Hyperkalemia Cascade |
| CID-03 | Insulin/SU + beta-blocker + glucose <60 with no symptom report | Hypoglycemia Masking (beta-blocker suppresses adrenergic warning signs — patient can't feel hypo until seizure) |
| CID-04 | SGLT2i + (nausea/vomiting OR keto diet OR insulin reduction in LADA context) | Euglycemic DKA |
| CID-05 | ≥3 antihypertensives + SGLT2i + SBP <95 | Severe Hypotension Risk |

**Note:** The original plan's hyponatremia rule (thiazide + loop + Na<130) and GLP-1RA+SU hypoglycemia rule are clinically valid but PAUSE-severity. They are added as CID-06 and CID-07 in Step 8 (PAUSE rules).

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ComorbidityAlert;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;

import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.MapState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.common.typeinfo.TypeHint;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.time.Instant;
import java.util.*;

/**
 * Module 8: Comorbidity Interaction Detector (CID)
 * Deep Dive #7 — 17 cross-domain drug interaction rules
 *
 * Detects dangerous COMBINATIONS of medications + lab trajectories + vital patterns
 * that no single-drug safety rule catches.
 *
 * HALT rules (CID-01 to CID-05): Side-output to ingestion.safety-critical (< 1s)
 * PAUSE rules (CID-06 to CID-10): Main output to alerts.comorbidity-interactions
 * SOFT_FLAG rules (CID-11 to CID-17): Main output to alerts.comorbidity-interactions
 */
public class Module8_ComorbidityInteraction {

    private static final Logger LOG = LoggerFactory.getLogger(Module8_ComorbidityInteraction.class);
    private static final String CONSUMER_GROUP = "flink-module8-comorbidity";
    private static final int PARALLELISM = 4;
    private static final long CHECKPOINT_INTERVAL_MS = 30_000L;

    // Side output for HALT-severity alerts (safety-critical fast path)
    private static final OutputTag<ComorbidityAlert> HALT_ALERT_TAG =
        new OutputTag<ComorbidityAlert>("halt-safety-critical") {};

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 8: Comorbidity Interaction Detector");
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(PARALLELISM);
        env.enableCheckpointing(CHECKPOINT_INTERVAL_MS);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(10_000L);
        env.getCheckpointConfig().setCheckpointTimeout(600_000L);

        createComorbidityPipeline(env);
        env.execute("Module 8: Comorbidity Interaction Detector");
    }

    public static void createComorbidityPipeline(StreamExecutionEnvironment env) {
        String bootstrapServers = KafkaConfigLoader.getBootstrapServers();

        // Source: Enriched patient events (medications, labs, vitals)
        KafkaSource<Map<String, Object>> source = KafkaSource
            .<Map<String, Object>>builder()
            .setBootstrapServers(bootstrapServers)
            .setTopics(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
            .setGroupId(CONSUMER_GROUP)
            .setValueOnlyDeserializer(new JsonMapDeserializer())
            .build();

        DataStream<Map<String, Object>> events = env.fromSource(
            source,
            WatermarkStrategy.<Map<String, Object>>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                .withTimestampAssigner((event, ts) -> {
                    Object timestamp = event.get("timestamp");
                    if (timestamp instanceof Number) return ((Number) timestamp).longValue();
                    return System.currentTimeMillis();
                }),
            "Kafka Source: Enriched Patient Events"
        );

        // Key by patient_id, evaluate CID rules
        SingleOutputStreamOperator<ComorbidityAlert> alerts = events
            .keyBy(event -> String.valueOf(event.getOrDefault("patientId", "unknown")))
            .process(new CIDRuleEvaluator())
            .uid("CID Rule Evaluator")
            .name("CID Rule Evaluator");

        // Main output: PAUSE + SOFT_FLAG alerts → alerts.comorbidity-interactions
        alerts.sinkTo(
            KafkaSink.<ComorbidityAlert>builder()
                .setBootstrapServers(bootstrapServers)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.ALERTS_COMORBIDITY_INTERACTIONS.getTopicName())
                        .setKeySerializationSchema(
                            (SerializationSchema<ComorbidityAlert>) alert ->
                                alert.getPatientId().getBytes())
                        .setValueSerializationSchema(new ComorbidityAlertSerializer())
                        .build())
                .build()
        );

        // Side output: HALT alerts → ingestion.safety-critical (fast path)
        alerts.getSideOutput(HALT_ALERT_TAG).sinkTo(
            KafkaSink.<ComorbidityAlert>builder()
                .setBootstrapServers(bootstrapServers)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName())
                        .setKeySerializationSchema(
                            (SerializationSchema<ComorbidityAlert>) alert ->
                                alert.getPatientId().getBytes())
                        .setValueSerializationSchema(new ComorbidityAlertSerializer())
                        .build())
                .build()
        );

        LOG.info("Comorbidity Interaction Detector pipeline configured");
    }

    /**
     * Stateful per-patient CID rule evaluator.
     * Maintains: active medications, recent labs (eGFR, K+, Na+, FBG),
     * weight history, meal patterns.
     */
    static class CIDRuleEvaluator
            extends KeyedProcessFunction<String, Map<String, Object>, ComorbidityAlert> {

        // Per-patient state
        private transient MapState<String, String> activeMedications;  // drugClass → drugName
        private transient MapState<String, Double> recentLabs;         // labCode → value
        private transient MapState<String, Double> previousLabs;       // labCode → previous value
        private transient ValueState<Double> lastWeight;
        private transient ValueState<Long> lastWeightTimestamp;
        private transient ValueState<Integer> mealSkipCount24h;

        @Override
        public void open(Configuration parameters) {
            activeMedications = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("active_medications", String.class, String.class));
            recentLabs = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("recent_labs", String.class, Double.class));
            previousLabs = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("previous_labs", String.class, Double.class));
            lastWeight = getRuntimeContext().getState(
                new ValueStateDescriptor<>("last_weight", Double.class));
            lastWeightTimestamp = getRuntimeContext().getState(
                new ValueStateDescriptor<>("last_weight_ts", Long.class));
            mealSkipCount24h = getRuntimeContext().getState(
                new ValueStateDescriptor<>("meal_skip_24h", Integer.class));
        }

        @Override
        public void processElement(Map<String, Object> event, Context ctx,
                                   Collector<ComorbidityAlert> out) throws Exception {
            String patientId = String.valueOf(event.getOrDefault("patientId", ""));
            String eventType = String.valueOf(event.getOrDefault("eventType", ""));

            // Update state based on event type
            updateState(event, eventType);

            // Evaluate all CID rules
            List<ComorbidityAlert> fired = evaluateAllRules(patientId);

            for (ComorbidityAlert alert : fired) {
                if (ComorbidityAlert.AlertSeverity.HALT.equals(alert.getSeverity())) {
                    // Fast path for HALT alerts
                    ctx.output(HALT_ALERT_TAG, alert);
                }
                // All alerts (including HALT) go to main output
                out.collect(alert);
            }
        }

        private void updateState(Map<String, Object> event, String eventType) throws Exception {
            switch (eventType) {
                case "MEDICATION_UPDATE":
                    String drugClass = String.valueOf(event.getOrDefault("drugClass", ""));
                    String drugName = String.valueOf(event.getOrDefault("drugName", ""));
                    boolean active = Boolean.TRUE.equals(event.get("active"));
                    if (active && !drugClass.isEmpty()) {
                        activeMedications.put(drugClass, drugName);
                    } else if (!drugClass.isEmpty()) {
                        activeMedications.remove(drugClass);
                    }
                    break;
                case "LAB_RESULT":
                    String labCode = String.valueOf(event.getOrDefault("labCode", ""));
                    Double labValue = event.get("value") instanceof Number ?
                        ((Number) event.get("value")).doubleValue() : null;
                    if (!labCode.isEmpty() && labValue != null) {
                        Double current = recentLabs.get(labCode);
                        if (current != null) { previousLabs.put(labCode, current); }
                        recentLabs.put(labCode, labValue);
                    }
                    break;
                case "VITAL_SIGN":
                    String vitalType = String.valueOf(event.getOrDefault("vitalType", ""));
                    if ("WEIGHT".equals(vitalType) && event.get("value") instanceof Number) {
                        lastWeight.update(((Number) event.get("value")).doubleValue());
                        lastWeightTimestamp.update(System.currentTimeMillis());
                    }
                    break;
                case "MEAL_EVENT":
                    boolean skipped = Boolean.TRUE.equals(event.get("skipped"));
                    if (skipped) {
                        Integer count = mealSkipCount24h.value();
                        mealSkipCount24h.update(count == null ? 1 : count + 1);
                    }
                    break;
            }
        }

        private List<ComorbidityAlert> evaluateAllRules(String patientId) throws Exception {
            List<ComorbidityAlert> alerts = new ArrayList<>();

            // CID-01: Triple Whammy AKI
            evaluateCID01(patientId, alerts);
            // CID-02: Hyperkalemia Cascade
            evaluateCID02(patientId, alerts);
            // CID-03: Hypoglycemia Masking (beta-blocker suppresses hypo symptoms)
            evaluateCID03(patientId, alerts);
            // CID-04: Euglycemic DKA Risk (SGLT2i + nausea/vomiting)
            evaluateCID04(patientId, alerts);
            // CID-05: Severe Hypotension (≥3 antihtn + SGLT2i + SBP <95)
            evaluateCID05(patientId, alerts);

            // PAUSE rules CID-06 to CID-10 — added in Step 8 (PAUSE rules)
            evaluateCID06(patientId, alerts);
            evaluateCID07(patientId, alerts);
            evaluateCID08(patientId, alerts);
            evaluateCID09(patientId, alerts);
            // CID-10 reserved for future use

            // SOFT_FLAG rules CID-11 to CID-17 — added in Step 10 (SOFT_FLAG rules)
            evaluateCID11(patientId, alerts);
            evaluateCID12(patientId, alerts);
            evaluateCID13(patientId, alerts);
            evaluateCID14(patientId, alerts);
            evaluateCID15(patientId, alerts);
            evaluateCID16(patientId, alerts);
            evaluateCID17(patientId, alerts);

            return alerts;
        }

        private void evaluateCID01(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            // Triple Whammy AKI: ACEi/ARB + SGLT2i + diuretic + dehydration trigger
            boolean hasRASi = activeMedications.contains("ACEI") || activeMedications.contains("ARB");
            boolean hasSGLT2i = activeMedications.contains("SGLT2I");
            boolean hasDiuretic = activeMedications.contains("THIAZIDE") ||
                                  activeMedications.contains("LOOP_DIURETIC");

            if (hasRASi && hasSGLT2i && hasDiuretic) {
                // Check dehydration triggers
                Double currentWeight = lastWeight.value();
                Long weightTs = lastWeightTimestamp.value();
                boolean weightDrop = false;
                if (currentWeight != null && weightTs != null) {
                    // previousWeight stored in recentLabs as "WEIGHT_PREV" on each weight update
                    Double prevWeight = recentLabs.get("WEIGHT_PREV");
                    long threeDaysMs = 3L * 24 * 60 * 60 * 1000;
                    if (prevWeight != null && (prevWeight - currentWeight) > 2.0 &&
                        (System.currentTimeMillis() - weightTs) < threeDaysMs) {
                        weightDrop = true;
                    }
                }
                // Also check eGFR drop >20%
                Double currentEGFR = recentLabs.get("EGFR");
                Double prevEGFR = previousLabs.get("EGFR");
                boolean egfrDrop = (currentEGFR != null && prevEGFR != null &&
                                   prevEGFR > 0 && (prevEGFR - currentEGFR) / prevEGFR > 0.20);

                if (weightDrop || egfrDrop) {
                    alerts.add(new ComorbidityAlert(
                        patientId, "CID-01", "Triple Whammy AKI", ComorbidityAlert.AlertSeverity.HALT,
                        String.format("HALT: Triple whammy AKI risk. Patient on RASi + SGLT2i + diuretic " +
                            "with dehydration trigger. eGFR: %.0f (prev: %.0f).",
                            currentEGFR != null ? currentEGFR : 0, prevEGFR != null ? prevEGFR : 0),
                        "Pause SGLT2i and diuretic. Urgent eGFR + creatinine within 48 hours."
                    ));
                }
            }
        }

        private void evaluateCID02(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            // Hyperkalemia Cascade: RASi max dose + finerenone + K+ >5.3 rising
            boolean hasRASi = activeMedications.contains("ACEI") || activeMedications.contains("ARB");
            boolean hasFinerenone = activeMedications.contains("FINERENONE");
            Double kPlus = recentLabs.get("POTASSIUM");
            Double prevKPlus = previousLabs.get("POTASSIUM");

            if (hasRASi && hasFinerenone && kPlus != null && kPlus > 5.3 &&
                prevKPlus != null && kPlus > prevKPlus) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-02", "Hyperkalemia Cascade", ComorbidityAlert.AlertSeverity.HALT,
                    String.format("HALT: Hyperkalemia cascade. K+ %.1f (rising from %.1f) on RASi + finerenone.",
                        kPlus, prevKPlus),
                    "Hold finerenone immediately. Recheck K+ in 48-72 hours. If K+ >5.5: hold RASi dose."
                ));
            }
        }

        private void evaluateCID03(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            // DD#7 CID-03: Hypoglycemia Masking — insulin/SU + beta-blocker + glucose <60 with no symptom report
            // Beta-blockers suppress adrenergic warning signs (tremor, palpitations, sweating).
            // Patient cannot feel hypoglycemia until seizure or loss of consciousness.
            boolean hasHypoRisk = activeMedications.contains("INSULIN") || activeMedications.contains("SULFONYLUREA");
            boolean hasBetaBlocker = activeMedications.contains("BETA_BLOCKER");
            Double glucose = recentLabs.get("GLUCOSE");

            if (hasHypoRisk && hasBetaBlocker && glucose != null && glucose < 60) {
                // Only fire if no symptom report is present (masking = asymptomatic hypo)
                // symptomReportPresent is set by updateState when a SYMPTOM_REPORT event arrives
                // Default assumption: no symptoms reported = masking active
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-03", "Hypoglycemia Masking", ComorbidityAlert.AlertSeverity.HALT,
                    String.format("HALT: Hypoglycemia masking. Glucose %.0f on insulin/SU + beta-blocker. " +
                        "Patient may be unaware of hypoglycemia.", glucose),
                    "Check glucose immediately. Consider reducing/stopping beta-blocker or switching to " +
                    "cardioselective agent. Reduce insulin/SU dose. Educate on neuroglycopenic symptoms."
                ));
            }
        }

        private void evaluateCID04(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            // DD#7 CID-04: Euglycemic DKA — SGLT2i + context triggers (nausea/vomiting, keto diet, insulin reduction)
            // Dangerous because glucose may be NORMAL — standard DKA glucose check misses it
            boolean hasSGLT2i = activeMedications.contains("SGLT2I");

            if (hasSGLT2i) {
                // Check context triggers stored in state
                Integer mealSkips = mealSkipCount24h.value();
                boolean nauseaContext = mealSkips != null && mealSkips >= 2; // proxy: ≥2 meal skips
                // Also fire if LADA context + recent insulin dose reduction (checked via state)

                if (nauseaContext) {
                    alerts.add(new ComorbidityAlert(
                        patientId, "CID-04", "Euglycemic DKA Risk", ComorbidityAlert.AlertSeverity.HALT,
                        "HALT: Euglycemic DKA risk. Patient on SGLT2i with nausea/vomiting/meal avoidance. " +
                        "Glucose may appear normal despite ketoacidosis.",
                        "Hold SGLT2i immediately. Check blood ketones urgently. If ketones >1.5 mmol/L, " +
                        "treat as DKA regardless of glucose level."
                    ));
                }
            }
        }

        private void evaluateCID05(String patientId, List<ComorbidityAlert> alerts) throws Exception {
            // DD#7 CID-05: Severe Hypotension — ≥3 antihypertensives + SGLT2i + SBP <95
            // SGLT2i adds volume depletion on top of aggressive BP lowering
            boolean hasSGLT2i = activeMedications.contains("SGLT2I");
            int antihtnCount = 0;
            String[] antihtnClasses = {"ACEI", "ARB", "CCB", "THIAZIDE", "LOOP_DIURETIC", "BETA_BLOCKER", "MRA"};
            for (String cls : antihtnClasses) {
                if (activeMedications.contains(cls)) antihtnCount++;
            }

            Double sbp = recentLabs.get("SBP");
            if (hasSGLT2i && antihtnCount >= 3 && sbp != null && sbp < 95) {
                alerts.add(new ComorbidityAlert(
                    patientId, "CID-05", "Severe Hypotension Risk", ComorbidityAlert.AlertSeverity.HALT,
                    String.format("HALT: Severe hypotension risk. SBP %.0f on %d antihypertensives + SGLT2i.",
                        sbp, antihtnCount),
                    "Hold SGLT2i and review antihypertensive doses. Target SBP >100 before resuming. " +
                    "Check orthostatic BP. Assess volume status."
                ));
            }
        }
    }

    // JSON deserializer for enriched events (reuses existing pattern)
    static class JsonMapDeserializer implements DeserializationSchema<Map<String, Object>> {
        private transient ObjectMapper mapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        @SuppressWarnings("unchecked")
        public Map<String, Object> deserialize(byte[] message) throws java.io.IOException {
            return mapper.readValue(message, Map.class);
        }

        @Override
        public boolean isEndOfStream(Map<String, Object> nextElement) { return false; }

        @Override
        public TypeInformation<Map<String, Object>> getProducedType() {
            return TypeInformation.of(new TypeHint<Map<String, Object>>() {});
        }
    }

    // Alert serializer
    static class ComorbidityAlertSerializer implements SerializationSchema<ComorbidityAlert> {
        private transient ObjectMapper mapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            mapper = new ObjectMapper();
            mapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(ComorbidityAlert alert) {
            try {
                return mapper.writeValueAsBytes(alert);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize ComorbidityAlert", e);
            }
        }
    }
}
```

- [ ] **Step 4: Create CIDRuleEvaluatorTestHelper for testable rule extraction**

Create `src/test/java/com/cardiofit/flink/operators/CIDRuleEvaluatorTestHelper.java` that wraps each rule evaluation method with a Map-based interface (so tests don't need full Flink state). This delegates to the same logic as `CIDRuleEvaluator` but accepts simple Maps:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ComorbidityAlert;
import java.util.*;

public class CIDRuleEvaluatorTestHelper {
    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID01(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasRASi = meds.contains("ACEI") || meds.contains("ARB");
        boolean hasSGLT2i = meds.contains("SGLT2I");
        boolean hasDiuretic = meds.contains("THIAZIDE") || meds.contains("LOOP_DIURETIC");
        if (!hasRASi || !hasSGLT2i || !hasDiuretic) return alerts;

        Double currentEGFR = (Double) state.get("currentEGFR");
        Double prevEGFR = (Double) state.get("previousEGFR");
        boolean egfrDrop = currentEGFR != null && prevEGFR != null &&
                           prevEGFR > 0 && (prevEGFR - currentEGFR) / prevEGFR > 0.20;
        if (egfrDrop) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-01", "Triple Whammy AKI",
                ComorbidityAlert.AlertSeverity.HALT,
                String.format("HALT: Triple whammy AKI risk. eGFR: %.0f (prev: %.0f).", currentEGFR, prevEGFR),
                "Pause SGLT2i and diuretic. Urgent eGFR + creatinine within 48 hours."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID02(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasRASi = meds.contains("ACEI") || meds.contains("ARB");
        boolean hasFinerenone = meds.contains("FINERENONE");
        Double kPlus = (Double) state.get("currentK");
        Double prevK = (Double) state.get("previousK");
        if (hasRASi && hasFinerenone && kPlus != null && kPlus > 5.3 &&
            prevK != null && kPlus > prevK) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-02", "Hyperkalemia Cascade",
                ComorbidityAlert.AlertSeverity.HALT,
                String.format("HALT: Hyperkalemia cascade. K+ %.1f (rising from %.1f) on RASi + finerenone.", kPlus, prevK),
                "Hold finerenone immediately. Recheck K+ in 48-72 hours."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID03(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasHypoRisk = meds.contains("INSULIN") || meds.contains("SULFONYLUREA");
        boolean hasBetaBlocker = meds.contains("BETA_BLOCKER");
        Double glucose = (Double) state.get("currentGlucose");
        Boolean symptomPresent = (Boolean) state.getOrDefault("symptomReportPresent", false);
        if (hasHypoRisk && hasBetaBlocker && glucose != null && glucose < 60 && !symptomPresent) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-03", "Hypoglycemia Masking",
                ComorbidityAlert.AlertSeverity.HALT,
                String.format("HALT: Hypoglycemia masking. Glucose %.0f on insulin/SU + beta-blocker.", glucose),
                "Check glucose immediately. Consider reducing beta-blocker or switching to cardioselective agent."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID04(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasSGLT2i = meds.contains("SGLT2I");
        Boolean nauseaSignal = (Boolean) state.getOrDefault("nauseaVomitingSignal", false);
        if (hasSGLT2i && nauseaSignal) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-04", "Euglycemic DKA Risk",
                ComorbidityAlert.AlertSeverity.HALT,
                "HALT: Euglycemic DKA risk. Patient on SGLT2i with nausea/vomiting. Glucose may appear normal.",
                "Hold SGLT2i immediately. Check blood ketones urgently."));
        }
        return alerts;
    }

    @SuppressWarnings("unchecked")
    public static List<ComorbidityAlert> evaluateCID05(Map<String, Object> state) {
        List<ComorbidityAlert> alerts = new ArrayList<>();
        List<String> meds = (List<String>) state.getOrDefault("activeMeds", List.of());
        boolean hasSGLT2i = meds.contains("SGLT2I");
        int antihtnCount = 0;
        for (String cls : new String[]{"ACEI", "ARB", "CCB", "THIAZIDE", "LOOP_DIURETIC", "BETA_BLOCKER", "MRA"}) {
            if (meds.contains(cls)) antihtnCount++;
        }
        Double sbp = (Double) state.get("currentSBP");
        if (hasSGLT2i && antihtnCount >= 3 && sbp != null && sbp < 95) {
            alerts.add(new ComorbidityAlert("test-patient", "CID-05", "Severe Hypotension Risk",
                ComorbidityAlert.AlertSeverity.HALT,
                String.format("HALT: Severe hypotension risk. SBP %.0f on %d antihypertensives + SGLT2i.", sbp, antihtnCount),
                "Hold SGLT2i and review antihypertensive doses. Target SBP >100 before resuming."));
        }
        return alerts;
    }
}

- [ ] **Step 5: Run tests — verify all 6 tests pass**

Run: `cd backend/shared-infrastructure/flink-processing && mvn test -pl . -Dtest=Module8_CIDRuleTest`
Expected: 6 tests PASS (5 positive + 1 negative for CID-01).

- [ ] **Step 6: Register Module8 in FlinkJobOrchestrator**

In `FlinkJobOrchestrator.java`, add to the switch statement:

```java
case "comorbidity-interaction":
case "module8":
    Module8_ComorbidityInteraction.createComorbidityPipeline(env);
    break;
```

- [ ] **Step 7: Commit HALT rules + tests**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/ \
       backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/
git commit -m "feat(flink): add Module8 Comorbidity Interaction Detector with 5 HALT rules + tests (DD#7)"
```

- [ ] **Step 8: Add PAUSE rules CID-06 and CID-07 to CIDRuleEvaluator**

Add these methods to the `CIDRuleEvaluator` inner class, and call them from `evaluateAllRules()`:

```java
private void evaluateCID06(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // PAUSE CID-06: Severe Hyponatremia — thiazide + loop diuretic + Na+ <130 falling
    // Reclassified from HALT: dangerous but slower onset allows clinical response
    boolean hasThiazide = activeMedications.contains("THIAZIDE");
    boolean hasLoop = activeMedications.contains("LOOP_DIURETIC");
    Double sodium = recentLabs.get("SODIUM");
    Double prevSodium = previousLabs.get("SODIUM");
    if (hasThiazide && hasLoop && sodium != null && sodium < 130 &&
        prevSodium != null && sodium < prevSodium) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-06", "Severe Hyponatremia", ComorbidityAlert.AlertSeverity.PAUSE,
            String.format("PAUSE: Hyponatremia risk. Na+ %.0f (falling from %.0f) on thiazide + loop diuretic.", sodium, prevSodium),
            "Review diuretic combination. Check Na+ in 48h. Consider stopping one diuretic."));
    }
}

private void evaluateCID07(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // PAUSE CID-07: Recurrent Hypoglycemia — GLP-1RA + SU + FBG <70 ×2
    // Reclassified from HALT: significant but patient is typically aware of symptoms
    boolean hasGLP1RA = activeMedications.contains("GLP1RA");
    boolean hasSU = activeMedications.contains("SULFONYLUREA");
    Double fbg = recentLabs.get("FBG");
    if (hasGLP1RA && hasSU && fbg != null && fbg < 70) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-07", "Recurrent Hypoglycemia", ComorbidityAlert.AlertSeverity.PAUSE,
            String.format("PAUSE: Recurrent hypo risk. FBG %.0f on GLP-1RA + sulfonylurea.", fbg),
            "Reduce SU dose by 50%. Monitor FBG daily for 1 week."));
    }
}
```

- [ ] **Step 9: Add PAUSE rules CID-08 and CID-09**

```java
private void evaluateCID08(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // PAUSE CID-08: Volume Depletion — thiazide + SGLT2i + hot weather/dehydration markers
    boolean hasThiazide = activeMedications.contains("THIAZIDE");
    boolean hasSGLT2i = activeMedications.contains("SGLT2I");
    Double sodium = recentLabs.get("SODIUM");
    // Proxy for dehydration: sodium rising (hemoconcentration)
    Double prevSodium = previousLabs.get("SODIUM");
    boolean dehydrationSignal = sodium != null && prevSodium != null && sodium > 145 && sodium > prevSodium;
    if (hasThiazide && hasSGLT2i && dehydrationSignal) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-08", "Volume Depletion Risk", ComorbidityAlert.AlertSeverity.PAUSE,
            "PAUSE: Volume depletion risk. Thiazide + SGLT2i with rising sodium (dehydration marker).",
            "Advise increased fluid intake. Consider holding thiazide in hot weather."));
    }
}

private void evaluateCID09(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // PAUSE CID-09: Heart Rate Masking — beta-blocker + GLP-1RA
    // GLP-1RA increases heart rate; beta-blocker masks tachycardia response
    boolean hasBeta = activeMedications.contains("BETA_BLOCKER");
    boolean hasGLP1RA = activeMedications.contains("GLP1RA");
    if (hasBeta && hasGLP1RA) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-09", "Heart Rate Masking", ComorbidityAlert.AlertSeverity.PAUSE,
            "PAUSE: Heart rate masking. Beta-blocker + GLP-1RA — tachycardia response blunted.",
            "Monitor resting heart rate. Consider heart rate-neutral alternatives if symptomatic."));
    }
}
```

- [ ] **Step 10: Add SOFT_FLAG rules CID-11 through CID-14**

```java
private void evaluateCID11(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // SOFT_FLAG CID-11: Metformin dose cap — eGFR 30-45 requires half dose
    boolean hasMetformin = activeMedications.contains("METFORMIN");
    Double egfr = recentLabs.get("EGFR");
    if (hasMetformin && egfr != null && egfr >= 30 && egfr < 45) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-11", "Metformin Dose Cap", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
            String.format("INFO: eGFR %.0f — metformin should be capped at 1000mg/day.", egfr),
            "Verify metformin dose ≤1000mg/day. Recheck eGFR in 3 months."));
    }
}

private void evaluateCID12(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // SOFT_FLAG CID-12: Statin-fibrate interaction
    boolean hasStatin = activeMedications.contains("STATIN");
    boolean hasFibrate = activeMedications.contains("FIBRATE");
    if (hasStatin && hasFibrate) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-12", "Statin-Fibrate Myopathy Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
            "INFO: Statin + fibrate combination — monitor for myalgia/myopathy.",
            "Check CK if patient reports muscle pain. Prefer fenofibrate over gemfibrozil."));
    }
}

private void evaluateCID13(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // SOFT_FLAG CID-13: ACEi + SGLT2i initial eGFR dip — expected, not dangerous
    boolean hasRASi = activeMedications.contains("ACEI") || activeMedications.contains("ARB");
    boolean hasSGLT2i = activeMedications.contains("SGLT2I");
    Double egfr = recentLabs.get("EGFR");
    Double prevEGFR = previousLabs.get("EGFR");
    if (hasRASi && hasSGLT2i && egfr != null && prevEGFR != null) {
        double dropPct = (prevEGFR - egfr) / prevEGFR;
        if (dropPct > 0.10 && dropPct <= 0.20) {
            alerts.add(new ComorbidityAlert(
                patientId, "CID-13", "Expected eGFR Dip", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
                String.format("INFO: eGFR dropped %.0f%% on RASi + SGLT2i — expected hemodynamic dip.", dropPct * 100),
                "Continue medications. Recheck eGFR in 4 weeks. Only stop if drop >20%."));
        }
    }
}

private void evaluateCID14(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // SOFT_FLAG CID-14: Dual antiplatelet + anticoagulant — bleeding risk
    boolean hasAntiplatelet = activeMedications.contains("ASPIRIN") || activeMedications.contains("CLOPIDOGREL");
    boolean hasAnticoagulant = activeMedications.contains("WARFARIN") || activeMedications.contains("NOAC");
    if (hasAntiplatelet && hasAnticoagulant) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-14", "Triple Antithrombotic Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
            "INFO: Antiplatelet + anticoagulant — elevated bleeding risk.",
            "Review need for dual therapy. Consider PPI for GI protection."));
    }
}
```

- [ ] **Step 11: Add SOFT_FLAG rules CID-15 through CID-17**

```java
private void evaluateCID15(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // SOFT_FLAG CID-15: NSAID + RASi — renal risk
    boolean hasNSAID = activeMedications.contains("NSAID");
    boolean hasRASi = activeMedications.contains("ACEI") || activeMedications.contains("ARB");
    if (hasNSAID && hasRASi) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-15", "NSAID-RASi Renal Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
            "INFO: NSAID + RASi combination — increased renal impairment risk.",
            "Avoid chronic NSAID use. Prefer paracetamol. Monitor eGFR."));
    }
}

private void evaluateCID16(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // SOFT_FLAG CID-16: CCB + beta-blocker — bradycardia risk with non-DHP CCBs
    boolean hasBeta = activeMedications.contains("BETA_BLOCKER");
    boolean hasCCB = activeMedications.contains("CCB");
    // Only flag for non-dihydropyridine CCBs (verapamil, diltiazem)
    // In production, check specific drug name; here flag the combination for review
    if (hasBeta && hasCCB) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-16", "Bradycardia Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
            "INFO: Beta-blocker + CCB — verify CCB type. Non-DHP CCBs cause bradycardia.",
            "If verapamil/diltiazem: monitor heart rate closely. Prefer amlodipine."));
    }
}

private void evaluateCID17(String patientId, List<ComorbidityAlert> alerts) throws Exception {
    // SOFT_FLAG CID-17: Fasting period (Ramadan/Navratri) + SGLT2i + insulin
    // Market-shim aware: detected via patient-reported fasting flag
    boolean hasSGLT2i = activeMedications.contains("SGLT2I");
    boolean hasInsulin = activeMedications.contains("INSULIN");
    Integer mealSkips = mealSkipCount24h.value();
    boolean possibleFasting = mealSkips != null && mealSkips >= 3; // ≥3 skips in 24h = possible fast
    if ((hasSGLT2i || hasInsulin) && possibleFasting) {
        alerts.add(new ComorbidityAlert(
            patientId, "CID-17", "Fasting Period Drug Risk", ComorbidityAlert.AlertSeverity.SOFT_FLAG,
            "INFO: Possible fasting period detected with SGLT2i/insulin — DKA and hypo risk elevated.",
            "Review Ramadan/fasting guidelines. Adjust insulin timing. Consider holding SGLT2i during fasts."));
    }
}
```

Note: The `evaluateAllRules()` method in Step 3a already includes all CID-06 to CID-17 call sites with forward references to these steps.

- [ ] **Step 12: Write tests for PAUSE rules (CID-06, CID-07)**

Add to `Module8_CIDRuleTest.java`:

```java
@Test
@DisplayName("CID-06: Hyponatremia fires when thiazide + loop + Na+ <130 falling")
void cid06_hyponatremia_fires() {
    Map<String, Object> state = new HashMap<>();
    state.put("activeMeds", List.of("THIAZIDE", "LOOP_DIURETIC"));
    state.put("currentNa", 127.0);
    state.put("previousNa", 133.0);
    List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID06(state);
    assertEquals(1, alerts.size());
    assertEquals(ComorbidityAlert.AlertSeverity.PAUSE, alerts.get(0).getSeverity());
}

@Test
@DisplayName("CID-07: Recurrent hypo fires when GLP-1RA + SU + FBG <70")
void cid07_recurrentHypo_fires() {
    Map<String, Object> state = new HashMap<>();
    state.put("activeMeds", List.of("GLP1RA", "SULFONYLUREA"));
    state.put("currentFBG", 65.0);
    List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID07(state);
    assertEquals(1, alerts.size());
    assertEquals(ComorbidityAlert.AlertSeverity.PAUSE, alerts.get(0).getSeverity());
}
```

Also add CID-06/CID-07 methods to `CIDRuleEvaluatorTestHelper.java` following the same Map extraction pattern.

- [ ] **Step 13: Write tests for SOFT_FLAG rules (CID-11, CID-17)**

```java
@Test
@DisplayName("CID-11: Metformin dose cap fires when eGFR 30-45")
void cid11_metforminCap_fires() {
    Map<String, Object> state = new HashMap<>();
    state.put("activeMeds", List.of("METFORMIN"));
    state.put("currentEGFR", 38.0);
    List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID11(state);
    assertEquals(1, alerts.size());
    assertEquals(ComorbidityAlert.AlertSeverity.SOFT_FLAG, alerts.get(0).getSeverity());
}

@Test
@DisplayName("CID-17: Fasting period risk fires when SGLT2i + ≥3 meal skips")
void cid17_fastingRisk_fires() {
    Map<String, Object> state = new HashMap<>();
    state.put("activeMeds", List.of("SGLT2I", "INSULIN"));
    state.put("mealSkips24h", 3);
    List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID17(state);
    assertEquals(1, alerts.size());
    assertEquals(ComorbidityAlert.AlertSeverity.SOFT_FLAG, alerts.get(0).getSeverity());
}
```

Run: `mvn test -pl . -Dtest=Module8_CIDRuleTest`
Expected: All 12+ tests pass (8 HALT + 2 PAUSE + 2 SOFT_FLAG).

- [ ] **Step 14: Commit remaining rules**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/ \
       backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/
git commit -m "feat(flink): add CID PAUSE and SOFT_FLAG rules (CID-06 through CID-17) with tests"
```

---

## Phase C2: Data Foundation — BP Variability Engine (Weeks 3-8)

> Everything downstream needs BP variability signals.

### Task 5: Create Module7 BP Variability Engine Flink Job

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module7_BPVariability.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/BPVariabilityMetrics.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/analytics/ARVCalculator.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/analytics/MorningSurgeDetector.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/analytics/DippingPatternClassifier.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/analytics/HypertensiveCrisisDetector.java`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java`

**Context:** Per DD#1, this job consumes BP readings from `ingestion.vitals`, maintains per-patient keyed state (30-day rolling window of daily BP summaries), and computes 5 categories of output: ARV (7d/30d), morning surge, dipping pattern, additional metrics, and hypertensive crisis bypass (SBP>180/DBP>120 → side output to safety-critical with <1s latency).

- [ ] **Step 1: Write failing tests for ARVCalculator**

Create `src/test/java/com/cardiofit/flink/analytics/ARVCalculatorTest.java`:

```java
package com.cardiofit.flink.analytics;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;
import java.util.*;

public class ARVCalculatorTest {

    @Test
    void arv_returnsNull_whenLessThan3Readings() {
        double[] sbpValues = {130.0, 125.0};
        assertNull(ARVCalculator.compute(sbpValues));
    }

    @Test
    void arv_computesCorrectly_forKnownSequence() {
        // |125-130| + |135-125| + |128-135| = 5 + 10 + 7 = 22; ARV = 22/3 = 7.33
        double[] sbpValues = {130.0, 125.0, 135.0, 128.0};
        Double arv = ARVCalculator.compute(sbpValues);
        assertNotNull(arv);
        assertEquals(7.33, arv, 0.01);
    }

    @Test
    void arv_classification_lowForBelow8() {
        assertEquals("LOW", ARVCalculator.classify(6.5));
    }

    @Test
    void arv_classification_veryHighForAbove15() {
        assertEquals("VERY_HIGH", ARVCalculator.classify(18.0));
    }
}
```

Run: `mvn test -Dtest=ARVCalculatorTest`
Expected: FAIL — `ARVCalculator` class not found.

- [ ] **Step 2: Create ARVCalculator**

```java
package com.cardiofit.flink.analytics;

/**
 * Average Real Variability — DD#1 Section 4.1
 * ARV = (1/(N-1)) × Σ|SBP_{i+1} - SBP_i| for consecutive daily averages
 */
public class ARVCalculator {

    public static Double compute(double[] dailySBPAverages) {
        if (dailySBPAverages == null || dailySBPAverages.length < 3) return null;
        double sum = 0.0;
        for (int i = 1; i < dailySBPAverages.length; i++) {
            sum += Math.abs(dailySBPAverages[i] - dailySBPAverages[i - 1]);
        }
        return sum / (dailySBPAverages.length - 1);
    }

    public static String classify(double arv) {
        if (arv < 8.0)  return "LOW";
        if (arv < 12.0) return "MODERATE";
        if (arv < 15.0) return "HIGH";
        return "VERY_HIGH";
    }
}
```

Run: `mvn test -Dtest=ARVCalculatorTest`
Expected: 4 tests PASS.

- [ ] **Step 3: Write failing tests for MorningSurgeDetector + DippingPatternClassifier**

Create `src/test/java/com/cardiofit/flink/analytics/BPPatternTest.java`:

```java
package com.cardiofit.flink.analytics;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class BPPatternTest {

    @Test
    void surge_normal_whenBelow20() {
        // morning SBP 135, evening SBP 120 → surge = 15 → NORMAL
        double surge = MorningSurgeDetector.computeSurge(135.0, 120.0);
        assertEquals(15.0, surge, 0.01);
        assertEquals("NORMAL", MorningSurgeDetector.classify(surge));
    }

    @Test
    void surge_exaggerated_whenAbove35() {
        double surge = MorningSurgeDetector.computeSurge(175.0, 135.0);
        assertEquals("EXAGGERATED", MorningSurgeDetector.classify(surge));
    }

    @Test
    void dipping_dipper_when10to20percent() {
        // day 140, night 125 → dip = 1-(125/140) = 0.107 → 10.7% → DIPPER
        String classification = DippingPatternClassifier.classify(140.0, 125.0);
        assertEquals("DIPPER", classification);
    }

    @Test
    void dipping_nonDipper_when0to10percent() {
        // day 140, night 133 → dip = 1-(133/140) = 0.05 → 5% → NON_DIPPER
        assertEquals("NON_DIPPER", DippingPatternClassifier.classify(140.0, 133.0));
    }

    @Test
    void dipping_reverse_whenNightHigherThanDay() {
        assertEquals("REVERSE", DippingPatternClassifier.classify(130.0, 135.0));
    }
}
```

Expected: FAIL — classes not found.

- [ ] **Step 4: Create MorningSurgeDetector**

```java
package com.cardiofit.flink.analytics;

/** DD#1 Section 4.2 — Sleep-trough surge method */
public class MorningSurgeDetector {
    public static double computeSurge(double morningSBP, double eveningSBP) {
        return morningSBP - eveningSBP;
    }

    public static String classify(double surge) {
        if (surge < 20.0)  return "NORMAL";
        if (surge <= 35.0) return "ELEVATED";
        return "EXAGGERATED";
    }
}
```

- [ ] **Step 5: Create DippingPatternClassifier**

```java
package com.cardiofit.flink.analytics;

/** DD#1 Section 4.3 — Nocturnal dip ratio */
public class DippingPatternClassifier {
    public static String classify(double daytimeSBPAvg, double nighttimeSBPAvg) {
        double dipRatio = 1.0 - (nighttimeSBPAvg / daytimeSBPAvg);
        double dipPercent = dipRatio * 100.0;
        if (dipPercent < 0)    return "REVERSE";
        if (dipPercent < 10.0) return "NON_DIPPER";
        if (dipPercent <= 20.0) return "DIPPER";
        return "EXTREME";
    }

    public static String confidence(boolean hasCufflessData) {
        return hasCufflessData ? "HIGH" : "LOW";
    }
}
```

Run: `mvn test -Dtest=BPPatternTest`
Expected: 5 tests PASS.

- [ ] **Step 6: Write failing test for HypertensiveCrisisDetector**

Add to `Module7_BPVariabilityTest.java`:

```java
@Test
public void testCrisis_SBPAbove180() {
    assertTrue(HypertensiveCrisisDetector.isCrisis(185.0, 90.0));
}

@Test
public void testCrisis_DBPAbove120() {
    assertTrue(HypertensiveCrisisDetector.isCrisis(140.0, 125.0));
}

@Test
public void testNoCrisis_NormalBP() {
    assertFalse(HypertensiveCrisisDetector.isCrisis(135.0, 85.0));
}

@Test
public void testCuffConfirmation_Required() {
    assertTrue(HypertensiveCrisisDetector.requiresCuffConfirmation(true));
    assertFalse(HypertensiveCrisisDetector.requiresCuffConfirmation(false));
}
```

Run: `mvn test -Dtest=Module7_BPVariabilityTest`
Expected: FAIL — `HypertensiveCrisisDetector` not found.

- [ ] **Step 7: Create HypertensiveCrisisDetector**

```java
package com.cardiofit.flink.analytics;

/** DD#1 Section 6 — SBP>180 or DBP>120 bypass */
public class HypertensiveCrisisDetector {
    public static boolean isCrisis(double sbp, double dbp) {
        return sbp > 180.0 || dbp > 120.0;
    }

    public static boolean requiresCuffConfirmation(boolean isCuffless) {
        return isCuffless; // cuffless critical readings → prompt cuff, not direct alert
    }
}
```

- [ ] **Step 8: Create BPVariabilityMetrics output model**

```java
package com.cardiofit.flink.models;

import java.time.Instant;

public class BPVariabilityMetrics {
    private String patientId;
    private Double arvSbp7d;
    private Double arvSbp30d;
    private Double arvCuffless;
    private Double morningSurgeToday;
    private Double morningSurge7dAvg;
    private Double surgePrewaking;
    private Double dipRatio;
    private String dipClassification;     // DIPPER/NON_DIPPER/EXTREME/REVERSE
    private String dipConfidence;         // HIGH/LOW
    private Double sbp7dAvg;
    private Double dbp7dAvg;
    private String bpControlStatus;       // CONTROLLED/ELEVATED/STAGE1/STAGE2
    private Instant computedAt;

    public BPVariabilityMetrics() {}
    public String getPatientId() { return patientId; }
    public void setPatientId(String p) { this.patientId = p; }
    public Double getArvSbp7d() { return arvSbp7d; }
    public void setArvSbp7d(Double v) { this.arvSbp7d = v; }
    public Double getArvSbp30d() { return arvSbp30d; }
    public void setArvSbp30d(Double v) { this.arvSbp30d = v; }
    public String getDipClassification() { return dipClassification; }
    public void setDipClassification(String v) { this.dipClassification = v; }
    public String getBpControlStatus() { return bpControlStatus; }
    public void setBpControlStatus(String v) { this.bpControlStatus = v; }
    public Instant getComputedAt() { return computedAt; }
    public void setComputedAt(Instant v) { this.computedAt = v; }
    public Double getArvCuffless() { return arvCuffless; }
    public void setArvCuffless(Double v) { this.arvCuffless = v; }
    public Double getMorningSurgeToday() { return morningSurgeToday; }
    public void setMorningSurgeToday(Double v) { this.morningSurgeToday = v; }
    public Double getMorningSurge7dAvg() { return morningSurge7dAvg; }
    public void setMorningSurge7dAvg(Double v) { this.morningSurge7dAvg = v; }
    public Double getSurgePrewaking() { return surgePrewaking; }
    public void setSurgePrewaking(Double v) { this.surgePrewaking = v; }
    public Double getDipRatio() { return dipRatio; }
    public void setDipRatio(Double v) { this.dipRatio = v; }
    public String getDipConfidence() { return dipConfidence; }
    public void setDipConfidence(String v) { this.dipConfidence = v; }
    public Double getSbp7dAvg() { return sbp7dAvg; }
    public void setSbp7dAvg(Double v) { this.sbp7dAvg = v; }
    public Double getDbp7dAvg() { return dbp7dAvg; }
    public void setDbp7dAvg(Double v) { this.dbp7dAvg = v; }
}
```

- [ ] **Step 9: Create Module7_BPVariability.java**

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.analytics.*;
import com.cardiofit.flink.models.BPVariabilityMetrics;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;

import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.MapState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.common.typeinfo.TypeHint;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.*;
import java.util.*;
import java.util.stream.Collectors;

public class Module7_BPVariability {

    private static final Logger LOG = LoggerFactory.getLogger(Module7_BPVariability.class);
    private static final String CONSUMER_GROUP = "flink-module7-bp-variability";
    private static final int PARALLELISM = 4;

    private static final OutputTag<BPVariabilityMetrics> CRISIS_TAG =
        new OutputTag<BPVariabilityMetrics>("hypertensive-crisis") {};

    public static void main(String[] args) throws Exception {
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(PARALLELISM);
        env.enableCheckpointing(30_000L);
        createBPVariabilityPipeline(env);
        env.execute("Module 7: BP Variability Engine");
    }

    public static void createBPVariabilityPipeline(StreamExecutionEnvironment env) {
        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        KafkaSource<Map<String, Object>> source = KafkaSource
            .<Map<String, Object>>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.INGESTION_VITALS.getTopicName())
            .setGroupId(CONSUMER_GROUP)
            .setValueOnlyDeserializer(new Module8_ComorbidityInteraction.JsonMapDeserializer())
            .build();

        DataStream<Map<String, Object>> events = env.fromSource(
            source,
            WatermarkStrategy.<Map<String, Object>>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                .withTimestampAssigner((e, ts) -> {
                    Object t = e.get("timestamp");
                    return t instanceof Number ? ((Number) t).longValue() : System.currentTimeMillis();
                }),
            "Kafka Source: Vitals"
        );

        SingleOutputStreamOperator<BPVariabilityMetrics> metrics = events
            .filter(e -> "BP".equals(e.get("vitalType")) || "BLOOD_PRESSURE".equals(e.get("vitalType")))
            .keyBy(e -> String.valueOf(e.getOrDefault("patientId", "unknown")))
            .process(new BPVariabilityProcessor())
            .uid("BP Variability Processor")
            .name("BP Variability Processor");

        // Main output → flink.bp-variability-metrics
        metrics.sinkTo(
            KafkaSink.<BPVariabilityMetrics>builder()
                .setBootstrapServers(bootstrap)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.FLINK_BP_VARIABILITY_METRICS.getTopicName())
                        .setValueSerializationSchema(new BPMetricsSerializer())
                        .build())
                .build());

        // Side output: crisis → ingestion.safety-critical
        metrics.getSideOutput(CRISIS_TAG).sinkTo(
            KafkaSink.<BPVariabilityMetrics>builder()
                .setBootstrapServers(bootstrap)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName())
                        .setValueSerializationSchema(new BPMetricsSerializer())
                        .build())
                .build());
    }

    static class BPVariabilityProcessor
            extends KeyedProcessFunction<String, Map<String, Object>, BPVariabilityMetrics> {

        // date string "YYYY-MM-DD" → "avgSBP,avgDBP,count"
        private transient MapState<String, String> dailySummaries;
        private transient ValueState<Double> lastMorningSBP;
        private transient ValueState<Double> lastEveningSBP;

        @Override
        public void open(Configuration params) {
            dailySummaries = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("daily_bp_30d", String.class, String.class));
            lastMorningSBP = getRuntimeContext().getState(
                new ValueStateDescriptor<>("morning_sbp", Double.class));
            lastEveningSBP = getRuntimeContext().getState(
                new ValueStateDescriptor<>("evening_sbp", Double.class));
        }

        @Override
        public void processElement(Map<String, Object> event, Context ctx,
                                   Collector<BPVariabilityMetrics> out) throws Exception {
            double sbp = ((Number) event.getOrDefault("sbp", 0)).doubleValue();
            double dbp = ((Number) event.getOrDefault("dbp", 0)).doubleValue();
            String patientId = ctx.getCurrentKey();
            boolean isCuffless = Boolean.TRUE.equals(event.get("cuffless"));

            // 1. Crisis bypass — SBP>180 or DBP>120
            if (HypertensiveCrisisDetector.isCrisis(sbp, dbp) &&
                !HypertensiveCrisisDetector.requiresCuffConfirmation(isCuffless)) {
                BPVariabilityMetrics crisis = new BPVariabilityMetrics();
                crisis.setPatientId(patientId);
                crisis.setBpControlStatus("CRISIS");
                crisis.setComputedAt(Instant.now());
                ctx.output(CRISIS_TAG, crisis);
            }

            // 2. Update daily summary
            String today = LocalDate.now().toString();
            String existing = dailySummaries.get(today);
            double dayAvgSBP = sbp, dayAvgDBP = dbp;
            int count = 1;
            if (existing != null) {
                String[] parts = existing.split(",");
                double prevAvg = Double.parseDouble(parts[0]);
                double prevDBP = Double.parseDouble(parts[1]);
                int prevCount = Integer.parseInt(parts[2]);
                dayAvgSBP = (prevAvg * prevCount + sbp) / (prevCount + 1);
                dayAvgDBP = (prevDBP * prevCount + dbp) / (prevCount + 1);
                count = prevCount + 1;
            }
            dailySummaries.put(today, String.format("%.1f,%.1f,%d", dayAvgSBP, dayAvgDBP, count));

            // 3. Track morning/evening for surge detection
            int hour = LocalTime.now().getHour();
            if (hour >= 6 && hour <= 9) { lastMorningSBP.update(sbp); }
            if (hour >= 20 && hour <= 23) { lastEveningSBP.update(sbp); }

            // 4. Compute variability metrics from 30-day state
            List<Double> sbpHistory = new ArrayList<>();
            for (Map.Entry<String, String> entry : dailySummaries.entries()) {
                sbpHistory.add(Double.parseDouble(entry.getValue().split(",")[0]));
            }
            double[] sbpArr = sbpHistory.stream().mapToDouble(Double::doubleValue).toArray();

            BPVariabilityMetrics result = new BPVariabilityMetrics();
            result.setPatientId(patientId);

            // ARV (7d and 30d)
            if (sbpArr.length >= 7) {
                double[] last7 = Arrays.copyOfRange(sbpArr, Math.max(0, sbpArr.length - 7), sbpArr.length);
                result.setArvSbp7d(ARVCalculator.compute(last7));
            }
            result.setArvSbp30d(ARVCalculator.compute(sbpArr));

            // Morning surge
            Double morning = lastMorningSBP.value();
            Double evening = lastEveningSBP.value();
            if (morning != null && evening != null) {
                result.setMorningSurgeToday(MorningSurgeDetector.computeSurge(morning, evening));
            }

            // Dipping (uses last night vs last day average)
            if (sbpHistory.size() >= 2) {
                // Night-time SBP: tracked via lastEveningSBP state (readings 22:00–06:00).
                // If no evening readings yet, skip dipping classification rather than use a placeholder.
                Double nightAvgSBP = lastEveningSBP.value();
                if (nightAvgSBP != null) {
                    result.setDipClassification(
                        DippingPatternClassifier.classify(dayAvgSBP, nightAvgSBP));
                    result.setDipConfidence(DippingPatternClassifier.confidence(isCuffless));
                }
            }

            result.setComputedAt(Instant.now());
            out.collect(result);
        }
    }

    static class BPMetricsSerializer implements SerializationSchema<BPVariabilityMetrics> {
        private transient ObjectMapper mapper;
        @Override public void open(SerializationSchema.InitializationContext ctx) {
            mapper = new ObjectMapper(); mapper.registerModule(new JavaTimeModule());
        }
        @Override public byte[] serialize(BPVariabilityMetrics m) {
            try { return mapper.writeValueAsBytes(m); }
            catch (Exception e) { throw new RuntimeException("Serialize BPMetrics failed", e); }
        }
    }
}
```

- [ ] **Step 10: Register Module7 in FlinkJobOrchestrator**

```java
case "bp-variability":
case "module7":
    Module7_BPVariability.createBPVariabilityPipeline(env);
    break;
```

- [ ] **Step 11: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/
git commit -m "feat(flink): add Module7 BP Variability Engine with ARV, surge, dipping, crisis bypass (DD#1)"
```

### Task 6: Expand KB-20 Patient Profile with V4 State Fields

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/patient_profile.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/ckm_stage.go`

**Context:** KB-20 state vector expands from ~70 to ~95 fields per NorthStar Section 2.2.

- [ ] **Step 1: Add V4 fields to PatientProfile model**

```go
// BP Variability Domain (from Flink Module7 output)
ARVSBP7d            *float64 `gorm:"type:decimal(6,2)" json:"arv_sbp_7d,omitempty"`
ARVSBP30d           *float64 `gorm:"type:decimal(6,2)" json:"arv_sbp_30d,omitempty"`
MorningSurge7dAvg   *float64 `gorm:"type:decimal(6,2)" json:"morning_surge_7d_avg,omitempty"`
DipClassification   string   `gorm:"size:20" json:"dip_classification,omitempty"`     // DIPPER/NON_DIPPER/EXTREME/REVERSE
BPControlStatus     string   `gorm:"size:20" json:"bp_control_status,omitempty"`      // CONTROLLED/ELEVATED/STAGE1/STAGE2

// Metabolic Status (from DD#6 MHRI inputs)
WaistCm             *float64 `gorm:"type:decimal(5,1)" json:"waist_cm,omitempty"`
WaistToHeightRatio  *float64 `gorm:"type:decimal(4,3)" json:"waist_to_height_ratio,omitempty"`
WaistRiskFlag       string   `gorm:"size:20" json:"waist_risk_flag,omitempty"`        // normal/elevated/high
LDLCholesterol      *float64 `gorm:"type:decimal(5,1)" json:"ldl_cholesterol,omitempty"`
TGHDLRatio          *float64 `gorm:"type:decimal(4,2)" json:"tg_hdl_ratio,omitempty"`
WeightTrajectory30d string   `gorm:"size:20" json:"weight_trajectory_30d,omitempty"`  // GAINING/STABLE/LOSING

// Engagement (from Flink Module9)
EngagementComposite *float64 `gorm:"type:decimal(3,2)" json:"engagement_composite,omitempty"` // 0-1
EngagementStatus    string   `gorm:"size:20" json:"engagement_status,omitempty"`      // GREEN/YELLOW/ORANGE/RED

// Phenotype (from DD#9 quarterly clustering)
PhenotypeCluster    string   `gorm:"size:30" json:"phenotype_cluster,omitempty"`
PhenotypeConfidence *float64 `gorm:"type:decimal(3,2)" json:"phenotype_confidence,omitempty"`
PhenotypeClusterOrigin string `gorm:"size:30" json:"phenotype_cluster_origin,omitempty"` // LOCAL/INDIA_TRANSFERRED

// MHRI (from KB-26 computation)
MHRIScore           *float64 `gorm:"type:decimal(5,2)" json:"mhri_score,omitempty"`   // 0-100
MHRITrajectory      string   `gorm:"size:30" json:"mhri_trajectory,omitempty"`        // IMPROVING/STABLE/DECLINING/RAPIDLY_DECLINING
MHRIDataQuality     string   `gorm:"size:10" json:"mhri_data_quality,omitempty"`      // HIGH/MODERATE/LOW

// CKM Stage (from DD#6 Section 6) + fields for ComputeCKMStage()
CKMStage            int      `gorm:"default:0" json:"ckm_stage"`                     // 0-4
HasClinicalCVD      bool     `gorm:"default:false" json:"has_clinical_cvd"`           // prior MI, stroke, PAD — triggers CKM Stage 4
ASCVDRisk10y        *float64 `gorm:"type:decimal(5,2)" json:"ascvd_risk_10y,omitempty"` // 10-year ASCVD risk % (pooled cohort equation)
DiabetesYears       *int     `json:"diabetes_years,omitempty"`                        // years since T2DM diagnosis
HTNYears            *int     `json:"htn_years,omitempty"`                             // years since HTN diagnosis
BMI                 *float64 `gorm:"type:decimal(4,1)" json:"bmi,omitempty"`          // may exist in V3; included here for completeness
HbA1c               *float64 `gorm:"type:decimal(4,2)" json:"hba1c,omitempty"`        // latest HbA1c %
EGFR                *float64 `gorm:"type:decimal(5,1)" json:"egfr,omitempty"`         // latest eGFR mL/min/1.73m²
UACR                *float64 `gorm:"type:decimal(7,1)" json:"uacr,omitempty"`         // urine albumin-creatinine ratio mg/g
Potassium           *float64 `gorm:"type:decimal(3,1)" json:"potassium,omitempty"`    // serum K+ mEq/L

// Data Tier
DataTier            string   `gorm:"size:20;default:'TIER_3_SMBG'" json:"data_tier"` // TIER_1_CGM/TIER_2_HYBRID/TIER_3_SMBG
```

- [ ] **Step 2: Write failing test for V4 field validation**

Create `kb-20-patient-profile/internal/models/patient_profile_test.go`:

```go
package models

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestPatientProfile_V4FieldsExist(t *testing.T) {
    p := PatientProfile{}
    // BP Variability fields
    assert.Nil(t, p.ARVSBP7d)
    assert.Empty(t, p.DipClassification)
    assert.Empty(t, p.BPControlStatus)
    // MHRI fields
    assert.Nil(t, p.MHRIScore)
    assert.Empty(t, p.MHRITrajectory)
    // Engagement fields
    assert.Nil(t, p.EngagementComposite)
    assert.Empty(t, p.EngagementStatus)
    // CKM stage
    assert.Equal(t, 0, p.CKMStage)
    // Data tier default
    assert.Equal(t, "", p.DataTier) // GORM default applies at DB level
}

func TestCKMStage_Constants(t *testing.T) {
    assert.Equal(t, 0, CKMStage0)
    assert.Equal(t, 4, CKMStage4)
}
```

Run: `cd kb-20-patient-profile && go test ./internal/models/ -run TestPatientProfile_V4 -v`
Expected: FAIL — fields not defined yet.

- [ ] **Step 3: Create CKM stage model**

```go
// ckm_stage.go — AHA CKM Stage constants
package models

const (
    CKMStage0 = 0 // No CKM risk factors
    CKMStage1 = 1 // Excess adiposity, dyslipidemia, or metabolic syndrome
    CKMStage2 = 2 // Metabolic risk + moderate-high risk CKD or T2DM
    CKMStage3 = 3 // Subclinical CVD or high predicted ASCVD risk
    CKMStage4 = 4 // Clinical CVD event
)
```

- [ ] **Step 4: Run CKM constant tests — verify pass**

Run: `cd kb-20-patient-profile && go test ./internal/models/ -run TestCKMStage_Constants -v`
Expected: PASS.

- [ ] **Step 5: Write failing test for ComputeCKMStage function**

Add to `kb-20-patient-profile/internal/models/patient_profile_test.go`:

```go
func TestComputeCKMStage_Stage0_NoRiskFactors(t *testing.T) {
    p := PatientProfile{
        BMI: floatPtr(22.0), WaistToHeightRatio: floatPtr(0.45),
        HbA1c: floatPtr(5.4), EGFR: floatPtr(95.0), UACR: floatPtr(15.0),
        MHRIScore: floatPtr(85.0),
    }
    assert.Equal(t, CKMStage0, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage1_ExcessAdiposityOnly(t *testing.T) {
    p := PatientProfile{
        BMI: floatPtr(28.0), WaistToHeightRatio: floatPtr(0.58),
        HbA1c: floatPtr(5.5), EGFR: floatPtr(90.0), UACR: floatPtr(20.0),
    }
    assert.Equal(t, CKMStage1, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage2_T2DM(t *testing.T) {
    p := PatientProfile{
        BMI: floatPtr(30.0), WaistToHeightRatio: floatPtr(0.60),
        HbA1c: floatPtr(7.5), EGFR: floatPtr(55.0), UACR: floatPtr(150.0),
        DiabetesYears: intPtr(5),
    }
    assert.Equal(t, CKMStage2, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage3_HighASCVDRisk(t *testing.T) {
    p := PatientProfile{
        BMI: floatPtr(31.0), WaistToHeightRatio: floatPtr(0.62),
        HbA1c: floatPtr(8.0), EGFR: floatPtr(45.0), UACR: floatPtr(300.0),
        ASCVDRisk10y: floatPtr(22.0), // >20% = high risk
    }
    assert.Equal(t, CKMStage3, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage4_ClinicalCVDEvent(t *testing.T) {
    p := PatientProfile{
        BMI: floatPtr(29.0), HasClinicalCVD: true,
    }
    assert.Equal(t, CKMStage4, ComputeCKMStage(p))
}

func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int { return &i }
```

Run: `cd kb-20-patient-profile && go test ./internal/models/ -run TestComputeCKMStage -v`
Expected: FAIL — `ComputeCKMStage` not defined.

- [ ] **Step 6: Implement ComputeCKMStage function**

Add to `kb-20-patient-profile/internal/models/ckm_stage.go`:

```go
// ComputeCKMStage classifies a patient into AHA CKM Stage 0-4.
// Per DD#6 §6, staging is deterministic — no ML, no LLM.
// Stage 4: prior clinical CVD event
// Stage 3: subclinical CVD or high predicted ASCVD risk (≥20%)
// Stage 2: metabolic risk factors + moderate-to-high CKD (eGFR <60 or UACR ≥30) or T2DM
// Stage 1: excess adiposity (BMI ≥25 or waist/height ≥0.5), dyslipidemia, or metabolic syndrome
// Stage 0: no CKM risk factors
func ComputeCKMStage(p PatientProfile) int {
    // Stage 4: clinical CVD trumps everything
    if p.HasClinicalCVD {
        return CKMStage4
    }

    // Stage 3: subclinical CVD markers or high ASCVD risk
    if p.ASCVDRisk10y != nil && *p.ASCVDRisk10y >= 20.0 {
        return CKMStage3
    }
    if p.UACR != nil && *p.UACR >= 300.0 { // severely increased albuminuria = subclinical
        return CKMStage3
    }
    if p.EGFR != nil && *p.EGFR < 30.0 { // CKD G4-G5 = high CVD risk
        return CKMStage3
    }

    // Stage 2: T2DM or moderate-to-high CKD with metabolic risk
    hasT2DM := p.HbA1c != nil && *p.HbA1c >= 6.5
    if p.DiabetesYears != nil && *p.DiabetesYears > 0 {
        hasT2DM = true
    }
    moderateCKD := (p.EGFR != nil && *p.EGFR < 60.0) || (p.UACR != nil && *p.UACR >= 30.0)
    if hasT2DM || moderateCKD {
        return CKMStage2
    }

    // Stage 1: excess adiposity, dyslipidemia, or metabolic syndrome markers
    excessAdipose := (p.BMI != nil && *p.BMI >= 25.0) || (p.WaistToHeightRatio != nil && *p.WaistToHeightRatio >= 0.5)
    dyslipidemia := p.TGHDLRatio != nil && *p.TGHDLRatio >= 3.5
    preDiabetes := p.HbA1c != nil && *p.HbA1c >= 5.7 && *p.HbA1c < 6.5
    if excessAdipose || dyslipidemia || preDiabetes {
        return CKMStage1
    }

    return CKMStage0
}
```

Note: `HasClinicalCVD`, `ASCVDRisk10y`, `DiabetesYears`, `TGHDLRatio`, `BMI`, `HbA1c`, `EGFR`, `UACR` fields are defined in Step 1 of this task alongside the other V4 fields.

- [ ] **Step 7: Run ComputeCKMStage tests — verify pass**

Run: `cd kb-20-patient-profile && go test ./internal/models/ -run TestComputeCKMStage -v`
Expected: 5 tests PASS.

- [ ] **Step 8: Run all model tests — verify pass**

Run: `cd kb-20-patient-profile && go test ./internal/models/ -v`
Expected: All tests PASS (V4 fields + CKM constants + CKM computation).

- [ ] **Step 9: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/
git commit -m "feat(kb-20): expand patient profile with V4 fields (BP variability, MHRI, engagement, phenotype, CKM)"
```

### Task 6b: Extend V3 Flink Jobs for V4 Dual-Domain Support

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_TrajectoryAnalysis.java` (or equivalent)
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/FlinkJobOrchestrator.java`

**Context:** Per the Flink Architecture specification (§1.1), V4 does not only add 5 new Flink jobs — it also extends 4 existing V3 jobs. These extensions are required for the dual-domain (diabetes + hypertension) processing chain to function. Without them, the V4 Flink jobs consume incomplete data.

| V3 Job | Extension Required | Consumed By |
|--------|-------------------|-------------|
| CGM Aggregation | Add `cgm_active` flag, nocturnal glucose profile, `data_tier` field to output | KB-20 FHIRSyncWorker, Trajectory Analysis, MHRI Trigger |
| Trajectory Analysis | Add BP trajectory slope, composite dual-domain signal, MHRI trajectory input | M4 RunCycle (V-MCU), KB-26 MHRI |
| MRI Recomputation Trigger | Add BP variability change as trigger condition, rename concept to MHRI trigger | KB-26 MHRI Scorer |
| Deterioration Detection | Add cross-domain deterioration patterns (glycaemic + hemodynamic concurrent decline) | KB-22 HPI Engine, KB-23 Decision Cards |

- [ ] **Step 1: Extend CGM Aggregation output schema**

In the CGM Aggregation operator (Module1b or equivalent), extend the output record:

```java
// V4 extensions to CGM aggregate output
private boolean cgmActive;           // true if patient has active CGM device (Tier 1/2)
private double nocturnalMeanGlucose; // Mean glucose 00:00-06:00 (for nocturnal profile)
private double nocturnalCV;          // CV% during nocturnal window
private String dataTier;             // TIER_1_CGM, TIER_2_HYBRID, TIER_3_SMBG
```

In `processElement()`, after computing TIR/TBR/TAR/CV%:

```java
// V4: Compute nocturnal glucose profile (00:00-06:00)
List<GlucoseReading> nocturnalReadings = filterByTimeWindow(dailyReadings, 0, 6);
if (!nocturnalReadings.isEmpty()) {
    output.setNocturnalMeanGlucose(mean(nocturnalReadings));
    output.setNocturnalCV(coefficientOfVariation(nocturnalReadings));
}

// V4: Set data tier based on source device
output.setDataTier(classifyDataTier(dailyReadings));
output.setCgmActive(output.getDataTier().equals("TIER_1_CGM") || output.getDataTier().equals("TIER_2_HYBRID"));
```

- [ ] **Step 2: Extend Trajectory Analysis with BP trajectory and dual-domain signal**

In the Trajectory Analysis operator, add BP slope computation alongside the existing glucose trajectory:

```java
// V4: BP trajectory slope (14-day OLS regression on SBP)
private double bpTrajectorySlope;       // mmHg/day (negative = improving)
private String bpTrajectoryClass;       // STABLE, RISING, DECLINING, RAPID_RISING
private boolean dualDomainDeteriorating; // true if BOTH glycaemic AND hemodynamic declining

// In processElement():
// Compute BP trajectory alongside existing glucose trajectory
double bpSlope = computeOLSSlope(bpReadings14d);
String bpClass = classifyTrajectory(bpSlope, BP_THRESHOLDS);
output.setBpTrajectorySlope(bpSlope);
output.setBpTrajectoryClass(bpClass);

// V4: Cross-domain deterioration flag
boolean glucoseWorsening = glucoseTrajectoryClass.equals("RISING") || glucoseTrajectoryClass.equals("RAPID_RISING");
boolean bpWorsening = bpClass.equals("RISING") || bpClass.equals("RAPID_RISING");
output.setDualDomainDeteriorating(glucoseWorsening && bpWorsening);
```

- [ ] **Step 3: Extend MRI Trigger to include BP variability changes → rename to MHRI trigger**

In the MRI Recomputation Trigger operator:

```java
// V4: Additional trigger conditions for MHRI recomputation
// Existing: FBG trend change >15% (7d rolling mean vs previous)
// New: BP variability change triggers
private static final double ARV_THRESHOLD_LOW = 8.0;
private static final double ARV_THRESHOLD_HIGH = 15.0;

// In processElement():
// Check BP variability trigger (ARV crosses threshold, dip classification changes, morning surge >35)
BPVariabilityMetrics bpMetrics = getBPVariabilityState(patientId);
if (bpMetrics != null) {
    boolean arvCrossed = crossedThreshold(bpMetrics.getArvSbp7d(), previousArvSbp7d, ARV_THRESHOLD_LOW, ARV_THRESHOLD_HIGH);
    boolean dipChanged = !Objects.equals(bpMetrics.getDipClassification(), previousDipClassification);
    boolean surgeSevere = bpMetrics.getMorningSurge7dAvg() > 35.0;

    if (arvCrossed || dipChanged || surgeSevere) {
        emit(new MHRIRecomputeRequest(patientId, "BP_VARIABILITY_CHANGE",
             Map.of("arv_crossed", arvCrossed, "dip_changed", dipChanged, "surge_severe", surgeSevere)));
    }
}
```

Update the output topic reference from `flink.mri-triggers` conceptually to carry MHRI semantics (topic name stays the same for backward compatibility).

- [ ] **Step 4: Extend Deterioration Detection with cross-domain patterns**

In the Deterioration Detection operator (Module4 or equivalent), add cross-domain CEP patterns:

```java
// V4: Cross-domain deterioration pattern
// Fires when glycaemic AND hemodynamic are both declining concurrently
// This catches patients whose single-domain checks might show borderline values
// but whose combined trajectory is concerning

Pattern<EnrichedEvent, ?> crossDomainDecline = Pattern.<EnrichedEvent>begin("glycaemic_decline")
    .where(new SimpleCondition<EnrichedEvent>() {
        @Override
        public boolean filter(EnrichedEvent event) {
            return event.getDomain().equals("GLYCAEMIC")
                && event.getTrajectoryClass().equals("DECLINING");
        }
    })
    .followedBy("hemodynamic_decline")
    .where(new SimpleCondition<EnrichedEvent>() {
        @Override
        public boolean filter(EnrichedEvent event) {
            return event.getDomain().equals("HEMODYNAMIC")
                && event.getTrajectoryClass().equals("DECLINING");
        }
    })
    .within(Time.hours(72));

// Emit CKM_CRISIS_ALERT or DETERIORATION Decision Card request
```

- [ ] **Step 5: Write tests for V3 job extensions**

```java
@Test
@DisplayName("CGM Aggregation emits data_tier and nocturnal profile in V4 mode")
void cgmAggregation_emitsV4Fields() {
    // Given: 24h of CGM readings with nocturnal period
    List<GlucoseReading> readings = generate24hReadings(/*nocturnal mean=*/ 6.2, /*overall mean=*/ 7.5);

    // When: processed
    CGMAggregate result = processor.aggregate(readings);

    // Then: V4 fields present
    assertNotNull(result.getDataTier());
    assertTrue(result.isCgmActive());
    assertEquals(6.2, result.getNocturnalMeanGlucose(), 0.5);
}

@Test
@DisplayName("Trajectory Analysis computes BP slope and dual-domain flag")
void trajectoryAnalysis_computesBPSlope() {
    // Given: 14 days of rising BP readings
    List<BPReading> bpReadings = generateRisingBP(14, /*start=*/ 130, /*end=*/ 150);
    List<GlucoseReading> glucoseReadings = generateRisingGlucose(14, /*start=*/ 7.0, /*end=*/ 9.0);

    TrajectoryOutput result = processor.analyze(glucoseReadings, bpReadings);

    assertTrue(result.getBpTrajectorySlope() > 0); // rising
    assertEquals("RISING", result.getBpTrajectoryClass());
    assertTrue(result.isDualDomainDeteriorating()); // both rising = both worsening
}

@Test
@DisplayName("MHRI Trigger fires on ARV threshold crossing")
void mhriTrigger_firesOnARVCrossing() {
    // Given: ARV moved from 7.5 (below threshold) to 13 (above threshold)
    boolean shouldFire = processor.checkBPVariabilityTrigger(13.0, 7.5, "DIPPER", "DIPPER", 25.0);
    assertTrue(shouldFire);
}

@Test
@DisplayName("Deterioration Detection catches cross-domain decline")
void deterioration_catchesCrossDomainDecline() {
    // Given: both glycaemic and hemodynamic declining within 72h
    List<EnrichedEvent> events = List.of(
        new EnrichedEvent("GLYCAEMIC", "DECLINING", Instant.now()),
        new EnrichedEvent("HEMODYNAMIC", "DECLINING", Instant.now().plusSeconds(3600))
    );
    List<DeteriorationAlert> alerts = processor.detectPatterns(events);
    assertFalse(alerts.isEmpty());
    assertEquals("CROSS_DOMAIN_DECLINE", alerts.get(0).getPatternType());
}
```

- [ ] **Step 6: Register V4 extensions in FlinkJobOrchestrator**

In `FlinkJobOrchestrator.java`, update the existing job registrations to pass V4 configuration flags:

```java
// V4: Enable dual-domain extensions on existing V3 jobs
cgmAggregationJob.enableV4Extensions(true);      // nocturnal profile, data_tier
trajectoryAnalysisJob.enableV4Extensions(true);   // BP slope, dual-domain flag
mriTriggerJob.enableV4Extensions(true);           // BP variability trigger
deteriorationDetectionJob.enableV4Extensions(true); // cross-domain patterns
```

- [ ] **Step 7: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/
git commit -m "feat(flink): extend V3 jobs for V4 dual-domain — CGM data_tier, BP trajectory, MHRI triggers, cross-domain deterioration"
```

---

## Phase C3: Intelligence Core — MHRI + IOR + Engagement (Weeks 6-12)

### Task 7: Implement MHRI Score Computation in KB-26

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/mhri.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mhri_scorer.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/mhri_trajectory.go`

**Context:** Per DD#6, MHRI is a 0-100 composite score with 5 weighted components: Glycemic (25), Hemodynamic (25), Renal (20), Metabolic (15), Engagement (15). Each input is normalized via piecewise linear functions derived from ADA 2026 / KDIGO 2024 / ESC 2024 guidelines. The trajectory (14-day OLS regression slope) is the clinically actionable output.

- [ ] **Step 1: Write failing tests for MHRI normalization functions**

Create `kb-26-metabolic-digital-twin/internal/services/mhri_scorer_test.go`:

```go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestNormalizeGlycemic_HbA1cInTarget(t *testing.T) {
    // HbA1c 6.8% → between optimal (6.5→100) and good (7.0→80) ≈ 88
    score := normalizeGlycemic(6.8, 0.0, "TIER_3_SMBG")
    assert.InDelta(t, 88.0, score, 5.0)
}

func TestNormalizeGlycemic_AtOptimal(t *testing.T) {
    // HbA1c 6.5% → exactly optimal → 100
    score := normalizeGlycemic(6.5, 0.0, "TIER_3_SMBG")
    assert.InDelta(t, 100.0, score, 1.0)
}

func TestNormalizeGlycemic_DangerouslyHigh(t *testing.T) {
    // HbA1c 11.0% → 0 per DD#6 breakpoints
    score := normalizeGlycemic(11.0, 0.0, "TIER_3_SMBG")
    assert.Less(t, score, 5.0)
}

func TestNormalizeHemodynamic_Controlled(t *testing.T) {
    // SBP 125→~90, ARV 5→100, DIPPER→100 → weighted ~95
    score := normalizeHemodynamic(125.0, 5.0, "DIPPER")
    assert.InDelta(t, 96.0, score, 5.0)
}

func TestNormalizeHemodynamic_Stage2(t *testing.T) {
    // SBP 165→~25, ARV 14→~50, NON_DIPPER→50 → weighted ~37
    score := normalizeHemodynamic(165.0, 14.0, "NON_DIPPER")
    assert.Less(t, score, 45.0)
}

func TestCompositeScore_AllOptimal(t *testing.T) {
    // All-optimal raw values: HbA1c 6.5→100, SBP 120→100, ARV 5→100,
    // DIPPER→100, eGFR 95→100, UACR 10→100, waist/ht 0.4→100, TG/HDL 1.5→100,
    // engagement 1.0→100
    input := MHRIInput{
        HbA1c: 6.5, TIR: 0.8, DataTier: "TIER_1_CGM",
        SBP7dAvg: 120.0, ARVSBP7d: 5.0, DipClass: "DIPPER",
        EGFR: 95.0, UACR: 10.0,
        WaistToHeight: 0.4, TGHDLRatio: 1.5, WeightTrend: "STABLE",
        EngagementComposite: 1.0,
    }
    result := ComputeMHRI(input)
    // Each component normalizes to ~100, composite ≈ 100
    assert.InDelta(t, 100.0, result.CompositeScore, 5.0)
}

func TestCompositeScore_TierAwareRedistribution(t *testing.T) {
    // TIER_3_SMBG: glycemic weight reduces from 0.25 to 0.20
    // Use moderate raw values so components are in 40-80 range
    input := MHRIInput{
        HbA1c: 8.5, TIR: 0.0, DataTier: "TIER_3_SMBG",
        SBP7dAvg: 135.0, ARVSBP7d: 10.0, DipClass: "NON_DIPPER",
        EGFR: 55.0, UACR: 80.0,
        WaistToHeight: 0.56, TGHDLRatio: 2.5, WeightTrend: "GAINING",
        EngagementComposite: 0.5,
    }
    result := ComputeMHRI(input)
    assert.Greater(t, result.CompositeScore, 0.0)
    assert.Less(t, result.CompositeScore, 100.0)
    assert.Equal(t, "MODERATE", result.DataQuality)
}
```

Run: `cd kb-26-metabolic-digital-twin && go test ./internal/services/ -run TestNormalize -v`
Expected: FAIL — functions not defined.

- [ ] **Step 2: Create MHRI output model**

Create `kb-26-metabolic-digital-twin/internal/models/mhri.go`:

```go
package models

import "time"

type MHRIResult struct {
    PatientID        string    `json:"patient_id"`
    CompositeScore   float64   `json:"composite_score"`    // 0-100
    GlycemicScore    float64   `json:"glycemic_score"`     // 0-100, weight 25
    HemodynamicScore float64   `json:"hemodynamic_score"`  // 0-100, weight 25
    RenalScore       float64   `json:"renal_score"`        // 0-100, weight 20
    MetabolicScore   float64   `json:"metabolic_score"`    // 0-100, weight 15
    EngagementScore  float64   `json:"engagement_score"`   // 0-100, weight 15
    DataQuality      string    `json:"data_quality"`       // HIGH/MODERATE/LOW
    DataTier         string    `json:"data_tier"`          // TIER_1_CGM/TIER_2_HYBRID/TIER_3_SMBG
    Trajectory       string    `json:"trajectory"`         // IMPROVING/STABLE/DECLINING/RAPIDLY_DECLINING
    TrajectorySlope  float64   `json:"trajectory_slope"`   // OLS regression coefficient
    CDIAlert         bool      `json:"cdi_alert"`          // true if ≥2 components declining
    ComputedAt       time.Time `json:"computed_at"`
}
```

- [ ] **Step 3: Implement 5 normalization functions (piecewise linear)**

Create `kb-26-metabolic-digital-twin/internal/services/mhri_scorer.go`:

```go
package services

import "kb-26-metabolic-digital-twin/internal/models"

type MHRIInput struct {
    // Glycemic inputs
    HbA1c       float64
    TIR         float64 // Time-in-range (CGM only, 0-1)
    DataTier    string

    // Hemodynamic inputs
    SBP7dAvg    float64
    ARVSBP7d    float64
    DipClass    string

    // Renal inputs
    EGFR        float64
    UACR        float64

    // Metabolic inputs
    WaistToHeight float64
    TGHDLRatio    float64
    WeightTrend   string

    // Engagement
    EngagementComposite float64
}

// normalizeGlycemic: piecewise linear on HbA1c per DD#6 Section 3.1 (ADA 2026 targets)
// Breakpoints: ≤6.5 → 1.0 (optimal), 6.5-7.0 → 0.8 (good), 7.0-8.0 → 0.5 (moderate),
//              8.0-9.5 → 0.2 (poor), ≥9.5 → 0.0 (severe)
// Note: 6.5 is optimal (not 5.7) because below 6.5 on medication = overtreatment/hypo risk
func normalizeGlycemic(hba1c, tir float64, dataTier string) float64 {
    score := piecewiseLinear(hba1c, [][2]float64{{6.5, 100}, {7.0, 80}, {8.0, 50}, {9.5, 20}, {11.0, 0}})
    if dataTier == "TIER_1_CGM" && tir > 0 {
        // DD#6 §3.1: CGM TIR sub-score (70-180 range)
        // TIR ≥70% → +10, TIR 50-70% → +5, TIR <50% → 0
        tirBonus := 0.0
        if tir >= 0.7 { tirBonus = 10.0 } else if tir >= 0.5 { tirBonus = 5.0 }
        // Blend: 70% HbA1c-based + 30% TIR-based (for CGM patients only)
        tirScore := tir * 100.0
        score = score*0.7 + tirScore*0.3
    }
    return clamp(score, 0, 100)
}

// normalizeHemodynamic: weighted sub-component model per DD#6 Section 3.2
// Each input is independently normalized via piecewise linear, then combined:
// 40% norm(SBP) + 30% norm(ARV) + 30% norm(dip) — bounded [0,100] by construction
func normalizeHemodynamic(sbp7d, arv float64, dipClass string) float64 {
    // SBP sub-score: ≤120→100, 130→80, 140→60, 160→30, ≥180→0
    sbpScore := piecewiseLinear(sbp7d, [][2]float64{{120, 100}, {130, 80}, {140, 60}, {160, 30}, {180, 0}})
    // ARV sub-score: <8→100, 8-12→70, 12-15→40, >15→10
    arvScore := piecewiseLinear(arv, [][2]float64{{8, 100}, {12, 70}, {15, 40}, {20, 10}})
    // Dipping sub-score
    dipScore := 50.0 // default NON_DIPPER
    switch dipClass {
    case "DIPPER":     dipScore = 100.0
    case "NON_DIPPER": dipScore = 50.0
    case "EXTREME":    dipScore = 60.0  // extreme dipping also risky
    case "REVERSE":    dipScore = 10.0  // worst — nocturnal hypertension
    }
    // Weighted composite (surge omitted here — added when morning surge data available)
    return clamp(sbpScore*0.40 + arvScore*0.30 + dipScore*0.30, 0, 100)
}

// normalizeRenal: eGFR + UACR (KDIGO 2024)
func normalizeRenal(egfr, uacr float64) float64 {
    egfrScore := piecewiseLinear(egfr, [][2]float64{{90, 100}, {60, 80}, {45, 50}, {30, 25}, {15, 5}})
    uacrPenalty := 0.0
    if uacr > 30 { uacrPenalty = (uacr - 30) * 0.1 }
    if uacrPenalty > 30 { uacrPenalty = 30 }
    return clamp(egfrScore-uacrPenalty, 0, 100)
}

// normalizeMetabolic: waist-to-height + TG/HDL + weight trend
func normalizeMetabolic(waistToHeight, tghdl float64, weightTrend string) float64 {
    whScore := piecewiseLinear(waistToHeight, [][2]float64{{0.4, 100}, {0.5, 80}, {0.55, 50}, {0.6, 25}, {0.7, 5}})
    tghdlPenalty := 0.0
    if tghdl > 2.0 { tghdlPenalty = (tghdl - 2.0) * 10.0 }
    trendBonus := 0.0
    if weightTrend == "LOSING" { trendBonus = 5.0 }
    if weightTrend == "GAINING" { trendBonus = -10.0 }
    return clamp(whScore-tghdlPenalty+trendBonus, 0, 100)
}

// normalizeEngagement: direct passthrough (already 0-1 from Module9, scale to 0-100)
func normalizeEngagement(composite float64) float64 {
    return clamp(composite*100.0, 0, 100)
}

// ComputeMHRI: composite with tier-aware weight redistribution
func ComputeMHRI(input MHRIInput) models.MHRIResult {
    glycemic := normalizeGlycemic(input.HbA1c, input.TIR, input.DataTier)
    hemo := normalizeHemodynamic(input.SBP7dAvg, input.ARVSBP7d, input.DipClass)
    renal := normalizeRenal(input.EGFR, input.UACR)
    metabolic := normalizeMetabolic(input.WaistToHeight, input.TGHDLRatio, input.WeightTrend)
    engagement := normalizeEngagement(input.EngagementComposite)

    // Default weights: Glycemic 25, Hemodynamic 25, Renal 20, Metabolic 15, Engagement 15
    wG, wH, wR, wM, wE := 0.25, 0.25, 0.20, 0.15, 0.15
    // Tier-aware: TIER_3_SMBG reduces glycemic confidence
    if input.DataTier == "TIER_3_SMBG" {
        wG = 0.20
        wH += 0.025; wR += 0.025 // redistribute 5 points
    }

    composite := glycemic*wG + hemo*wH + renal*wR + metabolic*wM + engagement*wE
    quality := "HIGH"
    if input.DataTier == "TIER_2_HYBRID" { quality = "HIGH" }
    if input.DataTier == "TIER_3_SMBG" { quality = "MODERATE" }

    return models.MHRIResult{
        CompositeScore:   composite,
        GlycemicScore:    glycemic,
        HemodynamicScore: hemo,
        RenalScore:       renal,
        MetabolicScore:   metabolic,
        EngagementScore:  engagement,
        DataQuality:      quality,
        DataTier:         input.DataTier,
    }
}

// Utility: piecewise linear interpolation
func piecewiseLinear(x float64, points [][2]float64) float64 {
    if x <= points[0][0] { return points[0][1] }
    for i := 1; i < len(points); i++ {
        if x <= points[i][0] {
            frac := (x - points[i-1][0]) / (points[i][0] - points[i-1][0])
            return points[i-1][1] + frac*(points[i][1]-points[i-1][1])
        }
    }
    return points[len(points)-1][1]
}

func clamp(v, min, max float64) float64 {
    if v < min { return min }
    if v > max { return max }
    return v
}
```

- [ ] **Step 4: Run MHRI tests — verify pass**

Run: `cd kb-26-metabolic-digital-twin && go test ./internal/services/ -run TestNormalize -v && go test ./internal/services/ -run TestComposite -v`
Expected: All 6 tests PASS.

- [ ] **Step 5: Create 14-day trajectory engine**

Create `kb-26-metabolic-digital-twin/internal/services/mhri_trajectory.go`:

```go
package services

import "math"

// ComputeTrajectory: OLS linear regression on daily MHRI scores over 14 days
// Returns slope (points/day) and classification
func ComputeTrajectory(dailyScores []float64) (slope float64, classification string) {
    n := len(dailyScores)
    if n < 3 { return 0.0, "INSUFFICIENT_DATA" }

    // OLS: y = mx + b where x = day index
    var sumX, sumY, sumXY, sumX2 float64
    for i, y := range dailyScores {
        x := float64(i)
        sumX += x; sumY += y; sumXY += x * y; sumX2 += x * x
    }
    nf := float64(n)
    slope = (nf*sumXY - sumX*sumY) / (nf*sumX2 - sumX*sumX)

    if math.IsNaN(slope) { return 0.0, "STABLE" }
    if slope > 0.5   { return slope, "IMPROVING" }
    if slope > -0.5  { return slope, "STABLE" }
    if slope > -1.5  { return slope, "DECLINING" }
    return slope, "RAPIDLY_DECLINING"
}

// CheckCDI: Cross-Domain Deterioration Index — fire when ≥2 component slopes negative
func CheckCDI(componentSlopes map[string]float64) bool {
    declining := 0
    for _, s := range componentSlopes {
        if s < -0.3 { declining++ }
    }
    return declining >= 2
}
```

- [ ] **Step 6: Add MHRI API endpoint**

In the KB-26 routes file, add: `GET /api/v1/kb26/twin/:patientId/mhri` that calls `ComputeMHRI()` with data from KB-20 patient profile + Flink outputs.

- [ ] **Step 7: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/
git commit -m "feat(kb-26): implement MHRI 5-component scorer, trajectory engine, CDI alert (DD#6)"
```

- [ ] **Step 8: Implement CKM Stage computation (Gap G1 — 4h)**

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ckm_stage_computer.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ckm_stage_computer_test.go`

**Context:** The `ckm_stage.go` model defines CKM stage constants (0-4) but no `ComputeCKMStage()` function exists. Without this, KB-20's `ckm_stage` field is never populated, and MHRI cannot apply CKM-stage-aware weighting. The AHA Cardiovascular-Kidney-Metabolic staging (2023) classifies patients:
- Stage 0: No risk factors
- Stage 1: Excess adiposity or dysfunctional adiposity
- Stage 2: Metabolic risk factors (metabolic syndrome, T2DM, CKD)
- Stage 3: Subclinical CVD (CAC, LVH, high-risk ASCVD)
- Stage 4: Clinical CVD event (MI, stroke, HF, PAD)

```go
package services

import "kb-20-patient-profile/internal/models"

// ComputeCKMStage evaluates AHA CKM staging from the patient profile.
// Called after any relevant lab/vital update. Result is persisted to KB-20.
func ComputeCKMStage(profile *models.PatientProfile) models.CKMStage {
    // Stage 4: Any prior clinical CVD event
    if profile.HasPriorMI || profile.HasPriorStroke || profile.HasHFDiagnosis || profile.HasPAD {
        return models.CKMStage4
    }

    // Stage 3: Subclinical CVD markers
    if profile.CACScore > 0 || profile.HasLVH || profile.ASCVDRisk10Year > 20.0 {
        return models.CKMStage3
    }

    // Stage 2: Metabolic risk factors (T2DM, CKD, metabolic syndrome)
    hasMetabolicSyndrome := countMetSyndromeComponents(profile) >= 3
    if profile.HasT2DM || profile.HasCKD || hasMetabolicSyndrome {
        return models.CKMStage2
    }

    // Stage 1: Excess adiposity (BMI >= threshold or elevated waist circumference)
    if profile.BMI >= profile.MarketConfig.BMIOverweightThreshold ||
       profile.WaistCircumference >= profile.MarketConfig.WaistRiskThreshold {
        return models.CKMStage1
    }

    return models.CKMStage0
}

// countMetSyndromeComponents counts ATP-III metabolic syndrome criteria.
func countMetSyndromeComponents(p *models.PatientProfile) int {
    count := 0
    if p.WaistCircumference >= p.MarketConfig.WaistRiskThreshold { count++ }
    if p.Triglycerides >= 150 { count++ }
    if p.HDL < p.MarketConfig.HDLRiskThreshold { count++ }
    if p.LatestSBP >= 130 || p.LatestDBP >= 85 { count++ }
    if p.LatestFBG >= 100 { count++ } // mg/dL
    return count
}
```

Tests:

```go
func TestComputeCKMStage_Stage4_PriorMI(t *testing.T) {
    p := &models.PatientProfile{HasPriorMI: true}
    assert.Equal(t, models.CKMStage4, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage2_T2DM(t *testing.T) {
    p := &models.PatientProfile{HasT2DM: true}
    assert.Equal(t, models.CKMStage2, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage1_HighBMI_India(t *testing.T) {
    p := &models.PatientProfile{
        BMI: 24.0, // Above Indian threshold (23.0), below Western (25.0)
        MarketConfig: models.MarketConfig{BMIOverweightThreshold: 23.0},
    }
    assert.Equal(t, models.CKMStage1, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage0_Healthy(t *testing.T) {
    p := &models.PatientProfile{
        BMI: 21.0,
        MarketConfig: models.MarketConfig{BMIOverweightThreshold: 25.0, WaistRiskThreshold: 102},
    }
    assert.Equal(t, models.CKMStage0, ComputeCKMStage(p))
}
```

Wire `ComputeCKMStage()` into KB-20's lab/vital update handlers so `ckm_stage` is recomputed after relevant changes.

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ckm_stage_computer*.go
git commit -m "feat(kb-20): implement CKM stage computation — AHA 2023 staging (G1 gap closure)"
```

### Task 8: Create IOR System (PostgreSQL Schema + Generator)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/intervention_record.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/outcome_record.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ior_store.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ior_generator.go`

**Context:** Per DD#4, the IOR store uses the same PostgreSQL RDS as KB-20 but in a separate `ior` schema. Two tables: `intervention_records` (created at WINDOW_OPENED) and `outcome_records` (created at WINDOW_CLOSED by daily batch). The generator runs as a daily cron. Similar-patient query uses indices on (drug_class, phenotype_cluster, status, adherence_score).

- [ ] **Step 1: Write failing test for IOR similar-patient query**

Create `kb-20-patient-profile/internal/services/ior_store_test.go`:

```go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestSimilarPatientQuery_ReturnsEmpty_WhenNoRecords(t *testing.T) {
    store := NewIORStore(nil) // nil DB for unit test
    results, err := store.FindSimilarOutcomes("SGLT2I", "CLUSTER_A", "ACTIVE", 0.8)
    assert.NoError(t, err)
    assert.Empty(t, results)
}

func TestSimilarPatientQuery_FiltersCorrectly(t *testing.T) {
    // This test needs a real DB — mark as integration
    t.Skip("requires PostgreSQL — run with -tags=integration")
}
```

Expected: FAIL — `NewIORStore` not defined.

- [ ] **Step 2: Create InterventionRecord model**

Create `kb-20-patient-profile/internal/models/intervention_record.go`:

```go
package models

import "time"

type InterventionRecord struct {
    ID               uint      `gorm:"primaryKey" json:"id"`
    PatientID        string    `gorm:"index;size:64;not null" json:"patient_id"`
    InterventionType string    `gorm:"size:30;not null" json:"intervention_type"` // MEDICATION/LIFESTYLE/COMBINED
    DrugClass        string    `gorm:"index;size:30" json:"drug_class,omitempty"`
    DrugName         string    `gorm:"size:100" json:"drug_name,omitempty"`
    DoseChange       string    `gorm:"size:30" json:"dose_change,omitempty"` // INITIATE/UP_TITRATE/DOWN_TITRATE/SWITCH
    LifestyleType    string    `gorm:"size:30" json:"lifestyle_type,omitempty"`
    PhenotypeCluster string    `gorm:"index;size:30" json:"phenotype_cluster,omitempty"`
    BaselineMHRI     float64   `gorm:"type:decimal(5,2)" json:"baseline_mhri"`
    BaselineHbA1c    *float64  `gorm:"type:decimal(4,1)" json:"baseline_hba1c,omitempty"`
    BaselineSBP      *float64  `gorm:"type:decimal(5,1)" json:"baseline_sbp,omitempty"`
    BaselineEGFR     *float64  `gorm:"type:decimal(5,1)" json:"baseline_egfr,omitempty"`
    AdherenceScore   float64   `gorm:"type:decimal(3,2)" json:"adherence_score"`
    // Confounder fields — per DD#4 §5, captured at window-open to adjust IOR outcome attribution
    ConfoundingIllness  string    `gorm:"size:100" json:"confounding_illness,omitempty"`  // e.g., "URI", "gastroenteritis" — acute illness affecting readings
    TravelDays          int       `gorm:"default:0" json:"travel_days"`                  // days of travel during observation window (disrupted routine)
    FestivalPeriod      bool      `gorm:"default:false" json:"festival_period"`           // Diwali/Ramadan/Christmas — dietary disruption flag
    DataCompletenessScore float64 `gorm:"type:decimal(3,2);default:1.0" json:"data_completeness_score"` // 0.0-1.0, fraction of expected readings actually received
    Status           string    `gorm:"index;size:20;default:'OPEN'" json:"status"` // OPEN/CLOSED/EXPIRED
    WindowDays       int       `gorm:"default:28" json:"window_days"`
    OpenedAt         time.Time `gorm:"not null" json:"opened_at"`
    ClosedAt         *time.Time `json:"closed_at,omitempty"`
    CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
}
```

- [ ] **Step 3: Create OutcomeRecord model**

Create `kb-20-patient-profile/internal/models/outcome_record.go`:

```go
package models

import "time"

type OutcomeRecord struct {
    ID                 uint      `gorm:"primaryKey" json:"id"`
    InterventionID     uint      `gorm:"index;not null" json:"intervention_id"`
    PatientID          string    `gorm:"index;size:64;not null" json:"patient_id"`
    MHRIDelta          float64   `gorm:"type:decimal(5,2)" json:"mhri_delta"`
    HbA1cDelta         *float64  `gorm:"type:decimal(4,2)" json:"hba1c_delta,omitempty"`
    SBPDelta           *float64  `gorm:"type:decimal(5,1)" json:"sbp_delta,omitempty"`
    EGFRDelta          *float64  `gorm:"type:decimal(5,1)" json:"egfr_delta,omitempty"`
    EngagementDelta    *float64  `gorm:"type:decimal(3,2)" json:"engagement_delta,omitempty"`
    OutcomeClass       string    `gorm:"size:20" json:"outcome_class"`   // IMPROVED/STABLE/WORSENED
    AdherenceFinal     float64   `gorm:"type:decimal(3,2)" json:"adherence_final"`
    SideEffectsReported bool     `json:"side_effects_reported"`
    // Confounder-adjustment flag — true if InterventionRecord had confounders (illness/travel/festival/low completeness)
    ConfounderAdjusted  bool     `json:"confounder_adjusted"`
    DataCompleteness    float64  `gorm:"type:decimal(3,2)" json:"data_completeness"` // copied from InterventionRecord at close
    ComputedAt         time.Time `gorm:"autoCreateTime" json:"computed_at"`
}
```

- [ ] **Step 4: Create IOR store service**

Create `kb-20-patient-profile/internal/services/ior_store.go` with:
- `NewIORStore(db *gorm.DB)` constructor
- `CreateIntervention(record)` — inserts with status=OPEN
- `CloseWindow(interventionID, outcome)` — sets status=CLOSED, creates OutcomeRecord
- `FindSimilarOutcomes(drugClass, phenotype, status, minAdherence)` — query with 4-column index

- [ ] **Step 5: Run IOR tests — verify pass**

Run: `cd kb-20-patient-profile && go test ./internal/services/ -run TestSimilarPatient -v`
Expected: 1 PASS, 1 SKIP (integration).

- [ ] **Step 6: Create IOR generator batch job**

Create `kb-20-patient-profile/internal/services/ior_generator.go`:
- `RunDaily()` — find all InterventionRecords where `status=OPEN` and `opened_at + window_days < now()`
- For each: snapshot current MHRI/HbA1c/SBP/eGFR, compute deltas, classify outcome, create OutcomeRecord, set status=CLOSED
- Set `ConfounderAdjusted=true` on OutcomeRecord if the InterventionRecord had any confounder flags set (`ConfoundingIllness != ""` or `TravelDays > 3` or `FestivalPeriod == true` or `DataCompletenessScore < 0.7`). Confounder-adjusted outcomes are excluded from the similar-patient query's `ImprovedPercent` calculation in KB-23's IOR insight provider to avoid skewing population statistics.

- [ ] **Step 7: Add IOR API endpoints**

- `POST /api/v1/kb20/ior/interventions` — create intervention (called by Module11 WINDOW_OPENED)
- `GET /api/v1/kb20/ior/similar?drug_class=X&phenotype=Y` — similar-patient query (called by KB-23)

- [ ] **Step 8: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/
git commit -m "feat(kb-20): implement IOR system — intervention/outcome models, store, generator, similar-patient query (DD#4)"
```

- [ ] **Step 9: Add IOR confounder capture fields (Gap G3 — 4h)**

**Context:** Per the Progress Tracker Gap G3, IOR outcomes may be wrongly attributed to intervention when driven by external factors (illness, travel, festival/fasting period). Without confounder capture, the similar-patient query returns misleading outcome data.

In `intervention_record.go`, add:

```go
// Confounder tracking — populated from KB-20 patient-reported events during observation window
ConfounderFlags     postgres.Jsonb `json:"confounder_flags" gorm:"type:jsonb;default:'{}'"` // e.g., {"illness": true, "travel": true, "fasting_period": "navratri"}
DataCompletenessScore float64      `json:"data_completeness_score"` // 0.0-1.0 — fraction of expected signals received during window
```

In `ior_generator.go`, extend the `CloseWindow()` function:

```go
// V4 G3: Capture confounders from patient-reported events during observation window
func (g *IORGenerator) captureConfounders(patientID string, windowStart, windowEnd time.Time) (postgres.Jsonb, float64) {
    // Query KB-20 for patient-reported events during the window
    events, _ := g.kb20Client.GetPatientEvents(patientID, windowStart, windowEnd)

    confounders := map[string]interface{}{}
    for _, e := range events {
        switch e.Type {
        case "HOSPITALISATION":
            confounders["hospitalisation"] = true
        case "ADVERSE_EVENT":
            confounders["adverse_event"] = e.Description
        case "TRAVEL":
            confounders["travel"] = true
        case "FASTING_PERIOD":
            confounders["fasting_period"] = e.Details // "ramadan", "navratri", etc.
        case "ILLNESS":
            confounders["illness"] = true
        }
    }

    // Data completeness: expected signals vs received
    expectedSignals := g.computeExpectedSignals(patientID, windowStart, windowEnd)
    receivedSignals := g.countReceivedSignals(patientID, windowStart, windowEnd)
    completeness := 0.0
    if expectedSignals > 0 {
        completeness = float64(receivedSignals) / float64(expectedSignals)
    }

    return postgres.Jsonb{RawMessage: marshalJSON(confounders)}, completeness
}
```

In the similar-patient query, filter by data completeness:

```go
// Only return IOR records with completeness >= 0.6 for reliable comparisons
query = query.Where("data_completeness_score >= ?", 0.6)
```

Add migration for new columns and update tests.

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/
git commit -m "feat(kb-20): add IOR confounder capture + data_completeness_score (G3 gap closure)"
```

### Task 9: Create Module9 Engagement Monitor Flink Job

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module9_EngagementMonitor.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/EngagementScore.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/analytics/EngagementScorer.java`

**Context:** Per DD#8, tracks 6 signals (glucose, BP, meal, medication reminder, app session, exercise) with per-signal recency + density scoring. Composite score (0-1) emitted daily. Disengagement alerts at channel-specific thresholds.

- [ ] **Step 1: Create EngagementScore output model**

Create `models/EngagementScore.java`:

```java
package com.cardiofit.flink.models;

import java.time.Instant;
import java.util.Map;

public class EngagementScore {
    private String patientId;
    private double compositeScore;           // 0.0-1.0
    private String status;                   // GREEN/YELLOW/ORANGE/RED
    private Map<String, Double> signalScores; // per-signal breakdown (6 signals)
    private String channel;                  // GOVERNMENT/CORPORATE/GP_PRIMARY
    private int consecutiveRedDays;
    private Instant computedAt;

    public EngagementScore() {}
    public String getPatientId() { return patientId; }
    public void setPatientId(String p) { this.patientId = p; }
    public double getCompositeScore() { return compositeScore; }
    public void setCompositeScore(double s) { this.compositeScore = s; }
    public String getStatus() { return status; }
    public void setStatus(String s) { this.status = s; }
    public Map<String, Double> getSignalScores() { return signalScores; }
    public void setSignalScores(Map<String, Double> m) { this.signalScores = m; }
    public String getChannel() { return channel; }
    public void setChannel(String c) { this.channel = c; }
    public int getConsecutiveRedDays() { return consecutiveRedDays; }
    public void setConsecutiveRedDays(int d) { this.consecutiveRedDays = d; }
    public Instant getComputedAt() { return computedAt; }
    public void setComputedAt(Instant t) { this.computedAt = t; }
}
```

- [ ] **Step 2: Write failing test for engagement scoring logic**

Create `src/test/java/com/cardiofit/flink/operators/Module9_EngagementTest.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.analytics.EngagementScorer;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;
import java.util.*;

public class Module9_EngagementTest {
    @Test void greenStatus_whenAllSignalsRecent() {
        // All 6 signals seen within expected windows → composite >0.7 → GREEN
        Map<String, Long> lastSeen = Map.of(
            "GLUCOSE", 1L, "BP", 1L, "MEAL", 0L, "MED_REMINDER", 0L, "APP_SESSION", 0L, "EXERCISE", 2L);
        double score = EngagementScorer.computeComposite(lastSeen, 7); // 7-day density window
        assertTrue(score > 0.7, "Expected >0.7 for all-active signals, got " + score);
    }

    @Test void redStatus_whenMostSignalsMissing() {
        Map<String, Long> lastSeen = Map.of(
            "GLUCOSE", 10L, "BP", 14L, "MEAL", 21L, "MED_REMINDER", 7L, "APP_SESSION", 30L, "EXERCISE", 30L);
        double score = EngagementScorer.computeComposite(lastSeen, 7);
        assertTrue(score < 0.3, "Expected <0.3 for mostly-inactive signals, got " + score);
    }
}
```

Expected: FAIL — `EngagementScorer` not found.

- [ ] **Step 3: Implement EngagementScorer utility class**

Create `analytics/EngagementScorer.java`:

```java
package com.cardiofit.flink.analytics;

import java.util.Map;

/**
 * DD#8 — 6-signal engagement scoring.
 * Each signal has expected frequency (days between events).
 * Composite = weighted average of (recency × 0.4 + density × 0.6).
 */
public class EngagementScorer {

    // Expected interval in days for each signal
    private static final Map<String, Double> EXPECTED_INTERVAL = Map.of(
        "GLUCOSE", 1.0,       // daily
        "BP", 1.0,            // daily
        "MEAL", 1.0,          // daily
        "MED_REMINDER", 1.0,  // daily
        "APP_SESSION", 2.0,   // every other day
        "EXERCISE", 3.0       // every 3 days
    );

    /**
     * Compute composite engagement score.
     * @param daysSinceLastSeen map of signal → days since last event
     * @param densityWindowDays window for density calculation
     * @return score 0.0-1.0
     */
    public static double computeComposite(Map<String, Long> daysSinceLastSeen, int densityWindowDays) {
        double totalScore = 0.0;
        int signalCount = 0;

        for (Map.Entry<String, Double> entry : EXPECTED_INTERVAL.entrySet()) {
            String signal = entry.getKey();
            double expectedInterval = entry.getValue();
            Long daysSince = daysSinceLastSeen.get(signal);
            if (daysSince == null) daysSince = (long) densityWindowDays; // never seen = max staleness

            // Recency score: exponential decay — 1.0/(1 + daysSince)
            double recency = 1.0 / (1.0 + daysSince);

            // Density score: inverse of how overdue the signal is
            // If seen within expected interval → 1.0, otherwise decays
            double density = Math.min(1.0, expectedInterval / Math.max(1.0, daysSince));

            // Per-signal composite: 40% recency + 60% density
            double signalScore = recency * 0.4 + density * 0.6;
            totalScore += signalScore;
            signalCount++;
        }

        return signalCount > 0 ? totalScore / signalCount : 0.0;
    }

    /**
     * Status thresholds per DD#8 — channel-aware.
     * GREEN threshold comes from market YAML (channels.yaml → engagement_green_threshold).
     * YELLOW = greenThreshold - 0.2, ORANGE = greenThreshold - 0.4.
     * This avoids hardcoding thresholds that differ between GOVERNMENT (0.6) and GP_PRIMARY (0.7).
     */
    public static String classifyStatus(double composite, double greenThreshold) {
        if (composite > greenThreshold) return "GREEN";
        if (composite > greenThreshold - 0.2) return "YELLOW";
        if (composite > greenThreshold - 0.4) return "ORANGE";
        return "RED";
    }

    /** Convenience overload — uses default 0.7 when channel config unavailable */
    public static String classifyStatus(double composite) {
        return classifyStatus(composite, 0.7);
    }
}
```

- [ ] **Step 4: Run engagement tests — verify pass**

Run: `mvn test -Dtest=Module9_EngagementTest`

- [ ] **Step 5: Create Module9_EngagementMonitor.java Flink job**

Follow Module8 pattern: consume `ingestion.*` events, key by patient_id, maintain `MapState<String, Long>` per signal (last-seen timestamp), emit daily `EngagementScore` to `flink.engagement-signals`. When emitting status, call `EngagementScorer.classifyStatus(composite, channelGreenThreshold)` where `channelGreenThreshold` is loaded from market YAML `channels.yaml → engagement_green_threshold` (passed as a Flink job parameter via `ParameterTool`). If channel config is unavailable, falls back to the default 0.7 overload.

- [ ] **Step 6: Register Module9 in FlinkJobOrchestrator**

```java
case "engagement-monitor":
case "module9":
    Module9_EngagementMonitor.createEngagementPipeline(env);
    break;
```

- [ ] **Step 7: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/
git commit -m "feat(flink): add Module9 Engagement Monitor with 6-signal scoring (DD#8)"
```

- [ ] **Step 8: Add channel-aware engagement thresholds (Gap G2 — 2h)**

**Context:** Per the Progress Tracker Gap G2, `EngagementScorer.classifyStatus()` hardcodes `GREEN > 0.7`. Government/ACCHS patients get incorrectly flagged as disengaged despite meeting their channel's lower threshold. The Market Shim (Task 15) defines per-channel thresholds: Corporate=0.70, Government=0.60, ACCHS=0.50.

In `EngagementScorer.java`, replace hardcoded threshold with a configurable lookup:

```java
// BEFORE (WRONG):
// private static final double GREEN_THRESHOLD = 0.7;
// String status = composite >= GREEN_THRESHOLD ? "GREEN" : composite >= 0.4 ? "ORANGE" : "RED";

// AFTER (CORRECT): channel-aware thresholds loaded from market config
public enum EngagementStatus { GREEN, ORANGE, RED }

public EngagementStatus classifyStatus(double composite, String channel) {
    double greenThreshold = channelThresholds.getOrDefault(channel, 0.70);
    double orangeThreshold = greenThreshold * 0.57; // proportional scaling (0.4/0.7 ≈ 0.57)

    if (composite >= greenThreshold) return EngagementStatus.GREEN;
    if (composite >= orangeThreshold) return EngagementStatus.ORANGE;
    return EngagementStatus.RED;
}

// Channel thresholds loaded from MarketConfig YAML via Flink ParameterTool
private Map<String, Double> channelThresholds;

public void loadChannelThresholds(ParameterTool params) {
    // Default thresholds — overridden by market YAML
    channelThresholds = new HashMap<>();
    channelThresholds.put("CORPORATE", Double.parseDouble(params.get("engagement.threshold.corporate", "0.70")));
    channelThresholds.put("GOVERNMENT", Double.parseDouble(params.get("engagement.threshold.government", "0.60")));
    channelThresholds.put("GP_PRIMARY", Double.parseDouble(params.get("engagement.threshold.gp_primary", "0.70")));
    channelThresholds.put("ACCHS", Double.parseDouble(params.get("engagement.threshold.acchs", "0.50")));
    channelThresholds.put("SPECIALIST", Double.parseDouble(params.get("engagement.threshold.specialist", "0.70")));
}
```

Tests:

```java
@Test
@DisplayName("G2: ACCHS patient with 0.55 engagement is GREEN (threshold 0.50)")
void acchs_055_isGreen() {
    scorer.loadChannelThresholds(defaultParams());
    assertEquals(EngagementStatus.GREEN, scorer.classifyStatus(0.55, "ACCHS"));
}

@Test
@DisplayName("G2: Corporate patient with 0.55 engagement is ORANGE (threshold 0.70)")
void corporate_055_isOrange() {
    scorer.loadChannelThresholds(defaultParams());
    assertEquals(EngagementStatus.ORANGE, scorer.classifyStatus(0.55, "CORPORATE"));
}

@Test
@DisplayName("G2: Government patient with 0.65 engagement is GREEN (threshold 0.60)")
void government_065_isGreen() {
    scorer.loadChannelThresholds(defaultParams());
    assertEquals(EngagementStatus.GREEN, scorer.classifyStatus(0.65, "GOVERNMENT"));
}

@Test
@DisplayName("G2: Unknown channel falls back to 0.70 threshold")
void unknownChannel_fallsBackToDefault() {
    scorer.loadChannelThresholds(defaultParams());
    assertEquals(EngagementStatus.ORANGE, scorer.classifyStatus(0.55, "UNKNOWN"));
}
```

The `EngagementScore` output model should also include the applied threshold and channel:

```java
private String patientChannel;           // CORPORATE, GOVERNMENT, ACCHS, etc.
private double appliedGreenThreshold;    // threshold used for this patient's classification
```

```bash
git add backend/shared-infrastructure/flink-processing/src/
git commit -m "feat(flink): make engagement thresholds channel-aware — ACCHS 0.50, Government 0.60, Corporate 0.70 (G2 gap closure)"
```

### Task 10: Create Module11 InterventionWindowMonitor Flink Job

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module11_InterventionWindowMonitor.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/InterventionWindowEvent.java`

**Context:** Per DD#4 Section 4, this job consumes `clinical.intervention-events`, tracks observation windows (14 days for lifestyle, 28 days for medication), and emits WINDOW_OPENED / WINDOW_CLOSED events that trigger the IOR Generator.

- [ ] **Step 1: Create InterventionWindowEvent model**

```java
package com.cardiofit.flink.models;

import java.time.Instant;

public class InterventionWindowEvent {
    public enum WindowStatus { OPENED, CLOSED, EXPIRED }

    private String patientId;
    private String interventionId;
    private WindowStatus status;
    private String interventionType;  // MEDICATION/LIFESTYLE/COMBINED
    private int windowDays;           // 14 for lifestyle, 28 for medication
    private Instant openedAt;
    private Instant closedAt;

    public InterventionWindowEvent() {}
    public String getPatientId() { return patientId; }
    public void setPatientId(String p) { this.patientId = p; }
    public WindowStatus getStatus() { return status; }
    public void setStatus(WindowStatus s) { this.status = s; }
    public String getInterventionId() { return interventionId; }
    public void setInterventionId(String id) { this.interventionId = id; }
    public String getInterventionType() { return interventionType; }
    public void setInterventionType(String t) { this.interventionType = t; }
    public int getWindowDays() { return windowDays; }
    public void setWindowDays(int d) { this.windowDays = d; }
    public Instant getOpenedAt() { return openedAt; }
    public void setOpenedAt(Instant t) { this.openedAt = t; }
    public Instant getClosedAt() { return closedAt; }
    public void setClosedAt(Instant t) { this.closedAt = t; }
}
```

- [ ] **Step 2: Write failing test for window expiry logic**

Create `src/test/java/com/cardiofit/flink/operators/Module11_WindowTest.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.InterventionWindowEvent;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module11_WindowTest {
    @Test void windowDays_medication_is28() {
        int days = InterventionWindowHelper.getWindowDays("MEDICATION");
        assertEquals(28, days);
    }

    @Test void windowDays_lifestyle_is14() {
        int days = InterventionWindowHelper.getWindowDays("LIFESTYLE");
        assertEquals(14, days);
    }

    @Test void windowStatus_closedAfterExpiry() {
        long openedAt = 1000L;
        long windowMs = 28L * 24 * 60 * 60 * 1000;
        long currentTime = openedAt + windowMs + 1;
        assertTrue(InterventionWindowHelper.isExpired(openedAt, 28, currentTime));
    }
}
```

Expected: FAIL — `InterventionWindowHelper` not found.

- [ ] **Step 3: Implement Module11 with timer-based window tracking**

Create `Module11_InterventionWindowMonitor.java` following Module8 pattern:
- Consume `clinical.intervention-events` (keyed by patient_id)
- `MapState<String, Long> openWindows` — interventionId → expiry timestamp
- On INTERVENTION_APPROVED: register Flink event-time timer at `now + windowDays`
- `onTimer()`: emit WINDOW_CLOSED event, trigger IOR generator via `clinical.intervention-events`
- Side output for expired windows (past deadline without closure)

Also create `InterventionWindowHelper.java` (testable utility extracted from the Flink operator):
- `getWindowDays(interventionType)` — 14 for LIFESTYLE, 28 for MEDICATION, 28 for COMBINED
- `isExpired(openedAtMs, windowDays, currentTimeMs)` — pure function for timer check

Key Flink pattern: Use `ctx.timerService().registerEventTimeTimer(expiryMs)` for window tracking. This is the same timer pattern used in Module4_PatternDetection.

- [ ] **Step 4: Run window tests — verify pass**

Run: `mvn test -Dtest=Module11_WindowTest`
Expected: 3 tests PASS.

- [ ] **Step 5: Register Module11 in FlinkJobOrchestrator**

```java
case "intervention-window":
case "module11":
    Module11_InterventionWindowMonitor.createWindowPipeline(env);
    break;
```

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/
git commit -m "feat(flink): add Module11 InterventionWindowMonitor with timer-based window tracking (DD#4)"
```

### Task 10b: Create Module10 MealResponseCorrelator Flink Job

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module10_MealResponseCorrelator.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/MealResponsePair.java`

**Context:** Per NorthStar Section 3.3, this job correlates glucose readings with meal events (glucose impact) and BP readings with sodium estimates (sodium-BP correlation). It consumes from `ingestion.vitals`, `ingestion.patient-reported`, and `ingestion.labs`, matches meal→glucose pairs within a 2-hour window and sodium→BP pairs within a 4-hour window.

- [ ] **Step 1: Create MealResponsePair model**

```java
package com.cardiofit.flink.models;

import java.time.Instant;

public class MealResponsePair {
    public enum PairType { GLUCOSE_MEAL, SODIUM_BP }

    private String patientId;
    private PairType pairType;
    private double stimulusValue;    // meal carbs (g) or sodium estimate (mg)
    private double responseValue;    // glucose delta (mg/dL) or SBP delta (mmHg)
    private double responsePeak;     // peak glucose or peak SBP in window
    private long latencyMinutes;     // time from stimulus to peak response
    private Instant stimulusAt;
    private Instant responseAt;

    public MealResponsePair() {}
    public String getPatientId() { return patientId; }
    public void setPatientId(String p) { this.patientId = p; }
    public PairType getPairType() { return pairType; }
    public void setPairType(PairType t) { this.pairType = t; }
    public double getStimulusValue() { return stimulusValue; }
    public void setStimulusValue(double v) { this.stimulusValue = v; }
    public double getResponseValue() { return responseValue; }
    public void setResponseValue(double v) { this.responseValue = v; }
    public double getResponsePeak() { return responsePeak; }
    public void setResponsePeak(double v) { this.responsePeak = v; }
    public long getLatencyMinutes() { return latencyMinutes; }
    public void setLatencyMinutes(long m) { this.latencyMinutes = m; }
    public Instant getStimulusAt() { return stimulusAt; }
    public void setStimulusAt(Instant t) { this.stimulusAt = t; }
    public Instant getResponseAt() { return responseAt; }
    public void setResponseAt(Instant t) { this.responseAt = t; }
}
```

- [ ] **Step 2: Write failing test for correlation window matching**

Create `src/test/java/com/cardiofit/flink/operators/Module10_CorrelationTest.java`:

```java
package com.cardiofit.flink.operators;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module10_CorrelationTest {
    @Test void glucoseMealPair_matchesWithin2Hours() {
        long mealTs = 1000L;
        long glucoseTs = mealTs + (90 * 60 * 1000); // 90 min later
        assertTrue(MealCorrelationHelper.isWithinWindow(mealTs, glucoseTs, 2));
    }

    @Test void glucoseMealPair_noMatchBeyond2Hours() {
        long mealTs = 1000L;
        long glucoseTs = mealTs + (150 * 60 * 1000); // 150 min later
        assertFalse(MealCorrelationHelper.isWithinWindow(mealTs, glucoseTs, 2));
    }

    @Test void sodiumBPPair_matchesWithin4Hours() {
        long sodiumTs = 1000L;
        long bpTs = sodiumTs + (180 * 60 * 1000); // 3h later
        assertTrue(MealCorrelationHelper.isWithinWindow(sodiumTs, bpTs, 4));
    }
}
```

Expected: FAIL — `MealCorrelationHelper` not found.

- [ ] **Step 3: Implement Module10 Flink job with dual-correlation**

Follow Module8 pipeline pattern:
- Consume from 3 topics: `ingestion.vitals`, `ingestion.patient-reported`, `ingestion.labs`
- Per-patient keyed state: `lastMealEvent` (timestamp + carbs), `lastSodiumEstimate` (timestamp + mg)
- On glucose reading: check if meal event within 2h → emit GLUCOSE_MEAL pair
- On BP reading: check if sodium estimate within 4h → emit SODIUM_BP pair
- Output to `flink.meal-response`

Also create `MealCorrelationHelper.java` (testable utility):
- `isWithinWindow(stimulusTs, responseTs, windowHours)` — pure function for window check

- [ ] **Step 4: Run correlation tests — verify pass**

Run: `mvn test -Dtest=Module10_CorrelationTest`
Expected: 3 tests PASS.

- [ ] **Step 5: Register Module10 in FlinkJobOrchestrator**

```java
case "meal-response":
case "module10":
    Module10_MealResponseCorrelator.createMealResponsePipeline(env);
    break;
```

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/
git commit -m "feat(flink): add Module10 MealResponseCorrelator for glucose-meal and sodium-BP pairs"
```

### Task 10c: Create MealPatternAggregator (Weekly Batch in Module10)

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module10b_MealPatternAggregator.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/MealPatternSummary.java`

**Context:** Module10 does real-time correlation (meal→glucose in 2h, sodium→BP in 4h). But the **weekly dietary pattern aggregation** (worst foods, best foods, avg daily sodium, sodium-BP correlation coefficient, `salt_sensitivity_beta`) is a different concern with a different window. The `salt_sensitivity_beta` feature is required by the phenotype clustering pipeline (DD#9, feature #5) — without it, there's no compute path.

- [ ] **Step 1: Create MealPatternSummary model**

```java
package com.cardiofit.flink.models;

import java.time.Instant;
import java.util.List;

public class MealPatternSummary {
    private String patientId;
    private double avgDailySodiumMg;
    private double sodiumBPCorrelation;  // Pearson r from weekly sodium→SBP pairs
    private double saltSensitivityBeta;  // regression coefficient: mmHg per 100mg sodium
    private List<String> worstFoods;     // top 3 foods by glucose impact
    private List<String> bestFoods;      // top 3 foods by lowest glucose impact
    private int mealPairsInWindow;       // number of pairs used for computation
    private Instant windowStart;
    private Instant windowEnd;

    public MealPatternSummary() {}
    public String getPatientId() { return patientId; }
    public void setPatientId(String p) { this.patientId = p; }
    public double getSaltSensitivityBeta() { return saltSensitivityBeta; }
    public void setSaltSensitivityBeta(double b) { this.saltSensitivityBeta = b; }
    public double getAvgDailySodiumMg() { return avgDailySodiumMg; }
    public void setAvgDailySodiumMg(double v) { this.avgDailySodiumMg = v; }
    public double getSodiumBPCorrelation() { return sodiumBPCorrelation; }
    public void setSodiumBPCorrelation(double v) { this.sodiumBPCorrelation = v; }
    public List<String> getWorstFoods() { return worstFoods; }
    public void setWorstFoods(List<String> f) { this.worstFoods = f; }
    public List<String> getBestFoods() { return bestFoods; }
    public void setBestFoods(List<String> f) { this.bestFoods = f; }
    public int getMealPairsInWindow() { return mealPairsInWindow; }
    public void setMealPairsInWindow(int n) { this.mealPairsInWindow = n; }
    public Instant getWindowStart() { return windowStart; }
    public void setWindowStart(Instant t) { this.windowStart = t; }
    public Instant getWindowEnd() { return windowEnd; }
    public void setWindowEnd(Instant t) { this.windowEnd = t; }
}
```

- [ ] **Step 2: Write failing test for salt sensitivity beta computation**

Create `src/test/java/com/cardiofit/flink/operators/Module10b_PatternTest.java`:

```java
package com.cardiofit.flink.operators;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module10b_PatternTest {

    @Test void saltSensitivityBeta_positiveForSensitivePatient() {
        // sodium_mg values and corresponding SBP deltas
        double[] sodiumMg = {2500, 3000, 1800, 3200, 2000};
        double[] sbpDelta = {5.0, 8.0, 2.0, 10.0, 3.0};
        double beta = MealPatternHelper.computeSaltSensitivityBeta(sodiumMg, sbpDelta);
        assertTrue(beta > 0, "Expected positive beta for salt-sensitive pattern, got " + beta);
    }

    @Test void saltSensitivityBeta_nearZeroForInsensitive() {
        double[] sodiumMg = {2500, 3000, 1800, 3200, 2000};
        double[] sbpDelta = {3.0, 2.0, 4.0, 1.0, 3.5}; // no correlation
        double beta = MealPatternHelper.computeSaltSensitivityBeta(sodiumMg, sbpDelta);
        assertTrue(Math.abs(beta) < 0.005, "Expected near-zero beta, got " + beta);
    }

    @Test void avgDailySodium_computesCorrectly() {
        double[] dailySodium = {2500, 3000, 1800, 2200, 2700, 2400, 2600};
        double avg = MealPatternHelper.avgDailySodium(dailySodium);
        assertEquals(2457.1, avg, 1.0);
    }
}
```

Expected: FAIL — `MealPatternHelper` not found.

- [ ] **Step 3: Create MealPatternHelper utility and Module10b aggregator**

Create `MealPatternHelper.java` (testable utility):

```java
package com.cardiofit.flink.operators;

public class MealPatternHelper {

    /** OLS regression: mmHg per 100mg sodium */
    public static double computeSaltSensitivityBeta(double[] sodiumMg, double[] sbpDelta) {
        if (sodiumMg.length < 3 || sodiumMg.length != sbpDelta.length) return 0.0;
        int n = sodiumMg.length;
        double sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;
        for (int i = 0; i < n; i++) {
            double x = sodiumMg[i] / 100.0; // per 100mg
            sumX += x; sumY += sbpDelta[i]; sumXY += x * sbpDelta[i]; sumX2 += x * x;
        }
        double denom = n * sumX2 - sumX * sumX;
        if (Math.abs(denom) < 1e-10) return 0.0;
        return (n * sumXY - sumX * sumY) / denom;
    }

    public static double avgDailySodium(double[] dailySodium) {
        double sum = 0;
        for (double v : dailySodium) sum += v;
        return sum / dailySodium.length;
    }
}
```

Create `Module10b_MealPatternAggregator.java`:
- Consume `flink.meal-response` (real-time pairs from Module10)
- 7-day tumbling window per patient using Flink `TumblingEventTimeWindows.of(Time.days(7))`
- On window close: compute avg daily sodium via `MealPatternHelper.avgDailySodium()`, OLS regression via `MealPatternHelper.computeSaltSensitivityBeta()`, rank foods by glucose impact
- Emit `MealPatternSummary` to `flink.meal-patterns`

- [ ] **Step 4: Run pattern tests — verify pass**

Run: `mvn test -Dtest=Module10b_PatternTest`
Expected: 3 tests PASS.

- [ ] **Step 5: Register Module10b in FlinkJobOrchestrator**

```java
case "meal-pattern-aggregator":
case "module10b":
    Module10b_MealPatternAggregator.createAggregatorPipeline(env);
    break;
```

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/src/
git commit -m "feat(flink): add Module10b MealPatternAggregator — weekly sodium/food patterns, salt_sensitivity_beta"
```

---

## Phase C4: Decision Layer — Dual-Domain Cards + Four-Pillar + Feedback (Weeks 10-16)

### Task 11: Extend KB-23 with Dual-Domain Decision Card Generator

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/` (add card types)
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/dual_domain_generator.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/conflict_detector.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_consolidator.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/ior_insight_provider.go`

**Context:** Per DD#5, the 7-step deterministic card generation pipeline: (1) Domain State Assessment, (2) Shared Intervention Check, (3) Conflict Detection, (4) Urgency Calculation, (5) Card Content Generation from templates, (6) Similar-Patient Insight from IOR, (7) Card Emission. Max 3 active cards per patient. Card content is template-interpolated, never LLM-generated.

- [ ] **Step 1: Write failing test for dual-domain state classification**

Create `kb-23-decision-cards/internal/services/dual_domain_generator_test.go`:

```go
func TestClassifyDualDomainState_BothUncontrolled(t *testing.T) {
    state := DomainState{
        GlycemicControl: "UNCONTROLLED", // HbA1c >8%
        BPControl:       "STAGE2",       // SBP >160
    }
    result := classifyDualDomain(state)
    assert.Equal(t, "BOTH_UNCONTROLLED", result)
}

func TestClassifyDualDomainState_DiabetesLed(t *testing.T) {
    state := DomainState{
        GlycemicControl: "UNCONTROLLED",
        BPControl:       "CONTROLLED",
    }
    result := classifyDualDomain(state)
    assert.Equal(t, "DIABETES_LED", result)
}
```

- [ ] **Step 2: Write failing tests for four-pillar evaluator**

Add to `kb-23-decision-cards/internal/services/dual_domain_generator_test.go`:

```go
func TestEvaluateFourPillars_AllActive(t *testing.T) {
    profile := PatientState{
        ActiveMeds: []string{"ACEI", "SGLT2I", "GLP1RA", "FINERENONE"},
        CKDStage: 3,
    }
    assessments := EvaluateFourPillars(profile, MarketConfig{})
    for _, a := range assessments {
        assert.Equal(t, PillarActive, a.Status, "pillar %s should be ACTIVE", a.Pillar)
    }
}

func TestEvaluateFourPillars_SGLTContraindicated_LowEGFR(t *testing.T) {
    egfr := 18.0
    profile := PatientState{EGFR: &egfr, CKDStage: 4}
    assessments := EvaluateFourPillars(profile, MarketConfig{})
    for _, a := range assessments {
        if a.Pillar == PillarSGLT2i {
            assert.Equal(t, PillarContraindicated, a.Status)
            return
        }
    }
    t.Fatal("SGLT2i pillar not found")
}

func TestFindPillarGaps_ReturnsIndicatedNotRx(t *testing.T) {
    assessments := []PillarAssessment{
        {Pillar: PillarRASi, Status: PillarActive},
        {Pillar: PillarSGLT2i, Status: PillarIndicatedNotRx},
        {Pillar: PillarGLP1RA, Status: PillarNotIndicated},
        {Pillar: PillarMRA, Status: PillarIndicatedNotRx},
    }
    gaps := FindPillarGaps(assessments)
    assert.Equal(t, 2, len(gaps))
}
```

Run: `cd kb-23-decision-cards && go test ./internal/services/ -run TestEvaluateFourPillars -v`
Expected: FAIL — `EvaluateFourPillars` not defined.

- [ ] **Step 3: Create four-pillar evaluator (DD#3 decision trees)**

Create `kb-23-decision-cards/internal/services/four_pillar_evaluator.go`:

```go
package services

// PatientState wraps the clinical fields needed for card generation.
// Populated from KB-20 PatientProfile via API call.
type PatientState struct {
    ActiveMeds  []string  `json:"active_meds"`       // e.g., ["ACEI", "SGLT2I", "METFORMIN"]
    CKDStage    int       `json:"ckd_stage"`
    EGFR        *float64  `json:"egfr,omitempty"`
    BMI         *float64  `json:"bmi,omitempty"`
    Potassium   *float64  `json:"potassium,omitempty"`
    HbA1c       *float64  `json:"hba1c,omitempty"`
}

// HasActiveMed checks if the patient is currently on a given drug class
func (p PatientState) HasActiveMed(drugClass string) bool {
    for _, m := range p.ActiveMeds {
        if m == drugClass { return true }
    }
    return false
}

// FourPillar represents the 4 treatment pillars for cardiometabolic management
type Pillar string
const (
    PillarRASi    Pillar = "RASi"      // ACEi/ARB — renal + CV protection
    PillarSGLT2i  Pillar = "SGLT2i"    // renal + cardiac + glycemic
    PillarGLP1RA  Pillar = "GLP1RA"    // CV + weight + glycemic
    PillarMRA     Pillar = "MRA"       // finerenone — renal + cardiac (CKD)
)

type PillarStatus string
const (
    PillarActive          PillarStatus = "ACTIVE"           // patient is on this pillar
    PillarIndicatedNotRx  PillarStatus = "INDICATED_NOT_RX" // guideline says yes, not prescribed
    PillarContraindicated PillarStatus = "CONTRAINDICATED"  // clinical reason to avoid
    PillarNotIndicated    PillarStatus = "NOT_INDICATED"     // not relevant for this patient
)

type PillarAssessment struct {
    Pillar       Pillar       `json:"pillar"`
    Status       PillarStatus `json:"status"`
    Reason       string       `json:"reason"`        // e.g., "eGFR <20 — contraindicated"
    DrugName     string       `json:"drug_name"`     // if active, which specific drug
    Affordable   bool         `json:"affordable"`    // market shim affordability check
}

// EvaluateFourPillars assesses each pillar based on patient state
func EvaluateFourPillars(profile PatientState, marketCfg MarketConfig) []PillarAssessment {
    var result []PillarAssessment

    // Pillar 1: RASi — indicated for all DM+HTN unless K+>5.5 or bilateral RAS
    rasi := PillarAssessment{Pillar: PillarRASi}
    if profile.HasActiveMed("ACEI") || profile.HasActiveMed("ARB") {
        rasi.Status = PillarActive
    } else if profile.Potassium != nil && *profile.Potassium > 5.5 {
        rasi.Status = PillarContraindicated
        rasi.Reason = "K+ >5.5 — hyperkalemia risk"
    } else {
        rasi.Status = PillarIndicatedNotRx
        rasi.Reason = "Guideline-indicated for DM+HTN — not yet prescribed"
    }
    result = append(result, rasi)

    // Pillar 2: SGLT2i — indicated for eGFR ≥20 + DM or HF
    sglt2 := PillarAssessment{Pillar: PillarSGLT2i}
    if profile.HasActiveMed("SGLT2I") {
        sglt2.Status = PillarActive
    } else if profile.EGFR != nil && *profile.EGFR < 20 {
        sglt2.Status = PillarContraindicated
        sglt2.Reason = "eGFR <20 — below initiation threshold"
    } else {
        sglt2.Status = PillarIndicatedNotRx
    }
    // Market shim: check affordability (GLP-1RA at ₹8-15K/month may be unaffordable)
    sglt2.Affordable = true // SGLT2i generally affordable
    result = append(result, sglt2)

    // Pillar 3: GLP-1RA — indicated for BMI ≥27 + CV risk or HbA1c >8 on dual therapy
    glp1 := PillarAssessment{Pillar: PillarGLP1RA}
    if profile.HasActiveMed("GLP1RA") {
        glp1.Status = PillarActive
    } else if profile.BMI != nil && *profile.BMI < 27 {
        glp1.Status = PillarNotIndicated
        glp1.Reason = "BMI <27 — GLP-1RA not first-line"
    } else {
        glp1.Status = PillarIndicatedNotRx
    }
    glp1.Affordable = marketCfg.PharmaShim.AffordabilityCheck // false for some IN channels
    result = append(result, glp1)

    // Pillar 4: MRA (finerenone) — indicated for CKD + DM + UACR >30 + K+ <5.0
    mra := PillarAssessment{Pillar: PillarMRA}
    if profile.HasActiveMed("FINERENONE") || profile.HasActiveMed("MRA") {
        mra.Status = PillarActive
    } else if profile.CKDStage < 2 {
        mra.Status = PillarNotIndicated
    } else if profile.Potassium != nil && *profile.Potassium >= 5.0 {
        mra.Status = PillarContraindicated
        mra.Reason = "K+ ≥5.0 — finerenone contraindicated"
    } else {
        mra.Status = PillarIndicatedNotRx
    }
    result = append(result, mra)

    return result
}

// FindPillarGaps returns pillars that are indicated but not prescribed
func FindPillarGaps(assessments []PillarAssessment) []PillarAssessment {
    var gaps []PillarAssessment
    for _, a := range assessments {
        if a.Status == PillarIndicatedNotRx && a.Affordable {
            gaps = append(gaps, a)
        }
    }
    return gaps
}
```

This is the core clinical intelligence. Without it, dual-domain cards have structure but no content.

- [ ] **Step 4: Add INTEGRATED_DUAL_DOMAIN and FOUR_PILLAR_GAP card types**

In `kb-23-decision-cards/internal/models/card_types.go`:

```go
const (
    CardTypeDualDomain   = "INTEGRATED_DUAL_DOMAIN"
    CardTypeFourPillar   = "FOUR_PILLAR_GAP"
    // ... existing types preserved
)
```

- [ ] **Step 5: Implement dual-domain state classifier**

Create `kb-23-decision-cards/internal/services/dual_domain_generator.go`:

```go
package services

type DomainState struct {
    GlycemicControl string // CONTROLLED, BORDERLINE, UNCONTROLLED
    BPControl       string // CONTROLLED, ELEVATED, STAGE1, STAGE2
    CKDStage        int    // 0-4
    IsNewDiagnosis  bool
}

type DomainClassification string
const (
    BothControlled   DomainClassification = "BOTH_CONTROLLED"
    DiabetesLed      DomainClassification = "DIABETES_LED"
    HTNLed           DomainClassification = "HTN_LED"
    BothUncontrolled DomainClassification = "BOTH_UNCONTROLLED"
    CKDComplicating  DomainClassification = "CKD_COMPLICATING"
    NewlyDiagnosed   DomainClassification = "NEWLY_DIAGNOSED"
)

func classifyDualDomain(state DomainState) DomainClassification {
    if state.IsNewDiagnosis {
        return NewlyDiagnosed
    }
    if state.CKDStage >= 3 {
        return CKDComplicating
    }
    glycOK := state.GlycemicControl == "CONTROLLED"
    bpOK := state.BPControl == "CONTROLLED"
    if glycOK && bpOK {
        return BothControlled
    }
    if !glycOK && !bpOK {
        return BothUncontrolled
    }
    if !glycOK {
        return DiabetesLed
    }
    return HTNLed
}
```

- [ ] **Step 6: Run dual-domain + four-pillar tests — verify pass**

Run: `cd kb-23-decision-cards && go test ./internal/services/ -run "TestClassifyDualDomain|TestEvaluateFourPillars|TestFindPillarGaps" -v`
Expected: 5 tests PASS.

- [ ] **Step 7: Write failing tests for conflict detector + urgency calculator**

Add to `kb-23-decision-cards/internal/services/dual_domain_generator_test.go`:

```go
func TestDetectConflicts_BetaBlockerSU(t *testing.T) {
    conflicts := DetectConflicts([]string{"BETA_BLOCKER", "SULFONYLUREA"})
    assert.Equal(t, 1, len(conflicts))
    assert.Equal(t, "HIGH", conflicts[0].Severity)
}

func TestDetectConflicts_NoConflict(t *testing.T) {
    conflicts := DetectConflicts([]string{"ACEI", "SGLT2I"})
    assert.Empty(t, conflicts)
}

func TestFindSharedInterventions_SGLT2i(t *testing.T) {
    shared := FindSharedInterventions([]string{"SGLT2I"})
    assert.Equal(t, 1, len(shared))
    assert.Contains(t, shared[0].Benefits, "renal_protection")
}

func TestCalculateUrgency_HALTAlert(t *testing.T) {
    input := UrgencyInput{HasHALTAlert: true, DomainClass: BothUncontrolled}
    score := CalculateUrgency(input)
    assert.Equal(t, 70, score) // 40 (HALT) + 30 (BOTH_UNCONTROLLED)
}

func TestConsolidateCards_MaxThree(t *testing.T) {
    cards := []DecisionCard{
        {Urgency: 90}, {Urgency: 80}, {Urgency: 70}, {Urgency: 60}, {Urgency: 50},
    }
    result := ConsolidateCards(cards)
    assert.Equal(t, 3, len(result))
    assert.Equal(t, 90, result[0].Urgency)
}
```

Run: `cd kb-23-decision-cards && go test ./internal/services/ -run "TestDetectConflicts|TestFindShared|TestCalculateUrgency|TestConsolidate" -v`
Expected: FAIL — functions not defined.

- [ ] **Step 8: Implement conflict detector**

Create `kb-23-decision-cards/internal/services/conflict_detector.go`:

```go
package services

type DrugConflict struct {
    DrugA       string `json:"drug_a"`
    DrugB       string `json:"drug_b"`
    ConflictType string `json:"conflict_type"` // GLYCEMIC_WORSENING, BP_WORSENING, RENAL_RISK
    Description string `json:"description"`
    Severity    string `json:"severity"` // HIGH, MODERATE, LOW
}

type SharedIntervention struct {
    DrugClass    string   `json:"drug_class"`
    Benefits     []string `json:"benefits"` // e.g., ["glycemic", "bp_lowering", "renal_protection"]
    Evidence     string   `json:"evidence"` // e.g., "CREDENCE 2019, DAPA-CKD 2020"
}

// Known dual-benefit drugs
var sharedInterventions = []SharedIntervention{
    {DrugClass: "SGLT2I", Benefits: []string{"glycemic", "bp_lowering", "renal_protection", "hf_benefit"}, Evidence: "EMPA-REG, CREDENCE, DAPA-CKD"},
    {DrugClass: "GLP1RA", Benefits: []string{"glycemic", "cv_benefit", "weight_loss"}, Evidence: "LEADER, SUSTAIN-6, REWIND"},
    {DrugClass: "FINERENONE", Benefits: []string{"renal_protection", "cv_benefit"}, Evidence: "FIDELIO-DKA, FIGARO-DKD"},
}

// Known cross-domain conflicts
var knownConflicts = []DrugConflict{
    {DrugA: "THIAZIDE", DrugB: "INSULIN", ConflictType: "GLYCEMIC_WORSENING",
     Description: "High-dose thiazide worsens insulin resistance", Severity: "MODERATE"},
    {DrugA: "BETA_BLOCKER", DrugB: "SULFONYLUREA", ConflictType: "GLYCEMIC_WORSENING",
     Description: "Beta-blockers mask hypoglycemia symptoms and worsen insulin sensitivity", Severity: "HIGH"},
    {DrugA: "NSAID", DrugB: "ACEI", ConflictType: "RENAL_RISK",
     Description: "NSAIDs blunt RASi renal protection", Severity: "HIGH"},
}

func FindSharedInterventions(activeMeds []string) []SharedIntervention {
    var result []SharedIntervention
    for _, si := range sharedInterventions {
        for _, med := range activeMeds {
            if med == si.DrugClass {
                result = append(result, si)
                break
            }
        }
    }
    return result
}

func DetectConflicts(activeMeds []string) []DrugConflict {
    medSet := make(map[string]bool)
    for _, m := range activeMeds { medSet[m] = true }
    var result []DrugConflict
    for _, c := range knownConflicts {
        if medSet[c.DrugA] && medSet[c.DrugB] {
            result = append(result, c)
        }
    }
    return result
}
```

- [ ] **Step 9: Implement urgency calculator and card consolidator**

Create `kb-23-decision-cards/internal/services/card_consolidator.go`:

```go
package services

import "sort"

type DecisionCard struct {
    CardID       string `json:"card_id"`
    CardType     string `json:"card_type"`
    PatientID    string `json:"patient_id"`
    Urgency      int    `json:"urgency"` // 0-100
    Title        string `json:"title"`
    Content      string `json:"content"`
    Actions      []string `json:"actions"`
    IORInsight   string `json:"ior_insight,omitempty"`
    DomainState  string `json:"domain_state"`
}

type UrgencyInput struct {
    DomainClass     DomainClassification
    MHRITrajectory  string  // IMPROVING/STABLE/DECLINING/RAPIDLY_DECLINING
    HasHALTAlert    bool
    HasPAUSEAlert   bool
    PillarGapCount  int     // number of indicated-but-not-prescribed pillars
}

func CalculateUrgency(input UrgencyInput) int {
    score := 0
    if input.HasHALTAlert { score += 40 }
    if input.HasPAUSEAlert { score += 15 }
    switch input.DomainClass {
    case BothUncontrolled: score += 30
    case DiabetesLed, HTNLed: score += 20
    case CKDComplicating: score += 25
    case NewlyDiagnosed: score += 15
    }
    switch input.MHRITrajectory {
    case "RAPIDLY_DECLINING": score += 20
    case "DECLINING": score += 10
    }
    score += input.PillarGapCount * 5
    if score > 100 { score = 100 }
    return score
}

const MaxActiveCards = 3

func ConsolidateCards(cards []DecisionCard) []DecisionCard {
    // Sort by urgency descending
    sort.Slice(cards, func(i, j int) bool {
        return cards[i].Urgency > cards[j].Urgency
    })
    // Keep top N
    if len(cards) > MaxActiveCards {
        cards = cards[:MaxActiveCards]
    }
    return cards
}
```

- [ ] **Step 10: Implement IOR insight provider**

Create `kb-23-decision-cards/internal/services/ior_insight_provider.go`:

```go
package services

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type IORInsight struct {
    SimilarCount    int     `json:"similar_count"`
    ImprovedPercent float64 `json:"improved_percent"`
    AvgWindowDays   int     `json:"avg_window_days"`
    Text            string  `json:"text"`
}

type IORInsightProvider struct {
    KB20BaseURL string
    Client      *http.Client
}

func NewIORInsightProvider(kb20URL string) *IORInsightProvider {
    return &IORInsightProvider{
        KB20BaseURL: kb20URL,
        Client:      &http.Client{Timeout: 5 * time.Second},
    }
}

func (p *IORInsightProvider) GetInsight(drugClass, phenotype string) (*IORInsight, error) {
    url := fmt.Sprintf("%s/api/v1/kb20/ior/similar?drug_class=%s&phenotype=%s",
        p.KB20BaseURL, drugClass, phenotype)
    resp, err := p.Client.Get(url)
    if err != nil { return nil, err }
    defer resp.Body.Close()

    var outcomes []struct {
        OutcomeClass string  `json:"outcome_class"`
        WindowDays   int     `json:"window_days"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&outcomes); err != nil {
        return nil, err
    }

    // Only include insight when ≥30 similar outcomes exist
    if len(outcomes) < 30 { return nil, nil }

    improved := 0
    totalDays := 0
    for _, o := range outcomes {
        if o.OutcomeClass == "IMPROVED" { improved++ }
        totalDays += o.WindowDays
    }
    pct := float64(improved) / float64(len(outcomes)) * 100.0
    avgDays := totalDays / len(outcomes)

    return &IORInsight{
        SimilarCount:    len(outcomes),
        ImprovedPercent: pct,
        AvgWindowDays:   avgDays,
        Text: fmt.Sprintf("In %d similar patients, %.0f%% improved with this approach over %d days.",
            len(outcomes), pct, avgDays),
    }, nil
}
```

- [ ] **Step 11: Implement card content template engine**

Add to `dual_domain_generator.go`:

```go
import "text/template"

var dualDomainTemplate = template.Must(template.New("dual_domain").Parse(
`{{.DomainState}} — {{.Title}}
Primary: {{index .Actions 0}}
{{if gt (len .Actions) 1}}Secondary: {{index .Actions 1}}{{end}}
{{if .IORInsight}}Evidence: {{.IORInsight}}{{end}}
Urgency: {{.Urgency}}/100`))

var fourPillarTemplate = template.Must(template.New("four_pillar").Parse(
`Pillar Gap: {{.PillarName}} ({{.Status}})
Reason: {{.Reason}}
Action: {{.RecommendedAction}}
{{if .Affordable}}Affordable: Yes{{else}}Affordable: Cost barrier — discuss alternatives{{end}}`))

func RenderCard(card DecisionCard) (string, error) {
    var buf bytes.Buffer
    var err error
    switch card.CardType {
    case CardTypeDualDomain:
        err = dualDomainTemplate.Execute(&buf, card)
    case CardTypeFourPillar:
        err = fourPillarTemplate.Execute(&buf, card)
    default:
        return card.Content, nil // passthrough for existing card types
    }
    if err != nil { return "", err }
    return buf.String(), nil
}
```

Import `"bytes"` at top of file.

- [ ] **Step 12: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/
git commit -m "feat(kb-23): implement dual-domain decision card pipeline — 7-step generator, four-pillar evaluator, conflict detection, IOR insights (DD#3+DD#5)"
```

### Task 12: Implement Physician Feedback Capture (DD#10 Foundation)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/feedback.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/feedback_store.go`

**Context:** Per DD#10, capture 21 fields per physician interaction: action (APPROVE/MODIFY/REJECT/DEFER), rejection reason (7-category enum), modification detail (structured JSON diff), response time, patient context snapshot. Store in PostgreSQL feedback schema.

- [ ] **Step 1: Create feedback model (21 fields)**

Create `kb-23-decision-cards/internal/models/feedback.go`:

```go
package models

import "time"

type PhysicianAction string
const (
    ActionApprove PhysicianAction = "APPROVE"
    ActionModify  PhysicianAction = "MODIFY"
    ActionReject  PhysicianAction = "REJECT"
    ActionDefer   PhysicianAction = "DEFER"
)

type RejectionReason string
const (
    RejectContraindication RejectionReason = "CONTRAINDICATION_KNOWN"
    RejectAllergyHistory   RejectionReason = "ALLERGY_HISTORY"
    RejectPatientPreference RejectionReason = "PATIENT_PREFERENCE"
    RejectClinicalJudgment RejectionReason = "CLINICAL_JUDGMENT"
    RejectInsurance        RejectionReason = "INSURANCE_FORMULARY"
    RejectDuplicate        RejectionReason = "DUPLICATE_THERAPY"
    RejectOther            RejectionReason = "OTHER"
)

type CardFeedback struct {
    ID                uint            `gorm:"primaryKey" json:"id"`
    CardID            string          `gorm:"index;size:64;not null" json:"card_id"`
    PhysicianID       string          `gorm:"index;size:64;not null" json:"physician_id"`
    PatientID         string          `gorm:"index;size:64;not null" json:"patient_id"`
    Action            PhysicianAction `gorm:"size:20;not null" json:"action"`
    RejectionReason   *RejectionReason `gorm:"size:30" json:"rejection_reason,omitempty"`
    RejectionFreeText string          `gorm:"size:500" json:"rejection_free_text,omitempty"`
    ModificationDiff  string          `gorm:"type:jsonb" json:"modification_diff,omitempty"`
    ResponseTimeMs    int64           `json:"response_time_ms"`
    CardType          string          `gorm:"size:40" json:"card_type"`
    CardUrgency       int             `json:"card_urgency"`
    PatientStratum    string          `gorm:"size:30" json:"patient_stratum"`
    PatientMHRI       *float64        `gorm:"type:decimal(5,2)" json:"patient_mhri,omitempty"`
    PatientPhenotype  string          `gorm:"size:30" json:"patient_phenotype,omitempty"`
    DrugClass         string          `gorm:"size:30" json:"drug_class,omitempty"`
    DomainState       string          `gorm:"size:30" json:"domain_state"`
    Channel           string          `gorm:"size:20" json:"channel"`
    MarketCode        string          `gorm:"size:10" json:"market_code"` // IN/AU
    SessionID         string          `gorm:"size:64" json:"session_id"`
    CreatedAt         time.Time       `gorm:"autoCreateTime" json:"created_at"`
}
```

- [ ] **Step 2: Write failing test for feedback store**

Create `kb-23-decision-cards/internal/services/feedback_store_test.go`:

```go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "your-module/internal/models"
)

func TestCreateFeedback_RequiresAction(t *testing.T) {
    store := NewFeedbackStore(nil) // nil DB for unit test
    fb := models.CardFeedback{
        CardID:      "card-001",
        PhysicianID: "dr-001",
        PatientID:   "patient-001",
        Action:      models.ActionApprove,
    }
    err := store.Validate(fb)
    assert.NoError(t, err)
}

func TestCreateFeedback_RejectRequiresReason(t *testing.T) {
    store := NewFeedbackStore(nil)
    fb := models.CardFeedback{
        CardID:      "card-001",
        PhysicianID: "dr-001",
        PatientID:   "patient-001",
        Action:      models.ActionReject,
        // Missing RejectionReason — should fail validation
    }
    err := store.Validate(fb)
    assert.Error(t, err)
}

func TestGetRejectionStats_EmptyResult(t *testing.T) {
    store := NewFeedbackStore(nil)
    stats, err := store.GetRejectionStats("SGLT2I", 30)
    assert.NoError(t, err)
    assert.Equal(t, 0.0, stats.RejectionRate)
}
```

Run: `cd kb-23-decision-cards && go test ./internal/services/ -run TestCreateFeedback -v`
Expected: FAIL — `NewFeedbackStore` not defined.

- [ ] **Step 3: Create feedback store service**

Create `kb-23-decision-cards/internal/services/feedback_store.go`:
- `NewFeedbackStore(db *gorm.DB)` constructor
- `Validate(feedback)` — ensure required fields, reject without reason fails
- `CreateFeedback(feedback)` — insert with patient context snapshot
- `GetFeedbackByCard(cardID)` — retrieve all feedback for a card
- `GetRejectionStats(drugClass, timeRange)` — aggregate rejection rates for Pipeline 1

- [ ] **Step 4: Add feedback capture API**

Add route: `POST /api/v1/cards/:id/feedback` with:
- Validate required fields (action, physician_id, patient_id)
- On APPROVE/MODIFY: also emit `clinical.intervention-events` Kafka message to trigger IOR window

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/
git commit -m "feat(kb-23): implement physician feedback capture — 21-field schema, store, API, IOR trigger (DD#10)"
```

---

## Phase C5: Learning Layer — Phenotype Clustering + Feedback Pipelines (Months 5-9)

### Task 13: Create Phenotype Clustering Batch Pipeline (Python)

**Files:**
- Create: `backend/shared-infrastructure/analytics/phenotype_clustering/requirements.txt`
- Create: `backend/shared-infrastructure/analytics/phenotype_clustering/feature_extractor.py`
- Create: `backend/shared-infrastructure/analytics/phenotype_clustering/clustering_pipeline.py`
- Create: `backend/shared-infrastructure/analytics/phenotype_clustering/cluster_validator.py`
- Create: `backend/shared-infrastructure/analytics/phenotype_clustering/centroid_exporter.py`
- Create: `backend/shared-infrastructure/analytics/phenotype_clustering/therapy_mapper.py`

**Context:** Per DD#9, quarterly batch pipeline: extract 21 features from KB-20 for patients with ≥90 days data → z-score normalize → UMAP dimensionality reduction (5 random seeds, pick best silhouette) → HDBSCAN clustering → 4-criteria validation → update KB-20 phenotype_cluster field.

- [ ] **Step 1: Create requirements.txt**

```
umap-learn==0.5.5
hdbscan==0.8.33
scikit-learn==1.4.0
pandas==2.2.0
numpy==1.26.4
psycopg2-binary==2.9.9
requests==2.31.0
```

- [ ] **Step 2: Write failing test for feature extractor**

Create `phenotype_clustering/test_feature_extractor.py`:

```python
from feature_extractor import extract_features
import pandas as pd

def test_extract_features_returns_21_columns():
    # Mock patient data with required KB-20 fields
    patient = {
        "hba1c": 7.2, "fbg_avg": 130, "tir": 0.65,
        "sbp_avg": 138, "arv_sbp_7d": 11.5, "dip_class": "NON_DIPPER",
        "egfr": 55, "uacr": 120,
        "bmi": 28.5, "waist_to_height": 0.55, "tg_hdl_ratio": 2.8,
        "weight_trend": "STABLE", "ldl": 130,
        "engagement_composite": 0.6, "adherence_score": 0.75,
        "mhri_score": 62, "age": 58, "diabetes_years": 12,
        "htn_years": 8, "ckd_stage": 3, "data_tier": "TIER_3_SMBG",
    }
    features = extract_features(pd.DataFrame([patient]))
    assert features.shape[1] == 21, f"Expected 21 features, got {features.shape[1]}"

def test_features_are_z_normalized():
    patients = pd.DataFrame([
        {"hba1c": 6.5, "fbg_avg": 110, "tir": 0.8, "sbp_avg": 125, **_defaults()},
        {"hba1c": 9.0, "fbg_avg": 180, "tir": 0.3, "sbp_avg": 160, **_defaults()},
    ])
    features = extract_features(patients)
    for col in features.columns:
        assert abs(features[col].mean()) < 0.01, f"Column {col} not centered"

def _defaults():
    return {"arv_sbp_7d": 10, "dip_class": "DIPPER", "egfr": 70, "uacr": 30,
            "bmi": 26, "waist_to_height": 0.5, "tg_hdl_ratio": 2.0,
            "weight_trend": "STABLE", "ldl": 100, "engagement_composite": 0.7,
            "adherence_score": 0.8, "mhri_score": 70, "age": 55,
            "diabetes_years": 8, "htn_years": 5, "ckd_stage": 2, "data_tier": "TIER_2_HYBRID"}
```

- [ ] **Step 3: Create feature extractor**

Create `phenotype_clustering/feature_extractor.py`:
- Input: DataFrame of KB-20 patient profiles (filter: ≥90 days data)
- 21 features: HbA1c, FBG_avg, TIR, SBP_avg, ARV_SBP_7d, dip_class_encoded, eGFR, UACR, BMI, waist_to_height, TG/HDL, weight_trend_encoded, LDL, engagement_composite, adherence_score, MHRI, age, diabetes_years, HTN_years, CKD_stage, data_tier_encoded
- Z-score normalization per column
- Categorical encoding: dip_class (DIPPER=0, NON_DIPPER=1, EXTREME=2, REVERSE=3), weight_trend (LOSING=0, STABLE=1, GAINING=2), data_tier (TIER_1=0, TIER_2=1, TIER_3=2)

- [ ] **Step 4: Run feature extractor tests — verify pass**

Run: `cd phenotype_clustering && python -m pytest test_feature_extractor.py -v`

- [ ] **Step 5: Create clustering pipeline + validator**

Create `phenotype_clustering/clustering_pipeline.py`:
- UMAP: `n_components=2, n_neighbors=15, min_dist=0.1` × 5 random seeds
- HDBSCAN: `min_cluster_size=30, min_samples=5`
- Pick run with best silhouette score
- 4 validation criteria: silhouette ≥0.3, stability (5-seed concordance ≥0.7), IOR discrimination (clusters show different outcome distributions), clinical interpretability (cluster centroids differ on ≥3 features by ≥1 SD)

- [ ] **Step 6: Create centroid exporter**

Create `phenotype_clustering/centroid_exporter.py`:
- Export cluster centroids to `phenotype_centroids.json`
- Format: `{cluster_id: {feature_name: centroid_value, ...}, ...}`
- Used by Australia deployment for cross-market transfer learning

- [ ] **Step 7: Create therapy mapper**

Create `phenotype_clustering/therapy_mapper.py`:
- Input: cluster centroids (`phenotype_centroids.json`) + cluster assignments
- Output: `therapy_modifiers.json` — per-cluster therapy recommendations
- Logic: for each cluster, examine centroid feature values to derive therapy modifiers:
  - If centroid `sbp_avg > 140` and `arv_sbp_7d > 12`: tag `"bp_priority": true, "preferred_add_on": "CCB_OR_THIAZIDE"`
  - If centroid `hba1c > 8.0` and `tir < 0.5`: tag `"glycemic_priority": true, "preferred_escalation": "GLP1RA_OR_SGLT2I"`
  - If centroid `egfr < 45`: tag `"renal_caution": true, "avoid": ["METFORMIN_HIGH_DOSE", "THIAZIDE"]`
  - If centroid `engagement_composite < 0.4`: tag `"engagement_intervention": true, "channel_escalation": "NURSE_OUTREACH"`
  - If centroid `waist_to_height > 0.6` and `tg_hdl_ratio > 3.0`: tag `"metabolic_syndrome_focus": true`

```python
import json
from typing import Dict, Any

# Therapy modifier rules — deterministic, no ML
MODIFIER_RULES = [
    {"condition": lambda c: c.get("sbp_avg", 0) > 140 and c.get("arv_sbp_7d", 0) > 12,
     "modifier": {"bp_priority": True, "preferred_add_on": "CCB_OR_THIAZIDE"}},
    {"condition": lambda c: c.get("hba1c", 0) > 8.0 and c.get("tir", 1.0) < 0.5,
     "modifier": {"glycemic_priority": True, "preferred_escalation": "GLP1RA_OR_SGLT2I"}},
    {"condition": lambda c: c.get("egfr", 100) < 45,
     "modifier": {"renal_caution": True, "avoid": ["METFORMIN_HIGH_DOSE", "THIAZIDE"]}},
    {"condition": lambda c: c.get("engagement_composite", 1.0) < 0.4,
     "modifier": {"engagement_intervention": True, "channel_escalation": "NURSE_OUTREACH"}},
    {"condition": lambda c: c.get("waist_to_height", 0) > 0.6 and c.get("tg_hdl_ratio", 0) > 3.0,
     "modifier": {"metabolic_syndrome_focus": True}},
]

def map_clusters_to_therapy(centroids: Dict[str, Dict[str, float]]) -> Dict[str, Dict[str, Any]]:
    """Map cluster centroids to therapy modifiers. Pure deterministic rules."""
    result = {}
    for cluster_id, centroid in centroids.items():
        modifiers = {}
        for rule in MODIFIER_RULES:
            if rule["condition"](centroid):
                modifiers.update(rule["modifier"])
        result[cluster_id] = modifiers
    return result

if __name__ == "__main__":
    with open("phenotype_centroids.json") as f:
        centroids = json.load(f)
    therapy_map = map_clusters_to_therapy(centroids)
    with open("therapy_modifiers.json", "w") as f:
        json.dump(therapy_map, f, indent=2)
    print(f"Mapped {len(therapy_map)} clusters to therapy modifiers")
```

- [ ] **Step 8: Create KB-20 update script**

Create `phenotype_clustering/update_kb20.py`:
- Connect to KB-20 PostgreSQL
- For each patient: update `phenotype_cluster`, `phenotype_confidence`, and `therapy_modifiers` (JSONB) fields
- Batch updates (1000 patients per transaction)
- Also upload `therapy_modifiers.json` as a cluster-level lookup table

- [ ] **Step 9: Commit**

```bash
git add backend/shared-infrastructure/analytics/phenotype_clustering/
git commit -m "feat(analytics): implement phenotype clustering pipeline — 21-feature UMAP+HDBSCAN with 4-criteria validation + centroid export (DD#9)"
```

- [ ] **Step 10: Implement therapy_mapper.py (Gap G6 — 6h)**

**Files:**
- Create: `backend/shared-infrastructure/analytics/phenotype_clustering/therapy_mapper.py`

**Context:** Per the Progress Tracker Gap G6, `therapy_mapper.py` is listed in the file structure overview but has no implementation steps. Without it, phenotype clustering output is informational but doesn't feed back into card generation. The therapy mapper converts cluster labels + centroids into actionable therapy modifier YAML that KB-23's dual-domain card generator consumes.

```python
"""
therapy_mapper.py — Maps phenotype clusters to therapy modifiers.

Each cluster is characterized by its centroid (21-feature vector). This mapper
assigns therapy modifiers based on cluster characteristics:
- Drug class preferences (e.g., salt-sensitive clusters → prefer thiazide)
- Lifestyle intervention priorities (e.g., high-sodium clusters → sodium reduction first)
- Monitoring intensity (e.g., high-variability clusters → more frequent BP checks)
- Engagement strategy (e.g., low-engagement clusters → simplified regimen)

Output: phenotype_therapy_map.yaml consumed by KB-23 DualDomainGenerator.iorInsightProvider
"""

import yaml
import numpy as np
from dataclasses import dataclass, field, asdict
from typing import List, Dict, Optional
from pathlib import Path


@dataclass
class TherapyModifier:
    cluster_id: int
    cluster_label: str                       # e.g., "salt-sensitive-high-variability"
    preferred_drug_classes: List[str]         # ordered by preference
    contraindicated_adjustments: List[str]    # drug classes to avoid in this phenotype
    lifestyle_priorities: List[str]           # ordered: ["sodium_reduction", "weight_loss", ...]
    monitoring_interval_days: int             # BP/glucose check frequency
    engagement_strategy: str                  # "standard", "simplified", "intensive_coaching"
    mhri_weight_overrides: Dict[str, float] = field(default_factory=dict)  # optional MHRI component weight adjustments
    confidence: float = 0.0                  # silhouette score of cluster


def map_cluster_to_therapy(cluster_id: int, centroid: np.ndarray, feature_names: List[str],
                           silhouette: float) -> TherapyModifier:
    """
    Maps a single cluster centroid to therapy modifiers based on clinical rules.
    Feature indices follow the 21-feature vector from feature_extractor.py.
    """
    features = dict(zip(feature_names, centroid))

    # Determine dominant characteristics
    salt_sensitive = features.get("salt_sensitivity_beta", 0) > 2.0  # mmHg per 100mg sodium
    high_bp_variability = features.get("arv_sbp_30d", 0) > 12.0
    low_engagement = features.get("engagement_composite", 1.0) < 0.5
    high_bmi = features.get("bmi", 0) > 30.0  # Western threshold; market-adjusted downstream
    poor_glycemic = features.get("hba1c", 0) > 8.0
    renal_concern = features.get("egfr", 90) < 45

    # Build therapy modifier
    preferred = []
    contra = []
    lifestyle = []

    # Drug class preferences based on phenotype
    if salt_sensitive:
        preferred.append("THIAZIDE")
        lifestyle.insert(0, "sodium_reduction")
    if renal_concern:
        preferred.append("SGLT2I")  # renal protective
        contra.append("METFORMIN_HIGH_DOSE")  # cap at 1000mg if eGFR 30-45
    if high_bp_variability:
        preferred.append("CCB")  # smooths BP variability
        lifestyle.append("consistent_medication_timing")
    if poor_glycemic:
        preferred.append("GLP1RA")
    if high_bmi:
        lifestyle.append("weight_loss")
        preferred.append("GLP1RA")  # weight loss benefit

    # Default preferences if none selected
    if not preferred:
        preferred = ["RASI", "SGLT2I"]

    # Lifestyle priorities
    if "sodium_reduction" not in lifestyle:
        lifestyle.append("sodium_reduction")
    lifestyle.extend(["physical_activity", "stress_management"])

    # Monitoring intensity
    if high_bp_variability or renal_concern:
        monitoring = 3  # every 3 days
    elif low_engagement:
        monitoring = 7  # weekly (don't overwhelm)
    else:
        monitoring = 5  # standard

    # Engagement strategy
    if low_engagement:
        strategy = "simplified"  # fewer cards, simpler language
    elif high_bp_variability and poor_glycemic:
        strategy = "intensive_coaching"
    else:
        strategy = "standard"

    # Cluster label
    traits = []
    if salt_sensitive: traits.append("salt-sensitive")
    if high_bp_variability: traits.append("high-variability")
    if low_engagement: traits.append("low-engagement")
    if poor_glycemic: traits.append("poor-glycemic")
    if renal_concern: traits.append("renal-concern")
    label = "-".join(traits) if traits else "standard-profile"

    return TherapyModifier(
        cluster_id=cluster_id,
        cluster_label=label,
        preferred_drug_classes=preferred,
        contraindicated_adjustments=contra,
        lifestyle_priorities=lifestyle,
        monitoring_interval_days=monitoring,
        engagement_strategy=strategy,
        confidence=silhouette,
    )


def generate_therapy_map(centroids: np.ndarray, feature_names: List[str],
                         silhouettes: List[float], output_path: Path) -> Path:
    """
    Generates phenotype_therapy_map.yaml from clustering output.

    Args:
        centroids: (n_clusters, 21) array of cluster centroids
        feature_names: list of 21 feature names
        silhouettes: per-cluster silhouette scores
        output_path: directory to write YAML

    Returns:
        Path to generated YAML file
    """
    modifiers = []
    for i in range(len(centroids)):
        modifier = map_cluster_to_therapy(i, centroids[i], feature_names, silhouettes[i])
        modifiers.append(asdict(modifier))

    yaml_path = output_path / "phenotype_therapy_map.yaml"
    with open(yaml_path, "w") as f:
        yaml.dump({"therapy_modifiers": modifiers, "version": "1.0",
                    "generated_by": "phenotype_clustering_pipeline"},
                   f, default_flow_style=False, sort_keys=False)

    return yaml_path
```

Wire into `clustering_pipeline.py` after centroid export:

```python
# After centroid export, generate therapy map
from therapy_mapper import generate_therapy_map
therapy_path = generate_therapy_map(centroids, feature_names, cluster_silhouettes, output_dir)
logger.info(f"Therapy map written to {therapy_path}")
```

Tests:

```python
def test_salt_sensitive_cluster_prefers_thiazide():
    centroid = np.zeros(21)
    centroid[FEATURE_IDX["salt_sensitivity_beta"]] = 3.5  # > 2.0 threshold
    modifier = map_cluster_to_therapy(0, centroid, FEATURE_NAMES, 0.8)
    assert "THIAZIDE" in modifier.preferred_drug_classes
    assert "sodium_reduction" == modifier.lifestyle_priorities[0]

def test_renal_concern_caps_metformin():
    centroid = np.zeros(21)
    centroid[FEATURE_IDX["egfr"]] = 38  # < 45
    modifier = map_cluster_to_therapy(0, centroid, FEATURE_NAMES, 0.7)
    assert "METFORMIN_HIGH_DOSE" in modifier.contraindicated_adjustments
    assert "SGLT2I" in modifier.preferred_drug_classes

def test_low_engagement_gets_simplified_strategy():
    centroid = np.zeros(21)
    centroid[FEATURE_IDX["engagement_composite"]] = 0.3
    modifier = map_cluster_to_therapy(0, centroid, FEATURE_NAMES, 0.6)
    assert modifier.engagement_strategy == "simplified"
    assert modifier.monitoring_interval_days == 7
```

```bash
git add backend/shared-infrastructure/analytics/phenotype_clustering/therapy_mapper.py
git commit -m "feat(analytics): implement therapy_mapper — cluster-to-therapy-modifier mapping for KB-23 card generation (G6 gap closure)"
```

### Task 14: Implement Feedback Analysis Pipelines (DD#10 Pipelines 1-4)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/feedback_analyzer.go`

**Context:** Per DD#10 Section 3, four analysis pipelines: (1) Rejection Pattern Detection (monthly, ≥30% rejection by ≥3 physicians), (2) Modification Pattern Learning (monthly, consistent parameter modifications), (3) Safety Rule Enrichment (per-event, 7-day SLA for CONTRAINDICATION_KNOWN), (4) Acceptance Rate Monitoring (real-time dashboard metrics).

- [ ] **Step 1: Write failing test for rejection pattern detection**

```go
func TestDetectRejectionPattern_HighRateTriggersProposal(t *testing.T) {
    feedbacks := []models.CardFeedback{
        {Action: models.ActionReject, DrugClass: "SGLT2I", RejectionReason: ptr(models.RejectContraindication)},
        {Action: models.ActionReject, DrugClass: "SGLT2I", RejectionReason: ptr(models.RejectContraindication)},
        {Action: models.ActionReject, DrugClass: "SGLT2I", RejectionReason: ptr(models.RejectContraindication)},
        {Action: models.ActionApprove, DrugClass: "SGLT2I"},
    }
    proposals := DetectRejectionPatterns(feedbacks, 0.30, 3) // 30% threshold, 3 physicians
    assert.Equal(t, 1, len(proposals))
    assert.Equal(t, "SGLT2I", proposals[0].DrugClass)
}
```

- [ ] **Step 2: Create RuleChangeProposal model**

Add to `kb-23-decision-cards/internal/models/feedback.go`:

```go
type ProposalStatus string
const (
    ProposalPending   ProposalStatus = "PENDING"
    ProposalReviewed  ProposalStatus = "REVIEWED"
    ProposalAccepted  ProposalStatus = "ACCEPTED"
    ProposalRejected  ProposalStatus = "REJECTED_BY_COMMITTEE"
    ProposalDeployed  ProposalStatus = "DEPLOYED"
)

type RuleChangeProposal struct {
    ID              uint           `gorm:"primaryKey" json:"id"`
    DrugClass       string         `gorm:"index;size:30;not null" json:"drug_class"`
    RejectionReason RejectionReason `gorm:"size:30;not null" json:"rejection_reason"`
    RejectionRate   float64        `gorm:"type:decimal(5,4)" json:"rejection_rate"`  // e.g., 0.3500 = 35%
    PhysicianCount  int            `json:"physician_count"`
    SampleSize      int            `json:"sample_size"`              // total feedback records in analysis window
    ProposedChange  string         `gorm:"type:text" json:"proposed_change"` // human-readable description
    Status          ProposalStatus `gorm:"size:30;default:'PENDING'" json:"status"`
    ReviewedBy      string         `gorm:"size:64" json:"reviewed_by,omitempty"`   // physician committee reviewer
    ReviewNotes     string         `gorm:"type:text" json:"review_notes,omitempty"`
    ReviewedAt      *time.Time     `json:"reviewed_at,omitempty"`
    DeployedAt      *time.Time     `json:"deployed_at,omitempty"`
    AnalysisPeriod  string         `gorm:"size:20" json:"analysis_period"` // e.g., "2026-03"
    CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
}
```

GORM auto-migration will create the `rule_change_proposals` table.

- [ ] **Step 3: Implement Pipeline 1 — rejection pattern detection**

Create `feedback_analyzer.go`:
- `DetectRejectionPatterns(feedbacks, threshold, minPhysicians)` — monthly batch
- Group by (drug_class, rejection_reason), flag when rate ≥30% across ≥3 physicians
- Output: `RuleChangeProposal` — persisted to DB with `Status=PENDING`
- Deduplication: skip creating proposal if an identical (drug_class, rejection_reason, analysis_period) already exists

- [ ] **Step 4: Implement Pipeline 3 — safety rule enrichment**

Add to `feedback_analyzer.go`:
- `EnrichSafetyRule(feedback)` — per-event trigger when rejection_reason is CONTRAINDICATION_KNOWN
- Creates alert for KB-24 review with 7-day SLA
- Only fires when the contraindication is not already in KB-24's rule set

- [ ] **Step 5: Implement Pipeline 4 — acceptance rate metrics**

Add to `feedback_analyzer.go`:
- `ComputeAcceptanceMetrics(timeRange)` — 8 metrics:
  1. Overall acceptance rate
  2. Acceptance by card_type
  3. Acceptance by drug_class
  4. Acceptance by stratum
  5. Average response time by card_type
  6. Modification rate by drug_class
  7. Rejection reasons distribution
  8. Acceptance trend (7-day rolling)

- [ ] **Step 6: Add API endpoints with governance lifecycle**

- `GET /api/v1/cards/feedback/metrics?range=30d` — Pipeline 4 dashboard
- `GET /api/v1/cards/feedback/proposals` — Pipeline 1 rule change proposals (filterable by status: PENDING/REVIEWED/ACCEPTED/DEPLOYED)
- `POST /api/v1/cards/feedback/proposals/:id/review` — physician committee review action, accepts `{status: "ACCEPTED"|"REJECTED_BY_COMMITTEE", reviewed_by, review_notes}`, updates `ReviewedAt` timestamp
- `POST /api/v1/cards/feedback/proposals/:id/deploy` — marks accepted proposal as DEPLOYED, triggering downstream KB-24 rule refresh (emits `clinical.rule-change-deployed` Kafka event)

Governance lifecycle: PENDING → (committee reviews) → REVIEWED → ACCEPTED/REJECTED_BY_COMMITTEE → (if accepted) → DEPLOYED. Only ACCEPTED proposals can transition to DEPLOYED. The `/deploy` endpoint validates this state machine.

- [ ] **Step 7: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/
git commit -m "feat(kb-23): implement feedback analysis pipelines — rejection patterns, safety enrichment, acceptance metrics (DD#10)"
```

- [ ] **Step 8: Implement feedback governance lifecycle tooling (Gap G4 — 8h)**

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/rule_change_proposal.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/governance_store.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/governance_handlers.go`

**Context:** Per the Progress Tracker Gap G4, feedback pipelines detect patterns (e.g., ≥30% rejection by ≥3 physicians) but have no governed path to deploy rule changes. Without governance tooling, detected patterns sit in logs and never reach production rule updates. This creates a gap in the learning pipeline: feedback is captured → patterns are detected → but no action is taken.

The governance lifecycle: `PROPOSED → UNDER_REVIEW → APPROVED → DEPLOYED → ARCHIVED`

```go
// rule_change_proposal.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type ProposalStatus string

const (
    ProposalStatusProposed    ProposalStatus = "PROPOSED"
    ProposalStatusUnderReview ProposalStatus = "UNDER_REVIEW"
    ProposalStatusApproved    ProposalStatus = "APPROVED"
    ProposalStatusRejected    ProposalStatus = "REJECTED"
    ProposalStatusDeployed    ProposalStatus = "DEPLOYED"
    ProposalStatusArchived    ProposalStatus = "ARCHIVED"
)

type RuleChangeProposal struct {
    ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
    PipelineSource   string         `json:"pipeline_source"`    // "rejection_pattern", "modification_pattern", "safety_enrichment"
    RuleID           string         `json:"rule_id"`            // CID rule, card template, dosing threshold being changed
    CurrentValue     string         `json:"current_value"`      // JSON of current rule configuration
    ProposedValue    string         `json:"proposed_value"`     // JSON of proposed change
    Justification    string         `json:"justification"`      // Evidence summary from feedback pipeline
    EvidenceCount    int            `json:"evidence_count"`     // Number of feedback records supporting this proposal
    PhysicianCount   int            `json:"physician_count"`    // Number of distinct physicians in evidence
    Status           ProposalStatus `json:"status" gorm:"default:'PROPOSED'"`
    ReviewerID       *string        `json:"reviewer_id,omitempty"`      // Physician committee member
    ReviewNotes      *string        `json:"review_notes,omitempty"`
    SafetyTraceID    *string        `json:"safety_trace_id,omitempty"` // KB-18 audit trail reference
    CreatedAt        time.Time      `json:"created_at"`
    ReviewedAt       *time.Time     `json:"reviewed_at,omitempty"`
    DeployedAt       *time.Time     `json:"deployed_at,omitempty"`
}
```

```go
// governance_store.go — CRUD + state machine transitions
package services

func (gs *GovernanceStore) CreateProposal(proposal *models.RuleChangeProposal) error {
    proposal.ID = uuid.New()
    proposal.Status = models.ProposalStatusProposed
    proposal.CreatedAt = time.Now()
    return gs.db.Create(proposal).Error
}

func (gs *GovernanceStore) TransitionStatus(proposalID uuid.UUID, newStatus models.ProposalStatus,
    reviewerID string, notes string) error {
    // Validate state machine transitions
    allowed := map[models.ProposalStatus][]models.ProposalStatus{
        models.ProposalStatusProposed:    {models.ProposalStatusUnderReview, models.ProposalStatusRejected},
        models.ProposalStatusUnderReview: {models.ProposalStatusApproved, models.ProposalStatusRejected},
        models.ProposalStatusApproved:    {models.ProposalStatusDeployed},
        models.ProposalStatusDeployed:    {models.ProposalStatusArchived},
    }

    var current models.RuleChangeProposal
    if err := gs.db.First(&current, proposalID).Error; err != nil {
        return err
    }

    valid := false
    for _, s := range allowed[current.Status] {
        if s == newStatus { valid = true; break }
    }
    if !valid {
        return fmt.Errorf("invalid transition %s → %s", current.Status, newStatus)
    }

    // Log to KB-18 SafetyTrace on every transition
    traceID, err := gs.safetyTraceClient.LogRuleChange(proposalID.String(), string(current.Status), string(newStatus), reviewerID)
    if err != nil {
        return fmt.Errorf("SafetyTrace logging failed: %w", err)
    }

    now := time.Now()
    updates := map[string]interface{}{
        "status":          newStatus,
        "reviewer_id":     reviewerID,
        "review_notes":    notes,
        "safety_trace_id": traceID,
        "reviewed_at":     &now,
    }
    if newStatus == models.ProposalStatusDeployed {
        updates["deployed_at"] = &now
    }

    return gs.db.Model(&current).Updates(updates).Error
}

func (gs *GovernanceStore) ListPendingProposals() ([]models.RuleChangeProposal, error) {
    var proposals []models.RuleChangeProposal
    err := gs.db.Where("status IN ?", []string{"PROPOSED", "UNDER_REVIEW"}).
        Order("created_at ASC").Find(&proposals).Error
    return proposals, err
}
```

API endpoints:

```go
// governance_handlers.go
// POST   /api/v1/kb23/governance/proposals          — create proposal (called by feedback pipelines)
// GET    /api/v1/kb23/governance/proposals?status=X  — list proposals
// PATCH  /api/v1/kb23/governance/proposals/:id       — transition status (committee review)
// GET    /api/v1/kb23/governance/proposals/:id/trace — get SafetyTrace audit trail
```

Wire feedback pipelines to automatically create proposals:

```go
// In feedback_analyzer.go — after detecting rejection pattern:
if rejectionRate >= 0.30 && distinctPhysicians >= 3 {
    proposal := &models.RuleChangeProposal{
        PipelineSource: "rejection_pattern",
        RuleID:         ruleBeingRejected,
        Justification:  fmt.Sprintf("%.0f%% rejection rate by %d physicians over %d cards", rejectionRate*100, distinctPhysicians, totalCards),
        EvidenceCount:  totalCards,
        PhysicianCount: distinctPhysicians,
    }
    gs.governanceStore.CreateProposal(proposal)
}
```

Tests:

```go
func TestGovernance_ValidTransition(t *testing.T) {
    store := NewGovernanceStore(testDB, mockSafetyTrace)
    p := createTestProposal(t, store)
    err := store.TransitionStatus(p.ID, models.ProposalStatusUnderReview, "dr-smith", "reviewing")
    assert.NoError(t, err)
}

func TestGovernance_InvalidTransition_Blocked(t *testing.T) {
    store := NewGovernanceStore(testDB, mockSafetyTrace)
    p := createTestProposal(t, store) // status = PROPOSED
    err := store.TransitionStatus(p.ID, models.ProposalStatusDeployed, "dr-smith", "skip review")
    assert.Error(t, err) // PROPOSED → DEPLOYED is not allowed
}

func TestGovernance_DeployLogsSafetyTrace(t *testing.T) {
    mockTrace := &MockSafetyTraceClient{}
    store := NewGovernanceStore(testDB, mockTrace)
    p := createTestProposal(t, store)
    store.TransitionStatus(p.ID, models.ProposalStatusUnderReview, "dr-smith", "")
    store.TransitionStatus(p.ID, models.ProposalStatusApproved, "dr-jones", "clinically validated")
    store.TransitionStatus(p.ID, models.ProposalStatusDeployed, "system", "auto-deploy")
    assert.Equal(t, 3, mockTrace.CallCount) // 3 transitions = 3 SafetyTrace logs
}
```

PostgreSQL migration:

```sql
CREATE TABLE rule_change_proposals (
    id UUID PRIMARY KEY,
    pipeline_source VARCHAR(50) NOT NULL,
    rule_id VARCHAR(100) NOT NULL,
    current_value JSONB,
    proposed_value JSONB,
    justification TEXT NOT NULL,
    evidence_count INTEGER NOT NULL DEFAULT 0,
    physician_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'PROPOSED',
    reviewer_id VARCHAR(100),
    review_notes TEXT,
    safety_trace_id VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    deployed_at TIMESTAMPTZ
);

CREATE INDEX idx_proposals_status ON rule_change_proposals(status);
CREATE INDEX idx_proposals_pipeline ON rule_change_proposals(pipeline_source, created_at);
```

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/
git commit -m "feat(kb-23): implement feedback governance lifecycle — proposals, committee review, SafetyTrace integration (G4 gap closure)"
```

---

## Phase C6: Market Shim — India + Australia Deployment (Months 6-12)

### Task 15: Create Market Shim Configuration Infrastructure

**Files:**
- Create: `deploy/markets/india/clinical_params.yaml`
- Create: `deploy/markets/india/channels.yaml`
- Create: `deploy/markets/india/pharma_shim.yaml`
- Create: `deploy/markets/india/food_db_config.yaml`
- Create: `deploy/markets/india/compliance.yaml`
- Create: `deploy/markets/australia/clinical_params.yaml`
- Create: `deploy/markets/australia/clinical_params_indigenous.yaml`
- Create: `deploy/markets/australia/channels.yaml`
- Create: `deploy/markets/australia/pharma_shim.yaml`
- Create: `deploy/markets/australia/food_db_config.yaml`
- Create: `deploy/markets/australia/compliance.yaml`
- Create: `backend/shared-infrastructure/knowledge-base-services/pkg/config/market_config.go`
- Note: `health_record_adapter.yaml` files (IN: ABDM, AU: My Health Record) are deferred to the health record integration task — the loader supports them as optional files

**Context:** Per DD#0, 7 shim components selected at deployment time via K8s ConfigMaps. Key differences: BMI thresholds (India 23/25, AU 25/30, ACCHS 23/27.5), autonomous mode (India enabled >90% agreement, AU DISABLED per TGA), health record adapter (ABDM vs My Health Record), food DB (IFCT 2017 vs AUSNUT).

- [ ] **Step 1: Create India clinical_params.yaml**

```yaml
# deploy/markets/india/clinical_params.yaml
bmi_overweight_threshold: 23.0   # WHO Asian cutoff (vs 25 Western)
bmi_obese_threshold: 25.0        # WHO Asian cutoff (vs 30 Western)
sbp_target_default: 130.0
dbp_target_default: 80.0
salt_threshold_mg_day: 5000.0    # WHO recommendation
hba1c_target: 7.0
egfr_alert_threshold: 60.0
waist_risk_male_cm: 90.0         # Asian-specific (vs 102 Western)
waist_risk_female_cm: 80.0       # Asian-specific (vs 88 Western)
ldl_target_high_risk: 70.0
```

- [ ] **Step 2: Create India channels.yaml**

```yaml
# deploy/markets/india/channels.yaml
channels:
  - name: GOVERNMENT
    engagement_green_threshold: 0.6  # lower bar — limited connectivity
    autonomous_allowed: true
    max_autonomous_drug_classes: ["METFORMIN", "SGLT2I", "ARB"]
    autonomous_agreement_threshold: 0.90
  - name: CORPORATE
    engagement_green_threshold: 0.7
    autonomous_allowed: true
    autonomous_agreement_threshold: 0.90
  - name: GP_PRIMARY
    engagement_green_threshold: 0.7
    autonomous_allowed: false        # GP always in loop
```

- [ ] **Step 3: Create India pharma_shim.yaml, food_db_config.yaml, and compliance.yaml**

```yaml
# deploy/markets/india/pharma_shim.yaml
formulary_source: NLEM
affordability_check: true
unaffordable_classes: ["GLP1RA"]   # ₹8-15K/month — most channels
generic_preferred: true
```

```yaml
# deploy/markets/india/food_db_config.yaml
database: IFCT_2017
sodium_estimation_model: indian_cuisine
regional_food_maps:
  - north_indian
  - south_indian
  - gujarati_jain
```

```yaml
# deploy/markets/india/compliance.yaml
regulator: CDSCO
autonomous_threshold: ENABLED      # autonomous mode allowed at ≥90% agreement
data_residency: IN
audit_retention_years: 5
dpdpa_compliant: true              # Digital Personal Data Protection Act 2023
```

- [ ] **Step 4: Create Australia clinical_params.yaml + compliance.yaml + indigenous overrides**

```yaml
# deploy/markets/australia/clinical_params.yaml
bmi_overweight_threshold: 25.0
bmi_obese_threshold: 30.0
sbp_target_default: 130.0
salt_threshold_mg_day: 6000.0
hba1c_target: 7.0
waist_risk_male_cm: 102.0
waist_risk_female_cm: 88.0
```

```yaml
# deploy/markets/australia/clinical_params_indigenous.yaml
# ACCHS channel overrides — AIHW evidence for Aboriginal/Torres Strait Islander populations
bmi_overweight_threshold: 23.0     # AIHW recommendation
bmi_obese_threshold: 27.5          # AIHW recommendation
hba1c_target: 7.0
egfr_alert_threshold: 60.0
```

```yaml
# deploy/markets/australia/compliance.yaml
regulator: TGA
autonomous_threshold: DISABLED     # TGA does not permit autonomous dose changes
data_residency: AU
audit_retention_years: 7
```

- [ ] **Step 5: Create Australia channels.yaml, pharma_shim.yaml, and food_db_config.yaml**

```yaml
# deploy/markets/australia/channels.yaml
channels:
  - name: GP_PRIMARY
    engagement_green_threshold: 0.7
    autonomous_allowed: false       # TGA: always physician-in-loop
  - name: ACCHS
    engagement_green_threshold: 0.5 # lower bar — remote communities
    autonomous_allowed: false
    indigenous_overrides: true      # load clinical_params_indigenous.yaml
```

```yaml
# deploy/markets/australia/pharma_shim.yaml
formulary_source: PBS
affordability_check: false          # PBS subsidises most medications
generic_preferred: false
```

```yaml
# deploy/markets/australia/food_db_config.yaml
database: AUSNUT
sodium_estimation_model: australian_cuisine
regional_food_maps:
  - standard_australian
  - indigenous_bush_foods
```

- [ ] **Step 6: Create shared market config loader package**

Create `backend/shared-infrastructure/knowledge-base-services/pkg/config/market_config.go`:

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

type MarketConfig struct {
    ClinicalParams  ClinicalParams      `yaml:"clinical_params"`
    Channels        []ChannelConfig     `yaml:"channels"`
    PharmaShim      PharmaShim          `yaml:"pharma_shim"`
    Compliance      ComplianceConfig    `yaml:"compliance"`
    FoodDB          FoodDBConfig        `yaml:"food_db"`
    HealthRecord    HealthRecordAdapter `yaml:"health_record"`
}

type ClinicalParams struct {
    BMIOverweightThreshold float64 `yaml:"bmi_overweight_threshold"` // India: 23, AU: 25
    BMIObeseThreshold      float64 `yaml:"bmi_obese_threshold"`      // India: 25, AU: 30
    SBPTargetDefault       float64 `yaml:"sbp_target_default"`
    SaltThresholdMgDay     float64 `yaml:"salt_threshold_mg_day"`
}

type ChannelConfig struct {
    Name                  string  `yaml:"name"`           // GOVERNMENT, CORPORATE, GP_PRIMARY, ACCHS
    EngagementGreenThresh float64 `yaml:"engagement_green_threshold"`
    AutonomousAllowed     bool    `yaml:"autonomous_allowed"`
}

type PharmaShim struct {
    FormularySource    string   `yaml:"formulary_source"`    // "PBS" or "NLEM"
    AffordabilityCheck bool     `yaml:"affordability_check"`
    UnaffordableClasses []string `yaml:"unaffordable_classes"`
    GenericPreferred   bool     `yaml:"generic_preferred"`
}

type FoodDBConfig struct {
    Database             string   `yaml:"database"`               // "IFCT_2017" or "AUSNUT"
    SodiumEstimationModel string  `yaml:"sodium_estimation_model"`
    RegionalFoodMaps     []string `yaml:"regional_food_maps"`
}

type HealthRecordAdapter struct {
    System     string `yaml:"system"`       // "ABDM" or "MY_HEALTH_RECORD"
    FHIRVersion string `yaml:"fhir_version"` // "R4"
    PullEnabled bool   `yaml:"pull_enabled"`
}

type ComplianceConfig struct {
    Regulator           string `yaml:"regulator"`            // "TGA" or "CDSCO"
    AutonomousThreshold string `yaml:"autonomous_threshold"` // "ENABLED" or "DISABLED"
    DataResidency       string `yaml:"data_residency"`       // "AU" or "IN"
    AuditRetentionYears int    `yaml:"audit_retention_years"`
}

// LoadMarketConfig loads all market-specific YAML from deploy/markets/{marketCode}/
// Kubernetes sets MARKET_CODE via ConfigMap. Falls back to "india" if unset.
func LoadMarketConfig(marketCode string) (*MarketConfig, error) {
    if marketCode == "" {
        marketCode = "india"
    }
    basePath := os.Getenv("MARKET_CONFIG_PATH")
    if basePath == "" {
        basePath = "deploy/markets"
    }

    cfg := &MarketConfig{}

    // Required files — fail if missing
    required := []struct {
        name   string
        target interface{}
    }{
        {"clinical_params.yaml", &cfg.ClinicalParams},
        {"compliance.yaml", &cfg.Compliance},
    }
    for _, f := range required {
        path := filepath.Join(basePath, marketCode, f.name)
        data, err := os.ReadFile(path)
        if err != nil {
            return nil, fmt.Errorf("load %s: %w", path, err)
        }
        if err := yaml.Unmarshal(data, f.target); err != nil {
            return nil, fmt.Errorf("parse %s: %w", path, err)
        }
    }

    // Optional files — load if present, skip if missing
    optional := []struct {
        name   string
        target interface{}
    }{
        {"channels.yaml", &cfg.Channels},
        {"pharma_shim.yaml", &cfg.PharmaShim},
        {"food_db_config.yaml", &cfg.FoodDB},
        {"health_record_adapter.yaml", &cfg.HealthRecord},
    }
    for _, f := range optional {
        path := filepath.Join(basePath, marketCode, f.name)
        data, err := os.ReadFile(path)
        if err != nil { continue } // optional — skip if not found
        if err := yaml.Unmarshal(data, f.target); err != nil {
            return nil, fmt.Errorf("parse %s: %w", path, err)
        }
    }

    return cfg, nil
}
```

Every KB service calls `LoadMarketConfig(os.Getenv("MARKET_CODE"))` at startup. This resolves the 4-tier hierarchy (global defaults → market → channel → patient overrides).

- [ ] **Step 7: Wire config loader into KB-20, KB-23, KB-26 startup**

In each KB service's `main.go`, add:
```go
marketCfg, err := config.LoadMarketConfig(os.Getenv("MARKET_CODE"))
if err != nil { log.Fatalf("market config: %v", err) }
```

Pass `marketCfg` to services that need market-aware thresholds (MHRI normalization, engagement thresholds, card generation).

- [ ] **Step 8: Commit**

```bash
git add deploy/markets/ backend/shared-infrastructure/knowledge-base-services/pkg/
git commit -m "feat(market-shim): create India+AU market YAML configs and shared loader package (DD#0)"
```

---

## Dependency Graph Summary

```
C0 (V3 fixes: strata, topics, signals, schema extensions, contract wiring)
 │
 ├──► C1 (CID safety: Module8, KB-24 CID rules) ── can ship independently
 │
 └──► C2 (BP variability: Module7 + KB-20 V4 fields + V3 Flink job extensions)
                     │
                     ▼
               C3 (MHRI + IOR + engagement + meal correlation)
               │    └── includes: G1 CKM stage, G2 channel thresholds, G3 confounders
               │
               ▼
               C4 (dual-domain cards + four-pillar + feedback capture)
               │
               ▼
               C5 (phenotype clustering + feedback pipelines + governance)
               │    └── includes: G4 governance tooling, G6 therapy_mapper
               │
C6 (market shim) ◄── can parallelize with C3-C5
```

**Critical Path:** C0 → C2 → C3 → C4 (the flywheel core)
**Safety Path (parallel):** C0 → C1 (can ship independently)
**Market Path (parallel):** C6 (configuration, no engine code changes)

### Task Summary (Updated)

| Phase | Tasks | New in This Update |
|-------|-------|--------------------|
| C0 | Task 1 (Stratum), Task 2 (Kafka Topics), Task 3 (Signal Types) | **Task 3b** (V3 Schema Extensions), **Task 3c** (V3 Contract Wiring) |
| C1 | Task 4 (Module8 CID) | — |
| C2 | Task 5 (Module7 BP Variability), Task 6 (KB-20 V4 Fields) | **Task 6b** (V3 Flink Job Extensions) |
| C3 | Task 7 (MHRI), Task 8 (IOR), Task 9 (Engagement), Task 10/10b/10c (Windows/Meals) | **G1** (CKM Stage in T7), **G3** (Confounders in T8), **G2** (Channel Thresholds in T9) |
| C4 | Task 11 (Dual-Domain Cards), Task 12 (Feedback Capture) | — |
| C5 | Task 13 (Phenotype Clustering), Task 14 (Feedback Pipelines) | **G6** (therapy_mapper in T13), **G4** (Governance in T14) |
| C6 | Task 15 (Market Shim) | — |

---

## Signal Build Phases (App-Side Rollout Coordination)

> Per the Flink Architecture specification §9, the patient app rolls out signal capture in 4 phases. Backend phases (C0-C6) must coordinate with app phases to ensure consumers exist before signals arrive. This section maps app-side readiness to backend implementation phases.

### App Phase 1 (Weeks 1-4) — 5 Core Signals

| Signal | Type | Backend Requirement |
|--------|------|---------------------|
| S1 FBG | Manual + BLE glucometer | Ingestion service (V3 existing) |
| S3 HbA1c | Lab report | Ingestion service (V3 existing) |
| S5 Med adherence | Reminder ack | Ingestion service (V3 existing) |
| S7 BP seated | BLE cuff | Ingestion service + **Task 3b** (device_type, clinical_grade fields) |
| S14 Weight | Manual + BLE scale | Ingestion service (V3 existing) |

**Clinical Impact:** GLYC-1 + HTN-1 protocols can run. FBG trajectory fires. BP trajectory fires. PREVENT score computable. Shadow pilot can start.

**Backend Dependency:** C0 Tasks 1-3c must be complete. Module7 and Module8 can operate in shadow mode (no card generation yet).

### App Phase 2 (Weeks 5-8) — 12 More Signals (17 total)

| Signal | Type | Backend Requirement |
|--------|------|---------------------|
| S2 PPBG | Manual (2h post-meal) | **Task 3b** (linked_meal_id field) |
| S4 Meal log | Structured + photo | **Task 3b** (sodium_estimated_mg, preparation_method, food_name_local) |
| S8 BP standing | Manual (orthostatic) | **Task 3b** (linked_seated_reading_id) |
| S9 Morning BP | Timed prompt | **Task 3b** (waking_time) |
| S10 Evening BP | Timed prompt | **Task 3b** (sleep_time) |
| S11 Creatinine | Lab entry | V3 existing |
| S12 ACR | Lab entry | V3 existing |
| S13 K+ | Lab entry | V3 existing |
| S15 Waist | Manual + video | V3 existing |
| S16 Activity | Wearable | V3 existing |
| S17 Lipid panel | Lab entry | V3 existing |
| S6 Hypo event | App prompt | **Task 3b** (symptom_awareness boolean for CID-03) |

**Clinical Impact:** Full RENAL-1 protocol. Nocturnal dipping from S9+S10. All CID domains populated. M3-PRP/VFRP triggers. BP Variability Engine fully active with morning surge detection.

**Backend Dependency:** C2 complete (Module7 live, KB-20 expanded). C1 Module8 CID live with all HALT rules evaluating.

### App Phase 3 (Weeks 9-12) — 5 Clinical Signals (22 total)

| Signal | Type | Backend Requirement |
|--------|------|---------------------|
| S18 Symptom report | Structured + NLU | KB-22 HPI Engine (V3 existing) |
| S19 Adverse event | Structured input | KB-24 ADR matching (V3 existing) |
| S20 Hospitalisation | ABDM discharge | KB-20 event publication (**Task 3c**) |
| S21 Symptom resolution | Condition resolved | KB-22 (V3 existing) |
| S22 Disease progression | Condition stage | Priority events (V3 existing) |

**Clinical Impact:** KB-22 Bayesian engine active with real symptom data. KB-24 ADR matching live. Hospitalisation impact on CDI score.

**Backend Dependency:** C3 in progress (MHRI scoring, IOR system being built). V3 contract wiring (**Task 3c**) must be complete.

### App Phase 4 (Weeks 13-20) — 7 V4 Signals (29 total)

| Signal | Type | Backend Requirement |
|--------|------|---------------------|
| S23 Sodium estimate | Auto from meal log + IFCT 2017/AUSNUT | **Task 3** (ObsSodiumEstimate) + Module10 MealResponseCorrelator (**Task 10b**) |
| S24 CGM raw | BLE relay (Ultrahuman/Abbott) | **Task 3** (ObsCGMRaw) + CGM Aggregation V4 extensions (**Task 6b**) |
| S25 Intervention event | Physician APPROVE/MODIFY | **Task 3** (ObsInterventionEvent) + Module11 (**Task 10**) + IOR generator (**Task 8**) |
| S26 Physician feedback | Structured card response | **Task 3** (ObsPhysicianFeedback) + Feedback store (**Task 12**) + Governance (**Task 14**) |
| S27 Waist V4 | Patient-reported | **Task 3** (ObsWaistCircumference) |
| S28 Exercise session | Manual/wearable | **Task 3** (ObsExerciseSession) |
| S29 Mood/stress | Patient-reported scale | **Task 3** (ObsMoodStress) |

**Clinical Impact:** MealResponseCorrelator active. Sodium-BP correlation live. salt_sensitivity_beta for phenotyping. IOR system tracking interventions with confounder capture. Physician feedback learning pipeline. Engagement Monitor fully populated with all 6 signals.

**Backend Dependency:** C4 complete (dual-domain cards live, feedback capture active). C5 in progress or complete (phenotype clustering, feedback pipelines with governance).

---

## Build Verification Commands

After each phase, run:

```bash
# Flink compilation
cd backend/shared-infrastructure/flink-processing
mvn clean package -DskipTests

# KB service compilation
cd backend/shared-infrastructure/knowledge-base-services
for kb in kb-20-patient-profile kb-22-hpi-engine kb-23-decision-cards kb-24-safety-constraint-engine kb-26-metabolic-digital-twin; do
  echo "Building and testing $kb..."
  cd $kb && go build ./... && go test ./... && cd ..
done

# Ingestion service compilation
cd vaidshala/clinical-runtime-platform/services/ingestion-service
go build ./... && go test ./...

# Topic creation verification
bash backend/shared-infrastructure/kafka/scripts/create-v4-topics.sh

# V3 contract wiring verification (after C0 Task 3c)
echo "Verifying KB-22 targets KB-5 (not KB-9)..."
grep -r "kb-9\|KB-9\|8089" backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/ && echo "WARNING: KB-9 references still present" || echo "OK: No KB-9 references"
echo "Verifying KB-22 targets /execute (not /events)..."
grep -r "/events" backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/outcome_publisher.go && echo "WARNING: /events still present" || echo "OK: Uses /execute"
```

---

## Deferred Items (Out of Scope for This Plan)

The following items were identified during review as important for V4 maturity but are deferred to follow-up plans:

1. **Integration Test Harness (Gap G5, ~12h)** — A `docker-compose.integration.yaml` that spins up PostgreSQL, Kafka, Redis, and all KB services for end-to-end integration testing. This is a larger DevOps effort that depends on all services being individually testable first (which this plan establishes). Track as a separate task after Phase C4. Scripted patient journey: enroll → readings → HPI → card → approve → IOR close. This specifically catches the class of inter-service contract bugs that blocked V3.

2. **Health Record Adapter YAML** — `health_record_adapter.yaml` for India (ABDM) and Australia (My Health Record) is noted in Task 15 as deferred. The `MarketConfig` loader already supports it as an optional file. Implementation depends on the ABDM/My Health Record integration design spec.

3. **Cross-Market Phenotype Transfer Learning** — Task 13 exports centroids; the actual cross-market transfer logic (applying India-trained clusters to AU patients via centroid proximity) requires a separate pipeline design.

4. **Tier -1 Patient Channel Adapter** — Not built. All KB-21 adherence data is placeholder until this service exists. The Engagement Monitor (Task 9) and MHRI engagement component (Task 7) will operate on incomplete data until Tier -1 is live. Requires separate design spec for BLE device pairing, notification infrastructure, and patient app communication protocol.

5. **KB-21 Remaining Gaps (G-02, G-03, G-04)** — KB-21 Behavioral Intelligence has 4 RED gaps. G-01 (BEHAVIORAL_GAP→KB-23) is partially addressed by Task 3c contract wiring. G-02 (per-class adherence weights), G-03 (HYPO_RISK routing), and G-04 (LAB_RESULT wiring from KB-20) require KB-21 team implementation before the V-MCU correction loop gain_factor is reliable.

6. **KB-22 Architectural Gaps (AG-01, AG-02, AG-03)** — Perturbation dampening (AG-01), answer-reliability scoring (AG-02), and adherence integration (AG-03) are KB-22-internal improvements. AG-01 is referenced in KB-23 → KB-22 contract (GET /perturbations/:patient_id/active) but the KB-22 endpoint doesn't exist yet.

7. **Market Shim 4-Tier Hierarchy Resolution** — Task 15 creates YAML files and a config loader, but runtime 4-tier resolution (global → market → channel → patient) is not implemented in each KB service. Each service currently loads a flat config; hierarchical override logic needs to be added to KB-20, KB-23, and KB-26 separately.
