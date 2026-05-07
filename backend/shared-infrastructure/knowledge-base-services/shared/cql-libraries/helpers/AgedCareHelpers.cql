library AgedCareHelpers version '1.0'

/*
 * AgedCareHelpers
 * ===============
 *
 * Layer 3 v2 helper library for Australian aged-care CQL defines.
 *
 * ICD-10 vs ICD-10-AM POLICY (Pre-Wave Task 4)
 * --------------------------------------------
 *
 * ICD-10 (international, WHO, FREE)        — usable
 * ICD-10-AM (IHACPA, license-blocked)      — NOT usable in CQL defines
 *
 * Until the ICD-10-AM licence procurement is closed (tracked under
 * the Layer 1c procurement plan), Layer 3 rules MUST encode their
 * condition logic against international ICD-10 codes only. The
 * `IcdCodeIsClinicallyEquivalent()` helper exists to make the
 * cross-walk explicit when ICD-10 is being used as a stand-in for an
 * ICD-10-AM-coded condition; while ICD-10-AM is unlicensed it returns
 * `false` so authors are nudged to use ICD-10 directly rather than
 * relying on the cross-walk.
 *
 * The `rule_specification.yaml` validator (Wave 1) rejects any rule
 * that references an ICD-10-AM code; this helper is the only hook in
 * the helper library that mentions the AM coding system.
 *
 * Policy memo: claudedocs/audits/2026-05-PreWave-icd10am-placeholder-policy.md
 *
 * Other helpers in this file are scoped to AU aged-care concerns
 * shared by multiple Tier 1 / Tier 2 rules: ACOP service-class lookup
 * (placeholder), AN-ACC class lookup (placeholder). Implementations
 * land in Wave 1; signatures are reserved here so authors can
 * forward-reference them.
 */

// -----------------------------------------------------------------
// ICD coding cross-walk
// -----------------------------------------------------------------

/*
 * IcdCodeIsClinicallyEquivalent(icd10Code, icd10AmCode)
 *
 * Returns true iff `icd10Code` and `icd10AmCode` reference the same
 * clinical concept under the Vaidshala-curated ICD-10 ↔ ICD-10-AM
 * mapping table. While ICD-10-AM is unlicensed, the helper returns
 * `false` for every input pair. Authors should write their conditions
 * against ICD-10 and remove the cross-walk call once the licence
 * lands.
 *
 * Performance contract: O(1) lookup against an in-memory map
 * (latency budget: <1ms p95).
 */
define function "IcdCodeIsClinicallyEquivalent"(icd10Code String, icd10AmCode String):
  // ICD-10-AM unlicensed under current procurement state.
  // Wave 1 implementation will swap to a real lookup once the
  // licence is in place; until then, force authors to use ICD-10.
  false

// -----------------------------------------------------------------
// AN-ACC funding class
// -----------------------------------------------------------------

/*
 * ResidentAnAccClass(residentRef)
 *
 * Returns the resident's current AN-ACC funding class (1..13) or
 * null if no class has been assessed. Implementation in Wave 1;
 * here as a forward-reference for rules that gate on funding class
 * (e.g. some Tier 2 deprescribing rules suppress for class 1
 * residents pending care-stability evaluation).
 *
 * Performance contract: <30ms p95 (kb-20 PersonRole API).
 */
define function "ResidentAnAccClass"(residentRef String):
  null  // Wave 1 implementation pending.

// -----------------------------------------------------------------
// ACOP credentialing
// -----------------------------------------------------------------

/*
 * PrescriberHasAcopCredential(prescriberRef, scope)
 *
 * Returns true iff the named prescriber has an active ACOP
 * (Authorised Care Organisation Practitioner) credential for the
 * given scope. Used by Authorisation-gating rules in Tier 1.
 * Implementation in Wave 1; reserved here.
 *
 * Performance contract: <50ms p95 (Authorisation evaluator).
 */
define function "PrescriberHasAcopCredential"(prescriberRef String, scope String):
  null  // Wave 1 implementation pending.
