library StoppV3 version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress

/*
 * StoppV3
 * =======
 * Tier 2 deprescribing rules sourced from STOPP v3 (O'Mahony et al.,
 * Eur Geriatr Med 2023; published criterion list). Wave 3 Task 2
 * vertical slice ships ONE rule (STOPP B1). Remaining ~30 STOPP v3
 * criteria are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave3-Task2-rule-queue.md.
 *
 * Defines:
 *   - STOPP B1 Aspirin Primary Prevention
 */

context Patient

// -----------------------------------------------------------------
// STOPP_B1_ASPIRIN_PRIMARY_PREVENTION
// -----------------------------------------------------------------
// Source: STOPP v3 §B1 — "Aspirin in primary prevention without
// cardiovascular indication" (O'Mahony et al. 2023, published list).
// Fires when aspirin (ATC B01AC06) is active AND the resident has no
// active cardiovascular secondary-prevention concern AND care_intensity
// is active_treatment (palliative residents handled by separate
// Tier 1 antiplatelet-palliative rule).
define "STOPP B1 Aspirin Primary Prevention":
  Meds."HasActiveAtcPrefix"(Patient.id, 'B01AC06')
  and not State."HasActiveConcernType"(Patient.id, 'CHD_SECONDARY_PREVENTION')
  and not State."HasActiveConcernType"(Patient.id, 'CVD_SECONDARY_PREVENTION')
  and not State."HasActiveConcernType"(Patient.id, 'CEREBROVASCULAR_SECONDARY_PREVENTION')
  and not State."HasActiveConcernType"(Patient.id, 'PERIPHERAL_ARTERIAL_DISEASE')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'STOPP_B1_ASPIRIN_PRIMARY_PREVENTION', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'STOPP_B1_ASPIRIN_PRIMARY_PREVENTION',
        Interval[Now() - 90 day, Now()])) = 0
