# Pipeline 1 V5 — Master Architecture Design

| Field | Value |
|-------|-------|
| Date | 2026-05-01 |
| Status | Approved (master spec) — sub-project specs pending |
| Author | drafted via superpowers:brainstorming session |
| Supersedes | n/a (V4 remains in production until V5 declared ready) |
| Decomposes into | 5 sub-project specs (one per V5 subsystem) |

## 1. Background

Pipeline 1 V4 currently runs in production:

```
PDF → MonkeyOCR L1 → Channel 0 normaliser → Channels A-H → Signal merger → merged_spans.json → push to KB-0 GCP
```

V4 has produced 1,492 multi-channel-corroborated spans across 7 Heart Foundation jobs on the KB-0 reviewer dashboard. It works, but research synthesised on 2026-05-01 (covering OCR/document-understanding innovations from late 2025 / early 2026 — Nemotron Parse, PaddleOCR-VL, MonkeyOCR v1.5, Guideline2Graph, MedKGent) identified five concrete improvements that are individually shippable and collectively address the three biggest pain points:

1. **Table extraction quality** (drug × condition × evidence-grade matrices, decision tables, lipid-target charts)
2. **Relationship preservation** across pages (recommendations referencing algorithms referencing drug classes)
3. **Auditability + provenance** (regulatory readiness, A/B comparison capability)

V5 incorporates these improvements as additive feature flags atop V4 — not a greenfield rewrite.

## 2. Goal + scope

Build V5 as five additive feature-flagged subsystems on the V4 backbone. Each subsystem is independently A/B-measurable against V4 baseline on the same PDFs. V5 is declared production-ready when all 5 subsystems pass their primary metrics on the full regression set + universal regression on the release set + 30 days of running with all flags default-on without rollback.

**In scope:**
- The 5 V5 subsystems described in §4
- Feature-flag mechanism (env var + profile YAML override)
- 3-tier golden test set (smoke / full regression / release)
- Per-subsystem primary success metrics + universal regression metrics
- V4 deprecation criteria

**Out of scope (explicitly):**
- Doc-CoB visual chain-of-boxes (answer-time reasoning, not ingestion)
- KB-0 dashboard UI changes (separate workstream)
- Pipeline 2 (L3 facts → KB-1/3/4/16/20) — different brainstorm needed
- Reviewer workflow changes
- Migration of pre-V5 jobs to new schema (kept as historical baseline)
- Multi-GPU pod orchestration (V5 stays single-GPU)
- Real-time / streaming extraction (V5 is batch-only)

## 3. Architecture — Approach C: Additive layers atop V4

V5 = V4 + 5 additive subsystems, each gated by an independent feature flag. Same backbone (`run_pipeline_targeted.py`, channels, signal merger, KB-0 push) augmented in 5 specific places. No greenfield rewrite. Default-off until each flag passes its acceptance gates, then default-on per-subsystem. V4-equivalent paths kept behind `V5_DISABLE_ALL=1` emergency rollback for 30 days post-cutover, then removed in a calendared retirement PR.

```
                  V4 backbone (unchanged, default-on always)
                                    │
   ┌────────────────────────────────┼────────────────────────────────────┐
   │                                ▼                                    │
   │  L1 MonkeyOCR ─► Ch 0 ─► Channels A-H ─► Signal Merger ─► KB-0 push │
   │                              │                                      │
   │                              │ table region detected by layout      │
   │                              ▼                                      │
   │                     [V5 #1 Table Specialist]                        │
   │                              │                                      │
   │                              ▼                                      │
   │                      [V5 #4 Consensus Gate]                         │
   │                              │                                      │
   │                              ▼                                      │
   │                     [V5 #3 Schema Validation]                       │
   │                              │                                      │
   │                              ▼                                      │
   │              [V5 #2 Bbox Provenance — wraps every span]             │
   │                              │                                      │
   │   ┌──────────────────────────┴──────────────────────────┐           │
   │   ▼                                                       ▼           │
   │  V4 Dossier Assembler         [V5 #5 Decomposition + interface-constraint]
   │  (default fallback)            (Guideline2Graph-style)              │
   └─────────────────────────────────────────────────────────────────────┘
```

**Why additive over greenfield**: ships value every 1-2 weeks, preserves V4 baseline for A/B comparison, single pipeline post-migration, reuses existing KB-0 push and dashboard infrastructure. The hardest subsystem (#5 decomposition) gets pressure-tested under Approach C; if it can't ship under a flag, we fall back to greenfield-branch for #5 only while the other four still ship additively.

## 4. Subsystem inventory + sequencing

| Order | ID | Subsystem | Effort | Depends on |
|------:|----|-----------|------:|-----------|
| **A** | #2 | **Bbox Provenance** — every merged span carries non-null bbox + per-channel attribution + model versions in jsonb | 1-2 days | none — foundational |
| **B** | #1 | **Table Specialist** — Nemotron Parse / Nemotron Nano VL routed by Channel D layout detection on table regions | 3-5 days | #2 (writes provenance) |
| **C** | #4 | **Consensus Entropy gate** — formal CE-Ensemble for selective escalation; replaces ad-hoc channel-merge confidence | 2-3 days | #2 (provenance enables disagreement diff) |
| **D** | #3 | **Schema-first extraction** — ~10 narrow Pydantic schemas, validation as routing signal | ~1 week | #2; benefits from #4 |
| **E** | #5 | **Decomposition** — Guideline2Graph-style topology-aware chunking with interface-constrained graph aggregation | 2-3 weeks | A-D stable |

Each subsystem gets its own brainstorm → spec → plan → implementation cycle after this master spec is approved. Sub-project specs land in `docs/superpowers/specs/2026-05-XX-v5-<subsystem>-design.md`.

## 5. Feature-flag mechanism

Env var (pod-level default) + profile YAML override (per-guideline).

```python
# In run_pipeline_targeted.py or shared config module
def is_v5_enabled(feature: str, profile) -> bool:
    """Resolve V5 feature flag with profile override > env var > default-off.

    Profile YAML override always wins (per-guideline tuning).
    Falls back to env var V5_<FEATURE>=1.
    Defaults to False if neither set.
    Emergency rollback V5_DISABLE_ALL=1 forces False regardless.
    """
    if os.environ.get("V5_DISABLE_ALL") == "1":
        return False
    profile_override = (profile.v5_features or {}).get(feature)
    if profile_override is not None:
        return profile_override
    return os.environ.get(f"V5_{feature.upper()}", "0") == "1"
```

**Flag inventory:**

```
V5_BBOX_PROVENANCE       # subsystem #2 (first to ship)
V5_TABLE_SPECIALIST      # subsystem #1
V5_CONSENSUS_ENTROPY     # subsystem #4
V5_SCHEMA_FIRST          # subsystem #3
V5_DECOMPOSITION         # subsystem #5
V5_DISABLE_ALL=1         # emergency rollback (overrides everything to V4-equivalent)
```

**Profile YAML opt-in example:**

```yaml
# heart_foundation_au_2025.yaml — guideline-specific override
v5_features:
  bbox_provenance: true     # opt in even if pod env is off
  table_specialist: false   # opt out even if pod env is on
  consensus_entropy: null   # null = fall through to env var
```

**Why env var + profile combination**: env var is one source of truth per pod (matches existing OLLAMA_URL pattern); profile YAML lets specific guidelines tune in/out (e.g., RANZCP psych pages may benefit from consensus entropy but not bbox provenance — different heuristic priors).

## 6. Golden test set (3-tier)

| Tier | Composition | Runtime on 4090 | Cadence |
|------|-------------|----------------:|---------|
| **Smoke** | `AU-HF-ACS-HCP-Summary-2025.pdf` (2 pp) + `AU-HF-Cholesterol-Action-Plan-2026.pdf` (5 pp) | ~5 min | Every code change / flag toggle |
| **Full regression** | All 7 HF jobs (`AU-HF-*.pdf` from `wave6/heart_foundation/`) | ~1.5 hr | Per-subsystem completion / before flag default-on |
| **Release** | 30 PDFs sampled from 3,069-corpus (10 clean-text + 10 hybrid-visual + 10 scanned, stratified via `profile_pdf_corpus.py`) | ~6-10 hr | Before declaring V5 production-ready |

**V4 baselines** for comparison: existing local `data/output/v4/job_monkeyocr_*/` directories + KB-0 entries (`l2_extraction_jobs` + `l2_merged_spans` rows). No re-extraction needed for V4 baseline establishment.

**Smoke set composition rationale**: HCP-Summary covers prose extraction + light layout; Cholesterol-Action-Plan covers designed visual content with a table. Together they exercise Channels 0, A, B, C, D, E, F, G — all but H (recovery, only triggers on disagreement).

**Release set composition rationale**: 30 PDFs is the cost-quality break-even point. <20 misses type-specific regressions; >50 is too slow to re-run frequently. Stratification ensures V5 doesn't regress on any one PDF type.

## 7. Success metrics (per-subsystem primary + universal regression)

Each V5 run produces a sidecar `metrics.json` next to `merged_spans.json` in the job dir.

### Universal regression metrics (every flag must pass)

| Metric | Threshold | Failure means |
|--------|-----------|---------------|
| Total spans per PDF | within ±15% of V4 baseline | V5 either deletes content or hallucinates; need investigation |
| TIER_1 span proportion | ≥ V4 baseline | V5 is producing more low-quality spans |
| KB-0 push success rate | 100% | Schema/format compatibility regression |
| New ERROR-level log patterns | 0 | Crash or unexpected error path |

### Per-subsystem primary metrics — full detail

Each subsystem has a **primary metric** (the "we improved" claim), one or more **secondary metrics** (regression-safety specific to that subsystem, beyond the universal regression checks), a **V4 baseline** (current measured value or "n/a, not implemented"), a **V5 target** (what we're shooting for), a **threshold** (minimum value to pass), the **test mechanism** (how it's automated), the **test data** (which PDFs/tables/relationships), the **failure action** (what happens if metric fails), and the **computation formula** (exact math so reproducibility is unambiguous).

#### Subsystem #2 — Bbox Provenance

| Field | Value |
|-------|-------|
| **Primary metric** | Span coverage with non-null `bbox` AND non-empty `channel_provenance` jsonb |
| **Threshold** | **≥99%** of merged spans on smoke set |
| **V4 baseline** | ~30% (only when MonkeyOCR L1 happens to capture bbox; channels A-H don't write provenance) |
| **V5 target** | 100% |
| **Computation formula** | `100 × count(spans where bbox IS NOT NULL AND jsonb_array_length(channel_provenance) >= 1) / count(*)` |
| **Test mechanism** | pytest assertion on `merged_spans.json`; CI hook fails the build if <99% |
| **Test data** | Smoke set (2 PDFs); promoted to full regression once stable |
| **Failure action** | Do NOT flip flag default; keep `V5_BBOX_PROVENANCE=0`; log exact span IDs missing provenance for debug |
| **Secondary metrics** | (a) per-channel bbox: each contributing channel writes its own bbox, jsonb keys = channels; threshold ≥95%. (b) provenance round-trips through KB-0 push: `push → SELECT → diff` is byte-identical, threshold 100%. (c) bbox coords are within page bounds (0 ≤ x,y ≤ page_w/h); threshold 100% |

#### Subsystem #1 — Table Specialist

| Field | Value |
|-------|-------|
| **Primary metric** | Table-cell extraction accuracy: per-table, % of (row, col) cells where extracted text matches ground truth (case-insensitive, whitespace-normalized exact match) |
| **Threshold** | **≥85%** mean across all 15 hand-graded tables (vs V4 ~50% measured baseline on 5 spot-checked tables) |
| **V4 baseline** | ~50% (measured: KDIGO drug-risk-rationale 3 of 6 cells correct, ACS Reference dosing 4 of 8 correct — full 15-table baseline curated as part of #1 work) |
| **V5 target** | ≥90% |
| **Computation formula** | `100 × Σ_table (correct_cells / total_cells) / num_tables` (macro-average across tables; rationale: avoid bias toward larger tables) |
| **Test mechanism** | pytest fixture loads ground-truth CSV per table → diffs against extracted table from job artefact → reports per-cell match % + macro-average |
| **Test data** | 15 hand-graded tables (composition below) |
| **Failure action** | Investigate per-table breakdown; if ≥3 of 15 score <70%, fall back to Nemotron Nano VL alternative; if ≥5 of 15 fail, revert flag and reopen brainstorm for #1 |
| **Hand-graded table composition (15)** | KDIGO drug-risk-rationale (1), ACS Reference dosing tables (3 — STEMI/NSTEMI/post-discharge), CVD Risk decision matrix (1), Lipid-lowering chart (1), Hypertension targets (1), Diabetes T2D algorithm tables (3 — KDIGO Wallace + ADS algo + ADS Position), Cholesterol-Action lipid table (1), ACS Supp-A endpoint trial table (2), Hypertension-Slides BP target slide (1), HF Heart Failure NYHA-class table (1) |
| **Secondary metrics** | (a) Cell-merge accuracy: spans that should be in one cell aren't split, threshold ≥95%. (b) Header detection: row-0/col-0 correctly identified as headers, threshold ≥95%. (c) Numeric coercion: numbers in cells parse as numbers (not "0.5%" left as string), threshold ≥98% |

#### Subsystem #4 — Consensus Entropy gate

| Field | Value |
|-------|-------|
| **Primary metric** | False-positive span rate — proportion of merged spans where only 1 channel contributed AND that channel's confidence is below the median; these are spans most likely to be noise |
| **Threshold** | **Drops ≥20%** on smoke set vs V4 (relative reduction) |
| **V4 baseline** | TBD-measure on smoke (auto-computed during #2 ground-truth pass) — typical for our V4 runs is ~12-18% of merged spans |
| **V5 target** | ≤10% (cuts ~40% relative) |
| **Computation formula** | `100 × count(spans where len(contributing_channels)=1 AND channel_confidence_median(span) < median(all_spans_confidence)) / count(*)` |
| **Test mechanism** | Compare V4 baseline run vs V5 flag-on run on smoke set; pytest computes both and asserts delta |
| **Test data** | Smoke set (2 PDFs) primary; full regression for confirmation |
| **Failure action** | Tune Consensus Entropy threshold (`tau` hyperparameter); if no `tau` achieves both -20% FP AND universal `±15%` total spans, escalate to design review |
| **Secondary metrics** | (a) Escalation rate (spans that triggered re-extraction): threshold ≤5% on smoke. (b) End-to-end run-time delta: ≤+10% wall time vs V4. (c) Channel-coverage rebalancing: which channels gain/lose share post-CE — sanity check, no threshold |

#### Subsystem #3 — Schema-first extraction

| Field | Value |
|-------|-------|
| **Primary metric** | Pydantic-schema validation pass rate: % of extracted facts (per region type) that successfully validate against their schema |
| **Threshold** | **≥95%** on smoke + full regression |
| **V4 baseline** | ~70-80% (estimated; V4 produces freeform text spans, not schema-validated facts — measured by post-hoc validation of existing merged_spans through draft schemas) |
| **V5 target** | ≥98% |
| **Computation formula** | `100 × count(facts where pydantic_validate(fact, schema_for(region_type)).is_valid) / count(facts)` |
| **Test mechanism** | pytest hooks Pydantic ValidationError; CI fails if <95% pass |
| **Test data** | Smoke for tight loop, full regression for completeness |
| **Failure action** | Per-schema failure breakdown — if 1 schema fails ≥20%, that schema is too strict (loosen); if all schemas fail ≥5%, the upstream extraction is producing junk (fix #1 / #4 first) |
| **Schema inventory (10 schemas to define before #3 starts)** | (1) RecommendationStatement, (2) DrugConditionMatrix, (3) eGFRThresholdTable, (4) MonitoringFrequencyRow, (5) EvidenceGradeBlock, (6) AlgorithmStep, (7) ContraindicationStatement, (8) DoseAdjustmentRow, (9) RiskScoreCalculator, (10) FollowUpScheduleEntry |
| **Secondary metrics** | (a) Validation latency: schema check adds ≤50 ms per fact, threshold ≤100 ms. (b) Schema-coverage breadth: ≥80% of extracted facts route to a defined schema (rest go to "freeform" catchall). (c) Escalation-on-fail rate: facts that fail validation get re-extracted; escalation rate ≤8% |

#### Subsystem #5 — Decomposition (Guideline2Graph-style)

| Field | Value |
|-------|-------|
| **Primary metric** | Edge precision AND edge recall on cross-section relationships — paired metric, both must clear threshold |
| **Threshold** | **Both ≥80%** on hand-graded relationship set |
| **V4 baseline** | Edge precision 19.6%, edge recall 16.1% (measured per Guideline2Graph paper's reproduction methodology on KDIGO 2022) |
| **V5 target** | Edge precision ≥87%, edge recall ≥87% (matches Guideline2Graph paper's 87.5% on the same task) |
| **Computation formula** | Precision: `count(extracted_edges ∩ graded_edges) / count(extracted_edges)`. Recall: `count(extracted_edges ∩ graded_edges) / count(graded_edges)`. Edge equality: same source-node-id, same target-node-id, same edge-type label |
| **Test mechanism** | pytest fixture loads ground-truth `relationships.yaml` → loads extracted graph → set-difference comparison |
| **Test data** | 15 hand-graded relationships across HF + KDIGO corpora |
| **Failure action** | If precision <80% but recall ≥80% → tighten interface constraint (over-linking; cut spurious edges). If recall <80% but precision ≥80% → loosen chunking boundaries (missing edges). If both <80% → architectural problem with #5; pause and re-design |
| **Hand-graded relationship composition (15)** | KDIGO 2022 — Recommendation 1.2.3 → Algorithm Fig 1 → Drug Class SGLT2i (3 edges, treat as 1 path = 1 graded relationship). Repeat pattern across: ACS-Guideline (Initial Assessment → Reperfusion Algorithm → Antithrombotic Class), CVD Risk (Score → Action → Drug Class), Cholesterol Action (Statin choice → LDL Target → Monitoring), HF Heart Failure (NYHA Stage → ACE-I/ARB → Dose Titration). Total: 15 such paths |
| **Secondary metrics** | (a) Triplet precision/recall (subject-predicate-object): ≥85% per Guideline2Graph numbers. (b) Node recall: ≥93% (matches paper). (c) Provenance preservation through chunking: every node in final graph has at least one chunk-id provenance reference, threshold 100% |

### Hand-graded ground truth (one-time investment)

| Asset | Count | Approx clinician effort | Sub-project that creates it | Reused by |
|-------|------:|-------------------------:|---------------------------|-----------|
| Tables (cell-by-cell ground truth CSVs) | 15 | ~3 hr | Subsystem #1 (Table Specialist) | #3 schema, #5 decomposition |
| Relationships (graded edge YAMLs) | 15 | ~3 hr | Subsystem #5 (Decomposition) | future V6 |
| Schema-validation negative samples (facts that *should* fail) | ~30 | ~2 hr | Subsystem #3 (Schema-first) | future V6 |
| **Total one-time effort** | | **~8 hr** | | |

Stored under `tests/v5/golden/{tables,relationships,negative_facts}/` in the repo. Versioned with the spec; updates to ground truth require co-located commit messages explaining the rationale.

### Sidecar metrics.json shape (extended)

```json
{
  "job_id": "<uuid>",
  "v5_features_enabled": ["bbox_provenance", "table_specialist"],
  "v4_baseline_job_id": "<uuid>",
  "v4_baseline_source_pdf": "AU-HF-ACS-Guideline-2025.pdf",
  "regression": {
    "total_spans":        {"v4": 398, "v5": 412, "delta_pct": 3.5,  "threshold_pct": 15.0, "status": "PASS"},
    "tier_1_pct":         {"v4": 87.4, "v5": 89.1, "delta_pp": 1.7,  "threshold_pp": 0.0, "status": "PASS"},
    "kb0_push":           {"status": "PASS"},
    "new_error_patterns": {"count": 0, "threshold": 0, "status": "PASS"}
  },
  "primary": {
    "bbox_coverage_pct":          {"v5": 99.7, "v4": 30.2, "threshold": 99.0, "status": "PASS"},
    "channel_provenance_pct":     {"v5": 99.4, "v4":  0.0, "threshold": 99.0, "status": "PASS"},
    "table_cell_accuracy_pct":    {"v5": 88.2, "v4": 51.3, "threshold": 85.0, "status": "PASS"},
    "table_header_detection_pct": {"v5": 96.8, "v4": 73.1, "threshold": 95.0, "status": "PASS"}
  },
  "secondary": {
    "bbox_in_page_bounds_pct":    {"v5": 100.0, "threshold": 100.0, "status": "PASS"},
    "kb0_round_trip_byte_identical": true
  },
  "verdict": "PASS"
}
```

### Hand-graded ground truth (one-time investment)

15 tables (covering KDIGO drug-risk-rationale, ACS Reference dosing, CVD risk decision matrix, lipid-lowering chart, hypertension targets) + 15 relationships (recommendation→algorithm→drug-class refs across the HF + KDIGO corpora) ≈ 6 hr clinician effort. Curated once; reused for every subsequent V5 subsystem comparison and for V6+ in the future.

### Sidecar metrics.json shape

```json
{
  "job_id": "<uuid>",
  "v5_features_enabled": ["bbox_provenance", "table_specialist"],
  "v4_baseline_job_id": "<uuid>",
  "regression": {
    "total_spans":        {"v4": 398, "v5": 412, "delta_pct": 3.5,  "status": "PASS"},
    "tier_1_pct":         {"v4": 87.4, "v5": 89.1, "delta_pp": 1.7, "status": "PASS"},
    "kb0_push":           {"status": "PASS"},
    "new_error_patterns": {"count": 0, "status": "PASS"}
  },
  "primary": {
    "bbox_coverage_pct":       {"v5": 99.7, "threshold": 99.0, "status": "PASS"},
    "table_cell_accuracy_pct": {"v5": 88.2, "v4": 51.3, "threshold": 85.0, "status": "PASS"}
  },
  "verdict": "PASS"
}
```

## 8. V4 deprecation criteria

V5 declared production-ready when **all** of:

1. All 5 subsystem flags pass their primary metrics on the **full regression set**
2. All 5 flags pass universal regression on the **release set** (30 PDFs)
3. 30 days of running with all flags default-on without `V5_DISABLE_ALL` being triggered
4. KB-0 reviewer-confirm rate ≥ V4 baseline (long-loop sanity check)

Post-cutover: V4-equivalent code paths kept behind `V5_DISABLE_ALL=1` for 30 more days, then removed in a single PR titled "V4 retirement: remove fallback paths". Removal PR must show: zero usage of V4 paths in last 30 days of logs, no rollback events, no open V5 bugs at SEV-1/SEV-2.

## 9. Risks + mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Subsystem #5 too coupled to ship as additive flag | Medium | High | If can't ship under flag, fall back to greenfield branch for #5 only; #1-4 still ship additively |
| Nemotron Parse model unavailable / paywalled | Low | Medium | Test Nemotron Nano VL alternative first; if both blocked, fall back to Qwen2.5-VL with table-specific system prompt at slightly higher cost-per-PDF |
| Hand-graded ground truth needs clinician time we don't have | Medium | Medium | Defer #1 + #5 primary metrics; ship behind flag with universal regression only as initial gate; backfill ground truth later |
| Schema definitions (#3) become design bottleneck | Medium | Medium | Define all 10 schemas in a separate brainstorm session before #3 implementation starts; reject scope creep within #3 |
| Feature flags accumulate technical debt | High over time | Low | Time-box V4-equivalent paths to 30 days post-V5-default-on; calendared removal PR |
| Smoke set too narrow, misses regression | Low | Medium | Promote PDFs from full-regression to smoke if they catch issues smoke misses; smoke composition is mutable |
| RunPod community pool has no 4090 stock when we need to test | Medium | Low | Fall back to RunPod secure tier ($0.69-0.79/hr) or A6000 ($0.33/hr, slightly slower); both adequate |
| GHCR push reliability (Docker Desktop proxy) | Medium | Low | Already experienced; mitigated by `--max-concurrent-uploads=1` and Docker Desktop restart between attempts |

## 10. References

| Source | Use |
|--------|-----|
| Nemotron Parse / Nemotron Nano VL papers + HF cards | Subsystem #1 model selection |
| Docling DoclingDocument schema | Subsystem #2 provenance schema reference |
| Guideline2Graph (Georgia Tech / MetaDialog 2026) | Subsystem #5 architectural pattern |
| MedKGent agent framework (2025) | Subsystem #3 + #5 confidence-scored triple inspiration |
| MonkeyOCR v1.5 | V4 backbone — kept |
| PaddleOCR-VL | Coarse-to-fine routing inspiration (subsystem #1) |
| Consensus Entropy paper (CE-Ensemble) | Subsystem #4 |
| OmniDocBench v1.5 / olmOCR-Bench | External benchmarks for sanity-check |

## 11. Approval + transition to implementation

This master spec governs the cross-cutting concerns of V5. Each of the 5 subsystems requires its own focused brainstorm session producing its own spec, then `superpowers:writing-plans` produces an implementation plan, then `superpowers:executing-plans` (or equivalent) drives the implementation in test-driven steps.

**Sub-project sequencing**:

1. Subsystem A (#2 Bbox Provenance) — brainstorm next
2. Subsystem B (#1 Table Specialist)
3. Subsystem C (#4 Consensus Entropy gate)
4. Subsystem D (#3 Schema-first extraction)
5. Subsystem E (#5 Decomposition)

Each sub-project's design doc, plan doc, and implementation lives behind its respective `V5_<FEATURE>` flag. V4 is unchanged until each flag's acceptance gates pass and default flips.

---

**End of master spec.** Sub-project specs to follow.
