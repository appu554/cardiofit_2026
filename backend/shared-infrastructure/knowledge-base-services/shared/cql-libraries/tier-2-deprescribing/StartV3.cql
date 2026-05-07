library StartV3 version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress

/*
 * StartV3
 * =======
 * Tier 2 PRESCRIBING_OMISSION rules from START v3 (O'Mahony et al.,
 * Eur Geriatr Med 2023). Wave 3 Task 2 vertical slice ships ONE rule
 * (START A1). Remaining ~16 START v3 criteria are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave3-Task2-rule-queue.md.
 *
 * START rules are deprescribing inverses: they fire when an indicated
 * medication is OMITTED. Output is a "consider initiation" drafted
 * recommendation (not a deprescribing recommendation).
 *
 * Defines:
 *   - START A1 Anticoagulation in Atrial Fibrillation
 */

context Patient

// -----------------------------------------------------------------
// START_A1_ANTICOAGULATION_AF
// -----------------------------------------------------------------
// Source: START v3 §A1 — "Vitamin K antagonists or direct oral
// anticoagulants in the presence of chronic atrial fibrillation"
// (O'Mahony et al. 2023, published list).
// Fires when AF diagnosis is active AND no anticoagulant (ATC B01AA,
// B01AE, B01AF) is currently active AND no anticoagulation
// contraindication concern is open AND care_intensity is
// active_treatment.
define "START A1 Anticoagulation AF":
  State."HasActiveConcernType"(Patient.id, 'ATRIAL_FIBRILLATION')
  and not Meds."HasActiveAtcPrefix"(Patient.id, 'B01AA')
  and not Meds."HasActiveAtcPrefix"(Patient.id, 'B01AE')
  and not Meds."HasActiveAtcPrefix"(Patient.id, 'B01AF')
  and not State."HasActiveConcernType"(Patient.id, 'ANTICOAGULATION_CONTRAINDICATED')
  and not State."HasActiveConcernType"(Patient.id, 'BLEEDING_RISK_HIGH')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'START_A1_ANTICOAGULATION_AF', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'START_A1_ANTICOAGULATION_AF',
        Interval[Now() - 90 day, Now()])) = 0
