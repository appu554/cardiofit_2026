# Layer 1C — Australian Aged Care Regulatory Sources Procurement Design

**Date:** 2026-05-04
**Phase:** 1C-α (procurement only — extraction, Source Registry seeding, and engine work deferred to later phases)
**Status:** Design — pending implementation
**Spec it implements:** `backend/shared-infrastructure/knowledge-base-services/kb-6-formulary/Layer1_v2_Australian_Aged_Care_Implementation_Guidelines.md` Part 4 (Category C Regulatory and Authority Sources)
**Parent strategy:** `backend/shared-infrastructure/knowledge-base-services/kb-6-formulary/Vaidshala_Final_Product_Proposal_v2_Revision_Mapping.md`

---

## 1. Context and motivation

The Layer 1 v2 specification introduces **Category C — Regulatory and Authority Sources** as an entirely new source category that did not exist in v1.0. Category C feeds the v2 substrate's Authorisation state machine, Consent state machine, and jurisdiction-aware ScopeRules engine. The spec lists ten Category C source bodies; this design covers procurement of the eight that are currently public documents.

### 1.1 Current state (verified 2026-05-04)

A repository-wide grep for the 8 Category C documents returned zero matches across 3,221 PDFs already on disk. Category A clinical guideline procurement (TGA PI corpus 3,029 PDFs, ACSQHC stewardship docs, ADG 2025 UWA, ADS-ADEA, KDIGO, Heart Foundation, RANZCP) is comprehensive; Category C is empty.

### 1.2 Deadline pressure

Two procurement-blocking deadlines are inside the next 60 days from spec date:
- **Victorian PCW exclusion** — DPCS Amendment Act 2025 commences 1 July 2026, 90-day grace period expires 29 September 2026. Operational rules must be authored against the legislation text.
- **NMBA Designated RN Prescriber endorsement** — first cohort expected mid-2026; credential schema must be ready before any actual prescribers exist.

Per Vaidshala v2 Revision Mapping Part 7, two further sources are **engagement-required**, not download-procurable, and the engagement window closes mid-2026:
- **Tasmanian co-prescribing pilot** — UTas (Salahudeen, Peterson, Curtain) + Tasmanian Department of Health
- **PHARMA-Care National Quality Framework** — UniSA (Sluggett, Javanparast)

### 1.3 Phase decomposition

Layer 1C divides into four sub-phases. This design covers only the first.

| Phase | Output | Cost | Scope |
|---|---|---|---|
| **1C-α (this design)** | PDFs on disk, runbooks, manifest | 1-2 days | Procurement only |
| 1C-β | Source Registry rows for 8 sources | ~1 day | Database seeding |
| 1C-γ | Structured rule extraction (manual for operationally critical, Pipeline 1 for reference) | 3-4 weeks | Rule authoring |
| 1C-δ | ScopeRules engine + Credential ledger + Consent state machine | 6-8 weeks | Runtime engines |

---

## 2. Scope

### 2.1 In scope (1C-α)

1. New top-level directory `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/regulatory/` with authority-tier subdirectory structure
2. Per-source `PROCUREMENT.md` runbook for each of the 8 sources
3. Top-level `regulatory/MANIFEST.md` tracking procurement status across all 8
4. `.gitignore` excluding `*.pdf` (consistent with `wave6/.gitignore`)
5. Procured PDFs on disk for the 6 download-procurable sources
6. Engagement-required placeholders for Tasmanian pilot and PHARMA-Care

### 2.2 Out of scope (deferred to later phases)

- Source Registry seed migration (Phase 1C-β)
- Structured rule extraction whether automated or manual (Phase 1C-γ)
- ScopeRules engine, Credential ledger schemas, Consent state machine (Phase 1C-δ)
- Layer 1B patient-state source integrations (eNRMC, MHR Gateway, hospital discharge, DAA, care management) — entirely separate workstream

### 2.3 Explicitly not built in this phase

The temptation in procurement work is to scope-creep into "well, while we're here, let's also seed the Source Registry / extract the obvious rules / draft the schema." Resisting this is intentional: each downstream phase has design choices (regulatory fact schema, scope-rule data model, jurisdiction taxonomy) that benefit from PDFs being on disk and readable first. Mixing procurement with extraction guarantees re-work.

---

## 3. Source inventory

The 8 Category C sources, with verified procurement classification:

| # | Source | Authority Tier | Jurisdiction | Procurement | Effective |
|---|---|---|---|---|---|
| 1 | Aged Care Act 2024 + Aged Care Rules 2025 + Strengthened Quality Standards | 1 (Cmwlth legislature) | National | ⏳ download | Act commenced 2025-11-01; Rules iterating; Standards in force |
| 2 | DPCS Amendment (Medication Administration RAC) Act 2025 — Victorian PCW exclusion | 1 (State legislature) | VIC | ⏳ download | 2026-07-01 commencement; 2026-09-29 grace expiry |
| 3 | NMBA Registration Standard — Endorsement for Scheduled Medicines (Designated RN Prescriber) | 2 (Professional body) | National | ⏳ download | 2025-09-30 |
| 4 | Tasmanian aged care pharmacist co-prescribing pilot | 1 (State govt) | TAS | 🔒 engagement-required | Trial 2026-2027 |
| 5 | APC accreditation standards for ACOP training programs | 2 (Professional body) | National | ⏳ download | Mandatory for ACOP from 2026-07-01 |
| 6 | PHARMA-Care National Quality Framework | 3 (Academic / research) | National | 🔒 engagement-required | Pilot phase active 2025-11 onward |
| 7 | Restrictive Practice — Aged Care and Other Legislation Amendment (Royal Commission Response) Act 2022 + subsequent regs | 1 (Cmwlth legislature) | National | ⏳ download | In force since 2019, amended 2022+ |
| 8 | Modernising My Health Record (Sharing by Default) Act 2025 | 1 (Cmwlth legislature) | National | ⏳ download | Royal Assent 2025-02-14; mandatory pathology upload from 2026-07-01 |

**Authority tiers** follow Layer 1 v2 §1.2:
- Tier 1 = primary regulator/legislature
- Tier 2 = peak professional body
- Tier 3 = academic / research
- Tier 4 = facility-level policy (none in this category)

---

## 4. Directory layout

New top-level directory under `kb-3-guidelines/knowledge/au/regulatory/`, organised by authority origin:

```
kb-3-guidelines/knowledge/au/regulatory/
├── MANIFEST.md
├── .gitignore                              # *.pdf
│
├── commonwealth/
│   ├── aged_care_act_2024/
│   │   ├── PROCUREMENT.md
│   │   └── *.pdf                           # Act, Rules, Strengthened Standards
│   ├── restrictive_practice/
│   │   ├── PROCUREMENT.md
│   │   └── *.pdf
│   └── mhr_sharing_by_default_2025/
│       ├── PROCUREMENT.md
│       └── *.pdf
│
├── states/
│   ├── vic/
│   │   └── pcw_exclusion_dpcs_amendment_2025/
│   │       ├── PROCUREMENT.md
│   │       └── *.pdf
│   └── tas/
│       └── co_prescribing_pilot_2026/
│           └── PROCUREMENT.md              # engagement-required placeholder
│
├── professional_standards/
│   ├── nmba_designated_rn_prescriber/
│   │   ├── PROCUREMENT.md
│   │   └── *.pdf
│   └── apc_acop_training/
│       ├── PROCUREMENT.md
│       └── *.pdf
│
└── frameworks/
    └── pharma_care_unisa/
        └── PROCUREMENT.md                  # engagement-required placeholder
```

### 4.1 Naming convention

Consistent with wave6: `<Body>-<Topic>-<Year>.pdf`

Examples:
- `Aged-Care-Act-2024.pdf`
- `Strengthened-Aged-Care-Quality-Standards-2025.pdf`
- `DPCS-Amendment-Medication-Administration-RAC-Act-2025.pdf`
- `NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf`
- `MHR-Sharing-by-Default-Act-2025.pdf`
- `APC-ACOP-Training-Accreditation-Standard.pdf`
- `Aged-Care-Quality-Restrictive-Practice-Regulations.pdf`

### 4.2 .gitignore

```
*.pdf
```

PDFs are not committed; only runbooks (`*.md`) and the MANIFEST are tracked. This matches wave6 policy and keeps repo size bounded as the regulatory corpus grows.

---

## 5. Procurement mechanism

### 5.1 Primary path — Playwright-driven browser fetch

Established pattern from wave6 ACSQHC procurement: from this dev environment, direct curl/wget against `.gov.au` and `agedcarequality.gov.au` typically returns status 000 (TLS/CDN/firewall block). Playwright's Chromium completes the TLS handshake; `browser_evaluate` with in-page `fetch()` + base64 encoding retrieves the bytes, which decode to disk locally.

Reference implementation pattern documented in `wave6/acsqhc_ams/PROCUREMENT.md` lines 1-5.

### 5.2 Fallback path — manual browser download

Each PROCUREMENT.md includes explicit manual steps so a non-Playwright operator can complete the runbook from a normal browser, save to the named directory, and verify file integrity.

### 5.3 Verification

For each downloaded PDF:
- File exists at expected path with expected name
- File size is non-zero
- File opens as a valid PDF (header check `%PDF-`)
- SHA-256 hash recorded in PROCUREMENT.md "After PDFs land" section for tamper detection

SHA recording is lightweight (one shell line per file) and unblocks future integrity checks without a heavier governance bolt-on.

---

## 6. PROCUREMENT.md template (per source)

Each leaf folder gets a runbook with the following structure. Empty fields are explicit, never silent.

```markdown
# <Source name> — Procurement Runbook

**Status (YYYY-MM-DD):** ⏳ pending | ✅ landed | 🔒 engagement-required
**Authority tier:** 1 | 2 | 3
**Jurisdiction scope:** national | VIC | TAS | …
**Effective period:** start_date → (end_date | open)
**Reproduction terms:** <verbatim from source>
**Layer 1 v2 spec section:** §4.<n>
**Spec deadline (if any):** <date>

## What to download
- Primary document URL (resolved at procurement time, recorded here for re-fetch)
- Supporting documents (regulations, explanatory memoranda, guidance notes)
- Why each matters
- Downstream KB / state machine consumption mapping

## Code path (Playwright)
[browser_navigate → browser_evaluate fetch+base64 → decode to disk]

## Manual fallback
[step-by-step browser instructions for a human operator]

## After PDFs land
- [ ] Files verified at expected paths
- [ ] SHA-256 hashes recorded below
- [ ] regulatory/MANIFEST.md row updated to ✅ landed
- [ ] Source Registry seed (deferred to Phase 1C-β)
- [ ] Pipeline 1 / manual extraction (deferred to Phase 1C-γ)

## File hashes (post-procurement)
| File | SHA-256 | Bytes |
|---|---|---|
```

### 6.1 Engagement-required variant

For the two engagement-required sources (Tasmanian pilot, PHARMA-Care framework), the runbook substitutes a different "How to procure" section listing:
- Named contacts from Vaidshala v2 Revision Mapping Part 7
- The engagement window deadline
- What documents we expect to receive once partnership is agreed
- Status: 🔒 engagement-required

This makes the deferred items visible in the manifest without pretending a curl script will fetch them.

---

## 7. Top-level MANIFEST.md

A single tracker at `regulatory/MANIFEST.md` providing at-a-glance procurement state. Form:

```markdown
# Layer 1C — Australian Aged Care Regulatory Source Manifest

**Spec:** Layer1_v2_Australian_Aged_Care_Implementation_Guidelines §4
**Last updated:** YYYY-MM-DD
**Phase:** 1C-α (procurement)

## Procurement state

| # | Source | Tier | Jurisdiction | Status | PDFs | Updated |
|---|---|---|---|---|---|---|
| 1 | Aged Care Act 2024 + Rules + Strengthened Quality Standards | 1 | National | ⏳ pending | 0 | — |
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

- [ ] 1C-α — Procurement (this design)
- [ ] 1C-β — Source Registry rows
- [ ] 1C-γ — Structured rule extraction
- [ ] 1C-δ — ScopeRules engine + Credential ledger + Consent state machine
```

---

## 8. Acceptance criteria — Definition of Done

Phase 1C-α is complete when **all** of the following hold:

1. Directory structure from §4 exists with all leaf folders created.
2. All 8 leaf folders contain a `PROCUREMENT.md` matching the §6 template (procurable variant for 6 sources, engagement-required variant for 2).
3. `regulatory/MANIFEST.md` exists and accurately reflects procurement status per source.
4. `regulatory/.gitignore` contains `*.pdf`.
5. PDFs landed for all 6 procurable sources (items 1, 2, 3, 5, 7, 8 from §3).
6. Each landed PDF has SHA-256 hash recorded in its folder's PROCUREMENT.md.
7. Each landed PDF has its source URL recorded in its folder's PROCUREMENT.md (for re-fetch and audit).
8. No commits contain `.pdf` files (gitignore working).

**Not required** to declare 1C-α done:
- Source Registry rows (1C-β)
- Any rule extraction (1C-γ)
- Any runtime engine code (1C-δ)
- Engagement-required item resolution (separate commercial workstream)

---

## 9. Risks and mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| TLS/CDN block prevents Playwright fetch from one or more .gov.au domains | Medium | Medium | Manual fallback documented in every PROCUREMENT.md; operator completes from normal browser |
| Aged Care Rules 2025 is "900+ pages, released iteratively" — what we procure now becomes stale | High | Low (for 1C-α; matters more for 1C-γ) | Record procurement date in PROCUREMENT.md; Phase 1C-γ extraction triggers Rule version check; Source Registry will track effective_period |
| Source URLs change (gov sites reorganise) | Medium | Low | Recording resolved URL at procurement time, not just navigation path; future re-fetches consult prior runbook for last-known-good URL |
| Reproduction terms restrict downstream use | Low | High (would block extraction) | Capture reproduction terms verbatim in PROCUREMENT.md; flag to legal review before any 1C-γ rule encoding traceable to the source |
| Tasmanian / PHARMA-Care engagement window closes before partnership lands | High | High (per spec Part 7) | Out of scope for this technical phase; flagged to commercial leadership in MANIFEST |
| Engagement-required placeholders are mistaken for "not yet attempted" rather than "blocked on partnership" | Medium | Low | 🔒 emoji + explicit status text + named contacts in PROCUREMENT.md |

---

## 10. Implementation outline

For the implementation plan that will follow:

1. Create directory tree per §4 (10 leaf folders)
2. Author MANIFEST.md per §7
3. Author 8 PROCUREMENT.md files (6 procurable + 2 engagement-required)
4. Author .gitignore
5. Resolve canonical PDF URLs for the 6 procurable sources via Playwright `browser_navigate` to each authority's website
6. Procure PDFs via Playwright `browser_evaluate` fetch + base64 + local decode
7. Verify each PDF (size, magic bytes, SHA-256)
8. Populate file-hashes table in each PROCUREMENT.md
9. Update MANIFEST.md status rows
10. Single git commit with the directory structure, runbooks, and manifest (no PDFs in commit)

---

## 11. Open questions for review

None at design time. The procurement phase deliberately defers all design-loaded questions (regulatory fact schema, ScopeRules data model, ScopeRules engine architecture, Consent state model) to Phases 1C-β through 1C-δ.

If new questions emerge during implementation (e.g. a source has unexpected reproduction terms; URL resolution fails for a specific document; an engagement-required source actually has a public summary PDF after all), the phase plan returns to design rather than papering over them.

---

## 12. References

- `backend/shared-infrastructure/knowledge-base-services/kb-6-formulary/Layer1_v2_Australian_Aged_Care_Implementation_Guidelines.md` Part 4
- `backend/shared-infrastructure/knowledge-base-services/kb-6-formulary/Vaidshala_Final_Product_Proposal_v2_Revision_Mapping.md` Parts 4, 7, 8
- `backend/shared-infrastructure/knowledge-base-services/kb-3-guidelines/knowledge/au/wave6/acsqhc_ams/PROCUREMENT.md` (procurement pattern reference)
- `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/migrations/004_clinical_source_registry.sql` (target schema for Phase 1C-β)
