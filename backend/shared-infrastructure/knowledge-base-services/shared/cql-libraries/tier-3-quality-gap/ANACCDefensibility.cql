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
 * Defines:
 *   - ANACC Functional Decline Evidence
 *   - ANACC Behavioural Evidence
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
