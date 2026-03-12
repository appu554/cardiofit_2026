package com.cardiofit.stream.models;

import java.io.Serializable;

/**
 * Base interface for all clinical events in the processing pipeline
 * Provides common methods for event identification and patient association
 */
public interface CanonicalEvent extends Serializable {

    /**
     * Get the unique event identifier
     * @return event ID
     */
    String getEventId();

    /**
     * Get the unique ID (alias for getEventId for compatibility)
     * @return event ID
     */
    default String getId() {
        return getEventId();
    }

    /**
     * Get the patient ID associated with this event
     * @return patient ID
     */
    String getPatientId();

    /**
     * Get the encounter ID if applicable
     * @return encounter ID or null
     */
    String getEncounterId();

    /**
     * Get the event type
     * @return event type
     */
    String getEventType();

    /**
     * Get the event timestamp
     * @return timestamp
     */
    long getTimestamp();

    /**
     * Get the event payload
     * @return event payload
     */
    default Object getPayload() {
        return null;
    }

    /**
     * Builder pattern support
     */
    static Builder builder() {
        return new CanonicalEventImpl.Builder();
    }

    /**
     * Ingestion metadata for tracking event processing
     */
    class IngestionMetadata implements Serializable {
        private static final long serialVersionUID = 1L;

        private String source;
        private long ingestionTime;
        private String partitionKey;
        private int subtaskIndex;

        public IngestionMetadata(String source, long ingestionTime) {
            this.source = source;
            this.ingestionTime = ingestionTime;
        }

        public IngestionMetadata(String source, long ingestionTime, int subtaskIndex) {
            this.source = source;
            this.ingestionTime = ingestionTime;
            this.subtaskIndex = subtaskIndex;
        }

        // Getters and setters
        public String getSource() { return source; }
        public void setSource(String source) { this.source = source; }

        public long getIngestionTime() { return ingestionTime; }
        public void setIngestionTime(long ingestionTime) { this.ingestionTime = ingestionTime; }

        public String getPartitionKey() { return partitionKey; }
        public void setPartitionKey(String partitionKey) { this.partitionKey = partitionKey; }

        public int getSubtaskIndex() { return subtaskIndex; }
        public void setSubtaskIndex(int subtaskIndex) { this.subtaskIndex = subtaskIndex; }
    }

    /**
     * Builder interface for CanonicalEvent
     */
    interface Builder {
        Builder id(String id);
        Builder eventId(String eventId);
        Builder patientId(String patientId);
        Builder eventType(String eventType);
        Builder timestamp(long timestamp);
        Builder payload(Object payload);
        CanonicalEvent build();
    }
}