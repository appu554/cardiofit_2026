package com.cardiofit.stream.models;

/**
 * Default implementation of CanonicalEvent interface
 * Used by the builder pattern for creating canonical events
 */
public class CanonicalEventImpl implements CanonicalEvent {
    private static final long serialVersionUID = 1L;

    private String eventId;
    private String patientId;
    private String encounterId;
    private String eventType;
    private long timestamp;
    private Object payload;

    public CanonicalEventImpl() {}

    public CanonicalEventImpl(String eventId, String patientId, String eventType, long timestamp) {
        this.eventId = eventId;
        this.patientId = patientId;
        this.eventType = eventType;
        this.timestamp = timestamp;
    }

    @Override
    public String getEventId() { return eventId; }
    public void setEventId(String eventId) { this.eventId = eventId; }

    @Override
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    @Override
    public String getEncounterId() { return encounterId; }
    public void setEncounterId(String encounterId) { this.encounterId = encounterId; }

    @Override
    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }

    @Override
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

    @Override
    public Object getPayload() { return payload; }
    public void setPayload(Object payload) { this.payload = payload; }

    @Override
    public String toString() {
        return "CanonicalEventImpl{" +
                "eventId='" + eventId + '\'' +
                ", patientId='" + patientId + '\'' +
                ", eventType='" + eventType + '\'' +
                ", timestamp=" + timestamp +
                '}';
    }

    /**
     * Builder implementation for CanonicalEvent
     */
    public static class Builder implements CanonicalEvent.Builder {
        private String eventId;
        private String patientId;
        private String encounterId;
        private String eventType;
        private long timestamp;
        private Object payload;

        @Override
        public Builder id(String id) {
            this.eventId = id;
            return this;
        }

        @Override
        public Builder eventId(String eventId) {
            this.eventId = eventId;
            return this;
        }

        @Override
        public Builder patientId(String patientId) {
            this.patientId = patientId;
            return this;
        }

        public Builder encounterId(String encounterId) {
            this.encounterId = encounterId;
            return this;
        }

        @Override
        public Builder eventType(String eventType) {
            this.eventType = eventType;
            return this;
        }

        @Override
        public Builder timestamp(long timestamp) {
            this.timestamp = timestamp;
            return this;
        }

        @Override
        public Builder payload(Object payload) {
            this.payload = payload;
            return this;
        }

        @Override
        public CanonicalEvent build() {
            CanonicalEventImpl event = new CanonicalEventImpl();
            event.eventId = this.eventId;
            event.patientId = this.patientId;
            event.encounterId = this.encounterId;
            event.eventType = this.eventType;
            event.timestamp = this.timestamp;
            event.payload = this.payload;
            return event;
        }
    }
}