# Page 44 Audit — SGLT2i Kidney Outcomes (CREDENCE, DAPA-CKD, SCORED) + Harms

| Field | Value |
|-------|-------|
| **Page** | 44 (PDF page S43) |
| **Content Type** | SGLT2i kidney trial outcomes (CREDENCE, DAPA-CKD, SCORED, EMPA-KIDNEY) + meta-analysis (39,000 participants) + real-world data + SGLT2i harms (DKA, fractures, genital mycotic infections) |
| **Extracted Spans** | 25 total (3 T1, 22 T2) |
| **Channels** | B, C, F |
| **Disagreements** | 3 |
| **Review Status** | PENDING: 25 |
| **Risk** | Disagreement |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — counts confirmed (25), channels confirmed (B/C/F), disagreements added (3), review status added |

---

## Source PDF Content

**Kidney Outcome Trials:**
- **eGFR 30-60 meta-analysis**: Composite kidney HR 0.67 (0.51-0.89)
- **VERTIS-CV**: Secondary kidney outcome HR 0.81 (0.63-1.04) — not significant
- **CREDENCE**: Primary kidney outcome (kidney failure, doubling SCr, death) HR 0.70 (0.59-0.82); dialysis/transplant/kidney death HR 0.72 (0.54-0.97); canagliflozin 100mg daily; eGFR 30-90 + ACR 300-5000; stopped early for superiority
- **DAPA-CKD**: Primary kidney outcome HR 0.61 (0.51-0.72); dapagliflozin; eGFR 25-75 + ACR 200-5000; similar with/without T2D
- **SCORED**: Secondary kidney endpoint HR 0.71 (0.46-1.08); eGFR 25-60
- **Updated meta-analysis (39,000 patients)**: Dialysis/transplant/kidney death RR 0.67 (0.52-0.86); reduction in kidney failure and AKI; benefits across all eGFR subgroups including eGFR 30-45
- **Real-world data**: Composite kidney outcome HR 0.49 (0.35-0.67) — 51% reduction
- **EMPA-KIDNEY**: eGFR ≥20 to <45 or ≥45 to <90 + ACR ≥200; stopped early for positive results

**SGLT2i Harms (CRITICAL SAFETY CONTENT):**
- **Diabetic ketoacidosis (DKA)**: Rare in T2D (<1 per 1000 patient-years); CREDENCE: 2.2 vs 0.2 per 1000 patient-years
- **Fractures**: Higher in CANVAS (canagliflozin); NOT in CREDENCE (100mg dose) or CANVAS-R
- **Genital mycotic infections**: Consistent across all trials; CREDENCE: 2.27% vs 0.59%; manageable with topical antifungals

---

## Key Spans Assessment

### Tier 1 Spans (3)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| **"SGLT2i reduced the risk of adverse kidney outcomes (composite worsening kidney failure, kidney failure, or death from ki..."** | B,F | 98% | **→ T2** — Meta-analysis result with drug class + HR (evidence data, not prescribing instruction) |
| **"SGLT2i conferred less annual eGFR decline and a reduction in albuminuria or decreased progression to severely increased..."** | B,C,F | 100% | **→ T2** — Drug class benefit summary across trials (evidence synthesis, not prescribing) |
| **"This analysis, which included nearly 39,000 participants with T2D, found that SGLT2i significantly reduced the risk of d..."** | B,F | 98% | **→ T2** — Meta-analysis result: 33% reduction in dialysis/transplant/kidney death (evidence data) |

### Tier 2 Spans (22)

| Category | Count | Assessment |
|----------|-------|------------|
| `<!-- PAGE 44 -->` | 1 | **⚠️ PIPELINE ARTIFACT** — Reject |
| **"serum creatinine"** (C channel, 85%) | **15** | **ALL → T3** — Lab test name extracted 15 times without any clinical context |
| **"creatinine"** (C channel, 85%) | **3** | **ALL → T3** — Lab name fragment extracted 3 times |
| **"There was also reduction in kidney failure and AKI."** | 1 | **✅ T2 OK** — Meta-analysis conclusion (but should include HR/CI for full value) |

---

## Critical Findings

### ❌ WORST C Channel Over-Decomposition: "serum creatinine" × 15
The C (Grammar/Regex) channel extracted the lab test name "serum creatinine" **15 separate times** and "creatinine" **3 additional times** — totaling **18 of 22 T2 spans** (82%) being the same lab test name without any clinical context.

This is the **third-worst over-decomposition** in the entire audit:
1. Page 39: "Mean difference" × 38 (D channel)
2. Page 33: "Cochrane systematic" × 15 (D channel)
3. **Page 44: "serum creatinine" × 18 (C channel)** ← NEW

The C channel regex presumably matches "creatinine" as a lab test term, but every mention of "doubling of serum creatinine" in trial endpoint definitions triggers extraction — creating massive noise from a narrative page.

### ⚠️ T1 Spans Are Evidence, Not Prescribing
All 3 T1 spans describe meta-analysis results showing SGLT2i benefit on kidney outcomes. While clinically important, these are T2 evidence summaries — they describe "what trials found" not "what to prescribe." The B channel triggers on "SGLT2i" drug class name, elevating F channel evidence sentences to T1.

### ❌ SGLT2i HARMS SECTION COMPLETELY MISSING (CRITICAL)
The bottom of this page contains the **first mention of SGLT2i adverse effects** — content that is **quintessentially T1 (patient safety)**:

| Missing Harm Content | Tier | Clinical Importance |
|----------------------|------|---------------------|
| DKA risk: <1 per 1000 patient-years in T2D | **T1** | Adverse effect incidence |
| CREDENCE DKA: 2.2 vs 0.2 per 1000 patient-years | **T1** | Drug-specific adverse effect rate |
| Fracture risk with canagliflozin (CANVAS) | **T1** | Drug-specific bone safety |
| No excess fractures at canagliflozin 100mg (CREDENCE) | **T1** | Dose-dependent safety |
| Genital mycotic infections: consistent across all SGLT2i trials | **T1** | Class-wide adverse effect |
| CREDENCE: genital infections 2.27% vs 0.59% | **T1** | Drug-specific adverse effect rate |
| "Most infections manageable with topical antifungals" | **T1** | Adverse effect management |

### ❌ Other Missing Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| CREDENCE: primary kidney HR 0.70 (0.59-0.82); canagliflozin 100mg daily | **T2** | Key kidney trial result |
| CREDENCE: eGFR 30-90 + ACR 300-5000 enrollment | **T2** | Trial population (defines applicability) |
| CREDENCE: "maximum tolerated dose of ACEi or ARB" as standard of care | **T1** | Background therapy requirement |
| DAPA-CKD: primary HR 0.61 (0.51-0.72); eGFR 25-75 + ACR 200-5000 | **T2** | Key kidney trial result |
| DAPA-CKD: benefit similar with and without T2D | **T2** | Extends applicability beyond diabetes |
| EMPA-KIDNEY: eGFR ≥20 enrollment + non-albuminuric CKD included | **T2** | Broadest trial population |
| Real-world: composite kidney HR 0.49 (0.35-0.67) | **T2** | Generalizability evidence |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — SGLT2i harms section (DKA, fractures, genital infections) completely missing; "serum creatinine" × 18 noise; C channel systemic failure |
| **Tier corrections** | 3 T1 (meta-analysis results): T1 → T2; 18 "serum creatinine"/"creatinine": T2 → T3; Pipeline artifact: REJECT |
| **Missing T1** | DKA incidence, fracture risk (canagliflozin), genital mycotic infections, CREDENCE background ACEi/ARB requirement |
| **Missing T2** | CREDENCE/DAPA-CKD primary outcome HRs, enrollment criteria, EMPA-KIDNEY population |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~15% — 3 meta-analysis summary sentences captured; SGLT2i harms completely missing |
| **Tier accuracy** | ~4% (1/25 correctly tiered — "reduction in kidney failure and AKI" as T2) |
| **False positive rate** | 96% — 18/22 T2 are "serum creatinine" repeats; 3/3 T1 should be T2 |
| **Genuine T1 content** | 0 extracted (7 harms spans missing) |
| **Noise ratio** | 76% — "serum creatinine" × 18 out of 25 total spans |
| **Overall quality** | **POOR** — C channel regex matching creates massive noise; critical SGLT2i safety content (DKA, fractures, infections) not extracted |

---

## Review Actions Completed (2026-02-27)

### API Actions (reviewer: claude-auditor)

| Action | Count | Details |
|--------|-------|---------|
| **REJECT** | 21 | 1 pipeline artifact (`<!-- PAGE 44 -->`) + 17 "serum creatinine" + 3 "creatinine" — all `out_of_scope` |
| **CONFIRM** | 4 | 3 T1 meta-analysis spans (with T1→T2 correction notes) + 1 T2 evidence span |

### UI-Added Facts (5 REVIEWER spans)

| # | Fact Added | Target KB | L3 Extraction Value |
|---|-----------|-----------|---------------------|
| 1 | DKA risk: <1/1000 pt-yrs (meta-analysis); CREDENCE 2.2 vs 0.2/1000 pt-yrs (canagliflozin) | KB-4 Safety | adverse_effect=DKA, drug_class=SGLT2i, drug=canagliflozin, incidence rates |
| 2 | Fracture risk: CANVAS (canagliflozin) yes; CREDENCE 100mg no excess | KB-4 Safety | adverse_effect=fracture, drug=canagliflozin, dose_dependency |
| 3 | Genital mycotic infections: class-wide; CREDENCE 2.27% vs 0.59%; manageable with topical antifungals | KB-4 Safety | adverse_effect=genital_mycotic_infection, incidence, management |
| 4 | CREDENCE: eGFR 30-90, ACR 300-5000, canagliflozin 100mg, background ACEi/ARB, HR 0.70 (0.59-0.82) | KB-1 Dosing / KB-4 | drug=canagliflozin, dose=100mg, HR, concomitant_therapy |
| 5 | DAPA-CKD: eGFR 25-75, ACR 200-5000, dapagliflozin, HR 0.61 (0.51-0.72), benefit with/without T2D | KB-4 Evidence | drug=dapagliflozin, HR, population, applicability |

### Raw PDF Gap Analysis (2026-02-27)

Cross-checked all 9 verified spans against raw PDF text. Found 4 gaps — 3 HIGH priority, 1 MODERATE.

| # | Gap Fact (exact PDF text) | Priority | Target KB | API Result |
|---|--------------------------|----------|-----------|------------|
| 6 | EMPA-KIDNEY: eGFR ≥20 to <45 or ≥45 to <90 + ACR ≥200, non-albuminuric CKD included, stopped early | HIGH | KB-4, KB-16 | 201 (UI modal) |
| 7 | CREDENCE secondary: dialysis/transplant/kidney death HR 0.72 (0.54–0.97) | HIGH | KB-4 | 201 |
| 8 | SGLT2i kidney benefits across ALL eGFR subgroups including eGFR 30–45 | HIGH | KB-16 | 201 |
| 9 | Real-world registry: HR 0.49 (0.35–0.67), 51% reduction, generalizable to clinical practice | MODERATE | KB-4 | 201 |

**Acceptable omissions** (no action):
- VERTIS-CV HR 0.81 (0.63–1.04) — non-significant
- SCORED HR 0.71 (0.46–1.08) — non-significant
- CREDENCE context (stopped early, 50% CVD, 2.6yr) — enrichment detail, not standalone fact

### Post-Review State (Final)

| Metric | Before | After Round 1 | After Gap Fill |
|--------|--------|---------------|----------------|
| **Total spans** | 25 | 30 | 34 |
| **Reviewed** | 0/25 | 30/30 | 34/34 |
| **Confirmed** | 0 | 4 | 4 |
| **Added (REVIEWER)** | 0 | 5 | 9 |
| **Rejected** | 0 | 21 | 21 |
| **Pipeline 2 ready** | No | 9 spans | **13 spans** (4 confirmed + 9 added) |
| **Safety content** | 0 extracted | 3 harms facts | 3 harms + EMPA-KIDNEY population |
| **Extraction completeness** | ~15% | ~55% | **~90%** (only non-significant trials omitted) |
