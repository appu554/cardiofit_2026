library StoppV3 version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress

/*
 * StoppV3
 * =======
 * Tier 2 deprescribing rules sourced from STOPP v3 (O'Mahony et al.,
 * Eur Geriatr Med 2023; published criterion list). Wave 3 Task 2
 * vertical slice shipped 1 rule (STOPP B1); Wave-extension batch
 * (2026-05) ships 4 more (STOPP D5, F2, K1, J6). Remaining ~26
 * STOPP v3 criteria are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave3-Task2-rule-queue.md.
 *
 * Defines:
 *   - STOPP B1 Aspirin Primary Prevention
 *   - STOPP D5 Long Term Benzodiazepine Hypnotic
 *   - STOPP F2 PPI Uncomplicated PUD Over 8 Weeks
 *   - STOPP K1 Anticholinergic In Delirium Or Dementia
 *   - STOPP J6 Sulfonylurea HbA1c Below 7
 */

context Patient

// -----------------------------------------------------------------
// STOPP_B1_ASPIRIN_PRIMARY_PREVENTION
// -----------------------------------------------------------------
// Source: STOPP v3 §B1 — "Aspirin in primary prevention without
// cardiovascular indication" (O'Mahony et al. 2023, published list).
// Fires when aspirin (ATC B01AC06) is active AND the resident has no
// active cardiovascular secondary-prevention concern AND care_intensity
// is active_treatment (palliative residents handled by separate
// Tier 1 antiplatelet-palliative rule).
define "STOPP B1 Aspirin Primary Prevention":
  Meds."HasActiveAtcPrefix"(Patient.id, 'B01AC06')
  and not State."HasActiveConcernType"(Patient.id, 'CHD_SECONDARY_PREVENTION')
  and not State."HasActiveConcernType"(Patient.id, 'CVD_SECONDARY_PREVENTION')
  and not State."HasActiveConcernType"(Patient.id, 'CEREBROVASCULAR_SECONDARY_PREVENTION')
  and not State."HasActiveConcernType"(Patient.id, 'PERIPHERAL_ARTERIAL_DISEASE')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'STOPP_B1_ASPIRIN_PRIMARY_PREVENTION', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'STOPP_B1_ASPIRIN_PRIMARY_PREVENTION',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// STOPP_D5_LONG_TERM_BENZODIAZEPINE_HYPNOTIC
// -----------------------------------------------------------------
// Source: STOPP v3 §D5 — "Benzodiazepines, z-drugs (zopiclone,
// zolpidem) or other hypnotics for >4 weeks in older adults
// (impaired sensorium, falls, fractures, road traffic accidents)."
// (O'Mahony et al. 2023, published criterion list).
// Fires when a benzodiazepine (ATC N05BA / N05CD) or z-drug
// (ATC N05CF) is active AND care_intensity is active_treatment AND
// no recent benzodiazepine-review concern is closed.
define "STOPP D5 Long Term Benzodiazepine Hypnotic":
  (
    Meds."HasActiveAtcPrefix"(Patient.id, 'N05BA')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N05CD')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N05CF')
  )
  and not State."HasActiveConcernType"(Patient.id, 'BENZODIAZEPINE_REVIEW_DONE')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'STOPP_D5_LONG_TERM_BENZODIAZEPINE_HYPNOTIC', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'STOPP_D5_LONG_TERM_BENZODIAZEPINE_HYPNOTIC',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// STOPP_F2_PPI_UNCOMPLICATED_PUD_OVER_8_WEEKS
// -----------------------------------------------------------------
// Source: STOPP v3 §F2 — "PPI for uncomplicated peptic ulcer
// disease or erosive peptic oesophagitis at full therapeutic dose
// for >8 weeks (dose reduction or earlier discontinuation
// indicated)." (O'Mahony et al. 2023).
// Fires when a PPI (ATC A02BC) is active AND no GORD/Barrett's
// concern is open AND no NSAID/antiplatelet GI-prophylaxis concern
// is open AND care_intensity is active_treatment.
define "STOPP F2 PPI Uncomplicated PUD Over 8 Weeks":
  Meds."HasActiveAtcPrefix"(Patient.id, 'A02BC')
  and not State."HasActiveConcernType"(Patient.id, 'GORD_COMPLICATED')
  and not State."HasActiveConcernType"(Patient.id, 'BARRETT_OESOPHAGUS')
  and not State."HasActiveConcernType"(Patient.id, 'NSAID_GI_PROPHYLAXIS_INDICATION')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'STOPP_F2_PPI_UNCOMPLICATED_PUD_OVER_8_WEEKS', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'STOPP_F2_PPI_UNCOMPLICATED_PUD_OVER_8_WEEKS',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// STOPP_K1_ANTICHOLINERGIC_IN_DELIRIUM_OR_DEMENTIA
// -----------------------------------------------------------------
// Source: STOPP v3 §K1 — "Drugs likely to cause anticholinergic
// effects (e.g. tricyclic antidepressants, first-generation
// antihistamines, antimuscarinic urinary spasmolytics) in patients
// with delirium or dementia (risk of exacerbation of cognitive
// impairment)." (O'Mahony et al. 2023).
// Fires when an active medication has a high-anticholinergic-burden
// score AND the resident has an active dementia or delirium concern
// AND care_intensity is active_treatment.
define "STOPP K1 Anticholinergic In Delirium Or Dementia":
  (
    Meds."HasActiveAtcPrefix"(Patient.id, 'R06AA')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'R06AB')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N06AA')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'G04BD')
  )
  and (
    State."HasActiveConcernType"(Patient.id, 'DEMENTIA')
      or State."HasActiveConcernType"(Patient.id, 'DELIRIUM')
  )
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'STOPP_K1_ANTICHOLINERGIC_IN_DELIRIUM_OR_DEMENTIA', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'STOPP_K1_ANTICHOLINERGIC_IN_DELIRIUM_OR_DEMENTIA',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// STOPP_J6_SULFONYLUREA_HBA1C_BELOW_7
// -----------------------------------------------------------------
// Source: STOPP v3 §J6 — "Long-term sulfonylureas (e.g. glibenclamide,
// glimepiride, glipizide) with HbA1c <7% (53 mmol/mol) in older
// adults with type 2 diabetes mellitus (risk of prolonged
// hypoglycaemia)." (O'Mahony et al. 2023).
// Fires when a sulfonylurea (ATC A10BB) is active AND a T2DM concern
// is open AND a documented HBA1C_BELOW_7 concern is open AND
// care_intensity is active_treatment.
define "STOPP J6 Sulfonylurea HbA1c Below 7":
  Meds."HasActiveAtcPrefix"(Patient.id, 'A10BB')
  and State."HasActiveConcernType"(Patient.id, 'T2DM')
  and State."HasActiveConcernType"(Patient.id, 'HBA1C_BELOW_7')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'STOPP_J6_SULFONYLUREA_HBA1C_BELOW_7', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'STOPP_J6_SULFONYLUREA_HBA1C_BELOW_7',
        Interval[Now() - 90 day, Now()])) = 0
