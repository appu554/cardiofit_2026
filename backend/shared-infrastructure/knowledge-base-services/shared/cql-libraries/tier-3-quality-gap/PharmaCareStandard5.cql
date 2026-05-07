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
 * Defines (Wave 4A vertical slice):
 *   - PharmaCare Antipsychotic Prevalence Without BPSD
 *   - PharmaCare Polypharmacy Trigger 10plus
 *   - PharmaCare Anticholinergic Burden ACB Above 3
 *   - PharmaCare BPSD First Line NonPharm
 * Defines (Wave-extension batch 2026-05):
 *   - PharmaCare D5 Pain Assessment Documentation Gap
 *   - PharmaCare D1 Polypharmacy Trigger 5plus High Risk Class
 *   - PharmaCare D3 Falls Risk Increasing Drug Burden
 *   - PharmaCare D4 Restraint Without SDM Consent Documentation
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

// -----------------------------------------------------------------
// VAIDSHALA_PC_D5_PAIN_ASSESSMENT_DOCUMENTATION_GAP
// -----------------------------------------------------------------
// PHARMA-Care domain D5 (Pain assessment / pain management).
// Standard 5 evidence: clinical-care comprehensive assessment.
// Fires when an analgesic (ATC N02A or N02B) is active AND no
// PAIN_ASSESSMENT_DOCUMENTED concern is closed in last 28 days AND
// care_intensity is active_treatment.
//
// TODO(clinical-author): confirm PHARMA-Care domain-D5 published label.
define "PharmaCare D5 Pain Assessment Documentation Gap":
  (
    Meds."HasActiveAtcPrefix"(Patient.id, 'N02A')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N02B')
  )
  and not State."HasActiveConcernType"(Patient.id, 'PAIN_ASSESSMENT_DOCUMENTED')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_PC_D5_PAIN_ASSESSMENT_DOCUMENTATION_GAP', 28)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_PC_D5_PAIN_ASSESSMENT_DOCUMENTATION_GAP',
        Interval[Now() - 28 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_PC_D1_POLYPHARMACY_5PLUS_HIGH_RISK
// -----------------------------------------------------------------
// PHARMA-Care domain D1 (Medicines optimisation / polypharmacy).
// Standard 5 evidence: clinical-care risk identification.
// Lower-threshold polypharmacy gate (>=5 active medicines) when a
// HIGH_RISK_DRUG_CLASS_PRESENT concern is open. Complements the
// existing 10-plus rule with a more sensitive flag for high-risk
// combinations.
//
// TODO(clinical-author): confirm PHARMA-Care domain-D1 published label.
define "PharmaCare D1 Polypharmacy Trigger 5plus High Risk Class":
  QG."ActiveMedicineUseCount"(Patient.id) >= 5
  and State."HasActiveConcernType"(Patient.id, 'HIGH_RISK_DRUG_CLASS_PRESENT')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_PC_D1_POLYPHARMACY_5PLUS_HIGH_RISK', 84)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_PC_D1_POLYPHARMACY_5PLUS_HIGH_RISK',
        Interval[Now() - 84 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_PC_D3_FALLS_RISK_DRUG_BURDEN
// -----------------------------------------------------------------
// PHARMA-Care domain D3 (Adverse-drug-event risk / falls-risk
// drugs). Standard 5 evidence: clinical-care safety. Fires when the
// resident has an active FALLS_RISK_INCREASING_DRUG_PRESENT concern
// AND a recent fall_within_90d Event is recorded AND care_intensity
// is active_treatment.
//
// TODO(clinical-author): confirm PHARMA-Care domain-D3 falls-risk
// drug taxonomy published label.
define "PharmaCare D3 Falls Risk Increasing Drug Burden":
  State."HasActiveConcernType"(Patient.id, 'FALLS_RISK_INCREASING_DRUG_PRESENT')
  and QG."HasRecentEventOfType"(Patient.id, 'fall', 90)
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_PC_D3_FALLS_RISK_DRUG_BURDEN', 60)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_PC_D3_FALLS_RISK_DRUG_BURDEN',
        Interval[Now() - 60 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_PC_D4_RESTRAINT_WITHOUT_SDM_CONSENT
// -----------------------------------------------------------------
// PHARMA-Care domain D4 (Restraint / behavioural-care). Quality of
// Care Principles 2025 + Aged Care Act 2024 require documented
// substitute-decision-maker (SDM) consent for chemical restraint.
// Fires when an antipsychotic (ATC N05A) or benzodiazepine (N05BA)
// is active AND a CHEMICAL_RESTRAINT_PRESCRIBING concern is open
// AND no SDM_CONSENT_DOCUMENTED concern is closed AND care_intensity
// is active_treatment.
//
// TODO(clinical-author): confirm PHARMA-Care domain-D4 restraint
// documentation published label.
define "PharmaCare D4 Restraint Without SDM Consent":
  (
    Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N05BA')
  )
  and State."HasActiveConcernType"(Patient.id, 'CHEMICAL_RESTRAINT_PRESCRIBING')
  and not State."HasActiveConcernType"(Patient.id, 'SDM_CONSENT_DOCUMENTED')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_PC_D4_RESTRAINT_WITHOUT_SDM_CONSENT', 14)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_PC_D4_RESTRAINT_WITHOUT_SDM_CONSENT',
        Interval[Now() - 14 day, Now()])) = 0
