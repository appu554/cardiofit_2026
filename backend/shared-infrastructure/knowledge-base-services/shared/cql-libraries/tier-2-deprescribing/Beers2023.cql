library Beers2023 version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress

/*
 * Beers2023
 * =========
 * Tier 2 deprescribing rules from the AGS Beers Criteria 2023 update
 * (J Am Geriatr Soc 2023; 71:2052-2081). Wave 3 Task 2 vertical slice
 * ships ONE rule (Table 2 first-generation antihistamine chronic use).
 * Remaining Beers 2023 criteria are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave3-Task2-rule-queue.md.
 *
 * Defines:
 *   - Beers 2023 Anticholinergic Chronic Use
 *   - Beers 2023 Sliding Scale Insulin Nursing Home
 *   - Beers 2023 Strong Opioid First Line Elderly
 *   - Beers 2023 NSAID In CKD Stage 3 Plus
 *   - Beers 2023 Benzodiazepine In Elderly
 */

context Patient

// -----------------------------------------------------------------
// BEERS_2023_ANTICHOLINERGIC_CHRONIC_USE
// -----------------------------------------------------------------
// Source: Beers 2023 Table 2 — "Anticholinergics: first-generation
// antihistamines (avoid; clearance reduced with advanced age and
// tolerance develops; risk of confusion, dry mouth, constipation,
// other anticholinergic effects/toxicity)" (AGS 2023).
//
// Fires when a first-generation antihistamine (ATC R06AA, R06AB,
// R06AD, R06AE) has been active for >30 days AND care_intensity is
// active_treatment AND no recent anticholinergic-burden review concern
// is open.
//
// Note: Beers Table 2 entries are STRONG recommendations against
// chronic use; this rule drafts a deprescribing recommendation.
define "Beers 2023 Anticholinergic Chronic Use":
  (
    Meds."HasActiveAtcPrefix"(Patient.id, 'R06AA')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'R06AB')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'R06AD')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'R06AE')
  )
  and not State."HasActiveConcernType"(Patient.id, 'ANTICHOLINERGIC_BURDEN_REVIEW_DONE')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'BEERS_2023_ANTICHOLINERGIC_CHRONIC_USE', 60)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'BEERS_2023_ANTICHOLINERGIC_CHRONIC_USE',
        Interval[Now() - 60 day, Now()])) = 0

// -----------------------------------------------------------------
// BEERS_2023_SLIDING_SCALE_INSULIN_NURSING_HOME
// -----------------------------------------------------------------
// Source: AGS Beers 2023 §K.1 — "Sliding scale insulin (short-acting
// insulins dosed according to current blood glucose level only,
// without basal/long-acting insulin) in nursing home residents
// (avoid; higher risk of hypoglycemia without improvement in
// hyperglycemia management)." (J Am Geriatr Soc 2023; 71:2052-2081).
// Fires when SLIDING_SCALE_INSULIN_REGIMEN concern is active AND
// resident is in a nursing-home setting AND care_intensity is
// active_treatment.
define "Beers 2023 Sliding Scale Insulin Nursing Home":
  Meds."HasActiveAtcPrefix"(Patient.id, 'A10AB')
  and State."HasActiveConcernType"(Patient.id, 'SLIDING_SCALE_INSULIN_REGIMEN')
  and State."HasActiveConcernType"(Patient.id, 'NURSING_HOME_RESIDENT')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'BEERS_2023_SLIDING_SCALE_INSULIN_NURSING_HOME', 60)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'BEERS_2023_SLIDING_SCALE_INSULIN_NURSING_HOME',
        Interval[Now() - 60 day, Now()])) = 0

// -----------------------------------------------------------------
// BEERS_2023_STRONG_OPIOID_FIRST_LINE_ELDERLY
// -----------------------------------------------------------------
// Source: AGS Beers 2023 §K.7 — "Avoid initiating strong opioids
// (e.g. morphine, oxycodone, hydromorphone) as first-line therapy
// for non-cancer pain in older adults; consider weak opioid trial
// or non-opioid analgesia first." (J Am Geriatr Soc 2023).
// Fires when a strong opioid (ATC N02AA) is active AND no opioid
// indication concern (cancer pain) is documented AND no prior
// weak-opioid trial is documented AND care_intensity is
// active_treatment.
//
// Note: criterion_id encodes 'OPIOID' so the Stage 1 validator's
// Schedule-8 authorisation gating check applies.
define "Beers 2023 Strong Opioid First Line Elderly":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N02AA')
  and not State."HasActiveConcernType"(Patient.id, 'CANCER_PAIN_INDICATION')
  and not State."HasActiveConcernType"(Patient.id, 'WEAK_OPIOID_TRIAL_DOCUMENTED')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'BEERS_2023_STRONG_OPIOID_FIRST_LINE_ELDERLY', 60)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'BEERS_2023_STRONG_OPIOID_FIRST_LINE_ELDERLY',
        Interval[Now() - 60 day, Now()])) = 0

// -----------------------------------------------------------------
// BEERS_2023_NSAID_IN_CKD_STAGE_3_PLUS
// -----------------------------------------------------------------
// Source: AGS Beers 2023 §H — "Avoid chronic use of non-COX-2
// selective NSAIDs (oral or parenteral) in adults with chronic
// kidney disease stage 3 or higher (eGFR <60 mL/min/1.73m^2);
// risk of acute kidney injury and progression of CKD."
// (J Am Geriatr Soc 2023).
// Fires when an NSAID (ATC M01A) is active AND a CKD_STAGE_3_PLUS
// concern is open AND care_intensity is active_treatment.
define "Beers 2023 NSAID In CKD Stage 3 Plus":
  Meds."HasActiveAtcPrefix"(Patient.id, 'M01A')
  and State."HasActiveConcernType"(Patient.id, 'CKD_STAGE_3_PLUS')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'BEERS_2023_NSAID_IN_CKD_STAGE_3_PLUS', 60)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'BEERS_2023_NSAID_IN_CKD_STAGE_3_PLUS',
        Interval[Now() - 60 day, Now()])) = 0

// -----------------------------------------------------------------
// BEERS_2023_BENZODIAZEPINE_IN_ELDERLY
// -----------------------------------------------------------------
// Source: AGS Beers 2023 §G — "Avoid benzodiazepines (short-,
// intermediate-, and long-acting) in older adults; increased
// sensitivity in older adults and slowed metabolism of long-acting
// agents; risk of cognitive impairment, delirium, falls, fractures,
// and motor vehicle crashes." (J Am Geriatr Soc 2023).
// Fires when a benzodiazepine (ATC N05BA) is active AND no
// indication concern (severe anxiety with documented review) is
// open AND care_intensity is active_treatment.
//
// Note: criterion_id encodes 'BENZODIAZEPINE' so the Stage 1
// validator's Schedule-8 authorisation gating check applies.
define "Beers 2023 Benzodiazepine In Elderly":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N05BA')
  and not State."HasActiveConcernType"(Patient.id, 'BENZODIAZEPINE_INDICATION_REVIEWED')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'BEERS_2023_BENZODIAZEPINE_IN_ELDERLY', 60)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'BEERS_2023_BENZODIAZEPINE_IN_ELDERLY',
        Interval[Now() - 60 day, Now()])) = 0
