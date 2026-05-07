library CareTransition version '1.0'

using FHIR version '4.0.1'

include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress
include QualityGapHelpers version '1.0' called QG
include ClinicalStateHelpers version '1.0' called State

/*
 * CareTransition
 * ==============
 * Tier 3 care-transition quality-gap rules. Wave 4A Task 2 vertical
 * slice ships 2 grounded rules. Remaining ~13 rules are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave4A-Task2-transition-rule-queue.md.
 *
 * These rules consume the Layer 2 Event taxonomy
 * ('hospital_discharge', 'admission_to_facility',
 * 'care_planning_meeting', 'anacc_reassessment') and the
 * Recommendation state machine.
 *
 * Defines:
 *   - Hospital Discharge Medications Not Reconciled 72h
 *   - RMMR Recommendation Overdue 6mo
 */

context Patient

// -----------------------------------------------------------------
// VAIDSHALA_TX_DISCHARGE_NOT_RECONCILED_72H
// -----------------------------------------------------------------
// Fires when the most recent hospital_discharge Event for the resident
// occurred >72h ago AND the paired reconciliation_worklist for that
// Event is not in a `completed` state. Care_intensity gating prevents
// firing for residents transitioned to comfort_focused / palliative.
define "Hospital Discharge Medications Not Reconciled 72h":
  QG."LatestHospitalDischargeOlderThan"(Patient.id, 72)
  and not QG."HasCompletedReconciliationFor"(
        Patient.id, QG."LatestHospitalDischargeId"(Patient.id))
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_TX_DISCHARGE_NOT_RECONCILED_72H', 14)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_TX_DISCHARGE_NOT_RECONCILED_72H',
        Interval[Now() - 14 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_TX_RMMR_OVERDUE_6MO
// -----------------------------------------------------------------
// Fires when a prior RMMR (Residential Medication Management Review)
// recommendation was deferred (snooze / requeue_later) >6 months ago
// AND no rmmr_followup Event has been recorded since. RMMR follow-up
// at 6 months is a published RACGP / Pharmaceutical Society of
// Australia recommendation.
define "RMMR Recommendation Overdue 6mo":
  Suppress."WasDeferredRecently"(
    Patient.id, 'VAIDSHALA_TX_RMMR_DEFERRED', 365)
  and not Suppress."WasDeferredRecently"(
    Patient.id, 'VAIDSHALA_TX_RMMR_DEFERRED', 180)
  and not QG."HasRecentEventOfType"(Patient.id, 'rmmr_followup', 180)
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_TX_RMMR_OVERDUE_6MO',
        Interval[Now() - 90 day, Now()])) = 0
