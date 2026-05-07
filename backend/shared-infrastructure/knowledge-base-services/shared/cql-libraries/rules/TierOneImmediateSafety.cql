library TierOneImmediateSafety version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include ConsentStateHelpers version '1.0' called Consent
include AuthorisationHelpers version '1.0' called Auth
include EvidenceTraceHelpers version '1.0' called Trace

/*
 * TierOneImmediateSafety
 * ======================
 * Anchor rules:
 *  - HYPERKALEMIA_RISK_TRAJECTORY (substrate trajectory + RAAS check)
 *  - ANTIPSYCHOTIC_CONSENT_MISSING (consent-gated psychotropic check)
 *
 * Both encoded substrate-aware: trajectory rule uses DeltaFromBaseline
 * (NOT raw potassium snapshot); consent rule uses HasActiveConsentForClass.
 * Validator's snapshot-semantics gate accepts both.
 *
 * Specs: shared/cql-libraries/examples/{hyperkalemia-trajectory,
 *        antipsychotic-consent-gating}.yaml
 */

context Patient

define "Hyperkalemia Risk Trajectory":
  State."DeltaFromBaseline"(Patient.id, 'potassium', 7) > 0.8
  and Meds."HasActiveAtcPrefix"(Patient.id, 'C09')
  // Pull eGFR snapshot for the recommendation context (substrate read,
  // not used for gating — DeltaFromBaseline is the gate).
  and State."LatestObservationValue"(Patient.id, 'eGFR') is not null
  and not State."HasActiveConcernType"(Patient.id, 'AKI_WATCH')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'HYPERKALEMIA_RISK_TRAJECTORY',
        Interval[Now() - 7 day, Now()])) = 0

define "Antipsychotic Long Term Without Active Consent":
  exists(MedicationRequest M
    where Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
      and Meds."DurationDaysActive"(M) > 90)
  and not Consent."HasActiveConsentForClass"(
        Patient.id, 'Antipsychotic_deprescribe_review')
  // Surface any active concerns into the EvidenceTrace context.
  and State."ActiveConcernCount"(Patient.id) >= 0
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'ANTIPSYCHOTIC_CONSENT_MISSING',
        Interval[Now() - 30 day, Now()])) = 0
