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
