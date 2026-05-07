library MedicationHelpers version '1.0'

using FHIR version '4.0.1'

include Vaidshala.Substrate version '1.0' called Substrate

/*
 * MedicationHelpers
 * =================
 *
 * Layer 3 v2 helper library for medication-state queries.
 *
 * SUBSTRATE ADAPTER NOTE
 * ----------------------
 * Bodies in this file reference `Substrate.*` external functions
 * exposed by the Vaidshala substrate adapter (Layer 2 → Layer 3
 * handoff doc, sections "MedicineUse fan-out" and "AMT/ATC join").
 * The substrate adapter is shipped as a separate CQL library with
 * the HAPI engine wiring; until that lands the bodies here are
 * declarative skeletons usable for static analysis but not runtime
 * evaluation.
 *
 * TODO(wave-1-runtime): connect under HAPI Test Engine when the
 * Substrate adapter library lands (Wave 2 dependency).
 *
 * Spec: docs/superpowers/specs/2026-05-Layer3-Wave0-cql-helper-surface.md
 *       (File 1 — MedicationHelpers).
 */

// -----------------------------------------------------------------
// IsActiveOnDate
// -----------------------------------------------------------------
// Returns true when the given MedicineUse is active on `asOf`.
// Performance contract: <30ms p95 (scoped substrate read).
define function "IsActiveOnDate"(med Tuple, asOf DateTime):
  Substrate.MedicineUseIsActive(med, asOf)

// -----------------------------------------------------------------
// HasActiveAtcPrefix
// -----------------------------------------------------------------
// True iff resident has any active MedicineUse whose ATC code begins
// with `atcPrefix`. Latency: <50ms p95.
define function "HasActiveAtcPrefix"(residentRef String, atcPrefix String):
  exists(
    Substrate.ListMedicineUsesByResident(residentRef) M
      where Substrate.AtcStartsWith(M, atcPrefix)
        and M.status = 'active'
  )

// -----------------------------------------------------------------
// GetIntentClass
// -----------------------------------------------------------------
// Returns one of: 'initiation', 'continuation', 'taper', 'prn',
// 'unknown'. Latency: <30ms p95.
define function "GetIntentClass"(med Tuple):
  Coalesce(Substrate.MedicineUseIntentCategory(med), 'unknown')

// -----------------------------------------------------------------
// GetTargetType
// -----------------------------------------------------------------
// Latency: <30ms p95.
define function "GetTargetType"(med Tuple):
  Substrate.MedicineUseTargetType(med)

// -----------------------------------------------------------------
// IsPrescribingOmission
// -----------------------------------------------------------------
// Pure helper over the criterion-set discriminator legend
// (kb-4 migration 007). Latency: <1ms.
define function "IsPrescribingOmission"(criterionSet String):
  criterionSet in { 'START_V3' }

// -----------------------------------------------------------------
// IsHighRiskFromBeers
// -----------------------------------------------------------------
// True iff the medication's RXCUI joins to the Beers high-risk
// ValueSet. Latency: <50ms p95.
define function "IsHighRiskFromBeers"(med Tuple):
  Substrate.RxcuiInValueSet(Substrate.MedicineUseRxcui(med),
                            'BEERS_HIGH_RISK_2023')

// -----------------------------------------------------------------
// ActiveDoseFor
// -----------------------------------------------------------------
// Returns the current active dose for the (resident, rxcui) pair.
// Latency: <50ms p95.
define function "ActiveDoseFor"(residentRef String, rxcui String):
  First(
    Substrate.ListMedicineUsesByResident(residentRef) M
      where Substrate.MedicineUseRxcui(M) = rxcui
        and M.status = 'active'
      return M.dose
  )

// -----------------------------------------------------------------
// DurationDaysActive
// -----------------------------------------------------------------
// Days since med start_date (inclusive). Latency: <30ms p95.
define function "DurationDaysActive"(med Tuple):
  difference in days between Substrate.MedicineUseStart(med) and Now()

// -----------------------------------------------------------------
// MorphineEquivalentMgPerDay
// -----------------------------------------------------------------
// Aggregate MEDD across all active opioid medicine_uses for a
// resident. Latency: <50ms p95.
define function "MorphineEquivalentMgPerDay"(residentRef String):
  Sum(
    Substrate.ListMedicineUsesByResident(residentRef) M
      where Substrate.AtcStartsWith(M, 'N02A') and M.status = 'active'
      return Substrate.MedicineUseMeddMg(M)
  )

// -----------------------------------------------------------------
// AnticholinergicBurdenScore
// -----------------------------------------------------------------
// KB-4 ACB rules joined to active meds. Latency: <50ms p95.
define function "AnticholinergicBurdenScore"(residentRef String):
  Sum(
    Substrate.ListMedicineUsesByResident(residentRef) M
      where M.status = 'active'
      return Substrate.AcbScoreFor(Substrate.MedicineUseRxcui(M))
  )
