library PharmaCareStandard5 version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress
include QualityGapHelpers version '1.0' called QG

/*
 * PharmaCareStandard5
 * ===================
 * Tier 3 quality-gap rules that produce PHARMA-Care 5-domain indicator
 * exhaust and Standard 5 (Aged Care Quality Standard 5 — Clinical
 * Care, in force 2026) evidence categories. Wave 4A Task 1 vertical
 * slice ships 4 grounded rules. Remaining ~21 rules are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave4A-Task1-quality-gap-rule-queue.md.
 *
 * Real framework references:
 *   - PHARMA-Care 5-domain indicator framework (UniSA Sluggett-led,
 *     published; domain naming follows the published v1 framework).
 *     TODO(clinical-author): confirm exact published domain IDs once
 *     the v1 framework PDF is available in the repo.
 *   - Aged Care Quality Standard 5 — Clinical Care (Australian
 *     Government Aged Care Quality and Safety Commission, in force
 *     2026). Standard 5 evidence categories cited per criterion below.
 *
 * Defines:
 *   - PharmaCare Antipsychotic Prevalence Without BPSD
 *   - PharmaCare Polypharmacy Trigger 10plus
 *   - PharmaCare Anticholinergic Burden ACB Above 3
 *   - PharmaCare BPSD First Line NonPharm
 */

context Patient

// -----------------------------------------------------------------
// VAIDSHALA_PC_D2_ANTIPSYCHOTIC_PREVALENCE
// -----------------------------------------------------------------
// PHARMA-Care domain D2 (Restraint / antipsychotic prescribing),
// Standard 5 evidence — clinical-care safety. Fires when an
// antipsychotic (ATC N05A) is active AND there is no active BPSD
// diagnosis OR no documented BPSD review concern closed within 12
// weeks. Differs from WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU in that
// this rule surfaces a quality-indicator gap regardless of dementia
// diagnosis (covers off-label antipsychotic prescribing).
//
// TODO(clinical-author): confirm PHARMA-Care domain-D2 published label.
define "PharmaCare Antipsychotic Prevalence Without BPSD":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
  and not State."HasActiveConcernType"(Patient.id, 'BPSD_DIAGNOSIS')
  and not State."HasActiveConcernType"(Patient.id, 'BPSD_REVIEW_DONE_12W')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_PC_D2_ANTIPSYCHOTIC_PREVALENCE', 84)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_PC_D2_ANTIPSYCHOTIC_PREVALENCE',
        Interval[Now() - 84 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_PC_D1_POLYPHARMACY_10PLUS
// -----------------------------------------------------------------
// PHARMA-Care domain D1 (Medicines optimisation / polypharmacy).
// Standard 5 evidence category: clinical-care risk identification.
// Fires when the resident has ≥10 distinct active MedicineUses AND
// care_intensity is active_treatment.
//
// TODO(clinical-author): confirm PHARMA-Care domain-D1 published label.
define "PharmaCare Polypharmacy Trigger 10plus":
  QG."ActiveMedicineUseCount"(Patient.id) >= 10
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_PC_D1_POLYPHARMACY_10PLUS', 84)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_PC_D1_POLYPHARMACY_10PLUS',
        Interval[Now() - 84 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_PC_D3_ACB_ABOVE_3
// -----------------------------------------------------------------
// PHARMA-Care domain D3 (Adverse-drug-event risk / anticholinergic
// burden). Standard 5 evidence: clinical-care safety. Fires when the
// resident's computed ACB score is >3 AND care_intensity is
// active_treatment AND no recent ACB-burden review concern is open.
//
// TODO(clinical-author): confirm PHARMA-Care domain-D3 published label.
define "PharmaCare Anticholinergic Burden ACB Above 3":
  QG."AcbScoreFor"(Patient.id) > 3
  and not State."HasActiveConcernType"(Patient.id, 'ANTICHOLINERGIC_BURDEN_REVIEW_DONE')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_PC_D3_ACB_ABOVE_3', 84)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_PC_D3_ACB_ABOVE_3',
        Interval[Now() - 84 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_PC_D4_BPSD_FIRST_LINE_NONPHARM
// -----------------------------------------------------------------
// PHARMA-Care domain D4 (Behavioural-care first-line non-pharm).
// Standard 5 evidence: clinical-care best practice. Fires when a
// BPSD diagnosis is active AND a psychotropic (ATC N05A or N05B) is
// active AND no documented non-pharm intervention trial concern is
// open AND care_intensity is active_treatment.
//
// TODO(clinical-author): confirm PHARMA-Care domain-D4 published label.
define "PharmaCare BPSD First Line NonPharm":
  State."HasActiveConcernType"(Patient.id, 'BPSD_DIAGNOSIS')
  and (
    Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
    or Meds."HasActiveAtcPrefix"(Patient.id, 'N05B')
  )
  and not State."HasActiveConcernType"(Patient.id, 'NONPHARM_INTERVENTION_TRIAL')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_PC_D4_BPSD_FIRST_LINE_NONPHARM', 84)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_PC_D4_BPSD_FIRST_LINE_NONPHARM',
        Interval[Now() - 84 day, Now()])) = 0
