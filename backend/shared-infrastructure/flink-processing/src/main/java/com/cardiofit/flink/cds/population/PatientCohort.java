package com.cardiofit.flink.cds.population;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.*;

/**
 * Phase 8 Module 4 - Population Health Module
 *
 * Represents a cohort of patients grouped by common clinical characteristics.
 * Used for population-level analytics, quality measurement, and care management.
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class PatientCohort implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core Identification
    private String cohortId;
    private String cohortName;
    private String description;
    private CohortType cohortType;

    // Patient Population
    private Set<String> patientIds;
    private int totalPatients;
    private LocalDateTime lastUpdated;

    // Inclusion/Exclusion Criteria
    private List<CriteriaRule> inclusionCriteria;
    private List<CriteriaRule> exclusionCriteria;

    // Risk Stratification
    private Map<RiskLevel, Integer> riskDistribution;
    private double averageRiskScore;

    // Demographics
    private DemographicProfile demographics;

    // Clinical Characteristics
    private Map<String, Integer> conditionDistribution;  // ICD-10 code -> count
    private Map<String, Integer> medicationDistribution; // Drug class -> count

    // Quality Metrics
    private Map<String, Double> qualityMetrics;          // Measure ID -> compliance %
    private int careGapsIdentified;

    // Metadata
    private LocalDateTime createdAt;
    private String createdBy;
    private boolean isActive;

    /**
     * Cohort types for population grouping
     */
    public enum CohortType {
        DISEASE_BASED,      // Grouped by diagnosis (e.g., diabetes, CHF)
        RISK_BASED,         // Grouped by risk score
        GEOGRAPHIC,         // Grouped by location
        DEMOGRAPHIC,        // Grouped by age, gender, etc.
        QUALITY_MEASURE,    // Grouped for quality reporting
        CARE_GAP,           // Patients with specific care gaps
        INSURANCE,          // Grouped by payer
        PROVIDER,           // Grouped by care team
        CUSTOM              // Custom criteria
    }

    /**
     * Risk levels for stratification
     */
    public enum RiskLevel {
        VERY_LOW(0, "Minimal intervention needed"),
        LOW(1, "Routine monitoring"),
        MODERATE(2, "Enhanced monitoring"),
        HIGH(3, "Care management intervention"),
        VERY_HIGH(4, "Intensive case management");

        private final int priority;
        private final String description;

        RiskLevel(int priority, String description) {
            this.priority = priority;
            this.description = description;
        }

        public int getPriority() { return priority; }
        public String getDescription() { return description; }
    }

    /**
     * Criteria rule for cohort inclusion/exclusion
     */
    public static class CriteriaRule implements Serializable {
        private static final long serialVersionUID = 1L;

        private String ruleId;
        private String description;
        private CriteriaType criteriaType;
        private String parameter;
        private String operator;        // >, <, =, IN, BETWEEN, CONTAINS
        private Object value;
        private Object secondValue;     // For BETWEEN

        public enum CriteriaType {
            AGE,
            GENDER,
            DIAGNOSIS,      // ICD-10 code
            MEDICATION,     // Drug name or class
            LAB_VALUE,      // LOINC code
            RISK_SCORE,
            VITAL_SIGN,
            INSURANCE,
            GEOGRAPHIC,
            CUSTOM
        }

        public CriteriaRule() {}

        public CriteriaRule(CriteriaType type, String parameter, String operator, Object value) {
            this.ruleId = generateRuleId();
            this.criteriaType = type;
            this.parameter = parameter;
            this.operator = operator;
            this.value = value;
        }

        private String generateRuleId() {
            return "RULE-" + System.currentTimeMillis();
        }

        // Getters and setters
        public String getRuleId() { return ruleId; }
        public void setRuleId(String ruleId) { this.ruleId = ruleId; }
        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
        public CriteriaType getCriteriaType() { return criteriaType; }
        public void setCriteriaType(CriteriaType criteriaType) { this.criteriaType = criteriaType; }
        public String getParameter() { return parameter; }
        public void setParameter(String parameter) { this.parameter = parameter; }
        public String getOperator() { return operator; }
        public void setOperator(String operator) { this.operator = operator; }
        public Object getValue() { return value; }
        public void setValue(Object value) { this.value = value; }
        public Object getSecondValue() { return secondValue; }
        public void setSecondValue(Object secondValue) { this.secondValue = secondValue; }
    }

    /**
     * Demographic profile of cohort
     */
    public static class DemographicProfile implements Serializable {
        private static final long serialVersionUID = 1L;

        private double averageAge;
        private int maleCount;
        private int femaleCount;
        private int otherGenderCount;
        private Map<String, Integer> ageRangeDistribution; // "18-34" -> count
        private Map<String, Integer> ethnicityDistribution;
        private Map<String, Integer> raceDistribution;

        public DemographicProfile() {
            this.ageRangeDistribution = new HashMap<>();
            this.ethnicityDistribution = new HashMap<>();
            this.raceDistribution = new HashMap<>();
        }

        // Getters and setters
        public double getAverageAge() { return averageAge; }
        public void setAverageAge(double averageAge) { this.averageAge = averageAge; }
        public int getMaleCount() { return maleCount; }
        public void setMaleCount(int maleCount) { this.maleCount = maleCount; }
        public int getFemaleCount() { return femaleCount; }
        public void setFemaleCount(int femaleCount) { this.femaleCount = femaleCount; }
        public int getOtherGenderCount() { return otherGenderCount; }
        public void setOtherGenderCount(int otherGenderCount) { this.otherGenderCount = otherGenderCount; }
        public Map<String, Integer> getAgeRangeDistribution() { return ageRangeDistribution; }
        public void setAgeRangeDistribution(Map<String, Integer> ageRangeDistribution) { this.ageRangeDistribution = ageRangeDistribution; }
        public Map<String, Integer> getEthnicityDistribution() { return ethnicityDistribution; }
        public void setEthnicityDistribution(Map<String, Integer> ethnicityDistribution) { this.ethnicityDistribution = ethnicityDistribution; }
        public Map<String, Integer> getRaceDistribution() { return raceDistribution; }
        public void setRaceDistribution(Map<String, Integer> raceDistribution) { this.raceDistribution = raceDistribution; }
    }

    // Constructors
    public PatientCohort() {
        this.cohortId = generateCohortId();
        this.patientIds = new HashSet<>();
        this.inclusionCriteria = new ArrayList<>();
        this.exclusionCriteria = new ArrayList<>();
        this.riskDistribution = new HashMap<>();
        this.conditionDistribution = new HashMap<>();
        this.medicationDistribution = new HashMap<>();
        this.qualityMetrics = new HashMap<>();
        this.demographics = new DemographicProfile();
        this.createdAt = LocalDateTime.now();
        this.lastUpdated = LocalDateTime.now();
        this.isActive = true;
    }

    public PatientCohort(String cohortName, CohortType cohortType) {
        this();
        this.cohortName = cohortName;
        this.cohortType = cohortType;
    }

    private String generateCohortId() {
        return "COHORT-" + System.currentTimeMillis();
    }

    /**
     * Add patient to cohort
     */
    public void addPatient(String patientId) {
        this.patientIds.add(patientId);
        this.totalPatients = this.patientIds.size();
        this.lastUpdated = LocalDateTime.now();
    }

    /**
     * Remove patient from cohort
     */
    public void removePatient(String patientId) {
        this.patientIds.remove(patientId);
        this.totalPatients = this.patientIds.size();
        this.lastUpdated = LocalDateTime.now();
    }

    /**
     * Add inclusion criteria
     */
    public void addInclusionCriteria(CriteriaRule rule) {
        if (this.inclusionCriteria == null) {
            this.inclusionCriteria = new ArrayList<>();
        }
        this.inclusionCriteria.add(rule);
    }

    /**
     * Add exclusion criteria
     */
    public void addExclusionCriteria(CriteriaRule rule) {
        if (this.exclusionCriteria == null) {
            this.exclusionCriteria = new ArrayList<>();
        }
        this.exclusionCriteria.add(rule);
    }

    /**
     * Update risk distribution
     */
    public void updateRiskDistribution(RiskLevel level, int count) {
        this.riskDistribution.put(level, count);
        this.lastUpdated = LocalDateTime.now();
    }

    /**
     * Update condition distribution
     */
    public void updateConditionDistribution(String icd10Code, int count) {
        this.conditionDistribution.put(icd10Code, count);
        this.lastUpdated = LocalDateTime.now();
    }

    /**
     * Update quality metric
     */
    public void updateQualityMetric(String measureId, double complianceRate) {
        this.qualityMetrics.put(measureId, complianceRate);
        this.lastUpdated = LocalDateTime.now();
    }

    /**
     * Get high-risk patient count
     */
    public int getHighRiskPatientCount() {
        int highRisk = this.riskDistribution.getOrDefault(RiskLevel.HIGH, 0);
        int veryHighRisk = this.riskDistribution.getOrDefault(RiskLevel.VERY_HIGH, 0);
        return highRisk + veryHighRisk;
    }

    /**
     * Get quality compliance rate
     */
    public double getOverallQualityCompliance() {
        if (qualityMetrics == null || qualityMetrics.isEmpty()) {
            return 0.0;
        }
        double sum = qualityMetrics.values().stream()
            .mapToDouble(Double::doubleValue)
            .sum();
        return sum / qualityMetrics.size();
    }

    // Getters and Setters
    public String getCohortId() {
        return cohortId;
    }

    public void setCohortId(String cohortId) {
        this.cohortId = cohortId;
    }

    public String getCohortName() {
        return cohortName;
    }

    public void setCohortName(String cohortName) {
        this.cohortName = cohortName;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public CohortType getCohortType() {
        return cohortType;
    }

    public void setCohortType(CohortType cohortType) {
        this.cohortType = cohortType;
    }

    public Set<String> getPatientIds() {
        return patientIds;
    }

    public void setPatientIds(Set<String> patientIds) {
        this.patientIds = patientIds;
        this.totalPatients = patientIds != null ? patientIds.size() : 0;
    }

    public int getTotalPatients() {
        return totalPatients;
    }

    public void setTotalPatients(int totalPatients) {
        this.totalPatients = totalPatients;
    }

    public LocalDateTime getLastUpdated() {
        return lastUpdated;
    }

    public void setLastUpdated(LocalDateTime lastUpdated) {
        this.lastUpdated = lastUpdated;
    }

    public List<CriteriaRule> getInclusionCriteria() {
        return inclusionCriteria;
    }

    public void setInclusionCriteria(List<CriteriaRule> inclusionCriteria) {
        this.inclusionCriteria = inclusionCriteria;
    }

    public List<CriteriaRule> getExclusionCriteria() {
        return exclusionCriteria;
    }

    public void setExclusionCriteria(List<CriteriaRule> exclusionCriteria) {
        this.exclusionCriteria = exclusionCriteria;
    }

    public Map<RiskLevel, Integer> getRiskDistribution() {
        return riskDistribution;
    }

    public void setRiskDistribution(Map<RiskLevel, Integer> riskDistribution) {
        this.riskDistribution = riskDistribution;
    }

    public double getAverageRiskScore() {
        return averageRiskScore;
    }

    public void setAverageRiskScore(double averageRiskScore) {
        this.averageRiskScore = averageRiskScore;
    }

    public DemographicProfile getDemographics() {
        return demographics;
    }

    public void setDemographics(DemographicProfile demographics) {
        this.demographics = demographics;
    }

    public Map<String, Integer> getConditionDistribution() {
        return conditionDistribution;
    }

    public void setConditionDistribution(Map<String, Integer> conditionDistribution) {
        this.conditionDistribution = conditionDistribution;
    }

    public Map<String, Integer> getMedicationDistribution() {
        return medicationDistribution;
    }

    public void setMedicationDistribution(Map<String, Integer> medicationDistribution) {
        this.medicationDistribution = medicationDistribution;
    }

    public Map<String, Double> getQualityMetrics() {
        return qualityMetrics;
    }

    public void setQualityMetrics(Map<String, Double> qualityMetrics) {
        this.qualityMetrics = qualityMetrics;
    }

    public int getCareGapsIdentified() {
        return careGapsIdentified;
    }

    public void setCareGapsIdentified(int careGapsIdentified) {
        this.careGapsIdentified = careGapsIdentified;
    }

    public LocalDateTime getCreatedAt() {
        return createdAt;
    }

    public void setCreatedAt(LocalDateTime createdAt) {
        this.createdAt = createdAt;
    }

    public String getCreatedBy() {
        return createdBy;
    }

    public void setCreatedBy(String createdBy) {
        this.createdBy = createdBy;
    }

    public boolean isActive() {
        return isActive;
    }

    public void setActive(boolean active) {
        isActive = active;
    }

    @Override
    public String toString() {
        return String.format("PatientCohort{id='%s', name='%s', type=%s, patients=%d, avgRiskScore=%.2f, careGaps=%d}",
            cohortId, cohortName, cohortType, totalPatients, averageRiskScore, careGapsIdentified);
    }
}
