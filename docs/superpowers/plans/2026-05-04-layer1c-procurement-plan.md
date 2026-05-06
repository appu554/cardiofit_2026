# Layer 1C Regulatory Source Procurement — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Procure 6 download-procurable Australian aged care regulatory PDFs into a new `kb-3-guidelines/knowledge/au/regulatory/` tree, with 8 per-source `PROCUREMENT.md` runbooks (6 procurable + 2 engagement-required) and a top-level `MANIFEST.md`. Verifiable by §8 acceptance criteria of the design spec.

**Architecture:** Static directory scaffolding (markdown runbooks + .gitignore + manifest) followed by Playwright-driven PDF procurement for each source, then SHA-256 hashing and manifest updates. No code changes — pure content authoring + browser-mediated procurement. Pattern lifted from `wave6/acsqhc_ams/PROCUREMENT.md`.

**Tech Stack:** Markdown, bash, Playwright MCP (browser_navigate + browser_evaluate fetch+base64 decode), shasum.

**Spec:** `docs/superpowers/specs/2026-05-04-layer1c-procurement-design.md`

---

## File Structure

**Created (markdown runbooks + manifest):**
- `kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md`
- `kb-3-guidelines/knowledge/au/regulatory/.gitignore`
- `kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024/PROCUREMENT.md`
- `kb-3-guidelines/knowledge/au/regulatory/commonwealth/restrictive_practice/PROCUREMENT.md`
- `kb-3-guidelines/knowledge/au/regulatory/commonwealth/mhr_sharing_by_default_2025/PROCUREMENT.md`
- `kb-3-guidelines/knowledge/au/regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025/PROCUREMENT.md`
- `kb-3-guidelines/knowledge/au/regulatory/states/tas/co_prescribing_pilot_2026/PROCUREMENT.md`
- `kb-3-guidelines/knowledge/au/regulatory/professional_standards/nmba_designated_rn_prescriber/PROCUREMENT.md`
- `kb-3-guidelines/knowledge/au/regulatory/professional_standards/apc_acop_training/PROCUREMENT.md`
- `kb-3-guidelines/knowledge/au/regulatory/frameworks/pharma_care_unisa/PROCUREMENT.md`

**Procured (PDFs, gitignored — not committed):**
- `commonwealth/aged_care_act_2024/Aged-Care-Act-2024.pdf` (+ Rules + Strengthened Standards)
- `commonwealth/restrictive_practice/*.pdf`
- `commonwealth/mhr_sharing_by_default_2025/MHR-Sharing-by-Default-Act-2025.pdf`
- `states/vic/pcw_exclusion_dpcs_amendment_2025/DPCS-Amendment-Medication-Administration-RAC-Act-2025.pdf`
- `professional_standards/nmba_designated_rn_prescriber/NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf`
- `professional_standards/apc_acop_training/APC-ACOP-Training-Accreditation-Standard.pdf`

**Convention:** Repository root for all paths is `/Volumes/Vaidshala/cardiofit/`. Within tasks, paths are relative to repo root unless stated otherwise.

---

## Task 1: Create directory tree

**Files:**
- Create directories only (10 leaf folders)

- [ ] **Step 1: Create the regulatory directory tree**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au

mkdir -p regulatory/commonwealth/aged_care_act_2024
mkdir -p regulatory/commonwealth/restrictive_practice
mkdir -p regulatory/commonwealth/mhr_sharing_by_default_2025
mkdir -p regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025
mkdir -p regulatory/states/tas/co_prescribing_pilot_2026
mkdir -p regulatory/professional_standards/nmba_designated_rn_prescriber
mkdir -p regulatory/professional_standards/apc_acop_training
mkdir -p regulatory/frameworks/pharma_care_unisa
```

- [ ] **Step 2: Verify structure**

```bash
find /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory -type d | sort
```

Expected output: 11 lines (regulatory + 10 subdirectories below it).

---

## Task 2: Create regulatory/.gitignore

**Files:**
- Create: `kb-3-guidelines/knowledge/au/regulatory/.gitignore`

- [ ] **Step 1: Write .gitignore**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/.gitignore`

Content:

```
# Layer 1C regulatory PDFs are NOT committed to the repo.
# PROCUREMENT.md runbooks document how to (re)fetch, MANIFEST.md tracks state.
*.pdf
```

- [ ] **Step 2: Verify**

```bash
cat /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/.gitignore
```

Expected: 3 lines including `*.pdf`.

---

## Task 3: Create regulatory/MANIFEST.md

**Files:**
- Create: `kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md`

- [ ] **Step 1: Write MANIFEST.md**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md`

Content:

```markdown
# Layer 1C — Australian Aged Care Regulatory Source Manifest

**Spec:** `kb-6-formulary/Layer1_v2_Australian_Aged_Care_Implementation_Guidelines.md` Part 4
**Design:** `docs/superpowers/specs/2026-05-04-layer1c-procurement-design.md`
**Last updated:** 2026-05-04
**Phase:** 1C-α (procurement)

## Procurement state

| # | Source | Tier | Jurisdiction | Status | PDFs | Updated |
|---|---|---|---|---|---|---|
| 1 | Aged Care Act 2024 + Rules 2025 + Strengthened Quality Standards | 1 | National | ⏳ pending | 0 | — |
| 2 | DPCS Amendment Act 2025 (VIC PCW exclusion) | 1 | VIC | ⏳ pending | 0 | — |
| 3 | NMBA Designated RN Prescriber Standard | 2 | National | ⏳ pending | 0 | — |
| 4 | Tasmanian co-prescribing pilot | 1 | TAS | 🔒 engagement-required | 0 | — |
| 5 | APC ACOP training accreditation | 2 | National | ⏳ pending | 0 | — |
| 6 | PHARMA-Care National Quality Framework | 3 | National | 🔒 engagement-required | 0 | — |
| 7 | Restrictive Practice regulations | 1 | National | ⏳ pending | 0 | — |
| 8 | Modernising MHR (Sharing by Default) Act 2025 | 1 | National | ⏳ pending | 0 | — |

## Status legend

- ⏳ pending — runbook drafted, PDFs not yet on disk
- ✅ landed — PDFs on disk and verified
- 🔒 engagement-required — procurement blocked on partnership / EOI engagement
- ❌ blocked — procurement attempted, failed; see linked PROCUREMENT.md

## Phase progress

- [ ] 1C-α — Procurement (this phase)
- [ ] 1C-β — Source Registry rows (deferred)
- [ ] 1C-γ — Structured rule extraction (deferred)
- [ ] 1C-δ — ScopeRules engine + Credential ledger + Consent state machine (deferred)

## Authority tiers

Per Layer 1 v2 §1.2:
- **Tier 1** — primary regulator/legislature (Commonwealth Acts, State Acts, Aged Care Quality Commission)
- **Tier 2** — peak professional body (NMBA, APC, PSA)
- **Tier 3** — academic / research (PHARMA-Care framework)
- **Tier 4** — facility-level policy (none in this category)
```

- [ ] **Step 2: Verify**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md && wc -l /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md
```

Expected: file exists, ≥30 lines.

---

## Task 4: Aged Care Act 2024 PROCUREMENT.md

**Files:**
- Create: `regulatory/commonwealth/aged_care_act_2024/PROCUREMENT.md`

- [ ] **Step 1: Write PROCUREMENT.md**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024/PROCUREMENT.md`

Content:

```markdown
# Aged Care Act 2024 + Aged Care Rules 2025 + Strengthened Quality Standards — Procurement Runbook

**Status (2026-05-04):** ⏳ pending
**Authority tier:** 1 (Commonwealth legislature + Aged Care Quality and Safety Commission)
**Jurisdiction scope:** national
**Effective period:** Aged Care Act 2024 commenced 2025-11-01; Aged Care Rules 2025 iterating; Strengthened Quality Standards in force.
**Reproduction terms:** Crown copyright Commonwealth of Australia. Free for reference, attribution required. Commercial reproduction permitted under Creative Commons Attribution 4.0 International (CC BY 4.0) per Federal Register of Legislation policy.
**Layer 1 v2 spec section:** §4.1
**Spec deadline:** none (already in force)

## What to download

This regulatory regime spans three primary documents:

1. **Aged Care Act 2024** (Cth)
   - Source: https://www.legislation.gov.au/ — search "Aged Care Act 2024"
   - Why: enabling legislation for the entire post-2025 aged care regulatory regime
   - Maps to: KB-0 governance (Source Registry); Authorisation state machine (statutory authority basis); Consent state machine (statutory consent requirements); KB-13 quality measures (statutory reporting obligations)

2. **Aged Care Rules 2025** (Cth, subordinate legislation, ~900+ pages, iterating)
   - Source: https://www.legislation.gov.au/ — search "Aged Care Rules 2025"
   - Why: operational rules implementing the Act. Medication-management-relevant sections feed structured rule data
   - Maps to: ScopeRules data (restrictive practice authorisation, worker screening, clinical governance)

3. **Strengthened Aged Care Quality Standards** (ACQSC, in force)
   - Source: https://www.agedcarequality.gov.au/providers/standards/strengthened-quality-standards
   - Why: Standard 5 (Clinical Care) is the primary driver of platform-facility alignment. Audit trail and EvidenceTrace graph produce Standard 5 evidence as workflow exhaust per Layer 1 v2 §4.1
   - Maps to: KB-13 quality measures; KB-18 governance / audit trail design; MVP-6 Standard 5 evidence panel

## Code path (Playwright)

```
1. browser_navigate → https://www.legislation.gov.au/
2. Search for "Aged Care Act 2024" — locate F-register PDF link
3. browser_evaluate with fetch() + base64 → decode locally to:
   Aged-Care-Act-2024.pdf
4. Repeat for Aged-Care-Rules-2025.pdf (likely multi-PDF; capture all parts)
5. browser_navigate → https://www.agedcarequality.gov.au/providers/standards/strengthened-quality-standards
6. Locate "Strengthened Quality Standards" PDF; fetch+base64; decode to:
   Strengthened-Aged-Care-Quality-Standards-2025.pdf
```

## Manual fallback

1. Open https://www.legislation.gov.au/ in a normal browser.
2. Search for "Aged Care Act 2024"; download the PDF; save as `Aged-Care-Act-2024.pdf`.
3. Search for "Aged Care Rules 2025"; download all parts; save as `Aged-Care-Rules-2025-part-N.pdf`.
4. Open https://www.agedcarequality.gov.au/providers/standards/strengthened-quality-standards.
5. Download "Strengthened Quality Standards" PDF; save as `Strengthened-Aged-Care-Quality-Standards-2025.pdf`.
6. Save all to: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024/`.

## After PDFs land

- [ ] Files verified at expected paths
- [ ] SHA-256 hashes recorded below
- [ ] regulatory/MANIFEST.md row 1 updated to ✅ landed
- [ ] Source Registry seed (deferred to Phase 1C-β)
- [ ] Pipeline 1 / manual extraction (deferred to Phase 1C-γ)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| _populated after procurement_ | | |
```

- [ ] **Step 2: Verify**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024/PROCUREMENT.md && head -1 /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024/PROCUREMENT.md
```

Expected: file exists, first line is the H1 heading.

---

## Task 5: Restrictive Practice PROCUREMENT.md

**Files:**
- Create: `regulatory/commonwealth/restrictive_practice/PROCUREMENT.md`

- [ ] **Step 1: Write PROCUREMENT.md**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/restrictive_practice/PROCUREMENT.md`

Content:

```markdown
# Restrictive Practice Regulations — Procurement Runbook

**Status (2026-05-04):** ⏳ pending
**Authority tier:** 1 (Commonwealth legislature + ACQSC)
**Jurisdiction scope:** national
**Effective period:** core legislation in force since 2019; amended by Aged Care and Other Legislation Amendment (Royal Commission Response) Act 2022; subsequent regulations iterating.
**Reproduction terms:** Crown copyright Commonwealth. CC BY 4.0 for the legislation; ACQSC guidance documents under similar permissive licensing.
**Layer 1 v2 spec section:** §4.3
**Spec deadline:** none (already in force; load-bearing for Consent state machine in Phase 1C-δ)

## What to download

1. **Aged Care and Other Legislation Amendment (Royal Commission Response) Act 2022**
   - Source: https://www.legislation.gov.au/
   - Why: introduced consolidated restrictive-practice authorisation framework
   - Maps to: Consent state machine inputs; Authorisation state machine (psychotropic + chemical restraint authority gating)

2. **Quality of Care Principles 2014 — Part 4A Restrictive Practices** (as amended)
   - Source: https://www.legislation.gov.au/
   - Why: operational definitions of behaviour-support plan, informed consent, monitoring requirements
   - Maps to: KB-29 psychotropic deprescribing template; Consent state machine

3. **ACQSC Guidance — Minimising Restrictive Practices**
   - Source: https://www.agedcarequality.gov.au/providers/standards/restrictive-practices
   - Why: regulator-facing operational guidance on what evidence the platform must produce
   - Maps to: KB-13 quality indicators; audit trail design

## Code path (Playwright)

```
1. browser_navigate → https://www.legislation.gov.au/
2. Search "Aged Care and Other Legislation Amendment (Royal Commission Response) Act 2022"
3. browser_evaluate with fetch()+base64 → decode to:
   Aged-Care-Royal-Commission-Response-Act-2022.pdf
4. Search "Quality of Care Principles 2014" → fetch latest compilation → save as:
   Quality-of-Care-Principles-2014-compilation.pdf
5. browser_navigate → https://www.agedcarequality.gov.au/providers/standards/restrictive-practices
6. Download published ACQSC restrictive-practices guidance PDFs; save with descriptive names
```

## Manual fallback

1. Browse https://www.legislation.gov.au/; search by short title; download PDF.
2. Browse https://www.agedcarequality.gov.au/providers/standards/restrictive-practices; download linked PDFs.
3. Save all to: `regulatory/commonwealth/restrictive_practice/`.

## After PDFs land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 7 updated to ✅ landed
- [ ] Source Registry seed (Phase 1C-β)
- [ ] Consent state machine inputs encoded (Phase 1C-δ)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| _populated after procurement_ | | |
```

- [ ] **Step 2: Verify**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/restrictive_practice/PROCUREMENT.md
```

Expected: exit code 0.

---

## Task 6: MHR Sharing-by-Default PROCUREMENT.md

**Files:**
- Create: `regulatory/commonwealth/mhr_sharing_by_default_2025/PROCUREMENT.md`

- [ ] **Step 1: Write PROCUREMENT.md**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/mhr_sharing_by_default_2025/PROCUREMENT.md`

Content:

```markdown
# Modernising My Health Record (Sharing by Default) Act 2025 — Procurement Runbook

**Status (2026-05-04):** ⏳ pending
**Authority tier:** 1 (Commonwealth legislature)
**Jurisdiction scope:** national
**Effective period:** Royal Assent 2025-02-14. Mandatory pathology + diagnostic imaging upload to MHR commences 2026-07-01. Civil penalties: 250 penalty units (~AUD 82,500) non-registration; 30 penalty units (~AUD 9,900) non-compliant upload.
**Reproduction terms:** Crown copyright Commonwealth. CC BY 4.0.
**Layer 1 v2 spec section:** §4.10
**Spec deadline:** mandatory pathology upload begins 2026-07-01 (~8 weeks from spec date)

## What to download

1. **Modernising My Health Record (Sharing by Default) Act 2025**
   - Source: https://www.legislation.gov.au/
   - Why: regulatory backbone behind pathology integration simplification per spec §3.2. Removes the per-pathology-vendor integration burden by mandating MHR upload by default.
   - Maps to: Layer 1B MHR FHIR Gateway integration (deferred); Source Registry; Authorisation state machine (consent-aware MHR access)

2. **Sharing by Default Rules** (subordinate instrument, expected iterative release)
   - Source: https://www.legislation.gov.au/
   - Why: operational definitions of which information types fall under "default sharing" and how consent overrides apply
   - Maps to: ingestion contracts for MHR FHIR Gateway

3. **Department of Health implementation guidance**
   - Source: https://www.health.gov.au/ — "Better and Faster Access to health information"
   - Why: spec §4.10 directs quarterly monitoring of this page for Sharing by Default Rule extensions; each extension is potentially a new Layer 1 source

## Code path (Playwright)

```
1. browser_navigate → https://www.legislation.gov.au/
2. Search "Modernising My Health Record" → locate Act PDF
3. fetch()+base64 → decode to: MHR-Sharing-by-Default-Act-2025.pdf
4. Search for associated subordinate Rules → fetch+decode if available
5. browser_navigate → Department of Health "Better and Faster Access" landing page
6. Download published implementation guidance PDFs
```

## Manual fallback

1. Browse https://www.legislation.gov.au/; search "Modernising My Health Record".
2. Download Act + any Rules PDFs.
3. Browse https://www.health.gov.au/; locate Better and Faster Access pages; download guidance PDFs.
4. Save to: `regulatory/commonwealth/mhr_sharing_by_default_2025/`.

## After PDFs land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 8 updated to ✅ landed
- [ ] Source Registry seed (Phase 1C-β)
- [ ] Layer 1B MHR FHIR Gateway integration (separate workstream)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| _populated after procurement_ | | |
```

- [ ] **Step 2: Verify**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/mhr_sharing_by_default_2025/PROCUREMENT.md
```

Expected: exit code 0.

---

## Task 7: Victorian PCW Exclusion PROCUREMENT.md

**Files:**
- Create: `regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025/PROCUREMENT.md`

- [ ] **Step 1: Write PROCUREMENT.md**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025/PROCUREMENT.md`

Content:

```markdown
# Drugs, Poisons and Controlled Substances Amendment (Medication Administration in Residential Aged Care) Act 2025 — Procurement Runbook

**Status (2026-05-04):** ⏳ pending  ⚠️ HIGH PRIORITY: enforcement begins 2026-09-29
**Authority tier:** 1 (State legislature — Victoria)
**Jurisdiction scope:** VIC only
**Effective period:** passed September 2025; commences 2026-07-01; 90-day grace period to 2026-09-29; enforcement from 2026-09-29.
**Reproduction terms:** Crown copyright Victoria. Open access on legislation.vic.gov.au; commercial reproduction generally permitted with attribution.
**Layer 1 v2 spec section:** §4.4
**Spec deadline:** **2026-09-29 enforcement** (~5 months from spec date). ScopeRules data must be authored before this date for Victorian RACFs.

## What changes (operational summary)

From 2026-07-01 in Victoria, **only registered nurses, enrolled nurses, pharmacists, or medical practitioners** may administer Schedule 4, 8, and 9 medications and drugs of dependence to residents who do not self-administer their own medication. PCWs (Personal Care Workers) may continue to assist competent self-administering residents only. This includes antibiotics, opioid analgesics, benzodiazepines, and clinical trial medications.

This is the spec's identified prototype for jurisdiction-aware ScopeRules. NSW, QLD, and SA branches have advocated similar restrictions; the platform's ScopeRules architecture must be data-not-code so additional jurisdictions add as data, not engineering work.

## What to download

1. **Drugs, Poisons and Controlled Substances Amendment (Medication Administration in Residential Aged Care) Act 2025** (Vic)
   - Source: https://www.legislation.vic.gov.au/ — search title
   - Why: primary statutory text; structured ScopeRule rows derive from §§ that name role × schedule × resident state
   - Maps to: ScopeRules engine (Phase 1C-δ); Authorisation state machine

2. **Drugs, Poisons and Controlled Substances Regulations** (Vic, as amended)
   - Source: https://www.legislation.vic.gov.au/
   - Why: operational definitions of "self-administering resident," authorisation thresholds, record-keeping requirements
   - Maps to: ScopeRules engine; KB-18 governance audit trail

3. **Department of Health Victoria — implementation guidance**
   - Source: https://www.health.vic.gov.au/ — search "medication administration aged care"
   - Why: regulator-facing guidance on transition planning, RN/EN coverage requirements, exemptions
   - Maps to: facility deployment runbooks (Buyer 2 RACH operator pitch material per spec Part 5)

## Code path (Playwright)

```
1. browser_navigate → https://www.legislation.vic.gov.au/
2. Search "Drugs Poisons Controlled Substances Amendment Medication Administration Residential Aged Care 2025"
3. Locate consolidated Act PDF
4. fetch()+base64 → decode to:
   DPCS-Amendment-Medication-Administration-RAC-Act-2025.pdf
5. Search base "Drugs Poisons and Controlled Substances Regulations" → fetch latest compilation
   DPCS-Regulations-compilation.pdf
6. browser_navigate → https://www.health.vic.gov.au/ implementation guidance pages
7. Download Department of Health guidance PDFs
```

## Manual fallback

1. Browse https://www.legislation.vic.gov.au/; search "DPCS Amendment 2025".
2. Download Act PDF; save as `DPCS-Amendment-Medication-Administration-RAC-Act-2025.pdf`.
3. Download DPCS Regulations compilation; save as `DPCS-Regulations-compilation.pdf`.
4. Browse https://www.health.vic.gov.au/; download guidance PDFs.
5. Save to: `regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025/`.

## After PDFs land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 2 updated to ✅ landed
- [ ] **PRIORITY:** ScopeRules authoring (Phase 1C-δ) — must complete before 2026-09-29
- [ ] Source Registry seed (Phase 1C-β)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| _populated after procurement_ | | |
```

- [ ] **Step 2: Verify**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025/PROCUREMENT.md
```

Expected: exit code 0.

---

## Task 8: Tasmanian Co-Prescribing Pilot — engagement-required placeholder

**Files:**
- Create: `regulatory/states/tas/co_prescribing_pilot_2026/PROCUREMENT.md`

- [ ] **Step 1: Write PROCUREMENT.md (engagement-required variant)**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/states/tas/co_prescribing_pilot_2026/PROCUREMENT.md`

Content:

```markdown
# Tasmanian Aged Care Pharmacist Co-Prescribing Pilot — Procurement Runbook

**Status (2026-05-04):** 🔒 engagement-required
**Authority tier:** 1 (State government — Tasmania)
**Jurisdiction scope:** TAS only
**Effective period:** development late 2025; 12-month trial through 2026 and 2027 (Australian first).
**Reproduction terms:** TBD — depends on partnership terms.
**Layer 1 v2 spec section:** §4.6
**Spec deadline:** engagement window per Vaidshala v2 Revision Mapping Part 7 closes mid-2026 (~2 months from spec date).

## Why this is engagement-required, not download-procurable

Per Layer 1 v2 §4.6 and Vaidshala v2 Revision Mapping Part 7, this pilot is structurally embedded in academic/government partnership rather than published as a standalone regulatory PDF. The pilot is in development late 2025 with $5M Tasmanian state budget. There is no canonical legal-instrument PDF equivalent to a state Act for this initiative.

The pilot needs a digital substrate to track pharmacist-GP co-prescribing per treatment plan. **The most natural Vaidshala partnership opportunity in Australia.** This runbook is the trigger for commercial action, not a download list.

## Engagement contacts (per spec Part 7)

- **Mohammed Salahudeen** — University of Tasmania School of Pharmacy
- **Gregory Peterson** — University of Tasmania School of Pharmacy
- **Curtain** — University of Tasmania School of Pharmacy (named co-author per spec)
- **Tasmanian Department of Health Pharmacy Projects team** — Duncan McKenzie named in budget announcement

## What we expect to procure once partnership is agreed

1. Pilot design document (collaborative practice agreement template)
2. Authorised medicine class scope (per pilot design)
3. Treatment plan template (GP-pharmacist co-prescribing)
4. Evaluation framework (likely PHARMA-Care aligned)

These would land here once provided. Until then this folder remains placeholder.

## Action required

Per Vaidshala v2 Revision Mapping Part 7 Move 1: Vaidshala leadership to engage UTas + Tasmanian Department of Health within 30-60 days of spec publication. Engagement window closes mid-2026.

## Status legend

- 🔒 engagement-required — procurement blocked on partnership / EOI engagement
```

- [ ] **Step 2: Verify**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/states/tas/co_prescribing_pilot_2026/PROCUREMENT.md
```

Expected: exit code 0.

---

## Task 9: NMBA Designated RN Prescriber PROCUREMENT.md

**Files:**
- Create: `regulatory/professional_standards/nmba_designated_rn_prescriber/PROCUREMENT.md`

- [ ] **Step 1: Write PROCUREMENT.md**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/professional_standards/nmba_designated_rn_prescriber/PROCUREMENT.md`

Content:

```markdown
# NMBA Registration Standard — Endorsement for Scheduled Medicines (Designated Registered Nurse Prescriber) — Procurement Runbook

**Status (2026-05-04):** ⏳ pending
**Authority tier:** 2 (peak professional body — Nursing and Midwifery Board of Australia under AHPRA)
**Jurisdiction scope:** national (subject to state legislation parity)
**Effective period:** standard took effect 2025-09-30. First endorsed prescribers expected mid-2026.
**Reproduction terms:** © NMBA / AHPRA. Permitted use generally allows reference and quotation with attribution; commercial reproduction may require permission. Capture verbatim from source page.
**Layer 1 v2 spec section:** §4.5
**Spec deadline:** first cohort lands mid-2026 (~2-3 months from spec date)

## What to download

1. **NMBA Registration Standard: Endorsement for Scheduled Medicines — Designated Registered Nurse Prescriber**
   - Source: https://www.nursingmidwiferyboard.gov.au/Registration-Standards.aspx
   - Why: definitional document for credential schema (eligibility, scope, partnership requirements, mentorship)
   - Maps to: Credential ledger schema (Phase 1C-δ); Authorisation state machine; spec's "new safety primitive almost nobody is building"

2. **NMBA Guidelines for Scheduled Medicines** (companion guidance)
   - Source: same domain
   - Why: operational interpretation of scope-of-practice for designated prescribers
   - Maps to: ScopeRules data; Credential ledger constraints

3. **ANMAC Accreditation Standards for the relevant postgraduate programs** (referenced by NMBA standard)
   - Source: https://www.anmac.org.au/
   - Why: defines "NMBA-approved postgraduate qualification" prerequisite tracked in Credential ledger
   - Maps to: Credential ledger qualification verification

## What enables (reminder for downstream phase)

Per Layer 1 v2 §4.5, the Authorisation state machine must track for each potential designated RN prescriber:
- Credential: Endorsement valid_from, valid_to, evidence_url
- PrescribingAgreement: linked to authorised health practitioner, scope (medicine classes, residents covered), validity period, mentorship_status (active/complete/breached), signed_packet_url
- MentorshipStatus: complete (post-six-month) vs in-progress
- ScopeMatch: per-action verification that the proposed action falls within the prescribing agreement's scope

These structures are Phase 1C-δ; this phase only procures the source.

## Code path (Playwright)

```
1. browser_navigate → https://www.nursingmidwiferyboard.gov.au/Registration-Standards.aspx
2. Locate "Endorsement for scheduled medicines — designated registered nurse prescriber" link
3. Navigate to detail page; locate Standard PDF + companion guidance
4. fetch()+base64 → decode to:
   NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf
   NMBA-Scheduled-Medicines-Guidelines.pdf
5. browser_navigate → https://www.anmac.org.au/standards-and-review
6. Locate accreditation standards relevant to prescribing programs; fetch+decode
```

## Manual fallback

1. Browse https://www.nursingmidwiferyboard.gov.au/Registration-Standards.aspx.
2. Locate the designated RN prescriber endorsement standard; download PDF.
3. Save as `NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf`.
4. Download accompanying guidelines.
5. Browse https://www.anmac.org.au/; locate accreditation standards.
6. Save all to: `regulatory/professional_standards/nmba_designated_rn_prescriber/`.

## After PDFs land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 3 updated to ✅ landed
- [ ] Credential ledger schema authoring (Phase 1C-δ)
- [ ] Source Registry seed (Phase 1C-β)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| _populated after procurement_ | | |
```

- [ ] **Step 2: Verify**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/professional_standards/nmba_designated_rn_prescriber/PROCUREMENT.md
```

Expected: exit code 0.

---

## Task 10: APC ACOP Training Accreditation PROCUREMENT.md

**Files:**
- Create: `regulatory/professional_standards/apc_acop_training/PROCUREMENT.md`

- [ ] **Step 1: Write PROCUREMENT.md**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/professional_standards/apc_acop_training/PROCUREMENT.md`

Content:

```markdown
# APC Accreditation Standards for ACOP (Aged Care On-site Pharmacist) Training Programs — Procurement Runbook

**Status (2026-05-04):** ⏳ pending
**Authority tier:** 2 (peak professional body — Australian Pharmacy Council)
**Jurisdiction scope:** national
**Effective period:** APC-accredited ACOP training mandatory for ACOP measure participation from 2026-07-01.
**Reproduction terms:** © APC. Reference and quotation permitted with attribution. Commercial reproduction may require permission. Verify on download.
**Layer 1 v2 spec section:** §4.7
**Spec deadline:** mandatory training requirement begins 2026-07-01

## What to download

1. **APC Accreditation Standards for ACOP Training Programs**
   - Source: https://www.pharmacycouncil.org.au/
   - Why: defines required training content + duration + assessment for ACOP credentialing
   - Maps to: Credential ledger (APC training completion: valid_from, valid_to, evidence) per spec §4.7

2. **APC accredited program list** (registry of approved providers)
   - Source: https://www.pharmacycouncil.org.au/
   - Why: operational lookup for credential verification
   - Maps to: Credential ledger evidence_url validation

3. **$350M ACOP Program operational rules** (PSA / Department of Health joint material)
   - Source: https://www.psa.org.au/ + https://www.health.gov.au/
   - Why: defines Tier 1 (community pharmacy) vs Tier 2 (facility-employed) ACOP claim scope; bed-allocation rules; daily rate (verified AUD 619.84/day per FTE Feb 2026)
   - Maps to: Credential ledger ACOP measure participation; KB-13 PHARMA-Care indicators

## Code path (Playwright)

```
1. browser_navigate → https://www.pharmacycouncil.org.au/
2. Locate ACOP training accreditation standards page
3. Download accreditation standard PDF; fetch()+base64 → decode to:
   APC-ACOP-Training-Accreditation-Standard.pdf
4. Locate accredited program list / register; fetch where available
5. browser_navigate → https://www.psa.org.au/ ACOP program pages
6. Download ACOP operational rules / measure guidelines
7. browser_navigate → https://www.health.gov.au/our-work/aged-care-on-site-pharmacist
8. Download Department of Health ACOP measure documentation
```

## Manual fallback

1. Browse https://www.pharmacycouncil.org.au/; locate ACOP accreditation page.
2. Download standard PDF; save as `APC-ACOP-Training-Accreditation-Standard.pdf`.
3. Browse https://www.psa.org.au/; locate ACOP program documentation.
4. Browse https://www.health.gov.au/; locate ACOP measure pages.
5. Save all to: `regulatory/professional_standards/apc_acop_training/`.

## After PDFs land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 5 updated to ✅ landed
- [ ] Credential ledger ACOP fields (Phase 1C-δ)
- [ ] Source Registry seed (Phase 1C-β)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| _populated after procurement_ | | |
```

- [ ] **Step 2: Verify**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/professional_standards/apc_acop_training/PROCUREMENT.md
```

Expected: exit code 0.

---

## Task 11: PHARMA-Care UniSA — engagement-required placeholder

**Files:**
- Create: `regulatory/frameworks/pharma_care_unisa/PROCUREMENT.md`

- [ ] **Step 1: Write PROCUREMENT.md (engagement-required variant)**

Path: `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/frameworks/pharma_care_unisa/PROCUREMENT.md`

Content:

```markdown
# PHARMA-Care National Quality Framework (UniSA) — Procurement Runbook

**Status (2026-05-04):** 🔒 engagement-required
**Authority tier:** 3 (academic / research — University of South Australia)
**Jurisdiction scope:** national
**Effective period:** active national pilot phase from late 2025; formally evaluating $350M ACOP program.
**Reproduction terms:** TBD — depends on EOI engagement terms.
**Layer 1 v2 spec section:** §4.8
**Spec deadline:** engagement window currently open per Vaidshala v2 Revision Mapping Part 7 Move 2.

## Why this is engagement-required

Per Layer 1 v2 §4.8 and Vaidshala v2 Revision Mapping Part 7 Move 2, the PHARMA-Care framework is in active national pilot phase. Indicator definitions may refine based on pilot findings. The framework is **UniSA-led, $1.5M MRFF-funded, 14 project partners, PSA-endorsed**. EOI is open for aged care providers and on-site pharmacists; the canonical operational framework is not a public download.

Per spec, the platform's PHARMA-Care indicator computation should be **configurable, not hardcoded** until indicator definitions stabilise. KB-13 already seeds 5 PHARMA-Care domain placeholders (`PHARMA-CARE-D1` through `D5`) with `status="PILOT_PLACEHOLDER"`.

## Engagement contacts (per spec Part 7)

- **Janet Sluggett** — UniSA, PHARMA-Care framework lead
- **Sara Javanparast** — UniSA, framework co-investigator
- **EOI address:** ALH-PHARMA-Care@unisa.edu.au

## What we expect to procure once partnership is agreed

1. PHARMA-Care five-domain framework specification
2. Indicator definitions (per domain, with measurement methodology)
3. Pilot evaluation framework + reporting templates
4. Data-sharing agreement template

These would land here once provided. Until then this folder remains placeholder.

## Action required

Per Vaidshala v2 Revision Mapping Part 7 Move 2: Vaidshala leadership to engage UniSA via published EOI within 30-60 days of spec publication. The published EOI is currently open.

**Strategic value (per spec):** "Position Vaidshala as the implementation partner that produces PHARMA-Care indicators automatically." Produces three things at once: (a) framework alignment for product design, (b) published academic evaluation of the platform's effectiveness, (c) endorsement-quality positioning with regulators and buyers.

## Status legend

- 🔒 engagement-required — procurement blocked on partnership / EOI engagement
```

- [ ] **Step 2: Verify**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/frameworks/pharma_care_unisa/PROCUREMENT.md
```

Expected: exit code 0.

---

## Task 12: Verify all scaffolding before procurement

**Files:** none (verification only)

- [ ] **Step 1: Confirm all 8 PROCUREMENT.md files exist**

```bash
find /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory -name "PROCUREMENT.md" | sort
```

Expected output (8 lines, one per source):

```
.../regulatory/commonwealth/aged_care_act_2024/PROCUREMENT.md
.../regulatory/commonwealth/mhr_sharing_by_default_2025/PROCUREMENT.md
.../regulatory/commonwealth/restrictive_practice/PROCUREMENT.md
.../regulatory/frameworks/pharma_care_unisa/PROCUREMENT.md
.../regulatory/professional_standards/apc_acop_training/PROCUREMENT.md
.../regulatory/professional_standards/nmba_designated_rn_prescriber/PROCUREMENT.md
.../regulatory/states/tas/co_prescribing_pilot_2026/PROCUREMENT.md
.../regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025/PROCUREMENT.md
```

- [ ] **Step 2: Confirm MANIFEST.md and .gitignore exist**

```bash
ls -la /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/.gitignore
```

Expected: both files exist.

- [ ] **Step 3: Confirm git ignores PDFs in this tree**

```bash
cd /Volumes/Vaidshala/cardiofit && touch backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024/test.pdf && git check-ignore backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024/test.pdf && rm backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024/test.pdf
```

Expected: `git check-ignore` prints the path (proving it's ignored), exit code 0; the test.pdf is then removed.

---

## Task 13: Procure Aged Care Act 2024 + Rules + Strengthened Quality Standards

**Files:**
- Create (gitignored): `regulatory/commonwealth/aged_care_act_2024/Aged-Care-Act-2024.pdf`
- Create (gitignored): `regulatory/commonwealth/aged_care_act_2024/Aged-Care-Rules-2025-part-N.pdf` (variable count)
- Create (gitignored): `regulatory/commonwealth/aged_care_act_2024/Strengthened-Aged-Care-Quality-Standards-2025.pdf`

- [ ] **Step 1: Use Playwright to navigate to legislation.gov.au**

Tool call sequence:
```
mcp__playwright__browser_navigate → "https://www.legislation.gov.au/"
mcp__playwright__browser_evaluate → search form: locate input, fill "Aged Care Act 2024", submit
mcp__playwright__browser_snapshot → identify result link with "Aged Care Act 2024" title
mcp__playwright__browser_navigate → result detail page
mcp__playwright__browser_snapshot → locate PDF download href
```

- [ ] **Step 2: Fetch the Act PDF as base64**

```
mcp__playwright__browser_evaluate → 
  const url = '<resolved_pdf_url>';
  const resp = await fetch(url);
  const buf = await resp.arrayBuffer();
  return btoa(String.fromCharCode(...new Uint8Array(buf)));
```

- [ ] **Step 3: Decode and save**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024
echo "<base64_string_from_step_2>" | base64 -d > "$TARGET_DIR/Aged-Care-Act-2024.pdf"
file "$TARGET_DIR/Aged-Care-Act-2024.pdf"
```

Expected: `file` command reports `PDF document, version <X>`.

- [ ] **Step 4: Repeat steps 1-3 for Aged Care Rules 2025**

Each Rule part may be a separate PDF; save with naming `Aged-Care-Rules-2025-<part-or-section>.pdf`.

- [ ] **Step 5: Repeat steps 1-3 for Strengthened Quality Standards (agedcarequality.gov.au)**

```
mcp__playwright__browser_navigate → "https://www.agedcarequality.gov.au/providers/standards/strengthened-quality-standards"
```

Save as `Strengthened-Aged-Care-Quality-Standards-2025.pdf`.

- [ ] **Step 6: Verify all PDFs**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024
for f in "$TARGET_DIR"/*.pdf; do
  size=$(stat -f%z "$f" 2>/dev/null || stat -c%s "$f")
  head=$(head -c 5 "$f")
  echo "$f  size=$size  header=$head"
done
```

Expected: each line shows non-zero size and `header=%PDF-` (or starts with `%PDF`).

- [ ] **Step 7: Compute SHA-256 hashes**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/aged_care_act_2024
shasum -a 256 "$TARGET_DIR"/*.pdf
```

Record output for Task 19 (file-hash table population).

**Note on procurement difficulty:** If Playwright fetch fails (TLS/CDN block), document the failure mode in PROCUREMENT.md "After PDFs land" section, mark MANIFEST status as ❌ blocked, and note manual fallback as next action. Do not silently skip.

---

## Task 14: Procure Victorian DPCS Amendment Act 2025

**Files:**
- Create (gitignored): `regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025/DPCS-Amendment-Medication-Administration-RAC-Act-2025.pdf`

- [ ] **Step 1: Navigate to legislation.vic.gov.au**

```
mcp__playwright__browser_navigate → "https://www.legislation.vic.gov.au/"
mcp__playwright__browser_evaluate → search "Drugs Poisons Controlled Substances Amendment Medication Administration Residential Aged Care 2025"
mcp__playwright__browser_snapshot → identify Act result
mcp__playwright__browser_navigate → result page
mcp__playwright__browser_snapshot → locate PDF link
```

- [ ] **Step 2: Fetch + decode + save**

```
mcp__playwright__browser_evaluate → fetch+arrayBuffer+btoa pattern
```

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025
echo "<base64>" | base64 -d > "$TARGET_DIR/DPCS-Amendment-Medication-Administration-RAC-Act-2025.pdf"
file "$TARGET_DIR/DPCS-Amendment-Medication-Administration-RAC-Act-2025.pdf"
```

- [ ] **Step 3: Procure DPCS Regulations compilation**

Repeat steps 1-2 with search "Drugs Poisons and Controlled Substances Regulations" → save as `DPCS-Regulations-compilation.pdf`.

- [ ] **Step 4: Procure Department of Health Victoria implementation guidance**

```
mcp__playwright__browser_navigate → "https://www.health.vic.gov.au/" → search "medication administration aged care"
```

Save published guidance PDFs with descriptive names.

- [ ] **Step 5: Verify + hash**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/states/vic/pcw_exclusion_dpcs_amendment_2025
for f in "$TARGET_DIR"/*.pdf; do
  size=$(stat -f%z "$f" 2>/dev/null || stat -c%s "$f")
  head=$(head -c 5 "$f")
  echo "$f  size=$size  header=$head"
done
shasum -a 256 "$TARGET_DIR"/*.pdf
```

Expected: each PDF non-zero, header `%PDF`. Hashes recorded for Task 19.

---

## Task 15: Procure NMBA Designated RN Prescriber Standard

**Files:**
- Create (gitignored): `regulatory/professional_standards/nmba_designated_rn_prescriber/NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf`
- Create (gitignored): `regulatory/professional_standards/nmba_designated_rn_prescriber/NMBA-Scheduled-Medicines-Guidelines.pdf`

- [ ] **Step 1: Navigate to NMBA registration standards**

```
mcp__playwright__browser_navigate → "https://www.nursingmidwiferyboard.gov.au/Registration-Standards.aspx"
mcp__playwright__browser_snapshot → locate "Endorsement for scheduled medicines — designated registered nurse prescriber" link
mcp__playwright__browser_navigate → detail page
mcp__playwright__browser_snapshot → locate Standard PDF + companion guidance links
```

- [ ] **Step 2: Fetch + decode + save Standard**

```
mcp__playwright__browser_evaluate → fetch+arrayBuffer+btoa
```

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/professional_standards/nmba_designated_rn_prescriber
echo "<base64>" | base64 -d > "$TARGET_DIR/NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf"
file "$TARGET_DIR/NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf"
```

- [ ] **Step 3: Fetch + decode + save companion Guidelines**

Repeat for the Scheduled Medicines Guidelines → save as `NMBA-Scheduled-Medicines-Guidelines.pdf`.

- [ ] **Step 4: Procure ANMAC accreditation standards**

```
mcp__playwright__browser_navigate → "https://www.anmac.org.au/standards-and-review"
```

Locate accreditation standards relevant to prescribing programs; fetch+decode; save with descriptive names.

- [ ] **Step 5: Verify + hash**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/professional_standards/nmba_designated_rn_prescriber
for f in "$TARGET_DIR"/*.pdf; do
  size=$(stat -f%z "$f" 2>/dev/null || stat -c%s "$f")
  head=$(head -c 5 "$f")
  echo "$f  size=$size  header=$head"
done
shasum -a 256 "$TARGET_DIR"/*.pdf
```

Expected: PDFs non-zero, headers `%PDF`. Hashes recorded for Task 19.

---

## Task 16: Procure APC ACOP Training Accreditation Standard

**Files:**
- Create (gitignored): `regulatory/professional_standards/apc_acop_training/APC-ACOP-Training-Accreditation-Standard.pdf` (+ supporting docs)

- [ ] **Step 1: Navigate to APC accreditation pages**

```
mcp__playwright__browser_navigate → "https://www.pharmacycouncil.org.au/"
mcp__playwright__browser_snapshot → locate ACOP / on-site pharmacist accreditation link
mcp__playwright__browser_navigate → detail page
```

- [ ] **Step 2: Fetch + decode + save Standard**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/professional_standards/apc_acop_training
echo "<base64>" | base64 -d > "$TARGET_DIR/APC-ACOP-Training-Accreditation-Standard.pdf"
file "$TARGET_DIR/APC-ACOP-Training-Accreditation-Standard.pdf"
```

- [ ] **Step 3: Procure PSA ACOP program documentation**

```
mcp__playwright__browser_navigate → "https://www.psa.org.au/"
```

Locate ACOP program pages; fetch operational rules, measure guidelines; save with descriptive names.

- [ ] **Step 4: Procure Department of Health ACOP measure documentation**

```
mcp__playwright__browser_navigate → "https://www.health.gov.au/our-work/aged-care-on-site-pharmacist"
```

Fetch + decode + save.

- [ ] **Step 5: Verify + hash**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/professional_standards/apc_acop_training
for f in "$TARGET_DIR"/*.pdf; do
  size=$(stat -f%z "$f" 2>/dev/null || stat -c%s "$f")
  head=$(head -c 5 "$f")
  echo "$f  size=$size  header=$head"
done
shasum -a 256 "$TARGET_DIR"/*.pdf
```

Expected: PDFs non-zero, `%PDF` headers, hashes recorded for Task 19.

---

## Task 17: Procure Restrictive Practice Regulations

**Files:**
- Create (gitignored): `regulatory/commonwealth/restrictive_practice/Aged-Care-Royal-Commission-Response-Act-2022.pdf`
- Create (gitignored): `regulatory/commonwealth/restrictive_practice/Quality-of-Care-Principles-2014-compilation.pdf`
- Create (gitignored): `regulatory/commonwealth/restrictive_practice/ACQSC-Restrictive-Practices-Guidance.pdf`

- [ ] **Step 1: Navigate to legislation.gov.au for Royal Commission Response Act**

```
mcp__playwright__browser_navigate → "https://www.legislation.gov.au/"
mcp__playwright__browser_evaluate → search "Aged Care and Other Legislation Amendment Royal Commission Response Act 2022"
mcp__playwright__browser_snapshot → identify Act result + locate PDF link
```

- [ ] **Step 2: Fetch + decode + save**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/restrictive_practice
echo "<base64>" | base64 -d > "$TARGET_DIR/Aged-Care-Royal-Commission-Response-Act-2022.pdf"
file "$TARGET_DIR/Aged-Care-Royal-Commission-Response-Act-2022.pdf"
```

- [ ] **Step 3: Procure Quality of Care Principles 2014 compilation**

Repeat search + fetch for "Quality of Care Principles 2014" current compilation. Save as `Quality-of-Care-Principles-2014-compilation.pdf`.

- [ ] **Step 4: Procure ACQSC restrictive practices guidance**

```
mcp__playwright__browser_navigate → "https://www.agedcarequality.gov.au/providers/standards/restrictive-practices"
```

Locate published guidance PDFs; fetch+decode; save with descriptive names.

- [ ] **Step 5: Verify + hash**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/restrictive_practice
for f in "$TARGET_DIR"/*.pdf; do
  size=$(stat -f%z "$f" 2>/dev/null || stat -c%s "$f")
  head=$(head -c 5 "$f")
  echo "$f  size=$size  header=$head"
done
shasum -a 256 "$TARGET_DIR"/*.pdf
```

Expected: PDFs non-zero, `%PDF` headers, hashes recorded for Task 19.

---

## Task 18: Procure Modernising MHR (Sharing by Default) Act 2025

**Files:**
- Create (gitignored): `regulatory/commonwealth/mhr_sharing_by_default_2025/MHR-Sharing-by-Default-Act-2025.pdf`

- [ ] **Step 1: Navigate to legislation.gov.au**

```
mcp__playwright__browser_navigate → "https://www.legislation.gov.au/"
mcp__playwright__browser_evaluate → search "Modernising My Health Record Sharing by Default Act 2025"
mcp__playwright__browser_snapshot → identify Act result + locate PDF link
```

- [ ] **Step 2: Fetch + decode + save**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/mhr_sharing_by_default_2025
echo "<base64>" | base64 -d > "$TARGET_DIR/MHR-Sharing-by-Default-Act-2025.pdf"
file "$TARGET_DIR/MHR-Sharing-by-Default-Act-2025.pdf"
```

- [ ] **Step 3: Procure subordinate Sharing by Default Rules (if available)**

Search legislation.gov.au for "Sharing by Default Rules"; fetch+decode if found. Save as `MHR-Sharing-by-Default-Rules.pdf`.

- [ ] **Step 4: Procure Department of Health implementation guidance**

```
mcp__playwright__browser_navigate → "https://www.health.gov.au/" → search "Better and Faster Access to health information"
```

Fetch+decode published guidance PDFs.

- [ ] **Step 5: Verify + hash**

```bash
TARGET_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/commonwealth/mhr_sharing_by_default_2025
for f in "$TARGET_DIR"/*.pdf; do
  size=$(stat -f%z "$f" 2>/dev/null || stat -c%s "$f")
  head=$(head -c 5 "$f")
  echo "$f  size=$size  header=$head"
done
shasum -a 256 "$TARGET_DIR"/*.pdf
```

Expected: PDFs non-zero, `%PDF` headers, hashes recorded for Task 19.

---

## Task 19: Populate file-hash tables in each PROCUREMENT.md

**Files:**
- Modify: each procured-source `PROCUREMENT.md` — fill in the "File hashes" table

- [ ] **Step 1: Generate combined hash report**

```bash
REG_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory
echo "=== File hashes by folder ==="
for d in $(find "$REG_DIR" -type d); do
  pdfs=$(find "$d" -maxdepth 1 -name "*.pdf" 2>/dev/null)
  if [ -n "$pdfs" ]; then
    echo "--- $d ---"
    for f in $pdfs; do
      size=$(stat -f%z "$f" 2>/dev/null || stat -c%s "$f")
      sha=$(shasum -a 256 "$f" | awk '{print $1}')
      bn=$(basename "$f")
      echo "| $bn | $sha | $size |"
    done
  fi
done
```

- [ ] **Step 2: For each procured PROCUREMENT.md, replace the placeholder hash table row**

For each of the 6 procurable folders, edit the `PROCUREMENT.md` file: replace the line:
```
| _populated after procurement_ | | |
```
with the concrete `| filename | sha256 | bytes |` rows from Step 1.

Use the Edit tool with `old_string="| _populated after procurement_ | | |"` and `new_string=` containing the actual rows.

- [ ] **Step 3: Verify all 6 procured PROCUREMENT.md files have populated hash tables**

```bash
REG_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory
grep -L "populated after procurement" \
  "$REG_DIR/commonwealth/aged_care_act_2024/PROCUREMENT.md" \
  "$REG_DIR/commonwealth/restrictive_practice/PROCUREMENT.md" \
  "$REG_DIR/commonwealth/mhr_sharing_by_default_2025/PROCUREMENT.md" \
  "$REG_DIR/states/vic/pcw_exclusion_dpcs_amendment_2025/PROCUREMENT.md" \
  "$REG_DIR/professional_standards/nmba_designated_rn_prescriber/PROCUREMENT.md" \
  "$REG_DIR/professional_standards/apc_acop_training/PROCUREMENT.md"
```

Expected: 6 file paths printed (none retain the placeholder text).

---

## Task 20: Update MANIFEST.md status rows

**Files:**
- Modify: `regulatory/MANIFEST.md`

- [ ] **Step 1: For each successfully procured source, update its row from ⏳ pending to ✅ landed**

Use the Edit tool. For each of the 6 procurable rows, replace the procurement-state column from `⏳ pending` to `✅ landed`, set the PDFs column to the actual count, and set Updated to `2026-05-04`.

Example edit (Aged Care Act row):
```
old: | 1 | Aged Care Act 2024 + Rules 2025 + Strengthened Quality Standards | 1 | National | ⏳ pending | 0 | — |
new: | 1 | Aged Care Act 2024 + Rules 2025 + Strengthened Quality Standards | 1 | National | ✅ landed | <count> | 2026-05-04 |
```

Repeat for rows 2, 3, 5, 7, 8 (Tasmanian row 4 and PHARMA-Care row 6 stay 🔒 engagement-required).

- [ ] **Step 2: Update "Last updated" date in MANIFEST header**

Edit MANIFEST.md to set `**Last updated:** 2026-05-04` (already correct from Task 3 if same day, but confirm).

- [ ] **Step 3: Verify**

```bash
grep -E "(✅ landed|🔒 engagement-required|⏳ pending)" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md | wc -l
```

Expected: 8 lines (one per source). Counts of ✅ should equal number of successfully procured sources (target 6); 🔒 = 2; ⏳ = 0 if all procurements succeeded.

---

## Task 21: Final acceptance check

**Files:** none (verification only)

- [ ] **Step 1: Confirm directory structure complete**

```bash
find /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory -type d | wc -l
```

Expected: ≥11 directories.

- [ ] **Step 2: Confirm all 8 PROCUREMENT.md files exist**

```bash
find /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory -name "PROCUREMENT.md" | wc -l
```

Expected: 8.

- [ ] **Step 3: Confirm MANIFEST + .gitignore exist**

```bash
test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md && test -f /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/.gitignore && echo "OK"
```

Expected: `OK`.

- [ ] **Step 4: Confirm PDFs landed for procurable sources**

```bash
REG_DIR=/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory
echo "Aged Care Act folder: $(ls "$REG_DIR/commonwealth/aged_care_act_2024"/*.pdf 2>/dev/null | wc -l)"
echo "Restrictive Practice: $(ls "$REG_DIR/commonwealth/restrictive_practice"/*.pdf 2>/dev/null | wc -l)"
echo "MHR Sharing-by-Default: $(ls "$REG_DIR/commonwealth/mhr_sharing_by_default_2025"/*.pdf 2>/dev/null | wc -l)"
echo "Victorian PCW: $(ls "$REG_DIR/states/vic/pcw_exclusion_dpcs_amendment_2025"/*.pdf 2>/dev/null | wc -l)"
echo "NMBA Designated RN: $(ls "$REG_DIR/professional_standards/nmba_designated_rn_prescriber"/*.pdf 2>/dev/null | wc -l)"
echo "APC ACOP Training: $(ls "$REG_DIR/professional_standards/apc_acop_training"/*.pdf 2>/dev/null | wc -l)"
echo "Tasmanian (engagement): $(ls "$REG_DIR/states/tas/co_prescribing_pilot_2026"/*.pdf 2>/dev/null | wc -l) (expected 0)"
echo "PHARMA-Care (engagement): $(ls "$REG_DIR/frameworks/pharma_care_unisa"/*.pdf 2>/dev/null | wc -l) (expected 0)"
```

Expected: 6 procurable folders with ≥1 PDF; 2 engagement-required folders with 0 PDFs.

- [ ] **Step 5: Confirm git status shows no PDFs staged**

```bash
cd /Volumes/Vaidshala/cardiofit && git status backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/ | grep -i ".pdf"
```

Expected: no output (PDFs are gitignored, not visible to git).

- [ ] **Step 6: Confirm MANIFEST status counts match**

```bash
echo "Landed: $(grep -c "✅ landed" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md) (expected 6)"
echo "Engagement: $(grep -c "🔒 engagement-required" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md) (expected 2)"
echo "Pending: $(grep -c "⏳ pending" /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/MANIFEST.md) (expected 0 if all procurements OK)"
```

Expected: 6 / 2 / 0 (or fewer landed and corresponding ❌ blocked count if any procurement failed — those failures are themselves an acceptance signal, not silent skip).

---

## Self-Review (post-write)

**Spec coverage check (against `2026-05-04-layer1c-procurement-design.md` §8 acceptance criteria):**

1. ✅ Directory structure (§4) — Task 1
2. ✅ All 8 PROCUREMENT.md per §6 template (procurable + engagement-required variants) — Tasks 4-11
3. ✅ MANIFEST.md updated per source — Tasks 3, 20
4. ✅ .gitignore present — Task 2
5. ✅ PDFs landed for 6 procurable sources — Tasks 13-18
6. ✅ SHA-256 recorded per file — Tasks 13-18 step 7, Task 19
7. ✅ Source URL recorded per file (in PROCUREMENT.md "What to download" section) — Tasks 4-11
8. ✅ No PDF commits (gitignore working) — Task 12 step 3, Task 21 step 5

Out-of-scope items per spec §8 (Source Registry rows, extraction, ScopeRules engine) intentionally not in this plan.

**Placeholder scan:** No "TBD/TODO/implement later" remain in plan body. The PROCUREMENT.md template for engagement-required sources has `Reproduction terms: TBD — depends on partnership terms` — this is correct content (the TBD is the actual answer until engagement occurs), not a plan-level placeholder.

**Type consistency:** Tasks consistently use `regulatory/<tier>/<source>/` paths and naming convention `<Body>-<Topic>-<Year>.pdf` per spec §4.1.

**Procurement failure path:** Each procurement task notes that fetch failures should be recorded in the source's PROCUREMENT.md and reflected as ❌ blocked in MANIFEST, not silently skipped. Task 21 step 6 verifies the manifest matches reality.
