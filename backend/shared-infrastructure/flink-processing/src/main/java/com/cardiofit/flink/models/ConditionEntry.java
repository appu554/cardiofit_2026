package com.cardiofit.flink.models;

import java.io.Serializable;
import java.time.LocalDateTime;

/**
 * ConditionEntry represents a patient condition/diagnosis
 */
public class ConditionEntry implements Serializable {
    private static final long serialVersionUID = 1L;

    private String conditionId;
    private String patientId;
    private String conditionCode;
    private String conditionName;
    private String category;
    private String severity;
    private String status;
    private LocalDateTime onsetDate;
    private LocalDateTime resolvedDate;
    private String notes;
    private String diagnosedBy;

    public ConditionEntry() {}

    public ConditionEntry(String conditionId, String patientId, String conditionCode, String conditionName) {
        this.conditionId = conditionId;
        this.patientId = patientId;
        this.conditionCode = conditionCode;
        this.conditionName = conditionName;
    }

    // Getters and Setters
    public String getConditionId() { return conditionId; }
    public void setConditionId(String conditionId) { this.conditionId = conditionId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getConditionCode() { return conditionCode; }
    public void setConditionCode(String conditionCode) { this.conditionCode = conditionCode; }

    public String getConditionName() { return conditionName; }
    public void setConditionName(String conditionName) { this.conditionName = conditionName; }

    public String getCategory() { return category; }
    public void setCategory(String category) { this.category = category; }

    public String getSeverity() { return severity; }
    public void setSeverity(String severity) { this.severity = severity; }

    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }

    public LocalDateTime getOnsetDate() { return onsetDate; }
    public void setOnsetDate(LocalDateTime onsetDate) { this.onsetDate = onsetDate; }

    public LocalDateTime getResolvedDate() { return resolvedDate; }
    public void setResolvedDate(LocalDateTime resolvedDate) { this.resolvedDate = resolvedDate; }

    public String getNotes() { return notes; }
    public void setNotes(String notes) { this.notes = notes; }

    public String getDiagnosedBy() { return diagnosedBy; }
    public void setDiagnosedBy(String diagnosedBy) { this.diagnosedBy = diagnosedBy; }
}