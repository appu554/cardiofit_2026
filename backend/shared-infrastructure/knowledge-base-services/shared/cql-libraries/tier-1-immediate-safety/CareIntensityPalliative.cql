library CareIntensityPalliative version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace

/*
 * CareIntensityPalliative
 * =======================
 * Tier 1 care-intensity-aware deprescribing rules (Wave 2 Task 5).
 * These rules INVERT the standard palliative guard: they fire ONLY
 * when the resident's care_intensity is palliative, surfacing
 * deprescribing opportunities for non-comfort-aligned medications.
 *
 * Defines:
 *   - Statin Palliative Primary Prevention
 *   - Bisphosphonate Palliative
 *   - Antihypertensive Over-Treated Palliative
 *   - Antiplatelet Palliative
 */

context Patient

// -----------------------------------------------------------------
// STATIN_PALLIATIVE_PRIMARY_PREVENTION
// -----------------------------------------------------------------
// Statin (C10AA) active AND palliative AND no active CHD/CVD concern
// (proxy for primary-prevention indication).
define "Statin Palliative Primary Prevention":
  Meds."HasActiveAtcPrefix"(Patient.id, 'C10AA')
  and State."CurrentCareIntensity"(Patient.id) = 'palliative'
  and not State."HasActiveConcernType"(Patient.id, 'CHD_SECONDARY_PREVENTION')
  and not State."HasActiveConcernType"(Patient.id, 'CVD_SECONDARY_PREVENTION')
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'STATIN_PALLIATIVE_PRIMARY_PREVENTION',
        Interval[Now() - 30 day, Now()])) = 0

// -----------------------------------------------------------------
// BISPHOSPHONATE_PALLIATIVE
// -----------------------------------------------------------------
// Bisphosphonate (M05BA / M05BB) active AND palliative.
define "Bisphosphonate Palliative":
  (
    Meds."HasActiveAtcPrefix"(Patient.id, 'M05BA')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'M05BB')
  )
  and State."CurrentCareIntensity"(Patient.id) = 'palliative'
  and not State."HasActiveConcernType"(Patient.id, 'BISPHOSPHONATE_REVIEW_DONE')
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'BISPHOSPHONATE_PALLIATIVE',
        Interval[Now() - 30 day, Now()])) = 0

// -----------------------------------------------------------------
// ANTIHYPERTENSIVE_OVERTREATED_PALLIATIVE
// -----------------------------------------------------------------
// Antihypertensive active AND palliative AND systolic BP baseline
// trending down (<-10 over 14d) — proxy for over-treatment in
// palliative care.
define "Antihypertensive Over-Treated Palliative":
  (
    Meds."HasActiveAtcPrefix"(Patient.id, 'C02')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C03')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C07')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C08')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'C09')
  )
  and State."CurrentCareIntensity"(Patient.id) = 'palliative'
  and State."DeltaFromBaseline"(Patient.id, 'bp_systolic', 14) < -10
  and not State."HasActiveConcernType"(Patient.id, 'ANTIHTN_PALL_REVIEW_DONE')
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'ANTIHYPERTENSIVE_OVERTREATED_PALLIATIVE',
        Interval[Now() - 30 day, Now()])) = 0

// -----------------------------------------------------------------
// ANTIPLATELET_PALLIATIVE
// -----------------------------------------------------------------
// Antiplatelet (B01AC) active AND palliative AND no acute event
// (myocardial infarction / stroke / TIA) in last 90 days.
define "Antiplatelet Palliative":
  Meds."HasActiveAtcPrefix"(Patient.id, 'B01AC')
  and State."CurrentCareIntensity"(Patient.id) = 'palliative'
  and not State."HasActiveConcernType"(Patient.id, 'ACUTE_CARDIOVASCULAR_EVENT_90D')
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'ANTIPLATELET_PALLIATIVE',
        Interval[Now() - 30 day, Now()])) = 0
