library Wang2024 version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress

/*
 * Wang2024
 * ========
 * Tier 2 deprescribing rules from the Australian PIMs criteria
 * (Wang et al. 2024 AU-PIMs list — AU-specific potentially
 * inappropriate medications in older adults). Wave 3 Task 2 vertical
 * slice ships ONE rule (Wang 2024 §3 antipsychotic in dementia).
 * Remaining ~18 Wang 2024 criteria are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave3-Task2-rule-queue.md.
 *
 * Defines:
 *   - Wang 2024 Antipsychotic Dementia AU
 *   - Wang 2024 Anticholinergic Cognitive Impairment AU
 *   - Wang 2024 Strong Opioid Without Cancer Pain AU
 *   - Wang 2024 Long Term PPI Without Indication AU
 */

context Patient

// -----------------------------------------------------------------
// WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU
// -----------------------------------------------------------------
// Source: Wang 2024 AU-PIMs §3 — "Antipsychotics in dementia: limit
// duration to ≤12 weeks; document BPSD review at each prescription
// renewal" (Wang et al., AU-PIMs criteria 2024).
//
// Fires when an antipsychotic (ATC N05A) is active AND a dementia
// diagnosis is active AND there is no documented BPSD review concern
// closed within the last 90 days AND care_intensity is
// active_treatment.
//
// Distinguished from the Tier 1 RestrictivePractice consent-gathering
// rule: this rule drafts a deprescribing/dose-review recommendation
// for the SDM/clinician, scoped to AU aged-care BPSD prescribing
// review cycle.
define "Wang 2024 Antipsychotic Dementia AU":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
  and State."HasActiveConcernType"(Patient.id, 'DEMENTIA')
  and not State."HasActiveConcernType"(Patient.id, 'BPSD_REVIEW_DONE_90D')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// WANG_2024_ANTICHOLINERGIC_COGNITIVE_IMPAIRMENT_AU
// -----------------------------------------------------------------
// Source: Wang 2024 AU-PIMs §1 — "Anticholinergic medications
// (first-generation antihistamines, tricyclic antidepressants,
// antimuscarinic urinary spasmolytics) in older adults with
// cognitive impairment or dementia (AU-specific PIM list)."
// (Wang et al., AU-PIMs criteria 2024).
// Fires when an anticholinergic-class medication (R06AA, R06AB,
// N06AA, G04BD) is active AND a cognitive-impairment or dementia
// concern is open AND care_intensity is active_treatment.
define "Wang 2024 Anticholinergic Cognitive Impairment AU":
  (
    Meds."HasActiveAtcPrefix"(Patient.id, 'R06AA')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'R06AB')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N06AA')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'G04BD')
  )
  and (
    State."HasActiveConcernType"(Patient.id, 'COGNITIVE_IMPAIRMENT')
      or State."HasActiveConcernType"(Patient.id, 'DEMENTIA')
  )
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'WANG_2024_ANTICHOLINERGIC_COGNITIVE_IMPAIRMENT_AU', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'WANG_2024_ANTICHOLINERGIC_COGNITIVE_IMPAIRMENT_AU',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// WANG_2024_STRONG_OPIOID_WITHOUT_CANCER_PAIN_AU
// -----------------------------------------------------------------
// Source: Wang 2024 AU-PIMs §7 — "Strong opioids (e.g. morphine,
// oxycodone, hydromorphone) in older adults without an active
// cancer-pain or palliative indication (AU-specific PIM list)."
// (Wang et al., AU-PIMs criteria 2024).
// Fires when a strong opioid (ATC N02AA) is active AND no cancer
// pain concern is open AND care_intensity is active_treatment.
define "Wang 2024 Strong Opioid Without Cancer Pain AU":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N02AA')
  and not State."HasActiveConcernType"(Patient.id, 'CANCER_PAIN_INDICATION')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'WANG_2024_STRONG_OPIOID_WITHOUT_CANCER_PAIN_AU', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'WANG_2024_STRONG_OPIOID_WITHOUT_CANCER_PAIN_AU',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// WANG_2024_LONG_TERM_PPI_WITHOUT_INDICATION_AU
// -----------------------------------------------------------------
// Source: Wang 2024 AU-PIMs §11 — "Long-term proton pump inhibitor
// (PPI; ATC A02BC) use in older adults without an evidence-based
// indication (e.g. complicated GORD, Barrett's oesophagus, NSAID
// GI prophylaxis) (AU-specific PIM list)." (Wang et al. 2024).
// Fires when a PPI is active AND no GORD/Barrett/NSAID-prophylaxis
// concern is open AND care_intensity is active_treatment.
define "Wang 2024 Long Term PPI Without Indication AU":
  Meds."HasActiveAtcPrefix"(Patient.id, 'A02BC')
  and not State."HasActiveConcernType"(Patient.id, 'GORD_COMPLICATED')
  and not State."HasActiveConcernType"(Patient.id, 'BARRETT_OESOPHAGUS')
  and not State."HasActiveConcernType"(Patient.id, 'NSAID_GI_PROPHYLAXIS_INDICATION')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'WANG_2024_LONG_TERM_PPI_WITHOUT_INDICATION_AU', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'WANG_2024_LONG_TERM_PPI_WITHOUT_INDICATION_AU',
        Interval[Now() - 90 day, Now()])) = 0
