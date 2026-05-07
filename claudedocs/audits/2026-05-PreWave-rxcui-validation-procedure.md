# Pre-Wave Task 3 — Final RxCUI gap closure (procedure)

**Date:** 2026-05-06
**Plan:** [2026-05-04-layer3-rule-encoding-plan.md](../../docs/superpowers/plans/2026-05-04-layer3-rule-encoding-plan.md) — Pre-Wave Task 3
**Audit reference:** [Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md](Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md) — "Tier 1+2 Execution Log" section
**Status:** Procedure documented. Tier 1+2 execution dated 2026-04-30
exits 0 (per audit). This runbook is the standing procedure for the
periodic re-validation that Wave 1+ ADG 2025 + Wang 2024 ingest will
trigger.

---

## What this validates

Across every KB rule pack, drug references must resolve to a live RxCUI
(or recognised SNOMED/ICD code). A "phantom" RxCUI — a reference to a
retired or unknown identifier — silently breaks Layer 3 rule firing
because the CQL `define` cannot resolve the medication ValueSet. The
validator is the gate that catches phantom references before they
land in the rule library.

The validator already exists at:
[`backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/scripts/validate_kb_codes.py`](../../backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/scripts/validate_kb_codes.py)

This Pre-Wave task does **not** modify it. The procedure below is the
contract Wave 1+ will follow whenever a new criterion set lands.

---

## Procedure

### 1 — Snapshot the kb-4 + kb-7 DBs

Confirm the validator targets are reachable:

```bash
psql "$KB7_DATABASE_URL" -c "SELECT count(*) FROM kb7_concept_rxnorm;"
psql "$KB4_DATABASE_URL" -c "SELECT criterion_set, count(*) FROM kb4_explicit_criteria GROUP BY 1;"
```

### 2 — Run the strict cross-KB validator

```bash
cd backend/shared-infrastructure/knowledge-base-services
python3 kb-7-terminology/scripts/validate_kb_codes.py --rxnav --strict
```

Flags:

| Flag | Meaning |
|---|---|
| `--rxnav` | resolve unknown codes against the live RxNav API (requires outbound HTTPS to `rxnav.nlm.nih.gov`); without this, retired RxCUIs that have been remapped will still flag as unresolved |
| `--strict` | exit 1 on any unresolved code (default behaviour is exit 0 with summary) |
| `--json` | structured output for CI capture |
| `--sample N` | print N example unresolved codes per failing check |

### 3 — Triage failures

For every unresolved RxCUI:

1. Look up the code on RxNav to determine status (active /
   remapped-to / retired-no-replacement).
2. If **remapped**, add a row to
   `claudedocs/audits/2026-04-30_retired_rxcui_remap_manifest_v2.json`
   under the criterion set's section.
3. If **retired no replacement**, the rule referencing it must be
   either retired (governance signoff) or rewritten against a
   different code.
4. Re-run the remap script:
   ```
   python3 kb-7-terminology/scripts/remap_retired_rxcuis_v2.py --apply
   ```
5. Re-run the validator. Exit 0 is the gate.

### 4 — Record the run

Append to the audit log file
`claudedocs/audits/Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md`
under "Tier 1+2 Execution Log" with:

* Run date + git SHA
* Number of mutations applied (manifest delta)
* Validator exit code
* Any rules retired / rewritten

---

## When to run

| Trigger | Required validator pass |
|---|---|
| New criterion set added (e.g. ADG 2025, Wang 2024 expansion) | Strict pass before promotion to ACTIVE |
| RxNorm monthly release lands | Strict pass within 7 days |
| KB-4 governance signs a new rule | Strict pass on the affected criterion set |
| Quarterly Layer 3 governance review | Full strict pass logged |

---

## Acceptance evidence

* This runbook documents the standing procedure.
* The validator script already exists and is unmodified.
* The 2026-04-30 strict-mode pass evidence is preserved in the v2
  audit log (Pre-Wave Task 3 inherits that pass; the next Layer 3
  wave that adds rule rows owns the next pass).

---

## Note on environment

The Pre-Wave dispatch environment does not hold RxNav credentials or
an outbound network policy permitting `rxnav.nlm.nih.gov`. The
validator was therefore not re-executed in this dispatch — the
2026-04-30 strict pass is the current evidence of record. Wave 1+
authors that touch criterion-set rows are responsible for the next
pass.
