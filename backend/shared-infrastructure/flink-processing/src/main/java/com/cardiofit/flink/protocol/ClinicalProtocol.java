package com.cardiofit.flink.protocol;

import java.io.Serializable;
import java.util.*;

/**
 * Clinical Protocol Definition
 */
public class ClinicalProtocol implements Serializable {
    private static final long serialVersionUID = 1L;

    private String id;
    private String name;
    private String description;
    private int priority; // 1-10, higher is more important
    private double minimumMatchThreshold; // 0-100 percentage

    // Matching criteria
    private List<String> requiredDiagnoses = new ArrayList<>();
    private List<String> riskFactors = new ArrayList<>();
    private Map<String, Criterion> vitalCriteria = new HashMap<>();
    private Map<String, Criterion> labCriteria = new HashMap<>();

    // Demographics criteria
    private Integer minAge;
    private Integer maxAge;
    private String gender; // M, F, or null for any

    // Recommendations
    private List<String> baseRecommendations = new ArrayList<>();
    private List<ConditionalRecommendation> conditionalRecommendations = new ArrayList<>();

    // Getters and setters
    public String getId() {
        return id;
    }

    public void setId(String id) {
        this.id = id;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public int getPriority() {
        return priority;
    }

    public void setPriority(int priority) {
        this.priority = priority;
    }

    public double getMinimumMatchThreshold() {
        return minimumMatchThreshold;
    }

    public void setMinimumMatchThreshold(double minimumMatchThreshold) {
        this.minimumMatchThreshold = minimumMatchThreshold;
    }

    public List<String> getRequiredDiagnoses() {
        return requiredDiagnoses;
    }

    public void setRequiredDiagnoses(List<String> requiredDiagnoses) {
        this.requiredDiagnoses = requiredDiagnoses;
    }

    public List<String> getRiskFactors() {
        return riskFactors;
    }

    public void setRiskFactors(List<String> riskFactors) {
        this.riskFactors = riskFactors;
    }

    public Map<String, Criterion> getVitalCriteria() {
        return vitalCriteria;
    }

    public void setVitalCriteria(Map<String, Criterion> vitalCriteria) {
        this.vitalCriteria = vitalCriteria;
    }

    public Map<String, Criterion> getLabCriteria() {
        return labCriteria;
    }

    public void setLabCriteria(Map<String, Criterion> labCriteria) {
        this.labCriteria = labCriteria;
    }

    public Integer getMinAge() {
        return minAge;
    }

    public void setMinAge(Integer minAge) {
        this.minAge = minAge;
    }

    public Integer getMaxAge() {
        return maxAge;
    }

    public void setMaxAge(Integer maxAge) {
        this.maxAge = maxAge;
    }

    public String getGender() {
        return gender;
    }

    public void setGender(String gender) {
        this.gender = gender;
    }

    public List<String> getBaseRecommendations() {
        return baseRecommendations;
    }

    public void setBaseRecommendations(List<String> baseRecommendations) {
        this.baseRecommendations = baseRecommendations;
    }

    public List<ConditionalRecommendation> getConditionalRecommendations() {
        return conditionalRecommendations;
    }

    public void setConditionalRecommendations(List<ConditionalRecommendation> conditionalRecommendations) {
        this.conditionalRecommendations = conditionalRecommendations;
    }

    /**
     * Criterion for numeric comparisons
     */
    public static class Criterion implements Serializable {
        private static final long serialVersionUID = 1L;
        private String operator; // >, <, >=, <=, =
        private double value;

        public Criterion() {}

        public Criterion(String operator, double value) {
            this.operator = operator;
            this.value = value;
        }

        public String getOperator() {
            return operator;
        }

        public void setOperator(String operator) {
            this.operator = operator;
        }

        public double getValue() {
            return value;
        }

        public void setValue(double value) {
            this.value = value;
        }
    }
}