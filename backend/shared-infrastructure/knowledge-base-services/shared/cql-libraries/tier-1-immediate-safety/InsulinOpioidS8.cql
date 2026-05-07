library InsulinOpioidS8 version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include AuthorisationHelpers version '1.0' called Auth
include EvidenceTraceHelpers version '1.0' called Trace

/*
 * InsulinOpioidS8
 * ===============
 * Tier 1 insulin/hypoglycemia + Schedule 8 routing rules
 * (Wave 2 Task 4). Heavy use of AuthorisationHelpers.
 *
 * Defines:
 *   - Insulin Hypoglycemia Imminent
 *   - Insulin Dose Adjustment Requiring S8
 *   - Opioid Escalation S8 Authorisation Check
 *   - Opioid Naive Start S8 Authorisation Check
 *   - Oxycodone Fentanyl Titration
 */

context Patient

// -----------------------------------------------------------------
// INSULIN_HYPOGLYCEMIA_IMMINENT
// -----------------------------------------------------------------
// BGL trajectory descending + insulin administered in last 4h +
// no oral intake observation in last 6h.
define "Insulin Hypoglycemia Imminent":
  State."IsTrending"(Patient.id, 'BGL', 'down')
  and State."LatestObservationAt"(Patient.id, 'insulin_administration')
        > (Now() - 4 hour)
  and (
    State."LatestObservationAt"(Patient.id, 'oral_intake') is null
      or State."LatestObservationAt"(Patient.id, 'oral_intake') < (Now() - 6 hour)
  )
  and not State."HasActiveConcernType"(Patient.id, 'HYPOGLYCEMIA_INTERVENTION_OPEN')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'INSULIN_HYPOGLYCEMIA_IMMINENT',
        Interval[Now() - 2 hour, Now()])) = 0

// -----------------------------------------------------------------
// INSULIN_DOSE_ADJUSTMENT_REQUIRING_S8
// -----------------------------------------------------------------
// Insulin dose change >20% (BGL baseline change driving titration)
// AND no S8-authorised prescriber available → routes via fallback.
define "Insulin Dose Adjustment Requiring S8":
  Meds."HasActiveAtcPrefix"(Patient.id, 'A10A')
  and State."DeltaFromBaseline"(Patient.id, 'BGL', 3) > 5
  and not Auth."HasAvailablePrescriberForClass"(
        'facility/default', 'Insulin_dose_adjustment')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'INSULIN_DOSE_ADJUSTMENT_REQUIRING_S8',
        Interval[Now() - 12 hour, Now()])) = 0

// -----------------------------------------------------------------
// OPIOID_ESCALATION_S8_AUTHORISATION_CHECK
// -----------------------------------------------------------------
// MEDD escalation against baseline + AuthorisationHelpers role check
// for Schedule 8 prescribing.
define "Opioid Escalation S8 Authorisation Check":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N02A')
  and Meds."MorphineEquivalentMgPerDay"(Patient.id) > 60
  and not Auth."HasAvailablePrescriberForClass"(
        'facility/default', 'Schedule_8_opioid')
  and not State."HasActiveConcernType"(Patient.id, 'PALLIATIVE_PAIN_PLAN_ACTIVE')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'OPIOID_ESCALATION_S8_AUTHORISATION_CHECK',
        Interval[Now() - 24 hour, Now()])) = 0

// -----------------------------------------------------------------
// OPIOID_NAIVE_START_S8_AUTHORISATION_CHECK
// -----------------------------------------------------------------
// First opioid prescription AND no S8-authorised prescriber on shift.
define "Opioid Naive Start S8 Authorisation Check":
  exists(MedicationRequest M
    where Meds."HasActiveAtcPrefix"(Patient.id, 'N02A')
      and Meds."DurationDaysActive"(M) <= 1)
  and not Auth."HasAvailablePrescriberForClass"(
        'facility/default', 'Schedule_8_opioid')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'OPIOID_NAIVE_START_S8_AUTHORISATION_CHECK',
        Interval[Now() - 24 hour, Now()])) = 0

// -----------------------------------------------------------------
// OXYCODONE_FENTANYL_TITRATION
// -----------------------------------------------------------------
// Both oxycodone and fentanyl active (suggesting conversion attempt)
// AND no specialist-input concern logged.
define "Oxycodone Fentanyl Titration":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N02AA05')
  and Meds."HasActiveAtcPrefix"(Patient.id, 'N02AB03')
  and not State."HasActiveConcernType"(Patient.id, 'PAIN_SPECIALIST_INPUT_LOGGED')
  and not Auth."HasAvailablePrescriberForClass"(
        'facility/default', 'Schedule_8_opioid_specialist')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'OXYCODONE_FENTANYL_TITRATION',
        Interval[Now() - 24 hour, Now()])) = 0
