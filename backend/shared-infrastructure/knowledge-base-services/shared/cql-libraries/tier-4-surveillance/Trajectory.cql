library Trajectory version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress
include QualityGapHelpers version '1.0' called QG

/*
 * Trajectory (Tier 4 surveillance — informational)
 * ================================================
 * Surveillance rules whose primary output is a trajectory signal.
 * They fire informationally (suppression-class 5 substrate-state
 * governs whether the trajectory crosses a clinical threshold and is
 * surfaced to the action surface; absent the threshold, the rule's
 * fire becomes EvidenceTrace exhaust only, per Layer 3 v2 doc Part
 * 2.1).
 *
 * Wave 5 Task 1 vertical slice ships 3 trajectory defines:
 *   - eGFR Trajectory Decline 90d
 *   - Weight Loss Trajectory 90d
 *   - Behavioural Episode Frequency Change 14d
 *
 * The remaining surveillance rules are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave5-Task1-tier4-rule-queue.md.
 *
 * All three rules use ClinicalStateHelpers substrate primitives
 * (DeltaFromBaseline / IsTrending / VelocityFlag) so the snapshot-
 * semantics gate is satisfied.
 */

context Patient

// -----------------------------------------------------------------
// VAIDSHALA_T4_EGFR_TRAJECTORY_DECLINE_90D
// -----------------------------------------------------------------
// eGFR trending DOWN over 90 days — surveillance signal feeding
// kb-20 stratum + Tier 1 hyperkalemia trajectory rule cluster. Uses
// the Layer 2 baseline_state machine via DeltaFromBaseline + the
// IsTrending primitive (Wave 3.4 trajectory detector). Care
// intensity is NOT gated — surveillance fires for palliative and
// active_treatment alike.
//
// TODO(layer1-bind): bind exact KDIGO 2024 eGFR rate-of-decline
// threshold once Pipeline-2 publishes the source.
define "eGFR Trajectory Decline 90d":
  State."IsTrending"(Patient.id, 'egfr', 90, 'down')
  and State."DeltaFromBaseline"(Patient.id, 'egfr', 90) <= -5.0
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_EGFR_TRAJECTORY_DECLINE_90D', 30)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_EGFR_TRAJECTORY_DECLINE_90D',
        Interval[Now() - 30 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_T4_WEIGHT_LOSS_TRAJECTORY_90D
// -----------------------------------------------------------------
// Unintentional weight loss > 5% over 90 days — surveillance signal.
// AN-ACC v1.1 + Aged Care Quality Standard 5 cite 5%/90d as the
// flagged-decline threshold for clinical-care review. Uses
// DeltaFromBaseline against the resident weight baseline.
define "Weight Loss Trajectory 90d":
  State."DeltaFromBaseline"(Patient.id, 'weight_kg', 90) <= -0.05
    * State."BaselineFor"(Patient.id, 'weight_kg')
  and not State."HasActiveConcernType"(
        Patient.id, 'WEIGHT_LOSS_INTENTIONAL')
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_WEIGHT_LOSS_TRAJECTORY_90D', 30)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_WEIGHT_LOSS_TRAJECTORY_90D',
        Interval[Now() - 30 day, Now()])) = 0

// -----------------------------------------------------------------
// VAIDSHALA_T4_BEHAVIOURAL_EPISODE_FREQUENCY_CHANGE_14D
// -----------------------------------------------------------------
// Behavioural-episode count over the trailing 14 days is >= 1.5x
// the prior 14-day baseline (50% increase). PHARMA-Care domain D4
// surveillance + AN-ACC behavioural-evidence support.
//
// Uses EventCountSince to count behavioural_episode events in the
// recent and baseline windows. Absent a baseline (resident is new),
// the rule does not fire.
define "Behavioural Episode Frequency Change 14d":
  QG."EventCountSince"(Patient.id, 'behavioural_episode', 14) >=
    1.5 * QG."EventCountSince"(Patient.id, 'behavioural_episode_baseline_14d', 14)
  and QG."EventCountSince"(Patient.id, 'behavioural_episode_baseline_14d', 14) > 0
  and not Suppress."WasActionedRecently"(
        Patient.id, 'VAIDSHALA_T4_BEHAVIOURAL_EPISODE_FREQUENCY_CHANGE_14D', 14)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'VAIDSHALA_T4_BEHAVIOURAL_EPISODE_FREQUENCY_CHANGE_14D',
        Interval[Now() - 14 day, Now()])) = 0
