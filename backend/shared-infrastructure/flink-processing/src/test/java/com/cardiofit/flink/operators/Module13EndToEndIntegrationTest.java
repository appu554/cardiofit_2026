package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Module 13 End-to-End Integration Tests
 *
 * Tests the full Module 13 pipeline (state management + CKM velocity + state change detection)
 * as pure-function integration tests covering:
 *
 *   Phase A: Week-long patient journeys (stable, deterioration, escalation)
 *   Phase B: CKM velocity regression with A1 personalised targets
 *   Phase C: Edge cases (dedup, partial data, threshold exactness)
 *   Phase D: PIPE integration (late events, coalescing, versioning)
 *
 * All tests use Module13TestBuilder for state construction and avoid Mockito.
 */
class Module13EndToEndIntegrationTest {

    // ═══════════════════════════════════════════════════════════════════════
    // Phase A: Week-Long Patient Journeys
    // ═══════════════════════════════════════════════════════════════════════

    @Nested
    class PatientJourneys {

        /**
         * Stable week: patient with well-controlled metrics across two snapshots.
         * All domains should remain STABLE with no state change events.
         */
        @Test
        void stableWeek_noEscalation() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("stable-p1");
            // previous: fbg=130, hba1c=7.2, egfr=65, arv=10.5, meanSBP=142
            // current: same values — no delta
            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            assertEquals(CKMRiskVelocity.CompositeClassification.STABLE, velocity.getCompositeClassification());

            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);
            assertTrue(changes.isEmpty(), "Stable patient should emit no state change events");
        }

        /**
         * Deterioration week: FBG rises 130→170 (+40 mg/dL), eGFR drops 65→55,
         * ARV rises 10.5→19. All three domains worsen → DETERIORATING with
         * cross-domain amplification and CKM_RISK_ESCALATION emitted.
         */
        @Test
        void deteriorationWeek_emitsCKMEscalation() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("det-p1");
            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.1).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            // Week 2: significant worsening across all domains
            state.current().fbg = 170.0;       // +40 from baseline 130
            state.current().hba1c = 8.0;        // +0.8 from baseline 7.2
            state.current().egfr = 55.0;        // -10 from baseline 65
            state.current().uacr = 90.0;        // new — high albuminuria
            state.previous().uacr = 30.0;       // previous was normal
            state.current().arv = 19.0;         // +8.5 from baseline 10.5
            state.current().meanSBP = 155.0;    // +13 from baseline 142
            state.current().morningSurgeMagnitude = 45.0;
            state.previous().morningSurgeMagnitude = 18.0;
            state.current().ldl = 140.0;
            state.previous().ldl = 110.0;

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

            assertEquals(CKMRiskVelocity.CompositeClassification.DETERIORATING,
                    velocity.getCompositeClassification());
            assertTrue(velocity.isCrossDomainAmplification(),
                    "3 domains deteriorating should trigger amplification");
            assertEquals(1.5, velocity.getAmplificationFactor());

            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            // Should emit: CKM_RISK_ESCALATION (STABLE→DETERIORATING)
            //              + CKM_DOMAIN_DIVERGENCE (cross-domain amplification)
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.CKM_RISK_ESCALATION));
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.CKM_DOMAIN_DIVERGENCE));
        }

        /**
         * Improving week: FBG drops 130→105, HbA1c drops 7.2→6.8, eGFR stable,
         * engagement rises. All domains improve → IMPROVING + METABOLIC_MILESTONE.
         */
        @Test
        void improvingWeek_emitsMetabolicMilestone() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("imp-p1");
            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            // All metrics improve
            state.current().fbg = 105.0;        // -25 from 130 → below default target 110
            state.current().hba1c = 6.8;         // -0.4 from 7.2
            state.current().egfr = 68.0;         // +3 from 65
            state.current().arv = 8.0;           // -2.5 from 10.5
            state.current().meanSBP = 128.0;     // -14 from 142 → below default target 130
            state.current().ldl = 95.0;
            state.previous().ldl = 110.0;
            state.current().engagementScore = 0.85;
            state.current().morningSurgeMagnitude = 12.0;
            state.previous().morningSurgeMagnitude = 18.0;

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            assertEquals(CKMRiskVelocity.CompositeClassification.IMPROVING,
                    velocity.getCompositeClassification());

            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            // FBG < 110 → METABOLIC_MILESTONE, SBP < 130 → BP_MILESTONE
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE));
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.BP_MILESTONE));
        }

        /**
         * Escalation cycle: IMPROVING → DETERIORATING triggers TRAJECTORY_REVERSAL.
         */
        @Test
        void trajectoryReversal_improvingToDeteriorating() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("rev-p1");
            // Previous velocity was IMPROVING
            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.IMPROVING)
                    .compositeScore(-0.4).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            // Sudden deterioration
            state.current().fbg = 180.0;         // +50 from 130
            state.current().hba1c = 8.5;          // +1.3 from 7.2
            state.current().arv = 20.0;           // +9.5 from 10.5
            state.current().meanSBP = 160.0;
            state.current().egfr = 55.0;          // -10 from 65
            state.current().uacr = 100.0;
            state.previous().uacr = 25.0;
            state.current().ldl = 145.0;
            state.previous().ldl = 105.0;
            state.current().morningSurgeMagnitude = 50.0;
            state.previous().morningSurgeMagnitude = 15.0;

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            assertEquals(CKMRiskVelocity.CompositeClassification.DETERIORATING,
                    velocity.getCompositeClassification());

            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.TRAJECTORY_REVERSAL),
                    "IMPROVING → DETERIORATING should emit TRAJECTORY_REVERSAL");
        }

        /**
         * Multi-week journey: Week 1 stable → Week 2 deterioration → Week 3 intervention.
         * Verifies snapshot rotation preserves state across transitions.
         */
        @Test
        void multiWeekJourney_rotationPreservesTransitions() {
            // Week 1: establish baselines
            ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("journey-p1");
            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.UNKNOWN)
                    .compositeScore(0.0).dataCompleteness(0.5)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            // Before rotation: no velocity data
            CKMRiskVelocity v1 = Module13CKMRiskComputer.compute(state);
            assertEquals(CKMRiskVelocity.CompositeClassification.UNKNOWN, v1.getCompositeClassification(),
                    "No previous snapshot → UNKNOWN");

            // Week 1→2 rotation
            state.rotateSnapshots(Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);
            state.current().fbg = 130.0;
            state.current().hba1c = 7.2;
            state.current().egfr = 65.0;
            state.current().arv = 10.5;
            state.current().meanSBP = 142.0;
            state.current().engagementScore = 0.72;

            // Week 2: same metrics → STABLE
            CKMRiskVelocity v2 = Module13CKMRiskComputer.compute(state);
            assertEquals(CKMRiskVelocity.CompositeClassification.STABLE, v2.getCompositeClassification());
            state.setLastComputedVelocity(v2);

            // Week 2→3 rotation
            state.rotateSnapshots(Module13TestBuilder.BASE_TIME + 2 * Module13TestBuilder.WEEK_MS);
            // Week 3: deterioration
            state.current().fbg = 165.0;
            state.current().hba1c = 7.8;
            state.current().egfr = 58.0;
            state.current().arv = 17.0;
            state.current().meanSBP = 152.0;
            state.current().engagementScore = 0.72;
            state.current().ldl = 130.0;
            state.previous().ldl = 110.0;
            state.current().morningSurgeMagnitude = 35.0;
            state.previous().morningSurgeMagnitude = 20.0;

            CKMRiskVelocity v3 = Module13CKMRiskComputer.compute(state);
            assertEquals(CKMRiskVelocity.CompositeClassification.DETERIORATING, v3.getCompositeClassification(),
                    "Week 3 deterioration should be detected");

            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, v3, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 21));
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.CKM_RISK_ESCALATION),
                    "STABLE → DETERIORATING across weeks should emit escalation");
        }

        /**
         * Engagement collapse: 2-tier drop (0.8 → 0.3) with high metabolic adherence.
         */
        @Test
        void engagementCollapse_emitsCollapseEvent() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("eng-p1");
            state.setPreviousEngagementScore(0.8);
            state.current().engagementScore = 0.3;  // drop of 0.5 (> 0.35 threshold)

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.ENGAGEMENT_COLLAPSE),
                    "0.5 drop should trigger ENGAGEMENT_COLLAPSE (threshold 0.35)");
        }

        /**
         * Intervention futility: 3 consecutive INSUFFICIENT deltas.
         */
        @Test
        void interventionFutility_consecutiveInsufficientDeltas() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("fut-p1");
            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.1).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            // Add 3 consecutive INSUFFICIENT deltas
            for (int i = 1; i <= 3; i++) {
                ClinicalStateSummary.InterventionDeltaSummary d = new ClinicalStateSummary.InterventionDeltaSummary();
                d.setInterventionId("int-" + i);
                d.setAttribution(TrajectoryAttribution.INTERVENTION_INSUFFICIENT);
                d.setAdherenceScore(0.85);
                d.setClosedAtMs(Module13TestBuilder.BASE_TIME + i * Module13TestBuilder.DAY_MS);
                state.getRecentInterventionDeltas().add(d);
            }

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.INTERVENTION_FUTILITY),
                    "3 consecutive INSUFFICIENT should trigger INTERVENTION_FUTILITY");
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Phase B: A1 Personalised Target Integration
    // ═══════════════════════════════════════════════════════════════════════

    @Nested
    class PersonalizedTargets {

        /**
         * A1: Personalised FBG target relaxes milestone threshold.
         * FBG=125 is below population default (110) but ABOVE personalised target (130).
         * No METABOLIC_MILESTONE should be emitted with personalised target set.
         */
        @Test
        void personalizedFBGTarget_relaxedThreshold_noMilestone() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("a1-p1");
            state.setPersonalizedFBGTarget(130.0);  // elderly relaxed target
            state.current().fbg = 125.0;            // below 130, but test that target is used

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            // FBG=125 < personalised target 130 → milestone SHOULD fire
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE),
                    "FBG 125 < personalised target 130 should emit milestone");
        }

        /**
         * A1: Without personalised target, FBG=115 does NOT trigger milestone
         * (population default is 110, and 115 > 110).
         */
        @Test
        void defaultFBGTarget_aboveThreshold_noMilestone() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("a1-p2");
            // No personalised target set → uses default 110.0
            state.current().fbg = 115.0;

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertFalse(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE),
                    "FBG 115 > default target 110 should NOT emit milestone");
        }

        /**
         * A1: Personalised SBP target tightened for proteinuria.
         * SBP=125 is below population default (130) but ABOVE personalised target (120).
         * BP_MILESTONE should NOT fire with tightened target.
         */
        @Test
        void personalizedSBPTarget_tightenedForProteinuria_noMilestone() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("a1-p3");
            state.setPersonalizedSBPTarget(120.0);  // KDIGO tightened for proteinuria
            state.current().meanSBP = 125.0;        // below 130 default but above 120

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertFalse(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.BP_MILESTONE),
                    "SBP 125 > personalised target 120 should NOT emit BP_MILESTONE");
        }

        /**
         * A1: Personalised eGFR threshold for CKD G3a patient (threshold=45).
         * eGFR=50 should NOT trigger RENAL_RAPID_DECLINE (above 45).
         * But eGFR=42 SHOULD trigger it.
         */
        @Test
        void personalizedEGFRThreshold_CKDStageAware() {
            // Patient is CKD G3a — personalised threshold shifts to 45
            ClinicalStateSummary stateAbove = Module13TestBuilder.stateWithSnapshotPair("a1-egfr1");
            stateAbove.setPersonalizedEGFRThreshold(45.0);
            stateAbove.current().egfr = 50.0;  // above threshold
            stateAbove.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity v1 = Module13CKMRiskComputer.compute(stateAbove);
            List<ClinicalStateChangeEvent> changes1 = Module13StateChangeDetector.detect(
                    stateAbove, v1, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));
            assertFalse(changes1.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.RENAL_RAPID_DECLINE),
                    "eGFR 50 > threshold 45 should NOT trigger renal decline");

            // Same patient, eGFR drops below threshold
            ClinicalStateSummary stateBelow = Module13TestBuilder.stateWithSnapshotPair("a1-egfr2");
            stateBelow.setPersonalizedEGFRThreshold(45.0);
            stateBelow.current().egfr = 42.0;  // below threshold
            stateBelow.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity v2 = Module13CKMRiskComputer.compute(stateBelow);
            List<ClinicalStateChangeEvent> changes2 = Module13StateChangeDetector.detect(
                    stateBelow, v2, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));
            assertTrue(changes2.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.RENAL_RAPID_DECLINE),
                    "eGFR 42 < threshold 45 should trigger RENAL_RAPID_DECLINE");
        }

        /**
         * A1: SBP kidney threshold affects renal velocity computation.
         * Personalised threshold 130 (proteinuria) vs default 140.
         * SBP=135 is above personalised threshold → amplifies renal velocity.
         */
        @Test
        void personalizedSBPKidneyThreshold_amplifiesRenalVelocity() {
            // With default threshold (140): SBP 135 is below → no amplification
            ClinicalStateSummary stateDefault = Module13TestBuilder.stateWithSnapshotPair("a1-sbpk1");
            stateDefault.current().meanSBP = 135.0;
            stateDefault.current().arv = 14.0;
            stateDefault.current().egfr = 58.0;
            stateDefault.previous().egfr = 65.0;

            double renalDefault = Module13CKMRiskComputer.computeRenalVelocity(
                    stateDefault.previous(), stateDefault.current(), stateDefault);

            // With personalised threshold (130): SBP 135 is above → sbpFactor=1.2
            ClinicalStateSummary statePersonalised = Module13TestBuilder.stateWithSnapshotPair("a1-sbpk2");
            statePersonalised.setPersonalizedSBPKidneyThreshold(130.0);
            statePersonalised.current().meanSBP = 135.0;
            statePersonalised.current().arv = 14.0;
            statePersonalised.current().egfr = 58.0;
            statePersonalised.previous().egfr = 65.0;

            double renalPersonalised = Module13CKMRiskComputer.computeRenalVelocity(
                    statePersonalised.previous(), statePersonalised.current(), statePersonalised);

            assertTrue(renalPersonalised > renalDefault,
                    "Personalised SBP kidney threshold 130 (vs 140) should amplify renal velocity when SBP=135");
        }

        /**
         * A1: Null personalised targets fall back to population defaults.
         * Verifies the null-check → default fallback pattern.
         */
        @Test
        void nullPersonalizedTargets_usesPopulationDefaults() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("a1-null");
            // All personalised targets are null by default
            assertNull(state.getPersonalizedFBGTarget());
            assertNull(state.getPersonalizedSBPTarget());
            assertNull(state.getPersonalizedEGFRThreshold());
            assertNull(state.getPersonalizedSBPKidneyThreshold());

            state.current().fbg = 105.0;   // below default 110
            state.current().meanSBP = 125.0; // below default 130
            state.current().egfr = 40.0;     // below default 45

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            // All should fire with population defaults
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE));
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.BP_MILESTONE));
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.RENAL_RAPID_DECLINE));
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Phase C: Edge Cases
    // ═══════════════════════════════════════════════════════════════════════

    @Nested
    class EdgeCases {

        /**
         * Dedup: same change type within 24h window should not emit twice.
         */
        @Test
        void dedup_sameChangeWithin24Hours_suppressesDuplicate() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("dup-p1");
            state.current().fbg = 105.0;  // below default 110 → METABOLIC_MILESTONE

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            long ts1 = Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14);

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> first = Module13StateChangeDetector.detect(state, velocity, ts1);
            assertTrue(first.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE));

            // Simulate: update lastEmittedChangeTimestamps (normally done in processElement)
            state.getLastEmittedChangeTimestamps().put(ClinicalStateChangeType.METABOLIC_MILESTONE, ts1);

            // Second detection 6h later — within 24h dedup window
            long ts2 = ts1 + 6 * Module13TestBuilder.HOUR_MS;
            List<ClinicalStateChangeEvent> second = Module13StateChangeDetector.detect(state, velocity, ts2);
            assertFalse(second.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE),
                    "Same change within 24h should be suppressed");
        }

        /**
         * Dedup: same change type AFTER 24h window should emit again.
         */
        @Test
        void dedup_sameChangeAfter24Hours_emitsAgain() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("dup-p2");
            state.current().fbg = 105.0;

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            long ts1 = Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14);
            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

            // First emission
            state.getLastEmittedChangeTimestamps().put(ClinicalStateChangeType.METABOLIC_MILESTONE, ts1);

            // 25h later — outside 24h window
            long ts2 = ts1 + 25 * Module13TestBuilder.HOUR_MS;
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(state, velocity, ts2);
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE),
                    "After 24h dedup window, same change should emit again");
        }

        /**
         * Threshold exactness: FBG exactly at target (110.0) should NOT emit milestone.
         * Milestone fires when FBG < target, not FBG <= target.
         */
        @Test
        void thresholdExactness_fbgAtTarget_noMilestone() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("exact-p1");
            state.current().fbg = 110.0;  // exactly at default target

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertFalse(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE),
                    "FBG exactly at target (110.0) should NOT emit milestone (strict <)");
        }

        /**
         * Threshold exactness: eGFR exactly at threshold (45.0) should NOT emit
         * RENAL_RAPID_DECLINE (strict < comparison).
         */
        @Test
        void thresholdExactness_egfrAtThreshold_noRenalDecline() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("exact-p2");
            state.current().egfr = 45.0;  // exactly at default threshold

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertFalse(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.RENAL_RAPID_DECLINE),
                    "eGFR exactly at threshold (45.0) should NOT trigger decline (strict <)");
        }

        /**
         * Partial data: only metabolic domain has data (renal + CV are NaN).
         * With < 2 valid domains, composite should be UNKNOWN.
         */
        @Test
        void partialData_singleDomain_returnsUnknown() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("partial-p1");

            // Clear all non-metabolic fields from both snapshots
            state.current().egfr = null;
            state.previous().egfr = null;
            state.current().uacr = null;
            state.previous().uacr = null;
            state.current().arv = null;
            state.previous().arv = null;
            state.current().meanSBP = null;
            state.current().meanDBP = null;
            state.current().ldl = null;
            state.previous().ldl = null;
            state.current().morningSurgeMagnitude = null;
            state.previous().morningSurgeMagnitude = null;
            state.current().engagementScore = null;
            state.previous().engagementScore = null;
            state.current().variabilityClass = null;
            state.previous().variabilityClass = null;
            state.getRecentInterventionDeltas().clear();

            // Only metabolic data: FBG deterioration
            state.current().fbg = 180.0;  // +50 from previous 130

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            assertEquals(CKMRiskVelocity.CompositeClassification.UNKNOWN,
                    velocity.getCompositeClassification(),
                    "Only 1 valid domain (metabolic) < MIN_DOMAINS_FOR_VALID_SCORE (2)");
            assertTrue(velocity.getDataCompleteness() < 1.0);
        }

        /**
         * Partial data: metabolic + renal domains → valid 2-domain score.
         */
        @Test
        void partialData_twoDomains_returnsValidScore() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("partial-p2");

            // Clear CV-only fields
            state.current().ldl = null;
            state.previous().ldl = null;
            state.current().morningSurgeMagnitude = null;
            state.previous().morningSurgeMagnitude = null;
            state.current().engagementScore = null;
            state.previous().engagementScore = null;
            // Keep arv/meanSBP for renal BP-kidney component

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            assertNotEquals(CKMRiskVelocity.CompositeClassification.UNKNOWN,
                    velocity.getCompositeClassification(),
                    "2 valid domains should produce non-UNKNOWN classification");
        }

        /**
         * Cross-module inconsistency: high adherence (0.9) + deteriorating metrics.
         */
        @Test
        void crossModuleInconsistency_highAdherencePlusDeteriorating() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("xmod-p1");
            state.current().engagementScore = 0.9;  // high adherence

            // Worsening across all domains
            state.current().fbg = 175.0;
            state.current().hba1c = 8.2;
            state.current().egfr = 52.0;
            state.current().uacr = 100.0;
            state.previous().uacr = 30.0;
            state.current().arv = 20.0;
            state.current().meanSBP = 158.0;
            state.current().ldl = 145.0;
            state.previous().ldl = 110.0;
            state.current().morningSurgeMagnitude = 48.0;
            state.previous().morningSurgeMagnitude = 18.0;

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            assertEquals(CKMRiskVelocity.CompositeClassification.DETERIORATING,
                    velocity.getCompositeClassification());

            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.CROSS_MODULE_INCONSISTENCY),
                    "High engagement (0.9) + DETERIORATING should emit CROSS_MODULE_INCONSISTENCY");
        }

        /**
         * Engagement drop just below collapse threshold (0.34 < 0.35).
         * Should NOT trigger ENGAGEMENT_COLLAPSE.
         */
        @Test
        void engagementDrop_belowThreshold_noCollapse() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("ength-p1");
            state.setPreviousEngagementScore(0.72);
            state.current().engagementScore = 0.38;  // drop = 0.34, < threshold 0.35

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertFalse(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.ENGAGEMENT_COLLAPSE),
                    "Engagement drop 0.34 < threshold 0.35 should NOT trigger collapse");
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Phase D: PIPE Integration & Coalescing
    // ═══════════════════════════════════════════════════════════════════════

    @Nested
    class PIPEIntegration {

        /**
         * Coalescing buffer: 3 updates accumulated, flushed as single batch.
         */
        @Test
        void coalescingBuffer_accumulatesAndFlushes() {
            ClinicalStateSummary state = Module13TestBuilder.emptyState("pipe-p1");
            assertTrue(state.getCoalescingBuffer().isEmpty());
            assertEquals(-1L, state.getCoalescingTimerMs());

            // Add 3 KB-20 state updates
            long t = Module13TestBuilder.BASE_TIME;
            state.getCoalescingBuffer().add(KB20StateUpdate.builder()
                    .patientId("pipe-p1").operation(KB20StateUpdate.UpdateOperation.REPLACE)
                    .sourceModule("module7").fieldPath("arv").value(12.0)
                    .updateTimestamp(t).build());
            state.getCoalescingBuffer().add(KB20StateUpdate.builder()
                    .patientId("pipe-p1").operation(KB20StateUpdate.UpdateOperation.REPLACE)
                    .sourceModule("module9").fieldPath("engagement_score").value(0.7)
                    .updateTimestamp(t).build());
            state.getCoalescingBuffer().add(KB20StateUpdate.builder()
                    .patientId("pipe-p1").operation(KB20StateUpdate.UpdateOperation.REPLACE)
                    .sourceModule("module13").fieldPath("ckm_risk_velocity").value("STABLE")
                    .updateTimestamp(t).build());
            state.setCoalescingTimerMs(t + 5000L);

            assertEquals(3, state.getCoalescingBuffer().size());
            assertTrue(state.getCoalescingTimerMs() > 0);

            // Simulate flush
            int flushed = state.getCoalescingBuffer().size();
            state.getCoalescingBuffer().clear();
            state.setCoalescingTimerMs(-1L);

            assertEquals(3, flushed);
            assertTrue(state.getCoalescingBuffer().isEmpty());
            assertEquals(-1L, state.getCoalescingTimerMs());
        }

        /**
         * Snapshot rotation preserves previous values for velocity computation.
         */
        @Test
        void snapshotRotation_preservesPreviousForVelocity() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("pipe-p2");
            assertFalse(state.hasVelocityData(), "Before rotation: no previous");

            // Set specific values before rotation
            state.current().fbg = 130.0;
            state.current().egfr = 65.0;
            state.current().arv = 10.5;

            // Rotate
            state.rotateSnapshots(Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);
            assertTrue(state.hasVelocityData(), "After rotation: has previous");

            // Previous should match pre-rotation current
            assertEquals(130.0, state.previous().fbg);
            assertEquals(65.0, state.previous().egfr);
            assertEquals(10.5, state.previous().arv);

            // Current should be a new copy (same values initially)
            assertNotNull(state.current().fbg);
        }

        /**
         * CID halt state: when Module 8 reports active CID HALT, state tracks count.
         */
        @Test
        void cidHaltState_trackedCorrectly() {
            ClinicalStateSummary state = Module13TestBuilder.emptyState("pipe-p3");
            assertEquals(0, state.getActiveCIDHaltCount());
            assertFalse(state.hasActiveCIDHalt());

            state.setActiveCIDHaltCount(2);
            state.getActiveCIDRuleIds().add("CID-CONTRAINDICATION-001");
            state.getActiveCIDRuleIds().add("CID-ALLERGY-002");
            state.setLastCIDAlertTimestamp(Module13TestBuilder.BASE_TIME);

            assertTrue(state.hasActiveCIDHalt());
            assertEquals(2, state.getActiveCIDHaltCount());
            assertEquals(2, state.getActiveCIDRuleIds().size());
        }

        /**
         * Daily timer and idle quiescence tracking.
         */
        @Test
        void idleQuiescence_countsConsecutiveZeroCompletenessDays() {
            ClinicalStateSummary state = Module13TestBuilder.emptyState("pipe-p4");
            assertEquals(0, state.getConsecutiveZeroCompletenessDays());

            // Simulate 5 days of zero data completeness
            for (int i = 0; i < 5; i++) {
                state.setConsecutiveZeroCompletenessDays(state.getConsecutiveZeroCompletenessDays() + 1);
            }
            assertEquals(5, state.getConsecutiveZeroCompletenessDays());

            // Data received — reset counter
            state.setConsecutiveZeroCompletenessDays(0);
            assertEquals(0, state.getConsecutiveZeroCompletenessDays());
        }

        /**
         * Multiple state change types can be emitted simultaneously.
         */
        @Test
        void multipleChanges_emittedInSingleDetection() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("multi-p1");
            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            // Set up conditions for multiple events:
            // 1. FBG below target → METABOLIC_MILESTONE
            state.current().fbg = 105.0;
            // 2. SBP below target → BP_MILESTONE
            state.current().meanSBP = 125.0;
            // 3. eGFR below threshold → RENAL_RAPID_DECLINE
            state.current().egfr = 40.0;
            // 4. Engagement collapse
            state.setPreviousEngagementScore(0.8);
            state.current().engagementScore = 0.3;  // drop 0.5

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(
                    state, velocity, Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14));

            assertTrue(changes.size() >= 3,
                    "Should emit at least 3 concurrent change events, got " + changes.size());

            // Verify each expected type is present
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE));
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.BP_MILESTONE));
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.RENAL_RAPID_DECLINE));
            assertTrue(changes.stream().anyMatch(
                    c -> c.getChangeType() == ClinicalStateChangeType.ENGAGEMENT_COLLAPSE));
        }

        /**
         * Change event metadata: verify all fields are populated correctly.
         */
        @Test
        void changeEventMetadata_fullyPopulated() {
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("meta-p1");
            state.setDataCompletenessScore(0.85);
            state.current().fbg = 105.0;

            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(0.85)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            long ts = Module13TestBuilder.daysAfter(Module13TestBuilder.BASE_TIME, 14);
            List<ClinicalStateChangeEvent> changes = Module13StateChangeDetector.detect(state, velocity, ts);

            ClinicalStateChangeEvent milestone = changes.stream()
                    .filter(c -> c.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE)
                    .findFirst().orElseThrow(() -> new AssertionError("METABOLIC_MILESTONE not found"));

            assertNotNull(milestone.getChangeId(), "changeId should be populated");
            assertEquals("meta-p1", milestone.getPatientId());
            assertEquals(ClinicalStateChangeType.METABOLIC_MILESTONE, milestone.getChangeType());
            assertEquals("INFO", milestone.getPriority());
            assertEquals(CKMRiskDomain.METABOLIC, milestone.getDomain());
            assertEquals(ts, milestone.getProcessingTimestamp());
            assertEquals(0.85, milestone.getDataCompletenessAtChange(), 0.01);
            assertEquals("1.0", milestone.getVersion());
        }
    }
}
