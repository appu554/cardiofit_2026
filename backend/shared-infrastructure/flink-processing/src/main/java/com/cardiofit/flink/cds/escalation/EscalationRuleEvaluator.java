package com.cardiofit.flink.cds.escalation;

import com.cardiofit.flink.cds.evaluation.ConditionEvaluator;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.EscalationRecommendation;
import com.cardiofit.flink.models.PatientState;
import com.cardiofit.flink.models.SimpleAlert;
import com.cardiofit.flink.models.protocol.EscalationRule;
import com.cardiofit.flink.models.protocol.Protocol;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Set;

/**
 * Escalation Rule Evaluator - Module 3 CDS Alignment
 *
 * Evaluates protocol escalation rules and generates evidence-based escalation recommendations.
 * Critical for detecting clinical deterioration and recommending appropriate level of care.
 *
 * Key Features:
 * - Evaluates escalation_rules from enhanced protocol YAML
 * - Detects clinical deterioration patterns
 * - Generates evidence-based rationale for escalation
 * - Supports multiple escalation levels (ICU_TRANSFER, SPECIALIST_CONSULT, etc.)
 *
 * Example Escalation Rule (from YAML):
 * escalation_rules:
 *   - rule_id: "SEPSIS-ESC-001"
 *     escalation_trigger:
 *       parameter: "lactate"
 *       operator: ">="
 *       threshold: 4.0
 *     recommendation:
 *       escalation_level: "ICU_TRANSFER"
 *       specialty: "Critical Care"
 *       rationale: "Septic shock requiring vasopressor support"
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class EscalationRuleEvaluator implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger logger = LoggerFactory.getLogger(EscalationRuleEvaluator.class);

    private final ConditionEvaluator conditionEvaluator;

    public EscalationRuleEvaluator(ConditionEvaluator conditionEvaluator) {
        this.conditionEvaluator = conditionEvaluator;
    }

    /**
     * Evaluate all escalation rules for a protocol and generate recommendations
     *
     * @param protocol The clinical protocol with escalation_rules
     * @param context Current patient context
     * @return List of triggered escalation recommendations (empty if none triggered)
     */
    public List<EscalationRecommendation> evaluateEscalationRules(
        Protocol protocol,
        EnrichedPatientContext context
    ) {
        List<EscalationRecommendation> recommendations = new ArrayList<>();

        if (protocol.getEscalationRules() == null || protocol.getEscalationRules().isEmpty()) {
            logger.debug("No escalation rules defined for protocol {}", protocol.getProtocolId());
            return recommendations; // No escalation rules defined
        }

        logger.debug("Evaluating {} escalation rules for protocol {} and patient {}",
                protocol.getEscalationRules().size(),
                protocol.getProtocolId(),
                context.getPatientId());

        for (EscalationRule rule : protocol.getEscalationRules()) {
            boolean triggered = conditionEvaluator.evaluateCondition(
                rule.getEscalationTrigger(),
                context,
                0 // Start at depth 0
            );

            if (triggered) {
                EscalationRecommendation recommendation = buildEscalationRecommendation(
                    rule,
                    protocol,
                    context
                );

                recommendations.add(recommendation);

                logger.warn("Escalation triggered for patient {}: {} - {}",
                    context.getPatientId(),
                    rule.getRuleId(),
                    rule.getRecommendation().getEscalationLevel());
            } else {
                logger.debug("Escalation rule {} not triggered for patient {}",
                        rule.getRuleId(), context.getPatientId());
            }
        }

        logger.info("Escalation evaluation complete for protocol {}: {} recommendations generated",
                protocol.getProtocolId(), recommendations.size());

        return recommendations;
    }

    /**
     * Build detailed escalation recommendation with evidence
     */
    private EscalationRecommendation buildEscalationRecommendation(
        EscalationRule rule,
        Protocol protocol,
        EnrichedPatientContext context
    ) {
        EscalationRecommendation recommendation = new EscalationRecommendation();

        recommendation.setRuleId(rule.getRuleId());
        recommendation.setProtocolId(protocol.getProtocolId());
        recommendation.setProtocolName(protocol.getName());
        recommendation.setEscalationLevel(rule.getRecommendation().getEscalationLevel());
        recommendation.setSpecialty(rule.getRecommendation().getSpecialty());
        recommendation.setRationale(rule.getRecommendation().getRationale());
        recommendation.setUrgency(rule.getRecommendation().getUrgency());
        recommendation.setTimestamp(Instant.now());

        // Gather supporting clinical evidence
        List<String> evidence = gatherClinicalEvidence(rule, context);
        recommendation.setEvidence(evidence);

        // Add patient identifiers for tracking
        recommendation.setPatientId(context.getPatientId());
        recommendation.setEncounterId(context.getEncounterId());

        logger.debug("Built escalation recommendation: {} for patient {} with {} evidence items",
                rule.getRuleId(), context.getPatientId(), evidence.size());

        return recommendation;
    }

    /**
     * Gather clinical evidence supporting escalation decision
     */
    private List<String> gatherClinicalEvidence(
        EscalationRule rule,
        EnrichedPatientContext context
    ) {
        List<String> evidence = new ArrayList<>();

        // Extract parameter value that triggered escalation
        String parameter = rule.getEscalationTrigger().getParameter();
        Object value = extractParameterValue(parameter, context);

        if (value != null) {
            evidence.add(String.format("%s: %s %s threshold",
                parameter,
                value.toString(),
                rule.getEscalationTrigger().getOperator()));
        }

        // Add vital signs if deteriorating
        addVitalSignsEvidence(evidence, context);

        // Add lab values if critical
        addLabValuesEvidence(evidence, context);

        // Add clinical alerts if present
        addClinicalAlertsEvidence(evidence, context);

        logger.debug("Gathered {} evidence items for escalation rule {}",
                evidence.size(), rule.getRuleId());

        return evidence;
    }

    /**
     * Extract parameter value from patient context
     */
    private Object extractParameterValue(String parameter, EnrichedPatientContext context) {
        try {
            return conditionEvaluator.extractParameterValue(parameter, context);
        } catch (Exception e) {
            logger.warn("Could not extract parameter value for {}: {}", parameter, e.getMessage());
            return null;
        }
    }

    /**
     * Add vital signs to evidence if abnormal
     */
    private void addVitalSignsEvidence(List<String> evidence, EnrichedPatientContext context) {
        if (context.getPatientState() == null) {
            return;
        }

        PatientState patientState = (PatientState) context.getPatientState();

        Double heartRate = patientState.getHeartRate();
        if (heartRate != null && (heartRate > 120 || heartRate < 50)) {
            evidence.add(String.format("Heart Rate: %.0f bpm (abnormal)", heartRate));
        }

        Double systolicBP = patientState.getSystolicBP();
        if (systolicBP != null && systolicBP < 90) {
            evidence.add(String.format("Systolic BP: %.0f mmHg (hypotensive)", systolicBP));
        }

        Double spo2 = patientState.getOxygenSaturation();
        if (spo2 != null && spo2 < 92) {
            evidence.add(String.format("SpO2: %.1f%% (hypoxic)", spo2));
        }

        Double respiratoryRate = patientState.getRespiratoryRate();
        if (respiratoryRate != null && (respiratoryRate > 22 || respiratoryRate < 10)) {
            evidence.add(String.format("Respiratory Rate: %.0f breaths/min (abnormal)", respiratoryRate));
        }

        Double temperature = patientState.getTemperature();
        if (temperature != null && (temperature > 38.3 || temperature < 36.0)) {
            evidence.add(String.format("Temperature: %.1f°C (abnormal)", temperature));
        }
    }

    /**
     * Add lab values to evidence if critical
     */
    private void addLabValuesEvidence(List<String> evidence, EnrichedPatientContext context) {
        if (context.getPatientState() == null) {
            return;
        }

        PatientState patientState = (PatientState) context.getPatientState();

        Double lactate = patientState.getLactate();
        if (lactate != null && lactate >= 2.0) {
            evidence.add(String.format("Lactate: %.1f mmol/L (elevated)", lactate));
        }

        Double creatinine = patientState.getCreatinine();
        if (creatinine != null && creatinine >= 2.0) {
            evidence.add(String.format("Creatinine: %.1f mg/dL (elevated)", creatinine));
        }

        Double wbc = patientState.getWhiteBloodCount();
        if (wbc != null && (wbc > 12.0 || wbc < 4.0)) {
            evidence.add(String.format("WBC: %.1f K/uL (abnormal)", wbc));
        }

        Double platelets = patientState.getPlatelets();
        if (platelets != null && platelets < 100.0) {
            evidence.add(String.format("Platelets: %.0f K/uL (thrombocytopenia)", platelets));
        }

        Double procalcitonin = patientState.getProcalcitonin();
        if (procalcitonin != null && procalcitonin > 0.5) {
            evidence.add(String.format("Procalcitonin: %.2f ng/mL (elevated)", procalcitonin));
        }
    }

    /**
     * Add active clinical alerts to evidence
     */
    private void addClinicalAlertsEvidence(List<String> evidence, EnrichedPatientContext context) {
        if (context.getPatientState() == null) {
            return;
        }

        Set<SimpleAlert> alerts = context.getPatientState().getActiveAlerts();
        if (alerts != null && !alerts.isEmpty()) {
            evidence.add(String.format("Active Clinical Alerts: %d present", alerts.size()));

            // Add specific high-priority alerts
            int criticalAlerts = 0;
            for (SimpleAlert alert : alerts) {
                if (alert.getPriorityLevel() != null) {
                    String priority = alert.getPriorityLevel().toString();
                    if ("P0".equals(priority) || "P1".equals(priority) || "P2".equals(priority)) {
                        criticalAlerts++;
                    }
                }
            }

            if (criticalAlerts > 0) {
                evidence.add(String.format("Critical/High Priority Alerts: %d", criticalAlerts));
            }
        }

        // Add acuity score if high
        Double acuityScore = context.getPatientState().getCombinedAcuityScore();
        if (acuityScore != null && acuityScore >= 5.0) {
            evidence.add(String.format("Combined Acuity Score: %.1f (high)", acuityScore));
        }

        // Add NEWS2 score if elevated
        Integer news2 = context.getPatientState().getNews2Score();
        if (news2 != null && news2 >= 5) {
            evidence.add(String.format("NEWS2 Score: %d (elevated)", news2));
        }

        // Add qSOFA score if positive
        Integer qsofa = context.getPatientState().getQsofaScore();
        if (qsofa != null && qsofa >= 2) {
            evidence.add(String.format("qSOFA Score: %d (positive for sepsis)", qsofa));
        }
    }

    /**
     * Evaluate escalation rules for a Map-based protocol (backward compatibility).
     *
     * This method provides backward compatibility for code that still uses Map-based protocols.
     * Returns empty list since escalation rules require structured Protocol objects.
     *
     * @param protocolMap Protocol as Map from YAML
     * @param context Patient context
     * @return Empty list (requires Protocol object for proper escalation evaluation)
     */
    public List<EscalationRecommendation> evaluateEscalationRules(
            java.util.Map<String, Object> protocolMap,
            EnrichedPatientContext context) {

        logger.warn("Map-based protocol escalation evaluation not supported. " +
                "Use Protocol object for escalation rules. Protocol ID: {}",
                protocolMap.get("protocolId"));

        // Return empty list - escalation rules require structured Protocol objects
        return new ArrayList<>();
    }
}
