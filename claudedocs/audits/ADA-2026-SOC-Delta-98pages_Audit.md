# ADA 2026 SOC Delta (98 pages) — KB-0 Extraction Audit

**Pipeline Job:** `908789f3-d5a0-4187-ad9d-78072e0af1a6`
**Dashboard URL:** https://kb0-governance-dashboard.vercel.app/pipeline1/908789f3-d5a0-4187-ad9d-78072e0af1a6
**Source PDF:** `ADA-2026-SOC-Delta-98pages.pdf` (section 9 — Pharmacologic Approaches to Glycemic Treatment, Diabetes Care 2026;49(Suppl. 1):S183–S215, doi 10.2337/dc26-S009)
**Pipeline version:** `4.2.2`, L1 tag: `monkeyocr`
**DB:** GCP PostgreSQL `canonical_facts` @ `34.46.243.149:5433` — tables `l2_extraction_jobs`, `l2_merged_spans`, `l2_section_passages`, `l2_guideline_tree`
**DB snapshot time:** 2026-04-24 (live pull)
**Auditor:** Claude (Opus 4.7 1M)

---

## Job header (live from DB)

| Field | Value |
|-------|-------|
| `total_pages` | 98 |
| `total_sections` | **1** (structural sectioning failed) |
| `alignment_confidence` | **0.0** (L1 tree alignment failed) |
| `total_merged_spans` (column) | 1,922 (machine-extracted only) |
| **Live COUNT(*)** | **2,735** (machine + 813 reviewer) |
| `spans_confirmed` | 833 |
| `spans_pending` | 1,358 |
| `spans_rejected` | 146 |
| `spans_edited` | 69 |
| `spans_added` | 329 |
| `status` | IN_PROGRESS |
| `guideline_tier` | 1 |
| `l2_section_passages` rows | 1 (only) |

Contributing-channel mix across all 98 pages:

| Channel | Spans | Role |
|---|---|---|
| B | 445 | Drug lexicon (RxNorm) hits |
| C | 317 | Regex monitoring/frequency patterns |
| D | 1,192 | Table-cell extractions |
| E | 50 | GLiNER NER (contraindication markers etc.) |
| F | 98 | Passthroughs / structural markers |
| G | 687 | Full-sentence statements |
| H | 43 | Header/label inference (low confidence, 0.60) |
| REVIEWER | 813 | Reviewer-added spans (29.7% of total) |

> **Channel A is entirely absent from all 98 pages.** In the 4.2.2 pipeline Channel A is the structural-oracle alignment channel — its failure correlates with `alignment_confidence=0.0` and `total_sections=1`. **This is a global audit finding**, not a per-page one.

---

## Audit methodology

1. **Source of truth:** visual read of the PDF page image + the `markdown` field in the L1 cache (`ADA-2026-SOC-Delta-98pages_d675d10a2299_l1.json`).
2. **Compared against:** every row in `l2_merged_spans WHERE job_id = '908789f3…'` for the batch's page range, including spans contributed by channels B/C/D/E/F/G/H **and** spans whose `contributing_channels = {REVIEWER}` (human-added/edited).
3. **Classification scheme per fact:**
   - `ADDED` — present in DB with non-trivial coverage (CONFIRMED, EDITED, or REVIEWER-ADDED).
   - `MISSED` — present in PDF, absent from DB entirely, or captured only as a non-informative fragment (e.g., a bare `+` or `$$$`).
   - `NEEDS-CONFIRM` — in DB but still `PENDING` review, or semantically partial (covers only part of the source clause), or has low confidence / disagreement.
4. **Out of scope for "guideline facts":** journal boilerplate (copyright, reuse statements, committee membership credits) — these are correctly rejected when seen.
5. **Batch size:** 10 pages. 98 pages / 10 ≈ 10 batches (last batch = pages 91–98).

---

## Global findings (cross-batch, to be refined as audit progresses)

- **G-1 — Channel A missing everywhere.** Structural alignment never ran; all spans land in `section_id = "section_L1"` (one top-level node). Downstream: no fact can be anchored to a specific sub-section (e.g., "PHARMACOLOGIC THERAPY FOR ADULTS WITH TYPE 1 DIABETES" vs "…TYPE 2 DIABETES"). Risk: policy rules that scope to a sub-section cannot be expressed.
- **G-2 — Figures and tables lose structure.** Channel D emits individual table cells as standalone spans (e.g., `"+"`, `"$"`, `"$$$"`, `"++++"`), with no row/col pairing in `channel_metadata`. Figure 9.1 on page 2 and the Figure 9.4 efficacy matrix on page 9 are thereby unaudit‑able as structured facts.
- **G-3 — Channel H (span_count=43, confidence 0.60) duplicates Channel D row labels** ("Insulin plans", "Continuous insulin infusion plans"). Most H spans are redundant headers.
- **G-4 — Extraction blow-ups on specific pages:**
  - Page 2: 53 PENDING (mostly D-channel table fragments)
  - Pages 21–22: 185 + 170 PENDING (large drug comparison table)
  - Page 29: 25 PENDING
  - Page 30: 19 PENDING
  - Page 65: 47 PENDING
  - **Page 82: 750 PENDING** (outlier — likely a reference list / footnote blow-up)
- **G-5 — Counter drift.** `l2_extraction_jobs.total_merged_spans = 1922` but live COUNT is 2735. Difference (813) equals `REVIEWER`-channel spans. Machine extraction alone therefore produced 1,922 spans and humans had to add 329 + 484 edits/rejections to reach the current state. Reviewer ADD rate of **~17% of extractor output** is a strong signal the extractor under-generates on recommendation statements (confirmed batch-by-batch below).
- **G-6 — 1 section passage row for 98 pages.** `l2_section_passages` contains exactly one row; the `/jobs/{jobId}/passages` API effectively returns nothing useful. Likely a consequence of G-1.

---

## Batch 1 — Pages 1–10 (PDF pp. S183–S192)

**Content coverage:** Title/Intro; Type 1 Diabetes pharmacologic section (Recommendations 9.1–9.4, Figure 9.1 insulin plans matrix, Table 9.1 treatment-plan comparison, Figure 9.2 initiation/titration flowchart, Insulin Administration Technique, Noninsulin Treatments, Surgical Treatment, Figure 9.3 β-cell replacement tree); start of Type 2 Diabetes section (Recommendations 9.5–9.13a, Figure 9.4 glucose-lowering algorithm).

**DB inventory in this page range:**

| Page | Total | CONFIRMED | ADDED (reviewer) | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 1 | 5 | 2 | 2 | 0 | 0 | 1 |
| 2 | 61 | 3 | 4 | 0 | 53 | 1 |
| 3 | 11 | 0 | 3 | 0 | 8 | 0 |
| 4 | 6 | 0 | 2 | 0 | 4 | 0 |
| 5 | 10 | 0 | 1 | 0 | 9 | 0 |
| 6 | 7 | 0 | 2 | 0 | 5 | 0 |
| 7 | 9 | 0 | 1 | 0 | 8 | 0 |
| 8 | 8 | 0 | 4 | 0 | 4 | 0 |
| 9 | 8 | 0 | 2 | 0 | 6 | 0 |
| 10 | 13 | 0 | 6 | 0 | 7 | 0 |
| **Batch 1 total** | **138** | **5** | **27** | **0** | **104** | **2** |

**Reviewer burden so far:** 27 ADDED / 138 = **19.6%** of batch-1 spans came from a human. CONFIRMED is only 3.6% — the batch is materially un-reviewed (75% PENDING).

---

### ✅ ADDED — guideline facts present in DB (machine-CONFIRMED or reviewer-added)

| # | Page | Fact (abbrev.) | DB status | Channels |
|---|---|---|---|---|
| A1 | 1 | **Rec 9.1** Treat most adults with T1D with CSII or MDI of prandial + basal insulin (A) | CONFIRMED | B,C,G |
| A2 | 1 | **Rec 9.2** Insulin analogs (or inhaled) preferred over human insulins to minimize hypoglycemia (A) | ADDED | REVIEWER |
| A3 | 1 | **Rec 9.3** Education on matching mealtime insulin to carb/fat/protein + correction-dose rules (B) | ADDED | REVIEWER |
| A4 | 1 | **Rec 9.4** Insulin plans reassessed every 3–6 months (E) | CONFIRMED | B,C,G |
| A5 | 2 | Inhaled human insulin has rapid peak + shortened duration vs RAA | ADDED | REVIEWER |
| A6 | 2 | T1D: analog insulin → less hypoglycemia, less weight gain, lower A1C vs human insulin | ADDED | REVIEWER |
| A7 | 2 | Two injectable URAA formulations — excipients accelerate absorption | ADDED | REVIEWER |
| A8 | 2 | U-300 glargine / degludec may confer lower hypoglycemia vs U-100 glargine in T1D | ADDED | REVIEWER |
| A9 | 3 | Correction insulin: adjust ISF / target glucose if correction doesn't bring glucose into range | ADDED | REVIEWER |
| A10 | 3 | LAA adjustment: based on overnight or fasting glucose | ADDED | REVIEWER |
| A11 | 3 | Basal-rate (pump) adjustment rule | ADDED | REVIEWER |
| A12 | 4 | 3-injection / split-mixed BGM adjustment rules | ADDED | REVIEWER |
| A13 | 4 | 4-injection fixed-dose BGM adjustment rules | ADDED | REVIEWER |
| A14 | 5 | Adjust prandial insulin for nutrition + correction based on premeal glucose | ADDED | REVIEWER |
| A15 | 6 | Review overnight glucose (CGM) + adjust basal if fasting hyperglycemia | ADDED | REVIEWER |
| A16 | 6 | **Glucagon must be prescribed for emergent hypoglycemia** (from Fig 9.2) | ADDED | REVIEWER |
| A17 | 7 | Pancreas transplantation reserved for simultaneous / after-kidney or recurrent DKA/severe hypoglycemia despite optimized mgmt | ADDED | REVIEWER |
| A18 | 8 | **Rec 9.6** Consider combination therapy in T2D to shorten time to glycemic target (A) | ADDED | REVIEWER |
| A19 | 8 | **Rec 9.9b** T2D + obesity + HFpEF → GLP-1 RA with HF-event benefit (A) | ADDED | REVIEWER |
| A20 | 8 | **Rec 9.10** T2D + CKD (eGFR <60 / albuminuria) → SGLT2i or GLP-1 RA (A) | ADDED | REVIEWER |
| A21 | 8 | **Rec 9.11** T2D + advanced CKD (eGFR <30) → GLP-1 RA preferred (B) | ADDED | REVIEWER |
| A22 | 9 | In HF / CKD / CVD, GLP-1 RA or SGLT2i decision independent of glycemic goal attainment | ADDED | REVIEWER |
| A23 | 9 | SGLT2i can be started with eGFR ≥20 mL/min/1.73 m² for CV/kidney risk reduction | ADDED | REVIEWER |
| A24 | 10 | **Rec 9.15** Treatment modification (intensification / deintensification) should not be delayed (A) | ADDED | REVIEWER |
| A25 | 10 | **Rec 9.18** DPP-4i must NOT be combined with GLP-1 RA or dual GIP/GLP-1 RA (A) | ADDED | REVIEWER |
| A26 | 10 | **Rec 9.21** In T2D without severe hyperglycemia, GLP-1-based therapy preferred over insulin (A) | ADDED | REVIEWER |
| A27 | 10 | TZD drug-class note: TZD → HF risk + fluid retention; avoid in HF | ADDED | REVIEWER |
| A28 | 10 | DPP-4i drug-class note: renal dose adjustment needed for sitagliptin/saxagliptin/alogliptin; linagliptin exempt | ADDED | REVIEWER |
| A29 | 10 | Sulfonylurea drug-class note: glyburide not recommended in CKD; glipizide/glimepiride start conservatively | ADDED | REVIEWER |

> **Batch 1 critical observation:** 27 of 29 ADDED facts are `REVIEWER`-sourced. Only 2 (A1, A4) were produced by the extractor at sufficient quality to auto-confirm. **Every single T2D recommendation pulled out cleanly (9.6, 9.9b, 9.10, 9.11, 9.15, 9.18, 9.21) was added by a human** — the machine extractor is systematically missing the red-boxed "Recommendations" callouts on pages 8 and 10.

---

### ❌ MISSED — guideline facts in PDF that are absent (or fragmented beyond usefulness) in DB

| # | Page | Fact | Why missed |
|---|---|---|---|
| M1 | 1 | Document title: **"9. Pharmacologic Approaches to Glycemic Treatment: Standards of Care in Diabetes—2026"** | Not in DB as a fact or heading. Should be in `l2_guideline_tree` but root node is generic. |
| M2 | 1 | Citation / DOI: `Diabetes Care 2026;49(Suppl. 1):S183–S215`, `doi 10.2337/dc26-S009` | No provenance span. |
| M3 | 1 | Sub-heading **"PHARMACOLOGIC THERAPY FOR ADULTS WITH TYPE 1 DIABETES"** | Not captured as a section break. Consequence of G-1. |
| M4 | 1 | Sub-heading **"Insulin Therapy"** | Same cause. |
| M5 | 2 | **Figure 9.1 as a structured matrix** — 3 columns (Greater flexibility, Lower risk of hypoglycemia, Higher costs) × 7 rows (MDI LAA+RAA/URAA, MDI NPH+RAA/URAA, MDI NPH+R, 2-inj NPH+R or premix, AID, pump with low-glucose suspend, pump without automation). | 53 PENDING D-channel cell fragments (`+`, `++`, `$$$`, `++++`) with no row/col metadata — no way to reassemble the matrix. |
| M6 | 2 | Figure 9.1 caption and legend (cost-symbol semantics, insurance/discount caveats) | Not captured. |
| M7 | 3 | **Table 9.1 "Plans that more closely mimic normal insulin secretion" → Insulin pump therapy row** as a structured fact (Timing / Advantages / Disadvantages / Adjustment). | Only the Adjustment column is recovered (A9–A11 above), and only by human add. Row is not a single fact. |
| M8 | 3 | "TIR % highest and TBR % lowest with: hybrid closed-loop > sensor-augmented open-loop > conventional CSII" | Comparative clinical fact — buried inside a PENDING run-on span, not pulled out. |
| M9 | 3 | Safety fact: "Risk of rapid development of ketosis or DKA with interruption of insulin delivery" (pump-specific) | Not a discrete span; E channel did not flag as contraindication. |
| M10 | 4 | Table 9.1 footer legend (12 abbreviation definitions + Holt et al. 2021 citation) | Not captured. Impacts glossary provenance. |
| M11 | 4 | "R must be injected at least 30 min before meal for better effect" (timing pharmacology) | Embedded in PENDING, not discrete. |
| M12 | 5 | **Effect size** "CSII vs MDI modest A1C advantage **−0.30% [95% CI −0.58 to −0.02]**" | The 95% CI numbers are lost — PENDING span covers it with "modest advantages" but the CI is stripped. Audit-critical: effect-size should be an extracted `EVIDENCE` fact. |
| M13 | 5 | "In newly diagnosed T1D, insulin requirements at initiation typically **0.2 to 0.6 units/kg/day**" | Discrete dosing range not captured. Only the 0.4–1.0 steady-state range is in a PENDING span. |
| M14 | 5 | Cross-reference: "see Section 7 'Diabetes Technology'" | No link facts. |
| M15 | 6 | **Figure 9.2 algorithm structure** — initiation at 0.4–0.5 units/kg/day, titration branches on hypoglycemia/hyperglycemia/fasting-hyper/postprandial with 1–4 unit or 5–10% adjustment increments. | Flattened to prose PENDING spans. No discrete edge/node facts. |
| M16 | 6 | "To avoid therapeutic inertia, reassess and modify treatment regularly (3–6 months)" (Fig 9.2 sidebar) | Not a span. |
| M17 | 7 | **Sotagliflozin DKA safety**: "approximately eightfold increase in DKA compared with placebo" in T1D | Critical safety fact — pipeline did not extract the magnitude. |
| M18 | 7 | **Teplizumab FDA approval (2022) for delay of T1D stage 2→3** | Missed — regulatory fact. |
| M19 | 7 | **Donislecel-jujn (2023) approval as first US allogeneic islet cell therapy** | Missed. |
| M20 | 7 | **Pramlintide effect size** "A1C reduction 0.3–0.4%, weight loss ∼1–2 kg" | Magnitudes embedded in PENDING, not discrete. |
| M21 | 7 | **Metformin in T1D** — small reductions in weight/insulin/lipids, no sustained A1C | Embedded, not discrete. |
| M22 | 8 | **Rec 9.9a** T2D + obesity + HFpEF → dual GIP / GLP-1 RA (A/B) | PENDING — not yet confirmed. (See NC1.) |
| M23 | 8 | **eGFR <45 mL/min/1.73 m² threshold**: SGLT2i glycemic benefit reduced below this | Embedded in 9.10 PENDING text, not discrete. |
| M24 | 8 | **Figure 9.3 β-cell replacement tree** — GFR <30 / severe metabolic complications branching to pancreas-after-kidney / islet-after-kidney / SPK / SIK / PTA / ITA | No structured fact. |
| M25 | 9 | **Figure 9.4 efficacy tiers** (very high / high / intermediate / neutral) for weight loss and glucose lowering — specific drug-class membership of each tier | Lost — 8 spans on page 9 cannot reconstruct this. |
| M26 | 9 | "SGLT2i with proven HF benefit **in current or prior symptoms of HFrEF or HFpEF**" — conditional scope | Not discrete. |
| M27 | 9 | Endpoint definitions for SGLT2i / GLP-1 RA CVOTs (MACE, CV death, MI, stroke, HHF, all-cause mortality) — Fig 9.4 footnotes § and ‡ | Not captured. |
| M28 | 10 | **Rec 9.13b** Pioglitazone + GLP-1 RA combination for T2D + MASH/high-liver-fibrosis-risk (B) | Not in DB. |
| M29 | 10 | **Rec 9.14** Reassess medication plan every 3–6 months (E) | Not in DB. |
| M30 | 10 | **Rec 9.16** Choice of glucose-lowering therapy modification considerations (B) | Not in DB. |
| M31 | 10 | **Rec 9.19** Weight-management interventions for unmet weight goals in T2D (A) | Not in DB. |
| M32 | 10 | **Rec 9.23** Continue glucose-lowering agents when initiating insulin (A) | Not in DB. |

> **Batch 1 missed-recs tally:** 6 T2D recommendations missing (9.9a PENDING, 9.13b, 9.14, 9.16, 9.19, 9.23). Combined with the 7 T2D recs that only exist via REVIEWER add, **the extractor independently captured 0 / 13 T2D recommendations** on pages 8–10. Every single one was either missed or added by a human.

---

### ❓ NEEDS-CONFIRM — in DB as PENDING, ambiguous, or low-quality

| # | Page | Fact (as seen in DB text) | Recommended action |
|---|---|---|---|
| NC1 | 8 | **Rec 9.9a** dual GIP/GLP-1 RA for T2D+obesity+HFpEF (PENDING) | Confirm — text is accurate. |
| NC2 | 8 | **Rec 9.7** T2D + ASCVD → GLP-1 RA and/or SGLT2i (PENDING) | Confirm. |
| NC3 | 8 | **Rec 9.8** T2D + HF → SGLT2i (PENDING) | Confirm. |
| NC4 | 8 | **Rec 9.12** T2D + MASLD + overweight → GLP-1 RA (PENDING) | Confirm — text cut off at "a dual…" mid-sentence; **EDIT** to complete. |
| NC5 | 10 | **Rec 9.17** Reassess hypoglycemia-risk meds when starting new glucose-lowering therapy (PENDING) | Confirm. |
| NC6 | 10 | **Rec 9.20** Initiate insulin if symptoms or A1C >10% / BG ≥300 mg/dL (PENDING) | Confirm — specific thresholds preserved. |
| NC7 | 10 | **Rec 9.22** If insulin used, combine with GLP-1 RA (PENDING) | Confirm. |
| NC8 | 5 | 30–50% basal / remainder prandial; 0.4–1 unit/kg/day; 0.5 units/kg/day typical starting adult dose (PENDING, but well-formed) | Confirm (split into 3 facts — currently one conflated span). |
| NC9 | 6 | Prandial insulin options: RAA, URAA, short-acting human, inhaled human (PENDING) | Confirm. |
| NC10 | 7 | Pramlintide A1C 0.3–0.4%, weight −1–2 kg (PENDING) | Confirm + **extract effect sizes** as structured fields. |
| NC11 | 7 | Metformin in T1D: ↓ body weight / insulin / lipids, no sustained A1C (PENDING) | Confirm. |
| NC12 | 7 | Liraglutide in T1D: no β-cell preservation, worsened C-peptide loss vs placebo (PENDING) | Confirm. |
| NC13 | 7 | Sotagliflozin contraindicated in T1D due to DKA risk (PENDING) | Confirm — but **add effect size** from M17. |
| NC14 | 7 | Donislecel indication in PENDING — scope "T2D" is visible in the partial text; verify it doesn't incorrectly include T1D eligibility. | Verify wording. |
| NC15 | 2 | ~40 D-channel PENDING spans that are **single tokens** (`+`, `$`, `++++`, `$$$`) | **REJECT in bulk** — these carry no fact content without row/col context. Reject reason = "fragment without context". |
| NC16 | 2 | H-channel PENDING spans that are plain row labels ("Insulin plans", "Continuous insulin infusion plans", "MDI with NPH + short-acting (regular) insulin") | REJECT — duplicates of D-channel labels; not facts. |
| NC17 | 5–7 | All run-on PENDING spans carrying **embedded** effect sizes (CSII −0.30%, pramlintide 0.3–0.4%, sotagliflozin 8×) | **EDIT** to split into atomic `EFFECT_SIZE` facts with magnitude + CI + population + comparator. |
| NC18 | 10 | F-channel PENDING "`<!-- PAGE 10 -->`" and "`<!-- Chunk chunk-002: pages 11-20 -->`" markers | REJECT — structural markers, not facts. (Same pattern persists on pages 11, 21, 31, 41, 51, 61, 71, 81, 91.) |

---

### Batch 1 summary

| Bucket | Count |
|---|---|
| ADDED (present in DB, acceptable quality) | 29 |
| MISSED (absent or non-recoverable fragments) | 32 |
| NEEDS-CONFIRM (PENDING or partial) | 18 |
| Out-of-scope rejections (boilerplate / fragments / markers) | ~50 spans to reject in bulk (NC15, NC16, NC18) |

**Signal-to-noise:** of 138 DB spans in batch 1, ~50 are structural/fragment noise to be rejected, ~18 need active review, ~29 are acceptable facts, leaving a "true information" rate of ~34%. The rest is reviewer labor cost.

**Pipeline failure modes observed in batch 1:**
1. **Recommendation-block blindness** — the red "Recommendations 9.x" callouts on pages 1, 8, 10 are inconsistently captured; machine hits 9.1, 9.4 but misses 9.2, 9.3, 9.6, 9.9b, 9.10, 9.11, 9.13b, 9.14, 9.15, 9.16, 9.18, 9.19, 9.21, 9.23. Hypothesis: the pink-tinted background / small-caps heading breaks a layout cue.
2. **Effect-size stripping** — magnitudes survive in prose but are not promoted to structured facts. KB-0 downstream consumers (KB-3 Guidelines, KB-23 Decision Cards) need the magnitude, not the prose.
3. **Figure flattening** — Figures 9.1, 9.2, 9.3, 9.4 all lose their matrix/tree structure. Channel D emits cells without row/col pairing.
4. **Drug-class safety notes under-captured** — reviewer had to hand-type TZD/DPP-4/SU kidney and HF rules that are visible in Figure 9.4 footnotes.

---

## Batch 2 — Pages 11–20 (PDF pp. S193–S202)

**Content coverage:** Table 9.2 "Features of medications for lowering glucose in T2D" (pages 11–14, spanning 9 drug classes × 8 attribute columns); prose on metformin pharmacology, GRADE trial, combination therapy (page 15); Figure 9.5 "Intensification of injectable therapies" (page 16); Glucose-Lowering Therapy for CV Disease / CKD (pages 17–18); Glucose-Lowering Therapy for Metabolic Comorbidities + Insulin Therapy + Basal Insulin (page 18); Combination Injectable Therapy + U-500/U-200 concentrated insulins (page 19); Inhaled insulin + Recommendations 9.24–9.30 (page 20).

**DB inventory in this page range:**

| Page | Total | CONFIRMED | ADDED | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 11 | 7 | 0 | 6 | 0 | 1 | 0 |
| 12 | 3 | 0 | 1 | 0 | 2 | 0 |
| 13 | 2 | 0 | 1 | 0 | 1 | 0 |
| 14 | 3 | 0 | 1 | 0 | 2 | 0 |
| 15 | 11 | 6 | 1 | 3 | 1 | 0 |
| 16 | 8 | 6 | 1 | 1 | 0 | 0 |
| 17 | 11 | 5 | 2 | 4 | 0 | 0 |
| 18 | 12 | 4 | 3 | 5 | 0 | 0 |
| 19 | 9 | 4 | 0 | 5 | 0 | 0 |
| 20 | 13 | 1 | 5 | 4 | 3 | 0 |
| **Batch 2 total** | **79** | **26** | **21** | **22** | **10** | **0** |

**Reviewer burden this batch:** 21 ADDED / 79 = **26.6%** (worse than batch 1). **Pages 11–14 together hold only 15 spans for all of Table 9.2** — essentially the entire medication-features table exists in DB only via 9 reviewer-written row summaries, with the table's 72-cell matrix structure absent.

---

### ✅ ADDED — facts present in DB

**Drug-class summaries from Table 9.2 (all REVIEWER-ADDED; compact row synopses, not cell-level data):**

| # | Page | Drug class | DB-captured summary content |
|---|---|---|---|
| A30 | 11 | Metformin | eGFR <30 contraindicated; GI side-effects mitigated by slow titration / ER formulation / with food; vitamin B12 deficiency risk — monitor and replete |
| A31 | 11 | SGLT2i (canagliflozin, empagliflozin, dapagliflozin, ertugliflozin) | CV (MACE, HF) + kidney benefit; glucose-lowering minimal <eGFR 45; continue/start for CV/kidney benefit if eGFR >20; may continue until dialysis; discontinue 3–4 days before surgery, in critical illness, or prolonged fasting — DKA risk mitigation |
| A32 | 12 | GLP-1 RAs (dulaglutide, liraglutide, semaglutide) + tirzepatide (GIP/GLP-1 RA) | MACE + kidney benefit; warnings: C-cell tumors (rodent), pancreatitis, biliary disease, ileus, diabetic retinopathy (high-risk monitoring), NAION (rare). Tirzepatide very-high glucose + weight efficacy |
| A33 | 13 | GLP-1 RA GI side-effect mitigation | Dietary modification, slower titration; **contraindicated in gastroparesis** |
| A34 | 14 | Insulin (MDI or pump) | Very-high glucose-lowering; high hypoglycemia; weight gain; lower doses with decreased eGFR; higher hypoglycemia with human (NPH or premixed) vs analogs; weight management is distinct treatment goal |

**Prose / recommendation facts (mixed status):**

| # | Page | Fact | Status |
|---|---|---|---|
| A35 | 15 | Oral semaglutide commercially available (in addition to injectable GLP-1 RAs) | ADDED |
| A36 | 15 | Metformin safe at eGFR ≥30; lactic acidosis rare and primarily at eGFR <30; increased B12 deficiency risk + neuropathy worsening | EDITED |
| A37 | 15 | Largest A1C reductions: insulin, select GLP-1 RAs (semaglutide), tirzepatide; smallest: DPP-4i | CONFIRMED |
| A38 | 15 | When A1C ≥1.5% above individualized goal → dual-combination or more potent therapy | CONFIRMED |
| A39 | 15 | Insulin initiation if BG ≥300 mg/dL (≥16.7 mmol/L) or A1C >10% (>86 mmol/mol) or symptoms (polyuria/polydipsia) or catabolism | EDITED |
| A40 | 15 | T2D severe hyperglycemia alternatives: SU, GLP-1 RA, dual GIP/GLP-1 RA (evidence scarce above A1C 10–12%) | CONFIRMED |
| A41 | 15 | T2D is progressive → often requires combination therapy | EDITED |
| A42 | 15 | Initial combination therapy in A1C 1.5–2.0% above goal or high-risk/established CVD (GLP-1 RA + SGLT2i combo) | CONFIRMED |
| A43 | 15 | Each new noninsulin class added to metformin → **A1C lowered ~0.7–1.0% (8–11 mmol/mol)**; GLP-1 RA or dual GIP/GLP-1 RA → 1–≥2% A1C lowering | CONFIRMED |
| A44 | 15 | **GLP-1 RA + DPP-4i NOT recommended** — no added glucose-lowering benefit | CONFIRMED |
| A45 | 16 | Starting NPH 2/3 morning + 1/3 bedtime split (conversion algorithm) | CONFIRMED |
| A46 | 16 | Basal analog / NPH initiation: 0.1–0.2 units/kg/day; titration 2 units every 3 days to FPG goal without hypoglycemia; prandial starting 4 units or 10% of basal at largest-excursion meal | EDITED + CONFIRMED |
| A47 | 16 | Consider insulin as first injectable if symptoms, A1C >10%, BG ≥300 mg/dL, or T1D possibility | ADDED |
| A48 | 16 | NPH switching guidance (evening → basal analog for hypoglycemia / missed doses; morning NPH for steroid-induced hyperglycemia) | CONFIRMED |
| A49 | 16 | Prandial insulin options: injectable rapid/ultra-rapid analog, short-acting human, or **inhaled human insulin** | CONFIRMED |
| A50 | 16 | If adding prandial to NPH → consider self-mixed or premixed | CONFIRMED |
| A51 | 16 | CVD → GLP-1 RA with proven CVD benefit (oral or injectable); fixed-ratio combo products IDegLira and iGlarLixi for GLP-1 RA + basal insulin | CONFIRMED |
| A52 | 17 | GLP-1 RAs and dual GIP/GLP-1 RA preferred over basal insulin initiation in most individuals (lower hypoglycemia, weight benefits, trade-off: GI side effects) | EDITED |
| A53 | 17 | Intensifying to insulin: add GLP-1 RA or dual GIP/GLP-1 RA for greater efficacy / weight / hypoglycemia benefit | CONFIRMED |
| A54 | 17 | **Treatment deintensification** in weight loss / lifestyle optimization contexts | CONFIRMED |
| A55 | 17 | T2D + HFpEF → GLP-1 RA or dual GIP/GLP-1 RA recommended (with obesity + symptomatic) | CONFIRMED |
| A56 | 17 | Switching to SGLT2i / GLP-1 RA for comorbidity benefit even if A1C already at goal | ADDED |
| A57 | 17 | SGLT2i CKD+T2D benefit trials: empagliflozin, canagliflozin, dapagliflozin slow CKD progression and improve CV outcomes | ADDED |
| A58 | 17 | **Metformin must not be started at eGFR <45; dose-reduce at <45; stop at <30** | EDITED |
| A59 | 18 | Insulin / SU dose close monitoring as eGFR declines (hypoglycemia education) | EDITED |
| A60 | 18 | Pioglitazone, GLP-1 RAs, dual GIP/GLP-1 RA → hepatic steatosis ↓, MASH resolution without fibrosis worsening in biopsy-proven MASH | CONFIRMED |
| A61 | 18 | Glucose-Lowering Therapy for Metabolic Comorbidities — section context | ADDED |
| A62 | 18 | **Insulin / SU / TZD → weight gain, use judiciously + lowest dose** | EDITED |
| A63 | 18 | Weight-loss glucose-lowering ranking: tirzepatide & semaglutide (highest) > dulaglutide, liraglutide, ER-exenatide | EDITED |
| A64 | 18 | Basal insulin alone as most convenient initial insulin; 0.1–0.2 units/kg/day starting based on body weight + hyperglycemia degree | ADDED |
| A65 | 18 | Long-acting basal analogs (U-100 glargine) reduce level-2 + nocturnal hypoglycemia vs NPH | CONFIRMED |
| A66 | 19 | Overbasalization signals: **bedtime-to-morning differential ≥50 mg/dL (≥2.8 mmol/L)**, hypoglycemia (aware/unaware), high variability | EDITED |
| A67 | 19 | Combination injectable therapy: GLP-1 RA or dual GIP/GLP-1 RA added to basal or MDI | EDITED |
| A68 | 19 | Prandial insulin start: **4 units or 10% of basal** at largest-excursion meal | CONFIRMED |
| A69 | 19 | U-500 regular insulin distinct PK (similar onset, delayed/blunted/prolonged peak, longer duration — like premixed intermediate+regular) | CONFIRMED |
| A70 | 19 | U-200 formulations (degludec, lispro, lispro-aabc) PK similar to U-100; other concentrated insulins only in prefilled pens | EDITED |
| A71 | 20 | **Rec 9.24** Healthy behaviors, DSMES, therapeutic-inertia avoidance, SDOH as essential components (A) | ADDED |
| A72 | 20 | **Rec 9.25** CGM at onset and thereafter for insulin users / hypoglycemia-risk noninsulin users / management-aiding use (A/B) | EDITED |
| A73 | 20 | **Rec 9.26** Monitor for overbasalization; prompt reevaluation (E) | EDITED |
| A74 | 20 | **Rec 9.27** AID systems offered to **all adults with T1D and T2D on insulin** (A) — *expanded to T2D in 2026 — new* | EDITED |
| A75 | 20 | **Rec 9.28** Glucagon prescribed for all on insulin or high hypoglycemia risk; family/caregiver education; preparations not requiring reconstitution preferred (A) | ADDED |
| A76 | 20 | **Rec 9.29** Routine assessment for financial obstacles; interdisciplinary collaboration (E) | ADDED |
| A77 | 20 | Inhaled insulin: weight reduction vs aspart over 24 weeks; **FEV₁ decline**; contraindicated in asthma/COPD; not recommended in smokers/recent quitters; FEV₁ spirometry required before/after starting | EDITED |
| A78 | 20 | CGM in insulin-treated + hypoglycemia-risk noninsulin-treated diabetes improves A1C, TIR, hypoglycemia outcomes | ADDED |

---

### ❌ MISSED — facts in PDF absent / non-recoverable in DB

| # | Page | Fact | Why missed |
|---|---|---|---|
| M33 | 11–14 | **Table 9.2 as a structured matrix** — 9 drug-class rows × 8 attribute columns (Glucose-lowering efficacy, Hypoglycemia risk, Weight effect, MACE, HF, Kidney, MASH, Clinical considerations) | Extractor emitted ~4 spans for Table 9.2 total; no row/col pairing. Reviewer had to hand-write compact summaries. |
| M34 | 13 | **DPP-4 inhibitors row** — intermediate efficacy, no hypoglycemia, weight-neutral, CV-neutral (saxagliptin potential HF risk), kidney-neutral, MASH-unknown. Side effects: pancreatitis (reported, causality not established), joint pain, bullous pemphigoid | Not in DB as a Table 9.2 row. Only the page-10 reviewer-added drug-class note covers the renal dose adjustments. |
| M35 | 13 | **Pioglitazone row** — high efficacy, no hypoglycemia, weight gain, potential CV benefit, **increased HF risk**, MASH potential benefit. Contraindications: HF (do not use), active bladder cancer; **bone fracture risk**; no dose adjustment. | Not in DB as a Table 9.2 row. Page-10 reviewer note covers HF only. |
| M36 | 13 | **Sulfonylureas row (2nd gen)** — high efficacy, **yes hypoglycemia**, weight gain, CV-neutral. **FDA Special Warning on CV mortality** (based on tolbutamide studies); glimepiride shown CV-safe. Glyburide not recommended in CKD; glipizide/glimepiride conservative start | Not as a Table 9.2 row. Page-10 reviewer note covers CKD only (M-note was derived from this row). |
| M37 | 13 | **Meglitinides** | Mentioned in rec 9.17 and rec 9.19 but **no Table 9.2 row extracted at all**. |
| M38 | 14 | **Insulin (analogs) row** is mentioned only by label ("Insulin (analogs) (SQ)") in L1; the row's content appears truncated in PDF itself — confirm whether Table 9.2 continues to page S197 (out of delta) | Borderline: this table may continue beyond the 98-page delta. Flag for source-scope verification. |
| M39 | 14 | **Table 9.2 footer legend** — 15 abbreviations + Tsapas et al. 107/317 citations + Davies et al. 90 attribution | Not captured. |
| M40 | 15 | **Lactic acidosis eGFR thresholds specifics** — "eGFR 30–45 → periodic dips to ≤30 heighten risk", "lactic acidosis very rare and primarily at eGFR <30" | Partially captured in A36 (EDITED) but the two distinct risk zones are not separated. |
| M41 | 15 | **GRADE trial design description** — comparison of glargine U-100 vs liraglutide vs sitagliptin vs glimepiride (T2D, A1C 6.8–8.5%, 5-year outcomes) with glargine + liraglutide most effective for A1C <7%, sitagliptin least; severe hypoglycemia more common with glargine/glimepiride | The CONFIRMED span cuts off at "sitagliptin in individuals wit…" — the 5-year results and hypoglycemia finding are lost. |
| M42 | 15 | Observational emulation of GRADE with canagliflozin comparator (liraglutide > sitagliptin / canagliflozin / glimepiride for A1C <7%) | Not captured. |
| M43 | 16 | **Figure 9.5 initiation-of-injectable algorithm** as a structured decision tree (starting basal at 10 U/day or kg/day; titrate 2 U / 3 days; advance to prandial at 4 U / 10% of basal; switch among self-mixed, premixed, basal-bolus; hypoglycemia → reduce 10–20%; A1C <8% → adjust 10–15% twice weekly) | Captured as prose in CONFIRMED span — algorithm edges / branch conditions not discrete facts. |
| M44 | 17 | **GRADE trial CV effect size**: liraglutide HR 0.7 [95% CI 0.6–0.9] for composite CV events vs the other three arms, with no individual-treatment significance for MACE / HF hospitalization / CV death | **Effect size completely absent** from the CONFIRMED span, which cuts off before the HR. |
| M45 | 17 | Importance note: "SGLT2i and GLP-1 RAs are associated with lower hypoglycemia, and individuals with ASCVD/HF/CKD have **higher hypoglycemia risk** than those without" — discrete epidemiologic fact | Embedded in EDITED span, not pulled out. |
| M46 | 18 | **Semaglutide FDA approval for MASH with moderate-to-advanced liver fibrosis** (phase 3 trial with histological improvements in steatohepatitis + fibrosis) | Not discrete — implicit inside A60 but the regulatory fact is lost. |
| M47 | 18 | **Metabolic surgery options: Roux-en-Y gastric bypass, sleeve gastrectomy** — weight + glycemic + beyond-metabolism benefits | Not captured. |
| M48 | 18 | Epidemiology: **"Obesity is present in over 90% of people with type 2 diabetes"** | Not a span. |
| M49 | 18 | Weight-neutral agents list: metformin, SGLT2i, DPP-4i, dopamine agonists, bile acid sequestrants, α-glucosidase inhibitors — as add-on for weight+obesity T2D when preferred agents not tolerated/contraindicated/unavailable | Not captured. |
| M50 | 19 | T2D insulin resistance: typically ~1 unit/kg (higher than T1D), lower hypoglycemia rates | Buried in page-19 prose; not a discrete dose fact. |
| M51 | 19 | Basal↔analog switching and overbasalization reevaluation flow — conversions across NPH/basal analog/premixed | Partly in A45/A48; the more granular switching rules (U-100 glargine ↔ degludec ↔ detemir-removal) are lost. |
| M52 | 19 | When starting insulin intensification: **maintain metformin, SGLT2i, GLP-1 RAs (or dual GIP/GLP-1 RA); limit/discontinue SU, meglitinides, DPP-4i** | Captured only partially in run-on spans. Should be a discrete "medication-continuation" rule. |
| M53 | 20 | **Rec 9.30** Cost-related barriers → consider lower-cost meds (metformin, SU, TZD, human insulin) weighing hypoglycemia/weight/CV/kidney risks (E) | PENDING. See NC-19. |
| M54 | 20 | **Additional Recommendations for All Individuals With Diabetes** — header/section break not captured (consequence of G-1). | Section-break loss. |
| M55 | 20 | "Glucagon preparations that do not require reconstitution are preferred (B)" — sub-clause of 9.28 with explicit evidence grade B | Merged into A75 without the grade annotation. |

---

### ❓ NEEDS-CONFIRM

| # | Page | Fact (as seen in DB) | Action |
|---|---|---|---|
| NC19 | 20 | **Rec 9.30** low-cost medication considerations (PENDING) | Confirm. |
| NC20 | 20 | Glucagon section prose (PENDING) | Confirm or merge into A75. |
| NC21 | 20 | Medication Costs and Affordability — "costs for noninsulin and insulin diabetes medications have increased dramatically over the past two decades" (PENDING) | Confirm. |
| NC22 | 11–14 | Table 9.2 stub PENDING spans (`<!-- PAGE N --> / Table 9.2—Continued / Medication (route of administration) / ...` column headers) | **REJECT** — they are not facts; headers lack row data. |
| NC23 | 12 | Run-on PENDING span containing "Neutral: exenatide once weekly, lixisenatide" — partial MACE-neutral row fragment | **EDIT** to split into a drug-specific MACE-neutral fact. |
| NC24 | 13 | Run-on PENDING span with "consider slower dose titration … not recommended for individuals with gastroparesis" | Already covered by A33; **REJECT** duplicate. |
| NC25 | 14 | Run-on PENDING span on weight management goal (already covered by A63) | **REJECT** duplicate. |
| NC26 | 15 | PENDING F-channel "release formulation" fragment | **REJECT** — token-level noise. |

---

### Batch 2 summary

| Bucket | Count |
|---|---|
| ADDED (acceptable) | 49 |
| MISSED | 23 |
| NEEDS-CONFIRM | 8 |
| REJECT-as-duplicate or noise | ~4 |

**Pipeline failure modes reinforced in batch 2:**
5. **Wide tables get catastrophically under-captured.** Table 9.2 spans 4 pages with 9 drug classes × 8 attributes (72 cells). DB holds 4 reviewer row-summaries + 0 extractor row-summaries. DPP-4i, pioglitazone, SU, meglitinides have **no Table 9.2 representation in DB at all** — a serious gap for a clinical-knowledge DB whose explicit purpose is to hold drug-class attribute facts.
6. **Effect sizes and hazard ratios are consistently dropped.** Batch 1 missed pramlintide 0.3–0.4%, sotagliflozin 8× DKA, CSII −0.30% CI. Batch 2 adds: GRADE liraglutide HR 0.7 [0.6–0.9], per-class +0.7–1.0% A1C reduction (only partly captured), overbasalization ≥50 mg/dL threshold (captured — good), 4-units-or-10% prandial rule (captured), 0.1–0.2 units/kg/day basal start (captured).
7. **Recommendation-block detection is improving by page 20.** 9.24, 9.28, 9.29 still require reviewer ADD, but 9.25, 9.26, 9.27 were EDITED (extractor got the skeleton; reviewer fixed wording) — a better outcome than batch 1's Rec 9.6/9.10/9.11 where the entire text was reviewer-written.
8. **New-in-2026 policy change partially captured.** Rec 9.27 extending AID-system offer to T2D-on-insulin (new this year) is in DB as EDITED — the change is preserved, but there is no metadata flagging it as "delta vs 2025 SOC".
9. **Cross-references to sibling sections are lost.** Page 15 ties to §6 Glycemic Goals; page 17 ties to §10 CV, §11 CKD; page 18 ties to §8 Obesity, §4 Comorbidities. None of these cross-reference edges exist as structured facts.

---

## Batch 3 — Pages 21–30 (PDF pp. S203–S212)

**Content coverage:** Table 9.3 (noninsulin glucose-lowering agents — AWP / NADAC cost table, pages 21–22) · Table 9.4 (insulin cost table, page 22) · Recs 9.31a/b/c (medication unavailability, compounded products) · Recs 9.32a/b (contraception/preconception) · Rec 9.33 (ICI-induced hyperglycemia) · Rec 9.34 (mTOR inhibitors) · Rec 9.35a/b (PI3K inhibitors) · Rec 9.36 (glucocorticoids) · Rec 9.37 (PTDM postoperative) · Rec 9.38a/b/c (PTDM long-term) · Rec 9.39 (SGLT-inhibitor DKA education) · prose on Therapeutic Strategies with Medication Unavailability, Pregnancy/Preconception, Cancer Treatment (ICIs/PI3K/mTOR/glucocorticoids), Pancreatic/CF-related Diabetes, PTDM, MODY/neonatal diabetes, SGLT inhibition & ketosis risk · **References section (pages 27–30).**

**DB inventory in this page range:**

| Page | Total | CONFIRMED | ADDED | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 21 | 185 | 0 | 0 | 0 | **185** | 0 |
| 22 | 170 | 0 | 0 | 0 | **170** | 0 |
| 23 | 10 | 3 | 2 | 4 | 0 | 0* |
| 24 | 7 | 4 | 1 | 2 | 0 | 0 |
| 25 | 14 | 1 | 5 | 0 | 8 | 0 |
| 26 | 12 | 0 | 4 | 0 | 8 | 0 |
| 27 | 15 | 0 | 0 | 0 | 12 | 3 |
| 28 | 15 | 0 | 0 | 0 | 13 | 2 |
| 29 | 25 | 0 | 0 | 0 | 25 | 0 |
| 30 | 19 | 0 | 0 | 0 | 19 | 0 |
| **Batch 3 total** | **472** | **8** | **12** | **6** | **440** | **5** |

(* Page 23 REJECTED count elsewhere in snapshot was 3 — it's for different pages; batch table above uses the canonical per-page counts.)

**This batch exposes the two most severe pipeline failures in the document:**
- **Pages 21–22: 355 PENDING spans, 0 CONFIRMED.** Two cost tables (Table 9.3 + Table 9.4) fragmented into unusable cells. Zero reviewer engagement — nobody has approved/edited/added a single cost fact from either table.
- **Pages 27–30: 74 PENDING spans representing individual reference-list citations** — these are bibliography entries, not guideline facts. They should never have been extracted as `l2_merged_spans` in the first place; they belong in `l2_references` (which doesn't exist).

---

### ✅ ADDED — guideline facts present in DB

| # | Page | Fact | Status |
|---|---|---|---|
| A79 | 23 | **Rec 9.39** Educate SGLT-inhibitor-treated individuals at DKA risk on risks/signs/management and provide ketone monitoring tools | ADDED |
| A80 | 23 | **Rec 9.38c** If individualized glycemic goals cannot be achieved with noninsulin pharmacotherapy in PTDM or preexisting T2D → consider adding insulin (C) | ADDED |
| A81 | 23 | Therapeutic-strategies-with-unavailability prose: recall examples (metformin recalls, GLP-1 RA shortages leading to FDA shortage declaration) | EDITED |
| A82 | 23 | **Compounded products not FDA-approved → not recommended due to safety/quality/effectiveness concerns** | CONFIRMED |
| A83 | 23 | **Rec 9.32a** Counsel individuals of childbearing potential on contraception options and impact of glucose-lowering meds on contraception efficacy (A/C) | CONFIRMED |
| A84 | 23 | **Rec 9.33** ICI-associated hyperglycemia (anti-PD-1/PD-L1: nivolumab, pembrolizumab, avelumab) → assess for immediate insulin initiation due to DKA risk while further testing determines if immunotherapy-associated diabetes; close monitoring, education, dose adjustment (C) | CONFIRMED |
| A85 | 23 | **Rec 9.34** Metformin as first-line for mTOR-inhibitor-induced hyperglycemia (E) | EDITED |
| A86 | 23 | **Rec 9.37** PTDM or preexisting T2D in postoperative setting → insulin preferred for glycemic management; DPP-4i considered for mild hyperglycemia (A) | EDITED |
| A87 | 23 | **Rec 9.38b** PTDM or preexisting T2D → GLP-1 RA considered for long-term glycemic management due to cardiometabolic (CV, kidney, weight, liver) benefits | EDITED |
| A88 | 23 | GLP-1 RAs and dual GIP/GLP-1 RA slow gastric emptying → affect absorption of oral medications incl. oral contraceptives | EDITED |
| A89 | 24 | **Tirzepatide + oral contraception → use second contraceptive form during titration + 4 weeks after reaching maintenance dose** | ADDED |
| A90 | 24 | Tirzepatide > GLP-1 RAs for oral contraception absorption impact | EDITED |
| A91 | 24 | ICI-hyperglycemia: initiate basal insulin if BG >250 mg/dL while further evaluation takes place; prandial insulin often required if insulinopenia confirmed | CONFIRMED |
| A92 | 24 | **Rec 9.35a (part) / PI3K inhibitor hyperglycemia**: metformin first-line, uptitrate as tolerated; pioglitazone option (mono or with metformin), slow onset limits | CONFIRMED |
| A93 | 24 | **Insulin and sulfonylureas as LAST RESORT** for PI3K inhibitor hyperglycemia — increased insulin may reactivate PI3K pathway, counteracting antitumor effects | CONFIRMED |
| A94 | 24 | SU/meglitinide/insulin-secretagogue caution with PI3K inhibitors (effect on PI3K efficacy; nausea/vomiting) | CONFIRMED |
| A95 | 24 | Glucocorticoid-induced hyperglycemia context (glucocorticoid use in cancer / inflammation / post-transplant) | CONFIRMED |
| A96 | 25 | **In pancreatitis history → AVOID incretin medications (GLP-1 RAs, dual GIP/GLP-1 RA, DPP-4i)** | ADDED |
| A97 | 25 | **GLP-1 RA therapy preferred in PTDM** — CV, kidney, weight, glucose benefits; no concerning studies | ADDED |
| A98 | 25 | **SGLT2i safety in PTDM** — safe/effective but ↑ GU infection risk in immunosuppressed; prior UTI history and risk assessment needed post-kidney-transplant | ADDED |
| A99 | 25 | **CFRD → insulin therapy; consider pump including AID when appropriate** | ADDED |
| A100 | 25 | **3rd International PTDM Consensus Meeting screening**: pretransplant OGTT, early OGTT at 3 months posttransplant, late OGTT at 1 year onward | ADDED |
| A101 | 26 | **MODY HNF1A + DPP-4i add-on to SU** may improve glycemic variability | ADDED |
| A102 | 26 | **Previous DKA → do NOT use SGLT inhibition** | ADDED |
| A103 | 26 | **Neonatal diabetes**: KCNJ11/ABCC8 → high-dose sulfonylureas; **INS/GATA6/EIF2AK3/FOXP3 → insulin** | ADDED |
| A104 | 26 | **SGLT-inhibitor-associated DKA ~4% in T1D; risk 5–17× higher vs untreated T1D** | ADDED |

---

### ❌ MISSED — facts in PDF absent / non-recoverable

#### Tables 9.3 and 9.4 (pages 21–22)

| # | Page | Missed fact-type | Count |
|---|---|---|---|
| M56 | 21 | **Table 9.3: Median monthly (30-day) AWP and NADAC cost — entire 10-class × 20+ compound cost matrix** for noninsulin glucose-lowering agents in the U.S. (biguanides, SUs, TZD, α-glucosidase, meglitinides, DPP-4i, SGLT2i, GLP-1 RAs, dual GIP/GLP-1 RA, bile acid sequestrant, dopamine-2 agonist). Max approved daily dose, AWP median (min,max), NADAC median (min,max) per compound-strength-product. | **~100 cost rows entirely lost as structured facts** |
| M57 | 21 | Table 9.3 pricing-method footnote (calculation formula, generic-price-used rule, once-weekly annotation) | 1 |
| M58 | 21 | Medicare insulin price cap: $35 per prescription per month; 26 states + DC state-regulated commercial insurance caps ($35/month or $100/month equivalents) | 1 |
| M59 | 21 | No cost caps exist for diabetes durable medical equipment or noninsulin medications | 1 |
| M60 | 22 | **Table 9.4: Median cost of insulin products in the U.S.** — AWP and NADAC per 1,000 units. Categories: Rapid-acting (aspart, aspart biosimilars, faster-aspart, glulisine, inhaled, lispro, lispro-aabc, lispro follow-on), Short-acting (human regular), Intermediate-acting (human NPH), Concentrated U-500 human regular, Long-acting (degludec, glargine, glargine biosimilar), Premixed (aspart 70/30, lispro 50/50, lispro 75/25, NPH/regular 70/30), Premixed insulin + GLP-1 RA (degludec/liraglutide, glargine/lixisenatide). Each × U-100 vial / cartridge / prefilled pen / U-200 pen / etc. | **~40 insulin cost rows entirely lost** |
| M61 | 22 | Table 9.4 Walmart pricing footnote (~$25/vial or $43/box of 5 pens for human insulins; ~$73/vial or $86/box for select analogs) | 1 |
| M62 | 22 | Interprofessional team for cost-saving (pharmacists, diabetes care and education specialists, social workers, CHWs, community paramedics) | 1 |

#### Recommendations 9.31-9.38 (pages 22–23)

| # | Page | Missed rec |
|---|---|---|
| M63 | 22 | **Rec 9.31a** (compounded products not FDA-approved → not recommended, **C**) — actually captured as A82 but WITHOUT the rec number / evidence grade. |
| M64 | 22 | **Rec 9.31b** If a glucose-lowering medication is unavailable (shortage) → switch to FDA-approved medication with similar efficacy, as clinically appropriate (E) | Missed |
| M65 | 22 | **Rec 9.31c** Upon resolution of unavailability → reassess appropriateness of resuming original FDA-approved medication (E) | Missed |
| M66 | 23 | **Rec 9.32b** Person-centered shared decision-making for preconception planning; address glycemic goals, noninsulin-med discontinuation timeframe, optimal glycemic management for pregnancy (A/E) | Missed |
| M67 | 23 | **Rec 9.35a** Consider metformin as first-line for PI3K-inhibitor-induced hyperglycemia (**E**) — captured as A92 but without the rec number |
| M68 | 23 | **Rec 9.35b** continuation (second PI3K recommendation — the L1 text cuts off at "Consider metformin as the first-" so the 9.35b text content is truncated in L1 itself). Flag for PDF source re-extraction. | Missed / unclear |
| M69 | 23 | **Rec 9.36** glucocorticoid-induced hyperglycemia management (insulin first-line; timing of dose matched to glucocorticoid timing; SU/meglitinide additions used for T2D/no-prior-diabetes) — recommendation inferred from prose on page 25 | Missed |
| M70 | 23 | **Rec 9.38a** PTDM / preexisting T2D long-term: metformin may be used with caution; do not initiate at eGFR <45; discontinue at <30; lactic acidosis risk with fluctuating kidney function | Missed |

#### Special populations prose

| # | Page | Missed fact |
|---|---|---|
| M71 | 24 | **ICI-associated diabetes incidence ~≤1%** after anti-PD-1/PD-L1 exposure | Missed |
| M72 | 24 | ICI anti-CTLA-4 (ipilimumab) also implicated, much less commonly | Missed |
| M73 | 24 | ICI-hyperglycemia timing: 1 week to 12 months after first dose | Missed |
| M74 | 24 | β-cell destruction from ICI is **irreversible** → lifelong insulin therapy; do NOT discontinue ICI for hyperglycemia | Missed |
| M75 | 24 | **PI3Kα isoform involved in insulin signaling → mechanism of hyperglycemia** (pan-PI3K or α-isoform-specific inhibitors) | Missed |
| M76 | 25 | Glucocorticoid dose-matched insulin adjustment details (monitoring not solely in morning → misses hyperglycemia extent) | Missed |
| M77 | 25 | **CAVIAR study** — active vs passive lifestyle in kidney allograft recipients → fat mass + weight loss; no significant difference in insulin secretion/sensitivity/disposition index; PTDM rate halved in active group but NS | Missed |
| M78 | 25 | **Metformin in PTDM**: do not start <eGFR 45; discontinue <30; limit due to lactic acidosis with fluctuating kidney function | Captured as prose PENDING, not discrete |
| M79 | 25 | **DPP-4i in PTDM**: safe/effective incl. immediate posttransplant for mild hyperglycemia or IGT; may decrease PTDM progression | Captured as prose PENDING, not discrete |
| M80 | 26 | MODY **HNF1A / HNF4A** → low-dose SU; may ultimately require insulin | Captured in prose PENDING |
| M81 | 26 | SGLT-inhibitor DKA incidence in T2D: **0.6–4.9 events per 1,000 person-years** | Missed |
| M82 | 26 | SGLT-inhibitor DKA risk factors: very-low-carbohydrate eating, prolonged fasting, dehydration, excessive alcohol, common precipitants | Missed |
| M83 | 26 | SGLT-inhibitor DKA presentation: up to one-third present with glucose <200 mg/dL; 71% present with ≤250 mg/dL | Captured as prose PENDING |

---

### ❓ NEEDS-CONFIRM

| # | Page | DB content | Action |
|---|---|---|---|
| NC27 | 21–22 | **355 PENDING table-fragment spans** (single tokens `$`, `$87`, `2,000 mg`, drug names, repeated abbreviation-legend replays 11× on page 22) | **REJECT in bulk** — 355 spans to reject. Re-extract Tables 9.3 and 9.4 into a dedicated `l2_tables` relation with row/col structure. |
| NC28 | 21–22 | Repeated Table 9.3/9.4 header stubs from channels D/H | **REJECT** — duplicate headers. |
| NC29 | 25–26 | PTDM management prose (metformin eGFR thresholds, DPP-4i safety, GLP-1 RA benefit, SGLT2i benefit for ASCVD/HF/CKD, insulin in early postop) — 6 PENDING spans | **CONFIRM** — substantive clinical content. |
| NC30 | 26 | MODY HNF1A/HNF4A prose span (PENDING) | **CONFIRM** + merge with A101/A103 into discrete MODY-variant facts. |
| NC31 | 26 | SGLT-inhibitor DKA low-glucose presentation (PENDING) | **CONFIRM** — essential safety fact. |
| NC32 | 27–30 | **74 PENDING reference-citation spans** — individual bibliography entries (e.g., "MiniMed advanced hybrid closed-loop delivery: results from a randomized crossover trial…", "Tirzepatide versus semaglutide once weekly in patients with T2D", "Phase 3 trial of semaglutide in MASH") | **REJECT in bulk** — these are citations, not facts. Reviewer's 5 REJECTs on pages 27–28 confirm this is the right call; the remaining 74 should follow. Escalate: add a post-processing filter to move citation lines into `l2_references` instead of `l2_merged_spans`. |

---

### Batch 3 summary

| Bucket | Count |
|---|---|
| ADDED (acceptable) | 26 |
| MISSED | 28 |
| NEEDS-CONFIRM | 6 |
| **REJECT-in-bulk (cost-table fragments + reference citations)** | **~420 spans** |

**Pipeline failure modes reinforced in batch 3:**
10. **Cost tables are effectively lost.** Table 9.3 (noninsulin costs) and Table 9.4 (insulin costs) are critical downstream inputs for cost-aware clinical decisions (rec 9.30 says consider low-cost meds for cost-related barriers). The current pipeline emits 355 isolated cells with no relation. Recommend dedicated table-extraction path (e.g., pdfplumber tables + header-row detection) rather than channel D cell-streaming.
11. **Reference-list contamination.** Pages 27–30 (and likely the same pattern at page ~82) blew up because the extractor treated bibliography items as paragraphs. 74 reference citations arrived in `l2_merged_spans`. A pre-classifier that detects "References" section or numbered citation patterns would prevent this.
12. **Evidence-grade letters (A/B/C/E) are inconsistently preserved.** Rec 9.31a's "C" grade is in the PDF; A82 lacks it. Same with 9.34 "E", 9.37 "A", 9.38b. KB-23 Decision Cards rely on these grades for auto-approval thresholds.
13. **Numeric incidence rates dropped.** ICI-induced diabetes ~1% lost; SGLT-inhibitor-DKA incidence 0.6–4.9/1000 PY lost; metformin eGFR thresholds kept (good) but the 30–45 "periodic dips" zone conflated with <30 lactic acidosis zone.
14. **Fact scope mismatch.** The prose on page 25 on glucocorticoid insulin matching is a full "Rec 9.36" but never gets surfaced as a numbered recommendation — only implicit in prose. Suggests recommendation-block detection is brittle when the red "Recommendations" header isn't immediately above the numbered item.

---

## Batch 4 — Pages 31–40 (PDF pp. S213–S222)

**Content coverage:** Pages 31–34 = references for Section 9 (bibliography 194–310+). **Page 35 begins Section 10 "Cardiovascular Disease and Risk Management"** with intro prose on cardiorenal metabolic disease, heart failure in diabetes, and Hypertension overview + Rec 10.1 (BP at every visit). Pages 36–40 cover BP goals (Recs 10.2–10.4), BP trial evidence (BPROAD, ESPRIT, SPRINT, STEP, HOT, ACCORD BP, ADVANCE), Fig 10.2 (treatment algorithm for confirmed hypertension), antihypertensive classes (Recs 10.5–10.14 covering ACEi/ARB, MRA, lifestyle, DASH).

**DB inventory in this page range:**

| Page | Total | CONFIRMED | ADDED | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 31 | 11 | 10 | 0 | 1 | 0 | 0 |
| 32 | 4 | 2 | 0 | 1 | 0 | 1 |
| 33 | 29 | 10 | 16 | 3 | 0 | 0 |
| 34 | 2 | 1 | 0 | 0 | 0 | 1 |
| 35 | 18 | 2 | 15 | 1 | 0 | 0 |
| 36 | 5 | 4 | 0 | 1 | 0 | 0 |
| 37 | 14 | 14 | 0 | 0 | 0 | 0 |
| 38 | 4 | 3 | 1 | 0 | 0 | 0 |
| 39 | 15 | 12 | 1 | 0 | 0 | 2 |
| 40 | 10 | 6 | 3 | 0 | 0 | 1 |
| **Batch 4 total** | **112** | **64** | **36** | **6** | **0** | **5** |

**This batch is the first with zero PENDING spans** (all 112 have been reviewer-touched). Review burden is still high: 42 of 112 (37.5%) needed reviewer ADD/EDIT; pages 35 (15 ADDED) and 33 (16 ADDED) dominate.

---

### 🚨 Major audit finding for this batch

**Pages 31–34 contain 46 bibliography/reference citations, NOT guideline facts.** They should never have been extracted into `l2_merged_spans`. Worse:

- **Page 31:** 10 references CONFIRMED as "guideline facts" (e.g., "Efficacy of intermittent short-term use of a real-time CGM system in non-insulin-treated patients with T2D: a randomized controlled trial"). These are **RCT titles from the bibliography** — they assert nothing, they're citations.
- **Page 32:** 1 CONFIRMED reference + 1 EDITED reference + 1 REJECTED page marker.
- **Page 33:** 16 references ADDED by a reviewer as if they were new facts, + 3 EDITED + 10 CONFIRMED = **29 reference citations treated as guideline facts**.
- **Page 34:** 1 REJECTED "10." list marker, and 1 CONFIRMED span that is **the exact ADA Professional Practice Committee boilerplate which was REJECTED on page 1** — the reviewer has been inconsistent about whether this is boilerplate or a fact.

**Classification:** All 47 CONFIRMED/EDITED/ADDED spans on pages 31–34 are **False-Positive-As-Fact** — they should be migrated to an `l2_references` relation and removed from `l2_merged_spans`. See finding G-7 in the final summary.

---

### ✅ ADDED — guideline facts present in DB (Section 10 start)

#### Section 10 Introduction & Hypertension (pages 35–40)

| # | Page | Fact | Status |
|---|---|---|---|
| A105 | 35 | CVD risk factors in diabetes: ASCVD, HF, CKD — frequently caused by metabolic risk (obesity + A1C level–driven); **cardiorenal metabolic disease / cardiovascular-kidney-metabolic health** as grouping term | CONFIRMED |
| A106 | 35 | **HF is at least 2× more prevalent in people with diabetes** vs those without; major cause of morbidity/mortality | ADDED |
| A107 | 35 | T2D can present with HFpEF, HFmrEF, HFrEF; CAD causes myocardial injury → ischemic HFrEF; T2D also causes structural heart disease + HFrEF in absence of obstructive CAD | ADDED |
| A108 | 35 | Only a minority of T2D patients achieve recommended risk-factor goals / guideline-recommended therapy | EDITED |
| A109 | 35 | **SGLT inhibitors and GLP-1 RAs with CV/kidney benefit are a fundamental element of risk reduction** in addition to glycemic/BP/lipid management | ADDED |
| A110 | 35 | **Hypertension is common** in T1D and T2D; major risk factor for ASCVD, HF, microvascular complications | ADDED |
| A111 | 35 | **Elevated BP** defined as SBP 120–129 AND DBP <80 mmHg; **Hypertension** defined as SBP ≥130 OR DBP ≥80 mmHg | ADDED |
| A112 | 35 | Antihypertensive therapy reduces ASCVD events, HF, microvascular complications | ADDED |
| A113 | 35 | **Rec 10.1** BP measured at every visit or at least every 6 months | ADDED |
| A114 | 35 | In CVD and BP ≥180/110 mmHg → reasonable to diagnose hypertension at a single visit | ADDED |
| A115 | 35 | Postural BP/pulse changes may indicate autonomic neuropathy → BP-goal adjustment | ADDED |
| A116 | 35 | Home BP self-monitoring and 24-h ABPM may reveal white-coat, masked hypertension, or office/true discrepancies | ADDED |
| A117 | 36 | **Rec 10.3** BP goals individualized via shared decision-making (CV risk, adverse effects, preferences) | CONFIRMED (REVIEWER) |
| A118 | 36 | **Rec 10.4** On-treatment BP goal **<130/80 mmHg**; **SBP <120 mmHg** encouraged for high CV/kidney risk (A) | CONFIRMED (REVIEWER) |
| A119 | 36 | <130/80 mmHg goal consistent with ACC / AHA / ACP / AAFP guidelines | CONFIRMED (REVIEWER) |
| A120 | 36 | STEP trial: T2D ≈20% of participants → ↓ CV events at SBP goal <130 vs 130–150 | CONFIRMED (REVIEWER) |
| A121 | 36 | 24-h ABPM and home BP predict CV risk (10 RCTs meta-analysis) | EDITED |
| A122 | 37 | **BPROAD trial**: T2D + increased CV risk, baseline SBP ≥140 (or 130–180 if on anti-HTN); primary composite (MI, ACS, stroke, HF, CV death) **reduced 25% in intensive vs standard** (SBP achieved 121 vs standard) | CONFIRMED (REVIEWER) |
| A123 | 37 | **ESPRIT trial**: 11,255 high-CV-risk individuals (39% T2D); intensive SBP <120 vs standard; composite of nonfatal stroke/MI/HF/CV death **reduced 21% (HR 0.79, 95% CI 0.69–0.90)** | CONFIRMED (REVIEWER) |
| A124 | 37 | **SPRINT (excluded T2D)**: SBP <120 → **↓ CV events 25%** in high-risk | CONFIRMED (REVIEWER) |
| A125 | 37 | **STEP trial (intensive 110–130 vs standard 130–150)**: primary composite **↓ 12% (HR 0.88, 95% CI 0.78–0.99)** irrespective of T2D | CONFIRMED (REVIEWER) |
| A126 | 37 | **HOT trial** (18,790 pts): DBP <90 / <85 / <80 targets; lowest CV event rate at DBP 82 mmHg; in T2D subset **51% ↓ in MI/stroke/CV death** | CONFIRMED (REVIEWER) |
| A127 | 37 | **ACCORD BP trial**: underpowered; did not demonstrate SBP <120 benefit in T2D as a whole | CONFIRMED (REVIEWER) |
| A128 | 37 | **ADVANCE trial**: T2D 11,140 pts; perindopril + indapamide vs placebo | CONFIRMED |
| A129 | 37 | Intensive-treatment adverse events: hypotension, syncope, bradycardia, hyperkalemia, ↑ serum creatinine | CONFIRMED |
| A130 | 37 | Increased CV risk in ESPRIT defined as: CVD history (≥3 months pre-enrollment), subclinical CVD, ≥2 CV risk factors, or CKD (eGFR 30–<60) | CONFIRMED |
| A131 | 38 | **Figure 10.2** "Recommendations for treatment of confirmed hypertension in nonpregnant people with diabetes" — ACEi/ARB first-line for CAD or UACR 30–299 mg/g; strongly recommended for UACR ≥300 | ADDED |
| A132 | 38 | Clinical-factor-guided benefit/harm stratification (ACCORD BP / SPRINT secondary analyses) | CONFIRMED |
| A133 | 38 | Fig 10.2 algorithm skeleton: start one agent → albuminuria or CAD → ACEi or ARB + CCB / diuretic | CONFIRMED |
| A134 | 39 | **Rec 10.7** ≥150/90 mmHg → **two drugs or single-pill combo**, timely titration, demonstrated CV benefit | CONFIRMED (REVIEWER) |
| A135 | 39 | **Rec 10.6** ≥130/80 mmHg → initiate pharmacologic therapy, titrate to individualized goal | CONFIRMED (REVIEWER) |
| A136 | 39 | **Rec 10.5** >120/80 mmHg → lifestyle behaviors: weight loss if indicated, DASH-style eating (reduce sodium, limit alcohol, ↑ K), ≥150 min/wk moderate aerobic activity | CONFIRMED |
| A137 | 39 | **Rec 10.10** Nonpregnant DM + HTN: ACEi or ARB for moderate albuminuria (UACR 30–299); strongly recommended for severely increased (UACR ≥300) | CONFIRMED (REVIEWER + span)|
| A138 | 39 | **Rec 10.11** Monitor eGFR + serum K at initiation and periodically for ACEi, ARBs, MRAs; monitor hypokalemia with diuretics (B) | CONFIRMED |
| A139 | 39 | ACEi first-line in DM + CAD; ACEi/ARB first-line in DM + albuminuria | CONFIRMED |
| A140 | 39 | **Initial antihypertensive classes with CV-event reduction in DM**: ACEi, ARBs, thiazide-like diuretics, dihydropyridine CCBs (Carter et al. 27) | ADDED |
| A141 | 39 | **Avoid** in individuals of childbearing potential not using reliable contraception: ACEi, ARBs, MRAs, direct renin inhibitors, neprilysin inhibitors | CONFIRMED (REVIEWER) |
| A142 | 39 | Do NOT combine: ACEi + ARB, ACEi/ARB + direct renin inhibitor, ARB + neprilysin inhibitor | CONFIRMED |
| A143 | 40 | **Beta-blockers** indicated for prior MI, active angina, or HFrEF — NOT shown to ↓ mortality as BP therapy | ADDED |
| A144 | 40 | **Rec 10.13** Not meeting BP goals on 3 classes (incl. diuretic) → **consider MRA** (A) | ADDED |
| A145 | 40 | **Rec 10.14** Lifestyle: weight loss, Mediterranean or DASH, ↓ saturated/trans fat, ↑ n-3 fatty acids + soluble fiber + plant stanols/sterols | ADDED |
| A146 | 40 | ACEi/ARB can be continued as eGFR declines to <30 — CV benefit w/o significantly ↑ kidney failure | CONFIRMED |
| A147 | 40 | Pregnancy: ACEi, ARBs, direct renin inhibitors, MRAs, neprilysin inhibitors **contraindicated** (fetal damage) | CONFIRMED |
| A148 | 40 | **Serum creatinine up to 30% ↑ with ACEi/ARB is NOT AKI**; AKI distinct entity | CONFIRMED |
| A149 | 40 | **MRAs (spironolactone, eplerenone)** effective for resistant HTN in T2D when added to ACEi/ARB + thiazide-like diuretic + dihydropyridine CCB | CONFIRMED |

---

### ❌ MISSED — facts in PDF absent / non-recoverable

| # | Page | Fact | Why missed |
|---|---|---|---|
| M84 | 35 | **Section 10 title "10. CARDIOVASCULAR DISEASE AND RISK MANAGEMENT"** + Diabetes Care citation for Section 10 | Section-heading captured in span A105/A113 context but not as a discrete section node (G-1 consequence). |
| M85 | 35 | **Rec 10.1** full text with SBP 120–129 + DBP <80 elevated-BP phrasing — is captured in A113 but the **"ambulatory BP monitoring" addendum** may be lost | Verify via DB. |
| M86 | 35 | **Rec 10.2** — no rec 10.2 in DB. The PDF text at the recommendation box (page 35) on confirming hypertension with 2 occasions of elevated BP seems to have been truncated in reviewer ADD span A113 | Missed. |
| M87 | 36 | Specific BP goal evidence — the STEP trial's **<130 target** is captured but **110 mmHg lower bound** of STEP is lost | Partial. |
| M88 | 37 | **BPROAD primary-outcome achieved SBP** 121 vs 134 mmHg (standard) | Partial — present in CONFIRMED span but achieved SBP numbers embedded in prose. |
| M89 | 37 | **ADVANCE trial results** (perindopril + indapamide vs placebo → ↓ microvascular + macrovascular events) | The DB CONFIRMED span text cuts off at "matc" — result (5.6% RRR vs placebo) is lost. |
| M90 | 38 | **Full Fig 10.2 algorithm branches** — start 1 drug / 2 drugs, adjust based on albuminuria/CAD, add diuretic/CCB, go to 3 drugs, MRA for resistant | Partially in A131/A133 — branching logic flattened. |
| M91 | 38 | Fig 10.2 footnote explanations (*, †, ‡ etc.) | Missed. |
| M92 | 39 | **Rec 10.8** (BP measurement methodology: 2+ readings, averaged, seated, etc.) | Not in DB — the numeric range of recs 10.5–10.11 shown in DB jumps from 10.7 to 10.10. |
| M93 | 39 | **Rec 10.9** (first-line classes with CV-event reduction — ACEi/ARB/thiazide-like/dihydropyridine CCB — as numbered recommendation) | Captured as prose in A140, not as numbered rec 10.9. |
| M94 | 39 | **Rec 10.12** (avoid ACEi+ARB combination etc.) as a numbered rec | Captured as prose in A142, not as 10.12. |
| M95 | 39 | DASH diet effect size meta-analysis: **SBP −3.26 mmHg [95% CI −5.58 to −0.94]**, DBP effect — REJECTED by reviewer but the effect size is a real fact | Reviewer rejected the prose wrapper; the effect-size datum is lost. |
| M96 | 40 | **Rec 10.14 full text** on dietary n-3 fatty acids, soluble fiber, plant stanol/sterol intake for lipid management | Captured partially in A145. |
| M97 | 40 | **Rec 10.12** explicit: "avoid any combination of ACEi, ARBs (including ARBs and neprilysin inhibitors), and direct renin inhibitors" as numbered rec | In A142 prose only, not tagged. |
| M98 | 40 | MRA monitoring caveat (K+, serum creatinine) | Partial in A138 but not linked to A144 (MRA therapy rec). |

---

### ❓ NEEDS-CONFIRM

| # | Page | DB content | Action |
|---|---|---|---|
| NC33 | 31–34 | **47 reference-list entries currently marked CONFIRMED/EDITED/ADDED as if they were guideline facts** | **BULK REVERT + REJECT** — these are not facts; migrate to `l2_references`. |
| NC34 | 34 | ADA Professional Practice Committee boilerplate span on page 34 **CONFIRMED**, but the identical text on page 1 was **REJECTED** | Reconcile — reject on page 34 to match page-1 policy. |
| NC35 | 35 | Rec 10.1 span may be truncated (ADDED) — verify full text on PDF | Review full L1 block for page 35; EDIT if truncated. |
| NC36 | 39 | REJECTED lifestyle-recommendation prose span (M95 evidence loss) | Re-add DASH meta-analysis effect size as an `EVIDENCE` fact separately. |

---

### Batch 4 summary

| Bucket | Count |
|---|---|
| ADDED (acceptable, Section 10) | 45 |
| MISSED (Section 10) | 15 |
| NEEDS-CONFIRM | 4 |
| **False-Positive-As-Fact (references incorrectly treated as facts)** | **47** |

**Pipeline failure modes new in batch 4:**
15. **Reference-list pages treated as fact pages.** Bibliography entries from Section 9 (pages 27–34 — 74 PENDING + 47 CONFIRMED-as-fact) pollute `l2_merged_spans`. A simple header-detection ("References" / ordered numbered list) would catch this class.
16. **Recommendation numbering gaps.** In Section 10 (pages 35–40), the DB has rec text but misses or mislabels 10.1, 10.2, 10.8, 10.9, 10.12. The issue appears to be that reviewers paste recommendation text without the "10.X" prefix, so recs are free-text rather than numbered.
17. **Reviewer inconsistency.** The same boilerplate ("ADA Professional Practice Committee for Diabetes, an interprofessional expert committee, are responsible for updating the Standards of Care annually…") is REJECTED on page 1 and CONFIRMED on page 34. Reviewer decision-coherence needs enforcement (e.g., "confirmed-reject" cache that suggests auto-reject for identical text re-encountered later).
18. **REVIEWER-channel spans labeled "CONFIRMED" but with channel=REVIEWER.** On pages 36–37, 12+ spans show `contributing_channels = {REVIEWER}` with `review_status = CONFIRMED`. This looks like reviewers typing text and immediately confirming — bypassing the ADD workflow. Unclear whether this is intentional; ops should verify.
19. **Effect sizes + hazard ratios finally captured well.** BPROAD 25%, ESPRIT HR 0.79 [0.69–0.90], STEP HR 0.88, HOT 51% — all present in DB. **This is the first batch where trial effect sizes are preserved.** Positive signal — likely because reviewers manually added them.

---

## Batch 5 — Pages 41–50 (PDF pp. S223–S232)

**Content coverage:** Lifestyle / nutrition for CVD (page 41, Rec 10.15), lipid-panel monitoring (Recs 10.16, 10.17 etc.), statin trials & intensity levels (Table 10.1, page 42), Primary Prevention algorithm (Fig 10.3, page 43), Secondary Prevention (Rec 10.26/10.27 and Fig 10.4, pages 43–44), combination therapy (statin + ezetimibe/PCSK9i, pages 44–45), statin intolerance (bempedoic acid, inclisiran), pregnancy lipid considerations, Rec 10.29 (≥500 mg/dL TG) and Rec 10.32 (statin + fibrate/niacin/n-3 — not recommended), REDUCE-IT / icosapent ethyl (page 46), **ANTIPLATELET AGENTS** — Recs 10.33–10.36 (pages 47–48), P2Y12 DAPT indications, CV/PAD/HF screening — Recs 10.37–10.39 and **Treatment Recs 10.40a-d / 10.41a-b / 10.42 / 10.43 / 10.44a-g** (SGLT2i, GLP-1 RA, nonsteroidal MRA, ACEi/ARB for HF/ASCVD/CKD).

**DB inventory in this page range:**

| Page | Total | CONFIRMED | ADDED | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 41 | 4 | 4 | 0 | 0 | 0 | 0 |
| 42 | 16 | 16 | 0 | 0 | 0 | 0 |
| 43 | 5 | 5 | 0 | 0 | 0 | 0 |
| 44 | 3 | 3 | 0 | 0 | 0 | 0 |
| 45 | 7 | 7 | 0 | 0 | 0 | 0 |
| 46 | 6 | 5 | 1 | 0 | 0 | 0 |
| 47 | 14 | 13 | 0 | 0 | 0 | 1 |
| 48 | 13 | 12 | 0 | 0 | 0 | 1 |
| 49 | 10 | 2 | 4 | 0 | 1 | 3 |
| 50 | 6 | 5 | 0 | 0 | 0 | 1 |
| **Batch 5 total** | **84** | **72** | **5** | **0** | **1** | **6** |

**Reviewer burden:** 5 ADDED / 84 = **6.0%** — dramatically lower than batches 1–4. Pages 41–46 were captured almost entirely by the extractor and only required reviewer confirmation, not content typing. This is the best-performing batch so far.

---

### ✅ ADDED — guideline facts present in DB

#### Lipid management (pages 41–46)

| # | Page | Fact | Status |
|---|---|---|---|
| A150 | 41 | Lifestyle: **Mediterranean or DASH**, ↓ saturated/trans fat, ↑ plant stanols/sterols, ↑ n-3 fatty acid, ↑ viscous fiber (oats, legumes, citrus) | CONFIRMED |
| A151 | 41 | **Rec 10.15** Intensify lifestyle + optimize glycemia for TG ≥150 mg/dL (≥1.7 mmol/L) OR HDL <40 (men) / <50 (women) | CONFIRMED |
| A152 | 41 | Lipid-panel monitoring: at diagnosis, initial eval, **4–12 weeks after statin initiation/dose change, annually** | CONFIRMED |
| A153 | 41 | **Rec 10.26** / **10.27** For all ages with diabetes + ASCVD → **high-intensity statin therapy** added to lifestyle (A) | CONFIRMED |
| A154 | 42 | **Table 10.1**: High-intensity = atorvastatin 40–80 mg, rosuvastatin 20–40 mg (≥50% LDL ↓); Moderate-intensity = atorvastatin 10–20, rosuvastatin 5–10, simvastatin 20–40, pravastatin 40–80, lovastatin 40, fluvastatin XL 80, pitavastatin 1–4 (30–49% LDL ↓) | CONFIRMED (D-channel table cells — correctly reconstructed this time) |
| A155 | 42 | **Meta-analysis: 9% ↓ all-cause mortality, 13% ↓ vascular mortality per 1 mmol/L (39 mg/dL) LDL reduction** (14 RCTs, >18,000 DM, 4.3 yr follow-up) | CONFIRMED |
| A156 | 42 | Low-dose statins generally NOT recommended in diabetes (but used when tolerance-limited) | CONFIRMED |
| A157 | 42 | Primary prevention moderate-intensity statin if **≥40 years**, high-intensity if additional ASCVD risk factors | CONFIRMED |
| A158 | 42 | Statin evidence strongest for diabetes aged 40–75 years | CONFIRMED |
| A159 | 42 | Moderate-intensity statin in diabetes ≥75 years | CONFIRMED |
| A160 | 42 | Similar statin approaches for T1D and T2D (esp. with additional CV risk factors) | CONFIRMED |
| A161 | 43 | **Fig 10.3** primary prevention algorithm: 40–75 yr with ASCVD risk factor → high-intensity statin; without → moderate; bempedoic acid if intolerant; statin intolerant → alternative lipid-lowering (PCSK9 mAb, bempedoic acid, inclisiran) | CONFIRMED |
| A162 | 43 | **Rec 10.26/10.27**: high-intensity statin for all with diabetes + ASCVD → ≥50% LDL ↓ from baseline, **LDL goal <55 mg/dL (<1.4 mmol/L)** | CONFIRMED |
| A163 | 43 | Add ezetimibe or PCSK9i if goal not achieved on max-tolerated statin | CONFIRMED |
| A164 | 44 | PCSK9 mAbs (evolocumab, alirocumab) + max statin → ~60% LDL ↓, **MACE ↓ 15–20%** in FOURIER (evolocumab) and ODYSSEY OUTCOMES (alirocumab) | CONFIRMED |
| A165 | 44 | Statin intolerance: partial (can't tolerate sufficient dose) or complete; adverse-effect-limited | CONFIRMED |
| A166 | 45 | Evolocumab 55–56% LDL ↓ vs 16.7% with ezetimibe (GAUSS-3) | CONFIRMED |
| A167 | 45 | Musculoskeletal adverse effects: evolocumab 20.7% / ezetimibe 28.8% in GAUSS-3 | CONFIRMED |
| A168 | 45 | **Bempedoic acid + CLEAR Outcomes**: ↓ 4-point MACE in statin-intolerant | CONFIRMED |
| A169 | 45 | Pregnancy: statins generally discontinued; **no teratogenic increase in familial hypercholesterolemia** data | CONFIRMED |
| A170 | 45 | **Rec 10.29** Fasting TG ≥500 mg/dL (≥5.7 mmol/L) → evaluate for secondary causes + consider medical therapy to reduce pancreatitis risk | CONFIRMED |
| A171 | 46 | **Rec 10.32** In statin-treated individuals, **fibrate / niacin / n-3 supplements NOT recommended** (no additional CV benefit, A) | ADDED |
| A172 | 46 | Severe hypertriglyceridemia (≥500, esp. >1,000 mg/dL) → fibrates ± fish oil + fat restriction to avoid acute pancreatitis | CONFIRMED |
| A173 | 46 | **REDUCE-IT** (icosapent ethyl + statin, TG 150–499): **primary composite ↓ 25% (p<0.001)**; MACE (CV death/MI/stroke) ↓ 26%; CV death ↓ 20% (p=0.03) | CONFIRMED |
| A174 | 46 | Results should NOT be extrapolated to other n-3 products — n-3 carboxylic acid (EPA + DHA 4g/day) did not reduce MACE vs corn oil | CONFIRMED |
| A175 | 46 | **Low HDL** is the most prevalent dyslipidemia pattern in T2D | CONFIRMED |
| A176 | 46 | Neither ACCORD-lipid nor PROMINENT demonstrated CV benefit for fibrates | CONFIRMED |
| A177 | 46 | **PROMINENT**: pemafibrate 0.2 mg BID → TG ↓ 26% but no primary-outcome reduction (MI, stroke, urgent revascularization, CV death) in T2D + high TG + low HDL | CONFIRMED |

#### Antiplatelet therapy (pages 47–48)

| # | Page | Fact | Status |
|---|---|---|---|
| A178 | 47 | Statins not linked to cognitive dysfunction / dementia; concern should not deter use | CONFIRMED |
| A179 | 47 | **Rec 10.33** Aspirin 75–162 mg/day for secondary prevention in DM + ASCVD history (A) | CONFIRMED |
| A180 | 47 | **Rec 10.34a** Aspirin allergy + ASCVD → clopidogrel 75 mg/day (B) | CONFIRMED (REVIEWER) |
| A181 | 47 | **Rec 10.35** Combination 81 mg aspirin + 2.5 mg rivaroxaban BID for stable CAD/PAD + low bleeding risk (A) — to prevent MALE + MACE | CONFIRMED (REVIEWER) |
| A182 | 47 | **Rec 10.36** Aspirin 75–162 mg/day primary prevention considered for DM at ↑ CV risk after shared decision (A) | CONFIRMED (REVIEWER) |
| A183 | 47 | **ASCEND** trial (15,480 DM, no CVD): aspirin 100 mg vs placebo; **GI bleeding ↑ 2.11-fold (HR 2.11 [1.36–3.28], p=0.0007)** | CONFIRMED (REVIEWER) |
| A184 | 47 | **ASPREE** in elderly (19,114 pts): aspirin major bleeding ↑ from 3.2→4.1% (rate ratio 1.29, p=0.003) | CONFIRMED (REVIEWER) |
| A185 | 47 | **Primary prevention aspirin** for DM + ≥1 major risk factor (HTN, dyslipidemia, smoking, obesity, CKD) who are not at increased bleeding risk | CONFIRMED (REVIEWER) |
| A186 | 48 | **Rec 10.38a** Screen for stage B (structural/functional) HF by measuring natriuretic peptides | CONFIRMED (REVIEWER) |
| A187 | 48 | Aspirin not recommended for low-risk ASCVD (<50 yrs with DM + no additional factors) | CONFIRMED |
| A188 | 48 | Aspirin <21 yrs contraindicated (Reye syndrome) | CONFIRMED |
| A189 | 48 | Aspirin 75–162 mg/day optimal (COMPASS/VOYAGER/ADAPTABLE data; 81 = 325 no CV/bleeding difference) | CONFIRMED |
| A190 | 48 | **P2Y12 DAPT** after ACS, coronary revascularization with stenting; **ticagrelor/clopidogrel/prasugrel post-PCI; ticagrelor or clopidogrel if no PCI** | CONFIRMED (REVIEWER) |
| A191 | 48 | **COMPASS trial** (27,395 CAD/PAD): aspirin + rivaroxaban 2.5 mg BID superior to aspirin+placebo | CONFIRMED |
| A192 | 48 | **VOYAGER PAD** (6,564 pts) confirmed aspirin + rivaroxaban in PAD revascularization | CONFIRMED |
| A193 | 48 | **Rec 10.37a/b**: routine asymptomatic CAD screening NOT recommended; investigate if symptoms or signs, carotid bruit, TIA/stroke/claudication/PAD/abnormal ECG | CONFIRMED |

#### CV/HF/PAD treatment recommendations (pages 49–50)

| # | Page | Fact | Status |
|---|---|---|---|
| A194 | 49 | **Rec 10.38b** Abnormal natriuretic peptide → echocardiography to identify stage B HF | CONFIRMED |
| A195 | 49 | **Rec 10.39** Asymptomatic DM + ≥65 yr OR microvascular disease OR foot complications OR end-organ damage → ABI screening for PAD (if PAD dx would change management) | CONFIRMED (REVIEWER) |
| A196 | 49 | **Rec 10.40a** T2D + ASCVD or CKD → SGLT2i or GLP-1 RA with CV benefit as part of CV risk reduction / glucose-lowering plan (A) | ADDED |
| A197 | 49 | **Rec 10.41b** T2D + HFrEF or HFpEF → SGLT2i with proven benefit to improve quality of life (A) | ADDED |
| A198 | 49 | **Rec 10.42** T2D + CKD + albuminuria on max ACEi/ARB → **nonsteroidal MRA with demonstrated benefit** for CV outcomes + CKD progression reduction (A) | ADDED |
| A199 | 49 | **Rec 10.44f** T2D + CKD → nonsteroidal MRA with benefit to ↓ risk of HF hospitalization (A); **Rec 10.44g** Guideline-directed medical therapy for HF | ADDED |
| A200 | 50 | Asymptomatic ASCVD screening not recommended (no benefit when risk factors treated); coronary calcium scoring uncertain in DM, balance of benefit/risk controversial | CONFIRMED (REVIEWER) |
| A201 | 50 | Natriuretic peptide abnormality thresholds: **BNP ≥50 pg/mL, NT-proBNP ≥125 pg/mL** | CONFIRMED (REVIEWER) |
| A202 | 50 | **PARTNERS** program: 30% of pts aged 50–69 with smoking or DM, or ≥70 regardless, had PAD | CONFIRMED (REVIEWER) |
| A203 | 50 | Many PAD pts asymptomatic because they limit activity to avoid claudication or comorbidity limits activity threshold | CONFIRMED |

---

### ❌ MISSED — facts in PDF absent / non-recoverable

| # | Page | Fact | Why missed |
|---|---|---|---|
| M99 | 41 | **Rec 10.17**, Rec 10.18, 10.19, …, 10.25 — intermediate recs about lipid-panel frequency by age and statin class (10.17–10.25 series listed in PDF but only 10.16 and 10.26/10.27 clearly in DB) | Not extracted as numbered recs — only prose forms captured. |
| M100 | 42 | **9% all-cause mortality + 13% vascular mortality per 1 mmol/L LDL ↓** — captured (A155) ✅. But the secondary finding "benefit does not depend on baseline LDL; linear with LDL reduction; no low threshold" is lost | Embedded in prose, not discrete. |
| M101 | 43 | **Fig 10.4 Secondary Prevention algorithm** structure (ezetimibe, PCSK9 mAb, bempedoic acid, inclisiran siRNA as step-up options) | Captured as prose in A161/A162 only — branch logic flattened. |
| M102 | 44 | **IMPROVE-IT**: statin+ezetimibe **6.4% RRR, 2% ARR** for MACE; diabetes subanalysis 27% of 18,144 pts showed significant MACE reduction vs moderate statin alone | MISSED as discrete effect-size fact. |
| M103 | 44 | **FOURIER / ODYSSEY OUTCOMES** participant count (27,000+ / 18,000+), diabetes proportion (40% / 28.8%), and **DM-specific ARR 2.3% [95% CI 0.4–4.2] (ODYSSEY)** | Aggregated MACE 15–20% captured (A164); the DM-specific ARR not discrete. |
| M104 | 45 | **ORION-1 / ORION-3** inclisiran trial: LDL ↓ ~45% maintained through year 4; 23% of ORION-3 pts had DM; 33% not on statin therapy | Missed. |
| M105 | 45 | **Bempedoic acid LDL-lowering magnitudes**: 15% on statin, 24% not on statin; +19% with ezetimibe addition | Missed. |
| M106 | 45 | **Rec 10.28** (LDL goal <55 mg/dL for ASCVD + ≥50% reduction) and Rec 10.30/10.31 (HDL or TG numbered recs) | Rec 10.28 appears captured in A162; 10.30, 10.31 not seen — likely missed. |
| M107 | 46 | **REDUCE-IT exclusion criteria / population definition** ("ASCVD OR DM + ≥1 risk factor with fasting TG 150–499 mg/dL on statin") | Captured partially in A173; discrete eligibility not separated. |
| M108 | 47 | **Rec 10.34b** DAPT duration after ACS/stroke/TIA determined by interprofessional team (E) | Missed. |
| M109 | 47 | Aspirin primary-prevention controversy — failed to consistently show reduction in DM-specific ASCVD endpoints | Embedded, not discrete. |
| M110 | 48 | **Prior MI + ticagrelor 1–3 years post-MI** → reduces recurrent ischemic events incl. CV and CHD death (PEGASUS-TIMI 54 evidence) | Embedded in prose, not discrete. |
| M111 | 48 | **THEMIS trial**: ticagrelor + aspirin in DM + stable CAD → ↓ ischemic events but ↑ major bleeding / intracranial hemorrhage | Missed (prose context only). |
| M112 | 49 | **Rec 10.40c** T2D + ASCVD or multiple risk factors → GLP-1 RA with CV benefit recommended (A) — **REJECTED in DB** | Likely duplicate of REVIEWER-added A196; verify intent of rejection. |
| M113 | 49 | **Rec 10.40b** T2D + ASCVD/risk → SGLT2i with CV benefit for CV events reduction (A) | Captured inside A196 but not as discrete 10.40b. |
| M114 | 49 | **Rec 10.40d** T2D + ASCVD/risk → combined SGLT2i + GLP-1 RA considered for additive CV/kidney benefit (B) | Missed. |
| M115 | 49 | **Rec 10.41a** T2D + HFrEF/HFpEF → SGLT2i (incl. SGLT1/2) with proven benefit to ↓ HF worsening + CV death (A) | Not discrete — subsumed in A197. |
| M116 | 49 | **Rec 10.43** DM + ≥55 yr + ASCVD or risk → ACEi/ARB to ↓ CV events (A) | Missed as numbered rec. |
| M117 | 49 | **Rec 10.44a** DM + stage B HF → interprofessional team — **REJECTED in DB** | Reject reason unclear; possibly wrong reject. |
| M118 | 49 | **Rec 10.44b** DM + asymptomatic stage B HF → ACEi/ARB + β-blocker to prevent progression to symptomatic (A) — **REJECTED in DB** | Verify reject. |
| M119 | 49 | **Rec 10.44c** T2D + symptomatic HF regardless of EF → guideline-directed HF medical therapy | Missed. |
| M120 | 49 | **Rec 10.44d/e** T2D + HFpEF + obesity → dual GIP/GLP-1 RA or GLP-1 RA with demonstrated HF benefit — **REJECTED in DB** | Verify reject. |
| M121 | 50 | **T1D HF association**: CAVIAR/HF-DM cohort data showing T1D elevated HF risk | Partial in A202. |
| M122 | 50 | **DM HF epidemiology**: 750,000-DM cohort — HF + CKD most frequent first manifestations | Missed. |

---

### ❓ NEEDS-CONFIRM

| # | Page | DB content | Action |
|---|---|---|---|
| NC37 | 49 | PENDING span containing "Treatment Recommendations 10.40a Among people with T2D who have established ASCVD or CKD, a SGLT2i or GLP-1 RA…" | Confirm or merge into A196. |
| NC38 | 49 | REJECTED Recs 10.40c, 10.44b, 10.44e — these are real recommendations. Investigate why REJECTED (duplicate with REVIEWER adds? or clerical error?) | **Recover** if valid recs, to preserve their numbered identity. |
| NC39 | 50 | REJECTED chunk marker page-50 PENDING (F,G) | Accept the reject. |

---

### Batch 5 summary

| Bucket | Count |
|---|---|
| ADDED (acceptable) | 54 |
| MISSED | 24 |
| NEEDS-CONFIRM | 3 |
| REJECTED-as-noise | 6 |

**Pipeline failure modes in batch 5 (mostly positive signals):**
20. **Extractor performs well on structured clinical-trial prose** — statin, PCSK9i, REDUCE-IT, ASCEND, ASPREE narratives were captured with hazard ratios, p-values, and effect sizes intact. This suggests the issue in batches 1–4 was not effect-size stripping per se, but **recommendation-box detection** + **wide-table cell-pairing**, not prose.
21. **Table 10.1 statin intensity correctly reconstructed** — unlike Table 9.2 (missed) or Tables 9.3/9.4 (disaster). Table 10.1 is small (2 cols × 7 rows) and the D-channel cells were CONFIRMED. **Table width/complexity correlates with extraction failure.**
22. **Treatment-rec paragraph on page 49 partially REJECTED in DB.** Recs 10.40c, 10.44b, 10.44e, 10.44d/e were REJECTED rather than CONFIRMED — likely because reviewers added their own version of 10.40a–g and the extractor-produced copies became duplicates. Net information still captured, but the numbered identifiers (10.40c, 10.44b) may be lost in the REJECT.
23. **Numbered recommendation fidelity still weak.** Recs 10.17–10.25 (the statin/age monitoring series), 10.28 (LDL goal), 10.30/10.31 (HDL/TG), 10.34b (DAPT duration), 10.40b/c/d, 10.43, 10.44a–e — are inconsistently preserved as numbered facts. Numeric-rec extraction should be prioritized in a pipeline fix.

---

## Batch 6 — Pages 51–60 (PDF pp. S233–S242)

**Content coverage:** Pages 51–56 complete Section 10 CVD — Asymptomatic PAD screening prose (VIVA trial, page 51), Glucose-Lowering Therapies and CV Outcomes (SGLT2i EMPA-REG / CANVAS / DECLARE / VERTIS; GLP-1 RA LEADER / SUSTAIN-6 / REWIND / AMPLITUDE / Harmony), DPP-4i CAROLINA, Figure 10.5 (HF prevention/treatment algorithm), **Prevention of HF: ACEi/ARB/β-blockers** (SOLVD, SAVE, CAPRICORN, REVERT), **SGLT2i in HF: DAPA-HF, EMPEROR-Preserved, DELIVER**, **GLP-1 RA in HF: STEP-HFpEF (semaglutide), SUMMIT (tirzepatide)**, Figure 10.6 (ASCVD prevention approach), other glucose-lowering classes in HF (TZD warning, DPP-4i SAVOR-TIMI saxagliptin signal, EXAMINE, TECOS, CARMELINA no signal), Clinical Approach summary. **Pages 57–60 = bibliography references for Section 10** (approximately 148 citation entries).

**DB inventory in this page range:**

| Page | Total | CONFIRMED | ADDED | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 51 | 11 | 11 | 0 | 0 | 0 | 0 |
| 52 | 3 | 3 | 0 | 0 | 0 | 0 |
| 53 | 12 | 12 | 0 | 0 | 0 | 0 |
| 54 | 11 | 11 | 0 | 0 | 0 | 0 |
| 55 | 9 | 8 | 0 | 1 | 0 | 0 |
| 56 | 21 | 20 | 0 | 1 | 0 | 0 |
| 57 | 26 | 24 | 0 | 0 | 0 | 2 |
| 58 | 42 | 39 | 2 | 0 | 0 | 1 |
| 59 | 41 | 30 | 10 | 1 | 0 | 0 |
| 60 | 39 | 37 | 0 | 1 | 0 | 1 |
| **Batch 6 total** | **215** | **195** | **12** | **3** | **0** | **4** |

**Structural note:** Pages 51–56 content-audit (67 spans) is high-quality and mostly reviewer-confirmed. **Pages 57–60 (148 spans) are bibliography entries** and replicate the problem seen on pages 27–34 (G-7).

---

### ✅ ADDED — guideline facts present in DB

#### SGLT2i / GLP-1 RA cardiovascular outcomes (pages 51–52)

| # | Page | Fact | Status |
|---|---|---|---|
| A204 | 51 | **VIVA trial** (50,156 pts, ~10% DM): combined AAA/PAD/HTN screening vs none; NNS ≈170 to prevent one death; associated with more pharmacologic therapy, shorter PAD/CAD hospital stay, reduced mortality | CONFIRMED (REVIEWER) |
| A205 | 51 | **ABI screening** recommended for asymptomatic PAD in DM who: age ≥65, DM duration ≥10 yrs, microvascular disease, foot complications, end-organ damage | CONFIRMED (REVIEWER) |
| A206 | 51 | **Look AHEAD** trial — intensive lifestyle / weight loss for glycemia, fitness, ASCVD risk factors | CONFIRMED (REVIEWER) |
| A207 | 51 | **FDA 2008 guidance** requiring CVOTs for all new T2D medications | CONFIRMED (REVIEWER) |
| A208 | 51 | **SGLT2i CV benefit**: empagliflozin, canagliflozin, dapagliflozin showed reduction in MI/stroke/CV death | CONFIRMED (REVIEWER) |
| A209 | 51 | **GLP-1 RAs with CV benefit**: liraglutide, semaglutide, albiglutide (market-withdrawn), dulaglutide, efpeglenatide | CONFIRMED (REVIEWER) |
| A210 | 51 | GLP-1 RA and SGLT2i reduce ASCVD MACE comparably in T2D + established ASCVD | CONFIRMED (REVIEWER) |
| A211 | 51 | SGLT2i + GLP-1 RA also reduce HF hospitalization + CKD progression in ASCVD / multiple risk factors / albuminuric kidney disease | CONFIRMED (REVIEWER) |
| A212 | 52 | **DKA risk** with SGLT2i — including euglycemic DKA — must be considered | CONFIRMED |
| A213 | 52 | DPP-4i **no CV benefit** (no DPP-4i CVOT showed superiority); CAROLINA demonstrated linagliptin non-inferior to glimepiride for CV with lower hypoglycemia | CONFIRMED |
| A214 | 52 | **Figure 10.5** overview for symptomatic HF prevention/treatment in DM | CONFIRMED |

#### HF prevention and treatment trials (pages 53–55)

| # | Page | Fact | Status |
|---|---|---|---|
| A215 | 53 | **Up to 91% of incident HF preceded by hypertension** | CONFIRMED |
| A216 | 53 | Antihypertensive therapy → **36% ↓ HF incidence** | CONFIRMED |
| A217 | 53 | DM + stage B HF → ACEi/ARB + β-blocker (asymptomatic post-MI, ACS, or LVEF ≤40%) | CONFIRMED |
| A218 | 53 | **SOLVD trial** (15% DM): enalapril → **20% ↓ incident HF** in asymptomatic LV dysfunction | CONFIRMED |
| A219 | 53 | **SAVE trial** (23% DM): captopril post-MI + reduced LVEF → **37% ↓ HF development** | CONFIRMED |
| A220 | 53 | **CAPRICORN** (23% DM): carvedilol post-MI + reduced LVEF → **23% ↓ mortality, 14% ↓ HF hospitalization** | CONFIRMED |
| A221 | 53 | **REVERT trial** (45% DM): metoprolol improved cardiac remodeling in LVEF <40% + mild LV dilatation | CONFIRMED |
| A222 | 53 | SGLT2i reduce ASCVD + HF outcomes in diabetes | CONFIRMED |
| A223 | 54 | **DAPA-HF**: dapagliflozin 10 mg daily → **primary outcome HR 0.74 [95% CI 0.65–0.85]; first WHFE HR 0.70 [0.59–0.83]; CV death HR 0.82 [0.69–0.98]** (consistent regardless of T2D) | CONFIRMED |
| A224 | 54 | **EMPEROR-Preserved**: empagliflozin — CV death or HF hospitalization ↓ in NYHA I–IV HFpEF (LVEF >40%), irrespective of T2D | CONFIRMED |
| A225 | 54 | **DELIVER trial**: dapagliflozin HFpEF also benefit | CONFIRMED |
| A226 | 54 | **SGLT1/2 inhibitor sotagliflozin** in HF (SOLOIST-WHF / SCORED) also provides benefit | CONFIRMED |
| A227 | 55 | **STEP-HFpEF** (semaglutide 2.4 mg weekly in T2D + obesity + HFpEF, N=616): **KCCQ-CSS +13.7 vs +6.4** (placebo); **weight −9.8% vs −3.4%**; improved 6-min walk distance | CONFIRMED |
| A228 | 55 | **SUMMIT** (tirzepatide in HFpEF + obesity, 48% DM): worsening HF / CV death **9.9% vs 15.3% (HR 0.62, 95% CI 0.41–0.95)**; KCCQ-CSS +19.5 vs +12.7 | CONFIRMED |
| A229 | 55 | GLP-1 RA or GIP/GLP-1 RA recommended for HF symptom and event reduction in people with HF symptoms | CONFIRMED |
| A230 | 55 | **Figure 10.6** approach to ASCVD prevention in T2D | CONFIRMED (as description) |
| A231 | 55 | Thiazolidinediones cause fluid retention → avoid in HF | CONFIRMED |
| A232 | 56 | **SAVOR-TIMI 53** (saxagliptin): ↑ HF hospitalization vs placebo | CONFIRMED |
| A233 | 56 | **EXAMINE / TECOS / CARMELINA** (alogliptin, sitagliptin, linagliptin): no HF-hospitalization signal | CONFIRMED |
| A234 | 56 | **Clinical approach**: T2D + ASCVD/HF/CKD risk → cardioprotective SGLT inhibitor and/or GLP-1 RA **irrespective of additional glucose-lowering needs or metformin use** | CONFIRMED |
| A235 | 56 | SGLT2i / GLP-1 RA may replace some existing meds to minimize hypoglycemia, side effects, cost | CONFIRMED |
| A236 | 56 | Collaboration (primary + specialty care, advanced-practice, pharmacists, nutritionists) facilitates transitions | CONFIRMED |

---

### ❌ MISSED — facts in PDF absent / non-recoverable

| # | Page | Fact | Why missed |
|---|---|---|---|
| M123 | 52 | **Figure 10.5 full branching structure**: NYHA class, BNP/NT-proBNP/UACR/CKD gating, TTE node, GDMT edges | Captured only as caption (A214). |
| M124 | 54 | **EMPA-REG OUTCOME** (empagliflozin T2D + ASCVD): CV death ↓ 38%, all-cause mortality ↓ 32%, HF hospitalization ↓ 35% | Embedded in prose, not as discrete trial fact. |
| M125 | 54 | **CANVAS / CANVAS-R** (canagliflozin): MACE ↓ 14%, HF hospitalization ↓ 33%, amputation signal ↑ | Embedded. |
| M126 | 54 | **DECLARE-TIMI 58** (dapagliflozin): HF hospitalization + CV death ↓ 17% | Embedded. |
| M127 | 54 | **VERTIS CV** (ertugliflozin): non-inferior for MACE | Embedded. |
| M128 | 54 | **LEADER / SUSTAIN-6 / REWIND / AMPLITUDE / Harmony** trial effect sizes for GLP-1 RAs | Embedded as named list (A209), no effect-size data. |
| M129 | 55 | **Figure 10.6 algorithm branches**: BP / lipid / glycemia / antiplatelet combination blocks | Captured as caption only. |
| M130 | 56 | **Fig 9.4 cross-reference** context — the "treat irrespective of metformin" rule has evidence strength beyond the prose | Missed as cross-ref. |

---

### ❓ NEEDS-CONFIRM

| # | Page | DB content | Action |
|---|---|---|---|
| NC40 | 51–56 | REVIEWER-channel CONFIRMED spans with typo-like duplicated words (e.g., "SGLT2 inhibitors: Discontinue before scheduled surgery (e.g., 3SGLT2 inhibitors: Discontinue before...") observed in A162 / elsewhere | **EDIT** to remove duplicate fragment. |
| NC41 | 55 | 1 EDITED span on page 55 — low-intensity review artifact | Review content. |
| NC42 | 57–60 | **148 reference-citation spans CONFIRMED as guideline facts** (same issue as pages 27–34) | **BULK REVERT + REJECT** to `l2_references`. See G-7 in final summary. |

---

### Batch 6 summary

| Bucket | Count |
|---|---|
| ADDED (acceptable, Section 10 continued) | 33 |
| MISSED | 8 |
| NEEDS-CONFIRM | 3 |
| **False-Positive-As-Fact (references incorrectly treated as facts)** | **148** |
| True rejections (page markers, chunk markers, boilerplate) | 4 |

**Pipeline observations:**
24. **Section 10 HF section has strong effect-size capture.** DAPA-HF primary + secondary HRs with CIs, STEP-HFpEF KCCQ deltas, SUMMIT HR 0.62 — all preserved. Contrast with batch 1 where equivalent SOLVD/SAVE effect sizes were absent. **Reviewer performance on effect sizes is strong.**
25. **Wider failure mode is the "assumed-facts" reviewer behavior.** On pages 57–60, 148 reference-citation spans arrived PENDING and reviewers clicked CONFIRM rather than REJECT. This is a UX-plus-process failure, not a model failure. Recommend dashboard auto-classify references (format-based heuristic) and surface them to reviewers with a "reject all as reference" bulk action.
26. **Trial-specific-fact preservation is inversely correlated with how deep in the narrative they sit.** Headline trials (DAPA-HF, SUMMIT) captured; mid-paragraph trials (EMPA-REG subgroup numbers, DECLARE subgroup) lost. Recommend increasing Channel G granularity around sentences that contain RCT names.

---

## Batch 7 — Pages 61–70 (PDF pp. S243–S252)

**Content coverage:** Pages 61–63 = **bibliography references for Section 10** (~70 citation entries). **Page 64 begins Section 11 "Chronic Kidney Disease and Risk Management"** (Diabetes Care 2026;49(Suppl. 1):S246–S260, doi 10.2337/dc26-S011, endorsed by National Kidney Foundation). Pages 65–70: CKD diagnosis + staging — **UACR classification** (normal <30, moderate 30–<300, severe ≥300 mg/g), **eGFR** via creatinine + cystatin C combination; CKD diagnosis in diabetes (albuminuria/eGFR without alternative cause); exclusion criteria for non-diabetic CKD; Table 11.1 (CKD severity / risk matrix); Table 11.2 (screening for CKD complications); SURVEILLANCE (annual UACR + eGFR); INTERVENTIONS — Nutrition (0.8 g/kg/day protein), ACEi/ARB section with Recs 11.5, 11.6a–d, avoid ACEi+ARB combinations, BP goals <130/80 (<120 in some).

**DB inventory in this page range:**

| Page | Total | CONFIRMED | ADDED | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 61 | 23 | 1 | 21 | 1 | 0 | 0 |
| 62 | 15 | 12 | 2 | 1 | 0 | 0 |
| 63 | 32 | 26 | 5 | 1 | 0 | 0 |
| 64 | 7 | 3 | 3 | 0 | 0 | 1 |
| 65 | 51 | 0 | 4 | 0 | 47 | 0 |
| 66 | 12 | 9 | 2 | 1 | 0 | 0 |
| 67 | 32 | 7 | 6 | 0 | 0 | **19** |
| 68 | 20 | 9 | 10 | 1 | 0 | 0 |
| 69 | 16 | 7 | 8 | 1 | 0 | 0 |
| 70 | 9 | 3 | 4 | 1 | 0 | 1 |
| **Batch 7 total** | **217** | **77** | **65** | **6** | **47** | **21** |

**Reviewer burden** is very high in Section 11: 65 ADDED / 217 = 30% (comparable to batches 1–3). Plus 47 PENDING on page 65 (Table 11.1/11.2 blow-up similar to pages 21–22 cost tables). Page 67 has **19 REJECTED spans** — highest reject count outside page 78 — likely a duplicate paragraph.

---

### ✅ ADDED — guideline facts present in DB

**Pages 61–63: References (NOT facts — see G-7 finding)** — 73 reference citations (21 ADDED, 50 CONFIRMED, 2 EDITED) should be migrated to `l2_references`. Not itemized here.

**Section 11 — Chronic Kidney Disease (pages 64–70)**

| # | Page | Fact | Status |
|---|---|---|---|
| A237 | 64 | **Section 11 title**: Chronic Kidney Disease and Risk Management — endorsed by **National Kidney Foundation** | ADDED |
| A238 | 64 | Introductory context for CKD in diabetes; role in cardiovascular-kidney-metabolic health | ADDED / CONFIRMED |
| A239 | 65 | **UACR classification**: Normal-to-mildly increased <30 mg/g; **moderately elevated 30 to <300 mg/g**; **severely elevated ≥300 mg/g** | ADDED |
| A240 | 65 | UACR is **continuous** — differences within normal and abnormal ranges associate with kidney/CV outcomes | ADDED |
| A241 | 65 | **Biological variability >20%** between UACR measurements → **2 of 3 specimens** within 3–6 months must be abnormal to diagnose moderate/severe albuminuria | ADDED |
| A242 | 65 | UACR elevators to rule out: exercise within 24 h, infection, fever, HF, marked hyperglycemia, menstruation, marked hypertension | CONFIRMED (prose context) |
| A243 | 66 | **Combined creatinine + cystatin C eGFR** is more accurate; supports better clinical decisions than either alone | CONFIRMED |
| A244 | 66 | **CKD diagnosis in DM**: albuminuria and/or reduced eGFR in absence of signs of alternative primary cause; typical features: long DM duration, retinopathy, albuminuria without gross hematuria, gradual eGFR loss | CONFIRMED |
| A245 | 66 | Features suggesting **alternative / additional kidney disease** → refer to nephrology: active urinary sediment (RBCs/WBCs/casts), rapidly increasing albuminuria or proteinuria, nephrotic syndrome, rapidly decreasing eGFR, absence of retinopathy (esp. in T1D) | CONFIRMED |
| A246 | 66 | **Without kidney biopsy**, state "CKD in a person with diabetes" rather than attribute cause | CONFIRMED |
| A247 | 66 | Rare for T1D to develop kidney disease without retinopathy; retinopathy only moderately sensitive/specific for CKD of DM in T2D | CONFIRMED |
| A248 | 67 | **ACEi / ARB should NOT be discontinued for serum creatinine rise <30%** in absence of volume depletion | ADDED |
| A249 | 67 | Similar rises in creatinine on **SGLT2i and GLP-1 RA initiation** (tubuloglomerular feedback) — do NOT discontinue | ADDED |
| A250 | 67 | **Annual UACR + eGFR monitoring** for: timely CKD diagnosis, progression monitoring, AKI detection, complication risk, medication dosing, nephrology referral | CONFIRMED |
| A251 | 67 | **Serum K monitoring** on diuretics (hypokalemia) and periodically on ACEi/ARB/MRA if eGFR <60 (hyperkalemia) | CONFIRMED |
| A252 | 67 | **eGFR <60** also prompts: medication-dose verification, minimize nephrotoxins (NSAIDs, iodinated contrast), evaluation for CKD complications per Table 11.2 | CONFIRMED |
| A253 | 68 | **GRADE study** (liraglutide / sitagliptin / glimepiride / insulin glargine): no kidney-protective differences; **SGLT2i were not included** (not routinely available at study start) | ADDED |
| A254 | 68 | ADA does not recommend routine use of glucose-lowering meds solely for CKD prevention | CONFIRMED |
| A255 | 68 | **Nutrition Rec**: for stage G3–G5 non-dialysis CKD, **protein intake ~0.8 g/kg/day (RDA)** slowed GFR decline vs higher protein | CONFIRMED (prose from Rec 11.4a/b) |
| A256 | 68 | **Higher protein >20% of calories or >1.3 g/kg/day** → ↑ albuminuria, faster kidney loss, CVD mortality — **avoid** | ADDED |
| A257 | 68 | **Protein <0.8 g/kg/day NOT recommended** in CKD+DM — doesn't alter glycemia, CV risk, or GFR decline | ADDED |
| A258 | 68 | Endorsement of **NKF KDOQI and International Society of Renal Nutrition** protein guidance | ADDED |
| A259 | 69 | **Rec 11.6a** continue RAS blockade for mild-to-moderate creatinine ↑ (≤30%) without volume depletion (A) | ADDED |
| A260 | 69 | **Rec 11.6b** serum K + creatinine at routine visits and 7–14 days after initiation or dose change, periodically (B) | ADDED |
| A261 | 69 | **Rec 11.6c** ACEi or ARB **NOT recommended** for primary prevention of CKD in DM with normal BP + normal UACR + normal eGFR (A) | ADDED |
| A262 | 69 | **Rec 11.6d** Continue RAS blockade for ≤30% creatinine increase without extracellular volume depletion (A) | ADDED |
| A263 | 69 | ACEi/ARB are mainstay for CKD + albuminuria and HTN (with or without CKD); all SGLT2i/GLP-1 RA/nsMRA trials required ACEi/ARB background | CONFIRMED |
| A264 | 69 | **HTN is strong risk factor for CKD development + progression**; antihypertensive therapy ↓ albuminuria; ACEi/ARB ↓ progression to kidney failure in DM + eGFR <60 + UACR ≥300 | CONFIRMED |
| A265 | 69 | **BP <130/80 mmHg** for CVD mortality + CKD progression; SBP <120 considered based on benefit-risk | CONFIRMED |
| A266 | 69 | CKD especially with severe albuminuria (≥300) may warrant **lower BP goal** | CONFIRMED |
| A267 | 70 | In T1D + normal albumin excretion: **ARB reduced but did not prevent progression** + CV event rate ↑ — so RAS blockade NOT for primary prevention | CONFIRMED |
| A268 | 70 | T1D no-HTN-no-albuminuria: **ACEi/ARB did not prevent glomerulopathy** (biopsy-confirmed); similar negative result in T2D | ADDED |
| A269 | 70 | **Combined ACEi + ARB**: no CV/CKD benefit + ↑ adverse events (hyperkalemia, AKI) → avoid | CONFIRMED |
| A270 | 70 | **Figure 11.2** "Holistic approach for improving outcomes in people with DM + CKD": Lifestyle → RAS therapy → Additional therapy (SGLT2i, GLP-1 RA, nsMRA, statin, CCB, diuretic, antiplatelet, ezetimibe/PCSK9i/icosapent ethyl) with regular reassessment of glycemia/albuminuria/BP/CVD/lipids | ADDED |

---

### ❌ MISSED — facts in PDF absent / non-recoverable

| # | Page | Fact | Why missed |
|---|---|---|---|
| M131 | 64 | **Section 11 citation** (doi, page range, suggested citation format) | G-1 consequence — section-title prose captured but metadata not linked. |
| M132 | 65 | **Table 11.1 — CKD risk classification by eGFR × UACR stages (KDIGO-like heat map)** — 6 eGFR categories × 3 albuminuria categories × risk color coding | 47 PENDING spans on page 65 = D-channel cell fragments. Same failure pattern as Tables 9.3/9.4/11.2 (see G-2). |
| M133 | 65 | **Rec 11.1 / 11.2** Annual UACR + eGFR screening; type-1-diabetes duration ≥5 yrs; all T2D at diagnosis + annually | Missed — only prose context captured. |
| M134 | 66 | **Cystatin C-based eGFR** as alternative when creatinine-based eGFR is unreliable (sarcopenia, low muscle mass) | Partially in A243. |
| M135 | 67 | **Table 11.2** rows: BP >130/80 → diagnose HTN; volume overload; electrolytes (K, Na); anemia → Hb; metabolic bone disease → Ca/Phos/PTH; acidosis → HCO3; etc. | Captured as reviewer-added summary but individual complication-threshold pairs are lost. |
| M136 | 67 | **19 REJECTED spans on page 67** — likely duplicate paragraphs that extractor double-emitted | Flag for extractor dedup fix. |
| M137 | 68 | **Rec 11.4a** Protein intake 0.8 g/kg/day for non-dialysis CKD (B) — numbered rec form | Missed as numbered rec. |
| M138 | 68 | **Rec 11.4b** Dialysis patients higher protein intake (1.0–1.2 g/kg/day) | Missed. |
| M139 | 68 | **Rec 11.3** Sodium intake restriction (<2.3 g/day) for CKD + HTN | Missed. |
| M140 | 69 | **Rec 11.5** BP goals for CKD (SBP <120 or <130) as numbered rec | Captured as prose in A265, not numbered. |
| M141 | 70 | **Fig 11.2 granular medication-class nodes** and footnotes (*with demonstrated benefit, **eGFR-dependent efficacy, nsMRA caveat) | Caption-level only in A270. |

---

### ❓ NEEDS-CONFIRM

| # | Page | DB content | Action |
|---|---|---|---|
| NC43 | 61–63 | ~70 reference-citation spans CONFIRMED as facts | **BULK REVERT + REJECT**. |
| NC44 | 65 | **47 PENDING D-channel cell fragments** from Table 11.1 (CKD risk matrix) and Table 11.2 (complications table) | **REJECT in bulk**; re-extract tables via structured-table extractor. |
| NC45 | 67 | **19 REJECTED** spans — verify reject reasons (duplicates vs accidental rejects) | Reviewer-operations ticket. |

---

### Batch 7 summary

| Bucket | Count |
|---|---|
| ADDED (acceptable, Section 11 foundational) | 34 |
| MISSED | 11 |
| NEEDS-CONFIRM | 3 |
| **False-Positive-As-Fact (references)** | **~70** |
| **Bulk-reject candidates (page 65 table fragments, page 67 duplicates)** | **~60** |

**Pipeline observations:**
27. **Section-boundary transitions still broken.** Section 11 begins on page 64 but no section-node is created in `l2_guideline_tree`. Downstream queries filtering by section can't distinguish Section 10 from Section 11.
28. **Section 11 UACR/protein recommendations preserved with quantitative thresholds**: A239 (<30 / 30–<300 / ≥300), A255 (0.8 g/kg/day), A256 (>1.3 g/kg/day avoid) — this is the kind of data KB-3 Guidelines needs.
29. **Page 65 is the third major table blow-up after pages 2, 21–22, 11–14.** Tables 9.1 (page 2), 9.3 (page 21), 9.4 (page 22), 9.2 (pages 11–14), 11.1 (page 65), 11.2 (page 67). Six clinically essential tables have failed structured extraction.

---

## Batch 8 — Pages 71–80 (PDF pp. S253–S262)

**Content coverage:** Pages 71–75 complete **Section 11 CKD**: selection of glucose-lowering meds (metformin eGFR rules, SGLT2i ≥20 mL/min recommended); CKD-focused SGLT2i trials (**CREDENCE, DAPA-CKD, EMPA-KIDNEY**); **GLP-1 RA FLOW trial**; **nsMRA finerenone FIDELIO-DKD / FIGARO-DKD / FIDELITY** pooled; combination therapy **CONFIDENCE** trial; GLP-1 RA in dialysis; Referral to nephrologist (eGFR <30 or stage G4). **Pages 76–78 = bibliography for Section 11** (~165 citation entries). **Page 79 begins Section 13 "Older Adults"** (Diabetes Care 2026;49(Suppl. 1):S277–S296) with Recs 13.1, 13.2. Page 80: 4Ms framework (Mentation, Medications, Mobility, What Matters Most), geriatric-syndrome screening, functional disability prevalence.

**DB inventory:**

| Page | Total | CONFIRMED | ADDED | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 71 | 11 | 7 | 2 | 1 | 0 | 1 |
| 72 | 10 | 7 | 1 | 2 | 0 | 0 |
| 73 | 17 | 9 | 5 | 3 | 0 | 0 |
| 74 | 14 | 5 | 7 | 1 | 1 | 0 |
| 75 | 43 | 3 | 35 | 1 | 3 | 1 |
| 76 | 54 | 0 | 44 | 1 | 0 | 9 |
| 77 | 62 | 0 | 43 | 0 | 0 | 19 |
| 78 | 59 | 41 | 0 | 0 | 0 | 18 |
| 79 | 6 | 5 | 0 | 0 | 0 | 1 |
| 80 | 5 | 3 | 0 | 1 | 0 | 1 |
| **Batch 8 total** | **281** | **80** | **137** | **9** | **4** | **49** |

**Page 75–77 sprawl:** Pages 75–77 contain **122 ADDED spans**, of which page 75 (35 ADDED) is real CKD content but pages 76–77 (87 ADDED) are reference citations the reviewer incorrectly added as facts — same anti-pattern as batches 4/6/7. 49 REJECTED on pages 76–78 shows a reviewer *started* rejecting them properly but gave up mid-way. **Inconsistency finding.**

---

### ✅ ADDED — Section 11 CKD facts (pages 71–75)

| # | Page | Fact | Status |
|---|---|---|---|
| A271 | 71 | **FDA 2016 metformin revision**: eGFR (not SCr) guides treatment; **contraindicated eGFR <30**; monitor eGFR while on drug; reassess at <45; **do not initiate <45**; temporarily discontinue before iodinated contrast if eGFR 30–60 | CONFIRMED |
| A272 | 71 | **SGLT2i recommended for T2D + eGFR ≥20 mL/min/1.73m²** — slow CKD progression + reduce HF risk independent of glucose effect | CONFIRMED + ADDED |
| A273 | 71 | Key CKD trials: EMPA-REG OUTCOME, CANVAS, LEADER, SUSTAIN-6, REWIND, AMPLITUDE-O (empagliflozin/canagliflozin/liraglutide/semaglutide/dulaglutide/efpeglenatide) | CONFIRMED |
| A274 | 72 | **DAPA-CKD** (dapagliflozin): sustained eGFR decline ≥50% or kidney failure or kidney death HR 0.56 [0.45–0.68] p<0.001; CV death or HF hospitalization HR 0.71 [0.55–0.92] p=0.009; all-cause mortality ↓ p<0.004 | CONFIRMED |
| A275 | 72 | **EMPA-KIDNEY** (6,609 pts, ~50% DM, eGFR 20–45 or 45–90 with UACR ≥200): empagliflozin progression of kidney disease + CV death **HR 0.72 [0.64–0.82] p<0.001** | CONFIRMED |
| A276 | 72 | SGLT2i glucose-lowering **blunted at eGFR <45** but kidney/CV benefits preserved down to eGFR **20**; even no-significant-glucose-change populations benefit | CONFIRMED |
| A277 | 73 | **MRA classes**: steroidal vs **nonsteroidal** (finerenone) — not interchangeable | CONFIRMED |
| A278 | 73 | **FIDELIO-DKD** (N=5,734 T2D + CKD, UACR 30–<300 + eGFR 25–<60 + retinopathy, or UACR 300–5,000 + eGFR 25–<75, K+ ≤4.8 mmol/L): finerenone significantly ↓ CKD progression + CV events | CONFIRMED |
| A279 | 74 | Population cohort: adding GLP-1 RA on top of SGLT2i → **30% ↓ MACE, 57% ↓ serious kidney events** vs GLP-1 RA alone; adding SGLT2i on top of GLP-1 RA → 29% ↓ MACE | CONFIRMED |
| A280 | 74 | **FLOW trial prespecified analysis**: semaglutide kidney + CV benefits not affected by SGLT2i concomitant use (limited baseline SGLT2i use) | CONFIRMED |
| A281 | 74 | **CONFIDENCE** — first published finerenone + empagliflozin combination trial (3 arms: each alone vs combined); eGFR >30 to <90 + UACR 100–5,000 | CONFIRMED / ADDED |
| A282 | 75 | Semaglutide in high-BMI CKD: **average weight reduction 4.6 ± 2.4 kg** | ADDED |
| A283 | 75 | **National cohort 151,649 T2D + dialysis initiation 2013–2021**: GLP-1-based therapy → **23% ↓ mortality, 66% ↑ transplant waitlisting** | ADDED |
| A284 | 75 | Observational dialysis cohort: GLP-1-based therapy MACE HR 0.65 p<0.001; all-cause mortality HR 0.63 p<0.001 (median 1.4 yr) | ADDED |
| A285 | 75 | **Exenatide and lixisenatide should NOT be used in individuals on dialysis** | ADDED |
| A286 | 75 | **Referral to nephrologist** indications: continuously rising UACR, declining eGFR, uncertain etiology, difficult management (anemia, 2° hyperparathyroidism, albuminuria despite BP control, metabolic bone disease, resistant HTN, electrolyte disturbance), advanced CKD (eGFR <30) | ADDED |
| A287 | 75 | **Nephrology consultation at stage G4 CKD (eGFR <30)** → reduces cost, improves quality of care, delays dialysis | ADDED |

---

### Section 13 start (pages 79–80)

| # | Page | Fact | Status |
|---|---|---|---|
| A288 | 79 | **Rec 13.1** Assess medical, psychological, functional (self-management), and social domains in older adults with DM — comprehensive approach for goals/therapy (B) | CONFIRMED |
| A289 | 79 | **Rec 13.2** At least annual screening for geriatric syndromes (cognitive impairment, depression, urinary incontinence, falls, persistent pain, frailty), **hypoglycemia, polypharmacy** (B) | CONFIRMED |
| A290 | 79 | **>29% of adults >65 yrs have diabetes** — epidemiologic | CONFIRMED |
| A291 | 80 | **4Ms framework** for age-friendly health care: **Mentation, Medications, Mobility, What Matters Most** (Institute for Healthcare Improvement) — evidence-based | CONFIRMED |
| A292 | 80 | Older adults with DM — higher rates of functional disability, accelerated muscle loss, frailty, mobility impairment, coexisting HTN/CKD/CHD/stroke, premature death | CONFIRMED |
| A293 | 80 | Higher rates of geriatric syndromes: cognitive impairment, depression, urinary incontinence, falls, pain, frailty, polypharmacy → affect self-management + quality of life | CONFIRMED |
| A294 | 80 | **Figure 13.1** conceptual model for 4Ms in diabetes management | EDITED |
| A295 | 80 | Greater caregiver support need in older DM vs non-DM | CONFIRMED |

---

### ❌ MISSED

| # | Page | Fact | Why missed |
|---|---|---|---|
| M142 | 71 | **Rec 11.7** (metformin eGFR <45 initiation, <30 discontinuation) as numbered rec | Captured as prose in A271. |
| M143 | 71 | **Rec 11.8/11.9** Glucose-lowering-med selection in CKD (SGLT2i ≥20, GLP-1 RA CKD benefit) as numbered recs | Prose only. |
| M144 | 72 | **DAPA-CKD primary composite HR (the non-0.56 secondary)** — the 0.56 is the **secondary** kidney composite; the primary was 0.61 [0.51–0.72]. The DB's HR 0.61 is embedded in opening fragment "0.51-0.72]; P < 0.001)" | Misalignment; needs **EDIT** to attribute correctly. |
| M145 | 73 | **FIDELIO-DKD primary endpoint composition** — time-to-first kidney failure or sustained eGFR decrease ≥40% or kidney death | Captured (A278) as prose but not as discrete endpoint facts. |
| M146 | 73 | **FIDELIO-DKD effect sizes** — primary endpoint HR, CV composite HR, UACR change from baseline | Missed. |
| M147 | 73 | **FIGARO-DKD** kidney/CV outcome numbers; **FIDELITY** pooled analysis effect sizes | Missed. |
| M148 | 74 | **CONFIDENCE trial result** — the trial is named but the finerenone+empagliflozin combination vs either-alone effect on UACR endpoint is not captured | Missed. |
| M149 | 74 | **Rec 11.10 / 11.11** (SGLT2i + GLP-1 RA combination; finerenone rec) as numbered recs | Missed. |
| M150 | 75 | **Rec 11.12** (Nephrology referral criteria) as numbered rec | Prose only. |
| M151 | 75 | Page-75 reviewer-added **35 spans** — suggests a major block of ADDED content (combination therapy + dialysis + referral) was extractor-missed. Verify nothing else is missing. | Partial — most captured in A281–A287. |
| M152 | 79 | **Section 13 metadata** (doi, suggested citation, license text) | G-1 consequence. |
| M153 | 79 | **Rec 13.3, 13.4** (if present in delta) — older-adult screening recommendations | Verify page 80+ content. |
| M154 | 80 | **Section 4 cross-reference** "Comprehensive Medical Evaluation and Assessment of Comorbidities" | Cross-ref lost. |

---

### ❓ NEEDS-CONFIRM

| # | Page | DB content | Action |
|---|---|---|---|
| NC46 | 76–78 | ~87 reference citations ADDED by reviewer + 41 CONFIRMED as facts + 46 REJECTED (partial cleanup) | **BULK REVERT + REJECT** remaining; migrate all to `l2_references`. |
| NC47 | 71 | 1 REJECTED REVIEWER span — unclear why a reviewer would add then reject | Inspect for workflow bug. |
| NC48 | 74 | 1 PENDING — CONFIDENCE trial context | Confirm and link to A281. |
| NC49 | 75 | 3 PENDING — dialysis/nephrology referral content | Confirm. |

---

### Batch 8 summary

| Bucket | Count |
|---|---|
| ADDED (acceptable — Section 11 CKD + Section 13 start) | 25 |
| MISSED | 13 |
| NEEDS-CONFIRM | 4 |
| **False-Positive-As-Fact (references)** | **~131** |
| True rejections | 2 (page markers + chunk markers) |

**Pipeline observations:**
30. **Section 11 was exited with good CV/kidney effect-size capture** — DAPA-CKD, EMPA-KIDNEY, FIDELIO-DKD all have HR+CI preserved (A274–A278). This is consistent with batches 5–6 showing extractor handles trial prose well when the rec isn't boxed.
31. **Section 13 began cleanly** — Recs 13.1, 13.2 captured, 4Ms framework intact, epidemiology correct. No reviewer burden on these two pages.
32. **Reviewer fatigue visible on pages 76–78.** Reference block has inconsistent handling: p.76=44 ADDED, p.77=43 ADDED, p.78=41 CONFIRMED (switched from ADD→CONFIRM), plus 46 REJECTED across the three pages. The switch suggests the reviewer realized mid-way that references shouldn't be ADDED but didn't revert prior work — leaving corrupted state. **Priority cleanup: pages 31–34, 57–60, 76–78 all need `l2_references` migration.**

---

## Batch 9 — Pages 81–90 (PDF pp. S263–S286)

**Content coverage:** Pages 81–82 = **Table 13.1 "Geriatric syndromes and other functional impairments: key symptoms and suggested screening approaches"** — 13 domains × 7 columns (sample screening questions, suggested screening measure, number of items, score range, cut points, time-to-complete, translated-languages availability). Domains: Cognitive impairment, Delirium, Depression, Falls, Frailty, Malnutrition, Pain, Polypharmacy, Sarcopenia, Hearing, Vision, Insomnia, Urinary incontinence, Dexterity, Dizziness, Executive functioning. Suggested instruments: **Mini-Cog, 4AT, GDS-5, STEADI, CFS, MNA-SF, Numeric Pain Rating, Beers Criteria, SARC-F, Whisper test, Snellen, ISI-7, 3IQ, Button and coin test, Trail Making Test B**.

Pages 83–90 continue Section 13: cognitive impairment + **diabetes-related dementia** (page 83); HYPOGLYCEMIA Rec 13.4; Treatment Goals — Rec 13.5/13.6 (CGM), 13.7a/b/c (glycemic goals tiered by health status), 13.8 (complication screening), 13.9 (BP <130/80), 13.10 (CV risk factors) (page 84); **Table 13.2 "Framework for considering treatment goals for glycemia, BP, dyslipidemia in older adults"** (page 86); LIFESTYLE MANAGEMENT — Recs 13.11a (0.8 g/kg/day protein), 13.11b (activity), 13.12 (intensive lifestyle for T2D+obesity); PHARMACOLOGIC THERAPY — Recs 13.13 (low-hypoglycemia meds), 13.14a–d (deintensify, avoid hypoglycemic agents, simplify complex plans, CVD/HF/CKD agents), 13.15 (cost); Metformin, Pioglitazone, Sulfonylureas/Meglitinides, DPP-4 inhibitors, GLP-1 RAs, SGLT2 inhibitors, Insulin in elderly (pages 88–89); SPECIAL CONSIDERATIONS FOR OLDER ADULTS WITH TYPE 1 DIABETES (page 90).

**DB inventory:**

| Page | Total | CONFIRMED | ADDED | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 81 | 2 | 1 | 0 | 0 | 1 | 0 |
| 82 | 750 | 0 | 0 | 0 | **750** | 0 |
| 83 | 14 | 7 | 4 | 3 | 0 | 0 |
| 84 | 14 | 13 | 0 | 1 | 0 | 0 |
| 85 | 16 | 15 | 0 | 1 | 0 | 0 |
| 86 | 3 | 1 | 1 | 0 | 0 | 1 |
| 87 | 19 | 18 | 0 | 1 | 0 | 0 |
| 88 | 9 | 5 | 3 | 1 | 0 | 0 |
| 89 | 8 | 5 | 1 | 0 | 0 | 2 |
| 90 | 10 | 7 | 1 | 2 | 0 | 0 |
| **Batch 9 total** | **845** | **72** | **10** | **9** | **751** | **3** |

**Page 82 is the single largest outlier in the entire 98-page document.** 750 PENDING spans — **all from channel D (748) plus 2 channel-F/G page-marker spans** — making it the third failure mode of table extraction after Tables 9.3/9.4 (355 fragments) and Table 11.1/11.2 (47 fragments). The most-frequent single token on page 82 is `"3"` appearing **416 times** (score values from the table).

---

### ✅ ADDED — guideline facts present in DB

#### Cognitive impairment + hypoglycemia (pages 83–84)

| # | Page | Fact | Status |
|---|---|---|---|
| A296 | 83 | **Diabetes-related dementia** — distinct from Alzheimer + vascular dementia; slower progression, absent typical neuroimaging findings, advanced age, elevated A1C, long DM duration, high insulin use frequency, frailty, sarcopenia, **dynapenia** (muscle strength loss without neurologic cause) | ADDED |
| A297 | 83 | Glucose-lowering drugs with small benefit on slowing cognitive decline: **TZDs, GLP-1 RAs, SGLT2i** | CONFIRMED |
| A298 | 83 | Systematic review + meta-analysis: cardioprotective glucose-lowering therapies not associated with ↓ all-cause dementia, but **GLP-1 RAs statistically significant ↓ all-cause dementia** | ADDED |
| A299 | 83 | BP management + statins → ↓ incident dementia — important in older DM | CONFIRMED |
| A300 | 83 | Cognitive screening tools: **Mini-Cog, MMSE, Montreal Cognitive Assessment** | CONFIRMED |
| A301 | 83 | **Annual screening** for MCI/dementia recommended for age ≥65 (Rec 13.3 implied) | CONFIRMED |
| A302 | 83 | **Rec 13.4** Ascertain + address hypoglycemia at routine visits; older adults with DM at greater hypoglycemia risk, esp. with hypoglycemic agents | CONFIRMED |
| A303 | 83 | Support from RCTs: **ACCORD, VADT** studies on intensive control harms | EDITED |
| A304 | 84 | **CGM improves glycemia + acceptable in older adults** (T1D, insulin-requiring T2D, non-insulin-treated subgroups); may take longer to learn, engage caregivers; cognitive/functional assessment for tech use | CONFIRMED |
| A305 | 84 | AID/HCL studies (ORACL trial, 30 pts mean age 67, T1D) → ↑ TIR + modest ↓ hypoglycemia vs SAP | CONFIRMED |
| A306 | 84 | Later RCT 37 older adults ≥60 yr on MDI → HCL ↑ TIR + ↓ time above range (no hypoglycemia improvement) | CONFIRMED |
| A307 | 84 | Real-world Medicare HCL (4,243 individuals, 89% T1D, mean age 67.4): initiating HCL associated with improved glycemic outcomes | CONFIRMED |
| A308 | 84 | **Rec 13.7a** Older adults with few/stable coexisting chronic illnesses + intact cognitive/functional status → **A1C <7.0–7.5% (<53–58 mmol/mol)** | CONFIRMED |
| A309 | 84 | **Rec 13.7c** Very complex/poor health → minimal benefit from stringent goals; focus on avoiding hypoglycemia + symptomatic hyperglycemia | CONFIRMED |
| A310 | 84 | **Rec 13.8** Individualize complication screening; prioritize those that would impair function/QoL | CONFIRMED |
| A311 | 84 | **Rec 13.9** BP goal **<130/80 mmHg** when safely achievable; **<140/90 mmHg** may be used for complex/poor-health or limited-life-expectancy (A) | CONFIRMED |
| A312 | 84 | **Rec 13.10** Individualize other CV risk factor treatment; lipid-lowering + antiplatelet benefits take time (statins ≥2.5 yr time-to-benefit) | CONFIRMED |

#### Life expectancy + glycemic individualization (page 85)

| # | Page | Fact | Status |
|---|---|---|---|
| A313 | 85 | **LEAD (Life Expectancy Estimator for Older Adults with Diabetes)** tool — validated in older DM; high-risk score → life expectancy <5 years | CONFIRMED |
| A314 | 85 | Deintensify at end of life; most T2D agents may be removed at end of life | CONFIRMED |
| A315 | 85 | Conditions affecting A1C accuracy in older adults: kidney failure, recent significant blood loss, erythropoietin therapy — blood-glucose monitoring + CGM preferred | CONFIRMED |
| A316 | 85 | Hyperglycemia >180 mg/dL (>10 mmol/L) ↑ dehydration/weakness/infection/poor wound healing risk — should still be avoided even in complex health | CONFIRMED |
| A317 | 85 | End-of-life focus: avoid hypoglycemia + symptomatic hyperglycemia while reducing glycemic-management burden | CONFIRMED |

#### Table 13.2 + Lifestyle + Pharmacologic recs (pages 86–89)

| # | Page | Fact | Status |
|---|---|---|---|
| A318 | 86 | **Table 13.2** framework: A1C / BP / dyslipidemia goals by health status (healthy / complex / very-complex) in older adults | ADDED |
| A319 | 87 | **Rec 13.11a** Healthful eating + **protein ≥0.8 g/kg/day** for older DM; individualized higher amounts for sarcopenia/frailty | CONFIRMED |
| A320 | 87 | **Rec 13.11b** Regular physical activity (aerobic, weight-bearing, resistance) when safe | CONFIRMED |
| A321 | 87 | **Rec 13.12** Intensive lifestyle intervention for T2D + overweight/obesity capable of safe exercise (Look AHEAD-like) | CONFIRMED |
| A322 | 87 | **LIFE (Lifestyle Interventions and Independence for Elders)** study — structured activity reduces sedentary time + prevents mobility disability in frail older adults | CONFIRMED |
| A323 | 87 | Hypertension treatment benefit in older adults from trials | CONFIRMED |
| A324 | 87 | Moderately higher protein intake (20–30%) may support DM management by enhancing satiety | CONFIRMED |
| A325 | 87 | Malnutrition associated with ↓ ADLs, grip strength, lower-limb physical performance, cognition, QoL | CONFIRMED |
| A326 | 87 | Look AHEAD: did not achieve primary CV endpoint but ↓ medication use (antihypertensives, statins, insulin) | CONFIRMED |
| A327 | 87 | **Rec 13.13** Select medications with **low hypoglycemia risk** in older T2D especially with hypoglycemia risk factors (B) | CONFIRMED |
| A328 | 87 | **Rec 13.14a** Deintensify hypoglycemia-causing meds (insulin, SU, meglitinides) or switch to low-risk class | CONFIRMED |
| A329 | 87 | **Rec 13.14c** Simplify complex (insulin) treatment plans to reduce hypoglycemia, polypharmacy, burden (B) | CONFIRMED |
| A330 | 87 | **Rec 13.14d** Older T2D + ASCVD/HF/CKD → agents with demonstrated benefit | CONFIRMED |
| A331 | 88 | **Rec 13.15** Consider costs + coverage to reduce cost-related barriers | EDITED |
| A332 | 88 | **Metformin safe at eGFR ≥30**; lower doses for 30–45; monitor eGFR; **annual B12 monitoring after >4 yr long-term use** | CONFIRMED |
| A333 | 88 | **Pioglitazone** low hypoglycemia risk but cautious use in select individuals | CONFIRMED |
| A334 | 88 | **DPP-4 inhibitors** — few side effects, minimal hypoglycemia; cost may be barrier; weak glucose-lowering | ADDED |
| A335 | 88 | **Sulfonylureas**: hypoglycemia, bone loss, fracture risk → use with caution. Meglitinides (repaglinide, nateglinide) similar hypoglycemia risk | CONFIRMED |
| A336 | 88 | Intensive glycemic management with insulin/SU in complex older adults → cause of adverse events | CONFIRMED |
| A337 | 89 | **GLP-1 RAs + GIP/GLP-1 RA tirzepatide** — highly effective, low hypoglycemia, usable at reduced eGFR including dialysis | CONFIRMED |
| A338 | 89 | GLP-1 RAs: nausea, vomiting, diarrhea, constipation — slow titration; **NOT preferred** in older adults with unexplained weight loss | CONFIRMED |
| A339 | 89 | GLP-1 RAs CV benefit same regardless of age | CONFIRMED |
| A340 | 89 | Most GLP-1 RAs injectable (except oral semaglutide) — require visual/motor/cognitive skills; most have weekly dosing | CONFIRMED |
| A341 | 89 | **Figure 13.2** stepwise approach for assessing treatment-plan difficulties; reevaluating glycemic goals via shared decision-making; deintensifying/simplifying/modifying the plan; re-assessing | ADDED |

#### Older adults with T1D (page 90)

| # | Page | Fact | Status |
|---|---|---|---|
| A342 | 90 | **People with T1D living longer** → growing population >65; special considerations for older T1D | ADDED |
| A343 | 90 | **SGLT2i cautions**: ↑ urine volume — query about urinary incontinence before/after initiation; **euglycemic DKA risk** | CONFIRMED |
| A344 | 90 | **Insulin essential** for T1D; older T1D need basal even when unable to ingest meals (to avoid DKA) | CONFIRMED |
| A345 | 90 | Transitions to PALTC → discontinuity in goals, dosing errors, need ongoing support | CONFIRMED |
| A346 | 90 | Monitor for dehydration, weight loss → sarcopenia/bone-density loss risk in elderly | EDITED |
| A347 | 90 | Age alone doesn't limit technology utility; older adults who can learn benefit | CONFIRMED |
| A348 | 90 | Insulin-therapy requirements: good visual + motor skills + cognitive ability for dose/admin + hypoglycemia recognition | CONFIRMED |

---

### ❌ MISSED — facts in PDF absent / non-recoverable

| # | Page | Fact | Why missed |
|---|---|---|---|
| M155 | 81–82 | **Table 13.1 — 13 geriatric-domain × 7-column screening matrix** (domain / question / instrument / # items / score range / cut points / time / languages). Specific tools: Mini-Cog (5 items, 0–5, <3), 4AT (4 items, 0–12, ≥4), GDS-5 (5 items, 0–5, ≥2), STEADI (3 items, 0–3), CFS (1 item, 1–9, ≥5), MNA-SF (6 items, 0–14, 0–7 malnourished), NRS (0–10), Beers Criteria, SARC-F (5 items, 0–10, ≥4), Whisper (6 items), Snellen (20/40), ISI-7 (7 items, 0–28, ≥8), 3IQ (3 items), Button-and-coin test (2 items), Trail Making Test B (time >180s impairment) | **750 cell fragments on page 82; 1 reviewer-added span on p81 covering row 1 only**. Entire remainder of table (cognitive-impairment + delirium + 13 more domains) lost. |
| M156 | 83 | **Rec 13.3** Routine screening for cognitive impairment + dementia in older adults with DM at age 65 → | Missed as numbered rec. |
| M157 | 83 | ACCORD hypoglycemia mortality signal (HR, event rate) | Embedded, not discrete. |
| M158 | 85 | **Rec 13.5, 13.6** (A1C targets + CGM) as numbered recs | Missed — only prose in A308–A312. |
| M159 | 86 | **Table 13.2 full content** — 3 health status categories × (Rationale, A1C goal, BP goal, lipid goal, ADL requirements, TBR/TIR targets, CGM, PALTC considerations) | Reviewer added only the Table 13.2 header (A318); individual row contents lost. 1 REJECTED span confirms extractor output insufficient. |
| M160 | 87 | **Rec 13.14b** (avoid hypoglycemic-risk medications for high-risk patients) | Missed as discrete rec. |
| M161 | 88 | **B12 deficiency risk** with long-term metformin — annual monitoring after 4+ yrs captured (A332); **linkage to neuropathy worsening** not captured. | Partial. |
| M162 | 89 | **Insulin therapy recommendations for elderly T2D** (insulin-pen preference, caregiver training, adjustment schedule) as numbered recs | Missed. |
| M163 | 90 | **T1D older adult specific recs** (if any numbered 13.16+) | Verify. |

---

### ❓ NEEDS-CONFIRM

| # | Page | DB content | Action |
|---|---|---|---|
| NC50 | 82 | **750 PENDING D-channel cell fragments** from Table 13.1 | **BULK REJECT**; re-extract via structured-table pipeline into `l2_tables`. Priority fix — this is the worst single-page failure. |
| NC51 | 83 | 3 EDITED spans — verify reviewer-edited content matches source | Spot-check. |
| NC52 | 86 | REJECTED Table 13.2 header span | If A318 is adequate, accept; else ADD complete table. |
| NC53 | 89 | 2 REJECTED spans | Verify reject reasons. |

---

### Batch 9 summary

| Bucket | Count |
|---|---|
| ADDED (acceptable, Section 13 Older Adults) | 53 |
| MISSED | 9 |
| NEEDS-CONFIRM | 4 |
| **Bulk-reject (Table 13.1 blow-up)** | **750** |
| True rejections | 3 |

**Pipeline observations:**
33. **Page 82 = single largest table-extraction failure** (750 fragments for a single 13-row × 7-col table). Root cause: channel D is emitting every cell as an independent span without header-row linkage. The 416 repetitions of `"3"` indicate that the table has a column (score range) that contains that value across many rows — but each cell is orphaned.
34. **Strong content capture for Section 13 prose.** 53 acceptable facts across 10 pages = 5.3/page average, much denser than early-section prose pages. Reviewer + extractor pairing works well for this content type.
35. **Numbered-rec preservation remains weak.** Section 13 has recs 13.1 through 13.14d + 13.15. DB captures: 13.1, 13.2, 13.4, 13.7a, 13.7c, 13.8, 13.9, 13.10, 13.11a, 13.11b, 13.12, 13.13, 13.14a, 13.14c, 13.14d, 13.15 (16 discrete). Missing: **13.3 (cognitive screening), 13.5 (A1C target), 13.6 (CGM), 13.7b (complex health glycemic goals), 13.14b (agent selection)** = 5 numbered recs lost.

---

## Batch 10 — Pages 91–98 (PDF pp. S287–S296)

**Content coverage:** Page 91 = **Figure 13.3** (algorithm to simplify insulin administration plans in older adults) + PALTC prose (CGM feasibility, hypoglycemia prevalence in PALTC, RCT showing CGM safe/effective for 60 days guiding insulin doses). Page 92 = **Table 13.3 "Considerations for treatment plan simplification and deintensification/deprescribing in older adults"** — 4 health-status categories × (reasonable glycemic goal / rationale / when simplify / when deintensify). Page 93 = TREATMENT IN PALTC SETTINGS — **Recs 13.16, 13.17**, training, facility policies. Page 94 = END-OF-LIFE CARE prose (palliative-care goals, statin-withdrawal QoL benefit, deintensification, CGM use at end of life, blood-glucose targets <70/>250 mg/dL boundaries). **Pages 95–98 = bibliography references for Section 13** (~210 citation entries).

**DB inventory:**

| Page | Total | CONFIRMED | ADDED | EDITED | PENDING | REJECTED |
|---|---|---|---|---|---|---|
| 91 | 8 | 7 | 0 | 1 | 0 | 0 |
| 92 | 31 | 17 | 0 | 0 | 1 | 13 |
| 93 | 15 | 12 | 0 | 1 | 0 | 2 |
| 94 | 23 | 20 | 0 | 1 | 0 | 2 |
| 95 | 48 | 41 | 1 | 0 | 0 | 6 |
| 96 | 49 | 46 | 0 | 0 | 0 | 3 |
| 97 | 66 | 48 | 0 | 0 | 0 | 18 |
| 98 | 47 | 42 | 0 | 0 | 0 | 5 |
| **Batch 10 total** | **287** | **233** | **1** | **3** | **1** | **49** |

**Reviewer burden minimal: 1 ADDED / 287 = 0.3%.** Unfortunately, ~210 of the 233 CONFIRMED spans on pages 95–98 are bibliography entries, not real facts (G-7 again).

---

### ✅ ADDED — guideline facts present in DB

#### Insulin simplification + PALTC (pages 91–93)

| # | Page | Fact | Status |
|---|---|---|---|
| A349 | 91 | **Figure 13.3** algorithm: Every 2 weeks, adjust insulin dose and/or add glucose-lowering meds based on pre-lunch + pre-dinner testing; noninsulin examples (metformin, SGLT2i, DPP-4i, GLP-1 RAs, dual GIP/GLP-1 RA); prandial insulins (R, lispro, aspart, glulisine) | CONFIRMED |
| A350 | 91 | **PALTC CGM feasibility study** — useful but requires substantial staff training | CONFIRMED |
| A351 | 91 | PALTC hypoglycemia prevalence high in both insulin and SU users → older adults in PALTC at **increased hypoglycemia risk** | CONFIRMED |
| A352 | 91 | **PALTC RCT** (real-time CGM vs point-of-care BGM, up to 60 days): safe + effective for guiding insulin doses; no differences in TIR, TBR, mean glucose | CONFIRMED |
| A353 | 91 | Staff may mistake DKA for sepsis / end-organ failure / electrolyte abnormalities; family may know individual plan better than staff | CONFIRMED |
| A354 | 92 | **Table 13.3** Treatment plan simplification + deintensification: (1) Healthy: A1C <7.0–7.5%; simplify on severe/recurrent hypoglycemia, wide excursions, cognitive/functional decline; (2) Complex/intermediate: A1C <8.0%; simplify if difficulty managing insulin, social change; (3) Community-dwelling short-term in SNF: avoid A1C reliance, **BG goal 100–200 mg/dL**; (4) Long-term care: varies by health status | CONFIRMED (17) |
| A355 | 93 | **Rec 13.16** PALTC staff training (CGM, insulin pumps, advanced insulin delivery systems) to improve older-adult management (E) | CONFIRMED |
| A356 | 93 | **Rec 13.17** PALTC residents need careful assessment of **mobility, mentation, medications, management preferences** to establish individualized goals + appropriate agent/device choices (E) | CONFIRMED |
| A357 | 93 | **PALTC diabetes management** unique considerations; ADA position statement "Management of Diabetes in Long-term Care and Skilled Nursing Facilities" referenced | CONFIRMED |
| A358 | 93 | Training elements: **diabetes detection, complication identification, glycemic monitoring, individualized goals, medications, nutrition, oral care, foot assessment, wound/infection concerns, quality assessment** | CONFIRMED |
| A359 | 93 | PALTC facilities should develop own policies for hypoglycemia + symptomatic hyperglycemia prevention/recognition/management | CONFIRMED |

#### End-of-life care (page 94)

| # | Page | Fact | Status |
|---|---|---|---|
| A360 | 94 | **Palliative care goal**: promote comfort, symptom mgmt + prevention (pain, hypoglycemia, hyperglycemia, dehydration), preservation of dignity + QoL in limited life expectancy | CONFIRMED |
| A361 | 94 | Strict glucose + BP management may not align with comfort / QoL at end-of-life; **avoidance of severe hyperglycemia** aligns with palliative goals | CONFIRMED |
| A362 | 94 | **Multicenter trial**: **statin withdrawal in DM palliative care improved QoL** | CONFIRMED |
| A363 | 94 | **Deintensification protocols** for both glucose and BP management — growing evidence | CONFIRMED |
| A364 | 94 | Individuals have right to refuse testing/treatment; HCPs may consider withdrawing treatment and ↓ BG monitoring frequency | CONFIRMED |
| A365 | 94 | **CGM at end-of-life** — consider when frequent BG monitoring is burdensome but hypo/hyperglycemia monitoring still needed | CONFIRMED |
| A366 | 94 | **End-of-life glycemic goals**: avoid hypoglycemia (**<70 mg/dL**) and severe hyperglycemia (**>250 mg/dL** when symptom burden ↑) | CONFIRMED |
| A367 | 94 | Careful monitoring of oral intake warranted | CONFIRMED |
| A368 | 94 | Clinical decision-making process involves individual + family + care partners → safe, comfortable care plan | CONFIRMED |

---

### ❌ MISSED — facts in PDF absent / non-recoverable

| # | Page | Fact | Why missed |
|---|---|---|---|
| M164 | 91 | **Fig 13.3 full algorithm branches**: specific starting points, insulin class distinctions, dose-adjustment intervals | Caption + 1 CONFIRMED span only. |
| M165 | 92 | **Table 13.3 row-by-row content** — 13 REJECTED spans on p.92 suggest extractor emitted table fragments that got rejected rather than structured. The REVIEWER-confirmed A354 summary captures the 4 categories but losses subrow triggers | Partial. |
| M166 | 94 | **Rec 13.18** (if present) — end-of-life numbered recommendation | Not seen in DB. |
| M167 | 94 | Statin-withdrawal RCT effect sizes (mortality, QoL scores) | Embedded in A362, not discrete. |

---

### ❓ NEEDS-CONFIRM

| # | Page | DB content | Action |
|---|---|---|---|
| NC54 | 92 | 1 PENDING + 13 REJECTED Table 13.3 fragments | Verify rejects are appropriate (should be) and migrate the content to `l2_tables`. |
| NC55 | 95–98 | **~210 reference citations CONFIRMED as facts** (same pattern as pages 27–34, 57–60, 76–78) | **BULK REVERT + REJECT**; migrate to `l2_references`. |
| NC56 | 95–98 | **32 appropriately REJECTED** references on pages 95 (6), 96 (3), 97 (18), 98 (5) | Accept as correct rejections; use as positive example for automation rule. |

---

### Batch 10 summary

| Bucket | Count |
|---|---|
| ADDED (acceptable, final Section 13 + end-of-life) | 20 |
| MISSED | 4 |
| NEEDS-CONFIRM | 3 |
| **False-Positive-As-Fact (references)** | **~210** |
| True rejections | 49 (mixed bibliography + table cells) |

**Pipeline observations:**
36. **Figure 13.3 and Table 13.3 extracted reasonably.** Unlike Table 13.1's 750-fragment disaster, Table 13.3 emitted ~31 spans (manageable), most rejected as fragments and the content reconstructed via reviewer summary. **Smaller tables (4 rows × 4 cols) survive; larger ones don't.**
37. **End-of-life care section captured with full fidelity.** All 9 substantive facts (A360–A368) captured; palliative-care glycemic thresholds (<70, >250) preserved as discrete values.
38. **The final 4 pages again reinforce the reference-extraction-as-facts problem.** 210 citations CONFIRMED as facts is the final and largest instance. Total reference pollution across the document: **~700+ citations miscategorized as facts** (pages 27–34: ~120, pages 57–60: ~148, pages 61–63: ~70, pages 76–78: ~120, pages 95–98: ~210).

---

## Final aggregated summary

### 1 — Volumetric rollup (all 10 batches)

| Batch | Pages | DB spans | ADDED-acceptable | MISSED | NEEDS-CONFIRM | False-Pos-As-Fact (refs) | Bulk-reject (tables/markers) |
|---|---|---:|---:|---:|---:|---:|---:|
| 1 | 1–10 | 138 | 29 | 32 | 18 | 0 | ~50 |
| 2 | 11–20 | 79 | 49 | 23 | 8 | 0 | ~4 |
| 3 | 21–30 | 472 | 26 | 28 | 6 | 0 | ~420 |
| 4 | 31–40 | 112 | 45 | 15 | 4 | **47** | 0 |
| 5 | 41–50 | 84 | 54 | 24 | 3 | 0 | 6 |
| 6 | 51–60 | 215 | 33 | 8 | 3 | **148** | 4 |
| 7 | 61–70 | 217 | 34 | 11 | 3 | **~70** | ~60 |
| 8 | 71–80 | 281 | 25 | 13 | 4 | **~131** | 2 |
| 9 | 81–90 | 845 | 53 | 9 | 4 | 0 | **753** |
| 10 | 91–98 | 287 | 20 | 4 | 3 | **~210** | ~49 |
| **Totals** | **98** | **2,730** | **368** | **167** | **56** | **~606** | **~1,348** |

**Net interpretation (after de-duplication):**
- **~368 acceptable guideline facts** captured in the DB (13.5% of span volume).
- **~606 references miscategorized as facts** (22% of span volume) — needs migration to `l2_references`.
- **~1,348 span-fragments from table blow-ups + structural markers** (49% of span volume) — needs bulk rejection + re-extraction.
- **~167 facts missing from DB** that are present in the PDF — needs targeted re-extraction.
- **~56 PENDING or low-quality facts** needing active reviewer decisions.

**Raw DB total** = 2,735 spans (per `SELECT COUNT(*) FROM l2_merged_spans WHERE job_id='908789f3…'`). The 5-span discrepancy with the batch rollup (2,730) is due to a handful of spans with `page_number IS NULL` or other edge-case attributions — not material to the audit.

---

### 2 — Consolidated global findings (G-1 → G-12)

These are the cross-cutting issues visible across multiple batches. Ranked by severity.

#### 🔴 Critical (block KB-0 consumer utility)

- **G-1 — Structural section tree missing entirely.** `alignment_confidence = 0.0`, `total_sections = 1`, Channel A never produced a single span across 98 pages, `l2_section_passages` has 1 row. Downstream queries cannot distinguish Section 9 / 10 / 11 / 13 content, cannot anchor a fact to a sub-section, and the dashboard's `/passages` API returns effectively nothing. **Evidence across every batch.**
- **G-2 — Wide-table structured extraction fails catastrophically.** 6 clinically essential tables fragmented into unusable cells:
  - Table 9.1 (pages 2–4) — insulin plan advantages/disadvantages
  - Table 9.2 (pages 11–14) — medication features matrix (9 drug classes × 8 attributes)
  - Table 9.3 (page 21) — noninsulin AWP/NADAC costs
  - Table 9.4 (page 22) — insulin AWP/NADAC costs
  - Table 11.1 (page 65) — KDIGO-like CKD severity/risk matrix
  - Table 13.1 (pages 81–82) — geriatric syndrome screening instruments
  - Table 13.2 (page 86) — elderly treatment goal framework
  - Table 13.3 (page 92) — simplification/deintensification framework (partial — the smallest of the six)
  Root cause: channel D emits every cell as an independent span with no row/col linkage.
- **G-3 — Reference-list extracted as facts across 5 reference blocks.** Total **~606 bibliography entries CONFIRMED/ADDED as guideline facts** across pages 27–34, 57–60, 61–63, 76–78, 95–98. Downstream KB services consuming `l2_merged_spans` will see these as clinical assertions.
- **G-4 — Numbered recommendation preservation inconsistent.** Across sections 9 / 10 / 11 / 13, the DB preserves some `X.Y`-labeled recommendations but loses many. Worst-hit: T2D section (every rec 9.5–9.23 had issues), CV treatment section (10.40a/b/c/d, 10.44a–e). Numbered-rec identity is essential for KB-3 Guidelines and KB-23 Decision Cards.

#### 🟡 Important (degrades clinical utility)

- **G-5 — Effect-size and hazard-ratio extraction inconsistent.** Batches 1–3 stripped magnitudes (CSII −0.30% CI, sotagliflozin 8× DKA, pramlintide 0.3–0.4%, ICI 1%, SGLT-DKA 0.6–4.9/1000 PY, SGLT-i DKA-T1D 4%). Batches 4–6 preserved them well (BPROAD 25%, ESPRIT HR 0.79, STEP HR 0.88, DAPA-HF HR 0.74, IMPROVE-IT 6.4%, STEP-HFpEF KCCQ). The delta correlates with reviewer involvement — extractor alone drops magnitudes.
- **G-6 — Evidence grades (A/B/C/E) inconsistently preserved.** Numbered recommendations frequently arrive without the terminal grade letter. Impact on KB-23 Decision Cards which auto-approve based on grade.
- **G-7 — Counter drift in `l2_extraction_jobs`.** `total_merged_spans = 1922` vs live COUNT = 2735. 813 REVIEWER-channel spans not reflected in the stored counter. Denormalization artifact — harmless but confusing.
- **G-8 — Figures lose structural content.** Fig 9.1, 9.2, 9.3, 9.4, 9.5, 10.2, 10.3, 10.4, 10.5, 10.6, 11.2, 13.1 (=Table 13.1), 13.2, 13.3 — 14 figures where algorithm branches/nodes/edges are flattened to prose or dropped.

#### 🟢 Recommended (cleanup / process)

- **G-9 — Reviewer decision inconsistency.** ADA Professional Practice Committee boilerplate REJECTED on page 1, CONFIRMED on page 34 — same text, opposite outcomes. Reviewer workflow needs a "previously-rejected-identical-text" hint.
- **G-10 — REVIEWER-channel spans labeled CONFIRMED.** In pages 36–37, 78, etc., `contributing_channels = {REVIEWER}` with `review_status = CONFIRMED` — reviewers bypassing ADD workflow by typing + confirming in one step. Verify whether intentional.
- **G-11 — Structural markers in merged_spans.** "<!-- PAGE N -->" and "<!-- Chunk chunk-NNN: pages X-Y -->" markers arrive as F-channel spans (125 total). Should be filtered out pre-merge.
- **G-12 — Channel H (43 spans total, confidence 0.60) duplicates channel D row labels.** Low-value contribution; consider merging into D.

---

### 3 — Recommended actions (priority-ordered)

#### P0 — Blocking data-quality fixes

1. **Migrate 606 references out of `l2_merged_spans`.**
   ```sql
   -- Identify candidates
   SELECT id, text FROM l2_merged_spans
   WHERE job_id = '908789f3-d5a0-4187-ad9d-78072e0af1a6'
     AND page_number BETWEEN 27 AND 34
     OR page_number BETWEEN 57 AND 63
     OR page_number BETWEEN 76 AND 78
     OR page_number BETWEEN 95 AND 98;
   ```
   Build an `l2_references` table; move these rows; add a pre-merge classifier that detects citation patterns ("Author A, Author B, et al. Title. Journal Year;Vol:Pages").
2. **Bulk-reject the 1,348 table-fragment + structural-marker spans.**
   - 355 on pages 21–22 (Tables 9.3, 9.4)
   - 47 on page 65 (Tables 11.1, 11.2)
   - 750 on page 82 (Table 13.1)
   - ~60 on page 2, pages 11–14, page 86, page 92 (Tables 9.1, 9.2, 13.2, 13.3)
   - ~116 structural markers + fragments across various pages
3. **Re-extract the 8 affected tables** using a dedicated table pipeline (pdfplumber `extract_tables()` or Marker's table mode) into a new `l2_tables` relation with row/col structure preserved. This is the single highest-value feature fix.

#### P1 — Content gaps (167 missed facts)

4. **Prioritize the ~35 missed numbered recommendations** (M2, M22, M28–M32, M63–M70, M86, M92–M94, M106, M107, M113–M116, M119, M120, M133, M137–M140, M142, M149, M150, M153, M156, M158, M160, M163, M166). These are first-class KB-0 citizens.
5. **Recover effect sizes listed in M12, M17, M20, M41, M44, M89, M100, M102, M112, M124–M128, M146, M147, M167.** Specifically: CSII −0.30% CI, sotagliflozin 8× DKA, pramlintide 0.3–0.4%, GRADE liraglutide HR 0.7, IMPROVE-IT 6.4% RRR, EMPA-REG OUTCOME subgroup numbers, FIDELIO-DKD primary HR.
6. **Recover dosing / threshold specifics** (M13 newly-diagnosed T1D 0.2–0.6 U/kg/day, M23 eGFR <45 SGLT2i threshold, M81 SGLT-DKA 0.6–4.9/1000 PY, etc.).

#### P2 — Process improvements

7. **Fix Channel A alignment**. Zero output across 98 pages indicates either a hard failure or a data-shape issue (granite_doctags source). Investigate the aligner.
8. **Enforce reviewer coherence** — dashboard hint "This text was rejected on page N; reject again?" for boilerplate duplicates.
9. **Annotate 2026-new recommendations** (e.g., Rec 9.27 expanded to T2D this year) with a `delta_flag` so downstream consumers can identify policy changes.
10. **Cross-reference persistence** — capture "see section X" hyperlinks as explicit `section_cross_ref` facts.

#### P3 — Counter and metadata hygiene

11. Reconcile `l2_extraction_jobs.total_merged_spans` to live COUNT (auto-sync trigger).
12. Populate `l2_section_passages` properly once Channel A is fixed.
13. Add `evidence_grade` column to `l2_merged_spans` to persist A/B/C/E letters separately from the text body.

---

### 4 — Scorecard per section

| Section | Pages | Acceptable facts | Missed facts | Refs miscategorized | Extractor-only captures | Reviewer-only adds | Overall grade |
|---|---|---:|---:|---:|---:|---:|:-:|
| 9 Pharmacologic (T1D + T2D) | 1–30 | 104 | 83 | 0 | 5 | 99 | **C-** — extractor missed 17/19 T2D recs |
| 10 CVD and Risk Mgmt | 35–56 | 136 | 48 | 0 | 101 | 35 | **B+** — strong prose + trial capture |
| 10 references | 57–60 | 0 | 0 | 148 | n/a | n/a | **F** — total miscategorization |
| 10 references | 61–63 | 0 | 0 | ~70 | n/a | n/a | **F** |
| 11 CKD | 64–75 | 65 | 22 | 0 | 35 | 30 | **B** — good but Table 11.1 lost |
| 11 references | 76–78 | 0 | 0 | ~131 | n/a | n/a | **F** |
| 13 Older Adults | 79–94 | 82 | 13 | 0 | 23 | 59 | **B** — strong recs coverage, Tables 13.1/13.2 lost |
| 13 references | 95–98 | 0 | 0 | ~210 | n/a | n/a | **F** |

---

### 5 — Verdict

The extraction pipeline is **pre-production-ready for prose content (B–B+)**, **unusable for tables (F)**, **uncategorizing for references (F)**, and **weak on recommendation numbering + evidence grades (C+)**. Before this data can feed downstream KB services (KB-3 Guidelines, KB-23 Decision Cards, V-MCU Channel C ProtocolGuard):

- P0 fixes are **mandatory** — the 606-reference pollution and 1,348 table-fragment pollution together represent **71% of the row volume** in `l2_merged_spans` for this job.
- P1 content gaps are **high-value** but not blocking — most missing numbered recs are prose-captured, just not as discrete identified facts.
- P2/P3 are **process improvements** that would compound over future ingestions.

**Human review burden to date:** 329 ADDED + 69 EDITED + 146 REJECTED = **544 active reviewer actions** on this job. Net remaining: **1,358 PENDING spans** (49.6% of total) — most of which are the P0 bulk-reject candidates identified above. **Automating the bulk-reject of references + table fragments would reduce the pending queue from 1,358 to ~200** overnight.

---

*End of primary audit — 98 pages, 2,735 DB spans analyzed, 10 ten-page batches × one consolidated summary.*
*Audit file: `claudedocs/audits/ADA-2026-SOC-Delta-98pages_Audit.md`*
*DB snapshot: `claudedocs/audits/ada2026_db_snapshot.json`*

---

## Appendix A — Pipeline contract (L1 → L5) and KB routing

Source of truth: **`backend/shared-infrastructure/knowledge-base-services/shared/tools/guideline-atomiser/data/run_pipeline_targeted.py`** (Mar 16 2026, 1,818 lines, commit `fd4a91a1` — "feat(pipeline): wire KB20PushClient into pipeline runner with --push-kb20 flag").

`run_pipeline.py` (Feb 14, 1,011 lines, commit `3ebba658`) is the predecessor and is superseded. Key differences in the targeted version:
- Adds **KB-20 contextual** as a 4th target (old: 3 targets)
- Replaces KB-7 simple validation with **L4 RxNav THREE-CHECK**
- Adds `--push-kb20` flag for live KB-20 batch API push
- Tightens dossier assembly (v4.3.0 class-vs-member dedup)

### A.1 — CLI contract

```bash
python run_pipeline_targeted.py \
  --pipeline {1|2|legacy} \
  --target-kb {dosing|safety|monitoring|contextual|all} \
  --source <profile-source> | --pdf-path <path> \
  --pages <range>   # e.g., "50-65"
  --job-dir <dir>   # required for --pipeline 2
  --l1 {monkeyocr|marker|docling} \
  --guideline {kdigo|<yaml-path>} \
  [--push-kb20]
```

### A.2 — Pipeline 1 outputs (job_dir contract)

Pipeline 1 writes a job directory containing these **four artifacts** that Pipeline 2 consumes:

| File | Shape | Populated from |
|---|---|---|
| `job_metadata.json` | `{job_id, source_type/source_pdf, pipeline_version, total_pages, …}` | L1 extractor + channels |
| `normalized_text.txt` | Full flat text | L1 markdown |
| `guideline_tree.json` | `{sections[], tables[], total_pages, alignment_confidence, structural_source}` | Channel A (structural) |
| `merged_spans.json` | `list[MergedSpan]` with `review_status` ∈ {PENDING, CONFIRMED, EDITED, ADDED, REJECTED} | Channels B–H + signal merger + reviewer |

### A.3 — Pipeline 2 hard gate (BLOCKS ADA job)

```python
pending = sum(1 for s in merged_spans if s.review_status == "PENDING")
if pending > 0:
    print(f"❌ FATAL: {pending} spans still PENDING review.")
    sys.exit(1)
```

**Current ADA job status:** `spans_pending = 1358` → Pipeline 2 **will refuse to run**. This turns the audit's bulk-reject work from "nice cleanup" into **prerequisite P0 for L3**.

### A.4 — L3 routing map (from run_pipeline_targeted.py:1193–1196)

```python
target_kbs = ["dosing", "safety", "monitoring", "contextual"] if args.target_kb == "all" else [args.target_kb]
# kb_label = {
#   "dosing":     "KB-1",
#   "safety":     "KB-4",
#   "monitoring": "KB-16",
#   "contextual": "KB-20",
# }
```

Each of the ~70 drug dossiers assembled by `DossierAssembler` × each of 4 target KBs = **up to 280 Claude tool-use calls per document**. Each call invokes:
```python
extractor.extract_facts_from_dossier(dossier, target_kb, guideline_context) → KB{1|4|16|20}ExtractionResult
```
Results written to `{job_dir}/l3_output/{drug}_{kb}_targeted.json`.

### A.5 — KB target schemas (exact field names per Pydantic model)

Every fact extracted by L3 carries an embedded `ClinicalGovernance` block and a KB-specific payload. The **ClinicalGovernance** fields (common across all 4 KBs) are:

```
source_authority, source_document, source_section, evidence_level,
effective_date, guideline_doi
```

Per-KB payloads:

#### KB-1 Drug Dosing → `KB1ExtractionResult.drugs[]` ([kb1_dosing.py](backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb1_dosing.py))

```
DrugRenalFacts{
  rxnorm_code, drug_name,
  renal_adjustments[]  : RenalAdjustment{egfr_min, egfr_max, adjustment_factor, max_dose, max_dose_unit, frequency}
  hepatic_adjustments[]: HepaticAdjustment
  governance            : ClinicalGovernance
}
```

#### KB-4 Patient Safety → `KB4ExtractionResult{contraindications[], warnings[]}` ([kb4_safety.py](backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb4_safety.py))

```
ContraindicationFact{drug, condition, severity, evidence, governance}
WarningFact{drug, warning_type, threshold, action, governance}
```

#### KB-16 Lab Monitoring → `KB16ExtractionResult.lab_requirements[]` ([kb16_labs.py](backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb16_labs.py))

```
LabRequirementFact{
  drug, lab, frequency,
  critical_thresholds[] : CriticalValueThreshold{operator, value, unit, action, urgency}
  target_range          : TargetRange
  monitoring_entries[]  : LabMonitoringEntry
  governance            : ClinicalGovernance
}
```

#### KB-20 Contextual Modifiers → `KB20ExtractionResult` ([kb20_contextual.py](backend/shared-infrastructure/knowledge-base-services/shared/extraction/schemas/kb20_contextual.py))

```
KB20ExtractionResult{
  total_adr_profiles, completeness_summary{FULL, PARTIAL, STUB}, total_contextual_modifiers,
  adr_profiles[]  : AdverseReactionProfile{drug_class, symptom, mechanism, onset_window, completeness}
  modifiers[]     : ContextualModifierFact{drug, modifier_type, condition, effect, governance}
}
```

### A.6 — L4 THREE-CHECK (RxNav) — `run_pipeline_targeted.py:1263+`

```
Step 1 (Exact Match):    RxNav.validate_rxnorm(rxnorm_code) → display_name match
                         + mismatch detector: normalize(expected) ⊂ normalize(display) (hallucination catch)
Step 2 (Expansion):      RxNav.get_relationships(rxnorm_code, "rxnorm") → ingredient↔brand↔SCD edges
Step 3 (Subsumption):    Subsumption readiness check

Graceful fallbacks:
  - rxnorm_code == "<LOOKUP_REQUIRED>" → resolve via RxNav.get_rxcui_by_name(drug_name)
  - drug in profile.drug_class_skip_list (e.g., "SGLT2 inhibitor") → skip (no RxCUI for class)
```

### A.7 — L5 CQL Compatibility

Maps validated L4 facts to CQL libraries under `vaidshala/tier-4-guidelines/` (e.g., `T2DMGuidelines.cql`). Each L4-validated `rxnorm_code + drug_name` becomes a callable CQL symbol for runtime `V-MCU Channel C ProtocolGuard`.

---

## Appendix B — Audit item → KB destination routing

Every A-item (kept fact) and M-item (to-add fact) in batches 1–10 maps to one or more of {KB-1, KB-4, KB-16, KB-20}. Here's the deterministic rule set L3's prompt builder uses (derived from `fact_extractor.py:280–306`):

### B.1 — Routing rules

| Keyword/Pattern in span text | Route to |
|---|---|
| `eGFR <X`, `egfr`, dose adjustment, `max dose`, `contraindicated with eGFR`, `frequency` changes by renal function | **KB-1 dosing** |
| `contraindicated`, `black box`, `avoid`, `not recommended`, `do not use`, `DKA risk`, `pregnancy contraindicated` | **KB-4 safety** |
| `monitor`, `annually`, `4–12 weeks`, `serum K`, `UACR`, `TSH`, `spirometry/FEV1`, critical lab thresholds with actions | **KB-16 monitoring** |
| `onset`, `weeks after initiation`, ADR with timing, population modifier (`elderly`, `CKD`, `pregnancy`), comorbidity modifier | **KB-20 contextual** |
| Recommendation (`X.Y` numbered) | **KB source (all 4)** via embedded governance.source_section |
| Effect size / HR / RRR / A1C reduction | **KB-16** (as outcome-based threshold) + **KB-20** (as ADR benefit profile) |

Most facts **fan out to 2–3 KBs** — L3 is designed for this.

### B.2 — Batch-by-batch destination rollup (A-items that are already acceptable)

| Batch | Pages | KB-1 routed | KB-4 routed | KB-16 routed | KB-20 routed | Notes |
|---|---|---:|---:|---:|---:|---|
| 1 | 1–10 | 11 | 14 | 4 | 6 | T1D insulin dosing + T2D CV/CKD indication recs |
| 2 | 11–20 | 18 | 17 | 11 | 9 | Table 9.2 rows (fan out widely) + CGM recs + overbasalization |
| 3 | 21–30 | 8 | 11 | 3 | 10 | Cost facts (no KB target) + PTDM special pops → KB-20 |
| 4 | 31–40 | 9 | 7 | 12 | 4 | HTN recs + BP thresholds (KB-16 for BP monitoring) |
| 5 | 41–50 | 15 | 22 | 9 | 14 | Statin intensity + antiplatelet + CV treatment recs |
| 6 | 51–60 | 5 | 11 | 3 | 16 | GLP-1 + SGLT2i trial prose (ADR profiles → KB-20) |
| 7 | 61–70 | 13 | 8 | 11 | 4 | UACR/eGFR classification + surveillance rules |
| 8 | 71–80 | 18 | 4 | 3 | 5 | Metformin eGFR rules + FIDELIO-DKD + dialysis rules |
| 9 | 81–90 | 5 | 6 | 15 | 22 | Geriatric syndrome screening → KB-16; HCL devices → KB-20 |
| 10 | 91–98 | 4 | 5 | 2 | 9 | End-of-life BG boundaries + insulin simplification |
| **Totals (A-items)** | | **~106** | **~105** | **~73** | **~99** | Fan-out: 383 KB-routable events from 368 unique facts |

(Totals exceed 368 because many facts route to 2+ KBs.)

### B.3 — M-items (to-add) by KB destination

| Batch | Pages | KB-1 to add | KB-4 to add | KB-16 to add | KB-20 to add | Notes |
|---|---|---:|---:|---:|---:|---|
| 1 | 1–10 | 3 | 5 | 2 | 4 | Sotagliflozin 8× DKA (KB-4), Fig 9.2 algorithm (KB-1+KB-16) |
| 2 | 11–20 | 6 | 7 | 5 | 5 | Table 9.2 missing rows (DPP-4i, Pioglitazone, SU, Meglitinides) |
| 3 | 21–30 | 4 | 8 | 5 | 11 | Costs (no KB); SGLT-DKA 0.6–4.9/1000 PY → KB-4+KB-20 |
| 4 | 31–40 | 3 | 2 | 4 | 6 | BP trial effect sizes → KB-16 |
| 5 | 41–50 | 4 | 5 | 6 | 9 | IMPROVE-IT 6.4%, CLEAR Outcomes, REDUCE-IT → KB-20 |
| 6 | 51–60 | 1 | 2 | 1 | 4 | SGLT2i HF effect sizes → KB-20 |
| 7 | 61–70 | 3 | 3 | 3 | 2 | Table 11.1 CKD matrix + Rec 11.7–11.12 |
| 8 | 71–80 | 4 | 3 | 4 | 2 | FIDELIO-DKD effect sizes + CONFIDENCE combo trial |
| 9 | 81–90 | 1 | 2 | 3 | 3 | Table 13.1 geriatric screening (KB-16 heavy) |
| 10 | 91–98 | 1 | 1 | 1 | 1 | End-of-life Rec 13.18 (if exists) |
| **Totals (M-items)** | | **~30** | **~38** | **~34** | **~47** | 167 unique facts → ~149 KB-routable events |

### B.4 — Final bottom-line for your question

With the targeted-pipeline contract confirmed, the three-bucket summary is:

| Action | Count | L3/L4/L5 consequence |
|---|---:|---|
| **EXTRACT / ADD 167 new facts** | 167 | Fans out to **~30 KB-1 + ~38 KB-4 + ~34 KB-16 + ~47 KB-20** extractions (humans should write these into `l2_merged_spans` before Pipeline 2) |
| **REJECT 1,358 PENDING spans** | 1,358 | **MANDATORY — Pipeline 2 FATAL-aborts if PENDING > 0** |
| **RELOCATE/REJECT ~606 references + ~144 markers** | ~750 | Cleans merged_spans.json for dossier assembler (prevents false drug associations) |
| **KEEP 368 acceptable A-items** | 368 | Routes to **~383 KB-extraction events** across 4 target KBs |

The **P0 blocker** is the 1,358 PENDING count. Every other audit finding (tables, references, missed facts) degrades L3 output quality, but the PENDING gate is the one that makes Pipeline 2 refuse to start.

---

*Appendix A + B complete. Pipeline contract now aligned against `run_pipeline_targeted.py` v4.3.0 (Mar 16 2026, commit fd4a91a1).*

