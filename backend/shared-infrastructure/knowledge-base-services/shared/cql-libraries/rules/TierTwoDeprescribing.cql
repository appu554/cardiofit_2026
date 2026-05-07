library TierTwoDeprescribing version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace

/*
 * TierTwoDeprescribing
 * ====================
 * Anchor rule: PPI_LONG_TERM_NO_INDICATION.
 *
 * Encoded substrate-aware (uses BaselineFor / DurationDaysActive /
 * HasActiveConcernType / CurrentCareIntensity helpers — NOT raw
 * observation values). Validator's snapshot-semantics gate will
 * accept this define.
 *
 * Spec: shared/cql-libraries/examples/ppi-deprescribe.yaml
 */

context Patient

define "Long-Term PPI Without Indication":
  exists(MedicationRequest M
    where Meds."HasActiveAtcPrefix"(Patient.id, 'A02BC')
      and Meds."DurationDaysActive"(M) > 56
  )
  and not State."HasActiveConcernType"(Patient.id, 'GI_BLEED_WATCH_14D')
  and State."CurrentCareIntensity"(Patient.id) <> 'palliative'
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'PPI_LONG_TERM_NO_INDICATION',
        Interval[Now() - 90 day, Now()])) = 0
