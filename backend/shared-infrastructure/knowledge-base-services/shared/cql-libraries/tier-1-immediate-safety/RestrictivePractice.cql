library RestrictivePractice version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include ConsentStateHelpers version '1.0' called Consent
include EvidenceTraceHelpers version '1.0' called Trace

/*
 * RestrictivePractice
 * ===================
 * Tier 1 restrictive-practice + consent-gated rules (Wave 2 Task 3).
 * Each fires a CONSENT-GATHERING recommendation (not a medication-
 * change recommendation) per Layer 3 v2 doc Part 2.2.
 *
 * Defines:
 *   - Antipsychotic BPSD Consent Missing Extended
 *   - Benzodiazepine BPSD Consent Missing
 *   - Restrictive Practice Authorisation Expiring
 *   - Antipsychotic BPSD Documentation Gap
 *   - Chemical Restraint Criteria
 */

context Patient

// -----------------------------------------------------------------
// ANTIPSYCHOTIC_BPSD_CONSENT_MISSING_EXTENDED
// -----------------------------------------------------------------
// Extends ANTIPSYCHOTIC_CONSENT_MISSING anchor: drops the 90-day
// minimum so newly initiated antipsychotics for BPSD without
// active consent fire immediately (acute consent capture).
define "Antipsychotic BPSD Consent Missing Extended":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
  and not Consent."HasActiveConsentForClass"(
        Patient.id, 'Antipsychotic_BPSD_authorisation')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'ANTIPSYCHOTIC_BPSD_CONSENT_MISSING_EXTENDED',
        Interval[Now() - 7 day, Now()])) = 0

// -----------------------------------------------------------------
// BENZODIAZEPINE_BPSD_CONSENT_MISSING
// -----------------------------------------------------------------
// Same pattern for benzodiazepines (N05BA, N05CD).
define "Benzodiazepine BPSD Consent Missing":
  (
    Meds."HasActiveAtcPrefix"(Patient.id, 'N05BA')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N05CD')
  )
  and not Consent."HasActiveConsentForClass"(
        Patient.id, 'Benzodiazepine_BPSD_authorisation')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'BENZODIAZEPINE_BPSD_CONSENT_MISSING',
        Interval[Now() - 7 day, Now()])) = 0

// -----------------------------------------------------------------
// RESTRICTIVE_PRACTICE_AUTHORISATION_EXPIRING
// -----------------------------------------------------------------
// Authorisation < 30 days from expiry → fire consent-refresh
// recommendation.
define "Restrictive Practice Authorisation Expiring":
  Consent."HasActiveConsentForClass"(
    Patient.id, 'Restrictive_practice_authorisation')
  and Consent."ConsentExpiringWithin"(
    Patient.id, 'Restrictive_practice_authorisation', 30)
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'RESTRICTIVE_PRACTICE_AUTHORISATION_EXPIRING',
        Interval[Now() - 30 day, Now()])) = 0

// -----------------------------------------------------------------
// ANTIPSYCHOTIC_BPSD_DOCUMENTATION_GAP
// -----------------------------------------------------------------
// On antipsychotic AND no behavioural-chart observation in last 7d.
define "Antipsychotic BPSD Documentation Gap":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
  and (
    State."LatestObservationAt"(Patient.id, 'behavioural_chart') is null
      or State."LatestObservationAt"(Patient.id, 'behavioural_chart') < (Now() - 7 day)
  )
  and not State."HasActiveConcernType"(Patient.id, 'BEHAVIOURAL_CHART_REVIEWED')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'ANTIPSYCHOTIC_BPSD_DOCUMENTATION_GAP',
        Interval[Now() - 7 day, Now()])) = 0

// -----------------------------------------------------------------
// CHEMICAL_RESTRAINT_CRITERIA
// -----------------------------------------------------------------
// Antipsychotic dose escalation AND behavioural-chart absent AND no
// de-escalation trial concern logged. Fires consent-gathering for
// chemical restraint authorisation.
define "Chemical Restraint Criteria":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
  and (
    State."LatestObservationAt"(Patient.id, 'behavioural_chart') is null
      or State."LatestObservationAt"(Patient.id, 'behavioural_chart') < (Now() - 7 day)
  )
  and not State."HasActiveConcernType"(Patient.id, 'DE_ESCALATION_TRIAL_LOGGED')
  and not Consent."HasActiveConsentForClass"(
        Patient.id, 'Chemical_restraint_authorisation')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'CHEMICAL_RESTRAINT_CRITERIA',
        Interval[Now() - 7 day, Now()])) = 0
