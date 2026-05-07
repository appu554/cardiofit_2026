library ANACCDefensibility version '1.0'

using FHIR version '4.0.1'

include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress
include QualityGapHelpers version '1.0' called QG

/*
 * ANACCDefensibility
 * ==================
 * Tier 3 AN-ACC (Australian National Aged Care Classification)
 * defensibility rules. Wave 4A Task 3 vertical slice ships 2 grounded
 * rules. Remaining ~8 rules are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave4A-Task3-anacc-rule-queue.md.
 *
 * AN-ACC is the published Australian Government residential aged-care
 * funding classification (replaced ACFI in October 2022). These rules
 * surface evidence packets that support an AN-ACC class reassessment
 * — they do NOT recommend a class; the rule output is workflow exhaust
 * for the RACH operator's AN-ACC assessor.
 *
 * Defines (Wave 4A vertical slice):
 *   - ANACC Functional Decline Evidence
 *   - ANACC Behavioural Evidence
 * Defines (Wave-extension batch 2026-05):
 *   - ANACC Cognitive Status Evidence
 *   - ANACC Continence Burden Evidence
 *   - ANACC Pressure Injury Risk Evidence
 */

context Patient

// -----------------------------------------------------------------
// VAIDSHALA_ANACC_FUNCTIONAL_DECLINE
// -----------------------------------------------------------------
// Fires when the resident's AKPS (Australian-modified Karnofsky
// Performance Scale) score has dropped >20 points in the last 90
// days AND no AN-ACC reassessment Event has been recorded since.
// AKPS is the AN-ACC v1.1 functional-status indicator.
define "ANACC Functional Decline Evidence":
  State."DeltaFromBaseline"(Patient.id, 'akps', 90) < -20
  and not QG."HasRecentEventOfType"(Patient.id, 'anacc_reassessment', 90)
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_ANACC_FUNCTIONAL_DECLINE', 30)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_ANACC_FUNCTIONAL_DECLINE',
        Interval[Now() - 30 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_ANACC_BEHAVIOURAL_EVIDENCE
// -----------------------------------------------------------------
// Fires when the resident has >5 behavioural-incident Events recorded
// in the last 30 days AND no AN-ACC reassessment Event has been
// recorded since. Behavioural-incident counts feed AN-ACC class 9-13
// indicators.
define "ANACC Behavioural Evidence":
  QG."EventCountSince"(Patient.id, 'behavioural_incident', 30) > 5
  and not QG."HasRecentEventOfType"(Patient.id, 'anacc_reassessment', 30)
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_ANACC_BEHAVIOURAL_EVIDENCE', 14)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_ANACC_BEHAVIOURAL_EVIDENCE',
        Interval[Now() - 14 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_ANACC_COGNITIVE_STATUS_EVIDENCE
// -----------------------------------------------------------------
// AIHW AN-ACC v1.1 cognitive-status indicator. Fires when the
// resident has a recently opened DEMENTIA_PROGRESSION_NOTED concern
// AND no AN-ACC reassessment Event in the last 90 days. Output is
// workflow exhaust supporting AN-ACC class reassessment for
// cognitive-status burden.
define "ANACC Cognitive Status Evidence":
  State."HasActiveConcernType"(Patient.id, 'DEMENTIA_PROGRESSION_NOTED')
  and not QG."HasRecentEventOfType"(Patient.id, 'anacc_reassessment', 90)
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_ANACC_COGNITIVE_STATUS_EVIDENCE', 30)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_ANACC_COGNITIVE_STATUS_EVIDENCE',
        Interval[Now() - 30 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_ANACC_CONTINENCE_BURDEN_EVIDENCE
// -----------------------------------------------------------------
// AIHW AN-ACC v1.1 continence-care indicator. Fires when more than
// 5 continence_care_event Events have been recorded in the last 14
// days AND no AN-ACC reassessment Event has been recorded since.
define "ANACC Continence Burden Evidence":
  QG."EventCountSince"(Patient.id, 'continence_care_event', 14) > 5
  and not QG."HasRecentEventOfType"(Patient.id, 'anacc_reassessment', 30)
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_ANACC_CONTINENCE_BURDEN_EVIDENCE', 14)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_ANACC_CONTINENCE_BURDEN_EVIDENCE',
        Interval[Now() - 14 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_ANACC_PRESSURE_INJURY_RISK_EVIDENCE
// -----------------------------------------------------------------
// AIHW AN-ACC v1.1 pressure-injury-risk indicator (skin integrity).
// Fires when the resident has an active PRESSURE_INJURY_STAGE_2_PLUS
// concern OR a recent pressure_injury_event Event has occurred AND
// no AN-ACC reassessment Event in the last 30 days.
define "ANACC Pressure Injury Risk Evidence":
  (
    State."HasActiveConcernType"(Patient.id, 'PRESSURE_INJURY_STAGE_2_PLUS')
      or QG."HasRecentEventOfType"(Patient.id, 'pressure_injury_event', 30)
  )
  and not QG."HasRecentEventOfType"(Patient.id, 'anacc_reassessment', 30)
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_ANACC_PRESSURE_INJURY_RISK_EVIDENCE', 14)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_ANACC_PRESSURE_INJURY_RISK_EVIDENCE',
        Interval[Now() - 14 day, Now()])) = 0
