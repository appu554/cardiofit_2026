package com.cardiofit.flink.models;

import java.io.Serializable;
import java.time.LocalDateTime;

/**
 * AdmissionRecord represents a patient admission record
 */
public class AdmissionRecord implements Serializable {
    private static final long serialVersionUID = 1L;

    private String admissionId;
    private String patientId;
    private LocalDateTime admissionDate;
    private LocalDateTime dischargeDate;
    private String admissionType;
    private String department;
    private String attending;
    private String room;
    private String bed;
    private String primaryDiagnosis;
    private String[] secondaryDiagnoses;
    private String status;

    public AdmissionRecord() {}

    public AdmissionRecord(String admissionId, String patientId, LocalDateTime admissionDate) {
        this.admissionId = admissionId;
        this.patientId = patientId;
        this.admissionDate = admissionDate;
    }

    // Getters and Setters
    public String getAdmissionId() { return admissionId; }
    public void setAdmissionId(String admissionId) { this.admissionId = admissionId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public LocalDateTime getAdmissionDate() { return admissionDate; }
    public void setAdmissionDate(LocalDateTime admissionDate) { this.admissionDate = admissionDate; }

    public LocalDateTime getDischargeDate() { return dischargeDate; }
    public void setDischargeDate(LocalDateTime dischargeDate) { this.dischargeDate = dischargeDate; }

    public String getAdmissionType() { return admissionType; }
    public void setAdmissionType(String admissionType) { this.admissionType = admissionType; }

    public String getDepartment() { return department; }
    public void setDepartment(String department) { this.department = department; }

    public String getAttending() { return attending; }
    public void setAttending(String attending) { this.attending = attending; }

    public String getRoom() { return room; }
    public void setRoom(String room) { this.room = room; }

    public String getBed() { return bed; }
    public void setBed(String bed) { this.bed = bed; }

    public String getPrimaryDiagnosis() { return primaryDiagnosis; }
    public void setPrimaryDiagnosis(String primaryDiagnosis) { this.primaryDiagnosis = primaryDiagnosis; }

    public String[] getSecondaryDiagnoses() { return secondaryDiagnoses; }
    public void setSecondaryDiagnoses(String[] secondaryDiagnoses) { this.secondaryDiagnoses = secondaryDiagnoses; }

    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }
}