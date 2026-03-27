package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class SafetyCheckResult implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("allergyAlerts")
    private List<String> allergyAlerts;

    @JsonProperty("contraindicationAlerts")
    private List<String> contraindicationAlerts;

    @JsonProperty("interactionAlerts")
    private List<String> interactionAlerts;

    @JsonProperty("hasCriticalAlert")
    private boolean hasCriticalAlert;

    @JsonProperty("totalAlerts")
    private int totalAlerts;

    @JsonProperty("highestSeverity")
    private String highestSeverity;

    public SafetyCheckResult() {
        this.allergyAlerts = new ArrayList<>();
        this.contraindicationAlerts = new ArrayList<>();
        this.interactionAlerts = new ArrayList<>();
        this.highestSeverity = "LOW";
    }

    public void addAllergyAlert(String alert) {
        allergyAlerts.add(alert);
        totalAlerts++;
        updateSeverity("HIGH");
    }

    public void addContraindicationAlert(String alert, boolean isCritical) {
        contraindicationAlerts.add(alert);
        totalAlerts++;
        if (isCritical) {
            hasCriticalAlert = true;
            updateSeverity("CRITICAL");
        } else {
            updateSeverity("MODERATE");
        }
    }

    public void addInteractionAlert(String alert, String severity) {
        interactionAlerts.add(alert);
        totalAlerts++;
        updateSeverity(severity);
    }

    private void updateSeverity(String newSeverity) {
        int current = severityRank(highestSeverity);
        int incoming = severityRank(newSeverity);
        if (incoming > current) {
            highestSeverity = newSeverity;
        }
    }

    private static int severityRank(String s) {
        if (s == null) return 0;
        switch (s) {
            case "CRITICAL": return 4;
            case "HIGH": return 3;
            case "MODERATE": return 2;
            case "LOW": return 1;
            default: return 0;
        }
    }

    // Getters
    public List<String> getAllergyAlerts() { return allergyAlerts; }
    public List<String> getContraindicationAlerts() { return contraindicationAlerts; }
    public List<String> getInteractionAlerts() { return interactionAlerts; }
    public boolean isHasCriticalAlert() { return hasCriticalAlert; }
    public int getTotalAlerts() { return totalAlerts; }
    public String getHighestSeverity() { return highestSeverity; }

    @Override
    public String toString() {
        return String.format("SafetyCheck{alerts=%d, severity=%s, critical=%s}",
                totalAlerts, highestSeverity, hasCriticalAlert);
    }
}
