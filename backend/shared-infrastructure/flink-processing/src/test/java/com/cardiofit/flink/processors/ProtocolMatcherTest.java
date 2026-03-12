package com.cardiofit.flink.processors;

import com.cardiofit.flink.cds.evaluation.ConditionEvaluator;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientState;
import com.cardiofit.flink.models.protocol.*;
import com.cardiofit.flink.protocol.models.Protocol;
import com.cardiofit.flink.processors.ProtocolMatcher.ProtocolMatch;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for ProtocolMatcher with ConditionEvaluator integration (Phase 1).
 *
 * Tests verify that:
 * - Protocols with simple trigger criteria match correctly
 * - Protocols with complex AND/OR triggers match correctly
 * - Non-matching protocols are filtered out
 * - Logging shows which protocols triggered and why
 *
 * @author Module 3 CDS Team - Phase 1
 * @version 1.0
 * @since 2025-10-21
 */
class ProtocolMatcherTest {

    private ProtocolMatcher protocolMatcher;
    private EnrichedPatientContext context;
    private PatientState patientState;

    @BeforeEach
    void setUp() {
        ConditionEvaluator conditionEvaluator = new ConditionEvaluator();
        protocolMatcher = new ProtocolMatcher(conditionEvaluator);

        // Setup patient context
        context = new EnrichedPatientContext();
        context.setPatientId("TEST-001");

        patientState = new PatientState();
        context.setPatientState(patientState);
    }

    /**
     * Test 1: Protocol with simple trigger matches when criteria met
     */
    @Test
    void testSimpleTriggerMatches_LactateHigh() {
        // Given: Patient with elevated lactate
        patientState.setLactate(3.5);

        // Create protocol with simple trigger: lactate >= 2.0
        Protocol sepsisProtocol = createSepsisProtocol();
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition lactateCondition = new ProtocolCondition();
        lactateCondition.setConditionId("lactate-elevated");
        lactateCondition.setParameter("lactate");
        lactateCondition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        lactateCondition.setThreshold(2.0);

        trigger.setConditions(Arrays.asList(lactateCondition));
        sepsisProtocol.setTriggerCriteria(trigger);

        // When: Match protocols
        List<ProtocolMatch> matches = protocolMatcher.matchProtocols(context, Arrays.asList(sepsisProtocol));

        // Then: Sepsis protocol should match
        assertEquals(1, matches.size(), "Should match 1 protocol");
        assertEquals("SEPSIS-BUNDLE-001", matches.get(0).getProtocolId());
        assertTrue(matches.get(0).getConfidence() >= 0.5, "Confidence should be >= 0.5");
    }

    /**
     * Test 2: Protocol with simple trigger does NOT match when criteria not met
     */
    @Test
    void testSimpleTriggerDoesNotMatch_LactateNormal() {
        // Given: Patient with normal lactate
        patientState.setLactate(1.2);

        // Create protocol with simple trigger: lactate >= 2.0
        Protocol sepsisProtocol = createSepsisProtocol();
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition lactateCondition = new ProtocolCondition();
        lactateCondition.setConditionId("lactate-elevated");
        lactateCondition.setParameter("lactate");
        lactateCondition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        lactateCondition.setThreshold(2.0);

        trigger.setConditions(Arrays.asList(lactateCondition));
        sepsisProtocol.setTriggerCriteria(trigger);

        // When: Match protocols
        List<ProtocolMatch> matches = protocolMatcher.matchProtocols(context, Arrays.asList(sepsisProtocol));

        // Then: No protocols should match
        assertEquals(0, matches.size(), "Should match 0 protocols");
    }

    /**
     * Test 3: Protocol with complex AND trigger matches when ALL criteria met
     */
    @Test
    void testComplexAndTriggerMatches_AllCriteriaMet() {
        // Given: Patient with elevated lactate AND low blood pressure
        patientState.setLactate(3.0);
        patientState.setSystolicBP(85.0);
        patientState.setInfectionSuspected(true);

        // Create protocol with AND trigger: lactate >= 2.0 AND systolic_bp < 90
        Protocol sepsisProtocol = createSepsisProtocol();
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition lactateCondition = new ProtocolCondition();
        lactateCondition.setConditionId("lactate-elevated");
        lactateCondition.setParameter("lactate");
        lactateCondition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        lactateCondition.setThreshold(2.0);

        ProtocolCondition bpCondition = new ProtocolCondition();
        bpCondition.setConditionId("hypotension");
        bpCondition.setParameter("systolic_bp");
        bpCondition.setOperator(ComparisonOperator.LESS_THAN);
        bpCondition.setThreshold(90.0);

        trigger.setConditions(Arrays.asList(lactateCondition, bpCondition));
        sepsisProtocol.setTriggerCriteria(trigger);

        // When: Match protocols
        List<ProtocolMatch> matches = protocolMatcher.matchProtocols(context, Arrays.asList(sepsisProtocol));

        // Then: Sepsis protocol should match
        assertEquals(1, matches.size(), "Should match 1 protocol");
        assertEquals("SEPSIS-BUNDLE-001", matches.get(0).getProtocolId());
    }

    /**
     * Test 4: Protocol with complex AND trigger does NOT match when one criteria fails
     */
    @Test
    void testComplexAndTriggerDoesNotMatch_OneCriteriaFails() {
        // Given: Patient with elevated lactate but NORMAL blood pressure
        patientState.setLactate(3.0);
        patientState.setSystolicBP(120.0); // Normal BP

        // Create protocol with AND trigger: lactate >= 2.0 AND systolic_bp < 90
        Protocol sepsisProtocol = createSepsisProtocol();
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition lactateCondition = new ProtocolCondition();
        lactateCondition.setConditionId("lactate-elevated");
        lactateCondition.setParameter("lactate");
        lactateCondition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        lactateCondition.setThreshold(2.0);

        ProtocolCondition bpCondition = new ProtocolCondition();
        bpCondition.setConditionId("hypotension");
        bpCondition.setParameter("systolic_bp");
        bpCondition.setOperator(ComparisonOperator.LESS_THAN);
        bpCondition.setThreshold(90.0);

        trigger.setConditions(Arrays.asList(lactateCondition, bpCondition));
        sepsisProtocol.setTriggerCriteria(trigger);

        // When: Match protocols
        List<ProtocolMatch> matches = protocolMatcher.matchProtocols(context, Arrays.asList(sepsisProtocol));

        // Then: No protocols should match (AND logic requires ALL conditions)
        assertEquals(0, matches.size(), "Should match 0 protocols (AND requires all conditions)");
    }

    /**
     * Test 5: Protocol with complex OR trigger matches when ANY criteria met
     */
    @Test
    void testComplexOrTriggerMatches_OneCriteriaMet() {
        // Given: Patient with elevated lactate but normal BP
        patientState.setLactate(3.0);
        patientState.setSystolicBP(120.0);

        // Create protocol with OR trigger: lactate >= 2.0 OR systolic_bp < 90
        Protocol sepsisProtocol = createSepsisProtocol();
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ANY_OF); // OR logic

        ProtocolCondition lactateCondition = new ProtocolCondition();
        lactateCondition.setConditionId("lactate-elevated");
        lactateCondition.setParameter("lactate");
        lactateCondition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        lactateCondition.setThreshold(2.0);

        ProtocolCondition bpCondition = new ProtocolCondition();
        bpCondition.setConditionId("hypotension");
        bpCondition.setParameter("systolic_bp");
        bpCondition.setOperator(ComparisonOperator.LESS_THAN);
        bpCondition.setThreshold(90.0);

        trigger.setConditions(Arrays.asList(lactateCondition, bpCondition));
        sepsisProtocol.setTriggerCriteria(trigger);

        // When: Match protocols
        List<ProtocolMatch> matches = protocolMatcher.matchProtocols(context, Arrays.asList(sepsisProtocol));

        // Then: Sepsis protocol should match (OR logic needs only one condition)
        assertEquals(1, matches.size(), "Should match 1 protocol (OR needs any condition)");
        assertEquals("SEPSIS-BUNDLE-001", matches.get(0).getProtocolId());
    }

    /**
     * Test 6: No protocols match when patient doesn't meet any criteria
     */
    @Test
    void testNoProtocolsMatch_NoCriteriaMet() {
        // Given: Patient with normal vitals
        patientState.setLactate(1.0);
        patientState.setSystolicBP(120.0);
        patientState.setHeartRate(75.0);

        // Create multiple protocols with various triggers
        List<Protocol> protocols = new ArrayList<>();

        // Sepsis protocol: lactate >= 2.0
        Protocol sepsisProtocol = createSepsisProtocol();
        TriggerCriteria sepsisTrigger = new TriggerCriteria();
        sepsisTrigger.setMatchLogic(MatchLogic.ALL_OF);
        ProtocolCondition lactateCondition = new ProtocolCondition();
        lactateCondition.setParameter("lactate");
        lactateCondition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        lactateCondition.setThreshold(2.0);
        sepsisTrigger.setConditions(Arrays.asList(lactateCondition));
        sepsisProtocol.setTriggerCriteria(sepsisTrigger);
        protocols.add(sepsisProtocol);

        // Hypotension protocol: systolic_bp < 90
        Protocol hypotensionProtocol = new Protocol("HYPOTENSION-001", "Hypotension Management", "CARDIOVASCULAR");
        TriggerCriteria hypotensionTrigger = new TriggerCriteria();
        hypotensionTrigger.setMatchLogic(MatchLogic.ALL_OF);
        ProtocolCondition bpCondition = new ProtocolCondition();
        bpCondition.setParameter("systolic_bp");
        bpCondition.setOperator(ComparisonOperator.LESS_THAN);
        bpCondition.setThreshold(90.0);
        hypotensionTrigger.setConditions(Arrays.asList(bpCondition));
        hypotensionProtocol.setTriggerCriteria(hypotensionTrigger);
        protocols.add(hypotensionProtocol);

        // When: Match protocols
        List<ProtocolMatch> matches = protocolMatcher.matchProtocols(context, protocols);

        // Then: No protocols should match
        assertEquals(0, matches.size(), "Should match 0 protocols");
    }

    // Helper methods

    private Protocol createSepsisProtocol() {
        Protocol protocol = new Protocol();
        protocol.setProtocolId("SEPSIS-BUNDLE-001");
        protocol.setName("Sepsis Management Bundle");
        protocol.setCategory("INFECTIOUS");
        protocol.setSpecialty("Emergency Medicine");
        return protocol;
    }
}
