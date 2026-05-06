# Layer 1C — Australian Aged Care Regulatory Source Manifest

**Spec:** `kb-6-formulary/Layer1_v2_Australian_Aged_Care_Implementation_Guidelines.md` Part 4
**Design:** `docs/superpowers/specs/2026-05-04-layer1c-procurement-design.md`
**Plan:** `docs/superpowers/plans/2026-05-04-layer1c-procurement-plan.md`
**Last updated:** 2026-05-04
**Phase:** 1C-α (procurement)

## Procurement state

| # | Source | Tier | Jurisdiction | Status | PDFs | Updated |
|---|---|---|---|---|---|---|
| 1 | Aged Care Act 2024 + Rules 2025 + Strengthened Quality Standards | 1 | National | ✅ landed | 5 | 2026-05-04 |
| 2 | DPCS Amendment Act 2025 (VIC PCW exclusion) | 1 | VIC | ✅ landed | 4 | 2026-05-04 |
| 3 | NMBA Designated RN Prescriber Standard | 2 | National | ✅ landed | 3 | 2026-05-04 |
| 4 | Tasmanian co-prescribing pilot | 1 | TAS | 🔒 engagement-required | 0 | — |
| 5 | APC ACOP training accreditation | 2 | National | ✅ landed | 6 | 2026-05-04 |
| 6 | PHARMA-Care National Quality Framework | 3 | National | 🔒 engagement-required | 0 | — |
| 7 | Restrictive Practice regulations | 1 | National | ✅ landed | 9 | 2026-05-04 |
| 8 | Modernising MHR (Sharing by Default) Act 2025 | 1 | National | ✅ landed | 2 | 2026-05-04 |

## Status legend

- ⏳ pending — runbook drafted, PDFs not yet on disk
- ✅ landed — PDFs on disk and verified
- 🔒 engagement-required — procurement blocked on partnership / EOI engagement
- ❌ blocked — procurement attempted, failed; see linked PROCUREMENT.md

## Phase progress

- [x] 1C-α — Procurement complete (2026-05-04). 29 PDFs across 6 sources; 2 engagement-required.
- [x] 1C-β — Source Registry rows authored (2026-05-04). Migration `kb-22-hpi-engine/migrations/007_au_regulatory_extension.sql` extends `clinical_sources` with v2 §1.2 fields (regulatory_category, jurisdiction, authority_tier, effective_period, reproduction_terms, procurement_path) and seeds 8 AU rows. **Migration not yet applied to any database.**
- [~] 1C-γ — Structured rule extraction partial. Same migration creates `regulatory_scope_rules` table and seeds 11 Victorian DPCS Amendment Act 2025 §36EA ScopeRules. **All rows ship `activation_status='draft'` + `requires_legal_review=TRUE`** — promotion to active requires explicit human review against statutory text. Other 7 sources not yet rule-extracted.
- [ ] 1C-δ — ScopeRules engine + Credential ledger + Consent state machine (deferred). Hard deadline: VIC PCW exclusion enforcement 2026-09-29.

## Authority tiers

Per Layer 1 v2 §1.2:
- **Tier 1** — primary regulator/legislature (Commonwealth Acts, State Acts, Aged Care Quality Commission)
- **Tier 2** — peak professional body (NMBA, APC, PSA)
- **Tier 3** — academic / research (PHARMA-Care framework)
- **Tier 4** — facility-level policy (none in this category)
