package com.cardiofit.flink.cdc;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;

/**
 * CDC Event Model for KB5 Drug Interactions
 *
 * Parses Debezium CDC events from kb5.drug_interactions.changes topic.
 *
 * @author Phase 2 CDC Integration Team
 * @version 1.0
 * @since 2025-11-22
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class DrugInteractionCDCEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("payload")
    private Payload payload;

    public DrugInteractionCDCEvent() {}

    public Payload getPayload() {
        return payload;
    }

    public void setPayload(Payload payload) {
        this.payload = payload;
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Payload implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("op")
        private String operation;

        @JsonProperty("before")
        private InteractionData before;

        @JsonProperty("after")
        private InteractionData after;

        @JsonProperty("source")
        private ProtocolCDCEvent.Source source;

        @JsonProperty("ts_ms")
        private Long timestampMs;

        public Payload() {}

        public String getOperation() { return operation; }
        public void setOperation(String operation) { this.operation = operation; }
        public InteractionData getBefore() { return before; }
        public void setBefore(InteractionData before) { this.before = before; }
        public InteractionData getAfter() { return after; }
        public void setAfter(InteractionData after) { this.after = after; }
        public ProtocolCDCEvent.Source getSource() { return source; }
        public void setSource(ProtocolCDCEvent.Source source) { this.source = source; }
        public Long getTimestampMs() { return timestampMs; }
        public void setTimestampMs(Long timestampMs) { this.timestampMs = timestampMs; }

        public boolean isCreate() { return "c".equals(operation) || "r".equals(operation); }
        public boolean isUpdate() { return "u".equals(operation); }
        public boolean isDelete() { return "d".equals(operation); }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class InteractionData implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("interaction_id")
        private String interactionId;

        @JsonProperty("drug_a")
        private String drugA;

        @JsonProperty("drug_b")
        private String drugB;

        @JsonProperty("severity")
        private String severity; // CRITICAL, HIGH, MODERATE, LOW

        @JsonProperty("mechanism")
        private String mechanism;

        @JsonProperty("clinical_effect")
        private String clinicalEffect;

        @JsonProperty("management_recommendation")
        private String managementRecommendation;

        @JsonProperty("evidence_level")
        private String evidenceLevel;

        @JsonProperty("references")
        private Object references; // JSONB array

        @JsonProperty("created_at")
        private String createdAt;

        @JsonProperty("updated_at")
        private String updatedAt;

        public InteractionData() {}

        public String getInteractionId() { return interactionId; }
        public void setInteractionId(String interactionId) { this.interactionId = interactionId; }
        public String getDrugA() { return drugA; }
        public void setDrugA(String drugA) { this.drugA = drugA; }
        public String getDrugB() { return drugB; }
        public void setDrugB(String drugB) { this.drugB = drugB; }
        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }
        public String getMechanism() { return mechanism; }
        public void setMechanism(String mechanism) { this.mechanism = mechanism; }
        public String getClinicalEffect() { return clinicalEffect; }
        public void setClinicalEffect(String clinicalEffect) { this.clinicalEffect = clinicalEffect; }
        public String getManagementRecommendation() { return managementRecommendation; }
        public void setManagementRecommendation(String managementRecommendation) { this.managementRecommendation = managementRecommendation; }
        public String getEvidenceLevel() { return evidenceLevel; }
        public void setEvidenceLevel(String evidenceLevel) { this.evidenceLevel = evidenceLevel; }
        public Object getReferences() { return references; }
        public void setReferences(Object references) { this.references = references; }
        public String getCreatedAt() { return createdAt; }
        public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
        public String getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(String updatedAt) { this.updatedAt = updatedAt; }

        @Override
        public String toString() {
            return "InteractionData{" +
                    "interactionId='" + interactionId + '\'' +
                    ", drugA='" + drugA + '\'' +
                    ", drugB='" + drugB + '\'' +
                    ", severity='" + severity + '\'' +
                    '}';
        }
    }

    @Override
    public String toString() {
        if (payload == null) return "DrugInteractionCDCEvent{payload=null}";
        return "DrugInteractionCDCEvent{" +
                "op=" + payload.getOperation() +
                ", interactionId=" + (payload.getAfter() != null ? payload.getAfter().getInteractionId() :
                                      payload.getBefore() != null ? payload.getBefore().getInteractionId() : "null") +
                '}';
    }
}
