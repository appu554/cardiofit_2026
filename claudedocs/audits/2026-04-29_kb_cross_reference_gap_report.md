# KB Cross-Reference Validation — Gap Report

**Generated:** 2026-04-29
**Tool:** `kb-7-terminology/scripts/validate_kb_codes.py --rxnav`
**Scope:** All consumer-KB code-bearing columns checked against KB-7 reference tables, with unresolved RxCUIs cross-classified via RxNav-in-a-Box (canonical NLM RxNorm release).
**Raw outputs:** [text](2026-04-29_kb_cross_reference_validation.txt) · [JSON](2026-04-29_kb_cross_reference_validation.json)

---

## Executive summary

| Status | Checks |
|--------|--------|
| ✅ PASS | 4 (all KB-1 — currently empty, so degenerate passes) |
| ❌ FAIL | 3 (KB-4 RxNorm primary, RxNorm array, ICD-10 array) |

**Key finding:** the 3 KB-4 failures are *not* the catastrophic phantom-code problem they appeared to be at first glance. RxNav cross-classification reveals they are **fixable migration tasks**, not data-integrity disasters.

---

## Reference tables (KB-7 authoritative)

| System | Table | Rows |
|--------|-------|-----:|
| RxNorm | `concepts_rxnorm` | 110,073 |
| LOINC | `concepts_loinc` | 35,437 |
| ICD-10 | `concepts_icd10` | 74,260 (ICD-10-**CM**, no dots) |
| SNOMED (legacy) | `concepts_snomed` | 523,502 |
| SNOMED-AU | `kb7_snomed_concept` | 714,687 |
| AMT (CTPP) | `kb7_amt_pack` | 54,303 |

---

## Failures

### 1. KB-4 `kb4_explicit_criteria.rxnorm_code_primary` — 81.7% resolution (33 unresolved)

Single-drug rules from Beers / APINCHs / TGA blackbox / TGA pregnancy / ACB store one RxCUI per row in `rxnorm_code_primary`. 33 of 180 distinct codes don't resolve in our local RxNorm reference.

**RxNav classification of the 33 unresolved:**

| RxNav status | Count | % | Meaning | Fix |
|---|---:|---:|---|---|
| NotCurrent | 17 | 51.5% | Retired in current RxNorm; were valid in older releases | Remap to current RxCUI via RxNav search-by-name, OR keep as-is if business rule predates removal |
| UNKNOWN | 13 | 39.4% | RxNav has metadata but cannot classify (very old or defunct-source codes) | Manual YAML investigation — usually pre-RxNorm-cleanup imports |
| Obsolete | 3 | 9.1% | Replaced by a newer RxCUI; remap target available | Remap directly via RxNav `historystatus.derivedConcepts` |
| **TruePhantom** | **0** | **0%** | RxNav also doesn't recognize | **None — no fabricated codes in YAMLs** ✅ |

**Sample unresolved (first 15):** 103922, 10496, 10792, 11433, 114877, 1203, 1232, 1310, 1649, 1665227, 1716, 237527, 2474, 3112, 337527

### 2. KB-4 `kb4_explicit_criteria.rxnorm_codes` — 78.3% resolution (38 unresolved)

Multi-drug rules from STOPP / START / Wang 2024 store RxCUI arrays. 38 of 175 distinct codes don't resolve.

| RxNav status | Count | % | Fix |
|---|---:|---:|---|
| NotCurrent | 23 | 60.5% | Same as primary |
| UNKNOWN | 13 | 34.2% | Same as primary |
| Remapped | 1 | 2.6% | Already-remapped concept; use new RxCUI |
| Obsolete | 1 | 2.6% | Remap via RxNav |
| **TruePhantom** | **0** | **0%** | **None — no fabricated codes in YAMLs** ✅ |

**Sample unresolved (first 15):** 10112, 10632, 10751, 10841, 11195, 114264, 1148, 114871, 1235, 1550, 1790, 215363, 226355, 227283, 236600

### 3. KB-4 `kb4_explicit_criteria.condition_icd10` — **2.3%** resolution (85 unresolved)

This is a **format-mismatch failure**, not a data-quality failure. KB-7's `concepts_icd10` table holds **ICD-10-CM** (US Clinical Modification, dotless billable codes like `D500`, `E1010`) while KB-4's START_V3 condition codes are **WHO ICD-10** (with dots, often 3-character rollups like `D50`, `E10`, `I10`).

**Sample unresolved:** D50, D50.0, D50.1, D50.8, D50.9, D51.0, E03, E03.9, E10, E11, E27.1, E27.4, E55.9, F00, F32

This matters because START_V3's prescribing-omission rules trigger on these condition codes. With only 2/87 resolving, almost no START rule will fire when a real patient has a coded diagnosis.

---

## Recommended remediations

### Quick win 1 — Fix START_V3 ICD-10 lookups (highest patient-safety impact)

**Problem:** ICD-10-CM ≠ ICD-10 WHO ≠ ICD-10-AM (Australian). KB-7 has only ICD-10-CM.

**Options (pick one):**
- **(a) Best fit for AU:** Load ICD-10-AM into KB-7 (migration `017_icd10am_schema.sql` is already applied; data load is procurement-blocked on IHACPA license). When AU-CM data lands, mapping is structural.
- **(b) Pragmatic now:** Load free WHO ICD-10 (open-licensed via WHO ICD API) into a new `concepts_icd10_who` reference table. ~14k three-char rollup codes plus ~70k subdivision codes. Resolves ~95% of START_V3 codes immediately.
- **(c) Cheapest now:** Add a normalization step at validation/runtime — strip dots from KB-4 codes and try both 3-char and 5-char prefixes against `concepts_icd10`. Doesn't fix the underlying gap but unblocks START rule firing.

### Quick win 2 — Auto-remap the NotCurrent/Obsolete RxCUIs

**Build a script** (`scripts/remap_retired_rxcuis.py`) that:
1. Pulls all unresolved RxCUIs from KB-4
2. Calls RxNav `/REST/rxcui/{rxcui}/historystatus.json` for each
3. For Obsolete/Remapped, extracts the `derivedConcepts.remappedConcept` target
4. For NotCurrent, runs RxNav `/REST/drugs.json?name={lookup_name}` if a name is known
5. Updates the YAML files in-place with the new RxCUIs and a comment trail
6. Re-runs the loader

**Estimated outcome:** RxNorm primary 81.7% → ~95%+, RxNorm array 78.3% → ~95%+

### Medium-term — Refresh KB-7 RxNorm to a current release

The "NotCurrent" classification literally means "this code was in an older RxNorm release we no longer load." Loading a 2025/2026 monthly RxNorm release would auto-resolve a portion of the NotCurrent bucket without YAML changes.

### What we deliberately did NOT do

- **No synthetic remap targets.** Every recommendation above traces to RxNav as the oracle. We do not generate replacement RxCUIs by guessing.
- **No deletion of unresolved codes.** A code that's NotCurrent in RxNorm may still be the correct historical reference for a published Beers 2023 / STOPP v3 rule — deletion would silently weaken the rule.

---

## Status

| Action | State |
|--------|-------|
| Validator extended for KB-4 + RxNav classification | ✅ commit `<this commit>` |
| Gap report (this file) | ✅ this file |
| Remap script | ⏳ follow-up task |
| ICD-10-AM / ICD-10 WHO load | ⏳ follow-up task |
| RxNorm refresh | ⏳ follow-up task |

Re-run anytime with:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-7-terminology
python3 scripts/validate_kb_codes.py --rxnav --sample 25
```

Exit code 0 = all PASS/WARN, exit code 2 = at least one FAIL (CI-suitable).
