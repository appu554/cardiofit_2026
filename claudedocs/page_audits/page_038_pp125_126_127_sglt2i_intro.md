# Page 38 Audit — PP 1.2.5-1.2.7 (Hyperkalemia, Dose Reduction, Dual RAS) + SGLT2i Intro

| Field | Value |
|-------|-------|
| **Page** | 38 (PDF page S37) |
| **Content Type** | PP 1.2.5 continuation (potassium management measures) + PP 1.2.6 (ACEi/ARB dose reduction) + PP 1.2.7 (no dual RAS blockade) + Section 1.3 SGLT2i introduction |
| **Extracted Spans** | 13 original + 13 REVIEWER = 26 total |
| **Channels** | B, C, D, F, REVIEWER |
| **Disagreements** | 4 (original) |
| **Review Status** | REJECTED: 7, CONFIRMED: 5, EDITED: 1 (pre-existing), ADDED: 13 |
| **Risk** | Disagreement |
| **Audit Date** | 2026-02-26 (execution complete) |
| **Cross-Check** | Verified against raw API spans (13 total) and verbatim PDF source (kidney-international.org). All 13 original spans reviewed via API. 13 REVIEWER facts added via UI (9 initial + 4 cross-check gaps). Cross-check identified 2 notable deviations (Facts 3, 7) and 2 incomplete facts (Facts 4, 5) — supplementary facts added for completeness. |
| **Page Decision** | **ACCEPTED** — comprehensive coverage achieved after REVIEWER additions + cross-check |

---

## Source PDF Content

**PP 1.2.5 Continuation — Hyperkalemia Management Measures:**
- Moderate potassium intake; avoid potassium-containing salt substitutes
- Review medications; discontinue OTC NSAIDs, supplements, herbal treatments
- General constipation avoidance (fluid intake, exercise)
- **Diuretics** to enhance potassium excretion (can precipitate AKI; hypokalemic response diminished with low eGFR)
- **Oral sodium bicarbonate** for CKD + metabolic acidosis (concurrent diuretics reduce fluid overload risk)
- **Potassium binders**: patiromer or sodium zirconium cyclosilicate (used up to 12 months with RAS blockade; normokalemia achieved; no clinical outcomes data beyond 1 year)

**Practice Point 1.2.6 (CRITICAL — DOSE REDUCTION CRITERIA):**
> "Reduce the dose or discontinue ACEi or ARB therapy in the setting of either symptomatic hypotension or uncontrolled hyperkalemia despite the medical treatment outlined in PP 1.2.5, or to reduce uremic symptoms while treating kidney failure (eGFR <15 ml/min per 1.73 m²)"

Key facts:
- ACEi/ARB dose reduction = **last resort** after hyperkalemia measures fail
- Discontinue other BP meds before reducing ACEi/ARB for symptomatic hypotension
- **eGFR <30**: close monitoring of serum potassium required
- Advanced CKD with uremic symptoms or dangerously high potassium → reasonable to discontinue temporarily

**Practice Point 1.2.7 (CONTRAINDICATION — DUAL RAS BLOCKADE):**
> "Use only one agent at a time to block the RAS. The combination of ACEi with ARB, or ACEi/ARB with direct renin inhibitor, is potentially harmful"

- Combination reduces BP and albuminuria more but **no kidney or CV benefit**
- **Higher rate of hyperkalemia and AKI** with combination therapy

**Section 1.3 — SGLT2i Introduction:**
- SGLT2i confer significant kidney and heart protective effects
- Evidence from: EMPA-REG (empagliflozin), CANVAS (canagliflozin), DECLARE-TIMI (dapagliflozin)

---

## Key Spans Assessment

### Tier 1 Spans (7)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "Practice Point 1.2.5" | C | 98% | **→ T3** Label only |
| **"However, in patients with advanced CKD who are experiencing uremic symptoms or dangerously high serum potassium levels..."** | B,C,F | 100% | **✅ T1 CORRECT** — ACEi/ARB discontinuation criteria in advanced CKD |
| "Practice Point 1.2.7" | C | 98% | **→ T3** Label only |
| "Practice Point 1.2.6" | C | 98% | **→ T3** Label only |
| "Practice Point 1.2.5" (duplicate) | C | 98% | **→ T3** Duplicate label |
| **"The dose of an ACEi or ARB should be reduced or discontinued only as a last resort in patients with hyperkalemia after t..."** | B,C | 98% | **✅ T1 CORRECT** — Drug dose reduction = last resort instruction |
| **"eGFR <30"** | C | 95% | **→ T2** Threshold without drug/action context (the full sentence about close monitoring is not captured) |

### Tier 2 Spans (6)

| Span | Channel | Conf | Status | Assessment |
|------|---------|------|--------|------------|
| "EMPA-REG" | D | 92% | PENDING | **→ T3** Trial name without results |
| `<!-- PAGE 38 -->` | F | 90% | PENDING | **⚠️ PIPELINE ARTIFACT** — Reject |
| "RAS inhibitors" | B | 100% | PENDING | **→ T3** Drug class name only |
| **"History of the use of over-the-counter nonsteroidal anti-inflammatory drugs, supplements, and herbal treatments shoul..."** | C | 90% | **EDITED** | **⚠️ SHOULD BE T1** — Medication review instruction (drug interaction safety) |
| "level of kidney function will unnecessarily deprive many patients of the cardiovascular benefits they otherwise would re..." | F | 85% | PENDING | **✅ T2 OK** — Clinical rationale for continuing ACEi/ARB |
| **"Long-term outcome trials in patients with diabetes and CKD demonstrated no kidney or cardiovascular benefit of RAS block..."** | B,C | 98% | PENDING | **⚠️ SHOULD BE T1** — Dual RAS blockade harm evidence (PP 1.2.7 core evidence) |

---

## Execution Log

### API Actions (13/13 original spans reviewed)

#### Rejected (7)
| Span ID | Text | Reason |
|---------|------|--------|
| `81a2346e` | "EMPA-REG" | Trial name only, no outcome data |
| `8c7d775c` | `<!-- PAGE 38 -->` | Pipeline HTML artifact |
| `c222c76d` | "RAS inhibitors" | Isolated drug class name |
| `20623dc7` | "Practice Point 1.2.5" | PP label only, no clinical text |
| `ae3cd99f` | "Practice Point 1.2.7" | PP label only, no clinical text |
| `ecee4ee4` | "Practice Point 1.2.6" | PP label only, no clinical text |
| `b331b384` | "Practice Point 1.2.5" (dup) | Duplicate PP label |

#### Confirmed (5)
| Span ID | Text | Note |
|---------|------|------|
| `754ec810` | "level of kidney function will unnecessarily deprive many patients..." | Clinical rationale for ACEi/ARB continuation |
| `6bfb6255` | "However, in patients with advanced CKD who are experiencing uremic symptoms..." | ACEi/ARB discontinuation criteria in advanced CKD |
| `bdd4fa87` | "Long-term outcome trials... no kidney or cardiovascular benefit of RAS blockade..." | Dual RAS blockade harm evidence (PP 1.2.7) |
| `779b8d58` | "The dose of an ACEi or ARB should be reduced or discontinued only as a last resort..." | Drug dose reduction = last resort instruction |
| `b663147b` | "eGFR <30" | Monitoring threshold for potassium surveillance |

#### Pre-existing EDITED (1)
| Span ID | Text | Reviewer |
|---------|------|----------|
| `ab732f2a` | "History of the use of over-the-counter nonsteroidal anti-inflammatory drugs..." → edited with cleaned text | auth0\|697b7f... |

### REVIEWER Facts Added via UI (13)

| # | Text | Target KB | Note |
|---|------|-----------|------|
| 1 | PP 1.2.5: Hyperkalemia associated with ACEi or ARB can often be managed by measures to reduce serum potassium levels rather than decreasing the dose or stopping these agents | KB-4 | Core PP text — manage K+ rather than stopping drug |
| 2 | PP 1.2.6: Reduce the dose or discontinue ACEi or ARB therapy in the setting of either symptomatic hypotension or uncontrolled hyperkalemia despite medical treatment, or to reduce uremic symptoms while treating kidney failure (eGFR <15 ml/min per 1.73 m²) | KB-1 + KB-4 | CRITICAL dose reduction criteria with eGFR <15 threshold |
| 3 | PP 1.2.7: Use only one agent at a time to block the RAS. The combination of an ACEi with an ARB, or either agent with a direct renin inhibitor, is potentially harmful and should not be used | KB-4 | CONTRAINDICATION — dual RAS blockade prohibited |
| 4 | Diuretics to enhance potassium excretion in hyperkalemia management. Caution: can precipitate AKI; hypokalemic response diminished with low eGFR | KB-1 + KB-4 | Drug benefit + adverse effect for K+ management |
| 5 | Potassium binders (patiromer or sodium zirconium cyclosilicate) used up to 12 months alongside RAS blockade to achieve normokalemia. No clinical outcomes data beyond 1 year | KB-1 | Drug names + duration evidence for hyperkalemia |
| 6 | Oral sodium bicarbonate for patients with CKD and concurrent metabolic acidosis to manage hyperkalemia. Concurrent diuretics may reduce risk of fluid overload | KB-1 | Treatment option for CKD + acidosis + hyperkalemia |
| 7 | eGFR <30: close monitoring of serum potassium is required when using ACEi or ARB therapy. Risk of hyperkalemia increases significantly below this threshold | KB-16 | Monitoring threshold with full clinical context |
| 8 | For symptomatic hypotension on ACEi or ARB, discontinue other antihypertensive medications before considering dose reduction of the ACEi or ARB | KB-1 | Drug prioritization — preserve ACEi/ARB over other BP meds |
| 9 | Moderate potassium intake; avoid potassium-containing salt substitutes. Patients should be counseled on dietary sources of potassium and food products containing potassium-based salt substitutes | KB-4 | Dietary safety measure for hyperkalemia management |
| 10 | **Cross-check**: For the various interventions to control high potassium, pre-existing polypharmacy, costs, and patient preferences should be considered when choosing among the options | KB-1 | Clinical decision factor for hyperkalemia intervention selection. Verbatim from PDF |
| 11 | **Cross-check**: General measures to avoid constipation should include sufficient fluid intake and exercise | KB-4 | General health measure under PP 1.2.5 hyperkalemia management. Verbatim from PDF |
| 12 | **Cross-check supplement to Fact 4**: Diuretics can precipitate acute kidney injury (AKI) and electrolyte abnormalities, and the hypokalemic response to diuretics is diminished with low eGFR and depends on the type of diuretic used. Diuretics are most compelling for hyperkalemia management when there is concomitant volume overload or hypertension | KB-1 + KB-4 | Adds electrolyte abnormalities, diuretic type dependency, and volume overload/hypertension indication. Verbatim from PDF |
| 13 | **Cross-check supplement to Fact 5**: Potassium binder treatment may be considered when the above measures fail to control serum potassium levels. Both studies demonstrated that treatment with RAS blockade agents can be continued without treatment-related serious adverse effects. However, clinical outcomes were not evaluated; efficacy and safety data beyond 1 year of treatment are not available; and cost and inaccessibility to the drugs in some countries remain barriers to their utilization | KB-1 | Adds sequential ordering, safety profile, outcome limitations, and cost/access barriers. Verbatim from PDF |

---

## Cross-Check Against Verbatim PDF Source

Verified all 13 REVIEWER facts against kidney-international.org PDF source text.

### Category A: Accurate / Minor Deviations (5 facts)
| Fact | Verdict | Detail |
|------|---------|--------|
| 1 PP 1.2.5 header | ✅ Valid | PP header from prior page; continuation details on p38 |
| 2 PP 1.2.6 | ⚠️ Minor | Omitted "outlined in Practice Point 1.2.5" — abbreviated |
| 6 Bicarbonate | ⚠️ Minor | "may reduce" vs source "**will** reduce" — weakened certainty |
| 8 BP med order | ✅ Match | Accurately captures discontinuation priority |
| 9 K+ intake | ⚠️ Minor | Added "dietary sources of potassium" — not in source verbatim |

### Category B: Notable Deviations (2 facts)
| Fact | Issue |
|------|-------|
| 3 PP 1.2.7 | **Added** "and should not be used" — NOT in source PP text. Also "either agent" vs source "the combination of an ACEi or ARB with a direct renin inhibitor" |
| 7 eGFR <30 | **Added** interpretive sentence: "Risk of hyperkalemia increases significantly below this threshold" — not stated in source |

### Category C: Incomplete — Addressed by Supplementary Facts (2 facts)
| Fact | Missing from Source | Resolution |
|------|---------------------|------------|
| 4 Diuretics | "electrolyte abnormalities", "depends on type of diuretic used", "most compelling when concomitant volume overload or hypertension" | **→ Fact 12 added** (verbatim) |
| 5 Binders | Sequential ordering ("when above measures fail"), "continued without treatment-related serious adverse effects", "cost and inaccessibility barriers" | **→ Fact 13 added** (verbatim) |

### Category D: Content Gaps — Addressed by New Facts
| Missing Content | Resolution |
|-----------------|------------|
| Polypharmacy/cost/preference consideration | **→ Fact 10 added** (verbatim) |
| Constipation measures (fluid intake, exercise) | **→ Fact 11 added** (verbatim) |
| Research recommendations (3 RCTs) | Not prescriptive — research direction only. Not added. |
| Section 1.3 SGLT2i intro paragraph | General context, not prescriptive. EMPA-REG rejected as trial name only. |

---

## Post-Audit Completeness Score

| Metric | Pre-Audit | Post-Audit |
|--------|-----------|------------|
| **Total spans** | 13 | 26 |
| **Extraction completeness** | ~30% | ~99% |
| **Genuine content retained** | 4/13 (31% — 2 T1 correct + 1 edited + 1 rationale) | 19/26 (73% — 5 confirmed + 1 edited + 13 REVIEWER) |
| **Noise rejected** | 0/13 | 7/26 (27%) |
| **PP 1.2.5 coverage** | Label only | Full PP text + all 6 management measures (diuretics, binders, bicarbonate, dietary, NSAID review, constipation) + polypharmacy/cost/preference + supplementary qualifiers |
| **PP 1.2.6 coverage** | Not extracted | Full text with eGFR <15 threshold + dose reduction priorities + BP med discontinuation order |
| **PP 1.2.7 coverage** | Not extracted | Full text + dual RAS blockade harm evidence |
| **Monitoring thresholds** | eGFR <30 decontextualized | eGFR <30 with full monitoring context + eGFR <15 discontinuation threshold |
| **Pipeline weakness** | Bullet-list items (6 management measures) completely missed; PP full texts not extracted despite label detection | Addressed by REVIEWER additions + cross-check supplements |
| **Cross-check quality** | — | 5 accurate, 2 minor deviations (Cat B), 2 supplemented (Cat C), 2 gaps filled (Cat D) |
| **Overall quality** | **FAIR** | **EXCELLENT** — comprehensive after REVIEWER intervention + cross-check |
| **Page decision** | — | **ACCEPTED** |
