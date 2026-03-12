package com.cardiofit.stream.models;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;

public class PathwayEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    private String eventId;
    private String patientId;
    private String pathwayId;
    private String eventType;
    private String phase;
    private LocalDateTime timestamp;
    private Map<String, Object> data;

    public PathwayEvent() {}

    public PathwayEvent(String eventId, String patientId, String pathwayId) {
        this.eventId = eventId;
        this.patientId = patientId;
        this.pathwayId = pathwayId;
        this.timestamp = LocalDateTime.now();
    }

    // Constructor required by ClinicalPathwayAdherenceFunction
    public PathwayEvent(PatientEvent patientEvent, java.util.List<com.cardiofit.stream.state.Protocol> protocols) {
        this.eventId = patientEvent.getEventId();
        this.patientId = ((CanonicalEvent)patientEvent).getPatientId();
        this.eventType = patientEvent.getClass().getSimpleName();
        this.timestamp = LocalDateTime.now();

        // Set pathway ID from first protocol if available
        if (protocols != null && !protocols.isEmpty()) {
            this.pathwayId = protocols.get(0).getId();
        } else {
            this.pathwayId = "default_pathway";
        }
    }

    // Getters and Setters
    public String getEventId() { return eventId; }
    public void setEventId(String eventId) { this.eventId = eventId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getPathwayId() { return pathwayId; }
    public void setPathwayId(String pathwayId) { this.pathwayId = pathwayId; }

    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }

    public String getPhase() { return phase; }
    public void setPhase(String phase) { this.phase = phase; }

    public LocalDateTime getTimestamp() { return timestamp; }
    public void setTimestamp(LocalDateTime timestamp) { this.timestamp = timestamp; }

    public Map<String, Object> getData() { return data; }
    public void setData(Map<String, Object> data) { this.data = data; }
}