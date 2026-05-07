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
 *   - START B5 Beta Blocker Post MI Reduced LVEF
 *   - START D2 Calcium Vitamin D Osteoporosis
 *   - START E1 Bone Protective Therapy Osteoporosis
 *   - START F4 Influenza Vaccine Annual Elderly
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

// -----------------------------------------------------------------
// START_B5_BETA_BLOCKER_POST_MI_REDUCED_LVEF
// -----------------------------------------------------------------
// Source: START v3 §B5 — "Beta-blocker (bisoprolol, carvedilol,
// nebivolol or sustained-release metoprolol) in patients with
// systolic heart failure and/or post-myocardial infarction with
// reduced left ventricular ejection fraction." (O'Mahony et al.
// 2023, published criterion list).
// Fires when prior MI and reduced LVEF concerns are active AND no
// beta-blocker (ATC C07A) is active AND no contraindication concern
// is open AND care_intensity is active_treatment.
define "START B5 Beta Blocker Post MI Reduced LVEF":
  State."HasActiveConcernType"(Patient.id, 'PRIOR_MI')
  and State."HasActiveConcernType"(Patient.id, 'REDUCED_LVEF')
  and not Meds."HasActiveAtcPrefix"(Patient.id, 'C07A')
  and not State."HasActiveConcernType"(Patient.id, 'BETA_BLOCKER_CONTRAINDICATED')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'START_B5_BETA_BLOCKER_POST_MI_REDUCED_LVEF', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'START_B5_BETA_BLOCKER_POST_MI_REDUCED_LVEF',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// START_D2_CALCIUM_VITAMIN_D_OSTEOPOROSIS
// -----------------------------------------------------------------
// Source: START v3 §D2 — "Calcium and vitamin D supplements in
// patients with known osteoporosis and/or previous fragility
// fracture(s) and/or BMD T-score < -2.5." (O'Mahony et al. 2023).
// Fires when an osteoporosis concern is active AND no calcium+vitD
// supplement (ATC A12AX or A11CC) is active AND care_intensity is
// active_treatment.
define "START D2 Calcium Vitamin D Osteoporosis":
  State."HasActiveConcernType"(Patient.id, 'OSTEOPOROSIS')
  and not Meds."HasActiveAtcPrefix"(Patient.id, 'A12AX')
  and not Meds."HasActiveAtcPrefix"(Patient.id, 'A11CC')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'START_D2_CALCIUM_VITAMIN_D_OSTEOPOROSIS', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'START_D2_CALCIUM_VITAMIN_D_OSTEOPOROSIS',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// START_E1_BONE_PROTECTIVE_THERAPY_OSTEOPOROSIS
// -----------------------------------------------------------------
// Source: START v3 §E1 — "Bone-protective therapy (e.g.
// bisphosphonate, denosumab, teriparatide, romosozumab) in patients
// with documented osteoporosis or with previous fragility
// fracture(s)." (O'Mahony et al. 2023).
// Fires when an osteoporosis concern is active AND no
// bisphosphonate (M05BA / M05BB) or denosumab (M05BX04) is active
// AND no contraindication concern is open AND care_intensity is
// active_treatment.
define "START E1 Bone Protective Therapy Osteoporosis":
  State."HasActiveConcernType"(Patient.id, 'OSTEOPOROSIS')
  and not Meds."HasActiveAtcPrefix"(Patient.id, 'M05BA')
  and not Meds."HasActiveAtcPrefix"(Patient.id, 'M05BB')
  and not Meds."HasActiveAtcPrefix"(Patient.id, 'M05BX')
  and not State."HasActiveConcernType"(Patient.id, 'BISPHOSPHONATE_CONTRAINDICATED')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'START_E1_BONE_PROTECTIVE_THERAPY_OSTEOPOROSIS', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'START_E1_BONE_PROTECTIVE_THERAPY_OSTEOPOROSIS',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// START_F4_INFLUENZA_VACCINE_ANNUAL_ELDERLY
// -----------------------------------------------------------------
// Source: START v3 §F4 — "Annual seasonal influenza vaccine in
// adults aged ≥65 years (and other high-risk older adults)."
// (O'Mahony et al. 2023).
// Fires when no INFLUENZA_VACCINATION_CURRENT_SEASON concern is
// closed AND no contraindication concern is open AND care_intensity
// is active_treatment.
define "START F4 Influenza Vaccine Annual Elderly":
  not State."HasActiveConcernType"(Patient.id, 'INFLUENZA_VACCINATION_CURRENT_SEASON')
  and not State."HasActiveConcernType"(Patient.id, 'INFLUENZA_VACCINE_CONTRAINDICATED')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'START_F4_INFLUENZA_VACCINE_ANNUAL_ELDERLY', 180)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'START_F4_INFLUENZA_VACCINE_ANNUAL_ELDERLY',
        Interval[Now() - 180 day, Now()])) = 0
