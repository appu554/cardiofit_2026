# kb-31-scope-rules

ScopeRules-as-data engine for the Vaidshala / CardioFit clinical runtime.
Implements Layer 3 v2 doc Part 5.5 (ScopeRule data model) and Part 4.2
(CompatibilityChecker Event D).

A `ScopeRule` is a jurisdiction- and time-aware data record that constrains
runtime authorisation decisions for a role / action_class / medication
schedule. The `AuthorisationRule` schema (kb-30) is the runtime evaluator
input; the `ScopeRule` schema is its regulator-defensible source-of-truth
twin. Both share the same YAML grammar (Layer 3 v2 doc Part 4.5.2 vs Part
5.5.2) — kb-31 simply adds a `category` discriminator.

## Service shape

- **Port:** 8139 (default; override with `PORT` env var).
- **REST endpoints:**
  - `GET  /health`
  - `GET  /v1/scope-rules?jurisdiction=AU/VIC&at=2026-07-01T00:00:00Z`
  - `GET  /v1/scope-rules/{id}`
  - `POST /v1/scope-rules` — accepts wrapped or unwrapped YAML, parses,
    validates, and inserts a new version.
- **Persistence:** PostgreSQL (`migrations/001_scope_rules.sql`) +
  in-memory `MemoryStore` for unit tests / local dev.
- **DB-gated tests:** integration tests skip cleanly when
  `KB31_TEST_DATABASE_URL` is unset.

## Bundled ScopeRules (`data/`)

| Path | Status | Source |
|---|---|---|
| `data/AU/VIC/pcw-s4-exclusion-2026-07-01.yaml` | ACTIVE | Drugs, Poisons and Controlled Substances Amendment (Medication Administration in Residential Aged Care) Act 2025 (Vic) |
| `data/AU/national/drnp-prescribing-agreement.yaml` | ACTIVE | NMBA Registration Standard: Endorsement for scheduled medicines — designated registered nurse prescriber, 30 Sep 2025 |
| `data/AU/national/acop-apc-credential.yaml` | ACTIVE | ACOP APC training requirement, 1 Jul 2026 ($350M ACOP program) |
| `data/AU/TAS/pharmacist-coprescribe-pilot-2026.yaml` | DRAFT | Tasmanian pharmacist co-prescribing pilot 2026-2027 (state $5M budget). Activation gate: pilot integration confirmation pending Vaidshala v2 Move 1. |

## Local development

```bash
go build ./...
go test ./...
go run ./cmd/server   # listens on :8139
```

To run integration tests against a real Postgres:

```bash
export KB31_TEST_DATABASE_URL='postgres://kb31:kb31@localhost:5433/kb31_test?sslmode=disable'
psql "$KB31_TEST_DATABASE_URL" -f migrations/001_scope_rules.sql
go test ./tests/integration/...
```
