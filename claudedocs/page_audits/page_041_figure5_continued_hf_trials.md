# Page 41 Audit — Figure 5 (Continued): SGLT2i Heart Failure and Additional Trials

| Field | Value |
|-------|-------|
| **Page** | 41 (PDF page S40) |
| **Content Type** | Figure 5 continuation: Heart failure trials (DAPA-HF, EMPEROR-Reduced/Preserved, DELIVER) + additional CV trials (DECLARE-TIMI 58, VERTIS-CV, SCORED, SOLOIST-WHF) |
| **Extracted Spans** | 22 total (9 pipeline + 13 REVIEWER) |
| **Channels** | C, D, F, REVIEWER |
| **Disagreements** | 0 |
| **Review Status** | 3 confirmed + 6 rejected + 13 REVIEWER added = 22/22 reviewed |
| **Risk** | Clean |
| **Page Decision** | **ACCEPTED** |
| **Audit Date** | 2026-02-27 (executed) |
| **Cross-Check** | Verified against raw PDF — API returned 9 pipeline spans; 13 REVIEWER facts added via UI (5 CV outcomes + 6 kidney outcomes + 2 drug dosings) |

---

## Source PDF Content

**Figure 5 (Continued) — SGLT2i Heart Failure and Additional Trials:**

| Trial | Drug/Dose | N | eGFR Criteria | CV Key Result | Kidney Outcome HR |
|-------|-----------|---|---------------|---------------|-------------------|
| DECLARE-TIMI 58 | Dapagliflozin 10 mg daily | 17,160 | CrCl ≥60 | MACE HR 0.93 (0.84-1.03); CV death/HF HR 0.83 (0.73-0.95) | HR 0.76 (0.67-0.87) |
| VERTIS-CV | Ertugliflozin 5/15 mg daily | 8246 | eGFR ≥30 | MACE HR 0.97 (0.85-1.11) | HR 0.81 (0.63-1.04) |
| SCORED | Sotagliflozin 200/400 mg daily | 10,584 | eGFR 25-60 | CV death/HF/urgent visit HR 0.74 (0.63-0.88) | HR 0.71 (0.46-1.08) |
| SOLOIST-WHF | Sotagliflozin 200/400 mg daily | 1222 | No eGFR criteria | CV death/HF/urgent visit HR 0.67 (0.52-0.85) | Not reported |
| DAPA-HF | Dapagliflozin 10 mg daily | 4744 | No eGFR criteria | Primary composite HR 0.74 (0.65-0.85) | HR 0.71 (0.44-1.16) |
| EMPEROR-Reduced | Empagliflozin 10 mg daily | 3730 | eGFR ≥20 | CV death/HF HR 0.75 (0.65-0.86) | **HR 0.50 (0.32-0.77)** |
| EMPEROR-Preserved | Empagliflozin 10 mg daily | 5988 | eGFR ≥20 | CV death/HF HR 0.79 (0.69-0.90) | N/A |
| DELIVER | Dapagliflozin 10 mg daily | 6263 | eGFR ≥25 | [Met primary endpoint] | HR 0.95 (0.73-1.24) |

**Note**: DELIVER CV outcome says "[Met primary endpoint]" on this page — the specific HR 0.82 (0.73-0.92) is from the published trial, not printed here.

---

## Key Spans Assessment

### Tier 1 Spans (3) — All REJECTED

| Span | Channel | Conf | Status | Assessment |
|------|---------|------|--------|------------|
| "CrCl ≥60" | C | 95% | **REJECTED** | Trial enrollment criterion (DECLARE-TIMI 58), not prescribing threshold |
| "GFR ≥50" | C | 95% | **REJECTED** | Trial enrollment criterion, not drug initiation threshold |
| "eGFR ≥50" | C | 95% | **REJECTED** | Trial enrollment criterion, not drug initiation threshold |

### Tier 2 Spans (6) — 3 CONFIRMED, 3 REJECTED

| Span | Channel | Conf | Status | Assessment |
|------|---------|------|--------|------------|
| "Deaths from CV causes, hospitalizations for HSE, and urgent visits for HF" | D | 92% | **CONFIRMED** | Composite endpoint definition (SCORED/SOLOIST-WHF) |
| "Deaths from CV causes and hospitalizations and urgent visits for HF" | D | 92% | **CONFIRMED** | Composite endpoint definition (variant) |
| "CV death or worsening HF" | D | 92% | **CONFIRMED** | Composite endpoint definition (DAPA-HF/EMPEROR/DELIVER) |
| `<!-- PAGE 41 -->` | F | 90% | **REJECTED** | Pipeline HTML artifact |
| "Figure 5 \| (Continued)" | F | 85% | **REJECTED** | Figure continuation label — no clinical content |
| "eGFR 60-90" | C | 90% | **REJECTED** | Trial enrollment range, not prescribing threshold |

---

## REVIEWER Facts Added (13)

### Phase 1: CV Outcome HRs (5)

| # | Trial | Verbatim Text (truncated) | Reviewer Note |
|---|-------|---------------------------|---------------|
| 1 | DAPA-HF | "DAPA-HF: Dapagliflozin 10 mg once daily, 4744 participants... Primary outcome HR 0.74 (0.65–0.85)" | KB-4 safety — HF trial CV outcome |
| 2 | EMPEROR-Reduced | "EMPEROR-Reduced: Empagliflozin 10 mg once daily, 3730 participants, eGFR ≥20... HR 0.75 (0.65–0.86)" | KB-4 safety — HFrEF CV outcome |
| 3 | EMPEROR-Preserved | "EMPEROR-Preserved: Empagliflozin 10 mg once daily, 5988 participants, eGFR ≥20... HR 0.79 (0.69–0.90)" | KB-4 safety — HFpEF CV outcome |
| 4 | DELIVER | "DELIVER: Dapagliflozin 10 mg once daily, 6263 participants... [Met primary endpoint]" | KB-4 safety — DELIVER CV outcome |
| 5 | SOLOIST-WHF | "SOLOIST-WHF: Sotagliflozin 200/400 mg once daily, 1222 participants... HR 0.67 (0.52–0.85)" | KB-4 safety — dual SGLT1/SGLT2i CV outcome |

### Phase 2: Kidney Outcome HRs (6) — from raw PDF cross-check

| # | Trial | Kidney Outcome HR | Significance | Reviewer Note |
|---|-------|-------------------|-------------|---------------|
| 6 | DECLARE-TIMI 58 | HR 0.76 (0.67–0.87) | **Significant** | KB-4 — SGLT2i renal protection |
| 7 | VERTIS-CV | HR 0.81 (0.63–1.04) | NS | KB-4 — ertugliflozin kidney outcome |
| 8 | SCORED | HR 0.71 (0.46–1.08) | NS | KB-4 — dual SGLT1/SGLT2i kidney in CKD |
| 9 | DAPA-HF | HR 0.71 (0.44–1.16) | NS | KB-4 — dapagliflozin kidney in HF |
| 10 | EMPEROR-Reduced | **HR 0.50 (0.32–0.77)** | **Highly significant** | KB-4 — empagliflozin 50% kidney risk reduction |
| 11 | DELIVER | HR 0.95 (0.73–1.24) | NS | KB-4 — dapagliflozin kidney in HFpEF |

### Phase 3: Drug Dosings (2) — originally flagged as missing T1

| # | Drug | Verbatim Text (truncated) | Reviewer Note |
|---|------|---------------------------|---------------|
| 12 | Ertugliflozin | "VERTIS-CV: Ertugliflozin 5 mg or 15 mg once daily, 8246 participants, eGFR ≥30..." | KB-1 dosing — unique SGLT2i not on page 40 |
| 13 | Sotagliflozin | "SCORED/SOLOIST-WHF: Sotagliflozin 200 mg or 400 mg once daily. Dual SGLT1/SGLT2 inhibitor..." | KB-1 dosing — dual SGLT1/SGLT2i, unique mechanism |

---

## Critical Findings

### ❌ Zero Genuine T1 Content (Pipeline)
All 3 T1 spans are decontextualized eGFR/CrCl thresholds from trial enrollment criteria (CrCl ≥60, GFR ≥50, eGFR ≥50). These are NOT drug prescribing thresholds — they describe which patients were enrolled in trials. All REJECTED.

### ✅ Three Genuine T2 Endpoint Definitions
The D channel captured 3 composite endpoint descriptions that provide context for trial results. These are correctly T2 — they define what was measured, not prescribing instructions. All CONFIRMED.

### ❌ D Channel Table Decomposition Failed on Continuation Page
Page 40 successfully extracted 5 drug dosing spans (empagliflozin, canagliflozin, dapagliflozin doses). Page 41 contains 4 additional drugs/doses — NONE were extracted. The D channel table decomposition succeeded on the first half of Figure 5 but completely failed on the continuation page. All drug dosing and trial outcomes added via REVIEWER.

### ✅ 13 REVIEWER Facts Fill All Major Gaps
- **5 CV outcome HRs**: DAPA-HF, EMPEROR-Reduced, EMPEROR-Preserved, DELIVER, SOLOIST-WHF
- **6 kidney outcome HRs**: DECLARE, VERTIS-CV, SCORED, DAPA-HF, EMPEROR-Reduced (HR 0.50!), DELIVER
- **2 drug dosings**: Ertugliflozin 5/15mg, sotagliflozin 200/400mg

### ⚠️ DELIVER CV Outcome Discrepancy
The raw PDF table for DELIVER shows "[Met primary endpoint]" — NOT the specific HR 0.82 (0.73-0.92). The earlier REVIEWER fact (Phase 1 #4) references the published DELIVER result, not verbatim page text. The kidney outcome HR 0.95 (0.73-1.24) IS verbatim from the table.

### ⚠️ EMPEROR-Reduced Kidney HR 0.50 — Key KDIGO Evidence
EMPEROR-Reduced showed a 50% reduction in kidney composite outcome (HR 0.50; 95% CI: 0.32-0.77). This is one of the strongest renal protection signals in the SGLT2i evidence base and directly relevant to KDIGO's kidney-focused guidelines.

---

## Final Disposition

| Action | Details |
|--------|---------|
| **Decision** | **ACCEPTED** |
| **Total Spans** | 22 (9 pipeline + 13 REVIEWER) |
| **Pipeline Confirmed** | 3 (endpoint definitions) |
| **Pipeline Rejected** | 6 (3 enrollment thresholds, pipeline artifact, figure label, eGFR range) |
| **REVIEWER Added** | 13 (5 CV outcomes + 6 kidney outcomes + 2 drug dosings) |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~85% — 13 REVIEWER additions fill CV outcomes, kidney outcomes, and drug dosing gaps |
| **Tier accuracy (pipeline)** | 33% (3/9 pipeline spans genuine) |
| **False positive T1 rate** | 100% (3/3 T1 are decontextualized enrollment thresholds) |
| **Genuine content** | 16 total (3 pipeline confirmed + 13 REVIEWER added) — was 3 pre-review |
| **REVIEWER Added** | 13 (in 3 phases: CV, kidney, dosing) |
| **Cross-check findings** | DELIVER CV HR not verbatim; kidney outcome column discovered via raw PDF |
| **Overall quality** | **VERY GOOD** — Dense table page required massive REVIEWER supplementation; all major trial data now captured including kidney outcomes critical for KDIGO |
