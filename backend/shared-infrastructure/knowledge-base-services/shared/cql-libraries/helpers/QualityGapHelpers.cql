library QualityGapHelpers version '1.0'

using FHIR version '4.0.1'

include Vaidshala.Substrate version '1.0' called Substrate

/*
 * QualityGapHelpers
 * =================
 *
 * Wave 4A Tier 3 quality-gap helper surface. These primitives back the
 * fact_types used by Tier 3 rules:
 *
 *   ActiveMedicineUseCount         → clinical.polypharmacy_count
 *   AcbScoreFor                    → clinical.acb_score
 *   HasDocumentedNonPharmTrial     → clinical.active_concerns
 *                                    (concern-state shorthand for the
 *                                    BPSD-first-line non-pharm marker)
 *   HasRecentEventOfType           → clinical.events
 *   EventCountSince                → clinical.events
 *   HasCompletedReconciliationFor  → clinical.discharge_reconciliation_status
 *   AkpsDeltaSince                 → clinical.baseline_state.akps
 *   AnaccReassessmentSince         → clinical.events
 *
 * Latency budget per helper: <80ms p95. Bodies fan out to the Vaidshala
 * substrate adapter; until the HAPI engine wiring lands these are
 * declarative skeletons (same convention as MedicationHelpers).
 *
 * TODO(wave-1-runtime): execute under HAPI Test Engine.
 *
 * Spec: docs/superpowers/plans/2026-05-04-layer3-rule-encoding-plan.md
 *       (Wave 4A — Tier 3 quality-gap rules).
 */

// -----------------------------------------------------------------
// ActiveMedicineUseCount
// -----------------------------------------------------------------
// Count of distinct active MedicineUses for the resident (raw
// polypharmacy count). Latency: <50ms p95.
define function "ActiveMedicineUseCount"(residentRef String):
  Count(
    Substrate.ListMedicineUsesByResident(residentRef) M
      where M.status = 'active'
  )

// -----------------------------------------------------------------
// AcbScoreFor
// -----------------------------------------------------------------
// Per-resident anticholinergic burden score (sum of per-medicine ACB
// scores from kb-4 ACB rules). Returns 0 when no anticholinergics
// active. Distinct from MedicationHelpers.AnticholinergicBurdenScore in
// that this helper is the canonical Tier-3-fact-type binding.
// Latency: <80ms p95.
define function "AcbScoreFor"(residentRef String):
  Sum(
    Substrate.ListMedicineUsesByResident(residentRef) M
      where M.status = 'active'
      return Substrate.AcbScoreFor(Substrate.MedicineUseRxcui(M))
  )

// -----------------------------------------------------------------
// HasRecentEventOfType
// -----------------------------------------------------------------
// True iff the resident has at least one Event of `eventType` whose
// occurredAt is within `withinDays`. eventType is a controlled string
// from Layer 2 Event taxonomy (e.g. 'hospital_discharge',
// 'admission_to_facility', 'care_planning_meeting',
// 'anacc_reassessment'). Latency: <80ms p95.
define function "HasRecentEventOfType"(
  residentRef String, eventType String, withinDays Integer
):
  Substrate.HasEventOfTypeWithin(residentRef, eventType, withinDays)

// -----------------------------------------------------------------
// EventCountSince
// -----------------------------------------------------------------
// Count of Events of `eventType` for the resident within the last
// `withinDays`. Used by AN-ACC behavioural-evidence rule and by
// transition rules that need a numeric count. Latency: <80ms p95.
define function "EventCountSince"(
  residentRef String, eventType String, withinDays Integer
):
  Substrate.EventCountOfTypeWithin(residentRef, eventType, withinDays)

// -----------------------------------------------------------------
// HasCompletedReconciliationFor
// -----------------------------------------------------------------
// True iff a hospital_discharge event tagged `dischargeEventId` has
// a corresponding completed reconciliation_worklist record. Backs
// the discharge-reconciliation-status fact type. Latency: <80ms p95.
define function "HasCompletedReconciliationFor"(
  residentRef String, dischargeEventId String
):
  Substrate.DischargeReconciliationCompleted(residentRef, dischargeEventId)

// -----------------------------------------------------------------
// LatestHospitalDischargeOlderThan
// -----------------------------------------------------------------
// True iff the most recent hospital_discharge Event for the resident
// occurred more than `hours` ago AND no other hospital_discharge
// event has occurred since. Latency: <80ms p95.
define function "LatestHospitalDischargeOlderThan"(
  residentRef String, hours Integer
):
  Substrate.LatestEventOfTypeOlderThan(
    residentRef, 'hospital_discharge', hours)

// -----------------------------------------------------------------
// LatestHospitalDischargeId
// -----------------------------------------------------------------
// Returns the Event id of the most recent hospital_discharge for the
// resident, or null. Used to look up a paired reconciliation worklist.
// Latency: <50ms p95.
define function "LatestHospitalDischargeId"(residentRef String):
  Substrate.LatestEventIdOfType(residentRef, 'hospital_discharge')

// -----------------------------------------------------------------
// AkpsDeltaSince
// -----------------------------------------------------------------
// Signed delta of the AKPS (Australian-modified Karnofsky Performance
// Scale) score over `lookbackDays`. Negative values indicate decline.
// Backs baseline_state.akps. Latency: <60ms p95.
define function "AkpsDeltaSince"(residentRef String, lookbackDays Integer):
  Substrate.FlaggedBaselineDelta(residentRef, 'akps', lookbackDays)
