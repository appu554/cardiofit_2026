package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

@JsonInclude(JsonInclude.Include.NON_EMPTY)
public class MedicationSafetyResult implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("medicationCode")
    private String medicationCode;

    @JsonProperty("medicationName")
    private String medicationName;

    @JsonProperty("isSafe")
    private boolean isSafe;

    @JsonProperty("contraindicationType")
    private String contraindicationType;

    @JsonProperty("reason")
    private String reason;

    @JsonProperty("recommendation")
    private String recommendation;

    @JsonProperty("severityScore")
    private Integer severityScore;

    @JsonProperty("interactions")
    private List<String> interactions;

    public MedicationSafetyResult() {
        this.interactions = new ArrayList<>();
        this.isSafe = true;
        this.contraindicationType = "NONE";
    }

    public MedicationSafetyResult(String code, String name) {
        this();
        this.medicationCode = code;
        this.medicationName = name;
    }

    // Getters and setters
    public String getMedicationCode() { return medicationCode; }
    public void setMedicationCode(String v) { this.medicationCode = v; }
    public String getMedicationName() { return medicationName; }
    public void setMedicationName(String v) { this.medicationName = v; }
    public boolean isSafe() { return isSafe; }
    public void setSafe(boolean v) { this.isSafe = v; }
    public String getContraindicationType() { return contraindicationType; }
    public void setContraindicationType(String v) { this.contraindicationType = v; }
    public String getReason() { return reason; }
    public void setReason(String v) { this.reason = v; }
    public String getRecommendation() { return recommendation; }
    public void setRecommendation(String v) { this.recommendation = v; }
    public Integer getSeverityScore() { return severityScore; }
    public void setSeverityScore(Integer v) { this.severityScore = v; }
    public List<String> getInteractions() { return interactions; }
    public void setInteractions(List<String> v) { this.interactions = v; }
}
