package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class Module13MultiModuleFusionIntegrationTest {

    // --- Test 1: Simultaneous events from 3 modules → fused state → correct velocity ---
    @Test
    void threeModulesFiring_producesCorrectCompositeVelocity() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");

        state.current().arv = 16.0;
        state.current().variabilityClass = VariabilityClassification.HIGH;
        state.recordModuleSeen("module7", Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        state.setPreviousEngagementScore(state.current().engagementScore);
        state.current().engagementScore = 0.4;
        state.current().engagementLevel = EngagementLevel.ORANGE;
        state.recordModuleSeen("module9", Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        state.current().egfr = 58.0;
        state.recordModuleSeen("enriched", Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

        assertNotNull(velocity);
        assertNotEquals(CKMRiskVelocity.CompositeClassification.UNKNOWN, velocity.getCompositeClassification());
        assertTrue(velocity.getDataCompleteness() > 0.5);
    }

    // --- Test 2: Engagement drop + trajectory deterioration → CROSS_MODULE emitted ---
    @Test
    void engagementDropWithDeteriorating_emitsCrossModuleChange() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithVelocity("p1",
                CKMRiskVelocity.CompositeClassification.STABLE);
        state.current().engagementScore = 0.85;
        state.setPreviousEngagementScore(0.85);

        ClinicalStateSummary.InterventionDeltaSummary d = new ClinicalStateSummary.InterventionDeltaSummary();
        d.setAdherenceScore(0.9);
        d.setAttribution(TrajectoryAttribution.INTERVENTION_INSUFFICIENT);
        state.getRecentInterventionDeltas().add(d);

        CKMRiskVelocity badVelocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, 0.6)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.DETERIORATING)
                .compositeScore(0.6)
                .dataCompleteness(0.9)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, badVelocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CROSS_MODULE_INCONSISTENCY));
    }

    // --- Test 3: Full cycle: baselines → events → velocity → state change ---
    @Test
    void fullCycle_baselinesEventsVelocityStateChange() {
        // Use stateWithSnapshotPair to ensure hasVelocityData() returns true
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        // Set previous velocity as STABLE so CKM escalation can be detected
        state.setLastComputedVelocity(CKMRiskVelocity.builder()
                .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                .compositeScore(0.1)
                .dataCompleteness(0.85)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build());

        // Deteriorating renal: egfr drops from 65→43 (below 45 threshold), uacr rises
        state.current().egfr = 43.0;
        state.current().uacr = 180.0;
        state.current().arv = 14.0;

        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CKM_RISK_ESCALATION
                        || e.getChangeType() == ClinicalStateChangeType.RENAL_RAPID_DECLINE));
    }

    // --- Test 4: Milestone emission when FBG improves below target ---
    @Test
    void metabolicMilestone_fbgBelowTarget_withVelocityImproving() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("p1");
        state.current().fbg = 105.0;
        state.current().hba1c = 6.5;
        state.current().egfr = 70.0;
        // Ensure CV domain is non-positive so IMPROVING classification is reached
        // (requires all domain velocities <= 0). Set meanSBP below Stage 1 threshold
        // and arv below elevated threshold to zero out absolute BP severity.
        state.current().meanSBP = 130.0;
        state.previous().meanSBP = 142.0;

        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

        assertEquals(CKMRiskVelocity.CompositeClassification.IMPROVING,
                velocity.getCompositeClassification());

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE));
    }
}
