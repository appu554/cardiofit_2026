# kb-30-authorisation-evaluator

Runtime authorisation evaluator subsystem for the Vaidshala / CardioFit
clinical platform. Implements Layer 3 v2 doc Part 4.5 (Authorisation
evaluator as a separate subsystem).

## What it does

Given a runtime authorisation query
`(jurisdiction, role, action_class, medication_class, resident, action_date)`,
the service:

1. Looks up active rules for the jurisdiction (with parent fallback —
   `AU/VIC` matches both `AU/VIC` and `AU` rules).
2. Filters by `applies_to` (role, action class, medication schedule, class).
3. Evaluates each rule's conditions via an injectable `ConditionResolver`
   that consults the Layer 2 v2 substrate (Roles, Credentials,
   PrescribingAgreements, Consent).
4. Combines results per Layer 3 v2 doc Part 5.5.4: denied >
   granted_with_conditions > granted (most-restrictive wins).
5. Caches the result with a Layer 3 v2 doc Part 4.5.3 TTL bucket
   (5min consent / 15min agreement / 1h credential / 24h static).
6. Records every evaluation on the audit trail for regulator queries.

## Layout

```
kb-30-authorisation-evaluator/
├── cmd/server/main.go           service entrypoint (port 8138)
├── internal/dsl                 AuthorisationRule schema + YAML parser
├── internal/store               Postgres + in-memory rule store
├── internal/evaluator           runtime decision engine
├── internal/cache               in-memory + Redis-stub cache
├── internal/invalidation        substrate-event-driven cache invalidation
├── internal/audit               regulator query API (4 sample queries)
├── internal/api                 REST + gRPC stub
├── migrations/001_*.sql         Postgres schema
├── tests/integration/           Sunday-night-fall walkthrough
└── examples/                    3 sample rules with real legislative refs
```

## Example rules

| File | Rule | Source |
|------|------|--------|
| `aus-vic-pcw-s4-exclusion.yaml` | PCW S4/S8/S9 administration excluded in Victorian RACFs | Drugs, Poisons and Controlled Substances Amendment (Medication Administration in Residential Aged Care) Act 2025 (Vic), commences 1 July 2026, hard enforcement 29 Sep 2026 |
| `designated-rn-prescriber.yaml` | DRNP partnership prescribing | NMBA Endorsement for Scheduled Medicines — Designated RN Prescriber, effective 30 Sep 2025 |
| `acop-credential-active.yaml` | ACOP pharmacist resident profile view | ACOP APC training requirement, mandatory 1 Jul 2026 |

## REST surface

```
GET  /health
POST /v1/authorise
GET  /v1/audit/resident/{id}                          (?from=&to=&format=)
GET  /v1/audit/credential/{id}                        (?format=)
GET  /v1/audit/jurisdiction/{juri}/medications/{s}    (?days=&format=)
GET  /v1/audit/authorisation/{id}/chain
```

`format` is one of `fhir` (default, FHIR R4 Bundle), `csv`, or `json`.

## Building & testing

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-30-authorisation-evaluator
go build ./...
go vet ./...
go test ./...
```

DB-gated tests run only when `KB30_TEST_DATABASE_URL` is set; otherwise
they skip cleanly. Redis is a stub for the MVP — production wiring is
documented inline (`TODO(layer3-v1)` markers).

## Status

V1 substrate work — not MVP. MVP can run with simple RBAC. This service
is what enables designated RN prescribers, the Tasmanian pharmacist
co-prescriber pilot, the Victorian PCW exclusion, and the audit query
API once those regimes activate in 2026-2027.
