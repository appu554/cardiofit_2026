package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Objects;

/**
 * Generic wrapper for all clinical event types in the unified pipeline.
 *
 * This class provides a common envelope for vitals, labs, and medications,
 * enabling union of multiple Kafka streams before processing through the
 * unified PatientContextAggregator operator.
 *
 * Architecture Pattern: Unified State Management
 * - All event types flow through single operator
 * - Guarantees state consistency (no race conditions)
 * - Switch-based processing on eventType discriminator
 *
 * @see com.cardiofit.flink.operators.PatientContextAggregator
 */
public class GenericEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Event type discriminator for switch-based processing
     * Valid values: "VITAL_SIGN", "LAB_RESULT", "MEDICATION_UPDATE"
     */
    @JsonProperty("eventType")
    private String eventType;

    /**
     * Patient identifier for keying streams
     * All events for same patient processed by same operator instance
     */
    @JsonProperty("patientId")
    private String patientId;

    /**
     * Event timestamp for watermark assignment and temporal ordering
     */
    @JsonProperty("eventTime")
    private long eventTime;

    /**
     * Type-erased payload - actual type determined by eventType
     * - VITAL_SIGN → VitalsPayload
     * - LAB_RESULT → LabPayload
     * - MEDICATION_UPDATE → MedicationPayload
     */
    @JsonProperty("payload")
    private Object payload;

    /**
     * Source system identifier (optional)
     * Example: "philips_monitor", "epic_labs", "meditech_pharmacy"
     */
    @JsonProperty("source")
    private String source;

    /**
     * Default constructor for deserialization
     */
    public GenericEvent() {
        this.eventTime = System.currentTimeMillis();
    }

    /**
     * Constructor with required fields
     */
    public GenericEvent(String eventType, String patientId, Object payload) {
        this.eventType = eventType;
        this.patientId = patientId;
        this.payload = payload;
        this.eventTime = System.currentTimeMillis();
    }

    /**
     * Full constructor with all fields
     */
    public GenericEvent(String eventType, String patientId, long eventTime, Object payload, String source) {
        this.eventType = eventType;
        this.patientId = patientId;
        this.eventTime = eventTime;
        this.payload = payload;
        this.source = source;
    }

    // Getters and setters

    public String getEventType() {
        return eventType;
    }

    public void setEventType(String eventType) {
        this.eventType = eventType;
    }

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public long getEventTime() {
        return eventTime;
    }

    public void setEventTime(long eventTime) {
        this.eventTime = eventTime;
    }

    public Object getPayload() {
        return payload;
    }

    public void setPayload(Object payload) {
        this.payload = payload;
    }

    public String getSource() {
        return source;
    }

    public void setSource(String source) {
        this.source = source;
    }

    /**
     * Type-safe payload extraction with validation
     * @param expectedType The expected payload class
     * @return Typed payload or null if type mismatch
     */
    @SuppressWarnings("unchecked")
    public <T> T getPayloadAs(Class<T> expectedType) {
        if (payload == null) {
            return null;
        }

        if (expectedType.isInstance(payload)) {
            return (T) payload;
        }

        return null;
    }

    /**
     * Check if this is a vital sign event
     */
    public boolean isVitalSign() {
        return "VITAL_SIGN".equals(eventType);
    }

    /**
     * Check if this is a lab result event
     */
    public boolean isLabResult() {
        return "LAB_RESULT".equals(eventType);
    }

    /**
     * Check if this is a medication update event
     */
    public boolean isMedicationUpdate() {
        return "MEDICATION_UPDATE".equals(eventType);
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        GenericEvent that = (GenericEvent) o;
        return eventTime == that.eventTime &&
                Objects.equals(eventType, that.eventType) &&
                Objects.equals(patientId, that.patientId) &&
                Objects.equals(payload, that.payload) &&
                Objects.equals(source, that.source);
    }

    @Override
    public int hashCode() {
        return Objects.hash(eventType, patientId, eventTime, payload, source);
    }

    @Override
    public String toString() {
        return "GenericEvent{" +
                "eventType='" + eventType + '\'' +
                ", patientId='" + patientId + '\'' +
                ", eventTime=" + eventTime +
                ", source='" + source + '\'' +
                ", payload=" + (payload != null ? payload.getClass().getSimpleName() : "null") +
                '}';
    }
}
