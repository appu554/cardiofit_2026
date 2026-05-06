# Layer 1B — Australian Aged Care Integration Specs Manifest

**Spec:** `kb-6-formulary/Layer1_v2_Australian_Aged_Care_Implementation_Guidelines.md` Part 3 (Category B Patient State Sources)
**Companion:** `kb-6-formulary/Vaidshala_Final_Product_Proposal_v2_Revision_Mapping.md` Parts 4, 6
**Last updated:** 2026-05-04
**Phase:** 1B-α (integration specs procurement)

## Purpose

Layer 1B requires structured ingestion from real-time data feeds (eNRMC medication charts, MHR pathology, hospital discharges, dispensing pharmacy DAA schedules, care management observations). Before any adapter code is written, this directory procures the integration specifications those adapters will consume — FHIR profiles, conformance specs, vendor API documentation.

Procurement scope mirrors Layer 1C-α: PDFs / IG bundles to disk, runbooks per source, top-level manifest. Adapters are downstream phases (1B-β substrate entities, 1B-γ first adapter).

## Procurement state

| # | Source | Type | Procurement | Files | Updated |
|---|---|---|---|---|---|
| 1 | ADHA MHR FHIR Gateway Implementation Guide v5.0 (R4) | public IG | ✅ landed | 1 | 2026-05-04 |
| 2 | ADHA eNRMC Conformance Specifications (EP TF v3.7 + CTS v3.0.6 + Registers Apr 2026) | public conformance | ✅ landed | 4 | 2026-05-04 |
| 3 | HL7 AU Base Implementation Guide v6.0.0 (R4) | public IG | ✅ landed | 2 | 2026-05-04 |
| 4 | NCTS / ADHA value sets + ATC profiles for ingestion | public profiles | ✅ landed | 0 (cross-ref to KB-7) | 2026-05-04 |
| 5 | AU FHIR Discharge Summary profile (DS v1.7 CDA + Aged Care Transfer Summary v1.1) | public IG | ✅ landed | 2 | 2026-05-04 |
| 6 | eNRMC vendor APIs (Telstra Health MedPoint, MIMS, ResMed Software) | proprietary | 🔒 engagement-required | 0 | — |
| 7 | Dispensing pharmacy software APIs (FRED, Z Solutions, Minfos, LOTS, Aquarius) | proprietary | 🔒 engagement-required | 0 | — |
| 8 | Care management software APIs (Leecare, AutumnCare, Person Centred Software, Mirus Australia) | proprietary | 🔒 engagement-required | 0 | — |

## Status legend

- ⏳ pending — runbook drafted, specs not yet on disk
- ✅ landed — specs on disk and verified
- 🔒 engagement-required — vendor partnership / NDA required for access
- ❌ blocked — procurement attempted, failed; see linked PROCUREMENT.md

## Phase progress

- [ ] 1B-α — Integration specs procurement (this phase)
- [ ] 1B-β — Substrate entities (Resident, Person, Role, MedicineUse, Observation, Event, EvidenceTrace) per Revision Mapping MVP-1 (~3-4 weeks)
- [ ] 1B-γ — First adapter (recommended: eNRMC CSV) end-to-end against substrate (~4 weeks)
- [ ] 1B-δ — Subsequent adapters (MHR, hospital discharge, DAA, care management) — parallel workstreams once 1B-γ proves out

## Subsystem mapping (Layer 1 v2 §3)

Each Layer 1B subsystem consumes one or more sources from this manifest:

| Subsystem (spec §) | Primary spec source(s) |
|---|---|
| eNRMC integration (§3.1) | #2 ADHA conformance + #3 HL7 AU + #6 vendor APIs |
| MHR pathology / discharge documents (§3.2, §3.3) | #1 MHR FHIR Gateway IG + #5 Discharge Summary profile |
| Hospital discharge reconciliation (§3.3) | #5 Discharge Summary + #4 NCTS/ATC + ACSQHC Stewardship Framework (already in Layer 1A) |
| Dispensing pharmacy DAA timing (§3.4) | #7 vendor APIs |
| Care management observations (§3.5) | #8 vendor APIs + #4 NCTS profiles for observation coding |
| NCTS terminology (§3.6) | #4 NCTS profiles (already populated in KB-7; integration profile here is for ingestion contract) |
