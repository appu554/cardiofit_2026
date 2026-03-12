package com.cardiofit.flink.cds.escalation;

import com.cardiofit.flink.cds.evaluation.ConditionEvaluator;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for EscalationRuleEvaluator.
 *
 * Tests escalation rule evaluation, recommendation generation, and evidence gathering
 * for clinical deterioration detection.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-10-21
 */
class EscalationRuleEvaluatorTest {

    private EscalationRuleEvaluator evaluator;
    private ConditionEvaluator conditionEvaluator;
    private Protocol protocol;
    private EnrichedPatientContext context;
    private PatientState patientState;

    @BeforeEach
    void setUp() {
        conditionEvaluator = new ConditionEvaluator();
        evaluator = new EscalationRuleEvaluator(conditionEvaluator);

        // Set up protocol
        protocol = new Protocol();
        protocol.setProtocolId("SEPSIS-001");
        protocol.setName("Sepsis Management Protocol");

        // Set up patient context
        patientState = new PatientState();
        patientState.setPatientId("patient-123");

        context = new EnrichedPatientContext();
        context.setPatientId("patient-123");
        context.setEncounterId("encounter-456");
        context.setPatientState(patientState);
    }

    @Test
    @DisplayName("Test 1: Escalation triggered - recommendation generated")
    void testEscalationTriggered() {
        // Arrange: Set lactate to 4.5 (triggers escalation)
        patientState.setLactate(4.5);

        EscalationRule rule = createSepsisEscalationRule();
        protocol.setEscalationRules(List.of(rule));

        // Act
        List<EscalationRecommendation> recommendations = evaluator.evaluateEscalationRules(protocol, context);

        // Assert
        assertNotNull(recommendations, "Recommendations list should not be null");
        assertEquals(1, recommendations.size(), "Should generate 1 recommendation");

        EscalationRecommendation rec = recommendations.get(0);
        assertEquals("SEPSIS-ESC-001", rec.getRuleId());
        assertEquals("ICU_TRANSFER", rec.getEscalationLevel());
        assertEquals("Critical Care", rec.getSpecialty());
        assertEquals("Septic shock requiring vasopressor support", rec.getRationale());
        assertEquals("IMMEDIATE", rec.getUrgency());
        assertEquals("patient-123", rec.getPatientId());
        assertEquals("encounter-456", rec.getEncounterId());
        assertNotNull(rec.getTimestamp());
    }

    @Test
    @DisplayName("Test 2: Escalation not triggered - empty list")
    void testEscalationNotTriggered() {
        // Arrange: Set lactate to 2.0 (below threshold)
        patientState.setLactate(2.0);

        EscalationRule rule = createSepsisEscalationRule();
        protocol.setEscalationRules(List.of(rule));

        // Act
        List<EscalationRecommendation> recommendations = evaluator.evaluateEscalationRules(protocol, context);

        // Assert
        assertNotNull(recommendations, "Recommendations list should not be null");
        assertEquals(0, recommendations.size(), "Should not generate recommendations");
    }

    @Test
    @DisplayName("Test 3: Multiple escalations triggered - multiple recommendations")
    void testMultipleEscalationsTriggered() {
        // Arrange: Set conditions that trigger multiple rules
        patientState.setLactate(4.5); // Triggers sepsis escalation
        patientState.setSystolicBP(80.0); // Triggers hypotension escalation

        EscalationRule sepsisRule = createSepsisEscalationRule();
        EscalationRule hypotensionRule = createHypotensionEscalationRule();
        protocol.setEscalationRules(List.of(sepsisRule, hypotensionRule));

        // Act
        List<EscalationRecommendation> recommendations = evaluator.evaluateEscalationRules(protocol, context);

        // Assert
        assertNotNull(recommendations);
        assertEquals(2, recommendations.size(), "Should generate 2 recommendations");

        // Verify both rules triggered
        List<String> ruleIds = recommendations.stream()
                .map(EscalationRecommendation::getRuleId)
                .collect(Collectors.toList());
        assertTrue(ruleIds.contains("SEPSIS-ESC-001"), "Should include sepsis escalation");
        assertTrue(ruleIds.contains("HYPOTENSION-ESC-001"), "Should include hypotension escalation");
    }

    @Test
    @DisplayName("Test 4: Evidence gathering - recommendation includes clinical evidence")
    void testEvidenceGathering() {
        // Arrange: Set multiple clinical indicators
        patientState.setLactate(4.5); // Primary trigger
        patientState.setSystolicBP(85.0); // Hypotensive
        patientState.setHeartRate(125.0); // Tachycardic
        patientState.setOxygenSaturation(90.0); // Hypoxic
        patientState.setWhiteBloodCount(15.0); // Elevated WBC
        patientState.setNews2Score(7); // Elevated NEWS2

        EscalationRule rule = createSepsisEscalationRule();
        protocol.setEscalationRules(List.of(rule));

        // Act
        List<EscalationRecommendation> recommendations = evaluator.evaluateEscalationRules(protocol, context);

        // Assert
        assertEquals(1, recommendations.size());
        EscalationRecommendation rec = recommendations.get(0);

        assertNotNull(rec.getEvidence(), "Evidence list should not be null");
        assertTrue(rec.getEvidence().size() > 0, "Should have at least one evidence item");

        // Verify evidence contains expected items
        List<String> evidence = rec.getEvidence();
        assertTrue(evidence.stream().anyMatch(e -> e.contains("lactate")),
                "Should include lactate evidence");
        assertTrue(evidence.stream().anyMatch(e -> e.contains("Systolic BP")),
                "Should include blood pressure evidence");
        assertTrue(evidence.stream().anyMatch(e -> e.contains("Heart Rate")),
                "Should include heart rate evidence");
        assertTrue(evidence.stream().anyMatch(e -> e.contains("SpO2")),
                "Should include oxygen saturation evidence");
        assertTrue(evidence.stream().anyMatch(e -> e.contains("NEWS2 Score")),
                "Should include NEWS2 score evidence");
    }

    @Test
    @DisplayName("Test 5: No escalation rules - empty list")
    void testNoEscalationRules() {
        // Arrange: Protocol with no escalation rules
        protocol.setEscalationRules(new ArrayList<>());

        // Act
        List<EscalationRecommendation> recommendations = evaluator.evaluateEscalationRules(protocol, context);

        // Assert
        assertNotNull(recommendations);
        assertEquals(0, recommendations.size(), "Should return empty list when no rules defined");
    }

    @Test
    @DisplayName("Test 6: Patient identifiers populated correctly")
    void testPatientIdentifiersPopulated() {
        // Arrange
        patientState.setLactate(4.5);
        context.setPatientId("test-patient-789");
        context.setEncounterId("test-encounter-999");

        EscalationRule rule = createSepsisEscalationRule();
        protocol.setEscalationRules(List.of(rule));

        // Act
        List<EscalationRecommendation> recommendations = evaluator.evaluateEscalationRules(protocol, context);

        // Assert
        assertEquals(1, recommendations.size());
        EscalationRecommendation rec = recommendations.get(0);

        assertEquals("test-patient-789", rec.getPatientId(),
                "Patient ID should match context");
        assertEquals("test-encounter-999", rec.getEncounterId(),
                "Encounter ID should match context");
        assertEquals("SEPSIS-001", rec.getProtocolId(),
                "Protocol ID should match protocol");
        assertEquals("Sepsis Management Protocol", rec.getProtocolName(),
                "Protocol name should match protocol");
    }

    // Helper methods to create test escalation rules

    private EscalationRule createSepsisEscalationRule() {
        EscalationRule rule = new EscalationRule();
        rule.setRuleId("SEPSIS-ESC-001");

        // Create escalation trigger: lactate >= 4.0
        ProtocolCondition trigger = new ProtocolCondition();
        trigger.setParameter("lactate");
        trigger.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        trigger.setThreshold(4.0);
        rule.setEscalationTrigger(trigger);

        // Create recommendation
        EscalationRule.EscalationRecommendationTemplate recommendation =
                new EscalationRule.EscalationRecommendationTemplate();
        recommendation.setEscalationLevel("ICU_TRANSFER");
        recommendation.setSpecialty("Critical Care");
        recommendation.setRationale("Septic shock requiring vasopressor support");
        recommendation.setUrgency("IMMEDIATE");
        rule.setRecommendation(recommendation);

        return rule;
    }

    private EscalationRule createHypotensionEscalationRule() {
        EscalationRule rule = new EscalationRule();
        rule.setRuleId("HYPOTENSION-ESC-001");

        // Create escalation trigger: systolic_bp <= 85
        ProtocolCondition trigger = new ProtocolCondition();
        trigger.setParameter("systolic_bp");
        trigger.setOperator(ComparisonOperator.LESS_THAN_OR_EQUAL);
        trigger.setThreshold(85.0);
        rule.setEscalationTrigger(trigger);

        // Create recommendation
        EscalationRule.EscalationRecommendationTemplate recommendation =
                new EscalationRule.EscalationRecommendationTemplate();
        recommendation.setEscalationLevel("RAPID_RESPONSE");
        recommendation.setSpecialty("Critical Care");
        recommendation.setRationale("Severe hypotension requiring immediate intervention");
        recommendation.setUrgency("IMMEDIATE");
        rule.setRecommendation(recommendation);

        return rule;
    }
}
