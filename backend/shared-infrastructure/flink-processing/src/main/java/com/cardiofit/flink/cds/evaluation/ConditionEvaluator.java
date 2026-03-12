package com.cardiofit.flink.cds.evaluation;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.PatientState;
import com.cardiofit.flink.models.protocol.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.List;

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
    private static final int MAX_RECURSION_DEPTH = 4;

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
        List<ProtocolCondition> conditions = trigger.getConditions();

        if (conditions == null || conditions.isEmpty()) {
            logger.warn("[COND-EVAL] No conditions defined in trigger criteria");
            return false;
        }

        logger.info("[COND-EVAL] START - Evaluating trigger with {} logic, {} conditions for patient {}",
                matchLogic, conditions.size(), context.getPatientId());

        if (matchLogic == MatchLogic.ALL_OF) {
            // AND logic - all conditions must be true
            int condIndex = 0;
            for (ProtocolCondition condition : conditions) {
                condIndex++;
                logger.info("[COND-EVAL] [{}/{}] Evaluating ALL_OF condition: {}",
                    condIndex, conditions.size(), condition.getConditionId());

                boolean result = evaluateCondition(condition, context, 0);

                logger.info("[COND-EVAL] [{}/{}] Condition {} result: {}",
                    condIndex, conditions.size(), condition.getConditionId(),
                    result ? "✅ PASS" : "❌ FAIL");

                if (!result) {
                    logger.info("[COND-EVAL] ALL_OF short-circuit: condition {} FAILED - returning false",
                        condition.getConditionId());
                    return false; // Short-circuit on first false
                }
            }
            logger.info("[COND-EVAL] ✅ ALL_OF evaluation: all {} conditions satisfied", conditions.size());
            return true;

        } else if (matchLogic == MatchLogic.ANY_OF) {
            // OR logic - at least one condition must be true
            int condIndex = 0;
            for (ProtocolCondition condition : conditions) {
                condIndex++;
                logger.info("[COND-EVAL] [{}/{}] Evaluating ANY_OF condition: {}",
                    condIndex, conditions.size(), condition.getConditionId());

                boolean result = evaluateCondition(condition, context, 0);

                logger.info("[COND-EVAL] [{}/{}] Condition {} result: {}",
                    condIndex, conditions.size(), condition.getConditionId(),
                    result ? "✅ PASS" : "❌ FAIL");

                if (result) {
                    logger.info("[COND-EVAL] ✅ ANY_OF short-circuit: condition {} SUCCEEDED - returning true",
                        condition.getConditionId());
                    return true; // Short-circuit on first true
                }
            }
            logger.info("[COND-EVAL] ❌ ANY_OF evaluation: NONE of {} conditions satisfied", conditions.size());
            return false;

        } else {
            logger.error("[COND-EVAL] ❌ Unknown match logic: {}", matchLogic);
            return false;
        }
    }

    /**
     * Evaluates a single condition, handling both leaf conditions and nested conditions.
     *
     * @param condition The condition to evaluate (may contain nested conditions)
     * @param context The patient context
     * @param depth Current recursion depth
     * @return true if condition is satisfied
     */
    public boolean evaluateCondition(ProtocolCondition condition, EnrichedPatientContext context, int depth) {
        // Prevent infinite recursion
        if (depth >= MAX_RECURSION_DEPTH) {
            logger.error("Maximum recursion depth {} exceeded", MAX_RECURSION_DEPTH);
            return false;
        }

        // Handle nested conditions (recursive case)
        if (condition.getConditions() != null && !condition.getConditions().isEmpty()) {
            logger.debug("Evaluating nested condition with {} sub-conditions at depth {}",
                    condition.getConditions().size(), depth);

            TriggerCriteria nestedTrigger = new TriggerCriteria();
            nestedTrigger.setMatchLogic(condition.getMatchLogic());
            nestedTrigger.setConditions(condition.getConditions());

            return evaluateNested(nestedTrigger, context, depth + 1); // RECURSION
        }

        // Handle leaf condition (base case)
        String parameter = condition.getParameter();
        ComparisonOperator operator = condition.getOperator();
        Object expectedValue = condition.getThreshold();

        logger.info("[COND-EVAL] Leaf condition: parameter='{}', operator='{}', threshold='{}'",
                parameter, operator, expectedValue);

        if (parameter == null || operator == null || expectedValue == null) {
            logger.warn("[COND-EVAL] ❌ Incomplete condition: parameter={}, operator={}, threshold={}",
                    parameter, operator, expectedValue);
            return false;
        }

        // Extract actual value from patient context
        logger.info("[COND-EVAL] Extracting parameter '{}' from patient context...", parameter);
        Object actualValue = extractParameterValue(parameter, context);

        if (actualValue == null) {
            logger.warn("[COND-EVAL] ❌ Parameter '{}' returned NULL - condition FAILS", parameter);
            return false;
        }

        logger.info("[COND-EVAL] Parameter '{}' extracted: {} (type: {})",
                parameter, actualValue, actualValue.getClass().getSimpleName());

        // Compare values using operator
        logger.info("[COND-EVAL] Comparing: {} {} {} (actual={}, expected={})",
                parameter, operator, expectedValue, actualValue, expectedValue);

        boolean result = compareValues(actualValue, expectedValue, operator);

        logger.info("[COND-EVAL] Comparison result: {} {} {} = {}",
                parameter, operator, expectedValue, result ? "✅ TRUE" : "❌ FALSE");

        return result;
    }

    /**
     * Internal method for nested evaluation to track depth.
     */
    private boolean evaluateNested(TriggerCriteria trigger, EnrichedPatientContext context, int depth) {
        MatchLogic matchLogic = trigger.getMatchLogic();
        List<ProtocolCondition> conditions = trigger.getConditions();

        if (matchLogic == MatchLogic.ALL_OF) {
            for (ProtocolCondition condition : conditions) {
                if (!evaluateCondition(condition, context, depth)) {
                    return false;
                }
            }
            return true;
        } else {
            for (ProtocolCondition condition : conditions) {
                if (evaluateCondition(condition, context, depth)) {
                    return true;
                }
            }
            return false;
        }
    }

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
    public boolean compareValues(Object actualValue, Object expectedValue, ComparisonOperator operator) {
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
                    // Use slightly larger tolerance to handle floating point precision (1e-4 = 0.0001)
                    return Math.abs(toDouble(actualValue) - toDouble(expectedValue)) < 0.00011;
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

    /**
     * Extracts a parameter value from the patient context.
     *
     * Supports dotted notation for nested fields (e.g., "vital_signs.systolic_bp").
     *
     * @param parameter The parameter name (e.g., "lactate", "systolic_bp", "allergies")
     * @param context The patient context
     * @return The parameter value, or null if not found
     */
    public Object extractParameterValue(String parameter, EnrichedPatientContext context) {
        if (parameter == null || parameter.isEmpty()) {
            return null;
        }

        PatientContextState contextState = context.getPatientState();
        if (contextState == null) {
            logger.warn("PatientState is null in context");
            return null;
        }

        // Cast to PatientState to access vital signs methods
        // If contextState is not a PatientState instance, most vital signs will be unavailable
        PatientState patientState;
        if (contextState instanceof PatientState) {
            patientState = (PatientState) contextState;
        } else {
            // For ranking tests that use PatientContextState directly,
            // create a PatientState wrapper to avoid ClassCastException
            patientState = new PatientState();
            // Copy common fields from PatientContextState
            patientState.setNews2Score(contextState.getNews2Score());
            patientState.setAllergies(contextState.getAllergies());
            patientState.setLatestVitals(contextState.getLatestVitals());
            // CRITICAL: Copy RiskIndicators for protocol matching (especially sepsisRisk)
            patientState.setRiskIndicators(contextState.getRiskIndicators());
            logger.info("[PARAM-EXTRACT] Copied RiskIndicators to PatientState wrapper (sepsisRisk={})",
                contextState.getRiskIndicators() != null ? contextState.getRiskIndicators().getSepsisRisk() : null);
        }

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

            case "platelets":
                return patientState.getPlatelets();

            case "inr":
                return patientState.getINR();

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
                List<String> allergies = patientState.getAllergies();
                // Convert to string for CONTAINS operations
                return allergies != null ? String.join(",", allergies) : "";

            case "infection_suspected":
                boolean infectionSuspected = patientState.isInfectionSuspected();
                logger.info("[PARAM-EXTRACT] infection_suspected → {} (from patientState.isInfectionSuspected())",
                    infectionSuspected);
                return infectionSuspected;

            case "pregnancy_status":
                return patientState.isPregnant();

            case "immunosuppressed":
                return patientState.isImmunosuppressed();

            // Scores
            case "news2":  // Alias for news2_score
            case "news2_score":
                return patientState.getNews2Score();

            case "qsofa":  // Alias for qsofa_score
            case "qsofa_score":
                Integer qsofaScore = patientState.getQsofaScore();
                logger.info("[PARAM-EXTRACT] qsofa_score → {} (from patientState.getQsofaScore())",
                    qsofaScore);
                return qsofaScore;

            case "sirs":  // Alias for sirs_score
            case "sirs_score":
                Integer sirsScore = patientState.getSirsScore();
                logger.info("[PARAM-EXTRACT] sirs_score → {} (from patientState.getSirsScore())",
                    sirsScore);
                return sirsScore;

            case "sofa_score":
                return patientState.getSofaScore();

            case "child_pugh_score":
                return patientState.getChildPughScore();

            case "number_of_failing_organs":
                return patientState.getNumberOfFailingOrgans();

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
}
