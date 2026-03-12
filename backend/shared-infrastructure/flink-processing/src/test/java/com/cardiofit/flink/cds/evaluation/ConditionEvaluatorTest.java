package com.cardiofit.flink.cds.evaluation;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientState;
import com.cardiofit.flink.models.protocol.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.Arrays;
import java.util.Collections;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for ConditionEvaluator.
 *
 * Tests cover:
 * - Simple condition evaluation (10 tests)
 * - ALL_OF logic (3 tests)
 * - ANY_OF logic (3 tests)
 * - Nested conditions (4 tests)
 * - Operator tests (6 tests)
 * - Parameter extraction (5 tests)
 *
 * Total: 31 tests
 */
class ConditionEvaluatorTest {

    private ConditionEvaluator evaluator;
    private EnrichedPatientContext context;

    @BeforeEach
    void setUp() {
        evaluator = new ConditionEvaluator();
        context = createTestContext();
    }

    // ========== Simple Condition Tests (10 tests) ==========

    @Test
    @DisplayName("Test simple condition: lactate >= 2.0 (TRUE)")
    void testSimpleCondition_GreaterThanOrEqual_True() {
        // lactate >= 2.0 (actual: 3.5)
        ProtocolCondition condition = new ProtocolCondition();
        condition.setConditionId("COND-001");
        condition.setParameter("lactate");
        condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        condition.setThreshold(2.0);

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertTrue(result, "Lactate 3.5 should be >= 2.0");
    }

    @Test
    @DisplayName("Test simple condition: lactate >= 4.0 (FALSE)")
    void testSimpleCondition_GreaterThanOrEqual_False() {
        // lactate >= 4.0 (actual: 3.5)
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("lactate");
        condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        condition.setThreshold(4.0);

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertFalse(result, "Lactate 3.5 should not be >= 4.0");
    }

    @Test
    @DisplayName("Test simple condition: systolic_bp < 90 (TRUE)")
    void testSimpleCondition_LessThan_True() {
        // systolic_bp < 90 (actual: 85)
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("systolic_bp");
        condition.setOperator(ComparisonOperator.LESS_THAN);
        condition.setThreshold(90);

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertTrue(result, "Systolic BP 85 should be < 90");
    }

    @Test
    @DisplayName("Test simple condition: systolic_bp <= 85 (TRUE)")
    void testSimpleCondition_LessThanOrEqual_True() {
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("systolic_bp");
        condition.setOperator(ComparisonOperator.LESS_THAN_OR_EQUAL);
        condition.setThreshold(85);

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertTrue(result, "Systolic BP 85 should be <= 85");
    }

    @Test
    @DisplayName("Test simple condition: age > 65 (TRUE)")
    void testSimpleCondition_GreaterThan_True() {
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("age");
        condition.setOperator(ComparisonOperator.GREATER_THAN);
        condition.setThreshold(65);

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertTrue(result, "Age 72 should be > 65");
    }

    @Test
    @DisplayName("Test simple condition: age == 72 (TRUE)")
    void testSimpleCondition_Equal_True() {
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("age");
        condition.setOperator(ComparisonOperator.EQUAL);
        condition.setThreshold(72);

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertTrue(result, "Age 72 should equal 72");
    }

    @Test
    @DisplayName("Test simple condition: age != 65 (TRUE)")
    void testSimpleCondition_NotEqual_True() {
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("age");
        condition.setOperator(ComparisonOperator.NOT_EQUAL);
        condition.setThreshold(65);

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertTrue(result, "Age 72 should not equal 65");
    }

    @Test
    @DisplayName("Test simple condition: infection_suspected == true (TRUE)")
    void testSimpleCondition_BooleanEqual_True() {
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("infection_suspected");
        condition.setOperator(ComparisonOperator.EQUAL);
        condition.setThreshold(true);

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertTrue(result, "Infection suspected should be true");
    }

    @Test
    @DisplayName("Test CONTAINS operator: allergies contains 'penicillin' (TRUE)")
    void testContainsOperator_AllergiesContainsPenicillin() {
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("allergies");
        condition.setOperator(ComparisonOperator.CONTAINS);
        condition.setThreshold("penicillin");

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertTrue(result, "Allergies should contain penicillin");
    }

    @Test
    @DisplayName("Test NOT_CONTAINS operator: allergies not contains 'sulfa' (TRUE)")
    void testNotContainsOperator() {
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("allergies");
        condition.setOperator(ComparisonOperator.NOT_CONTAINS);
        condition.setThreshold("sulfa");

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertTrue(result, "Allergies should not contain sulfa");
    }

    // ========== ALL_OF Logic Tests (3 tests) ==========

    @Test
    @DisplayName("Test ALL_OF: all conditions met (TRUE)")
    void testAllOfLogic_AllConditionsMet() {
        // lactate >= 2.0 AND systolic_bp < 90 AND infection_suspected == true
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition cond1 = new ProtocolCondition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        ProtocolCondition cond2 = new ProtocolCondition();
        cond2.setParameter("systolic_bp");
        cond2.setOperator(ComparisonOperator.LESS_THAN);
        cond2.setThreshold(90);

        ProtocolCondition cond3 = new ProtocolCondition();
        cond3.setParameter("infection_suspected");
        cond3.setOperator(ComparisonOperator.EQUAL);
        cond3.setThreshold(true);

        trigger.setConditions(Arrays.asList(cond1, cond2, cond3));

        boolean result = evaluator.evaluate(trigger, context);
        assertTrue(result, "All conditions met, should return true");
    }

    @Test
    @DisplayName("Test ALL_OF: one condition failed (FALSE)")
    void testAllOfLogic_OneConditionFailed() {
        // lactate >= 2.0 AND systolic_bp < 70 (will fail)
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition cond1 = new ProtocolCondition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        ProtocolCondition cond2 = new ProtocolCondition();
        cond2.setParameter("systolic_bp");
        cond2.setOperator(ComparisonOperator.LESS_THAN);
        cond2.setThreshold(70); // Actual is 85, will fail

        trigger.setConditions(Arrays.asList(cond1, cond2));

        boolean result = evaluator.evaluate(trigger, context);
        assertFalse(result, "One condition failed, should return false");
    }

    @Test
    @DisplayName("Test ALL_OF: empty conditions (FALSE)")
    void testAllOfLogic_EmptyConditions() {
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);
        trigger.setConditions(Collections.emptyList());

        boolean result = evaluator.evaluate(trigger, context);
        assertFalse(result, "Empty conditions should return false");
    }

    // ========== ANY_OF Logic Tests (3 tests) ==========

    @Test
    @DisplayName("Test ANY_OF: one condition met (TRUE)")
    void testAnyOfLogic_OneConditionMet() {
        // lactate >= 2.0 OR systolic_bp < 70 (first will pass)
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ANY_OF);

        ProtocolCondition cond1 = new ProtocolCondition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        ProtocolCondition cond2 = new ProtocolCondition();
        cond2.setParameter("systolic_bp");
        cond2.setOperator(ComparisonOperator.LESS_THAN);
        cond2.setThreshold(70); // Will fail

        trigger.setConditions(Arrays.asList(cond1, cond2));

        boolean result = evaluator.evaluate(trigger, context);
        assertTrue(result, "One condition met, should return true");
    }

    @Test
    @DisplayName("Test ANY_OF: no conditions met (FALSE)")
    void testAnyOfLogic_NoConditionsMet() {
        // lactate >= 5.0 OR systolic_bp < 70 (both fail)
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ANY_OF);

        ProtocolCondition cond1 = new ProtocolCondition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(5.0); // Will fail

        ProtocolCondition cond2 = new ProtocolCondition();
        cond2.setParameter("systolic_bp");
        cond2.setOperator(ComparisonOperator.LESS_THAN);
        cond2.setThreshold(70); // Will fail

        trigger.setConditions(Arrays.asList(cond1, cond2));

        boolean result = evaluator.evaluate(trigger, context);
        assertFalse(result, "No conditions met, should return false");
    }

    @Test
    @DisplayName("Test ANY_OF: all conditions met (TRUE)")
    void testAnyOfLogic_AllConditionsMet() {
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ANY_OF);

        ProtocolCondition cond1 = new ProtocolCondition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        ProtocolCondition cond2 = new ProtocolCondition();
        cond2.setParameter("age");
        cond2.setOperator(ComparisonOperator.GREATER_THAN);
        cond2.setThreshold(65);

        trigger.setConditions(Arrays.asList(cond1, cond2));

        boolean result = evaluator.evaluate(trigger, context);
        assertTrue(result, "All conditions met (ANY_OF), should return true");
    }

    // ========== Nested Condition Tests (4 tests) ==========

    @Test
    @DisplayName("Test nested: ALL_OF containing ANY_OF (TRUE)")
    void testNestedConditions_AllOfContainingAnyOf() {
        // lactate >= 2.0 AND (systolic_bp < 90 OR map < 65)
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition cond1 = new ProtocolCondition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        // Nested ANY_OF
        ProtocolCondition nestedCond = new ProtocolCondition();
        nestedCond.setMatchLogic(MatchLogic.ANY_OF);

        ProtocolCondition nested1 = new ProtocolCondition();
        nested1.setParameter("systolic_bp");
        nested1.setOperator(ComparisonOperator.LESS_THAN);
        nested1.setThreshold(90);

        ProtocolCondition nested2 = new ProtocolCondition();
        nested2.setParameter("map");
        nested2.setOperator(ComparisonOperator.LESS_THAN);
        nested2.setThreshold(65);

        nestedCond.setConditions(Arrays.asList(nested1, nested2));

        trigger.setConditions(Arrays.asList(cond1, nestedCond));

        boolean result = evaluator.evaluate(trigger, context);
        assertTrue(result, "Nested condition should evaluate correctly");
    }

    @Test
    @DisplayName("Test nested: ANY_OF containing ALL_OF (TRUE)")
    void testNestedConditions_AnyOfContainingAllOf() {
        // (lactate >= 2.0 AND systolic_bp < 90) OR age > 80
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ANY_OF);

        // Nested ALL_OF (will be true)
        ProtocolCondition nestedCond = new ProtocolCondition();
        nestedCond.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition nested1 = new ProtocolCondition();
        nested1.setParameter("lactate");
        nested1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        nested1.setThreshold(2.0);

        ProtocolCondition nested2 = new ProtocolCondition();
        nested2.setParameter("systolic_bp");
        nested2.setOperator(ComparisonOperator.LESS_THAN);
        nested2.setThreshold(90);

        nestedCond.setConditions(Arrays.asList(nested1, nested2));

        // Second condition (will be false)
        ProtocolCondition cond2 = new ProtocolCondition();
        cond2.setParameter("age");
        cond2.setOperator(ComparisonOperator.GREATER_THAN);
        cond2.setThreshold(80); // Age is 72, will fail

        trigger.setConditions(Arrays.asList(nestedCond, cond2));

        boolean result = evaluator.evaluate(trigger, context);
        assertTrue(result, "Nested ALL_OF should pass, making ANY_OF true");
    }

    @Test
    @DisplayName("Test nested: 3 levels deep (TRUE)")
    void testNestedConditions_ThreeLevelsDeep() {
        // Level 1: ALL_OF
        //   Level 2: ANY_OF
        //     Level 3: ALL_OF (lactate >= 2.0 AND systolic_bp < 90)
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        // Level 2: ANY_OF
        ProtocolCondition level2 = new ProtocolCondition();
        level2.setMatchLogic(MatchLogic.ANY_OF);

        // Level 3: ALL_OF
        ProtocolCondition level3 = new ProtocolCondition();
        level3.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition deepCond1 = new ProtocolCondition();
        deepCond1.setParameter("lactate");
        deepCond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        deepCond1.setThreshold(2.0);

        ProtocolCondition deepCond2 = new ProtocolCondition();
        deepCond2.setParameter("systolic_bp");
        deepCond2.setOperator(ComparisonOperator.LESS_THAN);
        deepCond2.setThreshold(90);

        level3.setConditions(Arrays.asList(deepCond1, deepCond2));
        level2.setConditions(Arrays.asList(level3));
        trigger.setConditions(Arrays.asList(level2));

        boolean result = evaluator.evaluate(trigger, context);
        assertTrue(result, "3-level nested condition should evaluate correctly");
    }

    @Test
    @DisplayName("Test nested: nested condition fails (FALSE)")
    void testNestedConditions_NestedFails() {
        // ALL_OF: (lactate >= 2.0) AND (systolic_bp < 70 OR age > 80) - nested will fail
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        ProtocolCondition cond1 = new ProtocolCondition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        // Nested ANY_OF (both will fail)
        ProtocolCondition nestedCond = new ProtocolCondition();
        nestedCond.setMatchLogic(MatchLogic.ANY_OF);

        ProtocolCondition nested1 = new ProtocolCondition();
        nested1.setParameter("systolic_bp");
        nested1.setOperator(ComparisonOperator.LESS_THAN);
        nested1.setThreshold(70); // Will fail

        ProtocolCondition nested2 = new ProtocolCondition();
        nested2.setParameter("age");
        nested2.setOperator(ComparisonOperator.GREATER_THAN);
        nested2.setThreshold(80); // Will fail

        nestedCond.setConditions(Arrays.asList(nested1, nested2));
        trigger.setConditions(Arrays.asList(cond1, nestedCond));

        boolean result = evaluator.evaluate(trigger, context);
        assertFalse(result, "Nested condition fails, ALL_OF should fail");
    }

    // ========== Operator Tests (6 tests) ==========

    @Test
    @DisplayName("Test all numeric operators work correctly")
    void testAllNumericOperators() {
        double actualValue = 85.0;
        double lowerThreshold = 70.0;
        double upperThreshold = 90.0;

        assertTrue(evaluator.compareValues(actualValue, lowerThreshold, ComparisonOperator.GREATER_THAN));
        assertTrue(evaluator.compareValues(actualValue, lowerThreshold, ComparisonOperator.GREATER_THAN_OR_EQUAL));
        assertTrue(evaluator.compareValues(actualValue, upperThreshold, ComparisonOperator.LESS_THAN));
        assertTrue(evaluator.compareValues(actualValue, upperThreshold, ComparisonOperator.LESS_THAN_OR_EQUAL));
        assertTrue(evaluator.compareValues(actualValue, 85.0, ComparisonOperator.EQUAL));
        assertTrue(evaluator.compareValues(actualValue, lowerThreshold, ComparisonOperator.NOT_EQUAL));
    }

    @Test
    @DisplayName("Test CONTAINS operator is case-insensitive")
    void testContainsCaseInsensitive() {
        assertTrue(evaluator.compareValues("PENICILLIN", "penicillin", ComparisonOperator.CONTAINS));
        assertTrue(evaluator.compareValues("penicillin", "PENICILLIN", ComparisonOperator.CONTAINS));
        assertTrue(evaluator.compareValues("Penicillin Allergy", "cillin", ComparisonOperator.CONTAINS));
    }

    @Test
    @DisplayName("Test EQUAL operator with different types")
    void testEqualOperatorDifferentTypes() {
        // Numeric equality
        assertTrue(evaluator.compareValues(85, 85.0, ComparisonOperator.EQUAL));
        assertTrue(evaluator.compareValues(85.0001, 85.0000, ComparisonOperator.EQUAL)); // Within tolerance

        // Boolean equality
        assertTrue(evaluator.compareValues(true, true, ComparisonOperator.EQUAL));
        assertFalse(evaluator.compareValues(true, false, ComparisonOperator.EQUAL));

        // String equality (case-insensitive)
        assertTrue(evaluator.compareValues("MALE", "male", ComparisonOperator.EQUAL));
    }

    @Test
    @DisplayName("Test NOT_EQUAL operator")
    void testNotEqualOperator() {
        assertTrue(evaluator.compareValues(85, 70, ComparisonOperator.NOT_EQUAL));
        assertFalse(evaluator.compareValues(85, 85, ComparisonOperator.NOT_EQUAL));
    }

    @Test
    @DisplayName("Test operators with null values")
    void testOperatorsWithNullValues() {
        // Null converts to 0 for numeric comparisons
        assertTrue(evaluator.compareValues(null, -1, ComparisonOperator.GREATER_THAN));
        assertTrue(evaluator.compareValues(5, null, ComparisonOperator.GREATER_THAN));

        // Null converts to empty string for string operations
        assertFalse(evaluator.compareValues(null, "test", ComparisonOperator.CONTAINS));
    }

    // ========== Parameter Extraction Tests (5 tests) ==========

    @Test
    @DisplayName("Test parameter extraction: vital signs")
    void testParameterExtraction_VitalSigns() {
        assertEquals(85.0, evaluator.extractParameterValue("systolic_bp", context));
        assertEquals(85.0, evaluator.extractParameterValue("systolic_blood_pressure", context)); // Alias
        assertEquals(60.0, evaluator.extractParameterValue("map", context));
        assertEquals(60.0, evaluator.extractParameterValue("mean_arterial_pressure", context)); // Alias
        assertEquals(98.0, evaluator.extractParameterValue("heart_rate", context));
    }

    @Test
    @DisplayName("Test parameter extraction: lab values")
    void testParameterExtraction_LabValues() {
        assertEquals(3.5, evaluator.extractParameterValue("lactate", context));
        assertEquals(15000.0, evaluator.extractParameterValue("wbc", context));
        assertEquals(15000.0, evaluator.extractParameterValue("white_blood_count", context)); // Alias
    }

    @Test
    @DisplayName("Test parameter extraction: demographics")
    void testParameterExtraction_Demographics() {
        assertEquals(72, evaluator.extractParameterValue("age", context));
        assertEquals("M", evaluator.extractParameterValue("sex", context));
        assertEquals("M", evaluator.extractParameterValue("gender", context)); // Alias
    }

    @Test
    @DisplayName("Test parameter extraction: clinical assessments")
    void testParameterExtraction_ClinicalAssessments() {
        String allergies = (String) evaluator.extractParameterValue("allergies", context);
        assertTrue(allergies.contains("penicillin"));
        assertTrue(allergies.contains("iodine"));

        assertEquals(true, evaluator.extractParameterValue("infection_suspected", context));
    }

    @Test
    @DisplayName("Test parameter extraction: non-existent parameter returns null")
    void testParameterExtraction_NonExistentReturnsNull() {
        assertNull(evaluator.extractParameterValue("non_existent_param", context));
        assertNull(evaluator.extractParameterValue("", context));
        assertNull(evaluator.extractParameterValue(null, context));
    }

    // ========== Edge Cases and Error Handling ==========

    @Test
    @DisplayName("Test null trigger throws exception")
    void testNullTriggerThrowsException() {
        assertThrows(IllegalArgumentException.class, () -> {
            evaluator.evaluate(null, context);
        });
    }

    @Test
    @DisplayName("Test null context throws exception")
    void testNullContextThrowsException() {
        TriggerCriteria trigger = new TriggerCriteria();
        assertThrows(IllegalArgumentException.class, () -> {
            evaluator.evaluate(trigger, null);
        });
    }

    @Test
    @DisplayName("Test parameter not found returns false")
    void testParameterNotFound_ReturnsFalse() {
        ProtocolCondition condition = new ProtocolCondition();
        condition.setParameter("non_existent_parameter");
        condition.setOperator(ComparisonOperator.EQUAL);
        condition.setThreshold(10);

        boolean result = evaluator.evaluateCondition(condition, context, 0);
        assertFalse(result, "Non-existent parameter should return false");
    }

    // ========== Helper Methods ==========

    private EnrichedPatientContext createTestContext() {
        EnrichedPatientContext ctx = new EnrichedPatientContext();
        PatientState state = new PatientState("TEST-PATIENT-001");

        // Set vitals
        state.setSystolicBP(85.0);
        state.setDiastolicBP(55.0);
        state.setMeanArterialPressure(60.0);
        state.setHeartRate(98.0);
        state.setRespiratoryRate(22.0);
        state.setTemperature(38.5);
        state.setOxygenSaturation(94.0);

        // Set labs
        state.setLactate(3.5);
        state.setWhiteBloodCount(15000.0);
        state.setCreatinine(1.5);
        state.setGlucose(180.0);

        // Set demographics
        state.setAge(72);
        state.setSex("M");
        state.setWeight(80.0);

        // Set clinical assessments
        state.setInfectionSuspected(true);
        state.setAllergies(Arrays.asList("penicillin", "iodine"));

        // Set scores
        state.setNews2Score(7);
        state.setQsofaScore(2);
        state.setCombinedAcuityScore(6.5);

        ctx.setPatientState(state);
        ctx.setPatientId("TEST-PATIENT-001");
        ctx.setEventType("VITAL_SIGN");
        ctx.setEventTime(System.currentTimeMillis());

        return ctx;
    }
}
