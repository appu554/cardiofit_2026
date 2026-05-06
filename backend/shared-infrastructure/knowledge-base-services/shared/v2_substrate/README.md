# v2_substrate — Vaidshala v2 Substrate Entities

Shared Go package providing types, FHIR mappers, validators, and clients
for the v2 reasoning-continuity substrate entities:

- **Resident, Person, Role, MedicineUse, Observation** — canonical storage in kb-20
- **Event, EvidenceTrace** — canonical storage in kb-22

## Phase delivery

- Phase 1B-β.1 (this milestone): Resident, Person, Role
- Phase 1B-β.2: MedicineUse, Observation
- Phase 1B-β.3: Event, EvidenceTrace

## Architecture

Each KB that needs a substrate entity imports the type from `models/` and
calls the corresponding canonical KB's gRPC/REST endpoint via `client/`.
FHIR R4 mappers (HL7 AU Base v6.0.0) live in `fhir/` and translate at
ingestion / egress boundaries — internal Vaidshala code uses the clean
internal types throughout.

## See also

- Spec: `docs/superpowers/specs/2026-05-04-1b-beta-substrate-entities-design.md`
- Plan: `docs/superpowers/plans/2026-05-04-1b-beta-substrate-entities-plan.md`
- Existing shared packages: `shared/factstore/`, `shared/governance/`, `shared/types/`
