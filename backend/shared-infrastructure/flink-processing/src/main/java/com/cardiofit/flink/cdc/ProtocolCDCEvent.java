package com.cardiofit.flink.cdc;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;
import java.util.Map;

/**
 * CDC Event Model for KB3 Clinical Protocols
 *
 * Parses Debezium CDC events from kb3.clinical_protocols.changes topic.
 *
 * Debezium Envelope Format:
 * {
 *   "payload": {
 *     "op": "c|u|d|r",  // create, update, delete, read
 *     "before": {...},   // State before change (null for INSERT)
 *     "after": {...},    // State after change (null for DELETE)
 *     "source": {
 *       "db": "kb3",
 *       "table": "clinical_protocols",
 *       "ts_ms": 1732233600000
 *     }
 *   }
 * }
 *
 * @author Phase 2 CDC Integration Team
 * @version 1.0
 * @since 2025-11-22
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class ProtocolCDCEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("payload")
    private Payload payload;

    public ProtocolCDCEvent() {}

    public Payload getPayload() {
        return payload;
    }

    public void setPayload(Payload payload) {
        this.payload = payload;
    }

    /**
     * Debezium payload containing operation type and data
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Payload implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("op")
        private String operation; // c=create, u=update, d=delete, r=read (snapshot)

        @JsonProperty("before")
        private ProtocolData before; // State before change (null for INSERT)

        @JsonProperty("after")
        private ProtocolData after; // State after change (null for DELETE)

        @JsonProperty("source")
        private Source source;

        @JsonProperty("ts_ms")
        private Long timestampMs; // Event timestamp

        public Payload() {}

        public String getOperation() {
            return operation;
        }

        public void setOperation(String operation) {
            this.operation = operation;
        }

        public ProtocolData getBefore() {
            return before;
        }

        public void setBefore(ProtocolData before) {
            this.before = before;
        }

        public ProtocolData getAfter() {
            return after;
        }

        public void setAfter(ProtocolData after) {
            this.after = after;
        }

        public Source getSource() {
            return source;
        }

        public void setSource(Source source) {
            this.source = source;
        }

        public Long getTimestampMs() {
            return timestampMs;
        }

        public void setTimestampMs(Long timestampMs) {
            this.timestampMs = timestampMs;
        }

        /**
         * Check if this is a CREATE operation
         */
        public boolean isCreate() {
            return "c".equals(operation) || "r".equals(operation);
        }

        /**
         * Check if this is an UPDATE operation
         */
        public boolean isUpdate() {
            return "u".equals(operation);
        }

        /**
         * Check if this is a DELETE operation
         */
        public boolean isDelete() {
            return "d".equals(operation);
        }
    }

    /**
     * Protocol data from PostgreSQL kb3_guidelines.clinical_protocols table
     *
     * Actual database schema:
     * - id: integer (primary key)
     * - protocol_name: varchar(255)
     * - specialty: varchar(100)
     * - version: varchar(50)
     * - content: text
     * - created_at: timestamp
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class ProtocolData implements Serializable {
        private static final long serialVersionUID = 1L;

        // Actual database fields (kb3_guidelines.clinical_protocols)
        @JsonProperty("id")
        private Integer id;

        @JsonProperty("protocol_name")
        private String protocolName;

        @JsonProperty("specialty")
        private String specialty;

        @JsonProperty("version")
        private String version;

        @JsonProperty("content")
        private String content;

        @JsonProperty("created_at")
        private Long createdAt; // PostgreSQL timestamp as microseconds

        // Legacy fields (for backward compatibility if other schemas exist)
        @JsonProperty("protocol_id")
        private String protocolId;

        @JsonProperty("name")
        private String name;

        @JsonProperty("category")
        private String category;

        public ProtocolData() {}

        // Getters and setters for actual database fields

        public Integer getId() {
            return id;
        }

        public void setId(Integer id) {
            this.id = id;
        }

        public String getProtocolName() {
            return protocolName;
        }

        public void setProtocolName(String protocolName) {
            this.protocolName = protocolName;
        }

        public String getSpecialty() {
            return specialty;
        }

        public void setSpecialty(String specialty) {
            this.specialty = specialty;
        }

        public String getVersion() {
            return version;
        }

        public void setVersion(String version) {
            this.version = version;
        }

        public String getContent() {
            return content;
        }

        public void setContent(String content) {
            this.content = content;
        }

        public Long getCreatedAt() {
            return createdAt;
        }

        public void setCreatedAt(Long createdAt) {
            this.createdAt = createdAt;
        }

        // Legacy getters/setters for backward compatibility

        public String getProtocolId() {
            // Fallback: if protocolId is null, return id as string
            return protocolId != null ? protocolId : (id != null ? String.valueOf(id) : null);
        }

        public void setProtocolId(String protocolId) {
            this.protocolId = protocolId;
        }

        public String getName() {
            // Fallback: if name is null, return protocolName
            return name != null ? name : protocolName;
        }

        public void setName(String name) {
            this.name = name;
        }

        public String getCategory() {
            return category;
        }

        public void setCategory(String category) {
            this.category = category;
        }


        @Override
        public String toString() {
            return "ProtocolData{" +
                    "id=" + id +
                    ", protocolName='" + protocolName + '\'' +
                    ", specialty='" + specialty + '\'' +
                    ", version='" + version + '\'' +
                    ", content='" + (content != null ? content.substring(0, Math.min(50, content.length())) + "..." : null) + '\'' +
                    ", createdAt=" + createdAt +
                    '}';
        }
    }

    /**
     * CDC source metadata from Debezium
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Source implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("db")
        private String database;

        @JsonProperty("table")
        private String table;

        @JsonProperty("ts_ms")
        private Long timestampMs;

        @JsonProperty("snapshot")
        private String snapshot; // "true" for initial snapshot, "false" for real-time

        public Source() {}

        public String getDatabase() {
            return database;
        }

        public void setDatabase(String database) {
            this.database = database;
        }

        public String getTable() {
            return table;
        }

        public void setTable(String table) {
            this.table = table;
        }

        public Long getTimestampMs() {
            return timestampMs;
        }

        public void setTimestampMs(Long timestampMs) {
            this.timestampMs = timestampMs;
        }

        public String getSnapshot() {
            return snapshot;
        }

        public void setSnapshot(String snapshot) {
            this.snapshot = snapshot;
        }

        public boolean isSnapshot() {
            return "true".equals(snapshot);
        }
    }

    @Override
    public String toString() {
        if (payload == null) {
            return "ProtocolCDCEvent{payload=null}";
        }
        return "ProtocolCDCEvent{" +
                "op=" + payload.getOperation() +
                ", protocolId=" + (payload.getAfter() != null ? payload.getAfter().getProtocolId() :
                                   payload.getBefore() != null ? payload.getBefore().getProtocolId() : "null") +
                ", timestamp=" + payload.getTimestampMs() +
                '}';
    }
}
