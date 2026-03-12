package com.cardiofit.flink.cds.evaluation;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientState;
import com.cardiofit.flink.models.protocol.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for ConfidenceCalculator.
 *
 * <p>Tests cover:
 * - Base confidence calculation (no modifiers)
 * - Positive modifiers (+0.10, +0.05)
 * - Negative modifiers (-0.10)
 * - Clamping above 1.0
 * - Clamping below 0.0
 * - Activation threshold (>=0.70 passes, <0.70 fails)
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
class ConfidenceCalculatorTest {

    private ConditionEvaluator conditionEvaluator;
    private ConfidenceCalculator calculator;
    private EnrichedPatientContext context;

    @BeforeEach
    void setUp() {
        conditionEvaluator = new ConditionEvaluator();
        calculator = new ConfidenceCalculator(conditionEvaluator);
        context = createTestContext();
    }

    // ====================
    // Test 1: No Modifiers (Base Confidence Only)
    // ====================

    @Test
    @DisplayName("Test 1: Calculate confidence with no modifiers - should return base confidence")
    void testCalculateConfidence_NoModifiers_ReturnsBaseConfidence() {
        // Arrange
        Protocol protocol = new Protocol("TEST-001", "Test Protocol");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setBaseConfidence(0.85);
        scoring.setActivationThreshold(0.70);
        protocol.setConfidenceScoring(scoring);

        // Act
        double confidence = calculator.calculateConfidence(protocol, context);

        // Assert
        assertEquals(0.85, confidence, 0.001, "Confidence should equal base confidence when no modifiers");
    }

    // ====================
    // Tests 2-4: Positive Modifiers
    // ====================

    @Test
    @DisplayName("Test 2: Positive modifier +0.10 when age >= 65 (age = 72)")
    void testCalculateConfidence_PositiveModifier_AgeGreaterThan65() {
        // Arrange
        Protocol protocol = new Protocol("TEST-002", "Sepsis Protocol");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setBaseConfidence(0.80);

        // Modifier: +0.10 if age >= 65 (context has age 72)
        ConfidenceModifier modifier = new ConfidenceModifier();
        modifier.setModifierId("AGE_MODIFIER");
        modifier.setAdjustment(0.10);

        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("age");
        condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        condition.setThreshold(65);
        modifier.setCondition(condition);

        scoring.setModifiers(Arrays.asList(modifier));
        protocol.setConfidenceScoring(scoring);

        // Act
        double confidence = calculator.calculateConfidence(protocol, context);

        // Assert
        assertEquals(0.90, confidence, 0.001, "Confidence should be 0.80 + 0.10 = 0.90");
    }

    @Test
    @DisplayName("Test 3: Multiple positive modifiers +0.10 and +0.05")
    void testCalculateConfidence_MultiplePositiveModifiers() {
        // Arrange
        Protocol protocol = new Protocol("TEST-003", "Multi-Modifier Protocol");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setBaseConfidence(0.75);

        // Modifier 1: +0.10 if lactate >= 2.0 (context has lactate 3.5)
        ConfidenceModifier modifier1 = new ConfidenceModifier();
        modifier1.setModifierId("LACTATE_MODIFIER");
        modifier1.setAdjustment(0.10);

        ProtocolCondition condition1 = new ProtocolCondition();
        condition1.setParameter("lactate");
        condition1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        condition1.setThreshold(2.0);
        modifier1.setCondition(condition1);

        // Modifier 2: +0.05 if systolic_bp < 100 (context has systolic_bp 85)
        ConfidenceModifier modifier2 = new ConfidenceModifier();
        modifier2.setModifierId("BP_MODIFIER");
        modifier2.setAdjustment(0.05);

        ProtocolCondition condition2 = new ProtocolCondition();
        condition2.setParameter("systolic_bp");
        condition2.setOperator(ComparisonOperator.LESS_THAN);
        condition2.setThreshold(100);
        modifier2.setCondition(condition2);

        scoring.setModifiers(Arrays.asList(modifier1, modifier2));
        protocol.setConfidenceScoring(scoring);

        // Act
        double confidence = calculator.calculateConfidence(protocol, context);

        // Assert
        assertEquals(0.90, confidence, 0.001, "Confidence should be 0.75 + 0.10 + 0.05 = 0.90");
    }

    @Test
    @DisplayName("Test 4: Positive modifier not applied when condition fails")
    void testCalculateConfidence_PositiveModifier_ConditionNotMet() {
        // Arrange
        Protocol protocol = new Protocol("TEST-004", "Failed Modifier Protocol");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setBaseConfidence(0.80);

        // Modifier: +0.10 if age >= 80 (context has age 72, condition will fail)
        ConfidenceModifier modifier = new ConfidenceModifier();
        modifier.setModifierId("AGE_80_MODIFIER");
        modifier.setAdjustment(0.10);

        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("age");
        condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        condition.setThreshold(80);
        modifier.setCondition(condition);

        scoring.setModifiers(Arrays.asList(modifier));
        protocol.setConfidenceScoring(scoring);

        // Act
        double confidence = calculator.calculateConfidence(protocol, context);

        // Assert
        assertEquals(0.80, confidence, 0.001, "Confidence should remain 0.80 when modifier condition not met");
    }

    // ====================
    // Tests 5-6: Negative Modifiers
    // ====================

    @Test
    @DisplayName("Test 5: Negative modifier -0.10 when lactate >= 4.0 (lactate = 3.5, no penalty)")
    void testCalculateConfidence_NegativeModifier_ConditionNotMet() {
        // Arrange
        Protocol protocol = new Protocol("TEST-005", "Negative Modifier Test");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setBaseConfidence(0.85);

        // Modifier: -0.10 if lactate >= 4.0 (context has lactate 3.5, won't apply)
        ConfidenceModifier modifier = new ConfidenceModifier();
        modifier.setModifierId("HIGH_LACTATE_PENALTY");
        modifier.setAdjustment(-0.10);

        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("lactate");
        condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        condition.setThreshold(4.0);
        modifier.setCondition(condition);

        scoring.setModifiers(Arrays.asList(modifier));
        protocol.setConfidenceScoring(scoring);

        // Act
        double confidence = calculator.calculateConfidence(protocol, context);

        // Assert
        assertEquals(0.85, confidence, 0.001, "Confidence should remain 0.85 when negative modifier not triggered");
    }

    @Test
    @DisplayName("Test 6: Negative modifier -0.10 applied when condition met")
    void testCalculateConfidence_NegativeModifier_ConditionMet() {
        // Arrange
        Protocol protocol = new Protocol("TEST-006", "Negative Modifier Applied");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setBaseConfidence(0.85);

        // Modifier: -0.10 if systolic_bp < 90 (context has systolic_bp 85, will apply)
        ConfidenceModifier modifier = new ConfidenceModifier();
        modifier.setModifierId("HYPOTENSION_PENALTY");
        modifier.setAdjustment(-0.10);

        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("systolic_bp");
        condition.setOperator(ComparisonOperator.LESS_THAN);
        condition.setThreshold(90);
        modifier.setCondition(condition);

        scoring.setModifiers(Arrays.asList(modifier));
        protocol.setConfidenceScoring(scoring);

        // Act
        double confidence = calculator.calculateConfidence(protocol, context);

        // Assert
        assertEquals(0.75, confidence, 0.001, "Confidence should be 0.85 - 0.10 = 0.75");
    }

    // ====================
    // Test 7: Clamping Above 1.0
    // ====================

    @Test
    @DisplayName("Test 7: Confidence clamped to 1.0 when sum exceeds maximum")
    void testCalculateConfidence_ClampingAbove1_0() {
        // Arrange
        Protocol protocol = new Protocol("TEST-007", "Over-Confidence Protocol");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setBaseConfidence(0.95);

        // Modifier: +0.15 (would result in 1.10, should clamp to 1.0)
        ConfidenceModifier modifier = new ConfidenceModifier();
        modifier.setModifierId("LARGE_BOOST");
        modifier.setAdjustment(0.15);

        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("age");
        condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        condition.setThreshold(65);
        modifier.setCondition(condition);

        scoring.setModifiers(Arrays.asList(modifier));
        protocol.setConfidenceScoring(scoring);

        // Act
        double confidence = calculator.calculateConfidence(protocol, context);

        // Assert
        assertEquals(1.0, confidence, 0.001, "Confidence should be clamped to 1.0");
    }

    // ====================
    // Test 8: Clamping Below 0.0
    // ====================

    @Test
    @DisplayName("Test 8: Confidence clamped to 0.0 when sum goes negative")
    void testCalculateConfidence_ClampingBelow0_0() {
        // Arrange
        Protocol protocol = new Protocol("TEST-008", "Under-Confidence Protocol");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setBaseConfidence(0.20);

        // Modifier: -0.30 (would result in -0.10, should clamp to 0.0)
        ConfidenceModifier modifier = new ConfidenceModifier();
        modifier.setModifierId("LARGE_PENALTY");
        modifier.setAdjustment(-0.30);

        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("systolic_bp");
        condition.setOperator(ComparisonOperator.LESS_THAN);
        condition.setThreshold(90);
        modifier.setCondition(condition);

        scoring.setModifiers(Arrays.asList(modifier));
        protocol.setConfidenceScoring(scoring);

        // Act
        double confidence = calculator.calculateConfidence(protocol, context);

        // Assert
        assertEquals(0.0, confidence, 0.001, "Confidence should be clamped to 0.0");
    }

    // ====================
    // Tests 9-11: Activation Threshold
    // ====================

    @Test
    @DisplayName("Test 9: Meets activation threshold when confidence = 0.85 and threshold = 0.70")
    void testMeetsActivationThreshold_True_WellAboveThreshold() {
        // Arrange
        Protocol protocol = new Protocol("TEST-009", "High Confidence Protocol");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setActivationThreshold(0.70);
        protocol.setConfidenceScoring(scoring);

        double confidence = 0.85;

        // Act
        boolean meets = calculator.meetsActivationThreshold(protocol, confidence);

        // Assert
        assertTrue(meets, "Confidence 0.85 should meet threshold 0.70");
    }

    @Test
    @DisplayName("Test 10: Meets activation threshold when confidence exactly equals threshold")
    void testMeetsActivationThreshold_True_ExactlyAtThreshold() {
        // Arrange
        Protocol protocol = new Protocol("TEST-010", "Exact Threshold Protocol");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setActivationThreshold(0.70);
        protocol.setConfidenceScoring(scoring);

        double confidence = 0.70;

        // Act
        boolean meets = calculator.meetsActivationThreshold(protocol, confidence);

        // Assert
        assertTrue(meets, "Confidence 0.70 should meet threshold 0.70 (inclusive)");
    }

    @Test
    @DisplayName("Test 11: Does not meet activation threshold when confidence < threshold")
    void testMeetsActivationThreshold_False_BelowThreshold() {
        // Arrange
        Protocol protocol = new Protocol("TEST-011", "Low Confidence Protocol");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setActivationThreshold(0.70);
        protocol.setConfidenceScoring(scoring);

        double confidence = 0.65;

        // Act
        boolean meets = calculator.meetsActivationThreshold(protocol, confidence);

        // Assert
        assertFalse(meets, "Confidence 0.65 should not meet threshold 0.70");
    }

    // ====================
    // Helper Methods
    // ====================

    /**
     * Creates a test context with typical patient data.
     *
     * Patient: 72-year-old with hypotension, elevated lactate, infection suspected
     */
    private EnrichedPatientContext createTestContext() {
        EnrichedPatientContext ctx = new EnrichedPatientContext();
        PatientState state = new PatientState();

        // Demographics
        state.setAge(72);
        state.setSex("M");
        state.setWeight(75.0);

        // Vitals
        state.setSystolicBP(85.0);
        state.setDiastolicBP(55.0);
        state.setMeanArterialPressure(60.0);
        state.setHeartRate(110.0);
        state.setRespiratoryRate(22.0);
        state.setTemperature(38.5);
        state.setOxygenSaturation(94.0);

        // Lab values
        state.setLactate(3.5);
        state.setWhiteBloodCount(15.2);
        state.setCreatinine(1.4);

        // Clinical assessment
        state.setInfectionSuspected(true);
        state.setAllergies(Arrays.asList("penicillin", "iodine"));

        ctx.setPatientState(state);
        ctx.setPatientId("TEST-PATIENT-001");

        return ctx;
    }

    // ====================
    // Edge Case Tests
    // ====================

    @Test
    @DisplayName("Edge case: Null protocol returns 0.0 confidence")
    void testCalculateConfidence_NullProtocol_Returns0() {
        double confidence = calculator.calculateConfidence(null, context);
        assertEquals(0.0, confidence, 0.001, "Null protocol should return 0.0 confidence");
    }

    @Test
    @DisplayName("Edge case: Null context returns 0.0 confidence")
    void testCalculateConfidence_NullContext_Returns0() {
        Protocol protocol = new Protocol("TEST-NULL", "Null Context Test");
        double confidence = calculator.calculateConfidence(protocol, null);
        assertEquals(0.0, confidence, 0.001, "Null context should return 0.0 confidence");
    }

    @Test
    @DisplayName("Edge case: Protocol with no confidence scoring uses default")
    void testCalculateConfidence_NoConfidenceScoring_UsesDefault() {
        Protocol protocol = new Protocol("TEST-DEFAULT", "No Scoring Protocol");
        // Don't set confidence scoring - should use default

        double confidence = calculator.calculateConfidence(protocol, context);

        assertEquals(ConfidenceCalculator.DEFAULT_BASE_CONFIDENCE, confidence, 0.001,
                "Protocol without scoring should use default base confidence");
    }

    @Test
    @DisplayName("Edge case: Null modifier in list is skipped")
    void testCalculateConfidence_NullModifierInList_Skipped() {
        Protocol protocol = new Protocol("TEST-NULL-MOD", "Null Modifier Test");
        ConfidenceScoring scoring = new ConfidenceScoring();
        scoring.setBaseConfidence(0.80);

        // Add null modifier (should be skipped)
        List<ConfidenceModifier> modifiers = new ArrayList<>();
        modifiers.add(null);
        scoring.setModifiers(modifiers);
        protocol.setConfidenceScoring(scoring);

        double confidence = calculator.calculateConfidence(protocol, context);

        assertEquals(0.80, confidence, 0.001, "Null modifiers should be skipped gracefully");
    }
}
