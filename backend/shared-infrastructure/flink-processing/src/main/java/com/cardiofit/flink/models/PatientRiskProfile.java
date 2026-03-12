package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * PatientRiskProfile tracks individual patient risk levels
 * Used by Module 6 Population Health Analytics for department-level aggregation
 *
 * Stored in Flink MapState<String, PatientRiskProfile> keyed by patientId
 * Updated whenever new ML predictions arrive from Module 5
 *
 * Stale profiles (>24 hours) are automatically removed during aggregation
 *
 * Example:
 * PatientRiskProfile {
 *   patientId: "PAT-001",
 *   department: "ICU",
 *   mortalityRisk: 0.15,
 *   sepsisRisk: 0.22,
 *   readmissionRisk: 0.08,
 *   overallRiskScore: 0.32,
 *   riskLevel: "MODERATE",
 *   lastUpdated: 1762601640144
 * }
 */
public class PatientRiskProfile implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("department")
    private String department;

    @JsonProperty("mortality_risk")
    private double mortalityRisk;

    @JsonProperty("sepsis_risk")
    private double sepsisRisk;

    @JsonProperty("readmission_risk")
    private double readmissionRisk;

    @JsonProperty("overall_risk_score")
    private double overallRiskScore;

    @JsonProperty("risk_level")
    private String riskLevel;

    @JsonProperty("last_updated")
    private long lastUpdated;

    // Default constructor
    public PatientRiskProfile() {}

    // Full constructor
    public PatientRiskProfile(String patientId, String department, double mortalityRisk,
                             double sepsisRisk, double readmissionRisk, double overallRiskScore,
                             String riskLevel, long lastUpdated) {
        this.patientId = patientId;
        this.department = department;
        this.mortalityRisk = mortalityRisk;
        this.sepsisRisk = sepsisRisk;
        this.readmissionRisk = readmissionRisk;
        this.overallRiskScore = overallRiskScore;
        this.riskLevel = riskLevel;
        this.lastUpdated = lastUpdated;
    }

    // Builder pattern
    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private PatientRiskProfile profile = new PatientRiskProfile();

        public Builder patientId(String patientId) {
            profile.patientId = patientId;
            return this;
        }

        public Builder department(String department) {
            profile.department = department;
            return this;
        }

        public Builder mortalityRisk(double mortalityRisk) {
            profile.mortalityRisk = mortalityRisk;
            return this;
        }

        public Builder sepsisRisk(double sepsisRisk) {
            profile.sepsisRisk = sepsisRisk;
            return this;
        }

        public Builder readmissionRisk(double readmissionRisk) {
            profile.readmissionRisk = readmissionRisk;
            return this;
        }

        public Builder overallRiskScore(double overallRiskScore) {
            profile.overallRiskScore = overallRiskScore;
            profile.riskLevel = calculateRiskLevel(overallRiskScore);
            return this;
        }

        public Builder riskLevel(String riskLevel) {
            profile.riskLevel = riskLevel;
            return this;
        }

        public Builder lastUpdated(long lastUpdated) {
            profile.lastUpdated = lastUpdated;
            return this;
        }

        private String calculateRiskLevel(double score) {
            if (score >= 0.75) return "CRITICAL";
            if (score >= 0.50) return "HIGH";
            if (score >= 0.25) return "MODERATE";
            return "LOW";
        }

        public PatientRiskProfile build() {
            return profile;
        }
    }

    // Getters and Setters
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getDepartment() { return department; }
    public void setDepartment(String department) { this.department = department; }

    public double getMortalityRisk() { return mortalityRisk; }
    public void setMortalityRisk(double mortalityRisk) { this.mortalityRisk = mortalityRisk; }

    public double getSepsisRisk() { return sepsisRisk; }
    public void setSepsisRisk(double sepsisRisk) { this.sepsisRisk = sepsisRisk; }

    public double getReadmissionRisk() { return readmissionRisk; }
    public void setReadmissionRisk(double readmissionRisk) { this.readmissionRisk = readmissionRisk; }

    public double getOverallRiskScore() { return overallRiskScore; }
    public void setOverallRiskScore(double overallRiskScore) {
        this.overallRiskScore = overallRiskScore;
        this.riskLevel = calculateRiskLevel(overallRiskScore);
    }

    public String getRiskLevel() { return riskLevel; }
    public void setRiskLevel(String riskLevel) { this.riskLevel = riskLevel; }

    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long lastUpdated) { this.lastUpdated = lastUpdated; }

    // Utility methods
    /**
     * Calculate risk level from overall risk score
     */
    private String calculateRiskLevel(double score) {
        if (score >= 0.75) return "CRITICAL";
        if (score >= 0.50) return "HIGH";
        if (score >= 0.25) return "MODERATE";
        return "LOW";
    }

    /**
     * Check if profile is stale (>24 hours old)
     */
    public boolean isStale() {
        return isStale(24);
    }

    /**
     * Check if profile is older than specified hours
     */
    public boolean isStale(int maxAgeHours) {
        long maxAgeMs = maxAgeHours * 3600 * 1000L;
        return (System.currentTimeMillis() - lastUpdated) > maxAgeMs;
    }

    /**
     * Check if patient is high risk (>= 0.50)
     */
    public boolean isHighRisk() {
        return overallRiskScore >= 0.50;
    }

    /**
     * Check if patient is critical risk (>= 0.75)
     */
    public boolean isCriticalRisk() {
        return overallRiskScore >= 0.75;
    }

    /**
     * Get the highest individual risk component
     */
    public String getHighestRiskComponent() {
        double maxRisk = Math.max(mortalityRisk, Math.max(sepsisRisk, readmissionRisk));
        if (maxRisk == mortalityRisk) return "MORTALITY";
        if (maxRisk == sepsisRisk) return "SEPSIS";
        return "READMISSION";
    }

    @Override
    public String toString() {
        return "PatientRiskProfile{" +
                "patientId='" + patientId + '\'' +
                ", department='" + department + '\'' +
                ", overallRiskScore=" + String.format("%.3f", overallRiskScore) +
                ", riskLevel='" + riskLevel + '\'' +
                ", mortalityRisk=" + String.format("%.3f", mortalityRisk) +
                ", sepsisRisk=" + String.format("%.3f", sepsisRisk) +
                ", readmissionRisk=" + String.format("%.3f", readmissionRisk) +
                ", lastUpdated=" + lastUpdated +
                '}';
    }
}
