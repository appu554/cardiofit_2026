package com.cardiofit.flink.cdc;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;

/**
 * CDC Event Model for KB7 Terminology Releases
 *
 * Parses Debezium CDC events from kb7.terminology.releases topic.
 * This is the "outbox" table for terminology version notifications.
 *
 * Architecture Flow:
 * 1. KB-7 Knowledge Factory Pipeline → GraphDB load complete
 * 2. Pipeline commits to kb_releases table (Commit-Last Strategy)
 * 3. Debezium captures INSERT/UPDATE → publishes to Kafka
 * 4. Flink BroadcastStream receives event → hot-swaps terminology cache
 *
 * Database Schema (kb_terminology.kb_releases):
 * - version_id: Unique version identifier (e.g., "20251203")
 * - snomed_version, rxnorm_version, loinc_version: Source terminology versions
 * - triple_count: Total RDF triples in GraphDB
 * - status: PENDING | LOADING | ACTIVE | ARCHIVED | FAILED
 * - gcs_uri: Cloud storage location of kernel file
 * - graphdb_endpoint: SPARQL endpoint URL
 *
 * @author KB-7 CDC Integration Team
 * @version 1.0
 * @since 2025-12-03
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class TerminologyReleaseCDCEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("payload")
    private Payload payload;

    public TerminologyReleaseCDCEvent() {}

    public Payload getPayload() {
        return payload;
    }

    public void setPayload(Payload payload) {
        this.payload = payload;
    }

    /**
     * Debezium payload containing operation type and release data
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Payload implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("op")
        private String operation; // c=create, u=update, d=delete, r=read (snapshot)

        @JsonProperty("before")
        private ReleaseData before; // State before change (null for INSERT)

        @JsonProperty("after")
        private ReleaseData after; // State after change (null for DELETE)

        @JsonProperty("source")
        private ProtocolCDCEvent.Source source;

        @JsonProperty("ts_ms")
        private Long timestampMs;

        public Payload() {}

        public String getOperation() { return operation; }
        public void setOperation(String operation) { this.operation = operation; }
        public ReleaseData getBefore() { return before; }
        public void setBefore(ReleaseData before) { this.before = before; }
        public ReleaseData getAfter() { return after; }
        public void setAfter(ReleaseData after) { this.after = after; }
        public ProtocolCDCEvent.Source getSource() { return source; }
        public void setSource(ProtocolCDCEvent.Source source) { this.source = source; }
        public Long getTimestampMs() { return timestampMs; }
        public void setTimestampMs(Long timestampMs) { this.timestampMs = timestampMs; }

        public boolean isCreate() { return "c".equals(operation) || "r".equals(operation); }
        public boolean isUpdate() { return "u".equals(operation); }
        public boolean isDelete() { return "d".equals(operation); }

        /**
         * Check if this release is now ACTIVE (should trigger cache refresh)
         */
        public boolean isActiveRelease() {
            return after != null && "ACTIVE".equals(after.getStatus());
        }
    }

    /**
     * Release data from kb_terminology.kb_releases table
     *
     * Maps to schema defined in:
     * backend/shared-infrastructure/kafka/cdc-connectors/sql/kb7-releases-schema.sql
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class ReleaseData implements Serializable {
        private static final long serialVersionUID = 1L;

        // Primary key
        @JsonProperty("id")
        private Integer id;

        // Version identification
        @JsonProperty("version_id")
        private String versionId;

        // Timestamps
        @JsonProperty("release_date")
        private Long releaseDate; // PostgreSQL timestamp as microseconds

        @JsonProperty("graphdb_load_started_at")
        private Long graphdbLoadStartedAt;

        @JsonProperty("graphdb_load_completed_at")
        private Long graphdbLoadCompletedAt;

        // Source terminology versions
        @JsonProperty("snomed_version")
        private String snomedVersion;

        @JsonProperty("rxnorm_version")
        private String rxnormVersion;

        @JsonProperty("loinc_version")
        private String loincVersion;

        // Content metrics
        @JsonProperty("triple_count")
        private Long tripleCount;

        @JsonProperty("concept_count")
        private Integer conceptCount;

        @JsonProperty("snomed_concept_count")
        private Integer snomedConceptCount;

        @JsonProperty("rxnorm_concept_count")
        private Integer rxnormConceptCount;

        @JsonProperty("loinc_concept_count")
        private Integer loincConceptCount;

        // File information
        @JsonProperty("kernel_file_size_bytes")
        private Long kernelFileSizeBytes;

        @JsonProperty("kernel_checksum")
        private String kernelChecksum;

        @JsonProperty("gcs_uri")
        private String gcsUri;

        // GraphDB information
        @JsonProperty("graphdb_repository")
        private String graphdbRepository;

        @JsonProperty("graphdb_endpoint")
        private String graphdbEndpoint;

        // Status tracking
        @JsonProperty("status")
        private String status; // PENDING, LOADING, ACTIVE, ARCHIVED, FAILED

        @JsonProperty("error_message")
        private String errorMessage;

        // Metadata
        @JsonProperty("created_by")
        private String createdBy;

        @JsonProperty("notes")
        private String notes;

        // Debezium metadata field (for unwrapped messages)
        @JsonProperty("__deleted")
        private String deleted;

        public ReleaseData() {}

        // Getters and setters
        public Integer getId() { return id; }
        public void setId(Integer id) { this.id = id; }

        public String getVersionId() { return versionId; }
        public void setVersionId(String versionId) { this.versionId = versionId; }

        public Long getReleaseDate() { return releaseDate; }
        public void setReleaseDate(Long releaseDate) { this.releaseDate = releaseDate; }

        public Long getGraphdbLoadStartedAt() { return graphdbLoadStartedAt; }
        public void setGraphdbLoadStartedAt(Long graphdbLoadStartedAt) { this.graphdbLoadStartedAt = graphdbLoadStartedAt; }

        public Long getGraphdbLoadCompletedAt() { return graphdbLoadCompletedAt; }
        public void setGraphdbLoadCompletedAt(Long graphdbLoadCompletedAt) { this.graphdbLoadCompletedAt = graphdbLoadCompletedAt; }

        public String getSnomedVersion() { return snomedVersion; }
        public void setSnomedVersion(String snomedVersion) { this.snomedVersion = snomedVersion; }

        public String getRxnormVersion() { return rxnormVersion; }
        public void setRxnormVersion(String rxnormVersion) { this.rxnormVersion = rxnormVersion; }

        public String getLoincVersion() { return loincVersion; }
        public void setLoincVersion(String loincVersion) { this.loincVersion = loincVersion; }

        public Long getTripleCount() { return tripleCount; }
        public void setTripleCount(Long tripleCount) { this.tripleCount = tripleCount; }

        public Integer getConceptCount() { return conceptCount; }
        public void setConceptCount(Integer conceptCount) { this.conceptCount = conceptCount; }

        public Integer getSnomedConceptCount() { return snomedConceptCount; }
        public void setSnomedConceptCount(Integer snomedConceptCount) { this.snomedConceptCount = snomedConceptCount; }

        public Integer getRxnormConceptCount() { return rxnormConceptCount; }
        public void setRxnormConceptCount(Integer rxnormConceptCount) { this.rxnormConceptCount = rxnormConceptCount; }

        public Integer getLoincConceptCount() { return loincConceptCount; }
        public void setLoincConceptCount(Integer loincConceptCount) { this.loincConceptCount = loincConceptCount; }

        public Long getKernelFileSizeBytes() { return kernelFileSizeBytes; }
        public void setKernelFileSizeBytes(Long kernelFileSizeBytes) { this.kernelFileSizeBytes = kernelFileSizeBytes; }

        public String getKernelChecksum() { return kernelChecksum; }
        public void setKernelChecksum(String kernelChecksum) { this.kernelChecksum = kernelChecksum; }

        public String getGcsUri() { return gcsUri; }
        public void setGcsUri(String gcsUri) { this.gcsUri = gcsUri; }

        public String getGraphdbRepository() { return graphdbRepository; }
        public void setGraphdbRepository(String graphdbRepository) { this.graphdbRepository = graphdbRepository; }

        public String getGraphdbEndpoint() { return graphdbEndpoint; }
        public void setGraphdbEndpoint(String graphdbEndpoint) { this.graphdbEndpoint = graphdbEndpoint; }

        public String getStatus() { return status; }
        public void setStatus(String status) { this.status = status; }

        public String getErrorMessage() { return errorMessage; }
        public void setErrorMessage(String errorMessage) { this.errorMessage = errorMessage; }

        public String getCreatedBy() { return createdBy; }
        public void setCreatedBy(String createdBy) { this.createdBy = createdBy; }

        public String getNotes() { return notes; }
        public void setNotes(String notes) { this.notes = notes; }

        public String getDeleted() { return deleted; }
        public void setDeleted(String deleted) { this.deleted = deleted; }

        /**
         * Check if this is a valid active release ready for consumption
         */
        public boolean isValid() {
            return versionId != null &&
                   status != null &&
                   !"FAILED".equals(status) &&
                   !"true".equals(deleted);
        }

        @Override
        public String toString() {
            return "ReleaseData{" +
                    "versionId='" + versionId + '\'' +
                    ", status='" + status + '\'' +
                    ", snomedVersion='" + snomedVersion + '\'' +
                    ", rxnormVersion='" + rxnormVersion + '\'' +
                    ", loincVersion='" + loincVersion + '\'' +
                    ", tripleCount=" + tripleCount +
                    ", graphdbEndpoint='" + graphdbEndpoint + '\'' +
                    '}';
        }
    }

    @Override
    public String toString() {
        if (payload == null) return "TerminologyReleaseCDCEvent{payload=null}";
        return "TerminologyReleaseCDCEvent{" +
                "op=" + payload.getOperation() +
                ", versionId=" + (payload.getAfter() != null ? payload.getAfter().getVersionId() :
                                  payload.getBefore() != null ? payload.getBefore().getVersionId() : "null") +
                ", status=" + (payload.getAfter() != null ? payload.getAfter().getStatus() : "null") +
                '}';
    }
}
