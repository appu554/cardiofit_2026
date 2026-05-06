# Drugs, Poisons and Controlled Substances Amendment (Medication Administration in Residential Aged Care) Act 2025 — Procurement Runbook

**Status (2026-05-04):** ✅ landed (4 PDFs)  ⚠️ HIGH PRIORITY: enforcement begins 2026-09-29
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
| DPCS-Amendment-Medication-Administration-RAC-Act-2025.pdf | 0e2d291313e08405010a8c020930584d8b0a92db4a761ea007d2c8fb3fc41870 | 255231 |
| DPCS-Regulations-2017-compilation-2025-11.pdf | ba90a7968e5f5270a728256a899cadda18d34b72cccc4d7a5c244cfae20e4889 | 1537051 |
| DH-Vic-Exposure-Draft-DPCS-Amendment-Regulations-2026-02.pdf | 862020f2ebdaf6e2f8b550e019d76f81d24ed572a23adeee15d666ce1eaf13af | 267151 |
| DH-Vic-Draft-Guidance-Regulation-149Q-2026-02.pdf | 2d33cdffe70b5e94d2e0c78ee85b3d28e9141d7cd04c03a3a49ab12d6a8327cc | 324460 |

## Procurement notes (2026-05-04)

- Amendment Act 25-037aa procured from `content.legislation.vic.gov.au/sites/default/files/2025-09/25-037aa-authorised.pdf` (authorised version).
- DPCS Regulations 2017 latest compilation (SR 29/2017 amended to 2025-11-25) procured from `17-29sra020-authorised.pdf`. New regulation 149Q is the operative provision targeted by the amendment.
- DH Victoria has published the Exposure Draft of the proposed DPCS Amendment Regulations and a Draft Guidance document for regulation 149Q (both dated 2026-02). These are pre-final consultation drafts — final versions expected before 2026-07-01 commencement. Re-procure quarterly until enforcement date.
- ScopeRules authoring (Phase 1C-δ) must source the §§ identifying role × schedule × resident state (self-administering vs not) from the as-made Act + the final regulation 149Q text once published.
