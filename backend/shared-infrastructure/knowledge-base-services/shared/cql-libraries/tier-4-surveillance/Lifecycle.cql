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
