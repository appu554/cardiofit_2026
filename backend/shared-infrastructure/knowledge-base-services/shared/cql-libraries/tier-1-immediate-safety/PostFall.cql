library PostFall version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include MonitoringHelpers version '1.0' called Monitor
include EvidenceTraceHelpers version '1.0' called Trace

/*
 * PostFall
 * ========
 * Tier 1 falls + post-fall rules (Wave 2 Task 2).
 * Integrates with the Clinical state machine's active_concerns
 * (`post_fall_72h`) and Monitoring state machine's neuro-check plan.
 *
 * Defines:
 *   - Post Fall Reassessment Required
 *   - Post Fall Medication Review
 *   - Post Fall Head Injury Watch 72h
 *   - Falls Risk Compounding Medications
 *   - Falls Risk Orthostatic Hypotension
 */

context Patient

// -----------------------------------------------------------------
// POST_FALL_REASSESSMENT_REQUIRED
// -----------------------------------------------------------------
// Fires when a fall happened 72-96h ago AND the post_fall_72h
// active concern has expired (i.e., is no longer active). The
// active_concern_resolution trigger source fires on concern expiry.
define "Post Fall Reassessment Required":
  State."LatestObservationAt"(Patient.id, 'fall') > (Now() - 96 hour)
  and State."LatestObservationAt"(Patient.id, 'fall') < (Now() - 72 hour)
  and not State."HasActiveConcernType"(Patient.id, 'post_fall_72h')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'POST_FALL_REASSESSMENT_REQUIRED',
        Interval[Now() - 7 day, Now()])) = 0

// -----------------------------------------------------------------
// POST_FALL_MEDICATION_REVIEW
// -----------------------------------------------------------------
// Fires on a fall event in last 7d AND any active anticoagulant
// (B01A), antihypertensive (C02/C03/C07/C08/C09), or psychotropic
// (N05/N06A) medication.
define "Post Fall Medication Review":
  State."LatestObservationAt"(Patient.id, 'fall') > (Now() - 7 day)
  and (
    Meds."HasActiveAtcPrefix"(Patient.id, 'B01A')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C02')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C03')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C07')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C08')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C09')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N05')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N06A')
  )
  and not State."HasActiveConcernType"(Patient.id, 'POST_FALL_MED_REVIEW_DONE')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'POST_FALL_MEDICATION_REVIEW',
        Interval[Now() - 14 day, Now()])) = 0

// -----------------------------------------------------------------
// POST_FALL_HEAD_INJURY_WATCH_72H
// -----------------------------------------------------------------
// Fall in last 72h AND on anticoagulant (B01A) AND no neuro check
// observation logged in the last 24h.
define "Post Fall Head Injury Watch 72h":
  State."LatestObservationAt"(Patient.id, 'fall') > (Now() - 72 hour)
  and Meds."HasActiveAtcPrefix"(Patient.id, 'B01A')
  and (
    State."LatestObservationAt"(Patient.id, 'neuro_check') is null
      or State."LatestObservationAt"(Patient.id, 'neuro_check') < (Now() - 24 hour)
  )
  and not State."HasActiveConcernType"(Patient.id, 'HEAD_INJURY_CLEARED')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'POST_FALL_HEAD_INJURY_WATCH_72H',
        Interval[Now() - 24 hour, Now()])) = 0

// -----------------------------------------------------------------
// FALLS_RISK_COMPOUNDING_MEDICATIONS
// -----------------------------------------------------------------
// Resident has ≥3 fall-risk meds active simultaneously across the
// psychotropic / antihypertensive / benzodiazepine combination.
define "Falls Risk Compounding Medications":
  (
    (if Meds."HasActiveAtcPrefix"(Patient.id, 'N05A') then 1 else 0)
    + (if Meds."HasActiveAtcPrefix"(Patient.id, 'N05BA') then 1 else 0)
    + (if Meds."HasActiveAtcPrefix"(Patient.id, 'N05CD') then 1 else 0)
    + (if Meds."HasActiveAtcPrefix"(Patient.id, 'N06A') then 1 else 0)
    + (if Meds."HasActiveAtcPrefix"(Patient.id, 'C02') then 1 else 0)
    + (if Meds."HasActiveAtcPrefix"(Patient.id, 'C03') then 1 else 0)
    + (if Meds."HasActiveAtcPrefix"(Patient.id, 'C07') then 1 else 0)
    + (if Meds."HasActiveAtcPrefix"(Patient.id, 'C08') then 1 else 0)
    + (if Meds."HasActiveAtcPrefix"(Patient.id, 'C09') then 1 else 0)
  ) >= 3
  and not State."HasActiveConcernType"(Patient.id, 'FALLS_REVIEW_COMPLETED')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'FALLS_RISK_COMPOUNDING_MEDICATIONS',
        Interval[Now() - 30 day, Now()])) = 0

// -----------------------------------------------------------------
// FALLS_RISK_ORTHOSTATIC_HYPOTENSION
// -----------------------------------------------------------------
// Standing-vs-lying systolic BP delta ≤ -20 mmHg (baseline-aware
// using bp_systolic baseline) AND on antihypertensive.
define "Falls Risk Orthostatic Hypotension":
  State."DeltaFromBaseline"(Patient.id, 'bp_systolic', 1) <= -20
  and (
    Meds."HasActiveAtcPrefix"(Patient.id, 'C02')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C03')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C07')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C08')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C09')
  )
  and not State."HasActiveConcernType"(Patient.id, 'ORTHOSTATIC_REVIEW_DONE')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'FALLS_RISK_ORTHOSTATIC_HYPOTENSION',
        Interval[Now() - 14 day, Now()])) = 0
