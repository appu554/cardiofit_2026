package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;
import java.util.Objects;

/**
 * Clinical condition or diagnosis.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class Condition implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("code")
    private String code; // ICD-10 or SNOMED code

    @JsonProperty("display")
    private String display; // Human-readable name

    @JsonProperty("status")
    private String status; // "active", "resolved", "inactive"

    @JsonProperty("severity")
    private String severity; // "mild", "moderate", "severe"

    @JsonProperty("onsetDate")
    private Long onsetDate; // When condition started

    public Condition() {
    }

    public Condition(String code, String display) {
        this.code = code;
        this.display = display;
        this.status = "active";
    }

    public static Condition fromPayload(Map<String, Object> payload) {
        Condition condition = new Condition();

        condition.code = (String) payload.get("condition_code");
        condition.display = (String) payload.get("condition_name");
        condition.status = (String) payload.getOrDefault("status", "active");
        condition.severity = (String) payload.get("severity");

        Object onsetObj = payload.get("onset_date");
        if (onsetObj instanceof Number) {
            condition.onsetDate = ((Number) onsetObj).longValue();
        }

        return condition;
    }

    // Getters and setters
    public String getCode() { return code; }
    public void setCode(String code) { this.code = code; }

    public String getDisplay() { return display; }
    public void setDisplay(String display) { this.display = display; }

    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public Long getOnsetDate() { return onsetDate; }
    public void setOnsetDate(Long onsetDate) { this.onsetDate = onsetDate; }

    // ============================================================
    // ALIAS METHODS (for Module 2 compatibility)
    // ============================================================

    /**
     * Get condition name (alias for getDisplay()).
     * Used by Module 2 for condition-based risk analysis.
     */
    public String getConditionName() {
        return display;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        Condition condition = (Condition) o;
        return Objects.equals(code, condition.code);
    }

    @Override
    public int hashCode() {
        return Objects.hash(code);
    }

    @Override
    public String toString() {
        return "Condition{" +
                "code='" + code + '\'' +
                ", display='" + display + '\'' +
                ", status='" + status + '\'' +
                '}';
    }
}
