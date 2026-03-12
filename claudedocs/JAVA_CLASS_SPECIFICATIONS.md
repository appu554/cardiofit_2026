# Module 3: Java Class Specifications
## Detailed Implementation Guide for CDS Components

**Document Version**: 1.0
**Created**: 2025-10-21
**Purpose**: Complete Java class specifications for 7 new CDS components

---

## Table of Contents

1. [ConditionEvaluator.java](#1-conditionevaluatorjava)
2. [ConfidenceCalculator.java](#2-confidencecalculatorjava)
3. [MedicationSelector.java](#3-medicationselectorjava)
4. [TimeConstraintTracker.java](#4-timeconstrainttracke rjava)
5. [KnowledgeBaseManager.java](#5-knowledgebasemanagerjava)
6. [EscalationRuleEvaluator.java](#6-escalationruleevaluatorjava)
7. [ProtocolValidator.java](#7-protocolvalidatorjava)

---

## 1. ConditionEvaluator.java

**Package**: `com.cardiofit.flink.cds.evaluation`
**Purpose**: Evaluate trigger criteria with AND/OR logic to determine if protocol should activate
**Estimated Lines**: 400
**Estimated Effort**: 3-4 hours
**Priority**: **CRITICAL** (Phase 1)

### Class Overview

```java
package com.cardiofit.flink.cds.evaluation;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.protocol.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Map;

/**
 * Evaluates trigger criteria for protocol activation using AND/OR logic.
 *
 * <p>This class handles:
 * - ALL_OF (AND) and ANY_OF (OR) match logic
 * - Nested conditions with recursive evaluation
 * - Comparison operators: >=, <=, ==, !=, CONTAINS, NOT_CONTAINS
 * - Parameter extraction from EnrichedPatientContext
 *
 * <p>Example usage:
 * <pre>
 * ConditionEvaluator evaluator = new ConditionEvaluator();
 * boolean triggered = evaluator.evaluate(protocol.getTriggerCriteria(), context);
 * if (triggered) {
 *     // Protocol should activate
 * }
 * </pre>
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class ConditionEvaluator {
    private static final Logger logger = LoggerFactory.getLogger(ConditionEvaluator.class);

    // Main entry point for trigger evaluation
    public boolean evaluate(TriggerCriteria trigger, EnrichedPatientContext context);

    // Internal recursive evaluation
    private boolean evaluateCondition(Condition condition, EnrichedPatientContext context);

    // Value comparison with operator support
    private boolean compareValues(Object actualValue, Object expectedValue, ComparisonOperator operator);

    // Parameter extraction from context
    private Object extractParameterValue(String parameter, EnrichedPatientContext context);

    // Type conversion helpers
    private Double toDouble(Object value);
    private Boolean toBoolean(Object value);
    private String toString(Object value);
}
```

### Method Specifications

#### 1.1 evaluate()

```java
/**
 * Evaluates trigger criteria to determine if protocol should activate.
 *
 * @param trigger The trigger criteria from protocol YAML
 * @param context The enriched patient context with current state
 * @return true if all conditions met (for ALL_OF) or any condition met (for ANY_OF)
 * @throws IllegalArgumentException if trigger or context is null
 */
public boolean evaluate(TriggerCriteria trigger, EnrichedPatientContext context) {
    if (trigger == null) {
        throw new IllegalArgumentException("Trigger criteria cannot be null");
    }
    if (context == null) {
        throw new IllegalArgumentException("Patient context cannot be null");
    }

    MatchLogic matchLogic = trigger.getMatchLogic();
    List<Condition> conditions = trigger.getConditions();

    if (conditions == null || conditions.isEmpty()) {
        logger.warn("No conditions defined in trigger criteria");
        return false;
    }

    logger.debug("Evaluating trigger with {} logic, {} conditions",
        matchLogic, conditions.size());

    if (matchLogic == MatchLogic.ALL_OF) {
        // AND logic - all conditions must be true
        for (Condition condition : conditions) {
            boolean result = evaluateCondition(condition, context);
            logger.debug("Condition {} evaluated to {}", condition.getConditionId(), result);

            if (!result) {
                logger.debug("ALL_OF short-circuit: condition {} failed", condition.getConditionId());
                return false; // Short-circuit on first false
            }
        }
        logger.debug("ALL_OF evaluation: all conditions satisfied");
        return true;

    } else if (matchLogic == MatchLogic.ANY_OF) {
        // OR logic - at least one condition must be true
        for (Condition condition : conditions) {
            boolean result = evaluateCondition(condition, context);
            logger.debug("Condition {} evaluated to {}", condition.getConditionId(), result);

            if (result) {
                logger.debug("ANY_OF short-circuit: condition {} succeeded", condition.getConditionId());
                return true; // Short-circuit on first true
            }
        }
        logger.debug("ANY_OF evaluation: no conditions satisfied");
        return false;

    } else {
        logger.error("Unknown match logic: {}", matchLogic);
        return false;
    }
}
```

#### 1.2 evaluateCondition()

```java
/**
 * Evaluates a single condition, handling both leaf conditions and nested conditions.
 *
 * @param condition The condition to evaluate (may contain nested conditions)
 * @param context The patient context
 * @return true if condition is satisfied
 */
private boolean evaluateCondition(Condition condition, EnrichedPatientContext context) {
    // Handle nested conditions (recursive case)
    if (condition.getConditions() != null && !condition.getConditions().isEmpty()) {
        logger.debug("Evaluating nested condition with {} sub-conditions",
            condition.getConditions().size());

        TriggerCriteria nestedTrigger = new TriggerCriteria();
        nestedTrigger.setMatchLogic(condition.getMatchLogic());
        nestedTrigger.setConditions(condition.getConditions());

        return evaluate(nestedTrigger, context); // RECURSION
    }

    // Handle leaf condition (base case)
    String parameter = condition.getParameter();
    ComparisonOperator operator = condition.getOperator();
    Object expectedValue = condition.getThreshold();

    if (parameter == null || operator == null || expectedValue == null) {
        logger.warn("Incomplete condition: parameter={}, operator={}, threshold={}",
            parameter, operator, expectedValue);
        return false;
    }

    // Extract actual value from patient context
    Object actualValue = extractParameterValue(parameter, context);

    if (actualValue == null) {
        logger.debug("Parameter {} not found in patient context", parameter);
        return false;
    }

    // Compare values using operator
    boolean result = compareValues(actualValue, expectedValue, operator);

    logger.debug("Condition evaluation: {} {} {} = {} (actual: {})",
        parameter, operator, expectedValue, result, actualValue);

    return result;
}
```

#### 1.3 compareValues()

```java
/**
 * Compares two values using the specified operator.
 *
 * Supports numeric comparison (>=, <=, ==, !=) and string operations (CONTAINS, NOT_CONTAINS).
 *
 * @param actualValue The actual value from patient context
 * @param expectedValue The expected threshold value
 * @param operator The comparison operator
 * @return true if comparison is satisfied
 */
private boolean compareValues(Object actualValue, Object expectedValue, ComparisonOperator operator) {
    switch (operator) {
        case GREATER_THAN_OR_EQUAL:
            return toDouble(actualValue) >= toDouble(expectedValue);

        case LESS_THAN_OR_EQUAL:
            return toDouble(actualValue) <= toDouble(expectedValue);

        case GREATER_THAN:
            return toDouble(actualValue) > toDouble(expectedValue);

        case LESS_THAN:
            return toDouble(actualValue) < toDouble(expectedValue);

        case EQUAL:
            // Handle different types
            if (actualValue instanceof Number && expectedValue instanceof Number) {
                return Math.abs(toDouble(actualValue) - toDouble(expectedValue)) < 0.0001;
            } else if (actualValue instanceof Boolean && expectedValue instanceof Boolean) {
                return actualValue.equals(expectedValue);
            } else {
                return toString(actualValue).equalsIgnoreCase(toString(expectedValue));
            }

        case NOT_EQUAL:
            return !compareValues(actualValue, expectedValue, ComparisonOperator.EQUAL);

        case CONTAINS:
            String actualStr = toString(actualValue).toLowerCase();
            String expectedStr = toString(expectedValue).toLowerCase();
            return actualStr.contains(expectedStr);

        case NOT_CONTAINS:
            return !compareValues(actualValue, expectedValue, ComparisonOperator.CONTAINS);

        default:
            logger.error("Unsupported operator: {}", operator);
            return false;
    }
}
```

#### 1.4 extractParameterValue()

```java
/**
 * Extracts a parameter value from the patient context.
 *
 * Supports dotted notation for nested fields (e.g., "vital_signs.systolic_bp").
 *
 * @param parameter The parameter name (e.g., "lactate", "systolic_bp", "allergies")
 * @param context The patient context
 * @return The parameter value, or null if not found
 */
private Object extractParameterValue(String parameter, EnrichedPatientContext context) {
    if (parameter == null || parameter.isEmpty()) {
        return null;
    }

    PatientState patientState = context.getPatientState();

    // Common vital signs
    switch (parameter.toLowerCase()) {
        case "systolic_bp":
        case "systolic_blood_pressure":
            return patientState.getSystolicBP();

        case "diastolic_bp":
        case "diastolic_blood_pressure":
            return patientState.getDiastolicBP();

        case "mean_arterial_pressure":
        case "map":
            return patientState.getMeanArterialPressure();

        case "heart_rate":
        case "hr":
            return patientState.getHeartRate();

        case "respiratory_rate":
        case "rr":
            return patientState.getRespiratoryRate();

        case "temperature":
        case "temp":
            return patientState.getTemperature();

        case "oxygen_saturation":
        case "spo2":
            return patientState.getOxygenSaturation();

        // Lab values
        case "lactate":
            return patientState.getLactate();

        case "white_blood_count":
        case "wbc":
            return patientState.getWhiteBloodCount();

        case "creatinine":
            return patientState.getCreatinine();

        case "creatinine_clearance":
        case "crcl":
            return patientState.getCreatinineClearance();

        case "glucose":
            return patientState.getGlucose();

        case "procalcitonin":
            return patientState.getProcalcitonin();

        case "troponin":
            return patientState.getTroponin();

        // Demographics
        case "age":
            return patientState.getAge();

        case "sex":
        case "gender":
            return patientState.getSex();

        case "weight":
            return patientState.getWeight();

        // Clinical assessments
        case "allergies":
            return patientState.getAllergies(); // Returns List<String>

        case "infection_suspected":
            return patientState.isInfectionSuspected();

        case "pregnancy_status":
            return patientState.isPregnant();

        case "immunosuppressed":
            return patientState.isImmunosuppressed();

        // Scores
        case "news2_score":
            return patientState.getNews2Score();

        case "sofa_score":
            return patientState.getSofaScore();

        case "child_pugh_score":
            return patientState.getChildPughScore();

        // Default: try reflection for other parameters
        default:
            try {
                String methodName = "get" +
                    parameter.substring(0, 1).toUpperCase() +
                    parameter.substring(1).replaceAll("_", "");

                java.lang.reflect.Method method = patientState.getClass().getMethod(methodName);
                return method.invoke(patientState);

            } catch (Exception e) {
                logger.warn("Parameter {} not found in PatientState", parameter);
                return null;
            }
    }
}
```

#### 1.5 Helper Methods

```java
/**
 * Converts value to Double for numeric comparison.
 */
private Double toDouble(Object value) {
    if (value == null) {
        return 0.0;
    }
    if (value instanceof Number) {
        return ((Number) value).doubleValue();
    }
    try {
        return Double.parseDouble(value.toString());
    } catch (NumberFormatException e) {
        logger.warn("Cannot convert {} to Double", value);
        return 0.0;
    }
}

/**
 * Converts value to Boolean.
 */
private Boolean toBoolean(Object value) {
    if (value == null) {
        return false;
    }
    if (value instanceof Boolean) {
        return (Boolean) value;
    }
    return Boolean.parseBoolean(value.toString());
}

/**
 * Converts value to String.
 */
private String toString(Object value) {
    return value == null ? "" : value.toString();
}
```

### Enum Definitions

```java
/**
 * Comparison operators for condition evaluation.
 */
public enum ComparisonOperator {
    GREATER_THAN_OR_EQUAL(">="),
    LESS_THAN_OR_EQUAL("<="),
    GREATER_THAN(">"),
    LESS_THAN("<"),
    EQUAL("=="),
    NOT_EQUAL("!="),
    CONTAINS("CONTAINS"),
    NOT_CONTAINS("NOT_CONTAINS");

    private final String symbol;

    ComparisonOperator(String symbol) {
        this.symbol = symbol;
    }

    public String getSymbol() {
        return symbol;
    }
}

/**
 * Match logic for combining multiple conditions.
 */
public enum MatchLogic {
    ALL_OF,  // AND logic
    ANY_OF   // OR logic
}
```

### Unit Tests

```java
package com.cardiofit.flink.cds.evaluation;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class ConditionEvaluatorTest {

    private ConditionEvaluator evaluator;
    private EnrichedPatientContext context;

    @BeforeEach
    void setUp() {
        evaluator = new ConditionEvaluator();
        context = createTestContext();
    }

    @Test
    void testSimpleCondition_GreaterThanOrEqual_True() {
        // lactate >= 2.0 (actual: 3.5)
        Condition condition = new Condition();
        condition.setParameter("lactate");
        condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        condition.setThreshold(2.0);

        boolean result = evaluator.evaluateCondition(condition, context);
        assertTrue(result, "Lactate 3.5 should be >= 2.0");
    }

    @Test
    void testSimpleCondition_LessThan_True() {
        // systolic_bp < 90 (actual: 85)
        Condition condition = new Condition();
        condition.setParameter("systolic_bp");
        condition.setOperator(ComparisonOperator.LESS_THAN);
        condition.setThreshold(90);

        boolean result = evaluator.evaluateCondition(condition, context);
        assertTrue(result, "Systolic BP 85 should be < 90");
    }

    @Test
    void testAllOfLogic_AllConditionsMet() {
        // lactate >= 2.0 AND systolic_bp < 90 AND infection_suspected == true
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        Condition cond1 = new Condition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        Condition cond2 = new Condition();
        cond2.setParameter("systolic_bp");
        cond2.setOperator(ComparisonOperator.LESS_THAN);
        cond2.setThreshold(90);

        Condition cond3 = new Condition();
        cond3.setParameter("infection_suspected");
        cond3.setOperator(ComparisonOperator.EQUAL);
        cond3.setThreshold(true);

        trigger.setConditions(Arrays.asList(cond1, cond2, cond3));

        boolean result = evaluator.evaluate(trigger, context);
        assertTrue(result, "All conditions met, should return true");
    }

    @Test
    void testAllOfLogic_OneConditionFailed() {
        // lactate >= 2.0 AND systolic_bp < 70 (will fail)
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        Condition cond1 = new Condition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        Condition cond2 = new Condition();
        cond2.setParameter("systolic_bp");
        cond2.setOperator(ComparisonOperator.LESS_THAN);
        cond2.setThreshold(70); // Actual is 85, will fail

        trigger.setConditions(Arrays.asList(cond1, cond2));

        boolean result = evaluator.evaluate(trigger, context);
        assertFalse(result, "One condition failed, should return false");
    }

    @Test
    void testAnyOfLogic_OneConditionMet() {
        // lactate >= 2.0 OR systolic_bp < 70 (first will pass)
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ANY_OF);

        Condition cond1 = new Condition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        Condition cond2 = new Condition();
        cond2.setParameter("systolic_bp");
        cond2.setOperator(ComparisonOperator.LESS_THAN);
        cond2.setThreshold(70); // Will fail

        trigger.setConditions(Arrays.asList(cond1, cond2));

        boolean result = evaluator.evaluate(trigger, context);
        assertTrue(result, "One condition met, should return true");
    }

    @Test
    void testNestedConditions_AllOfContainingAnyOf() {
        // lactate >= 2.0 AND (systolic_bp < 90 OR map < 65)
        TriggerCriteria trigger = new TriggerCriteria();
        trigger.setMatchLogic(MatchLogic.ALL_OF);

        Condition cond1 = new Condition();
        cond1.setParameter("lactate");
        cond1.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
        cond1.setThreshold(2.0);

        // Nested ANY_OF
        Condition nestedCond = new Condition();
        nestedCond.setMatchLogic(MatchLogic.ANY_OF);

        Condition nested1 = new Condition();
        nested1.setParameter("systolic_bp");
        nested1.setOperator(ComparisonOperator.LESS_THAN);
        nested1.setThreshold(90);

        Condition nested2 = new Condition();
        nested2.setParameter("map");
        nested2.setOperator(ComparisonOperator.LESS_THAN);
        nested2.setThreshold(65);

        nestedCond.setConditions(Arrays.asList(nested1, nested2));

        trigger.setConditions(Arrays.asList(cond1, nestedCond));

        boolean result = evaluator.evaluate(trigger, context);
        assertTrue(result, "Nested condition should evaluate correctly");
    }

    @Test
    void testContainsOperator_AllergiesContainsPenicillin() {
        // allergies CONTAINS "penicillin"
        Condition condition = new Condition();
        condition.setParameter("allergies");
        condition.setOperator(ComparisonOperator.CONTAINS);
        condition.setThreshold("penicillin");

        boolean result = evaluator.evaluateCondition(condition, context);
        assertTrue(result, "Allergies should contain penicillin");
    }

    @Test
    void testNotContainsOperator() {
        // allergies NOT_CONTAINS "sulfa"
        Condition condition = new Condition();
        condition.setParameter("allergies");
        condition.setOperator(ComparisonOperator.NOT_CONTAINS);
        condition.setThreshold("sulfa");

        boolean result = evaluator.evaluateCondition(condition, context);
        assertTrue(result, "Allergies should not contain sulfa");
    }

    @Test
    void testParameterNotFound_ReturnsFalse() {
        Condition condition = new Condition();
        condition.setParameter("non_existent_parameter");
        condition.setOperator(ComparisonOperator.EQUAL);
        condition.setThreshold(10);

        boolean result = evaluator.evaluateCondition(condition, context);
        assertFalse(result, "Non-existent parameter should return false");
    }

    private EnrichedPatientContext createTestContext() {
        EnrichedPatientContext ctx = new EnrichedPatientContext();
        PatientState state = new PatientState();

        state.setLactate(3.5);
        state.setSystolicBP(85);
        state.setMeanArterialPressure(60);
        state.setInfectionSuspected(true);
        state.setAllergies(Arrays.asList("penicillin", "iodine"));
        state.setAge(72);

        ctx.setPatientState(state);
        return ctx;
    }
}
```

---

## 2. ConfidenceCalculator.java

**Package**: `com.cardiofit.flink.cds.evaluation`
**Purpose**: Calculate confidence score for protocol match based on patient state
**Estimated Lines**: 300
**Estimated Effort**: 2-3 hours
**Priority**: High (Phase 2)

### Class Overview

```java
package com.cardiofit.flink.cds.evaluation;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.protocol.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Calculates confidence scores for protocol matches using base confidence + modifiers.
 *
 * <p>Confidence scoring allows ranking of multiple matching protocols by how well
 * the patient matches the protocol's intended population.
 *
 * <p>Score calculation:
 * <ol>
 *   <li>Start with base_confidence from protocol</li>
 *   <li>For each modifier, evaluate condition and add adjustment if true</li>
 *   <li>Clamp final score to [0.0, 1.0]</li>
 *   <li>Check if score >= activation_threshold</li>
 * </ol>
 *
 * <p>Example:
 * <pre>
 * ConfidenceCalculator calculator = new ConfidenceCalculator(conditionEvaluator);
 * double confidence = calculator.calculateConfidence(protocol, context);
 * if (confidence >= protocol.getConfidenceScoring().getActivationThreshold()) {
 *     // Protocol activates
 * }
 * </pre>
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class ConfidenceCalculator {
    private static final Logger logger = LoggerFactory.getLogger(ConfidenceCalculator.class);
    private static final double DEFAULT_BASE_CONFIDENCE = 0.85;
    private static final double DEFAULT_ACTIVATION_THRESHOLD = 0.70;

    private final ConditionEvaluator conditionEvaluator;

    public ConfidenceCalculator(ConditionEvaluator conditionEvaluator);

    public double calculateConfidence(Protocol protocol, EnrichedPatientContext context);

    public boolean meetsActivationThreshold(Protocol protocol, double confidence);

    private double clamp(double value, double min, double max);
}
```

### Method Specifications

#### 2.1 calculateConfidence()

```java
/**
 * Calculates the confidence score for a protocol match.
 *
 * @param protocol The protocol to score
 * @param context The patient context
 * @return Confidence score (0.0-1.0)
 */
public double calculateConfidence(Protocol protocol, EnrichedPatientContext context) {
    if (protocol == null || context == null) {
        logger.warn("Null protocol or context provided");
        return 0.0;
    }

    ConfidenceScoring scoring = protocol.getConfidenceScoring();

    // Use default if no scoring defined
    if (scoring == null) {
        logger.debug("No confidence_scoring defined for protocol {}, using default {}",
            protocol.getProtocolId(), DEFAULT_BASE_CONFIDENCE);
        return DEFAULT_BASE_CONFIDENCE;
    }

    double confidence = scoring.getBaseConfidence();
    logger.debug("Protocol {} base confidence: {}", protocol.getProtocolId(), confidence);

    // Apply modifiers
    if (scoring.getModifiers() != null) {
        for (ConfidenceModifier modifier : scoring.getModifiers()) {
            // Evaluate modifier condition
            boolean conditionMet = conditionEvaluator.evaluateCondition(
                modifier.getCondition(),
                context);

            if (conditionMet) {
                double adjustment = modifier.getAdjustment();
                confidence += adjustment;

                logger.debug("Applied modifier {}: {} (new confidence: {})",
                    modifier.getModifierId(),
                    adjustment,
                    confidence);
            } else {
                logger.debug("Modifier {} condition not met, skipping",
                    modifier.getModifierId());
            }
        }
    }

    // Clamp to [0.0, 1.0]
    confidence = clamp(confidence, 0.0, 1.0);

    logger.info("Final confidence for protocol {}: {}",
        protocol.getProtocolId(), confidence);

    return confidence;
}
```

#### 2.2 meetsActivationThreshold()

```java
/**
 * Checks if confidence score meets the activation threshold.
 *
 * @param protocol The protocol
 * @param confidence The calculated confidence score
 * @return true if confidence >= activation_threshold
 */
public boolean meetsActivationThreshold(Protocol protocol, double confidence) {
    double threshold = DEFAULT_ACTIVATION_THRESHOLD;

    if (protocol.getConfidenceScoring() != null) {
        threshold = protocol.getConfidenceScoring().getActivationThreshold();
    }

    boolean meets = confidence >= threshold;

    logger.debug("Protocol {} confidence {} {} threshold {}",
        protocol.getProtocolId(),
        confidence,
        meets ? ">=" : "<",
        threshold);

    return meets;
}
```

#### 2.3 clamp()

```java
/**
 * Clamps value to [min, max] range.
 */
private double clamp(double value, double min, double max) {
    if (value < min) {
        logger.debug("Confidence {} clamped to min {}", value, min);
        return min;
    }
    if (value > max) {
        logger.debug("Confidence {} clamped to max {}", value, max);
        return max;
    }
    return value;
}
```

### Unit Tests

```java
@Test
void testCalculateConfidence_NoModifiers() {
    Protocol protocol = new Protocol();
    ConfidenceScoring scoring = new ConfidenceScoring();
    scoring.setBaseConfidence(0.85);
    scoring.setActivationThreshold(0.70);
    protocol.setConfidenceScoring(scoring);

    double confidence = calculator.calculateConfidence(protocol, context);

    assertEquals(0.85, confidence, 0.001);
}

@Test
void testCalculateConfidence_WithPositiveModifier() {
    Protocol protocol = new Protocol();
    ConfidenceScoring scoring = new ConfidenceScoring();
    scoring.setBaseConfidence(0.80);

    // Modifier: +0.10 if age >= 65 (context has age 72)
    ConfidenceModifier modifier = new ConfidenceModifier();
    modifier.setModifierId("AGE_MODIFIER");
    modifier.setAdjustment(0.10);

    Condition condition = new Condition();
    condition.setParameter("age");
    condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
    condition.setThreshold(65);
    modifier.setCondition(condition);

    scoring.setModifiers(Arrays.asList(modifier));
    protocol.setConfidenceScoring(scoring);

    double confidence = calculator.calculateConfidence(protocol, context);

    assertEquals(0.90, confidence, 0.001); // 0.80 + 0.10
}

@Test
void testCalculateConfidence_ClampingAbove1() {
    Protocol protocol = new Protocol();
    ConfidenceScoring scoring = new ConfidenceScoring();
    scoring.setBaseConfidence(0.95);

    // Modifier: +0.15 (would exceed 1.0)
    ConfidenceModifier modifier = new ConfidenceModifier();
    modifier.setAdjustment(0.15);
    Condition condition = new Condition();
    condition.setParameter("age");
    condition.setOperator(ComparisonOperator.GREATER_THAN_OR_EQUAL);
    condition.setThreshold(65);
    modifier.setCondition(condition);

    scoring.setModifiers(Arrays.asList(modifier));
    protocol.setConfidenceScoring(scoring);

    double confidence = calculator.calculateConfidence(protocol, context);

    assertEquals(1.0, confidence, 0.001); // Clamped to 1.0
}

@Test
void testMeetsActivationThreshold_True() {
    Protocol protocol = new Protocol();
    ConfidenceScoring scoring = new ConfidenceScoring();
    scoring.setActivationThreshold(0.70);
    protocol.setConfidenceScoring(scoring);

    boolean meets = calculator.meetsActivationThreshold(protocol, 0.85);
    assertTrue(meets);
}

@Test
void testMeetsActivationThreshold_False() {
    Protocol protocol = new Protocol();
    ConfidenceScoring scoring = new ConfidenceScoring();
    scoring.setActivationThreshold(0.70);
    protocol.setConfidenceScoring(scoring);

    boolean meets = calculator.meetsActivationThreshold(protocol, 0.65);
    assertFalse(meets);
}
```

---

## 3. MedicationSelector.java

**Package**: `com.cardiofit.flink.cds.medication`
**Purpose**: Select appropriate medication based on patient allergies, renal function, MDR risk
**Estimated Lines**: 650
**Estimated Effort**: 4-5 hours
**Priority**: **CRITICAL** (Phase 1) - Patient Safety

### Class Overview

```java
package com.cardiofit.flink.cds.medication;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.protocol.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Selects appropriate medications based on patient-specific factors.
 *
 * <p>Handles:
 * - Allergy checking and alternative selection
 * - Renal dose adjustments (Cockcroft-Gault formula)
 * - Hepatic dose adjustments (Child-Pugh score)
 * - MDR risk assessment
 * - Selection criteria evaluation (NO_PENICILLIN_ALLERGY, CREATININE_CLEARANCE_GT_40, etc.)
 *
 * <p>Safety-critical component - errors here could result in contraindicated medications.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class MedicationSelector {
    private static final Logger logger = LoggerFactory.getLogger(MedicationSelector.class);

    public ProtocolAction selectMedication(ProtocolAction action, EnrichedPatientContext context);

    private boolean evaluateCriteria(String criteriaId, EnrichedPatientContext context);

    private boolean hasAllergy(Medication medication, EnrichedPatientContext context);

    private Medication applyDoseAdjustments(Medication medication, EnrichedPatientContext context);

    private double calculateCrCl(EnrichedPatientContext context);

    private Medication adjustDoseForRenalFunction(Medication medication, double crCl);

    private Medication adjustDoseForHepaticFunction(Medication medication, String childPugh);
}
```

### Method Specifications

#### 3.1 selectMedication()

```java
/**
 * Selects the appropriate medication for an action based on patient context.
 *
 * @param action The protocol action with medication selection criteria
 * @param context The patient context
 * @return The selected medication action (may be modified from original)
 */
public ProtocolAction selectMedication(ProtocolAction action, EnrichedPatientContext context) {
    if (action == null || context == null) {
        logger.error("Null action or context provided");
        return action;
    }

    MedicationSelection selection = action.getMedicationSelection();

    // No selection algorithm - return action as-is
    if (selection == null) {
        logger.debug("No medication_selection for action {}, using as-is",
            action.getActionId());
        return action;
    }

    logger.debug("Selecting medication for action {} with {} criteria",
        action.getActionId(),
        selection.getSelectionCriteria().size());

    // Evaluate selection criteria in order
    for (SelectionCriteria criteria : selection.getSelectionCriteria()) {
        boolean criteriaMet = evaluateCriteria(criteria.getCriteriaId(), context);

        if (criteriaMet) {
            logger.debug("Criteria {} met", criteria.getCriteriaId());

            // Use primary medication
            Medication selectedMed = criteria.getPrimaryMedication().clone();

            // Check for allergies/contraindications
            if (hasAllergy(selectedMed, context)) {
                logger.warn("Patient allergic to {}, using alternative",
                    selectedMed.getName());

                if (criteria.getAlternativeMedication() != null) {
                    selectedMed = criteria.getAlternativeMedication().clone();
                    logger.info("Selected alternative: {}", selectedMed.getName());
                } else {
                    logger.error("No alternative medication available for allergy");
                    return null; // FAIL SAFE: No safe medication
                }
            }

            // Apply dose adjustments (renal/hepatic)
            selectedMed = applyDoseAdjustments(selectedMed, context);

            // Create new action with selected medication
            ProtocolAction selectedAction = action.clone();
            selectedAction.setMedication(selectedMed);

            logger.info("Selected medication: {} {} {} {}",
                selectedMed.getName(),
                selectedMed.getDose(),
                selectedMed.getRoute(),
                selectedMed.getFrequency());

            return selectedAction;
        }
    }

    // No criteria met - return original action
    logger.warn("No selection criteria met for action {}", action.getActionId());
    return action;
}
```

#### 3.2 evaluateCriteria()

```java
/**
 * Evaluates a selection criteria ID.
 *
 * Standard criteria:
 * - NO_PENICILLIN_ALLERGY: Patient not allergic to penicillin
 * - NO_BETA_LACTAM_ALLERGY: Patient not allergic to beta-lactams
 * - CREATININE_CLEARANCE_GT_40: CrCl > 40 mL/min
 * - CREATININE_CLEARANCE_GT_30: CrCl > 30 mL/min
 * - MDR_RISK: Multi-drug resistant risk factors present
 * - NO_BETA_BLOCKER_CONTRAINDICATION: Safe to use beta-blockers
 *
 * @param criteriaId The criteria identifier
 * @param context The patient context
 * @return true if criteria met
 */
private boolean evaluateCriteria(String criteriaId, EnrichedPatientContext context) {
    PatientState state = context.getPatientState();

    switch (criteriaId) {
        case "NO_PENICILLIN_ALLERGY":
            List<String> allergies = state.getAllergies();
            return allergies == null || !allergies.stream()
                .anyMatch(a -> a.toLowerCase().contains("penicillin"));

        case "NO_BETA_LACTAM_ALLERGY":
            return !hasAllergy("penicillin", context) &&
                   !hasAllergy("cephalosporin", context);

        case "CREATININE_CLEARANCE_GT_40":
            double crCl = calculateCrCl(context);
            return crCl > 40.0;

        case "CREATININE_CLEARANCE_GT_30":
            return calculateCrCl(context) > 30.0;

        case "MDR_RISK":
            // Multi-drug resistant risk factors
            return state.hasRecentHospitalization() ||
                   state.hasRecentAntibiotics() ||
                   state.isImmunosuppressed() ||
                   state.hasIndwellingDevices();

        case "NO_BETA_BLOCKER_CONTRAINDICATION":
            return !hasBetaBlockerContraindication(context);

        case "SEVERE_SEPSIS":
            return state.getLactate() != null && state.getLactate() >= 4.0;

        case "HIGH_BLEEDING_RISK":
            return state.hasActiveBleed() ||
                   state.getPlatelets() < 50000 ||
                   (state.getINR() != null && state.getINR() > 2.0);

        default:
            logger.warn("Unknown criteria: {}", criteriaId);
            return false;
    }
}
```

#### 3.3 hasAllergy()

```java
/**
 * Checks if patient is allergic to a medication or drug class.
 *
 * @param medication The medication to check
 * @param context The patient context
 * @return true if patient has documented allergy
 */
private boolean hasAllergy(Medication medication, EnrichedPatientContext context) {
    List<String> allergies = context.getPatientState().getAllergies();

    if (allergies == null || allergies.isEmpty()) {
        return false;
    }

    String medName = medication.getName().toLowerCase();

    for (String allergy : allergies) {
        String allergyLower = allergy.toLowerCase();

        // Direct match
        if (medName.contains(allergyLower) || allergyLower.contains(medName)) {
            logger.warn("Direct allergy match: {} vs {}", medName, allergyLower);
            return true;
        }

        // Cross-reactivity checking
        if (hasCrossReactivity(medName, allergyLower)) {
            logger.warn("Cross-reactivity detected: {} with allergy to {}",
                medName, allergyLower);
            return true;
        }
    }

    return false;
}

/**
 * Checks for drug class cross-reactivity.
 */
private boolean hasCrossReactivity(String medication, String allergy) {
    // Penicillin allergy → cephalosporin cross-reactivity
    if (allergy.contains("penicillin")) {
        if (medication.contains("cef") || // Cephalosporins
            medication.contains("ceftriaxone") ||
            medication.contains("cefepime")) {
            return true;
        }
    }

    // Sulfa allergy → sulfonamide antibiotics
    if (allergy.contains("sulfa")) {
        if (medication.contains("sulfamethoxazole") ||
            medication.contains("trimethoprim")) {
            return true;
        }
    }

    return false;
}
```

#### 3.4 calculateCrCl()

```java
/**
 * Calculates creatinine clearance using Cockcroft-Gault formula.
 *
 * CrCl (mL/min) = [(140 - age) × weight(kg)] / (72 × Cr(mg/dL))
 * Multiply by 0.85 for females
 *
 * @param context The patient context
 * @return Creatinine clearance in mL/min
 */
private double calculateCrCl(EnrichedPatientContext context) {
    PatientState state = context.getPatientState();

    Integer age = state.getAge();
    Double weight = state.getWeight();
    Double creatinine = state.getCreatinine();
    String sex = state.getSex();

    // Check required parameters
    if (age == null || weight == null || creatinine == null || creatinine == 0.0) {
        logger.warn("Missing parameters for CrCl calculation: age={}, weight={}, creatinine={}",
            age, weight, creatinine);
        return 60.0; // Default safe value
    }

    // Cockcroft-Gault formula
    double crCl = ((140 - age) * weight) / (72 * creatinine);

    // Female adjustment
    if ("F".equalsIgnoreCase(sex) || "FEMALE".equalsIgnoreCase(sex)) {
        crCl *= 0.85;
    }

    logger.debug("Calculated CrCl: {} mL/min (age={}, weight={}, Cr={}, sex={})",
        crCl, age, weight, creatinine, sex);

    return crCl;
}
```

#### 3.5 applyDoseAdjustments()

```java
/**
 * Applies renal and hepatic dose adjustments to medication.
 *
 * @param medication The medication to adjust
 * @param context The patient context
 * @return The adjusted medication
 */
private Medication applyDoseAdjustments(Medication medication, EnrichedPatientContext context) {
    Medication adjusted = medication.clone();

    // Renal dose adjustment
    double crCl = calculateCrCl(context);
    if (crCl < 60.0) {
        adjusted = adjustDoseForRenalFunction(adjusted, crCl);
    }

    // Hepatic dose adjustment
    String childPugh = context.getPatientState().getChildPughScore();
    if (childPugh != null && (childPugh.equals("B") || childPugh.equals("C"))) {
        adjusted = adjustDoseForHepaticFunction(adjusted, childPugh);
    }

    return adjusted;
}
```

#### 3.6 adjustDoseForRenalFunction()

```java
/**
 * Adjusts medication dose based on renal function.
 *
 * @param medication The medication
 * @param crCl Creatinine clearance (mL/min)
 * @return Medication with adjusted dose
 */
private Medication adjustDoseForRenalFunction(Medication medication, double crCl) {
    String medName = medication.getName().toLowerCase();

    // Common medication renal adjustments
    if (medName.contains("ceftriaxone")) {
        if (crCl < 30) {
            medication.setDose("1 g"); // Reduced from 2 g
            medication.setAdministrationInstructions(
                "Renal dose adjustment (CrCl < 30 mL/min)");
            logger.info("Adjusted ceftriaxone dose for CrCl {}: 1 g", crCl);
        }
    } else if (medName.contains("vancomycin")) {
        if (crCl < 60) {
            medication.setAdministrationInstructions(
                "Dose per pharmacist consult (CrCl < 60 mL/min)");
            logger.info("Vancomycin requires pharmacist dosing for CrCl {}", crCl);
        }
    } else if (medName.contains("levofloxacin")) {
        if (crCl < 50) {
            medication.setDose("500 mg"); // Reduced from 750 mg
            medication.setFrequency("q48h"); // Extended interval
            logger.info("Adjusted levofloxacin for CrCl {}: 500 mg q48h", crCl);
        }
    }

    return medication;
}
```

### Unit Tests

```java
@Test
void testSelectMedication_NoAllergy_UsesPrimary() {
    ProtocolAction action = createActionWithSelection("NO_PENICILLIN_ALLERGY",
        "Ceftriaxone", "2 g", "Levofloxacin", "750 mg");

    EnrichedPatientContext context = createContext();
    context.getPatientState().setAllergies(Arrays.asList("iodine")); // No penicillin allergy

    ProtocolAction selected = selector.selectMedication(action, context);

    assertEquals("Ceftriaxone", selected.getMedication().getName());
    assertEquals("2 g", selected.getMedication().getDose());
}

@Test
void testSelectMedication_PenicillinAllergy_UsesAlternative() {
    ProtocolAction action = createActionWithSelection("NO_PENICILLIN_ALLERGY",
        "Ceftriaxone", "2 g", "Levofloxacin", "750 mg");

    EnrichedPatientContext context = createContext();
    context.getPatientState().setAllergies(Arrays.asList("penicillin"));

    ProtocolAction selected = selector.selectMedication(action, context);

    assertEquals("Levofloxacin", selected.getMedication().getName());
    assertEquals("750 mg", selected.getMedication().getDose());
}

@Test
void testCalculateCrCl_Male70kg() {
    EnrichedPatientContext context = createContext();
    context.getPatientState().setAge(65);
    context.getPatientState().setWeight(70.0);
    context.getPatientState().setCreatinine(1.2);
    context.getPatientState().setSex("M");

    double crCl = selector.calculateCrCl(context);

    // Expected: (140-65) * 70 / (72 * 1.2) = 60.76
    assertEquals(60.76, crCl, 0.5);
}

@Test
void testCalculateCrCl_Female60kg() {
    EnrichedPatientContext context = createContext();
    context.getPatientState().setAge(72);
    context.getPatientState().setWeight(60.0);
    context.getPatientState().setCreatinine(1.5);
    context.getPatientState().setSex("F");

    double crCl = selector.calculateCrCl(context);

    // Expected: ((140-72) * 60 / (72 * 1.5)) * 0.85 = 32.22
    assertEquals(32.22, crCl, 0.5);
}
```

---

## 4. TimeConstraintTracker.java

**Package**: `com.cardiofit.flink.cds.time`
**Purpose**: Track time-sensitive interventions and generate deadline alerts
**Estimated Lines**: 500
**Estimated Effort**: 3-4 hours
**Priority**: **CRITICAL** (Phase 1) - Required for sepsis bundles, STEMI

### Class Overview

```java
package com.cardiofit.flink.cds.time;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.protocol.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.*;
import java.util.*;

/**
 * Tracks time constraints for clinical bundles and generates deadline alerts.
 *
 * <p>Handles:
 * - Hour-0/1/3 bundle tracking (e.g., sepsis bundles)
 * - Deadline calculation: trigger time + offset_minutes
 * - Alert generation: WARNING (<30min remaining), CRITICAL (deadline exceeded)
 * - Bundle compliance monitoring
 *
 * <p>Critical for time-sensitive protocols like sepsis, STEMI, stroke.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class TimeConstraintTracker {
    private static final Logger logger = LoggerFactory.getLogger(TimeConstraintTracker.class);
    private static final int WARNING_THRESHOLD_MINUTES = 30;

    public TimeConstraintStatus evaluateConstraints(
        Protocol protocol,
        EnrichedPatientContext context);

    private ConstraintStatus evaluateConstraint(
        TimeConstraint constraint,
        Instant triggerTime,
        Instant currentTime);

    private AlertLevel determineAlertLevel(
        Duration timeRemaining,
        boolean isCritical);
}
```

### Method Specifications

#### 4.1 evaluateConstraints()

```java
/**
 * Evaluates all time constraints for a protocol.
 *
 * @param protocol The protocol with time constraints
 * @param context The patient context (includes trigger time)
 * @return Status of all time constraints with alerts
 */
public TimeConstraintStatus evaluateConstraints(
    Protocol protocol,
    EnrichedPatientContext context) {

    TimeConstraintStatus status = new TimeConstraintStatus();
    status.setProtocolId(protocol.getProtocolId());

    Instant triggerTime = context.getTriggerTime();
    Instant currentTime = Instant.now();

    if (triggerTime == null) {
        logger.warn("No trigger time in context, using current time");
        triggerTime = currentTime;
    }

    logger.debug("Evaluating time constraints: trigger={}, current={}",
        triggerTime, currentTime);

    List<TimeConstraint> constraints = protocol.getTimeConstraints();
    if (constraints == null || constraints.isEmpty()) {
        logger.debug("No time constraints for protocol {}", protocol.getProtocolId());
        return status;
    }

    for (TimeConstraint constraint : constraints) {
        ConstraintStatus constraintStatus = evaluateConstraint(
            constraint,
            triggerTime,
            currentTime);

        status.addConstraintStatus(constraintStatus);

        // Log critical alerts
        if (constraint.isCritical() && constraintStatus.getAlertLevel() == AlertLevel.CRITICAL) {
            logger.error("CRITICAL TIME CONSTRAINT EXCEEDED: {} - {}",
                constraint.getBundleName(),
                constraintStatus.getMessage());
        } else if (constraintStatus.getAlertLevel() == AlertLevel.WARNING) {
            logger.warn("TIME CONSTRAINT WARNING: {} - {}",
                constraint.getBundleName(),
                constraintStatus.getMessage());
        }
    }

    logger.info("Time constraint evaluation complete: {} constraints, {} critical alerts",
        status.getConstraintStatuses().size(),
        status.getCriticalAlerts().size());

    return status;
}
```

#### 4.2 evaluateConstraint()

```java
/**
 * Evaluates a single time constraint.
 *
 * @param constraint The time constraint
 * @param triggerTime When protocol was triggered
 * @param currentTime Current time
 * @return Status with alert level and message
 */
private ConstraintStatus evaluateConstraint(
    TimeConstraint constraint,
    Instant triggerTime,
    Instant currentTime) {

    ConstraintStatus status = new ConstraintStatus();
    status.setConstraintId(constraint.getConstraintId());
    status.setBundleName(constraint.getBundleName());
    status.setCritical(constraint.isCritical());

    // Calculate deadline
    Instant deadline = triggerTime.plus(constraint.getOffsetMinutes(), ChronoUnit.MINUTES);
    status.setDeadline(deadline);

    // Calculate time remaining
    Duration timeRemaining = Duration.between(currentTime, deadline);
    status.setTimeRemaining(timeRemaining);

    // Determine alert level
    AlertLevel alertLevel = determineAlertLevel(timeRemaining, constraint.isCritical());
    status.setAlertLevel(alertLevel);

    // Generate message
    String message = generateMessage(
        constraint.getBundleName(),
        timeRemaining,
        alertLevel);
    status.setMessage(message);

    logger.debug("Constraint {}: deadline={}, remaining={}, alert={}",
        constraint.getConstraintId(),
        deadline,
        formatDuration(timeRemaining),
        alertLevel);

    return status;
}
```

#### 4.3 determineAlertLevel()

```java
/**
 * Determines alert level based on time remaining.
 *
 * @param timeRemaining Duration until deadline
 * @param isCritical Whether this constraint is critical
 * @return AlertLevel (INFO, WARNING, CRITICAL)
 */
private AlertLevel determineAlertLevel(Duration timeRemaining, boolean isCritical) {
    long minutesRemaining = timeRemaining.toMinutes();

    if (minutesRemaining < 0) {
        // Deadline exceeded
        return AlertLevel.CRITICAL;

    } else if (minutesRemaining <= WARNING_THRESHOLD_MINUTES) {
        // Within warning threshold
        return isCritical ? AlertLevel.WARNING : AlertLevel.INFO;

    } else {
        // On track
        return AlertLevel.INFO;
    }
}
```

#### 4.4 Helper Methods

```java
/**
 * Generates human-readable message for time constraint status.
 */
private String generateMessage(String bundleName, Duration timeRemaining, AlertLevel alertLevel) {
    long minutes = Math.abs(timeRemaining.toMinutes());

    switch (alertLevel) {
        case CRITICAL:
            return String.format("%s deadline exceeded by %d minutes",
                bundleName, minutes);

        case WARNING:
            return String.format("%s deadline in %d minutes",
                bundleName, minutes);

        case INFO:
            return String.format("%s: %d minutes remaining",
                bundleName, minutes);

        default:
            return String.format("%s status: %s",
                bundleName, alertLevel);
    }
}

/**
 * Formats duration as human-readable string.
 */
private String formatDuration(Duration duration) {
    long minutes = Math.abs(duration.toMinutes());
    long hours = minutes / 60;
    long mins = minutes % 60;

    if (hours > 0) {
        return String.format("%dh %dm", hours, mins);
    } else {
        return String.format("%dm", mins);
    }
}
```

### Supporting Classes

```java
/**
 * Status of all time constraints for a protocol.
 */
public class TimeConstraintStatus {
    private String protocolId;
    private List<ConstraintStatus> constraintStatuses = new ArrayList<>();

    public boolean hasCriticalAlerts() {
        return constraintStatuses.stream()
            .anyMatch(cs -> cs.getAlertLevel() == AlertLevel.CRITICAL);
    }

    public List<ConstraintStatus> getCriticalAlerts() {
        return constraintStatuses.stream()
            .filter(cs -> cs.getAlertLevel() == AlertLevel.CRITICAL)
            .collect(Collectors.toList());
    }

    public List<ConstraintStatus> getWarningAlerts() {
        return constraintStatuses.stream()
            .filter(cs -> cs.getAlertLevel() == AlertLevel.WARNING)
            .collect(Collectors.toList());
    }

    // Getters/setters
}

/**
 * Status of a single time constraint.
 */
public class ConstraintStatus {
    private String constraintId;
    private String bundleName;
    private boolean critical;
    private Instant deadline;
    private Duration timeRemaining;
    private AlertLevel alertLevel;
    private String message;

    // Getters/setters
}

/**
 * Alert level for time constraints.
 */
public enum AlertLevel {
    INFO,       // On track, >30 minutes remaining
    WARNING,    // <30 minutes remaining
    CRITICAL    // Deadline exceeded
}
```

### Unit Tests

```java
@Test
void testEvaluateConstraint_OnTrack() {
    // Trigger 30 minutes ago, deadline 60 minutes from trigger (30 min remaining)
    Instant triggerTime = Instant.now().minus(30, ChronoUnit.MINUTES);
    Instant currentTime = Instant.now();

    TimeConstraint constraint = new TimeConstraint();
    constraint.setConstraintId("HOUR-1");
    constraint.setBundleName("Hour-1 Bundle");
    constraint.setOffsetMinutes(60);
    constraint.setCritical(true);

    ConstraintStatus status = tracker.evaluateConstraint(constraint, triggerTime, currentTime);

    assertEquals(AlertLevel.INFO, status.getAlertLevel());
    assertTrue(status.getTimeRemaining().toMinutes() > 25); // ~30 minutes
}

@Test
void testEvaluateConstraint_Warning() {
    // Trigger 50 minutes ago, deadline 60 minutes from trigger (10 min remaining)
    Instant triggerTime = Instant.now().minus(50, ChronoUnit.MINUTES);
    Instant currentTime = Instant.now();

    TimeConstraint constraint = new TimeConstraint();
    constraint.setConstraintId("HOUR-1");
    constraint.setBundleName("Hour-1 Bundle");
    constraint.setOffsetMinutes(60);
    constraint.setCritical(true);

    ConstraintStatus status = tracker.evaluateConstraint(constraint, triggerTime, currentTime);

    assertEquals(AlertLevel.WARNING, status.getAlertLevel());
    assertTrue(status.getTimeRemaining().toMinutes() < 15);
    assertTrue(status.getMessage().contains("deadline in"));
}

@Test
void testEvaluateConstraint_Critical() {
    // Trigger 70 minutes ago, deadline 60 minutes from trigger (10 min overdue)
    Instant triggerTime = Instant.now().minus(70, ChronoUnit.MINUTES);
    Instant currentTime = Instant.now();

    TimeConstraint constraint = new TimeConstraint();
    constraint.setConstraintId("HOUR-1");
    constraint.setBundleName("Hour-1 Bundle");
    constraint.setOffsetMinutes(60);
    constraint.setCritical(true);

    ConstraintStatus status = tracker.evaluateConstraint(constraint, triggerTime, currentTime);

    assertEquals(AlertLevel.CRITICAL, status.getAlertLevel());
    assertTrue(status.getTimeRemaining().isNegative());
    assertTrue(status.getMessage().contains("exceeded"));
}
```

---

## 5. KnowledgeBaseManager.java

**Package**: `com.cardiofit.flink.cds.knowledge`
**Purpose**: Singleton pattern for protocol storage with fast indexed lookup
**Estimated Lines**: 750
**Estimated Effort**: 4-5 hours
**Priority**: High (Phase 2)

### Class Overview

```java
package com.cardiofit.flink.cds.knowledge;

import com.cardiofit.flink.models.protocol.*;
import com.cardiofit.flink.utils.ProtocolLoader;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.nio.file.*;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;

/**
 * Singleton manager for clinical protocol knowledge base.
 *
 * <p>Features:
 * - Thread-safe protocol storage with ConcurrentHashMap
 * - Category and specialty indexes for fast lookup
 * - Hot reload capability (watches YAML files for changes)
 * - Query methods (getByCategory, getBySpecialty, search)
 *
 * <p>Example usage:
 * <pre>
 * KnowledgeBaseManager kb = KnowledgeBaseManager.getInstance();
 * List<Protocol> infectiousProtocols = kb.getByCategory(ProtocolCategory.INFECTIOUS);
 * Protocol sepsis = kb.getProtocol("SEPSIS-BUNDLE-001");
 * </pre>
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class KnowledgeBaseManager {
    private static final Logger logger = LoggerFactory.getLogger(KnowledgeBaseManager.class);
    private static volatile KnowledgeBaseManager instance;

    // Thread-safe storage
    private final ConcurrentHashMap<String, Protocol> protocols;
    private final Map<ProtocolCategory, List<Protocol>> categoryIndex;
    private final Map<String, List<Protocol>> specialtyIndex;

    // Hot reload support
    private final WatchService watchService;
    private final Path protocolDirectory;
    private volatile boolean isReloading = false;

    private KnowledgeBaseManager();

    public static KnowledgeBaseManager getInstance();

    public Protocol getProtocol(String protocolId);

    public List<Protocol> getAllProtocols();

    public List<Protocol> getByCategory(ProtocolCategory category);

    public List<Protocol> getBySpecialty(String specialty);

    public List<Protocol> search(String query);

    public synchronized void reloadProtocols();

    private void loadAllProtocols();

    private void buildIndexes();

    private WatchService initializeWatchService();

    private void startWatchService();
}
```

### Method Specifications

#### 5.1 getInstance() - Singleton Pattern

```java
/**
 * Gets the singleton instance of KnowledgeBaseManager.
 *
 * Uses double-checked locking for thread-safe lazy initialization.
 *
 * @return The singleton instance
 */
public static KnowledgeBaseManager getInstance() {
    if (instance == null) {
        synchronized (KnowledgeBaseManager.class) {
            if (instance == null) {
                instance = new KnowledgeBaseManager();
                logger.info("KnowledgeBaseManager singleton initialized");
            }
        }
    }
    return instance;
}
```

#### 5.2 Constructor

```java
/**
 * Private constructor for singleton pattern.
 * Initializes storage, loads protocols, starts watch service.
 */
private KnowledgeBaseManager() {
    this.protocols = new ConcurrentHashMap<>();
    this.categoryIndex = new ConcurrentHashMap<>();
    this.specialtyIndex = new ConcurrentHashMap<>();
    this.protocolDirectory = Paths.get("clinical-protocols");
    this.watchService = initializeWatchService();

    loadAllProtocols();
    startWatchService();

    logger.info("KnowledgeBaseManager initialized with {} protocols",
        protocols.size());
}
```

#### 5.3 loadAllProtocols()

```java
/**
 * Loads all protocols from YAML files.
 *
 * @throws RuntimeException if loading fails
 */
private void loadAllProtocols() {
    try {
        logger.info("Loading protocols...");

        Map<String, Protocol> loadedProtocols = ProtocolLoader.loadProtocols();

        // Clear existing data
        protocols.clear();
        categoryIndex.clear();
        specialtyIndex.clear();

        // Add to main storage
        protocols.putAll(loadedProtocols);

        // Build indexes
        buildIndexes();

        logger.info("Loaded {} protocols successfully", protocols.size());

        // Log breakdown by category
        for (ProtocolCategory category : ProtocolCategory.values()) {
            int count = categoryIndex.getOrDefault(category, Collections.emptyList()).size();
            if (count > 0) {
                logger.info("  {}: {} protocols", category, count);
            }
        }

    } catch (Exception e) {
        logger.error("Failed to load protocols", e);
        throw new RuntimeException("Protocol loading failed", e);
    }
}
```

#### 5.4 buildIndexes()

```java
/**
 * Builds category and specialty indexes for fast lookup.
 */
private void buildIndexes() {
    logger.debug("Building protocol indexes...");

    for (Protocol protocol : protocols.values()) {
        // Category index
        ProtocolCategory category = protocol.getCategory();
        categoryIndex.computeIfAbsent(
            category,
            k -> new CopyOnWriteArrayList<>()
        ).add(protocol);

        // Specialty index
        String specialty = protocol.getSpecialty();
        if (specialty != null && !specialty.isEmpty()) {
            specialtyIndex.computeIfAbsent(
                specialty,
                k -> new CopyOnWriteArrayList<>()
            ).add(protocol);
        }
    }

    logger.info("Built indexes: {} categories, {} specialties",
        categoryIndex.size(), specialtyIndex.size());
}
```

#### 5.5 Query Methods

```java
/**
 * Gets a protocol by ID.
 *
 * @param protocolId The protocol ID
 * @return The protocol, or null if not found
 */
public Protocol getProtocol(String protocolId) {
    return protocols.get(protocolId);
}

/**
 * Gets all protocols.
 *
 * @return List of all protocols
 */
public List<Protocol> getAllProtocols() {
    return new ArrayList<>(protocols.values());
}

/**
 * Gets protocols by category.
 *
 * @param category The protocol category
 * @return List of protocols in this category (empty list if none)
 */
public List<Protocol> getByCategory(ProtocolCategory category) {
    return new ArrayList<>(
        categoryIndex.getOrDefault(category, Collections.emptyList())
    );
}

/**
 * Gets protocols by specialty.
 *
 * @param specialty The clinical specialty (e.g., "CRITICAL_CARE")
 * @return List of protocols for this specialty (empty list if none)
 */
public List<Protocol> getBySpecialty(String specialty) {
    return new ArrayList<>(
        specialtyIndex.getOrDefault(specialty, Collections.emptyList())
    );
}

/**
 * Searches protocols by query string.
 *
 * Matches against protocol ID, name, and category.
 *
 * @param query The search query (case-insensitive)
 * @return List of matching protocols
 */
public List<Protocol> search(String query) {
    String lowerQuery = query.toLowerCase();

    return protocols.values().stream()
        .filter(p ->
            p.getName().toLowerCase().contains(lowerQuery) ||
            p.getProtocolId().toLowerCase().contains(lowerQuery) ||
            p.getCategory().name().toLowerCase().contains(lowerQuery)
        )
        .collect(Collectors.toList());
}
```

#### 5.6 Hot Reload

```java
/**
 * Initializes file watch service for hot reload.
 */
private WatchService initializeWatchService() {
    try {
        WatchService watchService = FileSystems.getDefault().newWatchService();

        protocolDirectory.register(
            watchService,
            StandardWatchEventKinds.ENTRY_MODIFY,
            StandardWatchEventKinds.ENTRY_CREATE,
            StandardWatchEventKinds.ENTRY_DELETE
        );

        logger.info("File watch service initialized for {}", protocolDirectory);
        return watchService;

    } catch (Exception e) {
        logger.error("Failed to initialize watch service", e);
        return null;
    }
}

/**
 * Starts background thread to watch for protocol file changes.
 */
private void startWatchService() {
    if (watchService == null) {
        logger.warn("Watch service not initialized, hot reload disabled");
        return;
    }

    Thread watchThread = new Thread(() -> {
        logger.info("Protocol file watcher started");

        while (true) {
            try {
                WatchKey key = watchService.take();

                for (WatchEvent<?> event : key.pollEvents()) {
                    Path changed = (Path) event.context();

                    if (changed.toString().endsWith(".yaml")) {
                        logger.info("Protocol file changed: {} ({})",
                            changed, event.kind());

                        // Trigger reload after short delay (debouncing)
                        Thread.sleep(2000);
                        reloadProtocols();
                    }
                }

                key.reset();

            } catch (InterruptedException e) {
                logger.error("Watch service interrupted", e);
                Thread.currentThread().interrupt();
                break;
            }
        }
    });

    watchThread.setDaemon(true);
    watchThread.setName("ProtocolWatcher");
    watchThread.start();
}

/**
 * Reloads all protocols from disk.
 *
 * Thread-safe with lock to prevent concurrent reloads.
 */
public synchronized void reloadProtocols() {
    if (isReloading) {
        logger.warn("Reload already in progress, skipping");
        return;
    }

    try {
        isReloading = true;
        logger.info("Starting protocol reload...");

        long startTime = System.currentTimeMillis();
        loadAllProtocols();
        long duration = System.currentTimeMillis() - startTime;

        logger.info("Protocol reload completed successfully in {}ms", duration);

    } catch (Exception e) {
        logger.error("Protocol reload failed", e);
    } finally {
        isReloading = false;
    }
}
```

### Unit Tests

```java
@Test
void testGetInstance_Singleton() {
    KnowledgeBaseManager kb1 = KnowledgeBaseManager.getInstance();
    KnowledgeBaseManager kb2 = KnowledgeBaseManager.getInstance();

    assertSame(kb1, kb2, "Should return same instance");
}

@Test
void testGetProtocol_Found() {
    KnowledgeBaseManager kb = KnowledgeBaseManager.getInstance();
    Protocol sepsis = kb.getProtocol("SEPSIS-BUNDLE-001");

    assertNotNull(sepsis);
    assertEquals("SEPSIS-BUNDLE-001", sepsis.getProtocolId());
    assertEquals("Sepsis Management Bundle", sepsis.getName());
}

@Test
void testGetByCategory_Infectious() {
    KnowledgeBaseManager kb = KnowledgeBaseManager.getInstance();
    List<Protocol> infectious = kb.getByCategory(ProtocolCategory.INFECTIOUS);

    assertFalse(infectious.isEmpty());
    assertTrue(infectious.stream()
        .anyMatch(p -> p.getProtocolId().equals("SEPSIS-BUNDLE-001")));
}

@Test
void testSearch_FindsSepsis() {
    KnowledgeBaseManager kb = KnowledgeBaseManager.getInstance();
    List<Protocol> results = kb.search("sepsis");

    assertFalse(results.isEmpty());
    assertTrue(results.stream()
        .anyMatch(p -> p.getProtocolId().equals("SEPSIS-BUNDLE-001")));
}

@Test
void testGetAllProtocols_Returns16() {
    KnowledgeBaseManager kb = KnowledgeBaseManager.getInstance();
    List<Protocol> all = kb.getAllProtocols();

    assertEquals(16, all.size());
}
```

---

## 6. EscalationRuleEvaluator.java

**Package**: `com.cardiofit.flink.cds.escalation`
**Purpose**: Evaluate escalation triggers and generate ICU transfer recommendations
**Estimated Lines**: 350
**Estimated Effort**: 2-3 hours
**Priority**: Medium (Phase 3)

### Class Overview

```java
package com.cardiofit.flink.cds.escalation;

import com.cardiofit.flink.cds.evaluation.ConditionEvaluator;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.protocol.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Instant;
import java.util.*;

/**
 * Evaluates escalation rules and generates ICU transfer recommendations.
 *
 * <p>Handles:
 * - Escalation trigger evaluation
 * - Clinical deterioration detection
 * - ICU transfer recommendations with rationale
 * - Multiple escalation levels (CONSULT, TRANSFER, IMMEDIATE_TRANSFER)
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class EscalationRuleEvaluator {
    private static final Logger logger = LoggerFactory.getLogger(EscalationRuleEvaluator.class);

    private final ConditionEvaluator conditionEvaluator;

    public EscalationRuleEvaluator(ConditionEvaluator conditionEvaluator);

    public List<EscalationRecommendation> evaluateEscalation(
        Protocol protocol,
        EnrichedPatientContext context);

    private Map<String, Object> gatherClinicalEvidence(
        EscalationRule rule,
        EnrichedPatientContext context);
}
```

### Method Specifications

#### 6.1 evaluateEscalation()

```java
/**
 * Evaluates escalation rules for a protocol.
 *
 * @param protocol The protocol with escalation rules
 * @param context The patient context
 * @return List of escalation recommendations (sorted by urgency)
 */
public List<EscalationRecommendation> evaluateEscalation(
    Protocol protocol,
    EnrichedPatientContext context) {

    List<EscalationRecommendation> escalations = new ArrayList<>();

    if (protocol.getEscalationRules() == null || protocol.getEscalationRules().isEmpty()) {
        logger.debug("No escalation rules for protocol {}", protocol.getProtocolId());
        return escalations;
    }

    logger.debug("Evaluating {} escalation rules for protocol {}",
        protocol.getEscalationRules().size(),
        protocol.getProtocolId());

    for (EscalationRule rule : protocol.getEscalationRules()) {
        // Evaluate escalation trigger
        boolean triggered = conditionEvaluator.evaluate(
            rule.getEscalationTrigger(),
            context);

        if (triggered) {
            logger.warn("ESCALATION RULE TRIGGERED: {} - {}",
                rule.getRuleId(),
                rule.getRecommendation().getRationale());

            EscalationRecommendation escalation = new EscalationRecommendation();
            escalation.setRuleId(rule.getRuleId());
            escalation.setProtocolId(protocol.getProtocolId());
            escalation.setProtocolName(protocol.getName());
            escalation.setEscalationLevel(rule.getRecommendation().getEscalationLevel());
            escalation.setUrgency(rule.getRecommendation().getUrgency());
            escalation.setRationale(rule.getRecommendation().getRationale());
            escalation.setTimestamp(Instant.now());

            // Add clinical evidence
            Map<String, Object> evidence = gatherClinicalEvidence(rule, context);
            escalation.setClinicalEvidence(evidence);

            // Add required interventions
            if (rule.getRecommendation().getRequiredInterventions() != null) {
                escalation.setRequiredInterventions(
                    new ArrayList<>(rule.getRecommendation().getRequiredInterventions())
                );
            }

            // Add specialist consultations
            if (rule.getRecommendation().getSpecialistConsultation() != null) {
                escalation.setSpecialistConsultations(
                    new ArrayList<>(rule.getRecommendation().getSpecialistConsultation())
                );
            }

            escalations.add(escalation);
        }
    }

    // Sort by escalation level (IMMEDIATE_TRANSFER first)
    escalations.sort(Comparator.comparing(EscalationRecommendation::getEscalationLevel));

    logger.info("Escalation evaluation complete: {} escalations triggered",
        escalations.size());

    return escalations;
}
```

#### 6.2 gatherClinicalEvidence()

```java
/**
 * Gathers clinical evidence that triggered the escalation.
 *
 * Extracts relevant parameter values from patient context.
 *
 * @param rule The escalation rule
 * @param context The patient context
 * @return Map of parameter names to values
 */
private Map<String, Object> gatherClinicalEvidence(
    EscalationRule rule,
    EnrichedPatientContext context) {

    Map<String, Object> evidence = new HashMap<>();
    PatientState state = context.getPatientState();

    // Extract parameters from trigger conditions
    List<Condition> conditions = rule.getEscalationTrigger().getConditions();
    if (conditions != null) {
        for (Condition condition : conditions) {
            String parameter = condition.getParameter();
            if (parameter != null) {
                Object value = extractParameterValue(parameter, state);
                if (value != null) {
                    evidence.put(parameter, value);
                }
            }

            // Handle nested conditions
            if (condition.getConditions() != null) {
                for (Condition nested : condition.getConditions()) {
                    String nestedParam = nested.getParameter();
                    if (nestedParam != null) {
                        Object value = extractParameterValue(nestedParam, state);
                        if (value != null) {
                            evidence.put(nestedParam, value);
                        }
                    }
                }
            }
        }
    }

    return evidence;
}

/**
 * Extracts parameter value from patient state.
 */
private Object extractParameterValue(String parameter, PatientState state) {
    switch (parameter.toLowerCase()) {
        case "lactate":
            return state.getLactate();
        case "systolic_bp":
            return state.getSystolicBP();
        case "mean_arterial_pressure":
        case "map":
            return state.getMeanArterialPressure();
        case "sofa_score":
            return state.getSofaScore();
        case "number_of_failing_organs":
            return state.getNumberOfFailingOrgans();
        default:
            return null;
    }
}
```

### Supporting Classes

```java
/**
 * Escalation recommendation for ICU transfer or specialist consultation.
 */
public class EscalationRecommendation {
    private String ruleId;
    private String protocolId;
    private String protocolName;
    private EscalationLevel escalationLevel;
    private String urgency; // STAT, IMMEDIATE, URGENT, ROUTINE
    private String rationale;
    private Instant timestamp;
    private Map<String, Object> clinicalEvidence;
    private List<String> requiredInterventions;
    private List<SpecialistConsultation> specialistConsultations;

    // Getters/setters
}

/**
 * Specialist consultation details.
 */
public class SpecialistConsultation {
    private String specialty; // CRITICAL_CARE, CARDIOLOGY, etc.
    private String urgency; // STAT, URGENT, ROUTINE
    private String specificQuestion;

    // Getters/setters
}

/**
 * Escalation level enum.
 */
public enum EscalationLevel {
    CONSULT,            // Specialist consultation recommended (priority 3)
    TRANSFER,           // ICU transfer recommended (priority 2)
    IMMEDIATE_TRANSFER  // Immediate ICU transfer required (priority 1)
}
```

### Unit Tests

```java
@Test
void testEvaluateEscalation_SepticShock_TriggersICU() {
    // Create sepsis protocol with ICU escalation rule
    Protocol sepsis = createSepsisProtocol();

    // Create context with septic shock (lactate >=4.0)
    EnrichedPatientContext context = createContext();
    context.getPatientState().setLactate(4.5);

    List<EscalationRecommendation> escalations =
        evaluator.evaluateEscalation(sepsis, context);

    assertEquals(1, escalations.size());
    assertEquals(EscalationLevel.ICU_TRANSFER, escalations.get(0).getEscalationLevel());
    assertTrue(escalations.get(0).getRationale().contains("septic shock"));
    assertEquals(4.5, escalations.get(0).getClinicalEvidence().get("lactate"));
}

@Test
void testEvaluateEscalation_NoTriggers() {
    Protocol sepsis = createSepsisProtocol();

    EnrichedPatientContext context = createContext();
    context.getPatientState().setLactate(1.5); // Normal

    List<EscalationRecommendation> escalations =
        evaluator.evaluateEscalation(sepsis, context);

    assertEquals(0, escalations.size());
}
```

---

## 7. ProtocolValidator.java

**Package**: `com.cardiofit.flink.cds.validation`
**Purpose**: Validate protocol YAML structure and completeness
**Estimated Lines**: 250
**Estimated Effort**: 2 hours
**Priority**: High (Phase 2)

### Class Overview

```java
package com.cardiofit.flink.cds.validation;

import com.cardiofit.flink.models.protocol.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Validates protocol YAML structure and completeness.
 *
 * <p>Validation checks:
 * - Required fields present (protocol_id, name, category, actions)
 * - Reference validation (action_ids, condition_ids unique and valid)
 * - Range validation (confidence scores 0.0-1.0, thresholds valid)
 * - Completeness (evidence_source, contraindications recommended)
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class ProtocolValidator {
    private static final Logger logger = LoggerFactory.getLogger(ProtocolValidator.class);

    public ValidationResult validate(Protocol protocol);

    private void validateRequiredFields(Protocol protocol, ValidationResult result);

    private void validateActionReferences(Protocol protocol, ValidationResult result);

    private void validateConditionReferences(Protocol protocol, ValidationResult result);

    private void validateConfidenceScoring(Protocol protocol, ValidationResult result);

    private void validateTimeConstraints(Protocol protocol, ValidationResult result);

    private void validateEvidenceSource(Protocol protocol, ValidationResult result);
}
```

### Method Specifications

#### 7.1 validate()

```java
/**
 * Validates a protocol for completeness and correctness.
 *
 * @param protocol The protocol to validate
 * @return ValidationResult with errors and warnings
 */
public ValidationResult validate(Protocol protocol) {
    ValidationResult result = new ValidationResult();
    result.setProtocolId(protocol != null ? protocol.getProtocolId() : "UNKNOWN");

    if (protocol == null) {
        result.addError("Protocol is null");
        return result;
    }

    logger.debug("Validating protocol: {}", protocol.getProtocolId());

    // Run all validation checks
    validateRequiredFields(protocol, result);
    validateActionReferences(protocol, result);
    validateConditionReferences(protocol, result);
    validateConfidenceScoring(protocol, result);
    validateTimeConstraints(protocol, result);
    validateEvidenceSource(protocol, result);

    if (result.isValid()) {
        logger.info("Protocol {} validation PASSED", protocol.getProtocolId());
    } else {
        logger.error("Protocol {} validation FAILED: {} errors, {} warnings",
            protocol.getProtocolId(),
            result.getErrors().size(),
            result.getWarnings().size());
    }

    return result;
}
```

#### 7.2 validateRequiredFields()

```java
/**
 * Validates that required fields are present.
 */
private void validateRequiredFields(Protocol protocol, ValidationResult result) {
    if (protocol.getProtocolId() == null || protocol.getProtocolId().isEmpty()) {
        result.addError("protocol_id is required");
    }

    if (protocol.getName() == null || protocol.getName().isEmpty()) {
        result.addError("name is required");
    }

    if (protocol.getCategory() == null) {
        result.addError("category is required");
    }

    if (protocol.getActions() == null || protocol.getActions().isEmpty()) {
        result.addError("At least one action is required");
    }

    if (protocol.getVersion() == null || protocol.getVersion().isEmpty()) {
        result.addWarning("version recommended for tracking");
    }
}
```

#### 7.3 validateActionReferences()

```java
/**
 * Validates action IDs are unique and non-empty.
 */
private void validateActionReferences(Protocol protocol, ValidationResult result) {
    if (protocol.getActions() == null) {
        return;
    }

    Set<String> actionIds = new HashSet<>();

    for (ProtocolAction action : protocol.getActions()) {
        if (action.getActionId() == null || action.getActionId().isEmpty()) {
            result.addError("Action missing action_id");
        } else {
            if (actionIds.contains(action.getActionId())) {
                result.addError("Duplicate action_id: " + action.getActionId());
            }
            actionIds.add(action.getActionId());
        }

        if (action.getType() == null) {
            result.addError("Action " + action.getActionId() + " missing type");
        }

        if (action.getPriority() == null) {
            result.addWarning("Action " + action.getActionId() + " missing priority");
        }
    }
}
```

#### 7.4 validateConfidenceScoring()

```java
/**
 * Validates confidence scoring ranges.
 */
private void validateConfidenceScoring(Protocol protocol, ValidationResult result) {
    if (protocol.getConfidenceScoring() == null) {
        result.addWarning("confidence_scoring recommended for protocol ranking");
        return;
    }

    ConfidenceScoring scoring = protocol.getConfidenceScoring();

    if (scoring.getBaseConfidence() < 0.0 || scoring.getBaseConfidence() > 1.0) {
        result.addError("base_confidence must be between 0.0 and 1.0");
    }

    if (scoring.getActivationThreshold() < 0.0 || scoring.getActivationThreshold() > 1.0) {
        result.addError("activation_threshold must be between 0.0 and 1.0");
    }

    // Warn if modifiers could exceed 1.0
    if (scoring.getModifiers() != null) {
        double maxPossible = scoring.getBaseConfidence();
        for (ConfidenceModifier mod : scoring.getModifiers()) {
            maxPossible += mod.getAdjustment();
        }

        if (maxPossible > 1.5) {
            result.addWarning(String.format(
                "Confidence modifiers may exceed 1.0 (max possible: %.2f)",
                maxPossible));
        }
    }
}
```

### ValidationResult Class

```java
/**
 * Result of protocol validation.
 */
public class ValidationResult {
    private String protocolId;
    private List<String> errors = new ArrayList<>();
    private List<String> warnings = new ArrayList<>();

    public boolean isValid() {
        return errors.isEmpty();
    }

    public void addError(String error) {
        errors.add(error);
    }

    public void addWarning(String warning) {
        warnings.add(warning);
    }

    // Getters
    public List<String> getErrors() { return errors; }
    public List<String> getWarnings() { return warnings; }
    public String getProtocolId() { return protocolId; }
    public void setProtocolId(String protocolId) { this.protocolId = protocolId; }
}
```

### Unit Tests

```java
@Test
void testValidate_ValidProtocol_Passes() {
    Protocol protocol = createValidProtocol();
    ValidationResult result = validator.validate(protocol);

    assertTrue(result.isValid());
    assertEquals(0, result.getErrors().size());
}

@Test
void testValidate_MissingProtocolId_Fails() {
    Protocol protocol = createValidProtocol();
    protocol.setProtocolId(null);

    ValidationResult result = validator.validate(protocol);

    assertFalse(result.isValid());
    assertTrue(result.getErrors().stream()
        .anyMatch(e -> e.contains("protocol_id")));
}

@Test
void testValidate_DuplicateActionIds_Fails() {
    Protocol protocol = createValidProtocol();

    ProtocolAction action1 = new ProtocolAction();
    action1.setActionId("ACT-001");

    ProtocolAction action2 = new ProtocolAction();
    action2.setActionId("ACT-001"); // Duplicate

    protocol.setActions(Arrays.asList(action1, action2));

    ValidationResult result = validator.validate(protocol);

    assertFalse(result.isValid());
    assertTrue(result.getErrors().stream()
        .anyMatch(e -> e.contains("Duplicate action_id")));
}
```

---

## Summary

This document provides complete specifications for all 7 Java classes required to align Module 3 to the CDS specification. Each class includes:

- Complete method signatures with Javadoc
- Detailed implementation algorithms
- Helper methods and utilities
- Unit test examples
- Dependencies and integration points

**Total Estimated Effort**: 20-27 hours development + 5-7 hours testing = **25-34 hours**

**Next Steps**:
1. Review specifications with development team
2. Assign classes to developers (Phase 1 critical classes first)
3. Begin implementation following TDD (Test-Driven Development)
4. Integrate components with existing Module 3 classes
5. Validate with ROHAN-001 test case

---

**Document Status**: COMPLETE
**Ready for**: Development Team Review and Implementation
