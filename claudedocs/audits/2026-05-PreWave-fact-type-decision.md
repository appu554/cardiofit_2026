# Pre-Wave Task 1 — `fact_type` schema decision for `PRESCRIBING_OMISSION`

**Date:** 2026-05-06
**Plan:** [2026-05-04-layer3-rule-encoding-plan.md](../../docs/superpowers/plans/2026-05-04-layer3-rule-encoding-plan.md) — Pre-Wave Task 1
**Audit blocker source:** [Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md](Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md)
**Status:** RESOLVED — option (A) chosen, no row-level data migration required.

---

## Problem statement

KB-4 stores 392 explicit-criteria rules across 8 criterion sets in
`kb4_explicit_criteria`. The Layer 1 v2 audit raised a question that
must be answered before any Layer 3 CQL `define` can reference an
explicit-criteria row:

> START rules (40 rows) and STOPP rules (80 rows) live side-by-side in
> the same table, distinguished today only by `criterion_set`
> (`START_V3` vs `STOPP_V3`). When a Layer 3 CQL `define` predicate
> wants to ask "is this rule a *prescribing omission*?" (i.e. a START
> rule), it needs an unambiguous answer. Should it look at
> `criterion_set='START_V3'` (option A — keep the implicit
> discriminator), or should we add an explicit `fact_type` column with
> a `'PRESCRIBING_OMISSION'` tag (option B — denormalise)?

Either choice works mechanically. The decision matters because it sets
the convention every downstream rule_specification.yaml will follow.

---

## Options considered

### Option A — Keep `criterion_set='START_V3'` discriminator (CHOSEN)

* No row-level migration. The 40 START rows already say `START_V3` in
  `criterion_set`; no new column, no backfill, no risk of data drift.
* Add a column-level `COMMENT ON COLUMN` (migration 007) that documents
  the discriminator legend so a reviewer reading the schema is in no
  doubt about the START vs STOPP semantics.
* Layer 3 ships a CQL helper `IsPrescribingOmission(criterionSet)` that
  is a pure expression: `criterion_set = 'START_V3'`. Any future
  prescribing-omission criterion set (a hypothetical `START_AU_V1`
  for example) just extends the helper.
* Audit trail stays clean: no governance signoff required for a
  documentation-only change.

### Option B — Add `fact_type='PRESCRIBING_OMISSION'` column (REJECTED)

* Requires a 40-row UPDATE migration plus a CHECK constraint.
* Introduces the possibility of `criterion_set` and `fact_type`
  drifting out of sync (e.g. a future row tagged `START_V3` but
  `fact_type='POTENTIALLY_INAPPROPRIATE'`).
* No semantic gain — `fact_type` would simply be a function of
  `criterion_set`, i.e. denormalised information.
* Adds governance overhead (any rule-set re-tagging change becomes a
  data-migration approval rather than a schema-comment update).

---

## Decision

**Option A — keep the discriminator. Ship a `COMMENT ON COLUMN`
migration and an `IsPrescribingOmission()` CQL helper.**

### Rationale (summary)

1. **No data migration risk.** The 40 START rows are signed under L6
   governance; touching them risks invalidating the Ed25519 signing
   chain (verified separately by Pre-Wave Task 2). A documentation
   change does not.
2. **Discriminator is already authoritative.** `criterion_set` is the
   PK component (UNIQUE `(criterion_set, criterion_id)`); a START rule
   cannot exist without `criterion_set='START_V3'`.
3. **Helper-based abstraction is the right layer.** The Layer 3 v2
   rule-encoding pattern is "CQL define talks to substrate via
   helpers." Encoding "is a START rule" as a helper rather than a
   schema column matches the architectural grain.
4. **Future criterion sets compose cleanly.** A future Australian
   START-equivalent would be added to the helper's match list, not to
   a schema column that requires a new migration.

---

## Migration

[`kb-4-patient-safety/migrations/007_fact_type_resolution.sql`](../../backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety/migrations/007_fact_type_resolution.sql)

* Idempotent: uses `COMMENT ON COLUMN` which is overwrite-safe.
* Touches no row data. Touches no constraint. Adds no column.
* Documents the discriminator legend covering all 8 criterion sets
  currently present (STOPP_V3, START_V3, BEERS_2023, BEERS_RENAL,
  ACB, PIMS_WANG, AU_APINCHS, AU_TGA_BLACKBOX) and the canonical
  `IsPrescribingOmission()` predicate.

---

## CQL helper contract

Layer 3 Wave 1 will implement a helper in
`shared/cql-libraries/helpers/MedicationHelpers.cql`:

```cql
define function "IsPrescribingOmission"(criterionSet String):
  criterionSet = 'START_V3'
```

The helper will be referenced by every rule_specification.yaml that
asks the question. CQL authors do **not** read `criterion_set`
directly; they read the helper. This preserves the option to extend
the helper in future without touching individual rules.

---

## Acceptance evidence

* Decision memo (this file) lists chosen option A and rationale.
* Migration 007 ships a `COMMENT ON COLUMN` with the full
  discriminator legend (idempotent; no row data changes).
* The helper signature is reserved in the Wave 0 CQL helper surface
  spec. The implementation lands in Wave 1 alongside the rest of
  `MedicationHelpers.cql`.

---

## Affected downstream

* **Wave 0 Task 2** — CQL helper surface spec: include
  `IsPrescribingOmission()` signature.
* **Wave 1 Task 1** — Helper library implementation: implement the
  helper as specified above; assert it returns true for all 40 START
  rows and false for the 80 STOPP rows (acceptance check from the
  plan).
* **Wave 1 Task 2** — rule_specification validator: enforce that any
  rule whose semantics depend on the prescribing-omission distinction
  references the helper, not a raw `criterion_set` literal.
