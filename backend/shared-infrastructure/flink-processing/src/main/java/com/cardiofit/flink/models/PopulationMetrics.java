package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * PopulationMetrics represents department-level population health statistics
 * Output by Module 6 Population Health Analytics every minute
 *
 * Aggregates patient risk profiles across a department to provide:
 * - Total patient count
 * - High/critical risk counts
 * - Average risk scores
 * - Risk distribution histogram
 * - Trend indicators
 *
 * Output Topic: analytics-population-health
 * Consumption: Dashboard APIs, alerting systems, reporting services
 *
 * Example:
 * PopulationMetrics {
 *   department: "ICU",
 *   timestamp: 1762601640144,
 *   totalPatients: 24,
 *   highRiskPatients: 5,
 *   criticalPatients: 2,
 *   avgMortalityRisk: 0.18,
 *   avgSepsisRisk: 0.22,
 *   riskDistribution: { LOW: 12, MODERATE: 7, HIGH: 3, CRITICAL: 2 },
 *   trendIndicator: "IMPROVING"
 * }
 */
public class PopulationMetrics implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("department")
    private String department;

    @JsonProperty("timestamp")
    private long timestamp;

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

    @JsonProperty("avg_readmission_risk")
    private double avgReadmissionRisk;

    @JsonProperty("risk_distribution")
    private Map<String, Integer> riskDistribution;

    @JsonProperty("trend_indicator")
    private String trendIndicator;

    @JsonProperty("department_risk_level")
    private String departmentRiskLevel;

    // Default constructor
    public PopulationMetrics() {
        this.riskDistribution = new HashMap<>();
        this.riskDistribution.put("LOW", 0);
        this.riskDistribution.put("MODERATE", 0);
        this.riskDistribution.put("HIGH", 0);
        this.riskDistribution.put("CRITICAL", 0);
        this.trendIndicator = "STABLE";
        this.departmentRiskLevel = "LOW";
    }

    // Full constructor
    public PopulationMetrics(String department, long timestamp, int totalPatients,
                            int highRiskPatients, int criticalPatients, double avgMortalityRisk,
                            double avgSepsisRisk, double avgReadmissionRisk,
                            Map<String, Integer> riskDistribution, String trendIndicator,
                            String departmentRiskLevel) {
        this.department = department;
        this.timestamp = timestamp;
        this.totalPatients = totalPatients;
        this.highRiskPatients = highRiskPatients;
        this.criticalPatients = criticalPatients;
        this.avgMortalityRisk = avgMortalityRisk;
        this.avgSepsisRisk = avgSepsisRisk;
        this.avgReadmissionRisk = avgReadmissionRisk;
        this.riskDistribution = riskDistribution != null ? riskDistribution : new HashMap<>();
        this.trendIndicator = trendIndicator != null ? trendIndicator : "STABLE";
        this.departmentRiskLevel = departmentRiskLevel != null ? departmentRiskLevel : "LOW";
    }

    // Builder pattern
    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private PopulationMetrics metrics = new PopulationMetrics();

        public Builder department(String department) {
            metrics.department = department;
            return this;
        }

        public Builder timestamp(long timestamp) {
            metrics.timestamp = timestamp;
            return this;
        }

        public Builder totalPatients(int totalPatients) {
            metrics.totalPatients = totalPatients;
            return this;
        }

        public Builder highRiskPatients(int highRiskPatients) {
            metrics.highRiskPatients = highRiskPatients;
            return this;
        }

        public Builder criticalPatients(int criticalPatients) {
            metrics.criticalPatients = criticalPatients;
            return this;
        }

        public Builder avgMortalityRisk(double avgMortalityRisk) {
            metrics.avgMortalityRisk = avgMortalityRisk;
            return this;
        }

        public Builder avgSepsisRisk(double avgSepsisRisk) {
            metrics.avgSepsisRisk = avgSepsisRisk;
            return this;
        }

        public Builder avgReadmissionRisk(double avgReadmissionRisk) {
            metrics.avgReadmissionRisk = avgReadmissionRisk;
            return this;
        }

        public Builder riskDistribution(Map<String, Integer> riskDistribution) {
            metrics.riskDistribution = riskDistribution;
            return this;
        }

        public Builder trendIndicator(String trendIndicator) {
            metrics.trendIndicator = trendIndicator;
            return this;
        }

        public Builder departmentRiskLevel(String departmentRiskLevel) {
            metrics.departmentRiskLevel = departmentRiskLevel;
            return this;
        }

        public PopulationMetrics build() {
            // Auto-calculate department risk level if not set
            if (metrics.departmentRiskLevel == null || metrics.departmentRiskLevel.equals("LOW")) {
                metrics.departmentRiskLevel = calculateDepartmentRiskLevel(
                    metrics.totalPatients,
                    metrics.highRiskPatients,
                    metrics.criticalPatients
                );
            }
            return metrics;
        }

        private String calculateDepartmentRiskLevel(int total, int highRisk, int critical) {
            if (total == 0) return "LOW";

            double criticalPct = (double) critical / total * 100.0;
            double highPct = (double) (highRisk + critical) / total * 100.0;

            if (criticalPct > 20.0) return "CRITICAL";
            if (highPct > 40.0) return "HIGH";
            if (highPct > 20.0) return "MODERATE";
            return "LOW";
        }
    }

    // Getters and Setters
    public String getDepartment() { return department; }
    public void setDepartment(String department) { this.department = department; }

    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

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

    public double getAvgReadmissionRisk() { return avgReadmissionRisk; }
    public void setAvgReadmissionRisk(double avgReadmissionRisk) { this.avgReadmissionRisk = avgReadmissionRisk; }

    public Map<String, Integer> getRiskDistribution() { return riskDistribution; }
    public void setRiskDistribution(Map<String, Integer> riskDistribution) { this.riskDistribution = riskDistribution; }

    public String getTrendIndicator() { return trendIndicator; }
    public void setTrendIndicator(String trendIndicator) { this.trendIndicator = trendIndicator; }

    public String getDepartmentRiskLevel() { return departmentRiskLevel; }
    public void setDepartmentRiskLevel(String departmentRiskLevel) { this.departmentRiskLevel = departmentRiskLevel; }

    // Utility methods
    /**
     * Get percentage of high risk patients
     */
    public double getHighRiskPercentage() {
        return totalPatients > 0 ? (double) highRiskPatients / totalPatients * 100.0 : 0.0;
    }

    /**
     * Get percentage of critical patients
     */
    public double getCriticalPercentage() {
        return totalPatients > 0 ? (double) criticalPatients / totalPatients * 100.0 : 0.0;
    }

    /**
     * Get count for specific risk level
     */
    public int getRiskCount(String riskLevel) {
        return riskDistribution.getOrDefault(riskLevel, 0);
    }

    /**
     * Check if department requires immediate attention
     */
    public boolean requiresImmediateAttention() {
        return "CRITICAL".equals(departmentRiskLevel) || getCriticalPercentage() > 25.0;
    }

    /**
     * Get overall risk score (weighted average)
     */
    public double getOverallRiskScore() {
        return (avgMortalityRisk * 0.4) + (avgSepsisRisk * 0.4) + (avgReadmissionRisk * 0.2);
    }

    @Override
    public String toString() {
        return "PopulationMetrics{" +
                "department='" + department + '\'' +
                ", timestamp=" + timestamp +
                ", totalPatients=" + totalPatients +
                ", highRiskPatients=" + highRiskPatients +
                " (" + String.format("%.1f%%", getHighRiskPercentage()) + ")" +
                ", criticalPatients=" + criticalPatients +
                " (" + String.format("%.1f%%", getCriticalPercentage()) + ")" +
                ", avgMortalityRisk=" + String.format("%.3f", avgMortalityRisk) +
                ", avgSepsisRisk=" + String.format("%.3f", avgSepsisRisk) +
                ", departmentRiskLevel='" + departmentRiskLevel + '\'' +
                ", trendIndicator='" + trendIndicator + '\'' +
                '}';
    }
}
