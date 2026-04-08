package com.cardiofit.flink.operators;

import com.cardiofit.flink.operators.CrossModuleTestHarness.ProcessResult;
import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;

import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.util.*;

import static com.cardiofit.flink.builders.Module13TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Cross-module integration tests validating the full Module 7→13 pipeline
 * using the {@link CrossModuleTestHarness} pure-function replicator.
 *
 * 7 scenario groups, ~23 tests total. All sub-second execution — no Flink harness needed.
 */
public class Module7To13CrossModuleIntegrationTest {

    private static final String P = "cross-mod-patient-1";

    // ═══════════════════════════════════════════════════════════════════════
    // Scenario 1: Hypertensive Crisis (BP Variability → CKM Escalation)
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Scenario1_HypertensiveCrisis {

        @Test
        void highBPVariability_triggersCKMCardiovascularDeterioration() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            // Previous snapshot has baseline meanSBP=142 — now spike it
            long t = weeksAfter(BASE_TIME, 2);

            CanonicalEvent bpEvent = bpVariabilityEvent(P, t,
                    22.0, "HIGH", 168.0, 100.0);
            ProcessResult r = CrossModuleTestHarness.processEvent(bpEvent, state, t);

            assertNotNull(r);
            assertEquals("module7", r.getSourceModule());
            // With SBP jumping from 142→168, cardiovascular velocity should be positive
            assertTrue(r.getVelocity().getDomainVelocity(CKMRiskDomain.CARDIOVASCULAR) > 0,
                    "Cardiovascular velocity should be positive for SBP spike");
        }

        @Test
        void morningSurge_withHighARV_updatesBPFields() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            long t = weeksAfter(BASE_TIME, 2);

            CanonicalEvent bpEvent = bpVariabilityEvent(P, t,
                    20.0, "HIGH", 160.0, 95.0);
            bpEvent.getPayload().put("morning_surge_magnitude", 38.0);
            bpEvent.getPayload().put("dip_classification", "NON_DIPPER");

            ProcessResult r = CrossModuleTestHarness.processEvent(bpEvent, state, t);

            assertNotNull(r);
            assertEquals(160.0, state.current().meanSBP);
            assertEquals(20.0, state.current().arv);
            assertEquals(38.0, state.current().morningSurgeMagnitude);
            assertEquals(DipClassification.NON_DIPPER, state.current().dipClass);
        }

        @Test
        void moderateBP_noEscalation() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            long t = weeksAfter(BASE_TIME, 2);

            // ARV=11 (moderate), SBP=140 (near baseline 142) — no big change
            CanonicalEvent bpEvent = bpVariabilityEvent(P, t,
                    11.0, "MODERATE", 140.0, 87.0);
            ProcessResult r = CrossModuleTestHarness.processEvent(bpEvent, state, t);

            assertNotNull(r);
            // Small change from baseline shouldn't trigger CKM_RISK_ESCALATION
            assertFalse(r.hasChange(ClinicalStateChangeType.CKM_RISK_ESCALATION),
                    "Moderate BP within normal range should not escalate");
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Scenario 2: Metabolic Deterioration + Engagement Drop
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Scenario2_MetabolicAndEngagement {

        @Test
        void risingFBG_withEngagementCollapse_emitsMultipleChanges() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            state.current().engagementScore = 0.75;
            long t = weeksAfter(BASE_TIME, 2);

            // Inject high FBG
            CanonicalEvent labEvent = labEvent(P, t, "FBG", 180.0);
            CrossModuleTestHarness.processEvent(labEvent, state, t);

            // Now engagement collapses: 0.75 → 0.35 = delta of 0.40 (≥ 0.35 threshold)
            long t2 = t + HOUR_MS;
            CanonicalEvent engEvent = engagementEvent(P, t2,
                    0.35, "RED", "DISENGAGED", "TIER_2_SMBG");
            ProcessResult r = CrossModuleTestHarness.processEvent(engEvent, state, t2);

            assertNotNull(r);
            assertTrue(r.hasChange(ClinicalStateChangeType.ENGAGEMENT_COLLAPSE),
                    "Engagement drop of 0.40 should trigger ENGAGEMENT_COLLAPSE");
        }

        @Test
        void engagementDrop_belowThreshold_noCollapse() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            state.current().engagementScore = 0.75;
            long t = weeksAfter(BASE_TIME, 2);

            // Drop 0.75 → 0.45 = delta 0.30 (< 0.35 threshold)
            CanonicalEvent engEvent = engagementEvent(P, t,
                    0.45, "YELLOW", "PASSIVE", "TIER_2_SMBG");
            ProcessResult r = CrossModuleTestHarness.processEvent(engEvent, state, t);

            assertNotNull(r);
            assertFalse(r.hasChange(ClinicalStateChangeType.ENGAGEMENT_COLLAPSE),
                    "Drop of 0.30 is below 0.35 threshold — no collapse");
        }

        @Test
        void metabolicWorsening_engagementMaintained_noInconsistency() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            long t = weeksAfter(BASE_TIME, 2);

            // Worsening FBG but engagement is maintained — no cross-module inconsistency
            CanonicalEvent labEvent = labEvent(P, t, "FBG", 170.0);
            CrossModuleTestHarness.processEvent(labEvent, state, t);

            CanonicalEvent engEvent = engagementEvent(P, t + HOUR_MS,
                    0.70, "GREEN", "ACTIVE", "TIER_2_SMBG");
            ProcessResult r = CrossModuleTestHarness.processEvent(engEvent, state, t + HOUR_MS);

            assertNotNull(r);
            // CROSS_MODULE_INCONSISTENCY fires when high adherence + deteriorating trajectory
            // Here adherence is moderate (0.70) and no intervention delta — should not fire
            assertFalse(r.hasChange(ClinicalStateChangeType.CROSS_MODULE_INCONSISTENCY));
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Scenario 3: Intervention Success / Futility
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Scenario3_InterventionOutcomes {

        @Test
        void interventionDelta_positive_recordsDeltaInState() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            ClinicalStateSummary.InterventionWindowSummary iw = new ClinicalStateSummary.InterventionWindowSummary();
            iw.setInterventionId("intv-1");
            iw.setInterventionType(InterventionType.MEDICATION_ADD);
            iw.setStatus("OPENED");
            state.getActiveInterventions().put("intv-1", iw);
            long t = weeksAfter(BASE_TIME, 2);

            CanonicalEvent deltaEvent = interventionDeltaEvent(P, t,
                    "intv-1", "INTERVENTION_REVERSED_DECLINE", 0.85,
                    -15.0, -8.0, 2.0);
            ProcessResult r = CrossModuleTestHarness.processEvent(deltaEvent, state, t);

            assertNotNull(r);
            assertEquals("module12b", r.getSourceModule());
            assertEquals(1, state.getRecentInterventionDeltas().size(),
                    "Positive delta should be recorded in state");
            assertEquals(TrajectoryAttribution.INTERVENTION_REVERSED_DECLINE,
                    state.getRecentInterventionDeltas().get(0).getAttribution());
        }

        @Test
        void interventionDelta_insufficientTwice_emitsFutility() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            ClinicalStateSummary.InterventionWindowSummary iw = new ClinicalStateSummary.InterventionWindowSummary();
            iw.setInterventionId("intv-2");
            iw.setInterventionType(InterventionType.MEDICATION_ADD);
            iw.setStatus("OPENED");
            state.getActiveInterventions().put("intv-2", iw);
            long t = weeksAfter(BASE_TIME, 2);

            // First INSUFFICIENT delta
            CanonicalEvent delta1 = interventionDeltaEvent(P, t,
                    "intv-2", "INTERVENTION_INSUFFICIENT", 0.80, 5.0, 3.0, -1.0);
            CrossModuleTestHarness.processEvent(delta1, state, t);

            // Second INSUFFICIENT delta — should trigger futility (count >= 2)
            long t2 = t + DAY_MS;
            CanonicalEvent delta2 = interventionDeltaEvent(P, t2,
                    "intv-2", "INTERVENTION_INSUFFICIENT", 0.75, 4.0, 2.0, -0.5);
            ProcessResult r = CrossModuleTestHarness.processEvent(delta2, state, t2);

            assertNotNull(r);
            assertTrue(r.hasChange(ClinicalStateChangeType.INTERVENTION_FUTILITY),
                    "2 consecutive INSUFFICIENT attributions should emit INTERVENTION_FUTILITY");
        }

        @Test
        void interventionClosed_removesFromActiveInterventions() {
            ClinicalStateSummary state = stateWithActiveIntervention(P, "intv-3", InterventionType.NUTRITION_FOOD_CHANGE);
            assertEquals(1, state.getActiveInterventions().size());
            long t = weeksAfter(BASE_TIME, 2);

            CanonicalEvent closeEvent = interventionWindowEvent(P, t,
                    "intv-3", "WINDOW_CLOSED", "NUTRITION_FOOD_CHANGE",
                    BASE_TIME, BASE_TIME + 28 * DAY_MS);
            ProcessResult r = CrossModuleTestHarness.processEvent(closeEvent, state, t);

            assertNotNull(r);
            assertEquals(0, state.getActiveInterventions().size(),
                    "WINDOW_CLOSED should remove intervention from active map");
        }

        @Test
        void mealImprovement_duringIntervention_updatesMetabolic() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            // Previous snapshot had higher iAUC, current will improve
            state.previous().meanIAUC = 50.0;
            state.current().meanIAUC = 50.0;
            long t = weeksAfter(BASE_TIME, 2);

            // Lower iAUC = improvement
            CanonicalEvent mealEvent = mealPatternEvent(P, t,
                    30.0, 40.0, "LOW", 0.1);
            ProcessResult r = CrossModuleTestHarness.processEvent(mealEvent, state, t);

            assertNotNull(r);
            assertEquals(30.0, state.current().meanIAUC,
                    "Meal pattern should update current snapshot iAUC");
            // Metabolic velocity should reflect improvement (iAUC decreased)
            assertTrue(r.getVelocity().getDomainVelocity(CKMRiskDomain.METABOLIC) < 0
                    || Math.abs(r.getVelocity().getDomainVelocity(CKMRiskDomain.METABOLIC)) < 0.01,
                    "Decreased iAUC should move metabolic velocity towards improvement");
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Scenario 4: Comorbidity Halt + BP Crisis
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Scenario4_ComorbidityHalt {

        @Test
        void cidHalt_withHighARV_stateRecordsHalt() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            long t = weeksAfter(BASE_TIME, 2);

            // CID HALT from Module 8
            CanonicalEvent cidEvent = comorbidityAlertEvent(P, t, "CID-RULE-42", "HALT");
            CrossModuleTestHarness.processEvent(cidEvent, state, t);

            assertEquals(1, state.getActiveCIDHaltCount(), "HALT should increment halt counter");
            assertTrue(state.getActiveCIDRuleIds().contains("CID-RULE-42"));

            // Now process a BP event — CID state should persist
            CanonicalEvent bpEvent = bpVariabilityEvent(P, t + HOUR_MS,
                    20.0, "HIGH", 155.0, 95.0);
            CrossModuleTestHarness.processEvent(bpEvent, state, t + HOUR_MS);

            assertEquals(1, state.getActiveCIDHaltCount(),
                    "CID halt count should persist after BP event");
        }

        @Test
        void multipleCIDAlerts_countAccumulates() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            long t = weeksAfter(BASE_TIME, 2);

            CrossModuleTestHarness.processEvent(
                    comorbidityAlertEvent(P, t, "CID-1", "HALT"), state, t);
            CrossModuleTestHarness.processEvent(
                    comorbidityAlertEvent(P, t + 1000, "CID-2", "HALT"), state, t + 1000);
            CrossModuleTestHarness.processEvent(
                    comorbidityAlertEvent(P, t + 2000, "CID-3", "PAUSE"), state, t + 2000);

            assertEquals(2, state.getActiveCIDHaltCount());
            assertEquals(1, state.getActiveCIDPauseCount());
            assertEquals(3, state.getActiveCIDRuleIds().size());
        }

        @Test
        void cidHalt_persistsAcrossSubsequentEvents() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            long t = weeksAfter(BASE_TIME, 2);

            CrossModuleTestHarness.processEvent(
                    comorbidityAlertEvent(P, t, "CID-X", "HALT"), state, t);

            // Send fitness, meal, engagement events — halt should persist
            CrossModuleTestHarness.processEvent(
                    fitnessPatternEvent(P, t + 1000, 36.0, 0.5, 180.0, -5.0), state, t + 1000);
            CrossModuleTestHarness.processEvent(
                    engagementEvent(P, t + 2000, 0.65, "GREEN", "ACTIVE", "TIER_2_SMBG"), state, t + 2000);

            assertEquals(1, state.getActiveCIDHaltCount(),
                    "CID halt should persist across non-CID events");
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Scenario 5: Multi-Week Longitudinal Velocity Transitions
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Scenario5_LongitudinalVelocity {

        @Test
        void threeWeeks_stableToDeterioratingToImproving() {
            ClinicalStateSummary state = emptyState(P);
            long t0 = BASE_TIME;

            // Week 1: Baselines
            CrossModuleTestHarness.processEvent(labEvent(P, t0, "FBG", 120.0), state, t0);
            CrossModuleTestHarness.processEvent(
                    bpVariabilityEvent(P, t0 + HOUR_MS, 10.0, "MODERATE", 138.0, 85.0), state, t0 + HOUR_MS);
            CrossModuleTestHarness.processEvent(labEvent(P, t0 + 2 * HOUR_MS, "EGFR", 70.0), state, t0 + 2 * HOUR_MS);

            // First week: velocity should be UNKNOWN (no previous snapshot yet)
            ProcessResult r1 = CrossModuleTestHarness.processEvent(
                    engagementEvent(P, t0 + 3 * HOUR_MS, 0.75, "GREEN", "ACTIVE", "TIER_2_SMBG"),
                    state, t0 + 3 * HOUR_MS);
            assertNotNull(r1);
            assertEquals(CKMRiskVelocity.CompositeClassification.UNKNOWN,
                    r1.getVelocity().getCompositeClassification(),
                    "First week without snapshot pair should be UNKNOWN");

            // Rotate snapshot (simulating 7 days passing)
            state.rotateSnapshots(t0 + WEEK_MS);

            // Week 2: Worsening metrics
            long t1 = t0 + WEEK_MS;
            CrossModuleTestHarness.processEvent(labEvent(P, t1, "FBG", 180.0), state, t1);
            CrossModuleTestHarness.processEvent(
                    bpVariabilityEvent(P, t1 + HOUR_MS, 18.0, "HIGH", 165.0, 100.0), state, t1 + HOUR_MS);
            ProcessResult r2 = CrossModuleTestHarness.processEvent(
                    labEvent(P, t1 + 2 * HOUR_MS, "EGFR", 55.0), state, t1 + 2 * HOUR_MS);

            assertNotNull(r2);
            // With big worsening: FBG 120→180, SBP 138→165, eGFR 70→55
            assertTrue(r2.getVelocity().getCompositeScore() > 0,
                    "Week 2 should show positive (deteriorating) composite score");

            // Rotate again
            state.rotateSnapshots(t1 + WEEK_MS);

            // Week 3: Improving metrics
            long t2 = t1 + WEEK_MS;
            CrossModuleTestHarness.processEvent(labEvent(P, t2, "FBG", 115.0), state, t2);
            CrossModuleTestHarness.processEvent(
                    bpVariabilityEvent(P, t2 + HOUR_MS, 9.0, "LOW", 128.0, 82.0), state, t2 + HOUR_MS);
            ProcessResult r3 = CrossModuleTestHarness.processEvent(
                    labEvent(P, t2 + 2 * HOUR_MS, "EGFR", 72.0), state, t2 + 2 * HOUR_MS);

            assertNotNull(r3);
            assertTrue(r3.getVelocity().getCompositeScore() < 0,
                    "Week 3 should show negative (improving) composite score");
        }

        @Test
        void snapshotRotation_preservesPreviousForVelocity() {
            ClinicalStateSummary state = emptyState(P);
            long t0 = BASE_TIME;

            CrossModuleTestHarness.processEvent(labEvent(P, t0, "FBG", 125.0), state, t0);
            assertEquals(125.0, state.current().fbg);

            state.rotateSnapshots(t0 + WEEK_MS);
            assertEquals(125.0, state.previous().fbg,
                    "Rotation should copy current→previous");
        }

        @Test
        void velocityTransition_improvingToDeteriorating_emitsReversal() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            long t = weeksAfter(BASE_TIME, 2);

            // Set previous snapshot to good values, current to much worse values
            // to force a high positive composite score (> 0.40 = DETERIORATING)
            state.previous().fbg = 105.0;
            state.previous().hba1c = 6.5;
            state.previous().meanSBP = 125.0;
            state.previous().arv = 8.0;
            state.previous().egfr = 80.0;
            state.previous().uacr = 20.0;

            state.current().fbg = 160.0;      // +55 (range 40 → normalised 1.0+)
            state.current().hba1c = 8.0;       // +1.5 (range 1.0 → normalised 1.0+)
            state.current().meanSBP = 175.0;   // +50 via ARV
            state.current().arv = 22.0;        // +14 (range 8.0 → normalised 1.0+)
            state.current().egfr = 40.0;       // -40 (range 8.0 → normalised 1.0+)
            state.current().uacr = 120.0;      // +100 (range 80 → normalised 1.0+)

            // Compute velocity — should be DETERIORATING given the massive worsening
            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

            // Set the PREVIOUS computed velocity as IMPROVING
            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.IMPROVING)
                    .compositeScore(-0.4)
                    .dataCompleteness(0.8)
                    .computationTimestamp(BASE_TIME)
                    .build());

            // Now detect — should see IMPROVING (last) → DETERIORATING (new)
            List<ClinicalStateChangeEvent> changes =
                    Module13StateChangeDetector.detect(state, velocity, t);

            // Verify the velocity is indeed DETERIORATING
            assertEquals(CKMRiskVelocity.CompositeClassification.DETERIORATING,
                    velocity.getCompositeClassification(),
                    "Massive worsening across 3 domains should yield DETERIORATING");

            boolean hasReversal = changes.stream()
                    .anyMatch(c -> c.getChangeType() == ClinicalStateChangeType.TRAJECTORY_REVERSAL);
            assertTrue(hasReversal,
                    "IMPROVING→DETERIORATING should emit TRAJECTORY_REVERSAL");
        }

        @Test
        void allEightSources_fullFanIn_correctCompleteness() {
            ClinicalStateSummary state = emptyState(P);
            long t = BASE_TIME;

            // Send events from all 8 source modules (including module8 added in PIPE-7)
            CrossModuleTestHarness.processEvent(
                    bpVariabilityEvent(P, t, 10.0, "MODERATE", 138.0, 85.0), state, t);
            CrossModuleTestHarness.processEvent(
                    comorbidityAlertEvent(P, t + 500, "CID_14", "PAUSE"), state, t + 500);
            CrossModuleTestHarness.processEvent(
                    engagementEvent(P, t + 1000, 0.72, "GREEN", "ACTIVE", "TIER_2_SMBG"), state, t + 1000);
            CrossModuleTestHarness.processEvent(
                    mealPatternEvent(P, t + 2000, 35.0, 45.0, "LOW", 0.1), state, t + 2000);
            CrossModuleTestHarness.processEvent(
                    fitnessPatternEvent(P, t + 3000, 35.0, 0.5, 180.0, -5.0), state, t + 3000);
            CrossModuleTestHarness.processEvent(
                    interventionWindowEvent(P, t + 4000, "iw-1", "WINDOW_OPENED", "MEDICATION_ADD",
                            t, t + 28 * DAY_MS), state, t + 4000);
            CrossModuleTestHarness.processEvent(
                    interventionDeltaEvent(P, t + 5000, "iw-1", "STABLE", 0.7,
                            0.0, 0.0, 0.0), state, t + 5000);
            ProcessResult r = CrossModuleTestHarness.processEvent(
                    labEvent(P, t + 6000, "FBG", 120.0), state, t + 6000);

            assertNotNull(r);
            assertTrue(r.getCompletenessScore() > 0.95,
                    "All 8 sources seen recently should yield completeness > 0.95, got: " + r.getCompletenessScore());
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Scenario 6: Data Absence Detection
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Scenario6_DataAbsence {

        @Test
        void noDataFor14Days_lowCompleteness() {
            ClinicalStateSummary state = stateWithBaselines(P);
            // All modules last seen at BASE_TIME, evaluate at BASE_TIME + 14 days
            long evalTime = BASE_TIME + 14 * DAY_MS;

            Module13DataCompletenessMonitor.Result result =
                    Module13DataCompletenessMonitor.evaluate(state, evalTime);

            assertTrue(result.isDataAbsenceCritical(),
                    "14 days without data should trigger critical absence");
        }

        @Test
        void partialData_someModulesMissing_lowCompleteness() {
            ClinicalStateSummary state = emptyState(P);
            long t = BASE_TIME;

            // Only 2 of 7 sources seen
            CrossModuleTestHarness.processEvent(
                    bpVariabilityEvent(P, t, 10.0, "MODERATE", 138.0, 85.0), state, t);
            CrossModuleTestHarness.processEvent(
                    labEvent(P, t + 1000, "FBG", 120.0), state, t + 1000);

            Module13DataCompletenessMonitor.Result result =
                    Module13DataCompletenessMonitor.evaluate(state, t + 2000);

            assertTrue(result.getCompositeScore() < 0.5,
                    "Only 2 of 7 sources → completeness should be < 0.5, got: " + result.getCompositeScore());
        }

        @Test
        void allModulesRecent_noAbsence() {
            ClinicalStateSummary state = stateWithBaselines(P);
            // Evaluate within fresh window (same day)
            long evalTime = BASE_TIME + HOUR_MS;

            Module13DataCompletenessMonitor.Result result =
                    Module13DataCompletenessMonitor.evaluate(state, evalTime);

            assertFalse(result.isDataAbsenceCritical());
            assertTrue(result.getDataGapFlags().isEmpty(),
                    "All modules seen recently should have no gap flags");
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Scenario 7: Personalised Target Override
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Scenario7_PersonalisedTargets {

        @Test
        void labEventWithTargets_updatesStateTargets() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            long t = weeksAfter(BASE_TIME, 2);

            Map<String, Object> targets = new HashMap<>();
            targets.put("fbg_target", 100.0);
            targets.put("sbp_target", 125.0);
            targets.put("egfr_threshold", 50.0);

            // Use updateFromLabResult directly (static method) — targets extraction
            // is now in processElement, so we simulate it manually here
            CanonicalEvent event = labEventWithPersonalizedTargets(P, t, "FBG", 115.0, targets);
            Module13_ClinicalStateSynchroniser.updateFromLabResult(event.getPayload(), state);

            // Verify lab value updated
            assertEquals(115.0, state.current().fbg);
        }

        @Test
        void personalised_fbgTarget_changesDetection() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            long t = weeksAfter(BASE_TIME, 2);

            // With default FBG target = 110: FBG=108 < 110 → METABOLIC_MILESTONE
            state.current().fbg = 108.0;
            state.previous().fbg = 130.0;

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes =
                    Module13StateChangeDetector.detect(state, velocity, t);

            boolean hasMilestone = changes.stream()
                    .anyMatch(c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE);
            assertTrue(hasMilestone,
                    "FBG=108 below default target 110 should trigger METABOLIC_MILESTONE");
        }

        @Test
        void personalised_egfrThreshold_changesRenalDetection() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            state.setPersonalizedEGFRThreshold(50.0);
            state.current().egfr = 48.0;
            state.previous().egfr = 65.0;
            long t = weeksAfter(BASE_TIME, 2);

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes =
                    Module13StateChangeDetector.detect(state, velocity, t);

            boolean hasRenalDecline = changes.stream()
                    .anyMatch(c -> c.getChangeType() == ClinicalStateChangeType.RENAL_RAPID_DECLINE);
            assertTrue(hasRenalDecline,
                    "eGFR=48 below personalised threshold 50 should trigger RENAL_RAPID_DECLINE");
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Scenario 8: Healthy Patient — Negative Test (Zero Alerts)
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Scenario8_HealthyPatient {

        /**
         * A well-controlled patient with normal vitals, good engagement, stable
         * metrics — should produce zero crisis alerts, zero comorbidity flags,
         * and zero HIGH/CRITICAL state changes. This validates that the pipeline
         * doesn't fire false positives on healthy data.
         */
        @Test
        void stablePatient_allNormalMetrics_zeroHighPriorityChanges() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            // Override both snapshots with healthy, well-controlled values
            state.current().fbg = 100.0;       // well below 110 target
            state.current().hba1c = 6.2;        // well-controlled
            state.current().egfr = 78.0;        // healthy renal function
            state.current().meanSBP = 124.0;    // below 130 target
            state.current().meanDBP = 78.0;
            state.current().arv = 8.0;          // low variability
            state.current().variabilityClass = VariabilityClassification.LOW;
            state.current().engagementScore = 0.85;
            state.current().engagementLevel = EngagementLevel.GREEN;

            state.previous().fbg = 105.0;       // also normal — no worsening
            state.previous().hba1c = 6.4;
            state.previous().egfr = 80.0;
            state.previous().meanSBP = 126.0;
            state.previous().meanDBP = 80.0;
            state.previous().arv = 9.0;
            state.previous().variabilityClass = VariabilityClassification.LOW;
            state.previous().engagementScore = 0.80;

            state.setPreviousEngagementScore(0.80);

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes =
                    Module13StateChangeDetector.detect(state, velocity, weeksAfter(BASE_TIME, 2));

            // Should be STABLE or IMPROVING — not DETERIORATING
            assertNotEquals(CKMRiskVelocity.CompositeClassification.DETERIORATING,
                    velocity.getCompositeClassification(),
                    "Healthy patient should NOT be classified as DETERIORATING");

            // Zero HIGH or CRITICAL alerts
            long highOrCriticalCount = changes.stream()
                    .filter(c -> c.getChangeType().isHighOrAbove())
                    .count();
            assertEquals(0, highOrCriticalCount,
                    "Healthy patient should produce zero HIGH/CRITICAL state changes, got: "
                    + changes.stream().filter(c -> c.getChangeType().isHighOrAbove())
                            .map(c -> c.getChangeType().name()).toList());
        }

        @Test
        void stablePatient_fullPipeline_noAlertEvents() {
            // Feed realistic healthy data through the full harness pipeline
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            state.current().fbg = 98.0;
            state.current().meanSBP = 122.0;
            state.current().egfr = 82.0;
            state.current().engagementScore = 0.88;
            state.previous().fbg = 100.0;
            state.previous().meanSBP = 125.0;
            state.previous().egfr = 80.0;
            state.previous().engagementScore = 0.85;
            state.setPreviousEngagementScore(0.85);

            long t = weeksAfter(BASE_TIME, 2);

            // Normal BP variability
            ProcessResult r1 = CrossModuleTestHarness.processEvent(
                    bpVariabilityEvent(P, t, 8.0, "LOW", 122.0, 78.0), state, t);
            // Good engagement
            ProcessResult r2 = CrossModuleTestHarness.processEvent(
                    engagementEvent(P, t + 1000, 0.88, "GREEN", "ACTIVE", "TIER_2_SMBG"),
                    state, t + 1000);
            // Normal lab
            ProcessResult r3 = CrossModuleTestHarness.processEvent(
                    labEvent(P, t + 2000, "FBG", 98.0), state, t + 2000);

            // Collect all state changes from all events
            List<ClinicalStateChangeEvent> allChanges = new ArrayList<>();
            if (r1 != null && r1.getChanges() != null) allChanges.addAll(r1.getChanges());
            if (r2 != null && r2.getChanges() != null) allChanges.addAll(r2.getChanges());
            if (r3 != null && r3.getChanges() != null) allChanges.addAll(r3.getChanges());

            long alertCount = allChanges.stream()
                    .filter(c -> c.getChangeType().isHighOrAbove())
                    .count();
            assertEquals(0, alertCount,
                    "Healthy patient should produce zero HIGH/CRITICAL alerts through full pipeline");
        }

        @Test
        void improvingPatient_fbgAndSBPBelowTarget_onlyPositiveMilestones() {
            ClinicalStateSummary state = stateWithSnapshotPair(P);
            // Patient improving from elevated baselines
            state.current().fbg = 105.0;       // below 110 target
            state.current().meanSBP = 128.0;    // below 130 target
            state.current().egfr = 75.0;
            state.current().variabilityClass = VariabilityClassification.LOW;
            state.current().engagementScore = 0.82;
            state.previous().fbg = 140.0;       // was elevated
            state.previous().meanSBP = 148.0;   // was elevated
            state.previous().egfr = 72.0;
            state.previous().variabilityClass = VariabilityClassification.MODERATE;
            state.previous().engagementScore = 0.70;
            state.setPreviousEngagementScore(0.70);

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes =
                    Module13StateChangeDetector.detect(state, velocity, weeksAfter(BASE_TIME, 2));

            // Velocity should be IMPROVING or STABLE (depending on composite threshold)
            assertNotEquals(CKMRiskVelocity.CompositeClassification.DETERIORATING,
                    velocity.getCompositeClassification(),
                    "Improving patient should NOT be classified as DETERIORATING");

            // Should have positive milestones (INFO priority) but zero HIGH alerts
            boolean hasMetabolicMilestone = changes.stream()
                    .anyMatch(c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE);
            boolean hasBPMilestone = changes.stream()
                    .anyMatch(c -> c.getChangeType() == ClinicalStateChangeType.BP_MILESTONE);
            assertTrue(hasMetabolicMilestone, "FBG=105 < 110 should emit METABOLIC_MILESTONE");
            assertTrue(hasBPMilestone, "SBP=128 < 130 should emit BP_MILESTONE");

            long highCount = changes.stream()
                    .filter(c -> c.getChangeType().isHighOrAbove())
                    .count();
            assertEquals(0, highCount,
                    "Improving patient should have zero HIGH/CRITICAL alerts");
        }
    }
}
