package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * DepartmentStats tracks aggregated statistics for a department
 * Used by Module 6 Population Health Analytics
 *
 * Stored in Flink ValueState and updated each time population metrics are calculated
 * Provides summary statistics for dashboard visualization and alerting
 *
 * Risk Distribution:
 * - LOW: 0.00-0.25
 * - MODERATE: 0.25-0.50
 * - HIGH: 0.50-0.75
 * - CRITICAL: 0.75-1.00
 *
 * Example:
 * DepartmentStats {
 *   totalPatients: 24,
 *   highRiskPatients: 5,
 *   criticalPatients: 2,
 *   avgMortalityRisk: 0.18,
 *   avgSepsisRisk: 0.22,
 *   riskDistribution: { LOW: 12, MODERATE: 7, HIGH: 3, CRITICAL: 2 },
 *   trendIndicator: "STABLE"
 * }
 */
public class DepartmentStats implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("total_patients")
    private int totalPatients;

    @JsonProperty("high_risk_patients")
    private int highRiskPatients;

    @JsonProperty("critical_patients")
    private int criticalPatients;

    @JsonProperty("avg_mortality_risk")
    private double avgMortalityRisk;

    @JsonProperty("avg_sepsis_risk")
    private double avgSepsisRisk;

    @JsonProperty("risk_distribution")
    private Map<String, Integer> riskDistribution;

    @JsonProperty("trend_indicator")
    private String trendIndicator;

    @JsonProperty("last_updated")
    private long lastUpdated;

    // Default constructor
    public DepartmentStats() {
        this.riskDistribution = new HashMap<>();
        this.riskDistribution.put("LOW", 0);
        this.riskDistribution.put("MODERATE", 0);
        this.riskDistribution.put("HIGH", 0);
        this.riskDistribution.put("CRITICAL", 0);
        this.trendIndicator = "STABLE";
        this.lastUpdated = System.currentTimeMillis();
    }

    // Full constructor
    public DepartmentStats(int totalPatients, int highRiskPatients, int criticalPatients,
                          double avgMortalityRisk, double avgSepsisRisk,
                          Map<String, Integer> riskDistribution, String trendIndicator) {
        this.totalPatients = totalPatients;
        this.highRiskPatients = highRiskPatients;
        this.criticalPatients = criticalPatients;
        this.avgMortalityRisk = avgMortalityRisk;
        this.avgSepsisRisk = avgSepsisRisk;
        this.riskDistribution = riskDistribution != null ? riskDistribution : new HashMap<>();
        this.trendIndicator = trendIndicator != null ? trendIndicator : "STABLE";
        this.lastUpdated = System.currentTimeMillis();
    }

    // Builder pattern
    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private DepartmentStats stats = new DepartmentStats();

        public Builder totalPatients(int totalPatients) {
            stats.totalPatients = totalPatients;
            return this;
        }

        public Builder highRiskPatients(int highRiskPatients) {
            stats.highRiskPatients = highRiskPatients;
            return this;
        }

        public Builder criticalPatients(int criticalPatients) {
            stats.criticalPatients = criticalPatients;
            return this;
        }

        public Builder avgMortalityRisk(double avgMortalityRisk) {
            stats.avgMortalityRisk = avgMortalityRisk;
            return this;
        }

        public Builder avgSepsisRisk(double avgSepsisRisk) {
            stats.avgSepsisRisk = avgSepsisRisk;
            return this;
        }

        public Builder riskDistribution(Map<String, Integer> riskDistribution) {
            stats.riskDistribution = riskDistribution;
            return this;
        }

        public Builder trendIndicator(String trendIndicator) {
            stats.trendIndicator = trendIndicator;
            return this;
        }

        public DepartmentStats build() {
            stats.lastUpdated = System.currentTimeMillis();
            return stats;
        }
    }

    // Getters and Setters
    public int getTotalPatients() { return totalPatients; }
    public void setTotalPatients(int totalPatients) { this.totalPatients = totalPatients; }

    public int getHighRiskPatients() { return highRiskPatients; }
    public void setHighRiskPatients(int highRiskPatients) { this.highRiskPatients = highRiskPatients; }

    public int getCriticalPatients() { return criticalPatients; }
    public void setCriticalPatients(int criticalPatients) { this.criticalPatients = criticalPatients; }

    public double getAvgMortalityRisk() { return avgMortalityRisk; }
    public void setAvgMortalityRisk(double avgMortalityRisk) { this.avgMortalityRisk = avgMortalityRisk; }

    public double getAvgSepsisRisk() { return avgSepsisRisk; }
    public void setAvgSepsisRisk(double avgSepsisRisk) { this.avgSepsisRisk = avgSepsisRisk; }

    public Map<String, Integer> getRiskDistribution() { return riskDistribution; }
    public void setRiskDistribution(Map<String, Integer> riskDistribution) { this.riskDistribution = riskDistribution; }

    public String getTrendIndicator() { return trendIndicator; }
    public void setTrendIndicator(String trendIndicator) { this.trendIndicator = trendIndicator; }

    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long lastUpdated) { this.lastUpdated = lastUpdated; }

    // Utility methods
    /**
     * Calculate percentage of high risk patients
     */
    public double getHighRiskPercentage() {
        return totalPatients > 0 ? (double) highRiskPatients / totalPatients * 100.0 : 0.0;
    }

    /**
     * Calculate percentage of critical patients
     */
    public double getCriticalPercentage() {
        return totalPatients > 0 ? (double) criticalPatients / totalPatients * 100.0 : 0.0;
    }

    /**
     * Get count of patients in specific risk category
     */
    public int getRiskCount(String riskLevel) {
        return riskDistribution.getOrDefault(riskLevel, 0);
    }

    /**
     * Check if department has critical alert threshold (>20% critical)
     */
    public boolean hasCriticalAlertThreshold() {
        return getCriticalPercentage() > 20.0;
    }

    /**
     * Check if department has high risk alert threshold (>40% high+critical)
     */
    public boolean hasHighRiskAlertThreshold() {
        double highAndCriticalPct = totalPatients > 0 ?
            (double) (highRiskPatients + criticalPatients) / totalPatients * 100.0 : 0.0;
        return highAndCriticalPct > 40.0;
    }

    /**
     * Get overall department risk level
     */
    public String getDepartmentRiskLevel() {
        if (hasCriticalAlertThreshold()) return "CRITICAL";
        if (hasHighRiskAlertThreshold()) return "HIGH";
        if (getHighRiskPercentage() > 20.0) return "MODERATE";
        return "LOW";
    }

    @Override
    public String toString() {
        return "DepartmentStats{" +
                "totalPatients=" + totalPatients +
                ", highRiskPatients=" + highRiskPatients +
                ", criticalPatients=" + criticalPatients +
                ", avgMortalityRisk=" + String.format("%.3f", avgMortalityRisk) +
                ", avgSepsisRisk=" + String.format("%.3f", avgSepsisRisk) +
                ", riskDistribution=" + riskDistribution +
                ", trendIndicator='" + trendIndicator + '\'' +
                ", departmentRiskLevel='" + getDepartmentRiskLevel() + '\'' +
                '}';
    }
}
