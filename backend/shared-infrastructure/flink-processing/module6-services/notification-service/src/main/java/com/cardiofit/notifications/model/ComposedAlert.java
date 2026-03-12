package com.cardiofit.notifications.model;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.Instant;
import java.util.List;
import java.util.Map;

/**
 * Composed Alert model from Kafka topic
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ComposedAlert {

    @JsonProperty("alert_id")
    private String alertId;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("patient_name")
    private String patientName;

    @JsonProperty("alert_type")
    private String alertType;

    @JsonProperty("severity")
    private AlertSeverity severity;

    @JsonProperty("title")
    private String title;

    @JsonProperty("message")
    private String message;

    @JsonProperty("timestamp")
    @JsonFormat(shape = JsonFormat.Shape.STRING, pattern = "yyyy-MM-dd'T'HH:mm:ss'Z'", timezone = "UTC")
    private Instant timestamp;

    @JsonProperty("clinical_context")
    private ClinicalContext clinicalContext;

    @JsonProperty("recommended_actions")
    private List<String> recommendedActions;

    @JsonProperty("assigned_to")
    private List<String> assignedTo;

    @JsonProperty("priority_score")
    private Double priorityScore;

    @JsonProperty("metadata")
    private Map<String, Object> metadata;

    public enum AlertSeverity {
        CRITICAL,
        HIGH,
        MEDIUM,
        LOW,
        INFO
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ClinicalContext {
        @JsonProperty("vital_signs")
        private Map<String, Object> vitalSigns;

        @JsonProperty("diagnosis")
        private String diagnosis;

        @JsonProperty("risk_factors")
        private List<String> riskFactors;

        @JsonProperty("active_medications")
        private List<String> activeMedications;

        @JsonProperty("location")
        private String location;
    }
}
