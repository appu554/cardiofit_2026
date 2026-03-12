package com.cardiofit.flink.cdc;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;

/**
 * CDC Event Model for KB7 Terminology & Concepts
 *
 * Parses Debezium CDC events from:
 * - kb7.terminology.changes
 * - kb7.terminology_concepts.changes
 *
 * Manages clinical terminology standards (SNOMED CT, LOINC, RxNorm)
 * and concept mappings for semantic interoperability.
 *
 * @author Phase 2 CDC Integration Team
 * @version 1.0
 * @since 2025-11-22
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class TerminologyCDCEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("payload")
    private Payload payload;

    public TerminologyCDCEvent() {}

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
        private TerminologyData before;

        @JsonProperty("after")
        private TerminologyData after;

        @JsonProperty("source")
        private ProtocolCDCEvent.Source source;

        @JsonProperty("ts_ms")
        private Long timestampMs;

        public Payload() {}

        public String getOperation() { return operation; }
        public void setOperation(String operation) { this.operation = operation; }
        public TerminologyData getBefore() { return before; }
        public void setBefore(TerminologyData before) { this.before = before; }
        public TerminologyData getAfter() { return after; }
        public void setAfter(TerminologyData after) { this.after = after; }
        public ProtocolCDCEvent.Source getSource() { return source; }
        public void setSource(ProtocolCDCEvent.Source source) { this.source = source; }
        public Long getTimestampMs() { return timestampMs; }
        public void setTimestampMs(Long timestampMs) { this.timestampMs = timestampMs; }

        public boolean isCreate() { return "c".equals(operation) || "r".equals(operation); }
        public boolean isUpdate() { return "u".equals(operation); }
        public boolean isDelete() { return "d".equals(operation); }
    }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class TerminologyData implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("concept_id")
        private String conceptId;

        @JsonProperty("concept_code")
        private String conceptCode;

        @JsonProperty("display_name")
        private String displayName;

        @JsonProperty("code_system")
        private String codeSystem; // SNOMED_CT, LOINC, RXNORM, ICD10, CPT

        @JsonProperty("code_system_version")
        private String codeSystemVersion;

        @JsonProperty("definition")
        private String definition;

        @JsonProperty("synonyms")
        private Object synonyms; // JSONB array

        @JsonProperty("mappings")
        private Object mappings; // JSONB cross-terminology mappings

        @JsonProperty("parent_concepts")
        private Object parentConcepts; // JSONB hierarchy

        @JsonProperty("child_concepts")
        private Object childConcepts; // JSONB hierarchy

        @JsonProperty("attributes")
        private Object attributes; // JSONB concept attributes

        @JsonProperty("status")
        private String status; // ACTIVE, DEPRECATED, RETIRED

        @JsonProperty("created_at")
        private String createdAt;

        @JsonProperty("updated_at")
        private String updatedAt;

        public TerminologyData() {}

        public String getConceptId() { return conceptId; }
        public void setConceptId(String conceptId) { this.conceptId = conceptId; }
        public String getConceptCode() { return conceptCode; }
        public void setConceptCode(String conceptCode) { this.conceptCode = conceptCode; }
        public String getDisplayName() { return displayName; }
        public void setDisplayName(String displayName) { this.displayName = displayName; }
        public String getCodeSystem() { return codeSystem; }
        public void setCodeSystem(String codeSystem) { this.codeSystem = codeSystem; }
        public String getCodeSystemVersion() { return codeSystemVersion; }
        public void setCodeSystemVersion(String codeSystemVersion) { this.codeSystemVersion = codeSystemVersion; }
        public String getDefinition() { return definition; }
        public void setDefinition(String definition) { this.definition = definition; }
        public Object getSynonyms() { return synonyms; }
        public void setSynonyms(Object synonyms) { this.synonyms = synonyms; }
        public Object getMappings() { return mappings; }
        public void setMappings(Object mappings) { this.mappings = mappings; }
        public Object getParentConcepts() { return parentConcepts; }
        public void setParentConcepts(Object parentConcepts) { this.parentConcepts = parentConcepts; }
        public Object getChildConcepts() { return childConcepts; }
        public void setChildConcepts(Object childConcepts) { this.childConcepts = childConcepts; }
        public Object getAttributes() { return attributes; }
        public void setAttributes(Object attributes) { this.attributes = attributes; }
        public String getStatus() { return status; }
        public void setStatus(String status) { this.status = status; }
        public String getCreatedAt() { return createdAt; }
        public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
        public String getUpdatedAt() { return updatedAt; }
        public void setUpdatedAt(String updatedAt) { this.updatedAt = updatedAt; }

        @Override
        public String toString() {
            return "TerminologyData{" +
                    "conceptId='" + conceptId + '\'' +
                    ", conceptCode='" + conceptCode + '\'' +
                    ", displayName='" + displayName + '\'' +
                    ", codeSystem='" + codeSystem + '\'' +
                    '}';
        }
    }

    @Override
    public String toString() {
        if (payload == null) return "TerminologyCDCEvent{payload=null}";
        return "TerminologyCDCEvent{" +
                "op=" + payload.getOperation() +
                ", conceptId=" + (payload.getAfter() != null ? payload.getAfter().getConceptId() :
                                  payload.getBefore() != null ? payload.getBefore().getConceptId() : "null") +
                '}';
    }
}
