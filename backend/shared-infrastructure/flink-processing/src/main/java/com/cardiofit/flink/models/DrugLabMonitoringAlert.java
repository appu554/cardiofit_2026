package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.List;

/**
 * Alert for missing required lab monitoring after high-risk medication administration.
 *
 * Certain medications require specific laboratory monitoring to detect adverse effects:
 *
 * Examples:
 * - Warfarin → INR/PT monitoring (every 1-4 weeks)
 * - Digoxin → Digoxin level, Potassium, Creatinine (baseline and as needed)
 * - Lithium → Lithium level, Creatinine, TSH (every 3-6 months)
 * - Methotrexate → CBC, LFTs, Creatinine (monthly initially)
 * - ACE Inhibitors → Potassium, Creatinine (within 1-2 weeks of initiation)
 * - Statins → LFTs (baseline, 12 weeks, then annually)
 * - Aminoglycosides → Peak/Trough levels, Creatinine (every 2-3 days)
 * - Vancomycin → Trough level, Creatinine (before 4th dose, then 2-3x/week)
 *
 * This alert triggers when:
 * 1. High-risk medication is administered
 * 2. Required monitoring timeframe has elapsed
 * 3. Required lab test has not been performed
 */
public class DrugLabMonitoringAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private String medicationName;
    private String medicationClass;
    private List<String> requiredLabs;
    private List<String> missingLabs;
    private Long medicationStartTime;
    private Long alertTime;
    private String urgency;
    private String recommendations;

    public DrugLabMonitoringAlert() {}

    // Getters and Setters
    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getMedicationName() {
        return medicationName;
    }

    public void setMedicationName(String medicationName) {
        this.medicationName = medicationName;
    }

    public String getMedicationClass() {
        return medicationClass;
    }

    public void setMedicationClass(String medicationClass) {
        this.medicationClass = medicationClass;
    }

    public List<String> getRequiredLabs() {
        return requiredLabs;
    }

    public void setRequiredLabs(List<String> requiredLabs) {
        this.requiredLabs = requiredLabs;
    }

    public List<String> getMissingLabs() {
        return missingLabs;
    }

    public void setMissingLabs(List<String> missingLabs) {
        this.missingLabs = missingLabs;
    }

    public Long getMedicationStartTime() {
        return medicationStartTime;
    }

    public void setMedicationStartTime(Long medicationStartTime) {
        this.medicationStartTime = medicationStartTime;
    }

    public Long getAlertTime() {
        return alertTime;
    }

    public void setAlertTime(Long alertTime) {
        this.alertTime = alertTime;
    }

    public String getUrgency() {
        return urgency;
    }

    public void setUrgency(String urgency) {
        this.urgency = urgency;
    }

    public String getRecommendations() {
        return recommendations;
    }

    public void setRecommendations(String recommendations) {
        this.recommendations = recommendations;
    }

    @Override
    public String toString() {
        return "DrugLabMonitoringAlert{" +
                "patientId='" + patientId + '\'' +
                ", medicationName='" + medicationName + '\'' +
                ", medicationClass='" + medicationClass + '\'' +
                ", requiredLabs=" + requiredLabs +
                ", missingLabs=" + missingLabs +
                ", medicationStartTime=" + medicationStartTime +
                ", alertTime=" + alertTime +
                ", urgency='" + urgency + '\'' +
                ", recommendations='" + recommendations + '\'' +
                '}';
    }
}
