package com.cardiofit.flink.cds.pathways;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Phase 8 Module 3 - Clinical Pathways Engine
 *
 * Core clinical pathway model representing a standardized care protocol.
 * Pathways guide clinical decision-making through evidence-based steps.
 *
 * Examples:
 * - Acute Coronary Syndrome (ACS) pathway
 * - Sepsis management (Surviving Sepsis Campaign)
 * - Stroke protocol (AHA guidelines)
 * - Heart Failure management
 * - Respiratory Failure protocol
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class ClinicalPathway implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core Identification
    private String pathwayId;
    private String pathwayName;
    private String pathwayVersion;
    private PathwayType pathwayType;

    // Metadata
    private String description;
    private String clinicalGuideline;       // e.g., "AHA 2020", "Surviving Sepsis 2021"
    private String evidenceLevel;           // "A" (high), "B" (moderate), "C" (low)
    private LocalDateTime createdDate;
    private LocalDateTime lastUpdatedDate;
    private String createdBy;
    private boolean isActive;

    // Pathway Structure
    private List<PathwayStep> steps;
    private String initialStepId;           // First step in pathway
    private Map<String, String> decisionPoints; // StepId -> next step based on condition

    // Clinical Context
    private List<String> applicableDiagnoses;   // ICD-10 codes this pathway applies to
    private List<String> inclusionCriteria;     // Patient must meet these to enter pathway
    private List<String> exclusionCriteria;     // Patient cannot meet these to enter pathway

    // Time Constraints
    private Integer expectedDurationMinutes;    // Expected total pathway completion time
    private Integer maxDurationMinutes;         // Maximum time before escalation required

    // Quality Metrics
    private Map<String, Object> qualityMetrics; // Track adherence and outcomes
    private List<String> criticalTimePoints;    // Steps with strict time requirements

    // Pathway Outcomes
    private List<String> expectedOutcomes;      // Desired clinical outcomes
    private Map<String, Double> historicalSuccessRates; // Outcome -> success rate

    /**
     * Pathway types based on clinical urgency and domain
     */
    public enum PathwayType {
        EMERGENCY,          // Time-critical (e.g., STEMI, stroke, sepsis)
        URGENT,             // Requires prompt action (e.g., ACS, PE)
        ROUTINE,            // Standard care protocols
        CHRONIC_MANAGEMENT, // Long-term disease management
        PREVENTIVE,         // Preventive care protocols
        DIAGNOSTIC,         // Diagnostic workup pathways
        THERAPEUTIC,        // Treatment protocols
        PALLIATIVE          // End-of-life care pathways
    }

    // Constructors
    public ClinicalPathway() {
        this.steps = new ArrayList<>();
        this.decisionPoints = new HashMap<>();
        this.applicableDiagnoses = new ArrayList<>();
        this.inclusionCriteria = new ArrayList<>();
        this.exclusionCriteria = new ArrayList<>();
        this.qualityMetrics = new HashMap<>();
        this.criticalTimePoints = new ArrayList<>();
        this.expectedOutcomes = new ArrayList<>();
        this.historicalSuccessRates = new HashMap<>();
        this.createdDate = LocalDateTime.now();
        this.isActive = true;
    }

    public ClinicalPathway(String pathwayId, String pathwayName, PathwayType pathwayType) {
        this();
        this.pathwayId = pathwayId;
        this.pathwayName = pathwayName;
        this.pathwayType = pathwayType;
    }

    /**
     * Add a step to the pathway
     */
    public void addStep(PathwayStep step) {
        if (this.steps == null) {
            this.steps = new ArrayList<>();
        }
        this.steps.add(step);

        // Set initial step if this is the first step
        if (this.steps.size() == 1 && this.initialStepId == null) {
            this.initialStepId = step.getStepId();
        }
    }

    /**
     * Add a decision point (conditional transition between steps)
     */
    public void addDecisionPoint(String fromStepId, String condition, String toStepId) {
        String key = fromStepId + ":" + condition;
        this.decisionPoints.put(key, toStepId);
    }

    /**
     * Get the next step based on current step and condition
     */
    public String getNextStep(String currentStepId, String condition) {
        String key = currentStepId + ":" + condition;
        return this.decisionPoints.get(key);
    }

    /**
     * Find a step by its ID
     */
    public PathwayStep findStep(String stepId) {
        if (steps == null) return null;
        return steps.stream()
            .filter(s -> s.getStepId().equals(stepId))
            .findFirst()
            .orElse(null);
    }

    /**
     * Get all steps of a specific type
     */
    public List<PathwayStep> getStepsByType(PathwayStep.StepType stepType) {
        if (steps == null) return new ArrayList<>();
        return steps.stream()
            .filter(s -> s.getStepType() == stepType)
            .toList();
    }

    /**
     * Get critical time-sensitive steps
     */
    public List<PathwayStep> getCriticalSteps() {
        if (steps == null) return new ArrayList<>();
        return steps.stream()
            .filter(PathwayStep::isTimeCritical)
            .toList();
    }

    /**
     * Calculate total expected duration from all steps
     */
    public int calculateTotalExpectedDuration() {
        if (steps == null || steps.isEmpty()) return 0;
        return steps.stream()
            .mapToInt(s -> s.getExpectedDurationMinutes() != null ? s.getExpectedDurationMinutes() : 0)
            .sum();
    }

    /**
     * Validate pathway structure
     */
    public boolean validate() {
        // Must have at least one step
        if (steps == null || steps.isEmpty()) {
            return false;
        }

        // Must have initial step defined
        if (initialStepId == null || initialStepId.isEmpty()) {
            return false;
        }

        // Initial step must exist in steps list
        if (findStep(initialStepId) == null) {
            return false;
        }

        // All steps must have valid IDs
        for (PathwayStep step : steps) {
            if (step.getStepId() == null || step.getStepId().isEmpty()) {
                return false;
            }
        }

        // All decision point references must point to valid steps
        for (String targetStepId : decisionPoints.values()) {
            if (findStep(targetStepId) == null) {
                return false;
            }
        }

        return true;
    }

    /**
     * Check if a patient meets pathway inclusion criteria
     */
    public boolean meetsInclusionCriteria(Map<String, Object> patientData) {
        if (inclusionCriteria == null || inclusionCriteria.isEmpty()) {
            return true; // No specific criteria means all patients qualify
        }

        // Implementation would check patient data against criteria
        // For now, return true (to be implemented with actual criteria evaluation)
        return true;
    }

    /**
     * Check if a patient meets pathway exclusion criteria
     */
    public boolean meetsExclusionCriteria(Map<String, Object> patientData) {
        if (exclusionCriteria == null || exclusionCriteria.isEmpty()) {
            return false; // No exclusions means patient is not excluded
        }

        // Implementation would check patient data against exclusion criteria
        return false;
    }

    // Getters and Setters
    public String getPathwayId() {
        return pathwayId;
    }

    public void setPathwayId(String pathwayId) {
        this.pathwayId = pathwayId;
    }

    public String getPathwayName() {
        return pathwayName;
    }

    public void setPathwayName(String pathwayName) {
        this.pathwayName = pathwayName;
    }

    public String getPathwayVersion() {
        return pathwayVersion;
    }

    public void setPathwayVersion(String pathwayVersion) {
        this.pathwayVersion = pathwayVersion;
    }

    public PathwayType getPathwayType() {
        return pathwayType;
    }

    public void setPathwayType(PathwayType pathwayType) {
        this.pathwayType = pathwayType;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public String getClinicalGuideline() {
        return clinicalGuideline;
    }

    public void setClinicalGuideline(String clinicalGuideline) {
        this.clinicalGuideline = clinicalGuideline;
    }

    public String getEvidenceLevel() {
        return evidenceLevel;
    }

    public void setEvidenceLevel(String evidenceLevel) {
        this.evidenceLevel = evidenceLevel;
    }

    public LocalDateTime getCreatedDate() {
        return createdDate;
    }

    public void setCreatedDate(LocalDateTime createdDate) {
        this.createdDate = createdDate;
    }

    public LocalDateTime getLastUpdatedDate() {
        return lastUpdatedDate;
    }

    public void setLastUpdatedDate(LocalDateTime lastUpdatedDate) {
        this.lastUpdatedDate = lastUpdatedDate;
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

    public List<PathwayStep> getSteps() {
        return steps;
    }

    public void setSteps(List<PathwayStep> steps) {
        this.steps = steps;
    }

    public String getInitialStepId() {
        return initialStepId;
    }

    public void setInitialStepId(String initialStepId) {
        this.initialStepId = initialStepId;
    }

    public Map<String, String> getDecisionPoints() {
        return decisionPoints;
    }

    public void setDecisionPoints(Map<String, String> decisionPoints) {
        this.decisionPoints = decisionPoints;
    }

    public List<String> getApplicableDiagnoses() {
        return applicableDiagnoses;
    }

    public void setApplicableDiagnoses(List<String> applicableDiagnoses) {
        this.applicableDiagnoses = applicableDiagnoses;
    }

    public void addApplicableDiagnosis(String diagnosisCode) {
        if (this.applicableDiagnoses == null) {
            this.applicableDiagnoses = new ArrayList<>();
        }
        this.applicableDiagnoses.add(diagnosisCode);
    }

    public List<String> getInclusionCriteria() {
        return inclusionCriteria;
    }

    public void setInclusionCriteria(List<String> inclusionCriteria) {
        this.inclusionCriteria = inclusionCriteria;
    }

    public void addInclusionCriterion(String criterion) {
        if (this.inclusionCriteria == null) {
            this.inclusionCriteria = new ArrayList<>();
        }
        this.inclusionCriteria.add(criterion);
    }

    public List<String> getExclusionCriteria() {
        return exclusionCriteria;
    }

    public void setExclusionCriteria(List<String> exclusionCriteria) {
        this.exclusionCriteria = exclusionCriteria;
    }

    public void addExclusionCriterion(String criterion) {
        if (this.exclusionCriteria == null) {
            this.exclusionCriteria = new ArrayList<>();
        }
        this.exclusionCriteria.add(criterion);
    }

    public Integer getExpectedDurationMinutes() {
        return expectedDurationMinutes;
    }

    public void setExpectedDurationMinutes(Integer expectedDurationMinutes) {
        this.expectedDurationMinutes = expectedDurationMinutes;
    }

    public Integer getMaxDurationMinutes() {
        return maxDurationMinutes;
    }

    public void setMaxDurationMinutes(Integer maxDurationMinutes) {
        this.maxDurationMinutes = maxDurationMinutes;
    }

    public Map<String, Object> getQualityMetrics() {
        return qualityMetrics;
    }

    public void setQualityMetrics(Map<String, Object> qualityMetrics) {
        this.qualityMetrics = qualityMetrics;
    }

    public void addQualityMetric(String metricName, Object value) {
        if (this.qualityMetrics == null) {
            this.qualityMetrics = new HashMap<>();
        }
        this.qualityMetrics.put(metricName, value);
    }

    public List<String> getCriticalTimePoints() {
        return criticalTimePoints;
    }

    public void setCriticalTimePoints(List<String> criticalTimePoints) {
        this.criticalTimePoints = criticalTimePoints;
    }

    public void addCriticalTimePoint(String stepId) {
        if (this.criticalTimePoints == null) {
            this.criticalTimePoints = new ArrayList<>();
        }
        this.criticalTimePoints.add(stepId);
    }

    public List<String> getExpectedOutcomes() {
        return expectedOutcomes;
    }

    public void setExpectedOutcomes(List<String> expectedOutcomes) {
        this.expectedOutcomes = expectedOutcomes;
    }

    public void addExpectedOutcome(String outcome) {
        if (this.expectedOutcomes == null) {
            this.expectedOutcomes = new ArrayList<>();
        }
        this.expectedOutcomes.add(outcome);
    }

    public Map<String, Double> getHistoricalSuccessRates() {
        return historicalSuccessRates;
    }

    public void setHistoricalSuccessRates(Map<String, Double> historicalSuccessRates) {
        this.historicalSuccessRates = historicalSuccessRates;
    }

    public void addHistoricalSuccessRate(String outcome, double rate) {
        if (this.historicalSuccessRates == null) {
            this.historicalSuccessRates = new HashMap<>();
        }
        this.historicalSuccessRates.put(outcome, rate);
    }

    @Override
    public String toString() {
        return String.format("ClinicalPathway{id='%s', name='%s', type=%s, steps=%d, guideline='%s'}",
            pathwayId, pathwayName, pathwayType, steps != null ? steps.size() : 0, clinicalGuideline);
    }
}
