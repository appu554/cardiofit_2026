# Aged Care Act 2024 + Aged Care Rules 2025 + Strengthened Quality Standards — Procurement Runbook

**Status (2026-05-04):** ✅ landed (5 PDFs)
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
| Aged-Care-Act-2024-asmade.pdf | 81cdbae636617f0cab0920e11d657e6c820aab617c11bb07c9d2731d62b847b7 | 1721837 |
| Aged-Care-Act-2024-compilation-2025-11-01.pdf | 8c17c69c406cacf39de8f36e076cd93aa61415d4ed0b600c227b921aa12ab548 | 1948081 |
| Aged-Care-Rules-2025.pdf | 9440eac5945bc316581353b669ea9330f336b5dea0d4cae0bec4e1ff77dd37e3 | 2266298 |
| Aged-Care-Rules-2025-Explanatory-Statement.pdf | 2e77463b8a327daa9472e3e19332fc2324039242b1a637f3cb97c98eadc04586 | 10663486 |
| Strengthened-Aged-Care-Quality-Standards-2025-08.pdf | 4cca2731f637d80cba17c08372276d7eef728997f5a972f15380ab987b286f3f | 1819187 |

## Procurement notes (2026-05-04)

- All three primary documents procured via direct PDF fetch from `legislation.gov.au` and `health.gov.au`.
- Source URLs: `C2024A00104` (as-made + 2025-11-01 compilation), `F2025L01173` (Rules + Explanatory Statement), `health.gov.au/sites/default/files/2025-08/strengthened-aged-care-quality-standards-august-2025.pdf` (Strengthened Standards).
- Aged Care Rules 2025 are 666 pages of subordinate legislation; Explanatory Statement is 729 pages.
- Both as-made and latest compilation captured for the Act to support diff-based change tracking.
