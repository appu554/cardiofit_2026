# Pre-Wave Task 4 — ICD-10-AM placeholder policy

**Date:** 2026-05-06
**Plan:** [2026-05-04-layer3-rule-encoding-plan.md](../../docs/superpowers/plans/2026-05-04-layer3-rule-encoding-plan.md) — Pre-Wave Task 4
**Audit blocker source:** [Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md](Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md)
**Status:** Policy adopted; helper reserved; validator enforcement
deferred to Wave 1.

---

## Why a policy is needed

ICD-10-AM (Australian Modification) is the AU clinical-coding
authority — but the publication is licence-blocked under IHACPA, and
no procurement path is currently open (tracked in the Layer 1c
procurement plan). Until that procurement closes, Layer 3 CQL rules
**cannot** reference ICD-10-AM codes without breaking the licence
posture of the platform.

International ICD-10 (WHO, free) is available and clinically
sufficient for the Tier 1 / Tier 2 rules being authored in Waves 2-3.
A formal policy is required so individual rule authors do not
inadvertently reach for ICD-10-AM in the absence of the licence.

---

## Policy

1. **Tier 1 + Tier 2 rules MUST use international ICD-10 codes only.**
   Where an AU-specific clinical concept exists in ICD-10-AM that
   does not have a clean ICD-10 equivalent, the rule is deferred
   until either:
   * the ICD-10-AM licence procurement closes, or
   * a clinically-equivalent ICD-10 mapping is signed off by
     Vaidshala clinical informatics.

2. **One helper, one cross-walk.** The
   `IcdCodeIsClinicallyEquivalent(icd10Code, icd10AmCode)` helper in
   `shared/cql-libraries/helpers/AgedCareHelpers.cql` is the **only**
   place in the helper library that mentions the AM coding system.
   Authors who need a cross-walk go through this helper; until the
   licence lands, the helper returns `false` for every input,
   forcing the author back to direct ICD-10 use.

3. **Validator enforcement (Wave 1).** The Wave 1
   `rule_specification_validator.py` will reject any rule_spec
   that:
   * names an ICD-10-AM code in `condition_codes` or `triggers`
   * imports a CQL define that itself references an ICD-10-AM
     ValueSet
   The validator emits a fix hint pointing at this policy memo.

4. **No silent fallback.** A rule that requires ICD-10-AM
   semantics MUST explicitly request a cross-walk via the helper,
   not silently coerce. This keeps the audit trail clean and makes
   the licence-gap exposure visible.

5. **Procurement watch.** When ICD-10-AM licence procurement closes,
   the helper implementation switches from "always false" to a real
   lookup (Vaidshala-curated mapping table). The validator
   simultaneously relaxes the ICD-10-AM rejection; affected rules
   are re-reviewed.

---

## Files

* [`shared/cql-libraries/helpers/AgedCareHelpers.cql`](../../backend/shared-infrastructure/knowledge-base-services/shared/cql-libraries/helpers/AgedCareHelpers.cql) — header documents the policy in-line; reserves `IcdCodeIsClinicallyEquivalent`, `ResidentAnAccClass`, `PrescriberHasAcopCredential` signatures.
* This memo.

---

## Acceptance evidence

* `AgedCareHelpers.cql` exists with the policy documented in the
  file header.
* `IcdCodeIsClinicallyEquivalent` is reserved with the
  "always false" placeholder body.
* This memo records the policy and the upstream/downstream
  dependencies (Layer 1c procurement → Wave 1 validator).
