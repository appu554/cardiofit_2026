package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;

import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class Module13StateChangeDetectorTest {

    @Test
    void detect_stableToDeteriorating_emitsCKMRiskEscalation() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithVelocity("p1",
                CKMRiskVelocity.CompositeClassification.STABLE);

        CKMRiskVelocity newVelocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.RENAL, 0.6)
                .domainVelocity(CKMRiskDomain.METABOLIC, 0.1)
                .domainVelocity(CKMRiskDomain.CARDIOVASCULAR, 0.1)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.DETERIORATING)
                .compositeScore(0.6)
                .dataCompleteness(0.85)
                .computationTimestamp(Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, newVelocity, Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CKM_RISK_ESCALATION));
    }

    @Test
    void detect_engagementCollapse_2TierDrop() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.current().engagementLevel = EngagementLevel.GREEN;
        state.setPreviousEngagementScore(0.8);
        state.current().engagementScore = 0.2;

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                .dataCompleteness(0.8)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.ENGAGEMENT_COLLAPSE));
    }

    @Test
    void detect_interventionFutility_2ConsecutiveInsufficient() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        ClinicalStateSummary.InterventionDeltaSummary d1 = new ClinicalStateSummary.InterventionDeltaSummary();
        d1.setInterventionId("i1");
        d1.setAttribution(TrajectoryAttribution.INTERVENTION_INSUFFICIENT);
        d1.setClosedAtMs(Module13TestBuilder.BASE_TIME);
        ClinicalStateSummary.InterventionDeltaSummary d2 = new ClinicalStateSummary.InterventionDeltaSummary();
        d2.setInterventionId("i2");
        d2.setAttribution(TrajectoryAttribution.INTERVENTION_INSUFFICIENT);
        d2.setClosedAtMs(Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);
        state.getRecentInterventionDeltas().add(d1);
        state.getRecentInterventionDeltas().add(d2);

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                .dataCompleteness(0.8)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.INTERVENTION_FUTILITY));
    }

    @Test
    void detect_crossModuleInconsistency_highAdherenceButDeteriorating() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.current().engagementScore = 0.85;
        state.current().engagementLevel = EngagementLevel.GREEN;
        ClinicalStateSummary.InterventionDeltaSummary d = new ClinicalStateSummary.InterventionDeltaSummary();
        d.setAdherenceScore(0.9);
        d.setAttribution(TrajectoryAttribution.INTERVENTION_INSUFFICIENT);
        d.setClosedAtMs(Module13TestBuilder.BASE_TIME);
        state.getRecentInterventionDeltas().add(d);

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, 0.5)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.DETERIORATING)
                .compositeScore(0.5)
                .dataCompleteness(0.9)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CROSS_MODULE_INCONSISTENCY));
    }

    @Test
    void detect_metabolicMilestone_fbgBelowTarget() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.current().fbg = 105.0;

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, -0.4)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.IMPROVING)
                .compositeScore(-0.3)
                .dataCompleteness(0.8)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE));
    }

    @Test
    void detect_highPriorityGated_belowConfidenceThreshold() {
        // CKM_RISK_ESCALATION is HIGH priority. At 0.375 confidence (3/8 modules),
        // it should be suppressed — not enough signal for "urgent review within 4h".
        ClinicalStateSummary state = Module13TestBuilder.stateWithVelocity("p1",
                CKMRiskVelocity.CompositeClassification.STABLE);
        state.setDataCompletenessScore(0.375); // below 0.50 threshold

        CKMRiskVelocity newVelocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.RENAL, 0.6)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.DETERIORATING)
                .compositeScore(0.6)
                .dataCompleteness(0.375)
                .computationTimestamp(Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, newVelocity, Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        assertTrue(events.stream().noneMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CKM_RISK_ESCALATION),
                "HIGH-priority CKM_RISK_ESCALATION should be suppressed at 0.375 confidence");
    }

    @Test
    void detect_criticalBypassesConfidenceGate() {
        // RENAL_RAPID_DECLINE is CRITICAL priority. It should fire even at low confidence
        // because patient safety trumps data quality.
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.setDataCompletenessScore(0.25); // very low confidence
        state.current().egfr = 38.0; // below 45.0 threshold

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                .dataCompleteness(0.25)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.RENAL_RAPID_DECLINE),
                "CRITICAL RENAL_RAPID_DECLINE should bypass confidence gate");
    }

    @Test
    void detect_highPriorityEmitted_aboveConfidenceThreshold() {
        // CKM_RISK_ESCALATION should fire at 0.625 confidence (5/8 modules)
        ClinicalStateSummary state = Module13TestBuilder.stateWithVelocity("p1",
                CKMRiskVelocity.CompositeClassification.STABLE);
        state.setDataCompletenessScore(0.625); // above 0.50 threshold

        CKMRiskVelocity newVelocity = CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.RENAL, 0.6)
                .compositeClassification(CKMRiskVelocity.CompositeClassification.DETERIORATING)
                .compositeScore(0.6)
                .dataCompleteness(0.625)
                .computationTimestamp(Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, newVelocity, Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS);

        assertTrue(events.stream().anyMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.CKM_RISK_ESCALATION),
                "HIGH-priority event should emit at 0.625 confidence");
    }

    @Test
    void detect_dedup_sameChangeNotEmittedTwiceIn24Hours() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.current().fbg = 105.0;
        state.getLastEmittedChangeTimestamps().put(
                ClinicalStateChangeType.METABOLIC_MILESTONE,
                Module13TestBuilder.BASE_TIME - Module13TestBuilder.HOUR_MS);

        CKMRiskVelocity velocity = CKMRiskVelocity.builder()
                .compositeClassification(CKMRiskVelocity.CompositeClassification.IMPROVING)
                .dataCompleteness(0.8)
                .computationTimestamp(Module13TestBuilder.BASE_TIME)
                .build();

        List<ClinicalStateChangeEvent> events = Module13StateChangeDetector.detect(
                state, velocity, Module13TestBuilder.BASE_TIME);

        assertTrue(events.stream().noneMatch(e ->
                e.getChangeType() == ClinicalStateChangeType.METABOLIC_MILESTONE),
                "Should NOT re-emit METABOLIC_MILESTONE within 24h dedup window");
    }
}
