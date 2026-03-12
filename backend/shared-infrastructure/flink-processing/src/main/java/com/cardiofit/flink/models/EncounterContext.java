package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Encounter context representing the current hospital encounter.
 *
 * This class tracks the patient's current encounter details including admission,
 * department transfers, and care team information.
 */
public class EncounterContext implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("encounterId")
    private String encounterId;

    @JsonProperty("encounterType")
    private String encounterType; // "inpatient", "outpatient", "emergency"

    @JsonProperty("admissionTime")
    private long admissionTime; // Epoch milliseconds

    @JsonProperty("dischargeTime")
    private Long dischargeTime; // Null if not discharged yet

    @JsonProperty("department")
    private String department; // "ICU", "Cardiology", "Emergency"

    @JsonProperty("room")
    private String room; // Room number

    @JsonProperty("bed")
    private String bed; // Bed identifier

    @JsonProperty("attendingPhysician")
    private String attendingPhysician; // Provider ID

    @JsonProperty("careTeam")
    private List<String> careTeam = new ArrayList<>(); // List of provider IDs

    @JsonProperty("admissionReason")
    private String admissionReason;

    // ============================================================
    // CONSTRUCTORS
    // ============================================================

    public EncounterContext() {
        this.admissionTime = System.currentTimeMillis();
    }

    public EncounterContext(String encounterId, String encounterType, String department) {
        this();
        this.encounterId = encounterId;
        this.encounterType = encounterType;
        this.department = department;
    }

    // ============================================================
    // FACTORY METHOD
    // ============================================================

    /**
     * Create encounter context from event payload.
     */
    public static EncounterContext fromPayload(Map<String, Object> payload) {
        EncounterContext context = new EncounterContext();

        context.encounterId = (String) payload.get("encounter_id");
        context.encounterType = (String) payload.get("encounter_type");
        context.department = (String) payload.get("department");
        context.room = (String) payload.get("room");
        context.bed = (String) payload.get("bed");
        context.attendingPhysician = (String) payload.get("attending_physician");
        context.admissionReason = (String) payload.get("admission_reason");

        // Parse care team if present
        Object careTeamObj = payload.get("care_team");
        if (careTeamObj instanceof List) {
            @SuppressWarnings("unchecked")
            List<String> team = (List<String>) careTeamObj;
            context.careTeam = team;
        }

        return context;
    }

    // ============================================================
    // UPDATE METHODS
    // ============================================================

    /**
     * Update department and room (for transfer events).
     */
    public void updateDepartment(String department, String room) {
        this.department = department;
        this.room = room;
    }

    /**
     * Add provider to care team.
     */
    public void addCareTeamMember(String providerId) {
        if (!careTeam.contains(providerId)) {
            careTeam.add(providerId);
        }
    }

    // ============================================================
    // GETTERS AND SETTERS
    // ============================================================

    public String getEncounterId() {
        return encounterId;
    }

    public void setEncounterId(String encounterId) {
        this.encounterId = encounterId;
    }

    public String getEncounterType() {
        return encounterType;
    }

    public void setEncounterType(String encounterType) {
        this.encounterType = encounterType;
    }

    public long getAdmissionTime() {
        return admissionTime;
    }

    public void setAdmissionTime(long admissionTime) {
        this.admissionTime = admissionTime;
    }

    public Long getDischargeTime() {
        return dischargeTime;
    }

    public void setDischargeTime(Long dischargeTime) {
        this.dischargeTime = dischargeTime;
    }

    public String getDepartment() {
        return department;
    }

    public void setDepartment(String department) {
        this.department = department;
    }

    public String getRoom() {
        return room;
    }

    public void setRoom(String room) {
        this.room = room;
    }

    public String getBed() {
        return bed;
    }

    public void setBed(String bed) {
        this.bed = bed;
    }

    public String getAttendingPhysician() {
        return attendingPhysician;
    }

    public void setAttendingPhysician(String attendingPhysician) {
        this.attendingPhysician = attendingPhysician;
    }

    public List<String> getCareTeam() {
        return careTeam;
    }

    public void setCareTeam(List<String> careTeam) {
        this.careTeam = careTeam;
    }

    public String getAdmissionReason() {
        return admissionReason;
    }

    public void setAdmissionReason(String admissionReason) {
        this.admissionReason = admissionReason;
    }

    // ============================================================
    // ALIAS METHODS (for serializer compatibility)
    // ============================================================

    /**
     * Get patient ID (alias, stored in parent PatientSnapshot).
     * For serializer compatibility with legacy format.
     */
    public String getPatientId() {
        // Patient ID is managed at the PatientSnapshot level
        // This is here for backward compatibility only
        return null;
    }

    /**
     * Set patient ID (no-op for serializer compatibility).
     */
    public void setPatientId(String patientId) {
        // No-op: Patient ID is managed at PatientSnapshot level
    }

    /**
     * Get encounter status (derived from discharge time).
     * @return "active" if not discharged, "finished" if discharged
     */
    public String getStatus() {
        return dischargeTime == null ? "active" : "finished";
    }

    /**
     * Set encounter status (updates discharge time).
     */
    public void setStatus(String status) {
        if ("finished".equals(status) && dischargeTime == null) {
            this.dischargeTime = System.currentTimeMillis();
        }
    }

    /**
     * Get encounter class (alias for encounterType).
     */
    public String getEncounterClass() {
        return encounterType;
    }

    /**
     * Set encounter class (alias for encounterType).
     */
    public void setEncounterClass(String encounterClass) {
        this.encounterType = encounterClass;
    }

    /**
     * Get start time (alias for admissionTime).
     */
    public long getStartTime() {
        return admissionTime;
    }

    /**
     * Set start time (alias for admissionTime).
     */
    public void setStartTime(long startTime) {
        this.admissionTime = startTime;
    }

    /**
     * Get end time (alias for dischargeTime).
     */
    public Long getEndTime() {
        return dischargeTime;
    }

    /**
     * Set end time (alias for dischargeTime).
     */
    public void setEndTime(Long endTime) {
        this.dischargeTime = endTime;
    }

    @Override
    public String toString() {
        return "EncounterContext{" +
                "encounterId='" + encounterId + '\'' +
                ", type='" + encounterType + '\'' +
                ", department='" + department + '\'' +
                ", room='" + room + '\'' +
                '}';
    }
}
