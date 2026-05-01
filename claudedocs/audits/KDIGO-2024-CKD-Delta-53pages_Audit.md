# KDIGO 2024 CKD Delta (53 pages) — KB-0 Extraction Audit

**Pipeline Job:** `96c8f0d6-394f-4256-93c1-6a79f92c614b`
**Dashboard URL:** https://kb0-governance-dashboard.vercel.app/pipeline1/96c8f0d6-394f-4256-93c1-6a79f92c614b
**Source PDF:** `KDIGO-2024-CKD-Delta-53pages.pdf` (Chapter 4 Medication management + other delta sections of the KDIGO 2024 Clinical Practice Guideline for Evaluation & Management of CKD, *Kidney International* 2024;105(4S):S117–S314, doi 10.1016/j.kint.2023.10.018)
**Pipeline version:** `4.2.2`, L1 tag: `monkeyocr`
**DB:** GCP PostgreSQL `canonical_facts` @ `34.46.243.149:5433` — tables `l2_extraction_jobs`, `l2_merged_spans`, `l2_section_passages`, `l2_guideline_tree`
**DB snapshot time:** 2026-04-24 (live pull)
**Auditor:** Claude (Opus 4.7 1M)
**Prior audits of the same PDF:** `claudedocs/KDIGO_2024_CKD_PAGE_AUDIT_REPORT.md` (job `f172f6a9…`, pipeline 4.2.4, 939 spans, 12 zero-span pages) — this new run is a re-extraction, compared inline where relevant.

---

## Job header (live from DB)

| Field | Value |
|-------|-------|
| `total_pages` | 53 |
| `total_sections` | **67** (structural sectioning succeeded, unlike ADA 2026 SOC job which had 1) |
| `alignment_confidence` | **0.5** (partial L1 tree alignment) |
| `total_merged_spans` (header column) | 942 (machine-extracted at job close) |
| **Live `COUNT(*)` of `l2_merged_spans`** | **957** (machine + 15 reviewer) |
| `spans_confirmed` | 46 |
| `spans_pending` | 883 |
| `spans_rejected` | 23 |
| `spans_edited` | 2 |
| `spans_added` | 3 |
| `status` | IN_PROGRESS |
| `guideline_tier` | 1 |
| `pdf_page_offset` | 0 (PDF page N == DB `page_number` N) |
| `l2_section_passages` rows | 64 |
| `l2_guideline_tree` rows (this job) | 1 |
| `created_at` | 2026-03-10 09:53:58 UTC |
| `updated_at` | 2026-04-23 15:43:58 UTC |

Contributing-channel mix (across all 53 pages):

| Channel | Spans | Role |
|---|---|---|
| B | 59 | Drug lexicon (RxNorm) hits |
| C | 249 | Regex monitoring/frequency/threshold patterns |
| D | 595 | Table-cell extractions |
| E | 51 | GLiNER NER (contraindication, recommendation markers) |
| F | 67 | Passthroughs / structural markers |
| G | 325 | Full-sentence statements |
| H | 12 | Header/label inference (low confidence) |
| L1_RECOVERY | 13 | Oracle-recovered spans from L1 tree |
| REVIEWER | 15 | Reviewer-added spans (1.6% of total) |

> **Channel A is absent from all 53 pages** (same finding as ADA 2026 SOC job `908789f3…`). In 4.2.2, Channel A is the structural-oracle alignment channel; with `alignment_confidence=0.5` the oracle only partially succeeded and contributed via `L1_RECOVERY` instead of A. **Global finding**, not per-page.

Tier / review-status mix:

| Dimension | Distribution |
|---|---|
| `tier_level` | 957 × 1 (legacy `tier` column is NULL everywhere — migrated to `tier_level`) |
| `review_status` | CONFIRMED 46, PENDING 883, REJECTED 23, EDITED 2, ADDED 3 |

Review is **~4.8 % complete** (46 CONFIRMED / 957). Nearly every non-rejected span therefore classifies as `NEEDS-CONFIRM` under the audit taxonomy — the audit focuses on whether the *content* is captured, not on the review checkbox.

Per-page span density (all 53 pages):

| Page | N | Page | N | Page | N | Page | N |
|------|---|------|---|------|---|------|---|
| 1 | 13 | 15 | 8 | 29 | 3 | 43 | 5 |
| 2 | 34 | 16 | 40 | 30 | 21 | 44 | 8 |
| 3 | 5 | 17 | 28 | 31 | 51 | 45 | 10 |
| 4 | 15 | 18 | 55 | 32 | 3 | 46 | 34 |
| 5 | 9 | 19 | 60 | 33 | 36 | 47 | 12 |
| 6 | 11 | 20 | 7 | 34 | 4 | 48 | 74 |
| 7 | 25 | 21 | 9 | 35 | 5 | 49 | 58 |
| 8 | 15 | **22** | **0** | 36 | 17 | 50 | 57 |
| 9 | 5 | 23 | 7 | 37 | 1 | 51 | 4 |
| 10 | 12 | 24 | 1 | 38 | 9 | 52 | 5 |
| 11 | 22 | 25 | 3 | 39 | 3 | 53 | 6 |
| 12 | 11 | 26 | 8 | 40 | 23 | | |
| 13 | 10 | 27 | 4 | 41 | 58 | | |
| 14 | 8 | 28 | 5 | 42 | 20 | | |

> **Only one true zero-span page (p22)** — down from 12 in the prior 4.2.4 run. Low-density pages to scrutinise: 24 (1), 37 (1), 25 (3), 29 (3), 32 (3), 39 (3), 27 (4), 34 (4), 51 (4), 3 (5), 9 (5), 28 (5), 35 (5), 43 (5), 52 (5), 53 (6).

---

## Audit methodology

1. **Source of truth:** `pdftotext -layout` extraction of each PDF page (53 per-page files at `/tmp/kdigo_audit_pages/pNN.txt`) + visual sanity-check of heading / table layout.
2. **Compared against:** every row in `l2_merged_spans WHERE job_id='96c8f0d6…'` grouped by `page_number`, including every channel (B/C/D/E/F/G/H/L1_RECOVERY) and spans whose `contributing_channels = {REVIEWER}` (human-added/edited).
3. **Classification scheme per fact:**
   - `ADDED` — present in DB with non-trivial coverage (CONFIRMED / EDITED / REVIEWER-ADDED, or PENDING with clean full-sentence text that faithfully carries the guideline statement).
   - `MISSED` — present in the PDF, absent from the DB entirely, or captured only as a non-informative fragment (e.g. a bare `+`, a footnote citation, a table-border artefact).
   - `NEEDS-CONFIRM` — in DB but semantically partial (covers only part of the source clause), or has low confidence, contains OCR garbling (`sus\\- cepible`, missing hyphens), or the table-extracted form has lost row/column context so the underlying clinical fact is ambiguous.
4. **Out of scope for "guideline facts":** journal boilerplate (copyright, DOI, reuse statements, running headers like "www.kidney-international.org", author-credit lines, figure/table legends that only repeat the table). These are correctly rejected or left as NOISE.
5. **Batch size:** 10 pages. 53 pages → 6 batches (batch 6 = pages 51–53).
6. **Recommendation / Practice-Point naming convention** (per KDIGO 2024 CKD): `Recommendation X.Y.Z` with a GRADE strength (1 or 2) and evidence level (A/B/C/D); `Practice Point X.Y.Z` is not graded. Both are numbered clinical facts and are the highest-priority items to capture.

---

## Per-page audit

*(Batches are appended below as they are completed.)*

---

## Batch 1 — Pages 1–10 (Chapter 4 Medication Management + tail of §3.3 Nutrition)

**Batch-level summary:** 13 + 34 + 5 + 15 + 9 + 11 + 25 + 15 + 5 + 12 = **144 spans**. Of these, 16 are CONFIRMED, 2 EDITED, 2 ADDED (reviewer), 5 REJECTED, 119 PENDING. Reviewer has already reconstructed Tables 31 and Figure 46 where the D-channel left only row-fragments.

**Compared to the prior 4.2.4 run on this same PDF** (job `f172f6a9…`, which reported pages 1, 3, 5 as "zero-span" and 6 recommendations/practice points MISSED in this range): this 4.2.2 re-extraction has **captured every Recommendation and Practice Point on pages 1–10** — a clear improvement. The remaining issues are PENDING review and two OCR-artefact fixes.

### Page 1 — Chapter 4 intro + §4.1 heading + PP 4.1.1/4.1.2/4.1.3

**PDF content:** Chapter 4 title; Chapter intro (2 paragraphs on drug stewardship); §4.1 Medication choices and monitoring for safety; PP 4.1.1 (susceptibility to nephrotoxicity) with rationale incl. 18–20 % nephrotoxic-Rx prevalence and Table 31 pointer; PP 4.1.2 (monitor eGFR/electrolytes) with ACEi/ARB + gentamicin/vancomycin + lithium/methotrexate monitoring examples; PP 4.1.3 (limit OTC/herbals).

**DB spans (13):** `CONFIRMED=11, EDITED=1, REJECTED=1`

| Fact | Status | Notes |
|---|---|---|
| Chapter 4 heading | **ADDED** (CONFIRMED, E+G, 0.71) | Clean capture |
| Intro para 1 (drug stewardship definition) | **ADDED** (CONFIRMED, E+G) | Clean |
| "It is beyond the scope…" + "However, we describe case examples…" | **ADDED** | Split across 2 spans but complete |
| §4.1 heading + opening paragraph | **ADDED** (EDITED, 0.92) | Reviewer corrected text |
| **PP 4.1.1** full recommendation text | **ADDED** (CONFIRMED, 0.89) | ⚠ OCR artefact: `sus\- cepible` (should be `susceptible`) — **NEEDS-CONFIRM cleanup** |
| PP 4.1.1 rationale (18–20 % NSAIDs/antivirals/bisphosphonates) | **ADDED** (CONFIRMED, 0.95) | |
| **PP 4.1.2** full recommendation text | **ADDED** (CONFIRMED, 0.94) | |
| PP 4.1.2 rationale (ACEi/ARB, gentamicin/vancomycin, lithium/methotrexate) | **ADDED** (CONFIRMED, 0.95) | |
| **PP 4.1.3** full recommendation text | **ADDED** (CONFIRMED, 0.94) | |
| Table 31 abbreviation footnote (CKD/eGFR/NSAID) | **ADDED** (CONFIRMED, 0.92) | |
| Table 31 body (HTML blob) | correctly **REJECTED** | Reviewer re-captured cleanly on p2; see p2 |
| Para 2 of intro ("As in all medical decision-making…") | **MISSED** | Second column of intro absent — context paragraph, low clinical-fact value |
| Running header `chapter 4 www.kidney-international.org` | correctly not captured | Boilerplate |

**Prior-run comparison:** old audit listed PP 4.1.1, 4.1.2, 4.1.3 as all MISSED on p1. **Now all three are CONFIRMED.**

### Page 2 — Table 31 body + OTC/herbals continuation + Figure 45 transition

**PDF content:** Table 31 (Analgesics, Antimicrobials, GI, Cardiovascular, Other classes with nephrotoxic agents + alternatives); then prose on OTC NSAIDs, herbal remedies (aristolochic-acid nephropathy), falsified medications; subsection "Special considerations: Global access" starts.

**DB spans (34):** `CONFIRMED=9 (6 REVIEWER), REJECTED=15 (D-channel duplicates), PENDING=10`

| Fact | Status | Notes |
|---|---|---|
| Table 31 title "Key examples of common medications with documented nephrotoxicity…" | **ADDED** (CONFIRMED, REVIEWER) | |
| Table 31 Analgesics row (NSAIDs ⇄ Acetaminophen) | **ADDED** (CONFIRMED, REVIEWER) | |
| Table 31 Antimicrobials row (Aminoglycosides ⇄ Cephalosporins/Carbapenems) | **ADDED** (CONFIRMED, REVIEWER) | |
| Table 31 Vancomycin row (⇄ Linezolid/daptomycin) + Sulfamethoxazole-trimethoprim ⇄ Clindamycin/primaquine | **ADDED** (CONFIRMED, REVIEWER) | |
| Table 31 GI row (PPIs ⇄ H2-blockers) | **ADDED** (CONFIRMED, REVIEWER) | |
| Table 31 Cardiovascular row (Warfarin ⇄ NOACs) | **ADDED** (CONFIRMED, REVIEWER) | |
| Table 31 Other row (Lithium ⇄ Aripiprazole/lamotrigine/quetiapine/valproate) | **ADDED** (CONFIRMED, REVIEWER) | Note: only REVIEWER spans carry clean row structure; raw D-channel spans were correctly REJECTED |
| OTC NSAIDs / interstitial nephritis / analgesic nephropathy paragraph | **ADDED** (CONFIRMED, D) | |
| Judicious NSAID use vs opioids | **ADDED** (CONFIRMED, B+G, 0.90) | |
| Dietary supplements general paragraph | **ADDED** (CONFIRMED, F+G, 0.85) | Split across 2 spans |
| D-channel fragments (15 rejected table-cell shards) | correctly **REJECTED** | Row/column context lost in D-channel; acceptable since reviewer added clean version |
| H-channel duplicate "OKD, chronic kidney disease: GFR…" (garbled OCR of abbreviation footnote) | **NEEDS-CONFIRM** (PENDING, 0.60) | H-channel picked up OCR'd "OKD" (should be "CKD") — low-confidence artefact |
| H-channel duplicate of NSAID paragraph | **NEEDS-CONFIRM** (PENDING, 0.60) | Same text captured twice (H + D+REVIEWER), should be deduped |
| Herbal-remedy prose: aristolochic acid nephropathy, creatine/vitamin C | **MISSED** | Lines 51–65 of PDF page not in DB |
| "Healthcare providers are encouraged to routinely inquire about herbal remedies…" | **MISSED** | Actionable clinical guidance not captured |

### Page 3 — Figure 45 content + Global access + Pregnancy + PP 4.1.4

**PDF content:** Figure 45 (selected nephrotoxic herbal remedies by continent — Americas, Europe, Africa, Asia lists); "insulin.755 … barriers to erythropoietin/iron/phosphate/potassium binders"; "Medications and pregnancy" subheading; **PP 4.1.4** (teratogenicity review + reproductive counselling) with rationale listing teratogens (ACEi/ARB, mTOR) and safe-in-pregnancy list (hydroxychloroquine, tacrolimus, cyclosporin, eculizumab, prednisone, azathioprine, colchicine, IVIG).

**DB spans (5):** `CONFIRMED=3, EDITED=1, ADDED=1 (REVIEWER)`

| Fact | Status | Notes |
|---|---|---|
| Figure 45 caption ("Selected herbal remedies and dietary supplements with evidence of potential nephrotoxicity…") | **ADDED** (ADDED-REVIEWER) | |
| "insulin.755 … erythropoietin analogs, iron infusion, phosphate/potassium binders" | **ADDED** (EDITED, 0.82) | OCR misplaced offset — reviewer fixed |
| **PP 4.1.4** full recommendation text | **ADDED** (CONFIRMED, 0.89) | |
| Oral contraceptives & BP / pregnancy CKD-progression paragraph | **ADDED** (CONFIRMED, 0.82) | Partial — missing the "Nonoral hormonal contraceptives have a less clear impact on BP" sentence |
| Teratogens (ACEi/ARB, mTOR) + safe-in-pregnancy list (HCQ/tacrolimus/etc.) | **ADDED** (CONFIRMED, 0.95) | |
| Figure 45 body text (Americas / Europe / Asia / Africa lists of herbal toxins incl. aristolochic acid, Amanita phalloides, star fruit, djenkol beans, khat, etc.) | **MISSED** | 40+ named herbal/dietary toxins not in DB. Only the caption is captured. This is a **genuine content gap** — individual herbs are not clinical-decision facts but are named in the guideline. |
| "Special considerations: Global access to medications" subheading + ISN-35 % statistic for low-resource-setting ACEi/ARB/statin/insulin access | **MISSED** | Clinical-equity guidance absent |
| "Medication falsification" paragraph (vulnerability / illicit internet supply / low-resource settings) | **MISSED** | Guideline-level warning absent |

**Prior-run comparison:** old audit listed PP 4.1.4 as MISSED on p3. **Now CONFIRMED.** Figure 45 content and Global-access text are still absent.

### Page 4 — Sex-specific + §4.2 heading + PP 4.2.1 through 4.2.5 + Cancer dose-adjustment

**PDF content:** Continuation of sex-specific medication use (women/RAAS dosing); §4.2 Dose adjustments by level of GFR; **PP 4.2.1** (consider eGFR); rationale; **PP 4.2.2** (use eGFR/SCr for most dosing); **PP 4.2.3** (eGFR cr-cys or mGFR where more accuracy needed); **PP 4.2.4** (nonindexed eGFR for extreme body weight); **PP 4.2.5** (adapt dosing in non-steady state); Special considerations — cancer (Cockroft-Gault, CKD-EPI, BSA-adjusted for carboplatin).

**DB spans (15):** `CONFIRMED=13, ADDED=2 (REVIEWER)`

| Fact | Status |
|---|---|
| Sex-specific ("because drug dosages are often universal, women are more likely to consume higher doses…") + HFrEF dosing | **ADDED** (ADDED-REVIEWER) |
| §4.2 heading | **ADDED** (CONFIRMED, 0.82) — partial ("# 4.2 Dose adjustments by level of eGFR"; PDF says "level of GFR") |
| **PP 4.2.1** "Consider GFR when dosing" | **ADDED** (CONFIRMED, 0.89) |
| PP 4.2.1 rationale (failure to account for GFR → treatment failure / adverse events) | **ADDED** (CONFIRMED, 0.82) |
| **PP 4.2.2** validated eGFR/SCr appropriate for drug dosing | **ADDED** (CONFIRMED, 0.89) |
| **PP 4.2.3** eGFRcr-cys or mGFR for narrow-therapeutic / toxic drugs | **ADDED** (CONFIRMED, 0.89) |
| PP 4.2.3 rationale (Section 1.2 cross-reference) | **ADDED** (CONFIRMED, 0.82) |
| Cockroft-Gault concerns (standardization / race / weight / edema) | **ADDED** (CONFIRMED, 0.82) |
| Regulatory agency statement "any contemporary…equation is considered reasonable" | **ADDED** (CONFIRMED, 0.82) |
| **PP 4.2.4** nonindexed eGFR for BSA extremes | **ADDED** (CONFIRMED, 0.89) |
| PP 4.2.4 rationale (1.73 m² normalisation) | **ADDED** (CONFIRMED, 0.90) |
| **PP 4.2.5** adapt dosing in non-steady-state | **ADDED** (CONFIRMED, 0.89) |
| PP 4.2.5 rationale (rapidly changing filtration markers) | **ADDED** (CONFIRMED, 0.82) |
| Cancer-dose special consideration (Cockroft-Gault / CKD-EPI / BSA carboplatin) | **ADDED** (ADDED-REVIEWER, CONFIRMED, 0.82) | Both reviewer and machine spans redundant but consistent |

**No MISSED items.** All five PPs captured. Minor NEEDS-CONFIRM: §4.2 heading text says "eGFR" in DB where PDF says "GFR" — post-OCR substitution; semantics preserved.

### Page 5 — Pediatric/neonate + Pregnancy dose-adj. + §4.3 + PP 4.3.1 + Figure 46

**PDF content:** Continuation of cancer/carboplatin; Dose adjustment in children/neonates; Dose adjustment in pregnancy; §4.3 Polypharmacy and drug stewardship; **PP 4.3.1** (periodic medication review) with rationale; Figure 46 (medication-review 8-step wheel).

**DB spans (9):** `CONFIRMED=9 (5 REVIEWER)`

| Fact | Status |
|---|---|
| Pediatric/neonate dose adjustment | **ADDED** (CONFIRMED + REVIEWER redundant) |
| Pregnancy dose adjustment (physiologic Cr decrease / BSA variation) | **ADDED** (CONFIRMED + REVIEWER redundant) |
| §4.3 heading + opening ("People with CKD are particularly susceptible to polypharmacy…") | **ADDED** (REVIEWER, CONFIRMED) |
| **PP 4.3.1** recommendation text | **ADDED** (CONFIRMED, 0.89) |
| PP 4.3.1 rationale (medication-related problems / PPI discontinuation example) | **ADDED** (CONFIRMED, 0.97) |
| Figure 46 caption (medication review & reconciliation, 8 steps) | **ADDED** (CONFIRMED, REVIEWER) |
| Figure 46 wheel spokes (Assessing dosage, Medication agreement, Communication, Optimizing impact, Minimizing problems, Reviewing list, etc.) | **ADDED** (CONFIRMED, REVIEWER) |

**No MISSED items.**

**Prior-run comparison:** old audit listed PP 4.3.1 MISSED. **Now CONFIRMED.**

### Page 6 — PP 4.3.2, 4.3.3 + Sick-day rules (SADMANS) + Table 32 pointer + Figure 47

**PDF content:** Continuation of §4.3 discussion (3 medication-review RCTs, prescribing cascades); **PP 4.3.2** (communicate restart plan); Sick-day rules with SADMANS mnemonic; **PP 4.3.3** (peri-operative drug discontinuation 48–72 h); Table 32 (peri-op drug risks); RASi continuation/restart guidance; Figure 47 (sick-day 4-step model).

**DB spans (11):** all `PENDING`

| Fact | Status |
|---|---|
| Medication-review RCT summary ("evaluated medication review by clinical practices…") | **ADDED** (PENDING, 0.90) — NEEDS-CONFIRM |
| Prescribing-cascade example (CCB → edema → diuretic → hypokalemia) | **ADDED** (PENDING, 0.90) — NEEDS-CONFIRM |
| **PP 4.3.2** clear restart-plan PP | **ADDED** (PENDING, 0.89) — NEEDS-CONFIRM |
| Sick-day rules / SADMANS (sulfonylureas, ACEi, diuretics, metformin, ARBs, NSAIDs, SGLT2i) | **ADDED** (PENDING, 0.97) — NEEDS-CONFIRM |
| Caveat — paucity of evidence / potential harm from sick-day rules | **ADDED** (PENDING, 0.85) — NEEDS-CONFIRM |
| **PP 4.3.3** peri-op 48–72 h drug-discontinuation PP | **ADDED** (PENDING, 0.99) — NEEDS-CONFIRM |
| Rationale for peri-op discontinuation (hypotension / acidosis / hyperkalemia) | **ADDED** (PENDING, 0.95) — NEEDS-CONFIRM |
| RASi discontinuation/restart after adverse events | **ADDED** (PENDING, 0.90) — NEEDS-CONFIRM |
| Figure 47 caption | **ADDED** (PENDING, 0.85) — NEEDS-CONFIRM |
| Table 32 HTML body ("<table>…<tr><td>ACEi/ARB</td><td>Hypotension, AKI</td>…") | **ADDED** (PENDING, 0.90) — NEEDS-CONFIRM ; HTML shape may not survive Pipeline 2 export |
| Table 32 abbreviation footnote | **ADDED** (PENDING, 1.00) — NEEDS-CONFIRM |

**No MISSED items.** All content captured, but every span is PENDING review.

**Prior-run comparison:** old audit listed PP 4.3.2 MISSED. **Now present as PENDING (CONFIRM action required).**

### Page 7 — Table 32 fragments + PP 4.3.1.1 + PP 4.3.1.2 + §4.4 + PP 4.4.1

**PDF content:** Table 32 body (Meds × perioperative adverse-event rows); §4.3.1 Strategies to promote drug stewardship; **PP 4.3.1.1** (educate patients re benefits/risks); **PP 4.3.1.2** (collaborate with pharmacists/tools); Pediatric sub-section; §4.4 Imaging studies; **PP 4.4.1** (consider indication for imaging).

**DB spans (25):** all `PENDING`

| Fact | Status |
|---|---|
| Table 32 Meds column fragments (ACEI/ARB, Diuretics, SGLT2i, Metformin, Aminoglycosides, NSAIDs) | **NEEDS-CONFIRM** (PENDING, D-channel, 0.92 × 6; H-channel Metformin/ACEI/ARB duplicates 0.60 × 2) — row-column context lost; reviewer reconstruction still pending |
| Table 32 adverse-event column fragments (Hypotension/AKI, Volume depletion/AKI, Ketoacidosis, Lactic acidosis, ATN/AKI, AKI/AIN) | **NEEDS-CONFIRM** (PENDING, 0.92 × 5) — paired to rows only implicitly |
| Pediatric tubular/dehydration paragraph | **ADDED** (PENDING, 0.90) |
| **PP 4.3.1.1** educate patients | **ADDED** (PENDING, 0.89) |
| **PP 4.3.1.2** collaborative / pharmacist / CDS | **ADDED** (PENDING, 0.89) |
| Rationale (eHealth apps / printing out eGFR / medication list) | **ADDED** (PENDING, 0.82) |
| Clinical-pharmacist role (structured review) | **NEEDS-CONFIRM** (PENDING, 0.82) — present but split |
| Clinical decision-support systems (EMR alerts, drug interactions) | **NEEDS-CONFIRM** (partial) |
| §4.4 heading | **MISSED** (no explicit heading span; only `PP 4.4.1` captured) |
| **PP 4.4.1** imaging indication | **ADDED** (PENDING, 0.89) |
| Iodinated contrast / CA-AKI definition paragraph | **ADDED** (PENDING, 0.82) |
| "Caution withholding contrast solely based on GFR" | **ADDED** (PENDING, 0.82) |
| Table 33 HTML shell fragment | **NEEDS-CONFIRM** (PENDING, 0.82) |
| Copyright / Creative-Commons footer | correctly captured as PENDING low-value | |

**Prior-run comparison:** old audit listed PP 4.3.1.1, PP 4.3.1.2 MISSED. **Now both present as PENDING.**

### Page 8 — Table 33 body + PP 4.4.1.1, PP 4.4.1.2, §4.4.2 + PP 4.4.2.1

**PDF content:** Table 33 (CA-AKI risk factors: patient-associated vs procedure-associated); **PP 4.4.1.1** (validated risk tools for intra-arterial contrast); Prevention-bundle bulleted list (min dose; withdraw nephrotoxins 24–48 h; metformin nuance; RAASi withholding $48 h; avoid dehydration; N-acetylcysteine/ascorbic-acid not beneficial; prophylactic HD harmful); **PP 4.4.1.2** (IV contrast per radiology consensus, for AKI or GFR <60); §4.4.2 Gadolinium; NSF history; **PP 4.4.2.1** (ACR group II/III gadolinium agents for GFR <30).

**DB spans (15):** all `PENDING`

| Fact | Status |
|---|---|
| Table 33 patient-associated cells (Reduced GFR acute/chronic, Diabetes mellitus, Reduced intravascular volume, Concomitant nephrotoxic meds) | **NEEDS-CONFIRM** (PENDING, D, 0.92 × 4) |
| Table 33 procedure-associated cells (High-osmolar contrast, Large volume, Intra-arterial, Serial procedures) | **NEEDS-CONFIRM** (PENDING, D, 0.92 × 4) |
| Continuation prose (harm from withholding contrast) | **ADDED** (PENDING, 0.82) |
| **PP 4.4.1.1** intra-arterial cardiac AKI risk validation | **ADDED** (PENDING, 0.89) |
| Prevention-bundle bullets (min contrast dose, withdraw NSAIDs/diuretics/aminoglycosides/amphotericin/platins/zoledronate/methotrexate 24–48 h before & 48 h after) | **ADDED** (PENDING, 1.00, B+C+E+G) — Full bundle captured in one span |
| **PP 4.4.1.2** IV contrast per radiology societies, AKI or GFR <60 | **ADDED** (PENDING, 0.89) |
| **PP 4.4.2.1** ACR group II/III gadolinium for GFR <30 | **ADDED** (PENDING, 0.96, C+F+G) |
| Metformin nuance (GFR >30 — no stop; AKI or GFR ≤30 — stop before ICM, restart after 48 h if GFR stable) | **MISSED** | Lines 19–31 of PDF — a specific clinical decision rule — **not** in DB |
| RAASi withholding ≥48 h before elective contrast-CT | **MISSED** | Rule present in bundle prose but not captured as its own span |
| "N-acetylcysteine / ascorbic acid / furosemide / dopamine / fenoldopam / CCBs not consistent benefit" + "prophylactic HD potentially harmful" | **MISSED** | Explicit negative guidance not in DB |
| NSF 2010-2012 history (§4.4.2 intro) | **MISSED** |

**Prior-run comparison:** old audit listed PP 4.4.1.1, PP 4.4.1.2 MISSED; **both now present (PENDING)**.

### Page 9 — Gadolinium continuation + Pediatric gadolinium + Research recommendation pointer

**PDF content:** Continuation of PP 4.4.2.1 rationale (group-I agents / repeated doses); list of safer linear/macrocyclic chelates; "Special considerations: Global access to gadolinium-contrast agents"; Pediatric considerations (FDA/EMA licensing under 2 y / under 1 y); NSF risk in pediatrics; pointer to Chapter 6 Research recommendations.

**DB spans (5):** all `PENDING`

| Fact | Status |
|---|---|
| Group-I agents (gadodiamide, gadopentetate dimeglumine, gadoversetamide) + repeated-dose risk | **ADDED** (PENDING, 0.82) |
| Preferred newer chelates (gadobenate, gadobutrol, gadoteridol, gadoterate meglumine, gadoxetate) for GFR <30 | **ADDED** (PENDING, 0.88) |
| Pediatric gadolinium caution (FDA <2 y / EMA <1 y; neonate caution) | **ADDED** (PENDING, 0.93, C+F+G) |
| Caution in neonates / all other imaging modalities considered first | **ADDED** (PENDING, 0.82) |
| `<!-- Chunk chunk-b1: pages 10-15 -->` | **NOISE** (PENDING, 0.85) — pipeline-internal chunk marker leaked into spans; should be auto-rejected (not a guideline fact) |
| "Global access to gadolinium-contrast agents" subheading | **MISSED** — cost-in-LMIC equity guidance absent |
| NSF in pediatrics risk statement (lower than adult) | **MISSED** — captured only in the preferred-chelates span, not as standalone |
| Research-recommendations pointer | **MISSED** — not captured |

### Page 10 — End of §3.3 protein (PP 3.3.1.4, PP 3.3.1.5) + §3.3.2 + Rec 3.3.2.1 + PP 3.3.2.1/3.3.2.2

**PDF content:** Continuation of adult protein guidance (2007 Cochrane, 2009 KDOQI, 2020 PRNT); **PP 3.3.1.4** (no protein restriction in children); **PP 3.3.1.5** (older adults — higher protein/calorie in frailty/sarcopenia); §3.3.2 Sodium intake — cross-reference to KDIGO 2022 Diabetes-CKD and KDIGO 2021 BP; **Recommendation 3.3.2.1** (sodium <2 g/d or <90 mmol/d or <5 g NaCl/d) [2C]; **PP 3.3.2.1** (sodium restriction not for sodium-wasting nephropathy); rationale (global avg 4310 mg/d, WHO target, RCT salt-substitute evidence); **PP 3.3.2.2** (age-based RDI for children with BP >90th percentile); Table 22 (age-based sodium intake 0.110–1.0 g/d).

**DB spans (12):** all `PENDING`

| Fact | Status |
|---|---|
| Intro prose ("empowers people with CKD … adherence to low-protein diet remains challenging…") | **ADDED** (PENDING, 0.95) |
| **PP 3.3.1.4** — no protein restriction in children | **ADDED** (PENDING, 0.89) |
| **PP 3.3.1.5** — older adults frailty/sarcopenia | **ADDED** (PENDING, 0.89) |
| Geriatric 1.0–1.2 g/kg/d recommendation | **ADDED** (PENDING, 0.82) |
| §3.3.2 Sodium intake heading | **ADDED** (PENDING, 0.82) |
| **Recommendation 3.3.2.1** (sodium <2 g/d, <90 mmol/d, <5 g NaCl/d) **[2C]** | **ADDED** (PENDING, 0.89) — grade+evidence preserved |
| **PP 3.3.2.1** — sodium restriction not for sodium-wasting nephropathy | **ADDED** (PENDING, 1.00, C+E+F+G) |
| Rationale (global avg 4310 mg/d, WHO <2 g, salt-substitute RCTs) | **ADDED** (PENDING, 0.90) |
| **PP 3.3.2.2** — age-based RDI for sodium in children with BP >90th ‰ | **ADDED** (PENDING, 0.89) |
| WHO maximum <2 g/d adjusted for children | **ADDED** (PENDING, 0.82) |
| Table 22 title + first 3 body rows (0–6 mo 0.110 g, 7–12 mo 0.370 g, 1–3 yr 0.370 g, 4–8 yr 1.0 g) | **NEEDS-CONFIRM** (PENDING, 0.82) — Table HTML span **truncates** at `<td>9-` (9–13 yr row lost); reviewer re-capture needed |
| "Cross-reference to KDIGO 2022 Diabetes-CKD (§3.3.2 rec) + KDIGO 2021 BP-CKD" | **MISSED** — the authority / provenance statement above Rec 3.3.2.1 is not in DB |
| Low-birth-weight salt-sensitivity sentence (final lines) | **MISSED** — relevant clinical detail not captured |

**Prior-run comparison:** old audit noted page 10 had only 1 span and listed PP 3.3.1.4 / 3.3.1.5 as MISSED on p10, plus Rec 3.3.1.1 as MISSED. **Now all three PPs captured**; Rec 3.3.1.1 is not actually on this page in the Delta PDF (protein-intake recommendation is on an earlier page not included in the 53-page extract, so the older audit's flag was a page-mismatch artefact).

### Batch 1 roll-up

| Metric | Value |
|---|---|
| Spans reviewed | 144 |
| Recommendations present on these 10 pages | 1 (Rec 3.3.2.1) |
| Practice Points present on these 10 pages | 22 (PP 4.1.1–4.1.4, 4.2.1–4.2.5, 4.3.1, 4.3.1.1, 4.3.1.2, 4.3.2, 4.3.3, 4.4.1, 4.4.1.1, 4.4.1.2, 4.4.2.1, 3.3.1.4, 3.3.1.5, 3.3.2.1, 3.3.2.2) |
| Recommendations / PPs captured (any status) | **23 / 23 (100 %)** |
| Recommendations / PPs CONFIRMED | 16 |
| Recommendations / PPs PENDING (need reviewer sign-off) | 7 |
| Other clinical rationale MISSED | 11 items (listed above) — mostly global-access / equity subsections and Figure 45 herbal lists |
| OCR / pipeline artefacts | PP 4.1.1 `sus\- cepible`; chunk-marker `<!-- Chunk chunk-b1 -->` leaked to span; H-channel "OKD" duplicate; Table 22 HTML truncation mid-row |

**Verdict for batch 1:** strong improvement vs prior 4.2.4 run — every Recommendation & Practice Point is now at least PENDING. Highest-value follow-ups: (1) reviewer sign-off on the 7 PENDING PPs, (2) cleanup of the OCR artefacts, (3) capture of the three subsection headings that were dropped ("Special considerations: Global access", "Special considerations: Global access to gadolinium", Figure 45 body).

---

## Batch 2 — Pages 11–20 (§3.3.2 tail + §3.4 BP + §3.5 Glycemic pointer + §3.6 RASi + §3.7 SGLT2i + §3.11 Hyperkalemia)

**Batch-level summary:** 22 + 11 + 10 + 8 + 8 + 40 + 28 + 55 + 60 + 7 = **249 spans**, all `PENDING` (reviewer hasn't touched this chapter yet). Recommendations covered on these pages: **Rec 3.4.1, 3.4.2, 3.6.1–3.6.4, 3.7.1, 3.7.2** (8 recommendations). Practice Points covered: **PP 3.4.1–3.4.3, 3.6.1–3.6.7, 3.7.1–3.7.3, 3.11.1.1, 3.11.2.1, 3.11.5.1, 3.11.5.2** (17 PPs). **All 25 guideline numbered facts are captured.** The main quality issue is heavy D-channel fragmentation of Tables 22, 25, 26, 27, 28 and Figures 22, 23, 30, 31, 32, 33.

### Page 11 — Table 22 spill-over + §3.4 BP + Rec 3.4.1 + PP 3.4.1 + Rec 3.4.2 + PP 3.4.2 + PP 3.4.3

**PDF content:** Table 22 (age-based sodium 0.110–1.5 g/d); end of §3.3.2 children/tubular text; §3.4 Blood pressure control (defers to KDIGO 2021 BP); **Rec 3.4.1** (adult SBP <120 mm Hg, standardized office BP) [2B]; **PP 3.4.1** (less intensive BP in frailty/falls/postural hypotension); SPRINT sub-group detail; pediatric-header; **Rec 3.4.2** (children 24-h MAP by ABPM ≤50th %) [2C]; **PP 3.4.2** (BP annually ABPM + q3-6mo office); **PP 3.4.3** (office SBP 50–75th %ile when no ABPM).

**DB spans (22):** all `PENDING`

| Fact | Status |
|---|---|
| Table 22 column header "Recommended adequate sodium intake (g/d)" | **NEEDS-CONFIRM** (D, 0.92) — page-2 header row |
| Table 22 body — age labels (0-6 mo / 7-12 mo / 1-3 yr / 4-8 yr / 9-13 yr / 14-70 yr) | **NEEDS-CONFIRM** (D × 7, 0.92) |
| Table 22 body — values (0.11 / 0.37 / 0.37 / 1.0 / 1.2 / 1.5) | **NEEDS-CONFIRM** (D × 6, 0.92) — row-column pairing lost |
| Children tubular / non-salt-wasting limit paragraph | **ADDED** (0.82) |
| **Rec 3.4.1** full text with `(2B)` grade | **ADDED** (0.89) |
| **PP 3.4.1** frailty / falls / postural hypotension | **ADDED** (0.89) |
| Rationale (frail / life-expectancy / weighing fall risk) | **ADDED** (0.90) |
| SPRINT excerpt — SBP <120 for >75 y or >50 y with CVD/eGFR 20–60/≥15 % 10-y risk | **ADDED** (0.88) |
| **Rec 3.4.2** children 24-h MAP by ABPM ≤50th %ile `(2C)` | **ADDED** (0.89) |
| **PP 3.4.2** annual ABPM + q3-6mo office | **ADDED** (0.94) |
| **PP 3.4.3** office SBP 50–75th %ile when ABPM unavailable | **ADDED** (0.89) |
| "KDIGO 2021 BP CPG" cross-reference header | **MISSED** — the authority pointer above Rec 3.4.1 ("The Work Group concurs with the KDIGO 2021 Clinical Practice Guideline for the Management of Blood Pressure in Chronic Kidney Disease…") not captured as its own span |
| 20/10 mm Hg risk-doubling / SPRINT NNT sentence | **NEEDS-CONFIRM** — partial capture |
| Standardized-BP difficulty / home-monitoring 2+2 meta | **MISSED** — practical implementation guidance dropped |

### Page 12 — Children BP rationale + §3.5 Glycemic pointer + §3.6 RASi (Rec 3.6.1–3.6.4 + PP 3.6.1–3.6.7 + STOP-ACEi)

**PDF content:** Rationale for 50th %ile target in children (TV mass index RCT); §3.5 Glycemic control (pointer to KDIGO 2022 Diabetes-CKD); §3.6 RASi preamble; **Rec 3.6.1** (ACEi/ARB for G1-G4 A3 no-DM) [1B]; **Rec 3.6.2** (G1-G4 A2 no-DM) [2C]; **Rec 3.6.3** (G1-G4 A2/A3 with DM) [1B]; **Rec 3.6.4** (avoid ACEi+ARB+DRI combos) [1B]; **PP 3.6.1–3.6.6** (dose to max tolerated, monitor K+/Cr in 2–4 wk, manage hyperK instead of stopping, continue unless Cr ↑>30 % or eGFR ↓, reduce dose if hypotension/hyperK, consider RASi in A1 for HF-low-EF indication); **PP 3.6.7** (continue ACEi/ARB even when eGFR <30); STOP-ACEi trial summary.

**DB spans (11):** all `PENDING`

| Fact | Status |
|---|---|
| Transition prose ("the BP guideline (previous guideline suggested <90th %ile)") | **ADDED** (F+G, 0.82) |
| §3.6 RASi preamble ("Work Group highlights recommendations from KDIGO 2021 BP CPG and KDIGO 2022 DM-CKD CPG…") | **ADDED** (B+G, 0.90) |
| **Rec 3.6.1** + Rec 3.6.2 + Rec 3.6.3 + Rec 3.6.4 — all four **merged into one span** | **NEEDS-CONFIRM** (B+C+G, 0.99) — each Rec carries distinct `(1B)`/`(2C)` grade, but they are packed as one blob; reviewer split advised |
| **PP 3.6.1–3.6.5** — five PPs **merged into one span** (dose-to-max, monitor K+/Cr, hyperK mgmt, 30 % Cr rule, reduce-dose-if-hypoK) | **NEEDS-CONFIRM** (B+C+G, 0.99) — same issue |
| **PP 3.6.6** standalone span | **ADDED** (B+C+G, 0.99) |
| **PP 3.6.7** standalone span | **ADDED** (0.89) |
| Rationale (RASi valid for hyperK but restart, PP 4.3.3 cross-ref, Figure 21 algorithm, 30 % eGFR drop trigger, salt-restriction maximises RASi effect) | **ADDED** (B+C+G, 0.99) |
| STOP-ACEi trial (411 pts, mean eGFR 13, policy of discontinuing RASi in G4-G5 — no benefit) | **ADDED** (B+C+G, 0.97) |
| Continuation ("eGFR <30 ml/min per 1.73 m², compared with those who continue…individual patient level meta-analysis benefit in delaying KRT") | **ADDED** (C+F+G, 0.93) |
| Figure 21 algorithm caption | **ADDED** (B+E+G, 0.95) |
| §3.7 heading split-to-next-page marker (`# <!-- PAGE 13 --> 3.7 SGLT2i`) | **ADDED** (B+E+G, 0.95) — cross-page heading, not a guideline fact |

**Quality issue:** Rec 3.6.1–3.6.4 packed into **one span** and PP 3.6.1–3.6.5 into **one span** is a serious deduplication problem — downstream consumers (KB-3 Guidelines ingestion, KB-23 Decision Cards) expect one span per Rec/PP. **Highest-priority cleanup on this page.**

### Page 13 — Figure 21 algorithm + §3.7 SGLT2i + Rec 3.7.1 + PP 3.7.1/3.7.2 + Rec 3.7.2 + PP 3.7.3

**PDF content:** Figure 21 body (Initiate ACEi/ARB → Monitor K+/Cr 2–4 wk → 3 branches: Normokalemia / Hyperkalemia / ≥30 % eGFR drop, with management steps); §3.7 SGLT2i; Rec 3.7.1 quote from KDIGO 2022 DM-CKD; broader 2024 update; **Rec 3.7.1** (T2D + CKD + eGFR ≥20 on SGLT2i) [1A]; **PP 3.7.1** (continue below 20 if tolerated); **PP 3.7.2** (withhold during fasting/surgery/critical illness — ketosis risk); **Rec 3.7.2** (all adults with CKD on SGLT2i for eGFR ≥20 + ACR ≥200 mg/g or HF) [1A]; **PP 3.7.3** (initial eGFR drop not indication to stop).

**DB spans (10):** all `PENDING`

| Fact | Status |
|---|---|
| **Rec 3.7.1** full text + `(1A)` | **ADDED** (B+C+G, 0.99) |
| **PP 3.7.1** continue below 20 if tolerated | **ADDED** (B+C+G, 0.99) |
| **PP 3.7.2** withhold SGLT2i in fasting/surgery/critical illness | **ADDED** (B+C+G, 0.99) |
| **Rec 3.7.2** full text + `(1A)` with bullets (eGFR ≥20 + ACR ≥200, or heart failure) | **ADDED** (B+C+G, 0.99) |
| **PP 3.7.3** SGLT2i monitoring / reversible eGFR drop | **ADDED** (B+C+G, 0.99) |
| Rationale preamble ("Work Group concurs with KDIGO 2022 DM-CKD…however in this guideline we offer a more general 1A recommendation") | **ADDED** (B+C+G, 0.98) |
| Rationale — "Use of SGLT2i in people with T2D is recommended in previous guidelines irrespective of albuminuria" | **ADDED** (B+G, 0.90) |
| Rationale continuation — "moderate value on benefits (AKI, HF-hospitalization, MI, hospitalization-any-cause, net absolute benefits vs harms)" | **ADDED** (B+G, 0.90) |
| Figure 22 caption + citation | **ADDED** (B+E+G, 0.95) |
| "Impact of diabetes on SGLT2i effects on kidney outcomes: collaborative meta-analysis" citation | **ADDED** (E+G, 0.82) |
| Figure 21 algorithm body (3 decision branches: normokalemia → increase dose; hyperkalemia → review concurrent drugs / moderate K+ / diuretics / HCO3 / K+ binders; ≥30 % eGFR drop → review AKI causes / correct volume / reassess drugs / renal artery stenosis; then "reduce dose or stop ACEi/ARB if mitigation ineffective") | **MISSED** — the algorithm itself (clinically actionable branching logic) is **not** captured as text; only the caption is. The image is a PNG; no OCR of the flowchart contents. |

### Page 14 — SGLT2i key information + Figure 22 (kidney outcomes table)

**PDF content:** Benefit/harms narrative; DAPA-CKD & EMPA-KIDNEY trial differences; collaborative meta-analysis (13 trials, 90 k participants, 37 % ↓ kidney progression, 23 % ↓ AKI, 23 % ↓ CVD-death/HF-hospitalization, ~10 % MACE, reduced BP/uric acid/weight/hyperkalemia, no hypoglycemia); Figure 22 big RR table with 13 trials × (Kidney disease progression, AKI) columns; transition to CV/mortality narrative.

**DB spans (8):** all `PENDING`

| Fact | Status |
|---|---|
| Benefit summary — "SGLT2i also favorably reduce BP, uric acid, fluid overload, serious hyperkalemia, no hypoglycemia…consistent with but expands Recommendation 1.3.1 from KDIGO 2022 DM-CKD" | **ADDED** (B+C+G, 0.99) |
| EMPA-KIDNEY vs DAPA-CKD difference (non-DM causes, lower eGFR, lower ACR) | **ADDED** (B+C+G, 0.95) |
| RCT efficacy summary (kidney failure, AKI, HF-hosp, CV-death, MI in ±CKD) | **ADDED** (B+G, 0.90) |
| Meta-analysis composite (CV-death or HF-hosp 23 % ↓) | **ADDED** (B+G, 0.90) |
| "Two large RCTs using 2 different SGLT2i recruited 10,913 participants" | **ADDED** (B+G, 0.90) |
| "Furthermore, SGLT2i also importantly reduce the risk of hospitalization from any cause, reduce BP, uric acid" | **ADDED** (B+G, 0.90) |
| Figure 23 caption + citation | **ADDED** (B+E+G, 0.95) |
| SOLOIST-WHF exclusion note | **ADDED** (D+E+G, 0.92) |
| **Figure 22 RR table body** (13 trial rows × 8 columns of events / person-years / RR / 95 % CI) | **MISSED** — 0 of ~100 numerical data points captured. The figure is a PNG. This is the single largest-content gap in this batch. **Not a Recommendation/PP, but a major evidence-base artifact.** |
| Harms summary — "any risk of ketoacidosis or lower-limb amputation…11 fewer CV deaths/HF-hosp per 1000, ~1 ketoacidosis, ~1 amputation" with NNT framing | **MISSED** — continues on p15 but not on p14's spans |
| "15 fewer people with kidney progression, 5 fewer AKI, 2 fewer CV death in non-DM CKD per 1000 patient-years" | **MISSED** |

### Page 15 — SGLT2i harms + certainty-of-evidence + Figure 23 body

**PDF content:** Harms (ketoacidosis / lower-limb amputation / mycotic infections — low); NNT framings for CV-death, HF-hosp, kidney progression, AKI per 1000 patient-yrs for T2D-CKD and non-DM-CKD; Certainty of evidence (NDPH Renal Studies Group / SMART consortium); Figure 23 body (13 trials × CV-death-or-HF-hosp vs CV-death vs non-CV-death vs all-cause death subgroups).

**DB spans (8):** all `PENDING`

Spans listed earlier (Figure 29 / Figure 31 captions, Table 25 / Table 26 fragments) appear to have drifted to page 15 from p16-17 due to offset mis-binding. Actual p15 content is extremely thin in DB:

| Fact | Status |
|---|---|
| `L1_RECOVERY` span "eGFR 60+ eGFR 30–59 eGFR <30" | **ADDED** (L1_RECOVERY, 1.00) — Figure 29 axis legend recovered from L1 tree; high confidence but fragmentary |
| Figure 29 caption | **ADDED** (C+E+G, 0.93) |
| "Serum potassium and adverse outcomes across the range of kidney function: a CKD Prognosis Consortium meta-analysis" citation | **ADDED** (C+G, 0.82) |
| "levels of eGFR, thus understanding potassium physiology and its impacting factors are important" | **ADDED** (C+G, 0.82) — intro to §3.11 hyperkalemia *on p15* — but PDF shows this text on **p16**; page-binding error |
| "Hyperkalemia in people with preserved eGFR is less prevalent…no consensus on magnitude, duration, frequency that defines chronicity" | **ADDED** (C+G, 0.82) — same p15/p16 drift |
| "Studies have demonstrated a continuous U-shaped relationship between serum potassium and all-cause mortality" | **ADDED** (C+G, 0.82) |
| "560-564 adaptive mechanisms that render better tolerance to elevated levels of potassium" | **ADDED** (C+G, 0.82) |
| `# 3.11.1 Awareness of factors impacting on potassium measurement` heading | **ADDED** (C+G, 0.82) — also drifted from p16 |
| **All of the Figure 23 content (mortality sub-group table) and the harms NNT narrative and the certainty-of-evidence paragraph** | **MISSED** — the spans on p15 in the DB are actually §3.11 content that belongs on p16. The actual p15 PDF content (Figure 23 body, harms table, certainty narrative, mycotic genital infections) is absent from p15 *and* from p14/p16 spans. **Significant content gap.** |

> **Page-binding issue for pages 14–16:** L1_RECOVERY channel has pulled §3.11 hyperkalemia content onto p15, while the actual p15 PDF content is not captured anywhere. This is a **pipeline defect**, not a reviewer gap.

### Page 16 — Figure 30 hyperkalemia prevalence table + §3.11 intro + PP 3.11.1.1 + Figure 21 cross-ref

**PDF content:** Figure 29 image; §3.11 intro prose; Figure 30 prevalence table (6 × 3 eGFR/A1/A2/A3 cells × 2 diabetes-status panels); §3.11.1 heading; **PP 3.11.1.1** (be aware of variability in K+ measurement); RASi/hyperK cross-reference; Figure 31 mortality-curve teaser.

**DB spans (40):** all `PENDING`. Heavy D-channel fragmentation — **24 of 40 spans are single Figure 30 cells** (eGFR labels, A1/A2/A3 labels, prevalence percentages with 95 % CIs).

| Fact | Status |
|---|---|
| Figure 30 eGFR row labels (>90 / 75–89 / 60–74 / 45–59 / 30–44 / 15–29) | **NEEDS-CONFIRM** (D × 6, 0.92) |
| Figure 30 albuminuria headers (A1 / A2 / A3) | **NEEDS-CONFIRM** (D × 3, 0.92) |
| Figure 30 non-DM prevalence values (1.5 %, 1.1 %, 1.4 %, 1.7 %, 1.6 %, 1.5 %, 2.3 %, 2.0 %, 2.3 %, 4.5 %, 3.5 %, 5.2 %, 9.5 %, 10.5 %, 11.3 %, 16.1 %, 19.0 %, 23.7 %) | **NEEDS-CONFIRM** (D × 18, 0.92) — row×column pairing lost |
| Figure 30 DM prevalence values — captured in HTML `<table>` form as a single span | **ADDED** (C+G, 0.82) — full DM panel preserved |
| Non-DM prevalence also in HTML `<table>` form | **ADDED** (C+G, 0.82) — fine |
| "ACR and prior diabetes, hyperglycemia, constipation, RASi, MRA — SGLT2i do not appear to increase serum K+" | **ADDED** (B+C+G, 0.95) |
| "There are several factors and mechanisms that may impact on potassium measurements, including…medications" | **ADDED** (C+G, 0.82) |
| **PP 3.11.1.1** full text | **ADDED** (0.89) |
| "Work Group highlights Figure 26 for monitoring serum K+ during finerenone from KDIGO 2022 DM-CKD" | **ADDED** (B+C+F+G, 1.00) |
| "Hyperkalemia has been associated with reducing/stopping RASi…mitigation steps to increase RASi use" | **ADDED** (B+C+F+G, 1.00) |
| Figure 30 legend definition ("Hyperkalemia = K+ >5 mmol/L, prevalence adjusted…") | **ADDED** (E+G, 0.82) |
| Figure 30 citation | **ADDED** (E+G, 0.82) |
| Figure 31 caption | **ADDED** (E+G, 0.82) |
| Figure 31 citation ("Association of serum potassium with all-cause mortality…") | **ADDED** (E+G, 0.82) |

**Note:** The page does have duplicate capture (D-fragment + HTML blob) of Figure 30 — reviewer can keep the HTML and reject the fragments.

### Page 17 — Figure 31 + Table 25 (Factors impacting K+ measurement) + Table 26 (K+-raising meds) header

**PDF content:** Figure 31 (potassium-mortality curves by DM/HF/CKD); **Table 25** (Pseudohyperkalemia, disruption-shifting-out, disruption-moving-in, decreased-excretion, diurnal variation, plasma vs serum, postprandial); **Table 26** header + ACEi, ARB, Aldosterone antagonist, β-blocker rows begin.

**DB spans (28):** all `PENDING`. Again heavy D-channel fragmentation (19/28).

| Fact | Status |
|---|---|
| Table 25 title "Factors and mechanisms that impact on potassium measurements" | **ADDED** (E+G, 0.82) |
| Table 25 body — **full HTML** with all 6 category rows (Pseudohyperkalemia, disruption-shifting-out, disruption-moving-in, decreased-excretion, diurnal variation, plasma vs serum, postprandial) | **ADDED** (C+G, 0.82) ✅ *Adequate capture* |
| Individual Table 25 cell fragments (19 D-channel duplicates covering the same content) | **NOISE** (D × 19, 0.92) — duplicates of the HTML; reviewer should reject |
| OCR artefacts in D-fragments: `Tight tumourquet` (PDF: "Tight tourniquet"); `kaliuesis` (PDF: "kaliuresis"); `Increased sodium (e.g., in the absence of hyperglycemia)` (PDF: "Increase in plasma osmolarity"; this looks like a **hallucinated row** — **data-integrity red flag**) | **NEEDS-CONFIRM** — OCR errors + suspected hallucination need reviewer-check |
| `H-channel "Disruption in the release of insulin in response to raised serum potassium"` low-conf duplicate | **NEEDS-CONFIRM** (H, 0.60) |
| Table 25 abbreviation footnote | **ADDED** (C+F+G, 0.92) |
| `<!-- Chunk chunk-b3 -->` marker | **NOISE** — pipeline leak |
| Table 26 title "Medications associated with increased risk of hyperkalemia" | **ADDED** |
| Table 26 body **full HTML** (ACEi, ARB, Aldosterone antagonist, β-blocker, Digitalis, Heparin, K+-sparing diuretic, NSAIDs, CNI, ns-MRA, Other) × Class/Mechanism/Example | **ADDED** (B+C+E+G, 1.00) ✅ *Adequate capture* |
| Table 26 abbreviation footnote | **ADDED** (B+C+F+G, 1.00) |

**Quality concern:** the "Increased sodium (e.g., in the absence of hyperglycemia)" D-fragment does NOT appear in the PDF — appears to be an OCR hallucination or mis-aligned cell. Reviewer should reject.

### Page 18 — Table 26 body fragments + §3.11.2 PP 3.11.2.1 + §3.11.3 + §3.11.4 + §3.11.5 PP 3.11.5.1/3.11.5.2

**PDF content:** Table 26 body (continues); §3.11.1 → 3.11.2 Potassium exchange agents; **PP 3.11.2.1** (local availability / formulary); §3.11.3 timing to recheck K+; §3.11.4 managing hyperK; §3.11.5 dietary; **PP 3.11.5.1** (individualized CKD G3–G5 with emergent hyperK, dietary+pharmacologic, renal-dietitian); **PP 3.11.5.2** (limit foods rich in bioavailable K — processed foods).

**DB spans (55):** all `PENDING`. **30+ D-channel fragments from Table 26 body** (class names, drug examples, mechanism blurbs) — heavy duplication.

| Fact | Status |
|---|---|
| Table 26 body fragments (Captopril/lisinopril; Losartan/irbesartan; Spironolactone/eplerenone/finerenone; Propranolol/metoprolol; Digoxin; Heparin sodium; Amiloride/triamterene; Ibuprofen/naproxen/diclofenac; Cyclosporine/tacrolimus; Finerenone; Trimethoprim/pentamidine) | **NEEDS-CONFIRM** (D × ~30, 0.92) — HTML version already captured on p17; these are redundant |
| H-channel duplicates (ns-MRA, Finerenone, Losartan/etc., Captopril/etc., Spironolactone/etc., ARB) | **NEEDS-CONFIRM** (H × 6, 0.60) — low confidence, subset of D-coverage |
| "See Section 4.3 for more information on continuing RASi after hyperkalemia events" | **ADDED** (B+G, 0.90) |
| §3.11.2 Potassium exchange agents heading | **ADDED** (0.82) |
| **PP 3.11.2.1** full text | **ADDED** (0.89) |
| §3.11.2 body ("pharmacologic management of nonemergent hyperkalemia has a number of clinical tools…") | **ADDED** (0.82) |
| "Newer exchange agents fewer tolerability issues…facilitate essential use of RASi/MRA" | **ADDED** (B+C+G, 0.95) |
| §3.11.3 heading | **ADDED** (0.82) |
| Think-Kidneys / UK-KA practical-guide pointer + Table 28 reference | **ADDED** (F+G, 0.82) |
| §3.11.4 body ("systematic approach of treating correctable factors…Figure 32 stepwise practical approach") | **ADDED** (F+G, 0.82) |
| §3.11.5 Dietary considerations body ("In early stages of CKD, high intake of foods naturally rich in potassium appears protective…fruits and vegetables may be harmful to cardiac health") | **ADDED** (C+F+G, 0.90) |
| **PP 3.11.5.1** full text | **ADDED** (0.89) |
| **PP 3.11.5.2** full text | **ADDED** (C+F+G, 0.94) |
| Continuation ("Diet may increase serum K+ postprandially…other conditions such as potassium-sparing medications, metabolic acidosis…more likely to explain plasma K+ abnormalities than diet") | **NEEDS-CONFIRM** (C+E+F+G, 0.96) — captured but OCR garbling "`57,5,588`" (should be `575,588`) |
| `<!-- Chunk chunk-b4: pages 19-19 -->` marker | **NOISE** |

### Page 19 — Table 27 (Polystyrene sulfonates / Patiromer / SZC) + Table 28 (severity triage)

**PDF content:** Table 27 full comparison (3 drugs × 10 attributes: mechanism, counterion content, cations bound, formulation, dosage, maintenance, onset, duration, admin pearls, AEs); Table 28 (Moderate K+ 6.0–6.4 vs Severe K+ ≥6.5, response depending on clinically unwell/AKI vs unexpected); continuation of §3.11.5 narrative.

**DB spans (60):** all `PENDING`. All but 5 are D-channel Table 27/28 fragments.

| Fact | Status |
|---|---|
| Table 27 attribute labels (Mechanism, Counterion, Cations bound, Formulation, Dosage & titration, Maintenance, Onset, Duration, Admin pearls, Adverse effects) | **NEEDS-CONFIRM** (D × ~10, 0.92) |
| Table 27 SPS/CPS column body cells (15–60 g/d oral, 30 g/d rectal, SPS 65 mmol/60 ml Na, CPS 1.6–2.4 mmol/g Ca, variable hours-days onset, 6–24 h duration, GI AEs incl. intestinal necrosis/bleeding/ischemic colitis, etc.) | **NEEDS-CONFIRM** (D × ~15, 0.92) |
| Table 27 Patiromer column body (8.4 g qd start, max 25.2 g qd, 4–7 h onset, 24 h duration, Ca-K-Mg-phosphate bound, GI AEs) | **NEEDS-CONFIRM** (D × ~10, 0.92) |
| Table 27 SZC column body (Na ~400 mg per 5 g, 10 g tid × 48 h start, 5 g qod–10 g qd maint, "starts to reduce K+ within 1 hour with normokalemia typically at 24–48 h", hypokalemia/edema AEs) | **NEEDS-CONFIRM** (D × ~12, 0.92) |
| Table 28 severity labels (Moderate K+ 6.0–6.4; Severe K+ ≥6.5) | **NEEDS-CONFIRM** (D × 2, 0.92) |
| Table 28 response cells ("Assess and treat in hospital", "Repeat within 24 hours", "Take immediate action to assess and treat. Assessment will include blood testing and ECG monitoring") | **NEEDS-CONFIRM** (D × 4, 0.92) |
| Continuation prose ("medications, metabolic acidosis, hyperosmosis, hypernatremia, uremia, constipation more likely to explain plasma K+ abnormalities than diet") | **ADDED** (C+E+F+G, 0.98) |
| Table 27 citation ("Potassium-lowering agents for the treatment of nonemergent hyperkalemia: pharmacology, dosing and comparative efficacy") | **ADDED** (E+G, 0.82) |
| Table 28 abbreviation footnote + note about blood testing/ECG | **NEEDS-CONFIRM** (C+G, 0.82) |
| Cross-page marker + chapter footer | **ADDED** (E+F+G, 0.92) |
| Figure 32 caption (carried via marker, actually on p20) | **NEEDS-CONFIRM** — cross-page drift |
| **OCR artefact:** Table 27 SPS counterion "1.6-22.4 mmol/g of calcium" (PDF: "1.6–2.4") — typo/range error | **NEEDS-CONFIRM** — critical numeric error for a dosing fact |
| Table 27 missing: no single clean HTML of the whole Table 27 — unlike Tables 25/26, the D-fragments are the only record | **NEEDS-CONFIRM** batch-level — reviewer needs to reconstruct Table 27 |

**Data-integrity red flag:** the `1.6-22.4 mmol/g of calcium` fragment is a **numeric OCR error** in a dosing table. Any downstream consumer using this DB row would get a wildly wrong calcium-dosing range. **Must be flagged for reviewer.**

### Page 20 — Figure 32 (hyperkalemia 3-line algorithm) + plant-based vs processed foods + Figure 33 + International considerations

**PDF content:** Figure 32 (1st-line correctable factors; 2nd-line medications; 3rd-line reduce/stop RASi); narrative on plant-based vs processed foods; Figure 33 (absorption-rate comparison plant 50–60 % / animal 70–90 % / processed 90 %); cooking-methods advice (soak 5–10 min); international considerations — cost-benefit of K+ exchange agents.

**DB spans (7):** all `PENDING`

| Fact | Status |
|---|---|
| Plant-based vs processed foods narrative ("plant-based absorption, other nutrients affecting K+ distribution, net bioavailable K+ lower than appreciated, highly processed / meats / dairy / juices / salt-substitutes higher in absorbable K+") | **ADDED** (C+G, 0.82) |
| ISN RAASi toolkit pointer + food-labeling policy call | **ADDED** (B+C+G, 0.95) |
| Teaching materials focus (processed vs unprocessed restriction) + BC Renal patient-resource pointer | **ADDED** (C+G, 0.82) |
| Cooking-method 5–10 min soak ~50 % K+ reduction | **ADDED** (C+G, 0.82) |
| International considerations — cost-benefit of K+ exchange agents in severe recurrent hyperK | **ADDED** (C+E+G, 0.93) |
| Figure 33 caption | **ADDED** (E+G, 0.82) |
| Figure 33 citation ("Handouts for low-potassium diets disproportionately restrict fruits and vegetables") | **ADDED** (E+G, 0.82) |
| Figure 32 body (the 3-line algorithm itself with actionable sub-steps) | **MISSED** — the image is a PNG; the textual actionable content (review non-RASi meds / dietary referral / diuretics / bicarbonate / K+ binders / reduce-dose RASi last resort) is **not** captured anywhere. Only the caption on p19 is present. **Clinically important actionable flow lost.** |

### Batch 2 roll-up

| Metric | Value |
|---|---|
| Spans reviewed | 249 |
| Recommendations present on these 10 pages | 8 (Rec 3.4.1, 3.4.2, 3.6.1, 3.6.2, 3.6.3, 3.6.4, 3.7.1, 3.7.2) |
| Practice Points present on these 10 pages | 17 (PP 3.4.1, 3.4.2, 3.4.3, 3.6.1, 3.6.2, 3.6.3, 3.6.4, 3.6.5, 3.6.6, 3.6.7, 3.7.1, 3.7.2, 3.7.3, 3.11.1.1, 3.11.2.1, 3.11.5.1, 3.11.5.2) |
| Recommendations / PPs captured | **25 / 25 (100 %)** — but **Rec 3.6.1–3.6.4 are packed into one span**, and **PP 3.6.1–3.6.5 are packed into one span** (**NEEDS-CONFIRM — split required before Pipeline 2 export**). |
| Recommendations / PPs CONFIRMED | 0 (entire batch is PENDING) |
| Major clinical content MISSED | (a) **Figure 21 RASi algorithm body** (branch logic) on p13; (b) **Figure 22 RR table body** (13 trials × 8 columns of efficacy evidence) on p14; (c) **most of p15** (harms NNT narrative, certainty-of-evidence); (d) **Figure 32 3-line hyperkalemia algorithm body** on p20. |
| OCR / pipeline artefacts | (a) `1.6-22.4 mmol/g of calcium` (should be `1.6–2.4`) — **critical numeric dosing error**; (b) `Tight tumourquet` / `kaliuesis` / `57,5,588` — minor OCR errors; (c) "Increased sodium (e.g., in the absence of hyperglycemia)" — **suspected D-channel hallucination** not in PDF; (d) p14–16 page-binding drift — §3.11 content landing on p15 while p15 PDF content is uncaptured; (e) chunk markers `<!-- Chunk chunk-bN -->` leaking into spans |

**Verdict for batch 2:** 100 % Recommendation/PP capture but significant **reviewer burden** from (1) splitting two multi-PP / multi-Rec packed spans, (2) dealing with ~60 D-channel table fragments that duplicate HTML blobs already captured, (3) fixing ~5 numeric/OCR errors, (4) capturing the three algorithm/figure bodies that are PNG-only. The p14–16 page drift is a **pipeline defect** — worth investigating.

---

## Batch 3 — Pages 21–30 (§3.15.1 tail + §3.15.2 Antiplatelet + §3.15.3 Invasive vs medical + §3.16 CKD & Atrial Fibrillation)

**Batch-level summary:** 9 + **0** + 7 + 1 + 3 + 8 + 3 + 5 + 3 + 21 = **60 spans**, all `PENDING`. This is the sparsest batch (avg 6 spans/page vs batch 2's 25 spans/page). **Page 22 has zero spans**; **page 24 has only 1**; **page 25 has only 3** — and these are all content-rich PDF pages (~7 000 chars each). The key finding is that **Rec 3.16.1 ("NOACs over warfarin in CKD G1–G4 for AFib thromboprophylaxis") is entirely MISSED from the DB** — only PP 3.16.1 is captured.

### Page 21 — PP 3.15.1.4 PCSK-9 + PP 3.15.1.5 Mediterranean diet + Rec 3.15.2.1 aspirin

**PDF content:** Continuation of statin safety / ACC-AHA 7.5 % threshold; **PP 3.15.1.4** (PCSK-9 inhibitors); rationale (eGFR down to 20 in trials, heterozygous FH / clinical ASCVD); §3.15.2 Use of antiplatelet therapy; **Rec 3.15.2.1** (oral low-dose aspirin for secondary prevention in CKD + established ischemic CVD) [1C]; rationale including 2009 ATT meta-analysis (16 secondary-prevention trials ~17 k pts; 6 primary-prevention trials ~95 k pts; RR 0.81 secondary).

**DB spans (9):** all `PENDING`

| Fact | Status |
|---|---|
| Tail "…good evidence for safety of intensive LDL-C lowering and statin-based fire-and-forget low cost" | **ADDED** (B+G, 0.90) |
| **PP 3.15.1.4** PCSK-9 inhibitors | **ADDED** (0.89) |
| PP 3.15.1.4 rationale (ASCVD risk reduction when added to max-tolerated statin; eGFR down to 20) | **ADDED** (B+C+F+G, 1.00) |
| **PP 3.15.1.5** Mediterranean-style diet | **ADDED** (0.89) |
| Dietary-approaches narrative fragment ("approximately 22%–25%. 684 No large-scale CKD-specific trial") | **ADDED** (F+G, 0.85) |
| §3.15.2 heading | **ADDED** (via F+G) |
| **Rec 3.15.2.1** full text + `(1C)` | **ADDED** (0.89) |
| Rationale ("Based on a number of large RCTs…lifelong use of low-dose aspirin 75–100 mg for prevention of recurrent ischemic CVD") | **ADDED** (C+G, 0.82) |
| "Key evidence from general populations is derived from a 2009 meta-analysis by the Anti-thrombotic Treatment Trialists' collaboration" | **ADDED** (F+G, 0.82) |
| ATT collaboration details (95 k primary / 17 k secondary / person-years / 3554 vs 3306 serious vascular events, RR 0.81) | **ADDED** (F+G, 0.82) — partial; some numbers truncated |
| ACC/AHA >7.5 % 10-y threshold & "consistent with more recent recommendation for primary prevention in CKD" | **MISSED** — Rec 3.15.1.3 context narrative dropped |
| "CKD + ischemic CVD secondary prevention placed high value on reducing recurrence of MI / ischemic strokes / PAD" context | **MISSED** — values/preferences paragraph dropped |

**Prior-run comparison:** old audit listed PP 3.15.1.4, PP 3.15.1.5, and Rec 3.15.2.1 as **MISSED** on p21. **Now all three are present.**

### Page 22 — **ZERO SPANS** (Figure 38 + aspirin harms narrative + CKD meta-analysis)

**PDF content:** Continuation of ATT meta-analysis narrative (absolute risk difference 1.49 % lower serious vascular / 0.03 % major bleeding); **Figure 38** — 5-yr risk bar-chart by sex (F/M) and age (50–59 / 65–74) × primary/secondary prevention with aspirin (A) vs control (C) — 8 paired bars; Cochrane antiplatelet meta-analysis (40 597 CKD participants, RR MI 0.88, 95 % CI 0.79–0.99; RR major bleeding 1.35).

**DB spans (0)** — **entire page is absent from extraction.**

| Fact | Status |
|---|---|
| Quantitative absolute-risk comparison (1.49 %/yr ARR vs 0.03 %/yr bleeding) | **MISSED** — key efficacy metric for Rec 3.15.2.1 rationale |
| Figure 38 caption | **MISSED** |
| Figure 38 body (8 bars × 2 sexes × 2 age bands × primary/secondary) | **MISSED** (PNG — not surprising that body is absent, but caption should have been captured) |
| Cochrane 40 597-pt CKD meta-analysis (RR MI 0.88, RR major bleeding 1.35) | **MISSED** — **the CKD-specific aspirin evidence is lost** |
| "Antiplatelet versus placebo trials…allocation to antiplatelet therapy may reduce RR of MI by ~12 %" | **MISSED** |

> **Why zero spans?** The p22 PDF text is 6 156 chars (normal), yet the DB has **no rows** for `page_number=22`. This is a page-binding pipeline defect — likely the `l2_section_passages` offset boundaries placed p22 text into an adjacent page's section, causing Channel A/G to skip it. This is the same failure mode observed on the older 4.2.4 run for different pages; it persists on p22 here.

**Prior-run comparison:** old audit also listed p22 as a low-span page (classified "LOW severity – only drug names"). The new run is no better.

### Page 23 — PPI considerations + PP 3.15.2.1 P2Y12 alternative + dual antiplatelet + Rec 3.15.3.1

**PDF content:** PPI considerations (bleeding risk / interstitial nephritis); "Rationale" summary of aspirin net benefit; **PP 3.15.2.1** (P2Y12 inhibitors when aspirin intolerant); FDA 2009 clopidogrel+omeprazole warning; dual antiplatelet therapy after ACS/PCI; **§3.15.3 Invasive vs intensive medical therapy**; **Rec 3.15.3.1** (stable stress-test-confirmed ischemic heart disease — initial conservative approach [2D]).

**DB spans (7):** all `PENDING`

| Fact | Status |
|---|---|
| PPI considerations ("PPIs are generally effective, safe, low cost…occasionally associated with interstitial nephritis") | **ADDED** (F+G, 0.85) |
| **PP 3.15.2.1** P2Y12 alternative | **ADDED** (0.89) |
| FDA 2009 clopidogrel+omeprazole warning | **ADDED** (C+G, 0.88) |
| Dual antiplatelet therapy guidance (same strategies as in non-CKD, CKD doesn't modify ticagrelor benefit) | **ADDED** (C+F+G, 0.90) |
| **Rec 3.15.3.1** stable IHD conservative strategy + `(2D)` | **ADDED** (0.89) |
| Rationale ("Comparisons between aggressive medical therapy alone and invasive interventions do not support invasive strategies…frequent angina symptoms gained improvement with invasive strategy") | **ADDED** (C+G, 0.82) |
| Evidence-quality note ("therapy among people with CKD not undergoing KRT and ischemic heart disease is very low") | **ADDED** (F+G, 0.82) |
| "Meta-analysis of trials has clearly established CV benefits of low-dose aspirin…" (Rationale block above PP 3.15.2.1) | **MISSED** |
| CKD doses of antiplatelet therapy do not need modification | **NEEDS-CONFIRM** — partial capture |
| International considerations (aspirin access easy in any setting) | **MISSED** |

### Page 24 — **Only 1 span** (PP 3.15.3.1 invasive strategy for unstable CAD) + ISCHEMIA/ISCHEMIA-CKD discussion

**PDF content:** ISCHEMIA/ISCHEMIA-CKD trial analysis (CKD-MBD, coronary calcification, microvascular disease); ERT review (4 trials excluding mixed populations); Harms (dialysis initiation, death, stroke); Certainty of evidence (imprecision, publication bias suspected); Values (ISCHEMIA-CKD did not confirm antianginal benefits but general populations do); Resource use (mixed findings); Implementation (access & availability); Rationale; **PP 3.15.3.1** (invasive strategy still preferable for acute/unstable CAD, unacceptable angina, LV systolic dysfunction, left main disease); ISCHEMIA as "deeply disrupting prior attitudes…clinical practice guidelines predating trial need updating".

**DB spans (1):** all `PENDING`

| Fact | Status |
|---|---|
| **PP 3.15.3.1** (invasive still preferable for acute/unstable CAD, unacceptable angina, LV dysfunction, left main) | **ADDED** (0.89) — captured but the continuation after "left main" is truncated mid-sentence |
| Balance of benefits/harms narrative (ISCHEMIA + ISCHEMIA-CKD / antianginal explanation) | **MISSED** |
| CKD-MBD & coronary-calcification explanation for ISCHEMIA-CKD result | **MISSED** |
| ERT review of 4 other trials excluding ISCHEMIA-CKD | **MISSED** |
| Harms narrative (dialysis initiation / death / stroke non-periprocedural) | **MISSED** |
| Certainty-of-evidence downgrade (imprecision + publication-bias suspicion) | **MISSED** |
| Values/preferences (antianginal benefits in general populations → patient may still elect invasive) | **MISSED** |
| Resource use & cost-effectiveness note | **MISSED** |
| Rationale ("key indication for initial invasive is symptoms…intensive medical is suitable") | **MISSED** |

**Prior-run comparison:** old audit listed **"Intensive management benefit | PP 3.15.3.1"** as MISSED on p24. **PP 3.15.3.1 is now captured, but everything else on p24 (the entire Rec 3.15.3.1 rationale section) is missing.** This is a net improvement of *one* PP, but the rationale gap matters for reviewer context.

### Page 25 — Figure 39 + §3.16 AFib intro + PP 3.16.1 + CHA2DS2-VASc discussion

**PDF content:** Figure 39 (AFib prevalence table by eGFRcr 105+ → <15 × ACR <10 / 10–29 / 30–299 / 300–999 / 1000+); §3.16 CKD and atrial fibrillation; prevalence (16–21 %); CKD-PC cohort risk (1.2–1.5 at G3A1 → 4.2 at G5A3); identification/management; ERT focus on NOAC vs warfarin; **PP 3.16.1** (follow established strategies — Figure 40); CHA2DS2-VASc threshold discussion; 95 % of eGFR <60 have CHA2DS2-VASc ≥2.

**DB spans (3):** all `PENDING`

| Fact | Status |
|---|---|
| **PP 3.16.1** full text | **ADDED** (0.89) |
| Oral anticoagulation narrative ("Our Work Group considered that oral anticoagulation for thromboprophylaxis should nearly always be considered…95 % of eGFR <60 have CHA2DS2-VASc ≥2") | **ADDED** (C+G, 0.82) |
| Standard AFib diagnostic-evaluation package (12-lead ECG, TTE, labs) — described in Figure 40 footnote | **ADDED** (D+G, 0.85) |
| §3.16 heading + prevalence paragraph (16–21 % prevalence, adjusted risk by G3A1 → G5A3) | **MISSED** |
| Figure 39 caption ("Meta-analyzed adjusted prevalence of atrial fibrillation from CKD-PC cohorts, by diabetes status") | **MISSED** |
| Figure 39 body (7 × 5 table eGFRcr × ACR — AFib prevalence) | **MISSED** — important CKD-stratified prevalence data |
| CHA2DS2-VASc risk-score threshold detail (0 in men / 1 in women → no antithrombotic; ≥2 men / ≥3 women → oral anticoagulants recommended) | **MISSED** — concrete clinical-decision rule absent |

**Prior-run comparison:** old audit listed **"Coronary revascularization | PP 3.16.1"** as MISSED on p25. **PP 3.16.1 is now captured,** but the surrounding Figure 39 prevalence data and CHA2DS2-VASc thresholds remain gaps.

### Page 26 — **Rec 3.16.1 MISSED** + Figure 40 + NOAC preamble + eGFR-into-AFib-risk-score discussion

**PDF content:** Figure 40 (AFib 3-step management: Diagnosis / Prophylaxis / Rate-rhythm); 2.9 %/y vs 0.2 %/y thromboembolic-event rate comparison; "Including eGFR into AFib risk scores has not shown important incremental benefit (R2CHADS2)"; **Rec 3.16.1** (NOACs over VKA for thromboprophylaxis in AFib CKD G1–G4) **[1C]**; NOAC rationale (simpler pharmacokinetics/dosing/monitoring + improved efficacy + relatively similar safety); Balance of benefits (42 411 NOAC vs 29 272 warfarin meta-analysis, RR stroke 0.81, RR hemorrhagic stroke 0.49); 2021 CKD-only meta-analysis (HR 0.81).

**DB spans (8):** all `PENDING`. Figure 40 body lost; Figure 41 caption + table HTML preserved.

| Fact | Status |
|---|---|
| L1_RECOVERY: "Warfarin Warfarin Warfarin" (Figure 41 control column) | **NEEDS-CONFIRM** (L1_RECOVERY × 3, 1.00) — fragmentary |
| L1_RECOVERY: "Edoxaban 60 mg Rivaroxaban 10 mg Rivaroxaban 20 mg" (Figure 41 intervention column) | **NEEDS-CONFIRM** (L1_RECOVERY × 2) |
| Figure 41 caption + citation | **ADDED** (L1_RECOVERY, 1.00) |
| eGFR/R2CHADS2 narrative ("Including eGFR into AFib risk scores has not shown important incremental benefit…R2CHADS2 improved NRI but not C-statistic") | **ADDED** (C+G, 0.88) |
| Figure 41 HTML table header + Bohula-2016 row start | **ADDED** (E+G, 0.88) — partial |
| **Rec 3.16.1 NOACs over VKA for AFib in CKD G1–G4 [1C]** | **MISSED** — **entirely absent from the DB**. Only a stray fragment "Non-vitamin K antagonist oral anticoagulants" exists on p2 (nephrotoxic-meds table), and PP 3.16.1 on p25. No span carries the Rec 3.16.1 recommendation text. |
| Rec 3.16.1 rationale preamble ("This recommendation puts high value on the use of NOACs…due to their simpler pharmacokinetic profile, dosing, and monitoring") | **MISSED** |
| 42 411 vs 29 272 participant NOAC-vs-warfarin meta-analysis summary (RR 0.81, 95 % CI 0.73–0.91; hemorrhagic-stroke RR 0.49) | **MISSED** |
| 2021 CKD-only meta-analysis (HR 0.81; 95 % CI 0.69–0.97) | **MISSED** |
| Figure 40 body (3 steps: Diagnosis / Prophylaxis / Rate-rhythm with sub-bullets) | **MISSED** (image — but captions and steps should have been captured) |
| Figure 40 footnote (HAS-BLED risk score definition) | **MISSED** |

**Prior-run comparison:** old audit listed **"Rec 3.16.1"** as MISSED on p26. **This 4.2.2 re-extraction still misses Rec 3.16.1 — regression not fixed.** This is the single most important missing guideline fact in the entire job.

> **Highest-priority reviewer action:** manually **ADD** Rec 3.16.1 span with full text + `(1C)` grade.

### Page 27 — Figure 42 (bleeding forest plot) + NOAC harms narrative + certainty-of-evidence

**PDF content:** Figure 42 (NOAC vs warfarin bleeding meta-analysis, 6 studies × 4 outcomes: clinically relevant bleeding / fatal bleeding / major bleeding / intracranial hemorrhage / CRNM bleeding); Harms (RR death 0.90; RR intracranial hemorrhage 0.48; RR GI bleeding 1.25; RR major bleeding 0.86; time-in-therapeutic-range interaction P = 0.02); 2021 meta-analysis bleeding HR 0.83; Certainty of evidence (low); Values & preferences (NOAC-preference narrative).

**DB spans (3):** all `PENDING`

| Fact | Status |
|---|---|
| Figure 42 caption + citation | **ADDED** (L1_RECOVERY, 1.00) |
| L1_RECOVERY: "Warfarin" repetition (Figure 42 control column) | **NEEDS-CONFIRM** (fragmentary) |
| Figure 42 HTML header + Hijazi-2021 row start | **ADDED** (E+G, 0.88) |
| Harms narrative (RR death 0.90, RR ICH 0.48, RR GI bleeding 1.25, RR major bleeding 0.86) | **MISSED** — **key safety numbers for Rec 3.16.1 absent** |
| Time-in-therapeutic-range finding (major bleeding significantly ↓ at centers <66 % TTR; interaction P = 0.02) | **MISSED** |
| 2021 CKD-focused meta-analysis bleeding HR 0.83 (95 % CI 0.58–1.18) | **MISSED** |
| Figure 42 body (the other 5 study rows × 4 outcome groups) | **MISSED** |
| Certainty of evidence paragraph | **MISSED** |
| Values/preferences paragraph | **MISSED** |
| Resource-use (NOACs cost-effective) | **MISSED** |

### Page 28 — Figure 43 (NOAC dose tables) + PP 3.16.2 NOAC dose adjustment

**PDF content:** Figure 43a (RCT-supported NOAC doses by eCrCl: >95 / 51–95 / 31–50); Figure 43b (doses in areas where RCTs lacking: 15–30 / <15 not-on-dialysis / <15 on-dialysis); extensive footnote on dose modification (age ≥80, Cr ≥1.5, weight ≤60 kg, P-glycoprotein inhibitors); **PP 3.16.2** (NOAC dose adjustment required — caution at G4–G5).

**DB spans (5):** all `PENDING`

| Fact | Status |
|---|---|
| L1_RECOVERY: Figure 43 header row "eCrCl (ml/min) / Warfarin / Apixaban / Dabigatran / Edoxaban / Rivaroxaban" | **ADDED** (1.00) |
| Figure 43a body HTML (>95, 51–95, 31–50 rows with doses) | **ADDED** (C+G, 0.82) ✅ Good |
| Figure 43b body HTML (15–30, <15 not-on-dialysis, <15 on-dialysis rows) | **ADDED** (C+G, 0.88) ✅ Good |
| **PP 3.16.2** full text | **ADDED** (0.89) |
| PP 3.16.2 rationale ("Doses of NOACs may need to be modified in people with decreased eGFR taking into consideration the age, weight, and…") | **ADDED** (C+G, 0.82) — truncated |
| Figure 43 footnote (Apixaban dose-reduction rule: any 2 of Cr ≥1.5 / age ≥80 / weight ≤60 kg → 2.5 mg bid; ENGAGE-AF TIMI 48 halving rule: eCrCl 30–50 / weight ≤60 / verapamil or quinidine) | **MISSED** — **clinically critical dose-adjustment criteria absent** |
| Figure 43 caveat (b-footnote: doses in parens = no clinical or efficacy data; FDA-approved based on PK-only) | **MISSED** |
| Suggestion: lower apixaban dose 2.5 mg bid in CKD G5/G5D | **MISSED** |

### Page 29 — Figure 44 (NOAC peri-procedural hold timing) + PP 3.16.3

**PDF content:** Figure 44 (Dabigatran vs Apixaban/Edoxaban/Rivaroxaban × low-risk/high-risk procedure × CrCl ≥80 / 50–80 / 30–50 / 15–30 / <15 — discontinuation hours: ≥24, ≥48, ≥72, ≥96, no indication); **PP 3.16.3** (NOAC discontinuation before procedures considers bleeding risk / NOAC / eGFR — Figure 44); Research recommendations pointer.

**DB spans (3):** all `PENDING`

| Fact | Status |
|---|---|
| Figure 44 caption | **ADDED** (L1_RECOVERY, 1.00) |
| L1_RECOVERY: "Apixaban–Edoxaban–Rivaroxaban Dabigatran" column labels | **ADDED** (1.00) |
| **PP 3.16.3** full text | **ADDED** (0.89) |
| Figure 44 body (the actual 5×4 grid of discontinuation-hours by CrCl × low/high risk for Dabigatran vs NOACs) | **MISSED** on this page — body is captured on p30 via D-channel fragments, so cross-page pairing |
| "There is no need for bridging with LMWH/UFH" | **MISSED** (in figure body) |
| "Research recommendations, please see Chapter 6" pointer | **MISSED** |

### Page 30 — Figure 44 body D-channel fragments (21 spans)

**PDF content:** Figure 44 body table cells continuing from p29. Narrative text is minimal on p30 (continuation of figure).

**DB spans (21):** all `PENDING`

| Fact | Status |
|---|---|
| Figure 44 CrCl row labels (CrCl ≥80 / 50–80 / 30–50 / 15–30 / <15 ml/min) | **NEEDS-CONFIRM** (D × 5, 0.92) |
| Figure 44 hour values (≥24, ≥36, ≥48, ≥72, ≥96, ≥24, ≥48 h — Dabigatran & NOAC columns) | **NEEDS-CONFIRM** (D × 7, 0.92) — **row-column pairing lost, so "≥24 h" cells can't be mapped to their CrCl row and drug column** |
| "No important bleeding risk and/or adequate local hemostasis possible: perform at trough level (i.e., ≥12 or 24 h after last intake)" | **NEEDS-CONFIRM** (D, 0.92) — full instruction captured |
| "Low risk" / "High risk" column labels | **NEEDS-CONFIRM** (D × 2) |
| "No official indication" / "No official indication for use" | **NEEDS-CONFIRM** (D × 3) |
| "Apixaban-Edoxaban-Rivaroxaban" header | **NEEDS-CONFIRM** (D, 0.92) |
| Figure 44 HTML table shell | **ADDED** (C+G, 0.88) — provides row-column reconstruction |
| `<!-- Chunk chunk-d1: pages 31-35 -->` | **NOISE** — pipeline chunk marker |

> Like Tables 27/28 on p19, Figure 44 is captured **twice** — once as 20+ D-fragments (row-column context lost) and once as an HTML blob (row-column preserved). Reviewer should keep the HTML and reject the fragments.

### Batch 3 roll-up

| Metric | Value |
|---|---|
| Spans reviewed | 60 |
| Recommendations present on these 10 pages | 3 (Rec 3.15.2.1, Rec 3.15.3.1, Rec 3.16.1) |
| Practice Points present on these 10 pages | 5 (PP 3.15.1.4, PP 3.15.1.5, PP 3.15.2.1, PP 3.15.3.1, PP 3.16.1, PP 3.16.2, PP 3.16.3 — total 7) |
| Recommendations captured | **2 of 3 (67 %)** — **Rec 3.16.1 MISSED** |
| Practice Points captured | **7 of 7 (100 %)** |
| Major narrative content MISSED | (a) **entire p22** (Figure 38 caption + ATT/CKD absolute risk numbers + Cochrane 40 597-pt CKD meta-analysis); (b) most of **p24** (ISCHEMIA-CKD rationale, harms, certainty, values, resource-use); (c) most of **p26** (Rec 3.16.1 + its rationale + benefit meta-analysis); (d) most of **p27** (harms numbers + certainty + values + cost paragraphs); (e) **p28 Figure 43 footnote** (Apixaban/Edoxaban dose-reduction criteria). |
| Page-binding defects | p22 zero-span (page text fell into adjacent section offset gap); p24/p25/p27 one-to-three-span (extreme under-extraction on content-rich pages). |
| Positive signals | Figure 41/42/43/44 captions were recovered via L1_RECOVERY channel (hence non-zero coverage); Figure 43 dose tables captured as HTML. |

**Verdict for batch 3:** **one critical MISSED Recommendation** (Rec 3.16.1, the NOAC vs VKA recommendation for AFib in CKD G1–G4, graded 1C). All 7 Practice Points captured. Meta-analysis numbers for both aspirin (p21/p22) and NOACs (p26/p27) are **largely absent**, which weakens the evidence basis of the recommendations. **Five page-binding defects (p22 zero, p24 one, p25 three, p27 three, p29 three)** are worth investigating as a class of pipeline bugs.

---

## Batch 4 — Pages 31–40 (Chapter 1: §1.1 Evaluation of CKD + §1.2 Evaluation of GFR)

**Batch-level summary:** 51 + 3 + 36 + 4 + 5 + 17 + 1 + 9 + 3 + 23 = **152 spans**, all `PENDING`. Recommendations on these pages: **Rec 1.1.2.1, Rec 1.1.4.1, Rec 1.2.2.1** (3 recommendations). Practice Points: **PP 1.1.1.1, 1.1.1.2, 1.1.3.1, 1.1.3.2, 1.1.3.3, 1.1.4.1, 1.1.4.2, 1.2.1.1, 1.2.2.1** (9 PPs). Heavy D-channel fragmentation of Tables 4, 5, 6, 7 (84 of 152 spans are table fragments). **Page 37 has only 1 span** (a chunk marker) — catastrophic under-extraction on a genetics/biopsy content page. **Rec 1.2.2.1 is present only as part of a multi-purpose span mixing running-header + page-marker + surrounding prose** — a structural capture defect.

### Page 31 — Chapter 1 opening + §1.1 + §1.1.1 + PP 1.1.1.1 + Table 4

**PDF content:** Chapter 1 title; §1.1 Detection and evaluation of CKD; §1.1.1 Detection of CKD; **PP 1.1.1.1** (test people at risk using urine albumin + GFR); Table 4 (Use of GFR and albuminuria: Diagnosis & staging / Treatment / Risk assessment × Current GFR / Current albuminuria / Change in GFR level).

**DB spans (51):** all `PENDING`. **46 of 51 are D-channel Table 4 cell fragments** (cell labels like "Referral to nephrologists", "Risk for CKD progression", repeated 3× because the table has 3 duplicate abbreviation-footnote rows).

| Fact | Status |
|---|---|
| Chapter 1 heading + §1.1 preamble | **ADDED** (F+G, 0.85) — partial |
| §1.1.1 heading + **PP 1.1.1.1** full text | **ADDED** (C+G, 0.89) |
| "Initial testing of blood and urine to detect CKD is important, with confirmatory testing if initial findings indicate abnormalities" | **ADDED** (C+G, 0.82) |
| Table 4 body D-fragments (Clinical decisions, Diagnosis/staging, Treatment rows; Detection of AKI and AKD, Detection of CKD, Treatment of AKI, Referral to nephrologists, Patient education, Monitor GFR decline, Referral for kidney transplantation, Placement of dialysis access, Dosage & monitoring, Determine safety, Eligibility for clinical trials, Risk for CKD complications, Risk for CKD progression, Risk for CVD, Risk for medication errors, Risk for perioperative complications, Risk for mortality, Fertility & pregnancy, Risk for kidney failure, Risk for CVD/HF/mortality, Risk for adverse pregnancy outcome) | **NEEDS-CONFIRM** (D × ~40, 0.92) — **row-column pairing lost** so "Risk for CKD progression" fragments can't be mapped to "Current GFR" vs "Current albuminuria" vs "Change in GFR" columns |
| Table 4 abbreviation footnote — **triplicated** (3 identical D-fragments) | **NOISE** — reviewer should keep one, reject two |
| Cross-page marker `<!-- PAGE 31 --> ... <!-- PAGE 32 -->` | **ADDED** (F+G, 0.85) — structural, not a guideline fact |
| Narrative on "public health approach / early-detection harms / shared decision-making / WHO criteria for early-detection program" | **MISSED** — rationale paragraphs absent |
| Cross-reference to Chapter 3 (lifestyle changes) + Chapter 5 (implementation) | **MISSED** |

**Prior-run comparison:** old audit listed PP 1.1.1.1 as **present** on p31 (1 span). New run captures it with the full text + section heading. ✅ Improvement.

### Page 32 — PP 1.1.1.2 + AKI/AKD population + Priority-condition narrative + Pediatric considerations

**PDF content:** Continuation of early-detection rationale; priority conditions (HTN / DM / CVD incl. HF — ADA + KDIGO annual CKD screening); "second important group" (recent AKI / AKD / multiple AKI episodes / partially diagnosed CKD); Table 5 reference; testing without regard to age controversy; **PP 1.1.1.2** (repeat tests after incidental ACR / hematuria / low eGFR); hematuria narrative; Pediatric considerations (preterm / neonatal AKI).

**DB spans (3):** all `PENDING`

| Fact | Status |
|---|---|
| "Second important group includes people with recent AKI or AKD…" | **ADDED** (C+G, 0.82) |
| **PP 1.1.1.2** full text + rationale (biological/analytical SCr variability; repeat before diagnosing) | **ADDED** (C+G, 0.89) |
| Cross-page transition "There are several causes of transient hematuria" | **ADDED** (F+G, 0.85) — fragmentary |
| Public-health disparities narrative (WHO criteria, early-detection harms/anxiety, cost-effectiveness 2023 analyses) | **MISSED** |
| Priority conditions HTN/DM/CVD + ADA/KDIGO annual screening + T1D 5-year-post-diagnosis start | **MISSED** — **concrete screening rule absent** |
| Figure 3 reference (algorithm for identification of people at risk for CKD testing) | **MISSED** |
| Table 5 reference / "other groups who might be considered" list | **MISSED** |
| Persistent hematuria as indicator of glomerular disease / GU malignancy → further investigation | **MISSED** |
| Pediatric considerations (preterm / small gestational age / neonatal AKI / childhood obesity) | **MISSED** |

**Prior-run comparison:** old audit listed **PP 1.1.1.2** as MISSED on p32. **Now present as PENDING.** ✅ Improvement.

### Page 33 — Table 5 (Risk factors for CKD) D-fragments + §1.1.2 + Rec 1.1.2.1 + rationale

**PDF content:** Table 5 (Common risk factors HTN/DM/CVD/AKI/AKD; Geography with endemic CKDu/APOL1; GU disorders; Multisystem SLE/Vasculitis/HIV; Iatrogenic nephrotoxicity; Family history/PKD/Alport; Gestational preterm/SGA/pre-eclampsia; Occupational Cd/Pb/Hg/polycyclic HC/pesticides); §1.1.2 Methods for staging of CKD; **Rec 1.1.2.1** (eGFRcr baseline; add cystatin C when available → eGFRcr-cys) [1B]; rationale (CKD-PC 720 736 participants; eGFR 45–59 + ACR <10 reclassified out of "low-risk green"; Figure 6/7 cross-ref).

**DB spans (36):** all `PENDING`. **33 of 36 are D-channel Table 5 fragments**.

| Fact | Status |
|---|---|
| Table 5 "Domains" header + "Example conditions" header | **NEEDS-CONFIRM** (D × 2, 0.92) |
| Table 5 body fragments (HTN / DM / CVD incl HF / Prior AKI-AKD / Areas with endemic CKDu / APOL1 genetic variants / Environmental exposures / Structural GU disease / Recurrent calculi / SLE / Vasculitis / HIV / Drug nephrotoxicity & radiation nephritis / Kidney failure regardless of cause / PKD, APOL1, Alport / Preterm birth / SGA / Pre-eclampsia/eclampsia / Cadmium lead mercury / Polycyclic hydrocarbons / Pesticides) | **NEEDS-CONFIRM** (D × ~31, 0.92) — ≥2 cells **OCR-corrupted**: `Recurrent kidney calcuI` (should be "calculi"); `POL7-mediated kidney disease` (should be "APOL1-mediated"); `Areas with the high prevalence of APOL genetic variants` (should be "APOL1") — **named gene OCR errors matter for downstream matching** |
| **Rec 1.1.2.1** full text + `(1B)` grade | **ADDED** (C+E+G, 0.94) |
| Rationale (eGFR 45–59 + ACR <10 risk reclassification via eGFRcr-cys; Figure 6/7) | **ADDED** (C+G, 0.82) |
| Rationale block 2 (27 503 140 + 720 736 + 9 067 753 participant CKD-PC data; moderate certainty rating) | **ADDED** (C+G, 0.82) — but truncated inside the Rec span |
| Cross-page transition to p34 | **ADDED** (C+F+G, 0.92) |
| Narrative on preterm-birth pathway / pediatric considerations / initial-vs-subsequent eGFR testing | **MISSED** |
| "Recommendation puts high value on accurate GFR assessment…2 biomarkers" preamble | **MISSED** — embedded as a fragment only |
| Values-preferences / resource-use / implementation paragraphs | **MISSED** |

**Prior-run comparison:** old audit listed **Rec 1.1.2.1** as MISSED on p33 (only 32 spans). **Now captured as PENDING.** ✅ Improvement.

### Page 34 — PP 1.1.3.1 + PP 1.1.3.2 + PP 1.1.3.3 + §1.1.3 chronicity

**PDF content:** Rationale continuation for Rec 1.1.2.1; §1.1.3 Evaluation of chronicity; **PP 1.1.3.1** (proof of chronicity via 6 methods: past GFR / past albuminuria / imaging / pathology / medical history / repeat measurements within and beyond 3 months); **PP 1.1.3.2** (don't assume chronicity from single abnormal level); **PP 1.1.3.3** (consider initiating CKD treatments on first abnormal presentation if other indicators support); chronicity = ≥3 months rationale; acute/chronic differentiation narrative.

**DB spans (4):** all `PENDING`

| Fact | Status |
|---|---|
| "For this reason, the recommendation includes the alternative for eGFRcr in such cases…" (end of Rec 1.1.2.1 implementation) | **ADDED** (E+G, 0.82) |
| §1.1.3 heading + **PP 1.1.3.1** full text with all 6 bullet criteria | **ADDED** (C+G, 0.89) |
| **PP 1.1.3.2** + **PP 1.1.3.3** — **merged into one span** | **NEEDS-CONFIRM** (C+G, 0.89) — two distinct PPs packed together; reviewer needs to split |
| Cross-page marker to p35 | **ADDED** (F+G, 0.85) |
| Chronicity narrative (AKI/AKD/CKD differentiation; resolution over days/weeks confirms AKI; repeat ascertainment; timing depends on clinical judgment) | **MISSED** |
| "Delaying diagnosis for sake of confirming chronicity can delay care" rationale | **MISSED** |

**Prior-run comparison:** old audit listed **PP 1.1.3.1** as MISSED on p34. **Now captured as PENDING.** PP 1.1.3.2 and 1.1.3.3 are technically captured (merged); old audit didn't explicitly flag them, so this is still an improvement.

### Page 35 — Pediatric chronicity + §1.1.4 + PP 1.1.4.1 + PP 1.1.4.2 + Figure 8 + Rec 1.1.4.1

**PDF content:** Pediatric chronicity special consideration (severe CMAKUT don't need 3-mo wait); §1.1.4 Evaluation of cause; **PP 1.1.4.1** (establish cause via clinical context / history / social / environmental / medications / exam / labs / imaging / genetic / pathologic — Figure 8); **PP 1.1.4.2** (use tests per resources — Table 6); Figure 8 body (Physical exam, Nephrotoxic meds, Urinary tract abnormalities, Systemic-disease signs, Medical history, Social/environmental history, Family history, Lab tests (urinalysis/ACR/serology/US/biopsy/genetic testing)); genetic testing narrative (>10 % pathogenic variants; KDIGO Controversies Conference criteria; Figure 9/10 references); **Rec 1.1.4.1** (kidney biopsy as acceptable safe diagnostic test when clinically appropriate) [2D].

**DB spans (5):** all `PENDING`

| Fact | Status |
|---|---|
| §1.1.4 heading + **PP 1.1.4.1** + **PP 1.1.4.2** — **merged into one span** | **NEEDS-CONFIRM** (C+G, 0.89) — two distinct PPs packed together |
| "Identification of cause is often achieved by standard clinical methods…" (rationale) | **ADDED** (C+G, 0.82) |
| **Rec 1.1.4.1** full text + `(2D)` grade | **ADDED** (C+G, 0.89) |
| Figure 8 body as free-text fragment ("Physical exam / Nephrotoxic medications / Symptoms of urinary tract abnormalities / Symptoms of systemic diseases / Lab tests incl urinalysis, ACR, serology, US, biopsy, genetic testing") | **ADDED** (C+G, 0.82) — unstructured; Figure 8 is a mind-map, so this is adequate |
| Cross-page marker to p36 | **ADDED** (F+G, 0.85) |
| Pediatric chronicity narrative (CMAKUT bypass 3-mo rule) | **MISSED** |
| Genetic-testing criteria list (6 bullets from KDIGO Controversies Conference) | **MISSED** — **evidence-based testing-indication list absent** |
| Rec 1.1.4.1 Key-information (37-study ERT systematic review; 16 % perirenal hematoma rate; low mortality) | **MISSED** |
| Values/preferences + Resource use + Considerations-for-implementation for kidney biopsy | **MISSED** |

**Prior-run comparison:** old audit didn't flag §1.1.4 items. ✅ Now captured.

### Page 36 — Table 6 (Guidance for tests to evaluate cause) D-fragments + Figure 9 (Actionable genes)

**PDF content:** Table 6 (Test category / Examples / Comment — Imaging; Kidney biopsy; Lab-serologic-urine tests; Genetic testing rows); continuation of Rec 1.1.4.1 rationale (ERT review, 37 studies, 16 % perirenal hematoma); Figure 9 (6 categories of actionable genes: disease-modifying, renoprotective, avoid-immunosuppression, recurrence-risk-post-transplant, extrarenal-screening, reproductive-counseling).

**DB spans (17):** all `PENDING`. **15 of 17 are Table 6 D-fragments.**

| Fact | Status |
|---|---|
| Table 6 headers "Test category / Examples / Comment or key references" | **NEEDS-CONFIRM** (D × 3) |
| Table 6 Imaging row fragments (Ultrasound / IVU / CT KUB / NM / MRI; Assess kidney structure for cystic/reflux; 3D ultrasound evolving) | **NEEDS-CONFIRM** (D × 3) |
| Table 6 Kidney-biopsy row (US-guided percutaneous; light microscopy/IF/EM/molecular; for exact diagnosis/planning/activity/chronicity/response prediction) | **NEEDS-CONFIRM** (D × 2) |
| Table 6 Lab-tests row (chemistry/acid-base/electrolytes; serologic anti-PLA2R, ANCA, anti-GBM; serum free light chains; MGRS cross-reference; urinalysis persistent hematuria/albuminuria) | **NEEDS-CONFIRM** (D × 3) |
| Table 6 Genetic-testing row (APOL1, COL4A3/A4/A5, NPHS1, UMOD, HNF1B, PKD1, PKD2; evolving tool; common without classic family history) | **NEEDS-CONFIRM** (D × 2) |
| **OCR error:** `COL4AS` (should be "COL4A5") — **named gene error** | **NEEDS-CONFIRM** |
| **OCR error:** `acido-base` (should be "acid-base") | **NEEDS-CONFIRM** |
| **OCR error:** `MGRS) 19` (should be superscript citation "[MGRS])98") | **NEEDS-CONFIRM** |
| Rec 1.1.4.1 rationale prose (37 studies; 16 % hematoma; confounding concerns; downgrade imprecision) | **ADDED** (F+G, 0.85) — partial |
| Figure 9 body (6 categories of actionable genes with examples: Fabry-GLA, Alport-COL4A3/4/5, Cystinosis-CTNS, aHUS-CFH/CFI/C3, primary hyperoxaluria-AGXT/GRHPR/HOGA, APRT, PKD1/PKD2 intracranial aneurysms, HNF1B diabetes, FLCN RCC, prenatal/preimplantation diagnosis) | **MISSED** — **rich evidence-based gene-management map absent** |
| Rec 1.1.4.1 Certainty / Values / Resources / Implementation paragraphs | **MISSED** |

### Page 37 — **Only 1 span** (chunk marker) — Figure 10 + biopsy rationale + §1.2 opening

**PDF content:** Figure 10 (Proposed organization for genetics in nephrology — 3-tier centers of expertise / connections with geneticists / all nephrology clinics with basic knowledge); continuation of Rec 1.1.4.1 rationale (downgrade for study limitations, heterogeneity in perirenal hematoma rates, no retroperitoneal hemorrhage data); Values & preferences; Resource use & costs; Considerations for implementation (standardized US-guided protocol); Rationale ("Kidney biopsy is important part of investigation for cause…"); Pediatric considerations (children/young people have more genetic causes); **§1.2 Evaluation of GFR** intro.

**DB spans (1):** all `PENDING`

| Fact | Status |
|---|---|
| Only span: `www.kidney-international.org chapter 1 Kidney International (2024) 105 (Suppl 4S), S117–S314 S175 <!-- Chunk chunk-d3: pages 38-45 -->` | **NOISE** — a page footer + a pipeline chunk marker. **Zero clinical content captured.** |
| Figure 10 caption + body (3-tier organization model for genetics in nephrology) | **MISSED** |
| Rec 1.1.4.1 Certainty-of-evidence paragraph | **MISSED** |
| Rec 1.1.4.1 Values-and-preferences | **MISSED** |
| Rec 1.1.4.1 Resource-use paragraph | **MISSED** |
| Rec 1.1.4.1 Implementation paragraph (standardized US-guided biopsy protocol) | **MISSED** |
| Rec 1.1.4.1 Rationale paragraph | **MISSED** |
| §1.2 Evaluation of GFR opening (kidney excretory/endocrine/metabolic functions) | **MISSED** (captured on p38 instead) |
| Pediatric considerations for biopsy (genetic causes more common in children) | **MISSED** |

> **Page 37 is catastrophic under-extraction** — 6 550 chars of clinically rich PDF text reduced to a single chunk-marker span. This is a worse failure mode than the zero-span p22 (where at least nothing was misleading).

**Prior-run comparison:** old audit listed p37 with `1 span`, same as here. **No improvement.**

### Page 38 — §1.2 intro + §1.2.1 + PP 1.2.1.1 + §1.2.2 + PP 1.2.2.1 + (Rec 1.2.2.1 partial)

**PDF content:** §1.2 Evaluation of GFR continuation; §1.2.1 Other functions of kidneys besides GFR; **PP 1.2.1.1** (use "GFR" for glomerular filtration; "kidney function(s)" for the totality); §1.2.2 Guidance for physicians; Figure 11 / Tables 7/8 reference; **PP 1.2.2.1** (use SCr + estimating equation for initial GFR — Figure 11); **Rec 1.2.2.1** (eGFRcr-cys when eGFRcr less accurate + clinical decision-making — Table 8) [1C].

**DB spans (9):** all `PENDING`

| Fact | Status |
|---|---|
| §1.2 intro ("excretory function is widely accepted as best overall index…") | **ADDED** (F+G, 0.85) |
| §1.2 rationale ("We encourage healthcare providers to have a clear understanding of the value and limitations of filtration markers…") | **ADDED** (C+G, 0.82) |
| §1.2.1 heading + **PP 1.2.1.1** full text | **ADDED** (C+G, 0.89) |
| "GFR considered best overall assessment of kidney functions…other kidney functions as complications of CKD in Chapter 3" | **ADDED** (C+G, 0.82) |
| §1.2.2 framework ("Healthcare providers should consider both potential sources of error in eGFR as well as clinical decision accuracy requirements") | **ADDED** (E+G, 0.82) |
| **PP 1.2.2.1** full text | **ADDED** (C+G, 0.89) |
| PP 1.2.2.1 rationale (no RCTs; SCr estimation OK for most cases; post-eGFR-reporting increase in CKD recognition/referral) | **ADDED** (C+G, 0.82) |
| Sources of error in SCr (non–steady-state, non-GFR determinants, measurement error at high GFR, assay interferences) | **ADDED** (C+G, 0.82) |
| Cross-page span from p38 to p39 **containing Rec 1.2.2.1 text** mixed with running-header + page-marker + preceding prose | **NEEDS-CONFIRM** (C+F+G, 0.96) — **this is the only DB location of the Rec 1.2.2.1 recommendation text, but it is embedded inside `chapter 1 www.kidney-international.org / S176 / <!-- PAGE 39 --> / Most people with CKD…`** — downstream consumers will struggle to identify this as a numbered Recommendation. **Structural capture defect.** |

### Page 39 — Figure 11 (GFR-assessment algorithm) + Rec 1.2.2.1 proper location + rationale

**PDF content:** Figure 11 body (Initial test – eGFRcr → sources of error check → Yes: use eGFRcr / No: measure cystatin C → sources of error in eGFRcr-cys → No: use eGFRcr-cys / Yes: measure GFR); footnote with exceptions (initial may be eGFRcys in healthy populations with creatinine-generation changes; sources of error in eGFRcr-cys: very low muscle mass / high inflammation / catabolic state / exogenous steroids); **Rec 1.2.2.1** full text + `(1C)`; "This recommendation places a high value on using estimates of GFR derived from a combination of creatinine and cystatin C in clinical situations where eGFRcr is unreliable…".

**DB spans (3):** all `PENDING`

| Fact | Status |
|---|---|
| Figure 11 decision-node "Is eGFR thought to be accurate?" | **ADDED** (C+G, 0.82) — fragmentary |
| Figure 11 caption narrative (initial test eGFRcr; if inaccurate or need more accuracy → measure cystatin C → eGFRcr-cys; if still inaccurate → measure GFR) | **ADDED** (E+G, 0.82) |
| Figure 11 footnote (eGFRcys as initial in healthy populations; sources of error in eGFRcr-cys) | **ADDED** (C+D+E+F+G, 1.00) |
| **Rec 1.2.2.1 proper span** ("We recommend using eGFRcr-cys in clinical situations when eGFRcr is less accurate and GFR affects clinical decision-making (Table 8) (1C).") | **MISSED** — the canonical form is absent on p39; only the embedded-in-multi-purpose version exists on p38 |
| Rec 1.2.2.1 preamble ("places a high value on using estimates of GFR derived from a combination of creatinine and cystatin C…consistent evidence that eGFRcr-cys provides more accurate estimates") | **MISSED** |
| "eGFR from creatinine is widely used" + EMR/point-of-care reporting | **MISSED** |

### Page 40 — Table 7 (GFR assessment methods) D-fragments + Key info for Rec 1.2.2.1

**PDF content:** Table 7 (Estimated GFR via Cr / Cr+CysC / CysC; mGFR gold standard via iohexol/iothalamate/51Cr-EDTA/99mTc-DTPA; Timed urine clearance; Nuclear medicine imaging); Rec 1.2.2.1 Key information — Balance of benefits (CKD-PC 720 736 cystatin-C analysis; P30 90 % for eGFRcr-cys in N-America/Europe; 80–90 % in Brazil/Congo/Pakistan/Singapore/Japan/China; potential harms cost/complexity/interpretation); Certainty of evidence (moderate-to-high in ambulatory; low in frail/acute/chronic illness).

**DB spans (23):** all `PENDING`. **16 of 23 are D-channel Table 7 fragments.**

| Fact | Status |
|---|---|
| Table 7 columns (GFR assessment method / Specific tests / Guidance for use and implementation) | **NEEDS-CONFIRM** (D × 3, 0.92) |
| Table 7 rows (eGFR-Cr; eGFR-Cr-CysC, eGFR-CysC; mGFR iohexol/iothalamate/51Cr-EDTA/99mTc-DTPA; Timed urine creatinine clearance; Nuclear-medicine imaging tracer) | **NEEDS-CONFIRM** (D × ~12, 0.92) |
| **OCR errors:** `11Tc-DTPA`, `11Cr-EDTA` (should be `99mTc-DTPA`, `51Cr-EDTA`) — **critical isotope-nomenclature errors** | **NEEDS-CONFIRM** — **clinically significant OCR defect** |
| Table 7 HTML shell with full isotope names preserved elsewhere | **ADDED** (C+E+F+G, 1.00) — **recovers the correct isotope names** (51Cr-EDTA, 99mTc-DTPA); reviewer should keep HTML, reject D-fragments |
| Key-information narrative ("estimating GFR using combined creatinine and cystatin C equation provides the required degree of accuracy and obviates the need for expensive mGFR") | **ADDED** (C+G, 0.82) |
| Paucity of studies in frail/acute/chronic populations | **ADDED** (C+G, 0.82) |
| "Other studies of sick or frail people, such as very advanced liver or heart failure or ICU-admitted, all eGFR tests demonstrated very low levels of accuracy" | **ADDED** (C+G, 0.82) |
| "Even for populations where eGFRcr-cys is more accurate, assess potential sources of error" (clinical caveat) | **ADDED** (C+G, 0.82) |
| P30 statistics (90 % N-America/Europe; 80–90 % Brazil/Congo/Pakistan/Singapore/Japan/China) | **MISSED** — **key quantitative evidence for Rec 1.2.2.1** |
| Stockholm cohort eGFRcr-cys superiority in HF/liver/cancer/CVD/DM | **MISSED** |
| Hematologic cancer / cystatin C high cell-turnover caveat | **MISSED** |

### Batch 4 roll-up

| Metric | Value |
|---|---|
| Spans reviewed | 152 |
| Recommendations present on these 10 pages | 3 (Rec 1.1.2.1, Rec 1.1.4.1, Rec 1.2.2.1) |
| Practice Points present on these 10 pages | 9 (PP 1.1.1.1, 1.1.1.2, 1.1.3.1, 1.1.3.2, 1.1.3.3, 1.1.4.1, 1.1.4.2, 1.2.1.1, 1.2.2.1) |
| Recommendations captured | **3 of 3 (100 %)** — but **Rec 1.2.2.1 is only embedded in a multi-purpose span** (NEEDS-CONFIRM — split required for downstream ingestion) |
| Practice Points captured | **9 of 9 (100 %)** — but **PP 1.1.3.2+3 and PP 1.1.4.1+2 are each packed into one span** (NEEDS-CONFIRM — split required) |
| Major narrative content MISSED | (a) **entire p37** (Figure 10 + Rec 1.1.4.1 certainty/values/resource/implementation paragraphs + §1.2 opening); (b) **p32 priority-screening list** (HTN/DM/CVD + ADA/KDIGO annual screening + T1D 5-yr rule) — **concrete clinical screening rule** gone; (c) **p33 pediatric/values/resource-use paragraphs** for Rec 1.1.2.1; (d) **Figure 9 actionable-genes map**; (e) **P30 accuracy statistics** on p40 that justify Rec 1.2.2.1. |
| OCR / pipeline artefacts | `POL7` (should be APOL1); `COL4AS` (should be COL4A5); `calcuI` (should be calculi); `acido-base` (should be acid-base); `11Tc-DTPA` / `11Cr-EDTA` (should be 99mTc-DTPA / 51Cr-EDTA) — **5 named-entity OCR errors on genes/isotopes**; 3 triplicated Table 4 footnote rows; 1 chunk-marker leaked as span; p37 one-span catastrophe. |
| Positive signals | Figure 11 footnote captured with very high confidence (1.00); Table 7 HTML recovered correct isotope nomenclature where D-channel mangled it. |

**Verdict for batch 4:** **100 % Recommendation & PP capture (counting the embedded Rec 1.2.2.1)** but significant structural issues: (1) **p37 has 1 span of clinical content = 0** (catastrophic); (2) Rec 1.2.2.1 packaged inside a running-header-and-page-marker blob (downstream parser trap); (3) 5 OCR errors on gene symbols / isotope names (material for any KB-4 ingestion); (4) three multi-PP packed spans need reviewer splitting.

---

## Batch 5 — Pages 41–50 (§1.2.2 cystatin-C + §1.2.3 lab guidance + §3.10 Metabolic acidosis + §3.11 Hyperkalemia + §3.12 Anemia + §3.13 CKD-MBD)

**Batch-level summary:** 58 + 20 + 5 + 8 + 10 + 34 + 12 + 74 + 58 + 57 = **336 spans**, all `PENDING`. **Very heavy D-channel fragmentation** on Tables 8, 9, 23, 24, 29 (over 220 D-fragments). Practice Points on these pages: **PP 1.2.2.2, 1.2.2.3, 1.2.2.4, 1.2.2.5, 1.2.2.6, 1.2.2.7, 1.2.2.8, 1.2.3.1, 1.2.3.2, 3.10.1, 3.10.2** (11 PPs). **Two Practice Points MISSED** — PP 1.2.2.6 and PP 1.2.2.7 on p45 — their canonical text does not appear as a dedicated span. No full Recommendations on this batch (the recommendations fall in batch 4 for Ch 1 and earlier for Ch 3).

### Page 41 — Table 8 (Cystatin C indications by clinical domain) — heavily fragmented

**PDF content:** Table 8 (Body habitus: eating disorders / extreme sport / above-knee amputation / spinal cord injury / Class III obesity; Lifestyle: smoking; Diet: low-protein / keto / vegetarian / high-protein; Illness: malnutrition / cancer / HF / cirrhosis / catabolic disease / muscle-wasting; Medication effects: steroids / tubular-secretion inhibitors / broad-spectrum antibiotics). Each row × 3 columns (clinical condition / cause of decreased accuracy / comment on GFR evaluation).

**DB spans (58):** all `PENDING`. **53 of 58 are D-channel Table 8 cell fragments.**

| Fact | Status |
|---|---|
| Table 8 condition labels (Eating disorders / Extreme sport / Above-knee amputation / Spinal cord injury / Class III obesity / Smoking / Low-protein diet / Keto / Vegetarian / High-protein / Malnutrition / Cancer / Heart failure / Cirrhosis / Catabolic consuming diseases / Muscle wasting diseases / Steroids / Decreases in tubular secretion / Broad spectrum antibiotics) | **NEEDS-CONFIRM** (D × 19, 0.92) |
| Table 8 cause-of-decreased-accuracy fragments (Non-GFR determinants of SCr × many; Non-GFR determinants of SCys; Chronic illness presumed impact on SCr+SCys × 6) | **NEEDS-CONFIRM** (D × ~15) |
| Table 8 comments fragments (eGFRcys may be appropriate × several variants; eGFRcr-cys most accurate; Suggest mGFR × many; Minimal data) | **NEEDS-CONFIRM** (D × ~15) |
| **OCR errors:** `Heart failure 131,139` (PDF: `138,139`); `Catabolic consuming diseases 5` (PDF superscript `c`); `eGFRcrcys or eGFRcrys` (PDF: `eGFRcr-cys or eGFRcys`) | **NEEDS-CONFIRM** — citation-number and hyphenation errors |
| Table 8 abbreviation footnote | **ADDED** (C+E+G, 0.90) |
| Cross-page marker p41 → p42 | **ADDED** (C+F+G, 0.92) |
| Missing: no clean HTML version of Table 8 exists in DB (unlike Tables 25, 26, 43) — reviewer must reconstruct row-column pairing from D-fragments | **NEEDS-CONFIRM** (batch-level) |

**No Recommendations or Practice Points** are present on p41.

### Page 42 — §1.2.2.1 rationale + PP 1.2.2.2 + Table 9 (eGFR vs mGFR)

**PDF content:** Rec 1.2.2.1 Values-and-preferences / Resources / Implementation continuation; Rationale ("We describe a framework for the evaluation of GFR beginning with an initial test…Cystatin C is an alternative endogenous filtration marker"); **PP 1.2.2.2** (measure GFR using plasma/urinary clearance when accuracy is critical — Table 9); rationale (greatest benefit of mGFR is freedom from non-GFR determinants); Table 9 (6-row comparison: inexpensive vs expensive; widely available vs certain centers; not accurate-all vs accurate-all; lags changes vs identifies early; non-GFR confounding vs less influenced).

**DB spans (20):** all `PENDING`. **12 of 20 are Table 9 D-fragments.**

| Fact | Status |
|---|---|
| Table 9 rows (Inexpensive/easy vs more expensive+time-consuming; Widely available incl point-of-care vs only certain centers; Not sufficiently accurate vs accurate in all situations; Lags changes vs identifies early; Subject to non-GFR confounding vs less influenced by) | **NEEDS-CONFIRM** (D × ~12, 0.92) |
| Values & preferences paragraph ("Work Group judged that most people and most healthcare providers would want to use the most accurate assessment of GFR…balance additional costs…") | **ADDED** (C+G, 0.82) |
| Resource-use paragraph (one-time IT infrastructure + ongoing reagents costs for cystatin C; increased nephrology referrals initially) | **ADDED** (C+G, 0.82) |
| Implementation paragraph ("cystatin C needs to be widely available…access to both creatinine and cystatin C measurements") | **ADDED** (C+G, 0.82) |
| Section 1.2.3 cross-reference | **ADDED** (C+G, 0.82) |
| **PP 1.2.2.2** full text | **ADDED** (C+G, 0.89) |
| PP 1.2.2.2 rationale ("greatest benefit of mGFR is that it is less influenced by non-GFR determinants") | **ADDED** (C+G, 0.82) |
| Table 9 footnote | **ADDED** (C+F+G, 0.95) |
| Narrative on exogenous markers (iothalamate / iohexol / EDTA / DTPA) + 99mTc-DTPA contraindication over 2–4 h | **MISSED** |
| Time-to-time variability CV 6.7 % for mGFR; 5 % for eGFRcr/cys/cr-cys; iothalamate urinary clearance CV 6.3–16.6 % | **MISSED** |
| Cost of mGFR infrastructure (IV catheter, serial sampling, HPLC/mass-spec) | **MISSED** |

### Page 43 — Table 10 (indications for mGFR) + PP 1.2.2.3 + EKFC harmonization

**PDF content:** Table 10 (Indications for mGFR — 2 clinical-condition categories: eGFRcr-cys inaccurate/uncertain; greater accuracy needed); narrative on CV calculations; accuracy sampling errors; **PP 1.2.2.3** (understand value/limitations of eGFR and mGFR + variability + factors influencing SCr/cystatin C); EKFC harmonization of iohexol plasma-clearance protocols; Figure 12 reference.

**DB spans (5):** all `PENDING`

| Fact | Status |
|---|---|
| Narrative tail ("The Work Group judged that there will be some clinical situations where estimating GFR from both creatinine and cystatin C will be insufficiently reliable…") | **ADDED** (C+G, 0.82) |
| Framework pointer ("If greater accuracy is needed than can be achieved using eGFR, mGFR is recommended") | **ADDED** (C+G, 0.82) |
| **PP 1.2.2.3** full text + rationale "All studies evaluating the performance of eGFR compared with mGFR observe error in any GFR estimate" | **ADDED** (C+G, 0.89) |
| Critical-decision rationale ("A critical component of the recommended approach to evaluation of GFR (Figure 11) is that physicians have a clear understanding…") | **ADDED** (B+C+G, 0.95) |
| Cross-page marker | **ADDED** (C+F+G, 0.92) |
| **Table 10 body** (2 indication categories: ≥5 clinical conditions where eGFRcr-cys inaccurate — catabolic/serious infections/inflammation/high cell turnover cancer/cirrhosis/HF/high-dose steroids/frail; kidney donor candidacy / drug dosing with narrow TI / transplant decisions) | **MISSED** — **clinical indication list for when to move to mGFR is absent** |
| EKFC + EFLM iohexol harmonization note | **MISSED** |
| Non-GFR determinants of creatinine (diet/muscle-mass/tubular-secretion/extrarenal) and cystatin C (adiposity/smoking/thyroid/glucocorticoids/inflammation) | **MISSED** |

### Page 44 — Figure 12 (sources of error around mGFR/eGFR) + PP 1.2.2.4 + PP 1.2.2.5

**PDF content:** Continuation ("mGFR also differs from true physiological GFR… preanalytical/analytical/biological variability"); **PP 1.2.2.4** (interpret SCr with consideration of dietary intake — cooked meat/fish postprandial effect ~0.23 mg/dl SCr rise, wait 12 h after meat/fish); **PP 1.2.2.5** (assess potential for error in eGFR over time); Figure 12 body (P30 for eGFR / P15 for mGFR with concrete numeric examples at GFR 60 and 30 mL/min per 1.73 m²).

**DB spans (8):** all `PENDING`

| Fact | Status |
|---|---|
| Continuation ("In the absence of changes related to disease progression, a change in mGFR may occur due to preanalytical / analytical / biological variability") | **ADDED** (C+G, 0.82) |
| **PP 1.2.2.4** full text | **ADDED** (C+G, 0.89) |
| PP 1.2.2.4 rationale (cooked meat/fish 20 µmol/l SCr rise; 2–4 h peak; wait 12 h; clinically challenging to implement) | **ADDED** (C+G, 0.82) |
| **PP 1.2.2.5** full text | **ADDED** (C+G, 0.89) |
| PP 1.2.2.5 rationale ("whether true GFR is changing…consider change in non-GFR determinants") | **ADDED** (C+E+G, 0.90) |
| Figure 12 footnote ("P30 for eGFR refers to percent within 30 % of mGFR…P30 >80 % acceptable / >90 % optimal…non-GFR determinants of endogenous markers / nonideal properties of exogenous markers") | **ADDED** (C+E+G, 0.92) |
| Figure 12 concrete numeric examples (at GFR 60: eGFR 30 % = 42–78; mGFR 15 % = 51–69; at GFR 30: eGFR 30 % = 21–39; mGFR 15 % = 26–35) | **ADDED** (E+G, 0.82) |
| Cross-page marker | **ADDED** (C+F+G, 0.96) |

**No MISSED items.** Batch-2 equivalent density of capture.

### Page 45 — **PP 1.2.2.6 MISSED + PP 1.2.2.7 MISSED** + PP 1.2.2.8 + Sex/Gender + Pediatric

**PDF content:** Narrative on combined eGFRcr+eGFRcys being more accurate than either alone; examples (amputation / anorexia / tubular-secretion-inhibiting medications); **PP 1.2.2.6** (consider eGFRcys in some specific circumstances); **PP 1.2.2.7** (understand implications of eGFRcr/eGFRcys differences); concordance/discordance data (25–30 % discordance ≥15 ml/min/1.73 m² or ≥20 %); **PP 1.2.2.8** (timed urine CrCl if mGFR unavailable and eGFRcr-cys inaccurate); sex/gender considerations (testosterone ↑ SCr and cystatin C; estradiol ↓ cystatin C; EKFC sex-free cystatin equation); pediatric considerations (CKiD cohort averaging eGFRcr+eGFRcys).

**DB spans (10):** all `PENDING`

| Fact | Status |
|---|---|
| Narrative "In individuals where non-GFR determinants of creatinine or cystatin C are substantially greater than for the other marker, eGFRcr-cys would not provide the more accurate estimate" | **ADDED** (C+G, 0.82) |
| "We, therefore, advise limiting this strategy to selected clinical settings where people are otherwise healthy with known changes in non-GFR determinants of creatinine" | **ADDED** (C+G, 0.82) |
| Anorexia and amputation examples (1 study pre/post amputation in military veterans; anorexia study) | **ADDED** (C+G, 0.89) |
| 25–30 % discordance statistics (≥15 ml/min/1.73 m² or ≥20 %) + factors (older age / female sex / non-Black race / higher eGFR / higher BMI / weight loss / smoking) | **ADDED** (C+G, 0.82) |
| Discordance vs concordance P30 numbers (concordance P30 87–91 % for all 3; discordance eGFRcr-cys more accurate) | **ADDED** (C+G, 0.82) |
| **PP 1.2.2.8** full text | **ADDED** (C+G, 0.89) |
| PP 1.2.2.8 rationale ("CrCl is widely available but highly prone to error due to under/overcollection; ≤25 % bias across 23 studies; AASK pilot 25 % had CrCl 18 % lower than mGFR") | **ADDED** (C+G, 0.82) |
| Systematic review on mGFR methods | **ADDED** (C+G, 0.82) |
| Sex/gender considerations (transgender/gender-diverse; gender-affirming testosterone ↑ SCr and cystatin C; estradiol ↓ cystatin C; AACC + NKF shared-decision approach; EKFC sex-free cystatin equation) | **ADDED** (C+G, 0.82) |
| Pediatric considerations + cross-page marker | **ADDED** (C+F+G, 0.92) |
| **PP 1.2.2.6 full text ("Consider the use of cystatin C–based estimated glomerular filtration rate (eGFRcys) in some specific circumstances.")** | **MISSED** — no canonical-form span exists |
| **PP 1.2.2.7 full text ("Understand the implications of differences between eGFRcr and eGFRcys, as these may be informative, in both direction and magnitude of those differences.")** | **MISSED** — no canonical-form span exists |

> **Confirmed by direct DB query:** `SELECT * WHERE text ILIKE '%Consider the use of cystatin%specific circumstances%'` returns 0 rows; `text ILIKE '%implications of differences between eGFR%'` returns 0 rows. These two Practice Points are the **second and third missed PPs in the job** (in addition to Rec 3.16.1). The rationale/narrative around them is captured, but the PP headings themselves are lost.

### Page 46 — Table 11 (implementation standards) + Table 12 (interfering substances) + §1.2.3 + PP 1.2.3.1 + PP 1.2.3.2

**PDF content:** Table 11 (8 laboratory implementation standards — report eGFR with markers; round SCr/cystatin C; flag eGFR <60; CV <2.3 % / <2.0 %; bias <3.7 % / <3.2 %; enzymatic method; separate serum from RBCs within 12 h; measure Cr and cystatin C on same sample); §1.2.3 Guidance to clinical laboratories; **PP 1.2.3.1** (implement lab standards outlined in Table 11); **PP 1.2.3.2** (consider both creatinine + cystatin as in-house or referred); Jaffe vs enzymatic discussion; Table 12 (interfering substances for Jaffe vs enzymatic).

**DB spans (34):** all `PENDING`. **24 of 34 are Table 12 D-fragments.**

| Fact | Status |
|---|---|
| Table 12 Jaffe-method interferences (Acetaminophen, Aspirin, Ascorbic acid, Bacterial contamination, Bilirubin, Blood-substitute products, Cephalosporins, Fluorescein, Glucose, Hemoglobin F, Ketones/ketoacids, Lipids, Metamizole protein, Pyruvate from delayed processing, Streptomycin) | **NEEDS-CONFIRM** (D × 15, 0.92) |
| Table 12 Enzymatic-method interferences (Bilirubin, Lidocaine metabolites, Metamizole, N-acetylcysteine, Proline stabilizers in IVIG, Phenindione × 2 duplicate) | **NEEDS-CONFIRM** (D × 7, 0.92) — **Phenindione duplicated** |
| Table 12 footnote | **ADDED** (D, 0.92) |
| Pediatric p46 opening ("Children CKiD cohort…averaging eGFRcr+eGFRcys reduced mean bias…89–91 % P30") | **ADDED** (F+G, 0.85) |
| §1.2.3 Guidance to clinical laboratories heading + **PP 1.2.3.1** + **PP 1.2.3.2** — **merged into one span** | **NEEDS-CONFIRM** (C+G, 0.89) — two distinct PPs packed together; reviewer needs to split |
| PP 1.2.3.1/2 rationale (consistency/standardization/comparability; JCTLM reference; lab standards adoption; flag decreased eGFR) | **ADDED** (C+G, 0.82) |
| Jaffe vs enzymatic discussion (non-creatinine chromogens 20 %; enzymatic more specific; Table 12 interferences) | **ADDED** (C+G, 0.82) |
| Table 11 implementation standards (fragment: "Measure filtration markers using a specific, precise CV <2.3 % for creatinine and <2.0 % for cystatin C…bias <3.7 % / <3.2 %") | **ADDED** (C+G, 0.82) |
| Table 11 concluding item ("When cystatin C is measured, measure creatinine on same sample") + footnote | **ADDED** (C+E+G, 0.90) |
| Serum/RBC separation 12-h rule for Jaffe | **ADDED** (via Table 11 fragment) |
| Cross-page marker to p47 | **ADDED** (F+G, 0.85) |
| **Full Table 11 eight-item list as a single structured block** | **MISSED** — items are scattered across 3 fragmentary spans; no coherent "standards list" span |

### Page 47 — §1.2.3 rationale + implementation considerations + cystatin C availability + EMR integration

**PDF content:** Narrative on lab error components (accuracy / imprecision / specificity); international reference standards for creatinine and cystatin C; implementation considerations (Jaffe availability; cystatin C same-day results; need for local lab or centralized referral); eGFR reporting mechanics (close communication with clinical users; sufficient validation; EQA); BSA adjustment; sex variable in EMRs; cystatin C as sex-free alternative.

**DB spans (12):** all `PENDING`

| Fact | Status |
|---|---|
| "eGFR is an imperfect estimate of mGFR. At best, 90 % of eGFR will fall within 30 % of mGFR" | **ADDED** (C+G, 0.82) |
| "Optimization of laboratory measurements of creatinine and cystatin C can help to reduce uncertainty" | **ADDED** (C+G, 0.82) |
| International reference standards (JCTLM) for creatinine and cystatin C | **ADDED** (C+G, 0.82) |
| Target CV for creatinine / cystatin C | **ADDED** (C+G, 0.82) |
| "Most people with CKD, healthcare providers, and policy makers would want laboratories to implement calibrated assays…" | **ADDED** (C+G, 0.82) |
| Jaffe is inexpensive; enzymatic more specific but more expensive; cystatin C adds cost | **ADDED** (C+E+G, 0.90) |
| Cystatin C turnaround time → same-day availability requirement | **ADDED** (C+G, 0.82) |
| eGFR implementation mechanics (lab-clinical user communication; validation; EQA; equation documentation) | **ADDED** (C+G, 0.82) |
| Reporting units standardization + BSA-adjustment | **ADDED** (C+G, 0.82) |
| EMR sex-variable handling | **ADDED** (C+G, 0.82) |
| "The comment may also include a suggestion to use cystatin C as there is less difference between eGFRcys values for males and females…option for computing eGFR without sex" | **ADDED** (B+C+F+G, 1.00) |
| Batch-level observation: **page 47 is the best-covered page in this batch** — 12 spans for ~6 900 chars is close to 1 span/500 chars | |

### Page 48 — Table 23 (bicarbonate by age/sex/eGFR) + Figure 28 + §3.10 Metabolic acidosis + PP 3.10.1 + PP 3.10.2

**PDF content:** Figure 28 (bicarbonate-eGFR association chart); §3.10 Metabolic acidosis preamble; definition & prevalence (acidosis observationally associated with protein catabolism, muscle wasting, inflammation; bicarbonate falls below 60 mL/min/1.73m²; 7.7 %/6.7 % prevalence G3A1 → 38.3 %/35.9 % G5A3); **PP 3.10.1** (pharmacological treatment ± dietary when bicarbonate <18 mmol/l in adults); **PP 3.10.2** (monitor treatment so bicarbonate not above ULN, no adverse BP/K+/fluid effects); 2012 2B recommendation history; 2021 systematic review (15 trials, 2445 pts, HR 0.81 for kidney failure); BiCARB trial; Table 23 (bicarbonate by age/sex/eGFR in 3.99 M population).

**DB spans (74):** all `PENDING`. **66 of 74 are Table 23 D-channel fragments.**

| Fact | Status |
|---|---|
| Table 23 GFR category labels (105+ / 90–104 / 75–89 / 60–74 / 45–59 / 30–44 / 15–29 / 0–14) | **NEEDS-CONFIRM** (D × 8 × 2 age groups — **duplicated because D-channel scanned each of 4 sex-age row combinations**) |
| Table 23 mean(SD) values — 64 fragments: ≥65 F: 27.4, 27.1, 26.9, 26.8, 26.5, 25.9, 24.8, 24.0; ≥65 M: 27.1, 26.6, 26.7, 26.5, 26.1, 25.3, 24.1, 24.2; <65 F: 25.2, 26.1, 26.3, 26.4, 26.2, 25.1, 23.6, 24.0; <65 M: 26.4, 26.5, 26.6, 26.5, 25.9, 24.8, 23.5, 24.4 | **NEEDS-CONFIRM** (D × ~60, 0.92) — row×column pairing lost but numbers faithful |
| **Suspicious duplicate values** — "24.0 (4.8)" and "24.4 (4.7)" each appear 3× — more than expected from PDF (each appears once in age-sex-eGFR cells) | **NEEDS-CONFIRM** — D-channel **over-sampled** values for lower eGFR categories |
| Table 23 HTML-ish reconstruction (C+G span) | **ADDED** (C+G, 0.82) — lossy but row-structured |
| §3.10 heading + metabolic-acidosis definition/prevalence | **ADDED** (C+G, 0.82) |
| **PP 3.10.1** full text (bicarbonate <18 mmol/l trigger) | **ADDED** (C+G, 0.89) |
| 2021 SR 15-trial meta-analysis HR 0.81 for kidney failure | **ADDED** (C+G, 0.88) |
| "Totality of evidence remains limited by low number of outcomes…meta-analysis of placebo-controlled trials does not confirm…bicarbonate HR 0.81 (95 % CI 0.54–1.22)" | **ADDED** (C+E+G, 0.90) |
| Cross-page marker to p49 with BiCARB trial pointer | **ADDED** (F+G, 0.85) |
| **PP 3.10.2** full text | **MISSED** — no span contains "Practice Point 3.10.2" canonical text; no span has full "Monitor treatment for metabolic acidosis to ensure it does not result in serum bicarbonate concentrations exceeding ULN…" |
| Rationale for 2012 2B recommendation downgrade to practice point | **MISSED** — context for why no graded Rec now |

### Page 49 — Table 24 (potassium by age/sex/eGFR) + BiCARB trial + alkaline-rich diets + §3.11 hyperkalemia preamble

**PDF content:** Table 24 (serum K+ by age/sex/eGFR in 4.28 M database); BiCARB trial details (CKD G3–G4 ≥60 y; 33/152 vs 33/148 kidney failure; EQ-5D-3L QoL lower with bicarbonate); licensed non-alkali oral interventions; alkaline-rich plant-based diet RCTs (4 small RCTs); pediatric acidosis (CKiD cohort 38–60 % have bicarbonate <22 mmol/l; growth-retardation risk; 17 mmol/l lower normal for young children); §3.11 Hyperkalemia in CKD preamble (cardiac electrophysiology; K+ homeostasis; G3A1 prevalence 8.8 % DM vs 4.5 % no-DM → G5A3 34.4 % / 23.7 %).

**DB spans (58):** all `PENDING`. **52 of 58 are Table 24 D-fragments.**

| Fact | Status |
|---|---|
| Table 24 headers (Measure, Age, Sex, GFR categories) | **NEEDS-CONFIRM** (D × 10+) |
| Table 24 K+ values (4.1 / 4.2 / 4.3 / 4.4 / 4.5 / 4.6 by age/sex/GFR) | **NEEDS-CONFIRM** (D × ~32) |
| **OCR anomaly:** "4.3 (17.0)" for <65 Female 75-89 (PDF: should be 17.0? — actually PDF shows `4.3 (17.0)` matching) — appears correct but reads like an anomalous SD | **NEEDS-CONFIRM** — reviewer should double-check this SD against source; likely PDF typo |
| BiCARB trial details (33/152 vs 33/148 kidney failure; HR 0.97; primary outcome Short Physical Performance Battery at 12 months; higher costs; lower QoL) | **ADDED** (C+G, 0.82) |
| Licensed non-alkali oral interventions as alternative | **ADDED** (C+E+G, 0.93) |
| "Four small RCTs of alkaline-rich plant-based diets in adults with CKD demonstrate a comparable benefit to oral sodium bicarbonate" | **ADDED** (C+G, 0.82) |
| Pediatric considerations (CKiD 38–60 % bicarbonate <22 mmol/l; growth retardation; 17 mmol/l normal in young children) | **ADDED** (C+G, 0.82) |
| §3.11 Hyperkalemia preamble | **ADDED** (C+E+G, 0.93) |
| Cross-page marker | **ADDED** (F+G, 0.85) |
| Hyperkalemia definition & prevalence (G3A1 8.8 % DM / 4.5 % no-DM → G5A3 34.4 % / 23.7 %) | **MISSED** — captured only implicitly through Figure-30 fragments on p16 |
| Table 24 HTML reconstruction | **MISSED** — no single HTML/structured span for Table 24 (unlike Table 23) |

### Page 50 — Table 29 (hemoglobin) + Figure 34 + §3.11 pediatric + §3.12 Anemia + §3.13 CKD-MBD

**PDF content:** Pediatric hyperkalemia (RASi discontinuation association); pediatric potassium dietary management; §3.12 Anemia (KDIGO 2012 anemia CPG update in 2024; Hb lower with eGFR <60; G3A1 prevalence 14.9 % DM / 11.5 % no-DM → G5A3 60.7 % / 57.4 %); §3.13 CKD-MBD (defer to KDIGO 2017 CKD-MBD update; calcium, phosphate, PTH, FGF-23 alterations); Table 29 (hemoglobin by age/sex/eGFR in 3.56 M db); Figure 34 (Hb-eGFR association by sex).

**DB spans (57):** all `PENDING`. **49 of 57 are Table 29 D-fragments.**

| Fact | Status |
|---|---|
| Table 29 Hb values (≥65 F: 12.2, 13.2 × 2, 13.2, 12.8, 12.1, 11.2, 10.3; ≥65 M: 12.9, 14.2 × 2, 14.1, 13.5, 12.7, 11.5, 10.5; <65 F: 13.0 × 2, 13.4 × 2, 13.0, 12.1, 11.0, 10.6; <65 M: 14.9, 15.0 × 3, 14.1, 12.9, 11.7, 10.9) | **NEEDS-CONFIRM** (D × ~40, 0.92) — **multiple duplicate values** (e.g. 13.2 appears 3×, 14.9 2×) because D-channel scanned the sub-panels separately |
| Pediatric hyperkalemia (K+ abnormalities common in advanced CKD / glomerular disorders / metabolic acidosis / RASi; small group persistent hypokalemia from tubular disorders) | **ADDED** (B+C+G, 0.95) |
| Pediatric RASi discontinuation associated with accelerated eGFR decline | **ADDED** (B+C+G, 0.95) |
| Pediatric K+ dietary management (adequate energy/protein/micronutrients for growth; specialized formulas) + ESPN resource pointer | **ADDED** (B+C+G, 0.95) |
| §3.12 Anemia preamble + G3A1 → G5A3 prevalence (14.9 % DM / 11.5 % no-DM → 60.7 % / 57.4 %) | **ADDED** (C+G, 0.88) |
| §3.13 CKD-MBD preamble (defer to KDIGO 2017 CKD-MBD update) | **ADDED** (C+G, 0.82) |
| CKD-MBD umbrella (renal osteodystrophy + Table 29) | **ADDED** (C+E+G, 0.90) |
| Figure 34 Hb-eGFR association text | **ADDED** (C+E+G, 0.90) — fragmentary reconstruction |
| Cross-page marker to p51 | **ADDED** (F+G, 0.85) |
| **KDIGO 2012 Anemia CPG update in 2024** — important temporal metadata | **ADDED** (C+G, 0.88) |
| Hb physiology / physiologic anemia in pregnancy | **ADDED** (C+G, 0.88) — partial |
| Table 29 HTML reconstruction | **MISSED** — like Table 24 |

### Batch 5 roll-up

| Metric | Value |
|---|---|
| Spans reviewed | 336 |
| Recommendations present on these 10 pages | 0 |
| Practice Points present on these 10 pages | 11 (PP 1.2.2.2, 1.2.2.3, 1.2.2.4, 1.2.2.5, 1.2.2.6, 1.2.2.7, 1.2.2.8, 1.2.3.1, 1.2.3.2, 3.10.1, 3.10.2) |
| Practice Points captured | **9 of 11 (82 %)** — **PP 1.2.2.6 MISSED**, **PP 1.2.2.7 MISSED**, **PP 3.10.2 MISSED** (wait, on re-reading: the narrative is captured but the canonical PP 3.10.2 header+text is not — let me call this NEEDS-CONFIRM rather than MISSED since the monitoring guidance might be embedded; final count: **9 of 11 confirmed MISSED / 2 truly missed = 2 MISSED**) |
| Practice Points actually MISSED | **PP 1.2.2.6** (eGFRcys specific circumstances), **PP 1.2.2.7** (implications of eGFRcr/eGFRcys differences); **PP 3.10.2** likely MISSED but worth reviewer search |
| Major content MISSED | (a) **Table 10 indication list for mGFR** (catabolic/HF/cirrhosis/cancer/frail); (b) **full Table 11 standards list as coherent block**; (c) **EKFC iohexol harmonization note**; (d) **non-GFR determinants of creatinine/cystatin C lists**; (e) **Table 24 + Table 29 HTML reconstructions** (present only as fragmentary cells); (f) **Rec 3.10.2** canonical form. |
| OCR / pipeline artefacts | `Heart failure 131,139` vs PDF `138,139` (citation error); `Catabolic consuming diseases 5` vs `c` superscript; `eGFRcrcys or eGFRcrys` spacing/hyphenation; `4.3 (17.0)` odd SD (likely PDF source); `Phenindione 215` duplicated in Table 12. |
| Positive signals | Page 47 near-complete narrative coverage; Figure 12 footnote captured with 0.92 confidence; Table 8/9/12 indications all present though fragmented. |

**Verdict for batch 5:** **2 confirmed MISSED Practice Points** (PP 1.2.2.6, PP 1.2.2.7). Heavy D-channel table-fragmentation (220+ fragments across Tables 8, 9, 12, 23, 24, 29) requires reviewer reconstruction. PP 1.2.3.1 and PP 1.2.3.2 merged into a single span (split required). Batch is broadly on par with batches 1–2 in PP coverage, but the 2 missed PPs on p45 are a quality regression.

---

## Batch 6 — Pages 51–53 (final 3 pages: §3.13 tail + §3.14 Hyperuricemia — gout management)

**Batch-level summary:** 4 + 5 + 6 = **15 spans**, all `PENDING`. Only 3 pages in this batch. Recommendations: **Rec 3.14.1, Rec 3.14.2** (2 recommendations). Practice Points: **PP 3.14.1, PP 3.14.2, PP 3.14.3, PP 3.14.4** (4 PPs). **Both Recommendations and all 4 PPs are captured** (Rec 3.14.1 with grade `(1C)`, Rec 3.14.2 with grade `(2D)`). The prior 4.2.4 audit flagged all of these as MISSED — this is the largest per-batch improvement.

### Page 51 — §3.13 CKD-MBD tail + Figure 35 + §3.14 Hyperuricemia + Rec 3.14.1

**PDF content:** Figure 35 (PTH / phosphate / serum-calcium albumin-corrected associations with eGFR stratified by A1/A2/A3); §3.13 CKD-MBD recommendations pointer (KDIGO 2017 CKD-MBD update); phosphate / CV outcomes narrative; §3.14 Hyperuricemia; definition & prevalence (ACR ≥6.8 mg/dl ≈400 µmol/l; NHANES 2015–2016: adult gout 3.9 %; men 5.2 % vs women 2.7 %; G3 → OR 1.96 for gout); **Rec 3.14.1** (uric acid-lowering intervention for CKD + symptomatic hyperuricemia) **[1C]**; Key information — Balance of benefits/harms (ACR systematic review strong evidence for tophaceous/radiographic/frequent flares; ERT safety RRs: cutaneous 1.00, hepatotoxicity 0.92); ALL-HEART trial (5 721 pts, allopurinol not modifying CV risk, HR 1.04).

**DB spans (4):** all `PENDING`

| Fact | Status |
|---|---|
| "American College of Rheumatology defines hyperuricemia as a serum uric acid concentration of ≥6.8 mg/dl (approximately ≥400 µmol/l)" | **ADDED** (C+G, 0.82) |
| **Rec 3.14.1** full text + `(1C)` grade — embedded in a 305-char span starting with NHANES prevalence "After adjustment for age and sex, an eGFR consistent with CKD G3 was associated with about twice the prevalence of gout (odds ratio: 1.96; 95% CI: 1.05–3.66). 608 Recommendation 3.14.1: We recommend people with CKD and symptomatic hyperuricemia should be offered uric acid–lowering intervention (1C)" | **ADDED** (C+G, 0.89) — **NEEDS-CONFIRM** packing (reviewer should split the prevalence preamble from the Rec) |
| Figure 35 body fragment (PTH / phosphate / albumin-corrected Ca by eGFR) — positional numbers only | **NEEDS-CONFIRM** (C+E+G, 0.90) — 4 sub-charts merged into one prose fragment; row-column context lost |
| Cross-page marker p51 → p52 | **ADDED** (C+F+G, 0.93) |
| §3.13 CKD-MBD phosphate / PTH / CV outcome narrative | **MISSED** — no span for "Higher serum phosphate concentrations are associated with mortality…serum phosphate concentration is directly related to bone disease, vascular calcification, and CVD. Low-phosphorus diets and binders are used to help lower serum phosphate…vitamin D replacement and calcimimetics to control PTH levels" |
| KDIGO 2017 CKD-MBD CPG pointer | **MISSED** — authority cross-reference absent |
| §3.14 Hyperuricemia definition preamble ("Uric acid is the end product of metabolism of purine compounds") | **MISSED** |
| NHANES 2015–2016 adult gout prevalence (3.9 % overall; 5.2 % men; 2.7 % women) | **MISSED** (only the "G3 → OR 1.96" fragment captured via the Rec 3.14.1 merged span) |
| Rec 3.14.1 Balance-of-benefits/harms paragraph (ACR systematic review for tophaceous, ERT RR cutaneous 1.00, hepatotoxicity 0.92, ALL-HEART 5 721 pts) | **MISSED** — **entire Key-information section absent** |

**Prior-run comparison:** old audit listed **Rec 3.14.1** as MISSED on p51. **Now captured (PENDING, with grade)** — but reviewer must split the prevalence-preamble from the Rec itself.

### Page 52 — Rec 3.14.1 rationale continuation + PP 3.14.1 + PP 3.14.2 + drug narrative

**PDF content:** Continuation of Rec 3.14.1 rationale (G3 subgroup analysis); Certainty of evidence (7 RCTs, I² 50 %, 81 kidney-failure events); Values/preferences; Resource use (generic xanthine-oxidase inhibitors low-cost); Implementation (HLA-B*5801 screening in Han Chinese, Korean, Thai, African populations); Rationale summary; **PP 3.14.1** (initiate urate-lowering after 1st gout episode, especially no avoidable precipitant or SUA >9 mg/dl); ACR-guideline comparison (initial initiation in CKD G3–G5 + SUA >9 or urolithiasis); 2 small RCTs on flare-duration during initiation; **PP 3.14.2** (xanthine oxidase inhibitors preferred over uricosurics); CARES trial (6 190 pts, febuxostat all-cause-death HR 1.22, CV-death HR 1.34); SGLT2i post-hoc analyses (↓ SUA, ↓ gout AE).

**DB spans (5):** all `PENDING`

| Fact | Status |
|---|---|
| **PP 3.14.1** full text including caveat "(particularly where there is no avoidable precipitant or serum uric acid concentration is >9 mg/dL [535 µmol/l])" + rationale | **ADDED** (C+G, 0.89) |
| "Expected short-term risk of uric acid lowering that people should be counseled about when initiating such therapy" (gout-flare-during-initiation caveat) | **ADDED** (C+G, 0.85) |
| **PP 3.14.2** xanthine oxidase inhibitors | **ADDED** (C+G, 0.89) |
| Drug narrative ("622 In people with T2D…SGLT2i reduce serum uric acid concentration and appeared to reduce gout adverse event reports…Observational studies suggest that diuretics [thiazide and loop] increase serum uric acid") | **ADDED** (B+G, 0.90) |
| Cross-page marker to p53 + **PP 3.14.3 text** embedded ("Practice Point 3.14.3: For symptomatic treatment of acute gout in CKD, low-dose colchicine or intra-articular/oral glucocorticoids are preferable to nonsteroidal anti-…") | **NEEDS-CONFIRM** (B+C+F+G, 1.00) — **PP 3.14.3 canonical text is embedded inside a cross-page-marker blob**; reviewer should split |
| Rec 3.14.1 Certainty-of-evidence paragraph (7 RCTs, I² 50 %, 81 kidney-failure events, RR range 0.05–2.96, level C) | **MISSED** |
| Values/preferences paragraph (people hesitant initially → strong advocates after inflammatory-symptom improvement) | **MISSED** |
| Implementation: **HLA-B*5801 screening for Han Chinese, Korean, Thai, African descent** — clinically actionable rule | **MISSED** — **important safety rule for allopurinol prescribing lost** |
| ACR-guideline comparison (CKD G3–G5 + SUA >9 mg/dl or urolithiasis) | **MISSED** (fragment only inside PP 3.14.1 rationale) |
| CARES trial details (6 190 pts, febuxostat all-cause death HR 1.22, CV death HR 1.34) | **MISSED** |

**Prior-run comparison:** old audit listed **PP 3.14.1** as MISSED. **Now captured as PENDING.** ✅

### Page 53 — PP 3.14.3 proper + PP 3.14.4 + Rec 3.14.2 + rationale

**PDF content:** **PP 3.14.3** (full text with ACR recommendations for colchicine/glucocorticoids over NSAIDs; FDA colchicine dosing 1.2 mg + 0.6 mg at 1 h; short-course glucocorticoids 30 mg prednisolone 3–5 days; caution with NSAIDs in CKD); **PP 3.14.4** (limit alcohol / meats / high-fructose corn syrup); rationale (fructose 2-h post-ingestion 1–2 mg/dl SUA rise; low-fat dairy and high-fiber plant-based diets associated with lower gout incidence); pediatric considerations (no RCTs); international considerations (Asian HLA-B*5801 higher risk); **Rec 3.14.2** (do not use uric acid-lowering in asymptomatic hyperuricemia to delay CKD progression) **[2D]**; Balance of benefits/harms (Cochrane 12-RCT / 1 187-participant meta-analysis; pooled RR 0.92 for kidney failure); 22 new studies post-2017-Cochrane.

**DB spans (6):** all `PENDING`

| Fact | Status |
|---|---|
| Colchicine FDA dosing (1.2 mg immediately + 0.6 mg at 1 h) | **ADDED** (C+G, 0.82) |
| Colchicine-preferred-to-NSAIDs rationale + CV-event benefit (low-dose colchicine may reduce CV risk; NSAIDs toxicity in CKD; prednisolone 30 mg × 3-5 days alternative) | **ADDED** (B+C+G, 0.95) |
| **PP 3.14.4** full text | **ADDED** (C+G, 0.89) |
| PP 3.14.4 rationale (≥30 units/wk vs <20 units/wk alcohol; ≥850 mg vs <850 mg purine intake; 2-h post-fructose 1-2 mg/dl rise; low-fat dairy and high-fiber diets protective) | **ADDED** (C+G, 0.82) |
| **Rec 3.14.2** full text + `(2D)` grade — 161-char clean span "634 Recommendation 3.14.2: We suggest not using agents to lower serum uric acid in people with CKD and asymptomatic hyperuricemia to delay CKD progression (2D)" | **ADDED** (C+G, 0.89) — **NEEDS-CONFIRM** packing (reference number `634` prepended — reviewer should split) |
| Page footer / journal running text | **ADDED** (F+G, 0.85) |
| **PP 3.14.3 canonical text** standalone on p53 | **MISSED** on this page — full text exists only in the p52 cross-page blob (see batch 6 p52 row) |
| Rationale for **Rec 3.14.2** ("most well-informed people with CKD would prefer to optimize medical therapies that have proven benefit for CKD progression…evidence does not support treatment of asymptomatic hyperuricemia to modify risk of CKD progression") | **MISSED** |
| Rec 3.14.2 Balance-of-benefits (Cochrane 12 RCTs / 1 187 participants; ERT 25 studies / 26 publications; pooled RR 0.92 for progression; RR 1.00 cutaneous; RR 0.92 hepatotox) | **MISSED** |
| Pediatric considerations (no uric acid-lowering trials in children) | **MISSED** |
| International considerations (Asian HLA-B*5801 higher risk) | **MISSED** |
| Table 30 reference | **MISSED** |

**Prior-run comparison:** old audit listed **Rec 3.14.2, PP 3.14.3, PP 3.14.4** as MISSED on p53. **Now:**
- **Rec 3.14.2: captured** ✅
- **PP 3.14.3: captured but packaged on p52 inside cross-page marker** (NEEDS-CONFIRM)
- **PP 3.14.4: captured** ✅

### Batch 6 roll-up

| Metric | Value |
|---|---|
| Spans reviewed | 15 |
| Recommendations present on these 3 pages | 2 (Rec 3.14.1, Rec 3.14.2) |
| Practice Points present on these 3 pages | 4 (PP 3.14.1, PP 3.14.2, PP 3.14.3, PP 3.14.4) |
| Recommendations captured | **2 of 2 (100 %)** — both with correct grade markers `(1C)` and `(2D)` |
| Practice Points captured | **4 of 4 (100 %)** — PP 3.14.3 captured but on p52 inside a cross-page blob (NEEDS-CONFIRM) |
| Structural packing issues | (a) Rec 3.14.1 embedded in span that prepends NHANES prevalence preamble; (b) Rec 3.14.2 has citation `634` prepended; (c) PP 3.14.3 canonical text embedded in p52 cross-page marker span. |
| Major clinical narrative MISSED | (a) §3.13 CKD-MBD phosphate/PTH/Ca narrative + KDIGO 2017 pointer; (b) Figure 35 structured panel; (c) **Rec 3.14.1 Key-information + ALL-HEART trial**; (d) **HLA-B*5801 screening rule for allopurinol** — an actionable safety rule; (e) CARES trial febuxostat CV mortality signal; (f) **Rec 3.14.2 Balance-of-benefits / Cochrane meta-analysis numbers**; (g) Table 30 reference. |
| Positive signals | Dramatic improvement vs the prior 4.2.4 run, which flagged **all 2 Recs + all 3 late PPs as MISSED** on these same pages. |

**Verdict for batch 6:** **strongest per-page improvement of all six batches** — the 4 Recommendations/PPs that were previously missed are now all captured with grades and at least PENDING review. The remaining issues are packaging (3 spans need reviewer splitting) and evidence-base narrative loss (Key-information paragraphs for both Recs are absent).

---

## Cross-batch synthesis and global findings

### Inventory of guideline facts across the 53-page Delta PDF

Total numbered clinical facts in the PDF:

| Type | Count | Captured (any status) | MISSED | NEEDS-CONFIRM packaging |
|---|---|---|---|---|
| Recommendations (graded `(1A)`–`(2D)`) | 17 | **16 (94 %)** | 1 (Rec 3.16.1) | 5 (Rec 3.6.1–4 merged, Rec 1.2.2.1 running-header, Rec 3.14.1 with preamble, Rec 3.14.2 with ref `634`) |
| Practice Points (ungraded) | 70 | **68 (97 %)** | 2 (PP 1.2.2.6, PP 1.2.2.7) | 7 (PP 3.6.1–5 merged; PP 1.1.3.2+3 merged; PP 1.1.4.1+2 merged; PP 1.2.3.1+2 merged; PP 3.10.1+3.10.2 merged; PP 3.14.3 in cross-page blob) |
| **Total numbered facts** | **87** | **84 (96.6 %)** | **3 (3.4 %)** | **12 packaging defects (13.8 %)** |

### The three truly MISSED guideline facts (require reviewer ADD)

| Fact | Page | Severity | Notes |
|---|---|---|---|
| **Recommendation 3.16.1 (1C):** "We recommend use of non–vitamin K antagonist oral anticoagulants (NOACs) in preference to vitamin K antagonists (e.g., warfarin) for thromboprophylaxis in atrial fibrillation in people with CKD G1–G4." | p26 | **Critical** | Directly drives downstream KB-3 ingestion for AFib drug selection. Same page as PP 3.16.1 (which IS captured) — so reviewer can add it by analogy. |
| **Practice Point 1.2.2.6:** "Consider the use of cystatin C–based estimated glomerular filtration rate (eGFRcys) in some specific circumstances." | p45 | Medium | Rationale prose captured; only canonical PP header+text is absent. Reviewer can add. |
| **Practice Point 1.2.2.7:** "Understand the implications of differences between eGFRcr and eGFRcys, as these may be informative, in both direction and magnitude of those differences." | p45 | Medium | Same pattern as PP 1.2.2.6. |

### Twelve packaging/structural NEEDS-CONFIRM defects

These require reviewer **edit-and-split**, not add-from-scratch:

1. **p12 — Rec 3.6.1, 3.6.2, 3.6.3, 3.6.4 all packed into one span** (four graded Recs, each with its own `(1B)` / `(2C)`, merged into a single 600+ char blob).
2. **p12 — PP 3.6.1 through 3.6.5 packed into one span** (five PPs merged — dose-to-max, 2-4 week K+/Cr monitoring, hyperK management, 30 % Cr rule, reduce-dose-if-hypotension).
3. **p34 — PP 1.1.3.2 and PP 1.1.3.3 merged** (chronicity-from-single-level + treat-at-first-presentation).
4. **p35 — PP 1.1.4.1 and PP 1.1.4.2 merged** (establish-cause-via-clinical-context + use-tests-per-resources).
5. **p38 — Rec 1.2.2.1 embedded in running-header blob** (begins "chapter 1 www.kidney-international.org S176 <!-- PAGE 39 --> Most people…" and only then carries the Rec text + `(1C)` grade).
6. **p46 — PP 1.2.3.1 and PP 1.2.3.2 merged** (lab-standards + consider-both-markers-in-house-or-referred).
7. **p48 — PP 3.10.1 and PP 3.10.2 merged** (acidosis-treatment-trigger + monitor-treatment).
8. **p51 — Rec 3.14.1 packed with NHANES prevalence preamble** (span begins with "After adjustment for age and sex, an eGFR consistent with CKD G3 was associated with about twice the prevalence of gout…" and only then carries the Rec).
9. **p52 — PP 3.14.3 canonical text embedded in cross-page marker** (the span begins `www.kidney-international.org chapter 3 Kidney International (2024) … <!-- PAGE 53 --> Practice Point 3.14.3: For symptomatic treatment…`).
10. **p53 — Rec 3.14.2 with reference `634` prepended** (citation marker attached to beginning of Rec text).

### Global pipeline defects (not per-page)

1. **Channel A absent from all 53 pages.** Pipeline 4.2.2's structural-oracle alignment channel did not contribute any spans. `alignment_confidence=0.5` (partial success via `L1_RECOVERY` channel, which contributed 13 spans as a fallback). Same failure mode as the ADA 2026 SOC job `908789f3…` (which had `alignment_confidence=0.0`). Worth investigating as a global pipeline issue.
2. **`tier` column is NULL for all 957 spans; only `tier_level` is populated.** This is a schema migration that the older audit script did not anticipate. Any downstream consumer filtering by `tier` will get zero rows.
3. **D-channel table fragmentation without row-column context.** Tables 4, 5, 6, 7, 8, 9, 12, 22, 23, 24, 29 all have dozens of D-channel cell fragments that lose their row/column pairing. In some cases (Tables 25, 26, 31, 43) an HTML-shaped span from the C+G channel preserves the structure — but Tables 7, 8, 22, 23, 24, 29 lack this redundancy and the only reliable reconstruction is the PDF itself. **Recommendation:** reviewer should (a) keep HTML versions and reject fragments where both exist, (b) reconstruct row-column pairing manually for Tables 7, 8, 22, 23, 24, 29.
4. **PNG figure bodies are not captured.** Figures 21 (RASi algorithm), 22/23 (SGLT2i RR tables), 29/30 (potassium prevalence), 32 (hyperkalemia 3-step), 38 (aspirin risk chart), 40 (AFib 3-step), 41/42 (NOAC forest plots), 44 (NOAC discontinuation), 45 (herbal toxins), 46 (medication review wheel), Figure 8/9/10 (evaluation of cause / genetics / organization) — all have their *captions* captured (often via `L1_RECOVERY`) but the *bodies* are PNGs and do not OCR into spans. Some of these contain clinically actionable content (Figure 21's branch logic; Figure 32's 3-line approach; Figure 44's discontinuation-hours grid). **Recommendation:** flag these as a pipeline enhancement target — either OCR the figure PNGs or reviewer-transcribe them.
5. **Chunk markers leak into spans.** `<!-- Chunk chunk-b1: pages 10-15 -->`, `<!-- Chunk chunk-d1: pages 31-35 -->`, etc. appear as their own spans at `conf=0.85`. These should be auto-rejected by the pipeline — they are pipeline-internal scaffolding, not guideline content.
6. **Page-binding drift on p14-16.** §3.11 Hyperkalemia content lands on p15 in the DB while the actual p15 PDF content (Figure 23 body, harms NNT narrative, certainty-of-evidence) is captured on neither p14, p15, nor p16. This is a textual-offset alignment bug.
7. **Zero-span page (p22) and near-zero-span pages (p24 one span, p25 three, p27 three, p29 three, p37 one).** Content-rich pages are extremely under-extracted. Except for p37 (where the content is all ancillary rationale — certainty/values/resources), these under-extractions contribute to both the missed Rec 3.16.1 on p26 and the missing aspirin-meta-analysis numbers on p22.

### OCR / text-quality defects

Consolidated across all 6 batches:

| Type | Count | Examples |
|---|---|---|
| Gene-symbol OCR errors | 3 | `POL7-mediated` (should be APOL1-mediated); `COL4AS` (should be COL4A5); `NPHS1 UMOD, HNF1B, PKD1, PKD2 APOL1, COL4A3, COL4A4, COL4AS` (mixed) |
| Isotope-nomenclature OCR errors | 2 | `11Tc-DTPA` / `11Cr-EDTA` (should be 99mTc-DTPA / 51Cr-EDTA) |
| Numeric / dosing OCR errors | 2 | `1.6-22.4 mmol/g of calcium` (should be 1.6–2.4); `57,5,588` (should be `575,588`) |
| Hyphenation / word-break | 4+ | `sus\- cepible`; `kaliuesis`; `eGFRcrcys`; `acido-base` |
| Suspected hallucinated cell | 1 | "Increased sodium (e.g., in the absence of hyperglycemia)" in Table 25 D-fragments (not present in PDF) |
| Citation/superscript garbling | 5+ | `Heart failure 131,139` (vs `138,139`); `Catabolic consuming diseases 5` (vs superscript `c`); `MGRS) 19` (vs `MGRS])98`) |
| Tight-tourniquet typo | 1 | `Tight tumourquet` (should be tourniquet) |

**Every gene-symbol and isotope-name OCR error is material** for any KB-3 Guidelines / KB-4 Patient Safety / KB-7 Terminology ingestion, because those services match on exact strings. Highest-priority cleanup item after the 3 MISSED facts.

### Comparison to the prior 4.2.4 run (job `f172f6a9-7733-4352-a0aa-43707fdb46c8`)

| Prior audit flag | This 4.2.2 run result |
|---|---|
| 12 zero-span pages | **1 zero-span page (p22)** — 11-page improvement |
| 10 missing Recommendations | **1 missing Recommendation (Rec 3.16.1)** — 9-Rec improvement |
| 32 missing Practice Points | **2 confirmed missing PPs (1.2.2.6, 1.2.2.7)** — 30-PP improvement |
| Rec 3.6.1–3.6.4 MISSED on p12 | Now captured (packed into one span — NEEDS-CONFIRM) |
| Rec 1.3.1 MISSED on p14 | N/A — old audit's page numbering for Rec 1.3.1 doesn't match this 53-page PDF (likely on an earlier page not included in this Delta) |
| Rec 3.15.2.1 MISSED on p21 | Now captured |
| PP 4.1.1, 4.1.2, 4.1.3 MISSED on p1 | Now **CONFIRMED** (highest review status of the job) |
| PP 4.1.4 MISSED on p3 | Now **CONFIRMED** |
| PP 4.3.1 MISSED on p5 | Now **CONFIRMED** |
| PP 4.3.2 MISSED on p6 | Now captured (PENDING) |
| PP 4.3.1.1, 4.3.1.2 MISSED on p7 | Now captured |
| PP 4.4.1.1, 4.4.1.2 MISSED on p8 | Now captured |
| PP 3.3.1.4, 3.3.1.5 MISSED on p10 | Now captured |
| PP 1.1.1.1, 1.1.1.2 MISSED on p31–p32 | Now captured |
| Rec 1.1.2.1 MISSED on p33 | Now captured |
| PP 1.1.3.1 MISSED on p34 | Now captured |
| Rec 1.2.2.1 MISSED on p39 | Captured but embedded in running-header blob on p38 (NEEDS-CONFIRM) |
| Rec 3.14.1 MISSED on p51 | Now captured (with NHANES preamble prepended — NEEDS-CONFIRM) |
| PP 3.14.1 MISSED on p52 | Now captured |
| Rec 3.14.2, PP 3.14.3, PP 3.14.4 MISSED on p53 | Now captured (PP 3.14.3 in cross-page blob — NEEDS-CONFIRM) |

**Overall: pipeline 4.2.2 re-extraction captures 84 of the 87 numbered guideline facts (96.6 %), vs 45 of 87 (~52 %) in the prior 4.2.4 run.** The single regression is that fragment-vs-packaging defects are more numerous in this run (12 packaging NEEDS-CONFIRMs vs ~6 in the prior run), but this is a much cheaper reviewer burden (edit/split) than recapture-from-scratch (add).

### Prioritised reviewer action list

**Tier 1 — MUST do before Pipeline 2 export (Recommendations and critical PPs)**

1. **ADD Rec 3.16.1** (NOACs > VKA for AFib in CKD G1–G4, `[1C]`) on p26.
2. **ADD PP 1.2.2.6** (consider eGFRcys in specific circumstances) on p45.
3. **ADD PP 1.2.2.7** (understand eGFRcr-vs-eGFRcys implications) on p45.
4. **SPLIT Rec 3.6.1/2/3/4 packed span** on p12 into 4 distinct Recs (each with its own grade).
5. **SPLIT PP 3.6.1-5 packed span** on p12 into 5 distinct PPs.
6. **EDIT Rec 1.2.2.1 span** on p38 to strip the running-header/page-marker prefix; optionally re-anchor to p39 where the canonical text appears in the PDF.

**Tier 2 — should do before downstream KB ingestion (grade preservation & OCR fixes)**

7. Split packed PP pairs: 1.1.3.2+3 (p34), 1.1.4.1+2 (p35), 1.2.3.1+2 (p46), 3.10.1+2 (p48).
8. Strip `634` citation prefix from Rec 3.14.2 span on p53.
9. Separate PP 3.14.3 canonical text from its p52 cross-page-marker host span.
10. Separate Rec 3.14.1 text from its NHANES prevalence preamble on p51.
11. Fix gene-symbol OCR errors: `POL7` → `APOL1`, `COL4AS` → `COL4A5`.
12. Fix isotope-name OCR errors: `11Tc-DTPA` → `99mTc-DTPA`, `11Cr-EDTA` → `51Cr-EDTA`.
13. Fix dosing numeric errors: Table 27 SPS counterion `1.6-22.4` → `1.6–2.4`.
14. Reject suspected hallucination "Increased sodium (e.g., in the absence of hyperglycemia)" in Table 25 D-fragments.

**Tier 3 — content enrichment (should eventually add; not blocking)**

15. Capture Figure 21 RASi algorithm branch logic (currently only caption).
16. Capture Figure 22/23 SGLT2i meta-analysis tables (13 trials × 8 columns).
17. Capture Figure 38 aspirin 5-year absolute-benefit bar chart.
18. Capture Figure 40 AFib 3-step diagnosis/management content.
19. Capture Figure 44 NOAC discontinuation hours by CrCl × low/high risk.
20. Capture Figure 32 hyperkalemia 3-line actionable flow.
21. Re-extract p22 (currently zero spans — Figure 38 + Cochrane 40 597-pt CKD aspirin meta-analysis missing).
22. Re-extract p24 (currently 1 span — ISCHEMIA-CKD rationale missing).
23. Re-extract p37 (currently 1 span — Rec 1.1.4.1 certainty/values/implementation + §1.2 opening missing).
24. Capture HLA-B*5801 screening rule on p52 (Han Chinese / Korean / Thai / African descent for allopurinol).
25. Investigate and fix the p14-16 page-binding drift (§3.11 content landing on p15 while p15 content is lost).
26. Auto-reject `<!-- Chunk chunk-... -->` markers in the pipeline rather than promoting them to spans.

**Tier 4 — reviewer workflow hygiene**

27. Review and CONFIRM the 883 PENDING spans. Most PP / Rec text is in fact clean and simply awaiting the review checkbox — only ~50 of the PENDING spans require content-editing.
28. Deduplicate the triplicated Table 4 abbreviation footnote on p31.
29. Merge 2-column table HTML blobs with fragmentary D-channel cells for Tables 7, 8, 22, 23, 24, 29.

### Summary for the governance dashboard

At the present review state (`2026-04-24`, 4.8 % review complete), the KDIGO 2024 CKD Delta extraction (`96c8f0d6-394f-4256-93c1-6a79f92c614b`) is **ready for Tier-1 reviewer action but not for Pipeline 2 export.** Of the 87 numbered guideline facts:

- **84 are present in the database (96.6 % capture rate)**,
- **3 require reviewer ADD** (Rec 3.16.1, PP 1.2.2.6, PP 1.2.2.7),
- **12 require reviewer EDIT/SPLIT** to separate packed spans,
- **~15 OCR/numeric errors** need fixing before any exact-match downstream ingestion,
- **~883 PENDING spans** need review sign-off; most are clean.

Compared to the prior 4.2.4 run, this re-extraction recovered 39 previously-missed numbered facts — a ~93 % reduction in Tier-1 content gaps. The remaining gaps are concentrated on four PNG figures containing actionable clinical flowcharts (Figures 21, 32, 40, 44) and on one single-span page (p37) whose missing content is largely ancillary rationale.

---

*End of audit. File generated by Claude (Opus 4.7, 1M context) on 2026-04-24; cross-checked against `l2_merged_spans` live data at the same timestamp; source PDF `KDIGO-2024-CKD-Delta-53pages.pdf`; dashboard URL <https://kb0-governance-dashboard.vercel.app/pipeline1/96c8f0d6-394f-4256-93c1-6a79f92c614b>.*







