package com.cardiofit.flink.processors;

import com.cardiofit.flink.cds.evaluation.ConditionEvaluator;
import com.cardiofit.flink.cds.evaluation.ConfidenceCalculator;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.protocol.ComparisonOperator;
import com.cardiofit.flink.models.protocol.ConfidenceModifier;
import com.cardiofit.flink.models.protocol.ConfidenceScoring;
import com.cardiofit.flink.models.protocol.ProtocolCondition;
import com.cardiofit.flink.models.protocol.TriggerCriteria;
import com.cardiofit.flink.models.protocol.MatchLogic;
import com.cardiofit.flink.protocol.models.Protocol;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for ProtocolMatcher confidence-based ranking functionality (Phase 2).
 *
 * Tests:
 * 1. Single protocol match with confidence score
 * 2. Multiple protocols ranked by confidence
 * 3. Filtering by activation threshold
 * 4. Empty results when none meet threshold
 * 5. Ranking stability (same input produces same order)
 *
 * @author Module 3 CDS Team - Agent 12
 * @version 1.0
 * @since 2025-10-21
 */
public class ProtocolMatcherRankingTest {

    private ProtocolMatcher protocolMatcher;
    private ConditionEvaluator conditionEvaluator;
    private ConfidenceCalculator confidenceCalculator;

    @BeforeEach
    public void setUp() {
        conditionEvaluator = new ConditionEvaluator();
        confidenceCalculator = new ConfidenceCalculator(conditionEvaluator);
        protocolMatcher = new ProtocolMatcher(conditionEvaluator, confidenceCalculator);
    }

    /**
     * Test 1: Single protocol match with confidence 0.90
     *
     * Verifies:
     * - Protocol triggers correctly
     * - Confidence is calculated
     * - Single match is returned
     * - Confidence score is as expected
     */
    @Test
    public void testRankingByConfidenceSingleMatch() {
        // Create patient context with high NEWS2 score
        EnrichedPatientContext context = createPatientContext("PATIENT-001", 7);

        // Create single protocol with high confidence
        Map<String, Protocol> protocols = new HashMap<>();
        Protocol sepsisProtocol = createProtocolWithConfidence(
                "SEPSIS-001",
                "Sepsis Management",
                0.90,  // base confidence
                0.70   // activation threshold
        );
        protocols.put("SEPSIS-001", sepsisProtocol);

        // Execute ranking
        List<ProtocolMatcher.ProtocolMatch> matches = protocolMatcher.matchProtocolsRanked(context, protocols);

        // Verify single match
        assertNotNull(matches);
        assertEquals(1, matches.size());

        ProtocolMatcher.ProtocolMatch match = matches.get(0);
        assertEquals("SEPSIS-001", match.getProtocolId());
        assertTrue(match.getConfidence() >= 0.70, "Confidence should be >= activation threshold");
        assertTrue(match.getConfidence() <= 1.0, "Confidence should be clamped to 1.0");
    }

    /**
     * Test 2: Multiple protocols ranked by confidence descending
     *
     * Verifies:
     * - Multiple protocols trigger
     * - Protocols are sorted by confidence (highest first)
     * - All above-threshold protocols are included
     */
    @Test
    public void testRankingByConfidenceMultipleMatches() {
        // Create patient context with moderate NEWS2 score
        EnrichedPatientContext context = createPatientContext("PATIENT-002", 6);

        // Create multiple protocols with different confidence scores
        Map<String, Protocol> protocols = new HashMap<>();

        // Protocol 1: High confidence (should be ranked 1st)
        Protocol protocol1 = createProtocolWithConfidence(
                "PROTOCOL-HIGH",
                "High Confidence Protocol",
                0.95,  // base confidence
                0.70   // activation threshold
        );
        protocols.put("PROTOCOL-HIGH", protocol1);

        // Protocol 2: Medium confidence (should be ranked 2nd)
        Protocol protocol2 = createProtocolWithConfidence(
                "PROTOCOL-MEDIUM",
                "Medium Confidence Protocol",
                0.80,  // base confidence
                0.70   // activation threshold
        );
        protocols.put("PROTOCOL-MEDIUM", protocol2);

        // Protocol 3: Lower confidence but above threshold (should be ranked 3rd)
        Protocol protocol3 = createProtocolWithConfidence(
                "PROTOCOL-LOWER",
                "Lower Confidence Protocol",
                0.72,  // base confidence
                0.70   // activation threshold
        );
        protocols.put("PROTOCOL-LOWER", protocol3);

        // Execute ranking
        List<ProtocolMatcher.ProtocolMatch> matches = protocolMatcher.matchProtocolsRanked(context, protocols);

        // Verify all protocols matched
        assertNotNull(matches);
        assertEquals(3, matches.size());

        // Verify ranking order (highest confidence first)
        assertEquals("PROTOCOL-HIGH", matches.get(0).getProtocolId());
        assertEquals("PROTOCOL-MEDIUM", matches.get(1).getProtocolId());
        assertEquals("PROTOCOL-LOWER", matches.get(2).getProtocolId());

        // Verify confidence ordering
        assertTrue(matches.get(0).getConfidence() >= matches.get(1).getConfidence(),
                "First protocol should have highest confidence");
        assertTrue(matches.get(1).getConfidence() >= matches.get(2).getConfidence(),
                "Second protocol should have higher confidence than third");
    }

    /**
     * Test 3: Filtering by activation threshold
     *
     * Verifies:
     * - Only protocols with confidence >= activation_threshold are returned
     * - Protocols below threshold are filtered out
     */
    @Test
    public void testFiltersByActivationThreshold() {
        // Create patient context
        EnrichedPatientContext context = createPatientContext("PATIENT-003", 5);

        // Create protocols with different thresholds
        Map<String, Protocol> protocols = new HashMap<>();

        // Protocol 1: Above threshold (should be included)
        Protocol protocol1 = createProtocolWithConfidence(
                "PROTOCOL-ABOVE",
                "Above Threshold",
                0.85,  // base confidence
                0.80   // activation threshold (will pass)
        );
        protocols.put("PROTOCOL-ABOVE", protocol1);

        // Protocol 2: Below threshold (should be filtered out)
        Protocol protocol2 = createProtocolWithConfidence(
                "PROTOCOL-BELOW",
                "Below Threshold",
                0.75,  // base confidence
                0.90   // activation threshold (will fail - base confidence too low)
        );
        protocols.put("PROTOCOL-BELOW", protocol2);

        // Execute ranking
        List<ProtocolMatcher.ProtocolMatch> matches = protocolMatcher.matchProtocolsRanked(context, protocols);

        // Verify only above-threshold protocol matched
        assertNotNull(matches);
        assertEquals(1, matches.size());
        assertEquals("PROTOCOL-ABOVE", matches.get(0).getProtocolId());
        assertTrue(matches.get(0).getConfidence() >= 0.80);
    }

    /**
     * Test 4: Empty result when none above threshold
     *
     * Verifies:
     * - When all protocols are below their activation thresholds
     * - Empty list is returned (not null)
     */
    @Test
    public void testEmptyResultWhenNoneAboveThreshold() {
        // Create patient context
        EnrichedPatientContext context = createPatientContext("PATIENT-004", 4);

        // Create protocols with very high thresholds
        Map<String, Protocol> protocols = new HashMap<>();

        Protocol protocol1 = createProtocolWithConfidence(
                "PROTOCOL-1",
                "High Threshold Protocol 1",
                0.60,  // base confidence
                0.95   // activation threshold (too high)
        );
        protocols.put("PROTOCOL-1", protocol1);

        Protocol protocol2 = createProtocolWithConfidence(
                "PROTOCOL-2",
                "High Threshold Protocol 2",
                0.65,  // base confidence
                0.95   // activation threshold (too high)
        );
        protocols.put("PROTOCOL-2", protocol2);

        // Execute ranking
        List<ProtocolMatcher.ProtocolMatch> matches = protocolMatcher.matchProtocolsRanked(context, protocols);

        // Verify empty result
        assertNotNull(matches);
        assertTrue(matches.isEmpty(), "Should return empty list when no protocols meet threshold");
    }

    /**
     * Test 5: Ranking stability - same input produces same order
     *
     * Verifies:
     * - Multiple executions with same input produce same ranking
     * - Ranking is deterministic and stable
     */
    @Test
    public void testRankingStability() {
        // Create patient context
        EnrichedPatientContext context = createPatientContext("PATIENT-005", 6);

        // Create multiple protocols
        Map<String, Protocol> protocols = new HashMap<>();
        protocols.put("PROTOCOL-A", createProtocolWithConfidence("PROTOCOL-A", "Protocol A", 0.90, 0.70));
        protocols.put("PROTOCOL-B", createProtocolWithConfidence("PROTOCOL-B", "Protocol B", 0.85, 0.70));
        protocols.put("PROTOCOL-C", createProtocolWithConfidence("PROTOCOL-C", "Protocol C", 0.80, 0.70));

        // Execute ranking multiple times
        List<ProtocolMatcher.ProtocolMatch> matches1 = protocolMatcher.matchProtocolsRanked(context, protocols);
        List<ProtocolMatcher.ProtocolMatch> matches2 = protocolMatcher.matchProtocolsRanked(context, protocols);
        List<ProtocolMatcher.ProtocolMatch> matches3 = protocolMatcher.matchProtocolsRanked(context, protocols);

        // Verify same results
        assertEquals(matches1.size(), matches2.size());
        assertEquals(matches1.size(), matches3.size());

        // Verify same order
        for (int i = 0; i < matches1.size(); i++) {
            assertEquals(matches1.get(i).getProtocolId(), matches2.get(i).getProtocolId(),
                    "Ranking order should be stable across executions");
            assertEquals(matches1.get(i).getProtocolId(), matches3.get(i).getProtocolId(),
                    "Ranking order should be stable across executions");
            assertEquals(matches1.get(i).getConfidence(), matches2.get(i).getConfidence(), 0.001,
                    "Confidence scores should be stable across executions");
        }
    }

    // ==================== Helper Methods ====================

    /**
     * Creates a patient context with specified NEWS2 score.
     */
    private EnrichedPatientContext createPatientContext(String patientId, int news2Score) {
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId(patientId);

        PatientContextState state = new PatientContextState();
        state.setNews2Score(news2Score);

        // Add some vitals for data completeness
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 90);
        vitals.put("bloodpressure_systolic", 120);
        vitals.put("bloodpressure_diastolic", 80);
        vitals.put("temperature", 37.0);
        state.setLatestVitals(vitals);

        context.setPatientState(state);
        return context;
    }

    /**
     * Creates a protocol with specified confidence scoring.
     */
    private Protocol createProtocolWithConfidence(
            String protocolId,
            String name,
            double baseConfidence,
            double activationThreshold) {

        Protocol protocol = new Protocol();
        protocol.setProtocolId(protocolId);
        protocol.setName(name);
        protocol.setCategory("TEST");
        protocol.setVersion("1.0");

        // Create simple trigger criteria (always triggers for NEWS2 >= 4)
        TriggerCriteria triggerCriteria = new TriggerCriteria();
        triggerCriteria.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition news2Condition = new ProtocolCondition();
        news2Condition.setConditionId("news2-check");
        news2Condition.setParameter("NEWS2");
        news2Condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        news2Condition.setThreshold(4);

        triggerCriteria.setConditions(Collections.singletonList(news2Condition));
        protocol.setTriggerCriteria(triggerCriteria);

        // Create confidence scoring
        ConfidenceScoring confidenceScoring = new ConfidenceScoring();
        confidenceScoring.setBaseConfidence(baseConfidence);
        confidenceScoring.setActivationThreshold(activationThreshold);
        protocol.setConfidenceScoring(confidenceScoring);

        return protocol;
    }
}
