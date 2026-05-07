library Lifecycle version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include ConsentStateHelpers version '1.0' called Consent
include MonitoringHelpers version '1.0' called Monitor
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress

/*
 * Lifecycle (Tier 4 surveillance — deadlines & expiries)
 * ======================================================
 * Lifecycle-aware surveillance rules: review cycles, consent
 * expiries, and monitoring-plan SLA breaches. They surface a
 * lifecycle event for the action surface (e.g. "antipsychotic review
 * is overdue") rather than a clinical-state finding.
 *
 * Wave 5 Task 1 vertical slice ships 3 lifecycle defines:
 *   - Antipsychotic Review Overdue 3mo
 *   - Consent Expiring Within 30d
 *   - Monitoring Plan Overdue Observation
 *
 * Wave-extension batch (2026-05) adds 6 more lifecycle defines:
 *   - PPI Review Overdue 6mo
 *   - Statin Review Overdue 12mo
 *   - Opioid Review Overdue 3mo
 *   - Antimicrobial Review Overdue 7d
 *   - Prescriber Credential Expiring Within 30d
 *   - Prescribing Agreement Expiring Within 30d
 */

context Patient

// -----------------------------------------------------------------
// VAIDSHALA_T4_ANTIPSYCHOTIC_REVIEW_OVERDUE_3MO
// -----------------------------------------------------------------
// ADG (Aged Care Diagnostics Guidelines) 2025 antipsychotic review
// cycle: residents on a continuous antipsychotic (ATC N05A) for
// > 12 weeks must have a documented review every 90 days.
//
// TODO(layer1-bind): bind to the published ADG 2025 review-cycle
// section once Pipeline-2 extracts the source PDF.
define "Antipsychotic Review Overdue 3mo":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
  and not State."HasActiveConcernType"(
        Patient.id, 'ANTIPSYCHOTIC_REVIEW_DONE_90D')
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_ANTIPSYCHOTIC_REVIEW_OVERDUE_3MO',
        Interval[Now() - 30 day, Now()])) = 0
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_ANTIPSYCHOTIC_REVIEW_OVERDUE_3MO', 30)

// -----------------------------------------------------------------
// VAIDSHALA_T4_CONSENT_EXPIRING_WITHIN_30D
// -----------------------------------------------------------------
// Restrictive practice authorisation (Quality of Care Principles
// 2025, Aged Care Act 2024) consent expires within 30 days. Surfaces
// a renewal action; suppressed once renewal action is recorded.
define "Consent Expiring Within 30d":
  Consent."ConsentExpiringWithin"(
        Patient.id, 'restrictive_practice', 30)
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_CONSENT_EXPIRING_WITHIN_30D', 14)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_CONSENT_EXPIRING_WITHIN_30D',
        Interval[Now() - 14 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_T4_MONITORING_PLAN_OVERDUE_OBSERVATION
// -----------------------------------------------------------------
// Active monitoring plan has at least one overdue observation per
// the Layer 2 §4.3 Monitoring state machine. Fires informationally
// to drive observation-collection workflow; suppressed if the action
// has been actioned in the last 7 days.
define "Monitoring Plan Overdue Observation":
  exists(
    Monitor."ActivePlansFor"(Patient.id) plan
      where Monitor."ObservationOverdueBy"(plan.planRef, 0) > 0
  )
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_MONITORING_PLAN_OVERDUE_OBSERVATION', 7)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_MONITORING_PLAN_OVERDUE_OBSERVATION',
        Interval[Now() - 7 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_T4_PPI_REVIEW_OVERDUE_6MO
// -----------------------------------------------------------------
// ADG (Aged Care Diagnostics Guidelines) 2025 PPI review cycle:
// PPI (ATC A02BC) on chronic prescription should have a documented
// step-down/cessation review every 6 months.
//
// TODO(layer1-bind): bind to ADG 2025 PPI section once Pipeline-2
// extracts the source PDF.
define "PPI Review Overdue 6mo":
  Meds."HasActiveAtcPrefix"(Patient.id, 'A02BC')
  and not State."HasActiveConcernType"(Patient.id, 'PPI_REVIEW_DONE_180D')
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_PPI_REVIEW_OVERDUE_6MO',
        Interval[Now() - 30 day, Now()])) = 0
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_PPI_REVIEW_OVERDUE_6MO', 30)

// -----------------------------------------------------------------
// VAIDSHALA_T4_STATIN_REVIEW_OVERDUE_12MO
// -----------------------------------------------------------------
// ADG (Aged Care Diagnostics Guidelines) 2025 statin review cycle:
// statin (ATC C10AA) on chronic prescription should have a
// documented continued-indication review every 12 months in the
// elderly aged-care population.
//
// TODO(layer1-bind): bind to ADG 2025 statin section once Pipeline-2
// extracts the source PDF.
define "Statin Review Overdue 12mo":
  Meds."HasActiveAtcPrefix"(Patient.id, 'C10AA')
  and not State."HasActiveConcernType"(Patient.id, 'STATIN_REVIEW_DONE_365D')
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_STATIN_REVIEW_OVERDUE_12MO',
        Interval[Now() - 60 day, Now()])) = 0
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_STATIN_REVIEW_OVERDUE_12MO', 60)

// -----------------------------------------------------------------
// VAIDSHALA_T4_OPIOID_REVIEW_OVERDUE_3MO
// -----------------------------------------------------------------
// ADG (Aged Care Diagnostics Guidelines) 2025 opioid review cycle:
// strong opioid (ATC N02AA) on chronic prescription should have a
// documented analgesic-effectiveness review every 90 days.
//
// TODO(layer1-bind): bind to ADG 2025 opioid section once Pipeline-2
// extracts the source PDF.
define "Opioid Review Overdue 3mo":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N02AA')
  and not State."HasActiveConcernType"(Patient.id, 'OPIOID_REVIEW_DONE_90D')
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_OPIOID_REVIEW_OVERDUE_3MO',
        Interval[Now() - 30 day, Now()])) = 0
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_OPIOID_REVIEW_OVERDUE_3MO', 30)

// -----------------------------------------------------------------
// VAIDSHALA_T4_ANTIMICROBIAL_REVIEW_OVERDUE_7D
// -----------------------------------------------------------------
// ADG (Aged Care Diagnostics Guidelines) 2025 antimicrobial review
// cycle (echoes ACSQHC AMS standard): antimicrobial (ATC J01) on
// active prescription should have a documented continuation review
// every 7 days.
//
// TODO(layer1-bind): bind to ADG 2025 antimicrobial section once
// Pipeline-2 extracts the source PDF.
define "Antimicrobial Review Overdue 7d":
  Meds."HasActiveAtcPrefix"(Patient.id, 'J01')
  and not State."HasActiveConcernType"(Patient.id, 'ANTIMICROBIAL_REVIEW_DONE_7D')
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_ANTIMICROBIAL_REVIEW_OVERDUE_7D',
        Interval[Now() - 3 day, Now()])) = 0
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_ANTIMICROBIAL_REVIEW_OVERDUE_7D', 3)

// -----------------------------------------------------------------
// VAIDSHALA_T4_PRESCRIBER_CREDENTIAL_EXPIRING_WITHIN_30D
// -----------------------------------------------------------------
// Prescriber credential / authorisation expiring within 30 days. The
// Aged Care Act 2024 + Quality of Care Principles 2025 require an
// active prescriber authorisation for restrictive-practice and
// Schedule-8 prescribing. Fires informationally to drive renewal
// workflow.
define "Prescriber Credential Expiring Within 30d":
  Consent."ConsentExpiringWithin"(
        Patient.id, 'prescriber_credential', 30)
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_PRESCRIBER_CREDENTIAL_EXPIRING_WITHIN_30D', 14)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_PRESCRIBER_CREDENTIAL_EXPIRING_WITHIN_30D',
        Interval[Now() - 14 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_T4_PRESCRIBING_AGREEMENT_EXPIRING_WITHIN_30D
// -----------------------------------------------------------------
// Resident-prescriber prescribing agreement expiring within 30 days
// (RACGP / PSA aged-care care-plan agreement). Fires informationally
// to drive renewal workflow.
define "Prescribing Agreement Expiring Within 30d":
  Consent."ConsentExpiringWithin"(
        Patient.id, 'prescribing_agreement', 30)
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_PRESCRIBING_AGREEMENT_EXPIRING_WITHIN_30D', 14)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_PRESCRIBING_AGREEMENT_EXPIRING_WITHIN_30D',
        Interval[Now() - 14 day, Now()])) = 0
