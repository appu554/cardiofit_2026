# Page 98 Audit — Methods: Guideline Development, Table 1 Outcome Hierarchy

## Page Identity
- **PDF page**: S97 (Methods for Guideline Development — www.kidney-international.org)
- **Content**: Methods overview (scope, process, Work Group, ERT, literature search), Table 1 (Hierarchy of Outcomes — critical, important, non-important)
- **Clinical tier**: T3 (Informational — methodology description, outcome classification table)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 1 |
| T1 (Patient Safety) | 0 |
| T2 (Clinical Accuracy) | 1 |
| T3 (Informational) | 0 |
| Channels present | F (NuExtract LLM) |
| Genuinely correct T2 | 0 |
| Tier accuracy | 0% (0/1) |
| Disagreements | 0 |
| Review Status | FINAL: 17 ADDED, 5 CONFIRMED, 1 REJECTED |
| Cross-Check | 2026-02-25 — Count corrected 6→1; D channel phantom removed; verified against raw extraction data |
| Raw PDF Cross-Check | 2026-02-28 — 5 agent spans CONFIRMED, 1 rejected (HTML artifact), 10 gaps added |
| Audit Date | 2026-02-25 (revised) |

## Channel Breakdown
| Channel | Count | Confidence | Content |
|---------|-------|------------|---------|
| D (Table Decomp) | 5 | 92% | Table 1 outcome names (critical + important + non-important) |
| F (NuExtract LLM) | 1 | 90% | HTML comment artifact |

## T2 Spans (6) — ALL MISTIERED

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "Albuminuria progression (onset of albuminuria, moderately increased to severely increased albuminuria)" | D | 92% | T3 | Table 1 "Important outcomes" row — outcome classification, not clinical threshold |
| 2 | "Individual cardiovascular events (myocardial infarction, stroke, heart failure)" | D | 92% | T3 | Table 1 "Critical outcomes" row |
| 3 | "3-point and 4-point MACE" | D | 92% | T3 | Table 1 "Critical outcomes" row |
| 4 | "Attaining HbA1c" | D | 92% | T3 | Table 1 "Critical outcomes" row |
| 5 | "Change in HbA1c" | D | 92% | T3 | Table 1 "Critical outcomes" row |
| 6 | "<!-- PAGE 98 -->" | F | 90% | NOISE/BUG | **HTML comment artifact** — 5th occurrence |

### D Channel Table 1 Analysis
The D (Table Decomp) channel correctly identified Table 1 and extracted 5 of the outcome names. However:
- **Missing critical outcomes**: All-cause mortality, Cardiovascular mortality, Kidney failure, Doubling of serum creatinine, Hypoglycemia requiring third-party assistance, Hyperkalemia — 6 out of 11 outcomes not extracted
- **Missing non-important outcome**: eGFR/creatinine clearance
- **No hierarchy context**: The "Critical", "Important", "Non-important" classification labels were not captured, so extracted outcomes lack their priority ranking

### F Channel HTML Artifact — 5th Occurrence
`<!-- PAGE 98 -->` at 90% confidence. Running tally: pages 91, 92, 94, 95, 98 (5 confirmed). Skipped pages 93, 96, 97.

## PDF Source Content Analysis

### Content Present on Page
1. **Methods overview** — Aim, overview of process (10-step list), commissioning of Work Group and ERT:
   - Scope: Update 2020 KDIGO guideline for diabetes + CKD
   - Process: AGREE II compliant, GRADE methodology
   - Work Group: Nephrology, cardiology, endocrinology, dietetics, epidemiology, primary care, public health + patient representatives
   - ERT: Cochrane Kidney and Transplant — systematic evidence review

2. **Defining scope** — Limited to RCTs for effectiveness/safety, clinical questions mapped to Cochrane systematic reviews

3. **Literature search** — Cochrane Kidney and Transplant Registry, GRADE standards

4. **Table 1: Hierarchy of Outcomes**:
   - **Critical outcomes** (10): All-cause mortality, Cardiovascular mortality, Kidney failure, 3-point and 4-point MACE, Individual cardiovascular events (MI, stroke, HF), Doubling of serum creatinine, Hypoglycemia requiring third-party assistance, Attaining HbA1c, Change in HbA1c, Hyperkalemia
   - **Important outcomes** (1): Albuminuria progression
   - **Non-important outcomes** (1): eGFR/creatinine clearance

### What Should Have Been Extracted
| Content | Correct Tier | Extracted? |
|---------|-------------|------------|
| Table 1 outcome hierarchy (all 12 rows) | T3 | Partially (5/12 outcomes) |
| "Critical", "Important", "Non-important" labels | T3 | NO |
| GRADE methodology reference | T3 | NO |
| AGREE II compliance statement | T3 | NO |
| Work Group composition | T3 | NO |
| "Hypoglycemia requiring third-party assistance" | T3 | NO (critical outcome missed) |
| "Hyperkalemia" | T3 | NO (critical outcome missed) |
| "All-cause mortality" / "Cardiovascular mortality" | T3 | NO (critical outcomes missed) |

## Cross-Page Patterns

### Transition from Clinical to Methods
Page 98 marks the transition from Chapter 5 clinical content to the Methods section. The extraction pattern shifts:
- No more B (Drug Dictionary) channel — no drug names in methods
- No more C temporal regex — no monitoring intervals
- D channel active on Table 1 (structured data)
- F channel continues HTML artifact pattern

### D Channel on Methods Tables
D channel extracted 5 of 12 Table 1 rows — 42% completeness. This is better than the forest plot extraction (pages 92-93) but still incomplete. The D channel appears to selectively extract longer, more descriptive outcome names while missing shorter ones ("Hyperkalemia", "Kidney failure").

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | LOW | 5/12 Table 1 rows, no methods prose |
| Tier accuracy | 0% | All 6 spans should be T3 or NOISE |
| Clinical safety risk | NONE | Methods section — zero clinical content |
| Channel diversity | LOW | D + F only |
| Noise level | HIGH | 1/6 pure noise (HTML artifact), 5/6 mistiered |
| Pipeline bugs | 1 | HTML comment artifact (5th occurrence) |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Entire page is guideline development methodology and outcome classification. No patient-facing clinical content.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 13
- **ADDED**: 7 (from agent bulk audit: All-cause mortality, Cardiovascular mortality, Kidney failure, Doubling of serum creatinine, Hypoglycemia requiring third-party assistance, Hyperkalemia, eGFR/creatinine clearance)
- **PENDING**: 5 (original D channel: Change in HbA1c, Albuminuria progression, Individual CV events, MACE, Attaining HbA1c)
- **REJECTED**: 1 (`<!-- PAGE 98 -->` HTML artifact)

### Agent Spans CONFIRMED (5)
| # | ID | Text | Note |
|---|-----|------|------|
| 1 | `4fb72332` | Change in HbA1c | Table 1 Critical outcome |
| 2 | `750b88f9` | Albuminuria progression (onset of albuminuria, moderately increased to severely increased albuminuria) | Table 1 Important outcome |
| 3 | `93256585` | Individual cardiovascular events (myocardial infarction, stroke, heart failure) | Table 1 Critical outcome |
| 4 | `1f0879cf` | 3-point and 4-point MACE | Table 1 Critical outcome |
| 5 | `6c2624c8` | Attaining HbA1c | Table 1 Critical outcome |

### Agent Spans Kept (7 — from bulk audit, ADDED)
| # | ID | Text | Note |
|---|-----|------|------|
| 1 | `a65ec47a` | All-cause mortality | Table 1 Critical outcome |
| 2 | `d25c1ff5` | Cardiovascular mortality | Table 1 Critical outcome |
| 3 | `7fb283ed` | Kidney failure | Table 1 Critical outcome |
| 4 | `a3e01f50` | Doubling of serum creatinine | Table 1 Critical outcome |
| 5 | `41bc85c7` | Hypoglycemia requiring third-party assistance | Table 1 Critical outcome |
| 6 | `a8b06adf` | Hyperkalemia | Table 1 Critical outcome |
| 7 | `131c3032` | eGFR/creatinine clearance | Table 1 Non-important outcome |

### Gaps Added (10) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G98-A | The aim of this project was to update the evidence-based clinical practice guideline for the monitoring, prevention of disease progression, and treatment in patients with diabetes and CKD published in 2020. | Guideline aim |
| G98-B | These guidelines adhered to international best practices for guideline development, and have been reported in accordance with the Appraisal of Guidelines for Research and Evaluation (AGREE) II reporting checklist. | AGREE II compliance |
| G98-C | Defining the scope of the guideline update. Implementing literature search strategies... Finalizing and publishing the guideline. | 11-step development process |
| G98-D | the previously assembled Work Group with expertise in adult nephrology, cardiology, endocrinology, dietetics, epidemiology, primary care, and public health, as well as people living with diabetes and kidney disease were engaged. | Work Group composition |
| G98-E | Cochrane Kidney and Transplant, with expertise in adult and pediatric nephrology, evidence synthesis, and guideline development, was again contracted as the ERT tasked with updating the systematic evidence review. | ERT — Cochrane role |
| G98-F | The Work Group was responsible for writing the graded recommendations and the underlying rationale, grading the strength of the recommendations, and developing practice points. | Work Group responsibilities |
| G98-G | clinical questions on effectiveness and safety of interventions included in the guideline update were limited to RCTs. Clinical questions were mapped to existing Cochrane Kidney and Transplant systematic reviews... | Scope — limited to RCTs |
| G98-H | All evidence reviews were conducted in accordance with the Cochrane Handbook, and guideline development adhered to the standards of GRADE (Grading of Recommendation, Assessment, Development, and Evaluation). | GRADE + Cochrane methodology |
| G98-I | Searches for RCTs utilized the Cochrane Kidney and Transplant Registry of studies. | Literature search source |
| G98-J | Table 1 \| Hierarchy of outcomes. Critical outcomes: All-cause mortality, Cardiovascular mortality, Kidney failure, 3-point and 4-point MACE, Individual cardiovascular events... Important outcomes: Albuminuria progression... Non-important outcomes: eGFR/creatinine clearance. | Table 1 with hierarchy labels |

### Post-Review State
- **Total spans**: 23
- **ADDED**: 17 (7 from bulk audit + 10 new gaps)
- **CONFIRMED**: 5 (original D channel outcomes)
- **REJECTED**: 1 (HTML artifact)
- **P2-ready facts**: 22

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G98-A–C | KB-7 (Terminology) | Guideline methodology — aim, AGREE II, process |
| G98-D–F | KB-7 (Terminology) | Work Group + ERT composition and roles |
| G98-G–I | KB-7 (Terminology) | Scope, GRADE methodology, literature search |
| G98-J | KB-7 (Terminology) | Table 1 outcome hierarchy with classification labels |
| All 12 outcome spans | KB-16 (Monitoring) | Outcome definitions for evidence weighting |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 98 of 126
