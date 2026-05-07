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
