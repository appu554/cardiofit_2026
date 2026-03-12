package com.cardiofit.flink.cdc;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;

/**
 * CDC Event Model for KB2 Clinical Phenotypes
 *
 * Parses Debezium CDC events from kb2.clinical_phenotypes.changes topic.
 *
 * Clinical phenotypes define patient archetypes with risk factors and
 * context patterns used by Module 2 for enhanced context assembly.
 *
 * @author Phase 2 CDC Integration Team
 * @version 1.0
 * @since 2025-11-22
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class ClinicalPhenotypeCDCEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("payload")
    private Payload payload;

    public ClinicalPhenotypeCDCEvent() {}

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
        private PhenotypeData before;

        @JsonProperty("after")
        private PhenotypeData after;

        @JsonProperty("source")
        private ProtocolCDCEvent.Source source;

        @JsonProperty("ts_ms")
        private Long timestampMs;

        public Payload() {}

        public String getOperation() { return operation; }
        public void setOperation(String operation) { this.operation = operation; }
        public PhenotypeData getBefore() { return before; }
        public void setBefore(PhenotypeData before) { this.before = before; }
        public PhenotypeData getAfter() { return after; }
        public void setAfter(PhenotypeData after) { this.after = after; }
        public ProtocolCDCEvent.Source getSource() { return source; }
        public void setSource(ProtocolCDCEvent.Source source) { this.source = source; }
        public Long getTimestampMs() { return timestampMs; }
        public void setTimestampMs(Long timestampMs) { this.timestampMs = timestampMs; }

        public boolean isCreate() { return "c".equals(operation) || "r".equals(operation); }
        public boolean isUpdate() { return "u".equals(operation); }
        public boolean isDelete() { return "d".equals(operation); }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class PhenotypeData implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("phenotype_id")
        private String phenotypeId;

        @JsonProperty("name")
        private String name;

        @JsonProperty("description")
        private String description;

        @JsonProperty("risk_factors")
        private Object riskFactors; // JSONB

        @JsonProperty("context_patterns")
        private Object contextPatterns; // JSONB

        @JsonProperty("priority")
        private String priority;

        @JsonProperty("version")
        private String version;

        @JsonProperty("created_at")
        private String createdAt;

        @JsonProperty("updated_at")
        private String updatedAt;

        public PhenotypeData() {}

        public String getPhenotypeId() { return phenotypeId; }
        public void setPhenotypeId(String phenotypeId) { this.phenotypeId = phenotypeId; }
        public String getName() { return name; }
        public void setName(String name) { this.name = name; }
        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
        public Object getRiskFactors() { return riskFactors; }
        public void setRiskFactors(Object riskFactors) { this.riskFactors = riskFactors; }
        public Object getContextPatterns() { return contextPatterns; }
        public void setContextPatterns(Object contextPatterns) { this.contextPatterns = contextPatterns; }
        public String getPriority() { return priority; }
        public void setPriority(String priority) { this.priority = priority; }
        public String getVersion() { return version; }
        public void setVersion(String version) { this.version = version; }
        public String getCreatedAt() { return createdAt; }
        public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
        public String getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(String updatedAt) { this.updatedAt = updatedAt; }

        @Override
        public String toString() {
            return "PhenotypeData{" +
                    "phenotypeId='" + phenotypeId + '\'' +
                    ", name='" + name + '\'' +
                    ", priority='" + priority + '\'' +
                    '}';
        }
    }

    @Override
    public String toString() {
        if (payload == null) return "ClinicalPhenotypeCDCEvent{payload=null}";
        return "ClinicalPhenotypeCDCEvent{" +
                "op=" + payload.getOperation() +
                ", phenotypeId=" + (payload.getAfter() != null ? payload.getAfter().getPhenotypeId() :
                                    payload.getBefore() != null ? payload.getBefore().getPhenotypeId() : "null") +
                '}';
    }
}
