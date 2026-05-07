library ElectrolyteRenalTrajectory version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace

/*
 * ElectrolyteRenalTrajectory
 * ==========================
 * Tier 1 immediate-safety baseline-aware electrolyte/renal rules.
 * Wave 2 Task 1 of the Layer 3 Rule Encoding Plan.
 *
 * All rules use DeltaFromBaseline / VelocityFlag (NOT raw snapshot
 * threshold compares) — the snapshot-semantics gate enforces this.
 *
 * Defines:
 *   - Hyperkalemia Risk Trajectory Extended (extends anchor with NSAID gate)
 *   - Hyponatremia Risk Trajectory
 *   - eGFR Decline Velocity
 *   - AKI Post NSAID
 *   - AKI Post Contrast
 *   - Rhabdomyolysis Statin Post Fall
 */

context Patient

// -----------------------------------------------------------------
// HYPERKALEMIA_RISK_TRAJECTORY_EXTENDED
// -----------------------------------------------------------------
// EXTENDS the anchor (TierOneImmediateSafety) with an explicit
// edge case: also include patients on spironolactone (C03DA01)
// even when the C09 (RAAS) prefix isn't present.
define "Hyperkalemia Risk Trajectory Extended":
  State."DeltaFromBaseline"(Patient.id, 'potassium', 7) > 0.8
  and (
    Meds."HasActiveAtcPrefix"(Patient.id, 'C09')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C03DA')
  )
  and not State."HasActiveConcernType"(Patient.id, 'AKI_WATCH')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'HYPERKALEMIA_RISK_TRAJECTORY_EXTENDED',
        Interval[Now() - 7 day, Now()])) = 0

// -----------------------------------------------------------------
// HYPONATREMIA_RISK_TRAJECTORY
// -----------------------------------------------------------------
// Sodium baseline trending down (delta < -3 mmol/L over 7d) AND
// resident on SSRI (N06AB) or thiazide (C03A).
define "Hyponatremia Risk Trajectory":
  State."DeltaFromBaseline"(Patient.id, 'sodium', 7) < -3
  and (
    Meds."HasActiveAtcPrefix"(Patient.id, 'N06AB')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C03A')
  )
  and not State."HasActiveConcernType"(Patient.id, 'SIADH_WATCH')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'HYPONATREMIA_RISK_TRAJECTORY',
        Interval[Now() - 7 day, Now()])) = 0

// -----------------------------------------------------------------
// EGFR_DECLINE_VELOCITY
// -----------------------------------------------------------------
// eGFR velocity flag (Wave 2.2 baseline_config: ≥20% decline / 14d)
// AND on a nephrotoxic medication (NSAID M01A, ACEi/ARB C09, or
// loop diuretic C03C).
define "eGFR Decline Velocity":
  State."VelocityFlag"(Patient.id, 'eGFR')
  and (
    Meds."HasActiveAtcPrefix"(Patient.id, 'M01A')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C09')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C03C')
  )
  and not State."HasActiveConcernType"(Patient.id, 'CKD_PROGRESSION_REVIEWED')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'EGFR_DECLINE_VELOCITY',
        Interval[Now() - 14 day, Now()])) = 0

// -----------------------------------------------------------------
// AKI_POST_NSAID
// -----------------------------------------------------------------
// eGFR delta < -10 over last 14d AND active NSAID prescribed.
define "AKI Post NSAID":
  State."DeltaFromBaseline"(Patient.id, 'eGFR', 14) < -10
  and Meds."HasActiveAtcPrefix"(Patient.id, 'M01A')
  and not State."HasActiveConcernType"(Patient.id, 'AKI_WATCH')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'AKI_POST_NSAID',
        Interval[Now() - 7 day, Now()])) = 0

// -----------------------------------------------------------------
// AKI_POST_CONTRAST
// -----------------------------------------------------------------
// eGFR delta < -10 over last 14d AND a contrast_administration event
// recorded within the last 7 days.
define "AKI Post Contrast":
  State."DeltaFromBaseline"(Patient.id, 'eGFR', 14) < -10
  and State."LatestObservationAt"(Patient.id, 'contrast_administration')
        > (Now() - 7 day)
  and not State."HasActiveConcernType"(Patient.id, 'AKI_WATCH')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'AKI_POST_CONTRAST',
        Interval[Now() - 7 day, Now()])) = 0

// -----------------------------------------------------------------
// RHABDOMYOLYSIS_STATIN_POST_FALL
// -----------------------------------------------------------------
// CK baseline delta > 2x over 14d window AND on statin (C10AA) AND
// fall event in last 14 days.
define "Rhabdomyolysis Statin Post Fall":
  State."DeltaFromBaseline"(Patient.id, 'CK', 14) > 200
  and Meds."HasActiveAtcPrefix"(Patient.id, 'C10AA')
  and State."LatestObservationAt"(Patient.id, 'fall')
        > (Now() - 14 day)
  and not State."HasActiveConcernType"(Patient.id, 'RHABDO_WORKUP_OPEN')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'RHABDOMYOLYSIS_STATIN_POST_FALL',
        Interval[Now() - 14 day, Now()])) = 0
