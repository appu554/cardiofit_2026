package com.cardiofit.flink.processors;

import com.cardiofit.flink.cds.evaluation.ConditionEvaluator;
import com.cardiofit.flink.cds.evaluation.ConfidenceCalculator;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.protocol.models.Protocol;
import com.cardiofit.flink.models.protocol.TriggerCriteria;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Protocol Matcher - Match patient conditions to applicable clinical protocols
 *
 * Evaluates protocol activation criteria against patient context to determine
 * which evidence-based protocols should be recommended for the patient.
 *
 * Matching Process:
 * 1. Evaluate activation criteria (clinical triggers from protocol YAML) using ConditionEvaluator
 * 2. Calculate confidence score (0.0-1.0) based on how well patient matches
 * 3. Filter out low-confidence matches (< threshold)
 * 4. Sort by confidence descending
 *
 * Activation Criteria Evaluation:
 * - Clinical scores (NEWS2, qSOFA, etc.)
 * - Lab value thresholds
 * - Vital sign abnormalities
 * - Diagnostic codes
 * - Time-based criteria
 *
 * Phase 1 Integration:
 * - Uses ConditionEvaluator to evaluate trigger_criteria from protocol YAML
 * - Supports ALL_OF (AND) and ANY_OF (OR) match logic
 * - Handles nested conditions and comparison operators
 *
 * Phase 2 Integration:
 * - Uses ConfidenceCalculator for protocol ranking
 * - Filters by activation_threshold from confidence_scoring
 * - Ranks protocols by confidence score descending
 *
 * @author CardioFit Platform - Module 3
 * @version 1.2 - Phase 2 Integration
 * @since 2025-10-20
 */
public class ProtocolMatcher implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ProtocolMatcher.class);

    /**
     * Condition evaluator for trigger criteria evaluation (Phase 1 integration)
     */
    private final ConditionEvaluator conditionEvaluator;

    /**
     * Confidence calculator for protocol ranking (Phase 2 integration)
     */
    private final ConfidenceCalculator confidenceCalculator;

    /**
     * Minimum confidence threshold for protocol matching (0.0-1.0)
     * Protocols with confidence below this threshold are filtered out
     */
    private static final double MIN_CONFIDENCE_THRESHOLD = 0.5;

    /**
     * Maximum number of protocols to match per patient
     * Prevents overwhelming clinicians with too many recommendations
     */
    private static final int MAX_MATCHED_PROTOCOLS = 3;

    /**
     * Constructor with dependency injection (Phase 2 integration).
     *
     * @param conditionEvaluator Evaluator for trigger criteria
     * @param confidenceCalculator Calculator for protocol confidence scores
     */
    public ProtocolMatcher(ConditionEvaluator conditionEvaluator, ConfidenceCalculator confidenceCalculator) {
        this.conditionEvaluator = conditionEvaluator;
        this.confidenceCalculator = confidenceCalculator;
    }

    /**
     * Constructor with ConditionEvaluator only (Phase 1 compatibility).
     *
     * @param conditionEvaluator Evaluator for trigger criteria
     */
    public ProtocolMatcher(ConditionEvaluator conditionEvaluator) {
        this.conditionEvaluator = conditionEvaluator;
        this.confidenceCalculator = new ConfidenceCalculator(conditionEvaluator);
    }

    /**
     * Default constructor (creates new ConditionEvaluator and ConfidenceCalculator).
     */
    public ProtocolMatcher() {
        this.conditionEvaluator = new ConditionEvaluator();
        this.confidenceCalculator = new ConfidenceCalculator(this.conditionEvaluator);
    }

    /**
     * Match and rank protocols by confidence score (Phase 2 integration).
     *
     * Process:
     * 1. Iterate through all protocols in the map
     * 2. If protocol has trigger_criteria, evaluate using ConditionEvaluator
     * 3. If triggered, calculate confidence using ConfidenceCalculator
     * 4. Filter by protocol's activation_threshold (from confidence_scoring)
     * 5. Sort by confidence descending (highest confidence first)
     *
     * @param context Patient clinical context
     * @param protocols Map of protocol_id -> Protocol objects
     * @return List of matched protocols ranked by confidence
     */
    public List<ProtocolMatch> matchProtocolsRanked(
            EnrichedPatientContext context,
            Map<String, Protocol> protocols) {

        if (context == null || protocols == null || protocols.isEmpty()) {
            LOG.warn("Cannot match protocols: null context or empty protocol library");
            return new ArrayList<>();
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            LOG.warn("Cannot match protocols: patient state is null for patient {}", context.getPatientId());
            return new ArrayList<>();
        }

        List<ProtocolMatch> matches = new ArrayList<>();

        // Evaluate each protocol against patient context
        for (Protocol protocol : protocols.values()) {
            String protocolId = protocol.getProtocolId();

            try {
                // Check if protocol has trigger_criteria
                TriggerCriteria triggerCriteria = protocol.getTriggerCriteria();

                if (triggerCriteria != null) {
                    // Use ConditionEvaluator to evaluate trigger criteria
                    boolean triggered = conditionEvaluator.evaluate(triggerCriteria, context);

                    if (triggered) {
                        LOG.info("Protocol {} triggered for patient {} - trigger criteria met",
                                protocolId, context.getPatientId());

                        // Convert protocol to models.protocol.Protocol for ConfidenceCalculator
                        com.cardiofit.flink.models.protocol.Protocol modelProtocol = convertToModelProtocol(protocol);

                        // Calculate confidence score using ConfidenceCalculator
                        double confidence = confidenceCalculator.calculateConfidence(modelProtocol, context);

                        // Get activation threshold from protocol's confidence_scoring
                        double activationThreshold = protocol.getConfidenceScoring() != null
                                ? protocol.getConfidenceScoring().getActivationThreshold()
                                : ConfidenceCalculator.DEFAULT_ACTIVATION_THRESHOLD;

                        // Filter by activation threshold
                        if (confidence >= activationThreshold) {
                            ProtocolMatch match = new ProtocolMatch(protocolId, protocol, confidence);
                            matches.add(match);

                            LOG.debug("Protocol {} matched for patient {} with confidence {} (threshold: {})",
                                    protocolId, context.getPatientId(), confidence, activationThreshold);
                        } else {
                            LOG.debug("Protocol {} triggered but below activation threshold {} (actual: {})",
                                    protocolId, activationThreshold, confidence);
                        }
                    } else {
                        LOG.debug("Protocol {} trigger criteria NOT met for patient {}",
                                protocolId, context.getPatientId());
                    }
                } else {
                    LOG.debug("Protocol {} has no trigger_criteria, skipping",
                            protocolId);
                }
            } catch (Exception e) {
                LOG.error("Error matching protocol {} for patient {}: {}",
                        protocolId, context.getPatientId(), e.getMessage(), e);
            }
        }

        // Sort by confidence descending (highest confidence first)
        matches.sort((a, b) -> Double.compare(b.getConfidence(), a.getConfidence()));

        LOG.info("Matched {} protocols for patient {} (ranked by confidence)", matches.size(), context.getPatientId());
        return matches;
    }

    /**
     * Match applicable protocols to patient's clinical context (Phase 1 enhanced).
     *
     * Process:
     * 1. Iterate through all protocols
     * 2. If protocol has trigger_criteria, evaluate using ConditionEvaluator
     * 3. If triggered, calculate confidence score
     * 4. Filter by minimum threshold and sort by confidence
     *
     * @param context Patient clinical context
     * @param protocols List of Protocol objects
     * @return List of matched protocols sorted by confidence (highest first)
     */
    public List<ProtocolMatch> matchProtocols(
            EnrichedPatientContext context,
            List<Protocol> protocols) {

        if (context == null || protocols == null || protocols.isEmpty()) {
            LOG.warn("Cannot match protocols: null context or empty protocol list");
            return new ArrayList<>();
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            LOG.warn("Cannot match protocols: patient state is null for patient {}", context.getPatientId());
            return new ArrayList<>();
        }

        List<ProtocolMatch> matches = new ArrayList<>();

        // Evaluate each protocol against patient context
        for (Protocol protocol : protocols) {
            String protocolId = protocol.getProtocolId();

            try {
                // Check if protocol has trigger_criteria
                TriggerCriteria triggerCriteria = protocol.getTriggerCriteria();

                if (triggerCriteria != null) {
                    // Use ConditionEvaluator to evaluate trigger criteria
                    boolean triggered = conditionEvaluator.evaluate(triggerCriteria, context);

                    if (triggered) {
                        LOG.info("Protocol {} triggered for patient {} - trigger criteria met",
                                protocolId, context.getPatientId());

                        // Calculate confidence score
                        double confidence = calculateConfidenceForProtocol(protocol, context);

                        // Filter by minimum threshold
                        if (confidence >= MIN_CONFIDENCE_THRESHOLD) {
                            ProtocolMatch match = new ProtocolMatch(protocolId, protocol, confidence);
                            matches.add(match);

                            LOG.debug("Protocol {} matched for patient {} with confidence {}",
                                    protocolId, context.getPatientId(), confidence);
                        } else {
                            LOG.debug("Protocol {} triggered but below confidence threshold {} (actual: {})",
                                    protocolId, MIN_CONFIDENCE_THRESHOLD, confidence);
                        }
                    } else {
                        LOG.debug("Protocol {} trigger criteria NOT met for patient {}",
                                protocolId, context.getPatientId());
                    }
                } else {
                    // Fallback to legacy activation criteria if no trigger_criteria
                    LOG.debug("Protocol {} has no trigger_criteria, using legacy activation",
                            protocolId);
                    if (evaluateActivationCriteriaLegacy(protocol, context)) {
                        double confidence = calculateConfidenceForProtocol(protocol, context);
                        if (confidence >= MIN_CONFIDENCE_THRESHOLD) {
                            ProtocolMatch match = new ProtocolMatch(protocolId, protocol, confidence);
                            matches.add(match);
                        }
                    }
                }
            } catch (Exception e) {
                LOG.error("Error matching protocol {} for patient {}: {}",
                        protocolId, context.getPatientId(), e.getMessage(), e);
            }
        }

        // Sort by confidence descending
        matches.sort(Comparator.comparingDouble(ProtocolMatch::getConfidence).reversed());

        // Limit to top N protocols
        if (matches.size() > MAX_MATCHED_PROTOCOLS) {
            matches = matches.subList(0, MAX_MATCHED_PROTOCOLS);
            LOG.debug("Limited protocol matches to top {} for patient {}",
                    MAX_MATCHED_PROTOCOLS, context.getPatientId());
        }

        LOG.info("Matched {} protocols for patient {}", matches.size(), context.getPatientId());
        return matches;
    }

    /**
     * Legacy method for backward compatibility with Map-based protocols.
     *
     * @param context Patient clinical context
     * @param protocols Map of protocol_id -> protocol definition
     * @return List of matched protocols sorted by confidence (highest first)
     * @deprecated Use matchProtocols(context, List<Protocol>) instead
     */
    @Deprecated
    public List<ProtocolMatch> matchProtocolsLegacy(
            EnrichedPatientContext context,
            Map<String, Map<String, Object>> protocols) {

        if (context == null || protocols == null || protocols.isEmpty()) {
            LOG.warn("Cannot match protocols: null context or empty protocol library");
            return new ArrayList<>();
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            LOG.warn("Cannot match protocols: patient state is null for patient {}", context.getPatientId());
            return new ArrayList<>();
        }

        List<ProtocolMatch> matches = new ArrayList<>();

        // Evaluate each protocol against patient context
        for (Map.Entry<String, Map<String, Object>> entry : protocols.entrySet()) {
            String protocolId = entry.getKey();
            Map<String, Object> protocol = entry.getValue();

            try {
                // Check if activation criteria are met
                if (evaluateActivationCriteria(protocol, context)) {
                    // Calculate confidence score
                    double confidence = calculateConfidence(protocol, context);

                    // Filter by minimum threshold
                    if (confidence >= MIN_CONFIDENCE_THRESHOLD) {
                        ProtocolMatch match = new ProtocolMatch(protocolId, protocol, confidence);
                        matches.add(match);

                        LOG.debug("Protocol {} matched for patient {} with confidence {}",
                                protocolId, context.getPatientId(), confidence);
                    }
                }
            } catch (Exception e) {
                LOG.error("Error matching protocol {} for patient {}: {}",
                        protocolId, context.getPatientId(), e.getMessage(), e);
            }
        }

        // Sort by confidence descending
        matches.sort(Comparator.comparingDouble(ProtocolMatch::getConfidence).reversed());

        // Limit to top N protocols
        if (matches.size() > MAX_MATCHED_PROTOCOLS) {
            matches = matches.subList(0, MAX_MATCHED_PROTOCOLS);
            LOG.debug("Limited protocol matches to top {} for patient {}",
                    MAX_MATCHED_PROTOCOLS, context.getPatientId());
        }

        LOG.info("Matched {} protocols for patient {}", matches.size(), context.getPatientId());
        return matches;
    }

    /**
     * Calculate confidence score for a Protocol object.
     *
     * @param protocol Protocol object
     * @param context Patient context
     * @return Confidence score 0.0-1.0
     */
    private double calculateConfidenceForProtocol(Protocol protocol, EnrichedPatientContext context) {
        PatientContextState state = context.getPatientState();

        double baseConfidence = 0.5; // Start at 50%

        // Factor 1: Clinical score severity (0.0-0.3)
        double scoreComponent = calculateScoreSeverityComponent(state);

        // Factor 2: Data completeness (0.0-0.1)
        double dataComponent = calculateDataCompletenessComponent(state);

        // Factor 3: Alert presence (0.0-0.1)
        double alertComponent = calculateAlertComponent(state);

        double totalConfidence = baseConfidence + scoreComponent + dataComponent + alertComponent;

        // Cap at 1.0
        return Math.min(totalConfidence, 1.0);
    }

    /**
     * Legacy activation criteria evaluation (backward compatibility).
     *
     * @param protocol Protocol object
     * @param context Patient context
     * @return true if activation criteria are met
     */
    private boolean evaluateActivationCriteriaLegacy(Protocol protocol, EnrichedPatientContext context) {
        String protocolId = protocol.getProtocolId();
        PatientContextState state = context.getPatientState();

        // Protocol-specific activation logic
        if ("SEPSIS-001".equals(protocolId)) {
            return evaluateSepsisActivation(state);
        } else if ("STEMI-001".equals(protocolId)) {
            return evaluateStemActivation(state);
        } else if ("RESPIRATORY-FAILURE-001".equals(protocolId)) {
            return evaluateRespiratoryFailureActivation(state);
        }

        // Default: check for high NEWS2 score as generic activation criteria
        Integer news2 = state.getNews2Score();
        return news2 != null && news2 >= 5;
    }

    /**
     * Evaluate if protocol activation criteria are met (Map-based legacy).
     *
     * Checks protocol's activation_criteria section against patient state
     *
     * @param protocol Protocol definition (Map)
     * @param context Patient context
     * @return true if activation criteria are met
     * @deprecated Use Protocol objects with trigger_criteria instead
     */
    @Deprecated
    public boolean evaluateActivationCriteria(
            Map<String, Object> protocol,
            EnrichedPatientContext context) {

        String protocolId = (String) protocol.get("protocol_id");
        PatientContextState state = context.getPatientState();

        // Protocol-specific activation logic
        // In production, this would parse activation_criteria from YAML
        // For now, we implement specific logic for known protocols

        if ("SEPSIS-001".equals(protocolId)) {
            return evaluateSepsisActivation(state);
        } else if ("STEMI-001".equals(protocolId)) {
            return evaluateStemActivation(state);
        } else if ("RESPIRATORY-FAILURE-001".equals(protocolId)) {
            return evaluateRespiratoryFailureActivation(state);
        }

        // Default: check for high NEWS2 score as generic activation criteria
        Integer news2 = state.getNews2Score();
        return news2 != null && news2 >= 5;
    }

    /**
     * Evaluate sepsis protocol activation criteria
     *
     * Criteria:
     * - NEWS2 >= 5 (early warning score)
     * - qSOFA >= 2 (sepsis screening)
     * - Lactate >= 2.0 mmol/L
     * - Temperature abnormal (< 36°C or > 38°C)
     * - WBC abnormal
     *
     * @param state Patient context state
     * @return true if sepsis criteria met
     */
    private boolean evaluateSepsisActivation(PatientContextState state) {
        int criteriaCount = 0;

        // NEWS2 >= 5
        if (state.getNews2Score() != null && state.getNews2Score() >= 5) {
            criteriaCount++;
        }

        // qSOFA >= 2
        if (state.getQsofaScore() != null && state.getQsofaScore() >= 2) {
            criteriaCount++;
        }

        // Lactate >= 2.0
        if (state.getRecentLabs() != null) {
            Object lactate = state.getRecentLabs().get("lactate");
            if (lactate != null) {
                double lactateValue = extractNumericValue(lactate);
                if (lactateValue >= 2.0) {
                    criteriaCount++;
                }
            }
        }

        // Temperature abnormal
        if (state.getLatestVitals() != null) {
            Object temp = state.getLatestVitals().get("temperature");
            if (temp != null) {
                double tempValue = extractNumericValue(temp);
                if (tempValue < 36.0 || tempValue > 38.0) {
                    criteriaCount++;
                }
            }
        }

        // Require at least 2 criteria for sepsis activation
        return criteriaCount >= 2;
    }

    /**
     * Evaluate STEMI protocol activation criteria
     *
     * Criteria:
     * - Troponin elevated (LOINC 10839-9 > 0.04 ng/mL)
     * - Chest pain alert active
     * - ECG changes (if available)
     *
     * @param state Patient context state
     * @return true if STEMI criteria met
     */
    private boolean evaluateStemActivation(PatientContextState state) {
        boolean troponinElevated = false;
        boolean chestPainPresent = false;

        // Check troponin
        if (state.getRecentLabs() != null) {
            Object troponin = state.getRecentLabs().get("10839-9"); // LOINC for Troponin I
            if (troponin != null) {
                double troponinValue = extractNumericValue(troponin);
                if (troponinValue > 0.04) {
                    troponinElevated = true;
                }
            }
        }

        // Check for chest pain alert
        if (state.getActiveAlerts() != null) {
            chestPainPresent = state.getActiveAlerts().stream()
                    .anyMatch(alert -> alert.getMessage() != null &&
                            alert.getMessage().toLowerCase().contains("chest pain"));
        }

        return troponinElevated || chestPainPresent;
    }

    /**
     * Evaluate respiratory failure protocol activation criteria
     *
     * Criteria:
     * - Oxygen saturation < 90%
     * - Respiratory rate > 24 or < 10
     * - PaCO2 elevated (if available)
     *
     * @param state Patient context state
     * @return true if respiratory failure criteria met
     */
    private boolean evaluateRespiratoryFailureActivation(PatientContextState state) {
        boolean hypoxia = false;
        boolean abnormalRespRate = false;

        if (state.getLatestVitals() != null) {
            // Check oxygen saturation
            Object spo2 = state.getLatestVitals().get("oxygensaturation");
            if (spo2 != null) {
                double spo2Value = extractNumericValue(spo2);
                if (spo2Value < 90.0) {
                    hypoxia = true;
                }
            }

            // Check respiratory rate
            Object respRate = state.getLatestVitals().get("respiratoryrate");
            if (respRate != null) {
                double respRateValue = extractNumericValue(respRate);
                if (respRateValue > 24 || respRateValue < 10) {
                    abnormalRespRate = true;
                }
            }
        }

        return hypoxia || abnormalRespRate;
    }

    /**
     * Calculate confidence score for protocol match
     *
     * Confidence is based on:
     * - How many activation criteria are met (more = higher confidence)
     * - Severity of clinical abnormalities (more severe = higher confidence)
     * - Data completeness (more data = higher confidence)
     * - Temporal recency (recent data = higher confidence)
     *
     * @param protocol Protocol definition
     * @param context Patient context
     * @return Confidence score 0.0-1.0
     */
    public double calculateConfidence(
            Map<String, Object> protocol,
            EnrichedPatientContext context) {

        PatientContextState state = context.getPatientState();
        String protocolId = (String) protocol.get("protocol_id");

        double baseConfidence = 0.5; // Start at 50%

        // Factor 1: Clinical score severity (0.0-0.3)
        double scoreComponent = calculateScoreSeverityComponent(state);

        // Factor 2: Data completeness (0.0-0.1)
        double dataComponent = calculateDataCompletenessComponent(state);

        // Factor 3: Alert presence (0.0-0.1)
        double alertComponent = calculateAlertComponent(state);

        double totalConfidence = baseConfidence + scoreComponent + dataComponent + alertComponent;

        // Cap at 1.0
        return Math.min(totalConfidence, 1.0);
    }

    /**
     * Calculate confidence component from clinical score severity
     */
    private double calculateScoreSeverityComponent(PatientContextState state) {
        double component = 0.0;

        Integer news2 = state.getNews2Score();
        if (news2 != null) {
            if (news2 >= 7) {
                component += 0.3; // Critical score
            } else if (news2 >= 5) {
                component += 0.2; // High score
            } else if (news2 >= 3) {
                component += 0.1; // Medium score
            }
        }

        return component;
    }

    /**
     * Calculate confidence component from data completeness
     */
    private double calculateDataCompletenessComponent(PatientContextState state) {
        int dataPoints = 0;
        int maxDataPoints = 3;

        if (state.getLatestVitals() != null && !state.getLatestVitals().isEmpty()) {
            dataPoints++;
        }
        if (state.getRecentLabs() != null && !state.getRecentLabs().isEmpty()) {
            dataPoints++;
        }
        if (state.getActiveMedications() != null && !state.getActiveMedications().isEmpty()) {
            dataPoints++;
        }

        return (double) dataPoints / maxDataPoints * 0.1;
    }

    /**
     * Calculate confidence component from active alerts
     */
    private double calculateAlertComponent(PatientContextState state) {
        if (state.getActiveAlerts() != null && !state.getActiveAlerts().isEmpty()) {
            return 0.1;
        }
        return 0.0;
    }

    /**
     * Extract numeric value from Object (handles Double, Integer, String)
     */
    private double extractNumericValue(Object value) {
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        } else if (value instanceof String) {
            try {
                return Double.parseDouble((String) value);
            } catch (NumberFormatException e) {
                LOG.warn("Cannot parse numeric value from string: {}", value);
                return 0.0;
            }
        }
        return 0.0;
    }

    /**
     * Convert protocol.models.Protocol to models.protocol.Protocol for ConfidenceCalculator.
     *
     * Phase 2 Integration: Adapter method for type compatibility
     */
    private com.cardiofit.flink.models.protocol.Protocol convertToModelProtocol(Protocol protocolModels) {
        com.cardiofit.flink.models.protocol.Protocol modelProtocol =
                new com.cardiofit.flink.models.protocol.Protocol();

        // Copy basic fields
        modelProtocol.setProtocolId(protocolModels.getProtocolId());
        modelProtocol.setName(protocolModels.getName());
        modelProtocol.setCategory(protocolModels.getCategory());
        modelProtocol.setSpecialty(protocolModels.getSpecialty());
        modelProtocol.setVersion(protocolModels.getVersion());

        // Copy trigger criteria (same package)
        modelProtocol.setTriggerCriteria(protocolModels.getTriggerCriteria());

        // Copy confidence scoring (same package)
        modelProtocol.setConfidenceScoring(protocolModels.getConfidenceScoring());

        return modelProtocol;
    }

    /**
     * Protocol Match result wrapper (Phase 1 enhanced).
     */
    public static class ProtocolMatch implements Serializable {
        private static final long serialVersionUID = 1L;

        private String protocolId;
        private Protocol protocolObject;
        private Map<String, Object> protocolMap; // For backward compatibility
        private double confidence;

        /**
         * Constructor with Protocol object (Phase 1).
         */
        public ProtocolMatch(String protocolId, Protocol protocol, double confidence) {
            this.protocolId = protocolId;
            this.protocolObject = protocol;
            this.confidence = confidence;
        }

        /**
         * Legacy constructor with Map (backward compatibility).
         */
        public ProtocolMatch(String protocolId, Map<String, Object> protocol, double confidence) {
            this.protocolId = protocolId;
            this.protocolMap = protocol;
            this.confidence = confidence;
        }

        public String getProtocolId() {
            return protocolId;
        }

        public Protocol getProtocolObject() {
            return protocolObject;
        }

        public Map<String, Object> getProtocol() {
            return protocolMap;
        }

        public double getConfidence() {
            return confidence;
        }

        @Override
        public String toString() {
            return "ProtocolMatch{" +
                    "protocolId='" + protocolId + '\'' +
                    ", confidence=" + String.format("%.2f", confidence) +
                    '}';
        }
    }
}
