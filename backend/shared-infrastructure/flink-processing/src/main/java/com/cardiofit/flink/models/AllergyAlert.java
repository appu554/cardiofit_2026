package com.cardiofit.flink.models;

import java.io.Serializable;
import java.time.LocalDateTime;

/**
 * AllergyAlert represents an allergy-related alert for a patient
 */
public class AllergyAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    private String alertId;
    private String patientId;
    private String allergen;
    private String allergyType;
    private String severity;
    private String reaction;
    private String triggerMedication;
    private LocalDateTime alertTime;
    private boolean requiresImmediateAction;
    private String recommendedAction;

    public AllergyAlert() {}

    public AllergyAlert(String alertId, String patientId, String allergen, String severity) {
        this.alertId = alertId;
        this.patientId = patientId;
        this.allergen = allergen;
        this.severity = severity;
        this.alertTime = LocalDateTime.now();
    }

    // Getters and Setters
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getAllergen() { return allergen; }
    public void setAllergen(String allergen) { this.allergen = allergen; }

    public String getAllergyType() { return allergyType; }
    public void setAllergyType(String allergyType) { this.allergyType = allergyType; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public String getReaction() { return reaction; }
    public void setReaction(String reaction) { this.reaction = reaction; }

    public String getTriggerMedication() { return triggerMedication; }
    public void setTriggerMedication(String triggerMedication) { this.triggerMedication = triggerMedication; }

    public LocalDateTime getAlertTime() { return alertTime; }
    public void setAlertTime(LocalDateTime alertTime) { this.alertTime = alertTime; }

    public boolean isRequiresImmediateAction() { return requiresImmediateAction; }
    public void setRequiresImmediateAction(boolean requiresImmediateAction) {
        this.requiresImmediateAction = requiresImmediateAction;
    }

    public String getRecommendedAction() { return recommendedAction; }
    public void setRecommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; }

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String alertId;
        private String patientId;
        private String allergen;
        private String allergyType;
        private String severity;
        private String reaction;
        private String triggerMedication;
        private LocalDateTime alertTime;
        private boolean requiresImmediateAction;
        private String recommendedAction;

        public Builder alertId(String alertId) { this.alertId = alertId; return this; }
        public Builder patientId(String patientId) { this.patientId = patientId; return this; }
        public Builder allergen(String allergen) { this.allergen = allergen; return this; }
        public Builder allergyType(String allergyType) { this.allergyType = allergyType; return this; }
        public Builder severity(String severity) { this.severity = severity; return this; }
        public Builder reaction(String reaction) { this.reaction = reaction; return this; }
        public Builder triggerMedication(String triggerMedication) { this.triggerMedication = triggerMedication; return this; }
        public Builder alertTime(LocalDateTime alertTime) { this.alertTime = alertTime; return this; }
        public Builder requiresImmediateAction(boolean requiresImmediateAction) { this.requiresImmediateAction = requiresImmediateAction; return this; }
        public Builder recommendedAction(String recommendedAction) { this.recommendedAction = recommendedAction; return this; }

        public AllergyAlert build() {
            AllergyAlert alert = new AllergyAlert();
            alert.alertId = this.alertId;
            alert.patientId = this.patientId;
            alert.allergen = this.allergen;
            alert.allergyType = this.allergyType;
            alert.severity = this.severity;
            alert.reaction = this.reaction;
            alert.triggerMedication = this.triggerMedication;
            alert.alertTime = this.alertTime;
            alert.requiresImmediateAction = this.requiresImmediateAction;
            alert.recommendedAction = this.recommendedAction;
            return alert;
        }
    }
}