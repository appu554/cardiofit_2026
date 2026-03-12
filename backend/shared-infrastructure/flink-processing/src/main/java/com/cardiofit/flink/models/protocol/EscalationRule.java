package com.cardiofit.flink.models.protocol;

import java.io.Serializable;

/**
 * Escalation Rule for clinical protocol escalation criteria.
 *
 * Represents a single escalation rule from the protocol YAML escalation_rules section.
 * When triggered, generates an escalation recommendation (e.g., ICU transfer, specialist consult).
 *
 * Example from YAML:
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
 *       urgency: "IMMEDIATE"
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-10-21
 */
public class EscalationRule implements Serializable {
    private static final long serialVersionUID = 1L;

    private String ruleId;
    private ProtocolCondition escalationTrigger;
    private EscalationRecommendationTemplate recommendation;

    public EscalationRule() {
    }

    public EscalationRule(String ruleId) {
        this.ruleId = ruleId;
    }

    // Getters and setters

    public String getRuleId() {
        return ruleId;
    }

    public void setRuleId(String ruleId) {
        this.ruleId = ruleId;
    }

    public ProtocolCondition getEscalationTrigger() {
        return escalationTrigger;
    }

    public void setEscalationTrigger(ProtocolCondition escalationTrigger) {
        this.escalationTrigger = escalationTrigger;
    }

    public EscalationRecommendationTemplate getRecommendation() {
        return recommendation;
    }

    public void setRecommendation(EscalationRecommendationTemplate recommendation) {
        this.recommendation = recommendation;
    }

    @Override
    public String toString() {
        return "EscalationRule{" +
                "ruleId='" + ruleId + '\'' +
                ", escalationLevel=" + (recommendation != null ? recommendation.getEscalationLevel() : null) +
                '}';
    }

    /**
     * Template for escalation recommendation from YAML.
     * Contains the recommendation details before patient-specific evidence is added.
     */
    public static class EscalationRecommendationTemplate implements Serializable {
        private static final long serialVersionUID = 1L;

        private String escalationLevel; // ICU_TRANSFER, SPECIALIST_CONSULT, RAPID_RESPONSE, etc.
        private String specialty; // Critical Care, Cardiology, Infectious Disease, etc.
        private String rationale; // Evidence-based reason for escalation
        private String urgency; // IMMEDIATE, URGENT, ROUTINE

        public EscalationRecommendationTemplate() {
        }

        // Getters and setters

        public String getEscalationLevel() {
            return escalationLevel;
        }

        public void setEscalationLevel(String escalationLevel) {
            this.escalationLevel = escalationLevel;
        }

        public String getSpecialty() {
            return specialty;
        }

        public void setSpecialty(String specialty) {
            this.specialty = specialty;
        }

        public String getRationale() {
            return rationale;
        }

        public void setRationale(String rationale) {
            this.rationale = rationale;
        }

        public String getUrgency() {
            return urgency;
        }

        public void setUrgency(String urgency) {
            this.urgency = urgency;
        }

        @Override
        public String toString() {
            return "EscalationRecommendationTemplate{" +
                    "escalationLevel='" + escalationLevel + '\'' +
                    ", specialty='" + specialty + '\'' +
                    ", urgency='" + urgency + '\'' +
                    '}';
        }
    }
}
