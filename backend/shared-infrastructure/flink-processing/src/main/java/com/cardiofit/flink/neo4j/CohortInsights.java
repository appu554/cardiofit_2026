package com.cardiofit.flink.neo4j;

import java.io.Serializable;

/**
 * Cohort Analytics Insights from Neo4j
 *
 * Aggregated statistics about the patient's risk cohort
 */
public class CohortInsights implements Serializable {
    private static final long serialVersionUID = 1L;

    private String cohortName;
    private int cohortSize;
    private double readmissionRate30Day;      // Percentage (0.0 to 1.0)
    private double avgSystolicBP;
    private double avgDiastolicBP;
    private double avgHeartRate;
    private int activeMembers;
    private String riskLevel;                 // LOW, MEDIUM, HIGH

    public CohortInsights() {
    }

    public CohortInsights(String cohortName, int cohortSize) {
        this.cohortName = cohortName;
        this.cohortSize = cohortSize;
    }

    // Getters and Setters

    public String getCohortName() {
        return cohortName;
    }

    public void setCohortName(String cohortName) {
        this.cohortName = cohortName;
    }

    public int getCohortSize() {
        return cohortSize;
    }

    public void setCohortSize(int cohortSize) {
        this.cohortSize = cohortSize;
    }

    public double getReadmissionRate30Day() {
        return readmissionRate30Day;
    }

    public void setReadmissionRate30Day(double readmissionRate30Day) {
        this.readmissionRate30Day = readmissionRate30Day;
    }

    public double getAvgSystolicBP() {
        return avgSystolicBP;
    }

    public void setAvgSystolicBP(double avgSystolicBP) {
        this.avgSystolicBP = avgSystolicBP;
    }

    public double getAvgDiastolicBP() {
        return avgDiastolicBP;
    }

    public void setAvgDiastolicBP(double avgDiastolicBP) {
        this.avgDiastolicBP = avgDiastolicBP;
    }

    public double getAvgHeartRate() {
        return avgHeartRate;
    }

    public void setAvgHeartRate(double avgHeartRate) {
        this.avgHeartRate = avgHeartRate;
    }

    public int getActiveMembers() {
        return activeMembers;
    }

    public void setActiveMembers(int activeMembers) {
        this.activeMembers = activeMembers;
    }

    public String getRiskLevel() {
        return riskLevel;
    }

    public void setRiskLevel(String riskLevel) {
        this.riskLevel = riskLevel;
    }

    @Override
    public String toString() {
        return "CohortInsights{" +
                "cohortName='" + cohortName + '\'' +
                ", cohortSize=" + cohortSize +
                ", readmissionRate30Day=" + String.format("%.1f%%", readmissionRate30Day * 100) +
                ", avgSystolicBP=" + String.format("%.1f", avgSystolicBP) +
                ", riskLevel='" + riskLevel + '\'' +
                '}';
    }
}
