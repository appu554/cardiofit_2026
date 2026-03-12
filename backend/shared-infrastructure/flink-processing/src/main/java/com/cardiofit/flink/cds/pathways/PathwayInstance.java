package com.cardiofit.flink.cds.pathways;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.time.Duration;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Phase 8 Module 3 - Clinical Pathways Engine
 *
 * Represents an actual patient's progression through a clinical pathway.
 * Tracks state, timing, deviations, and outcomes.
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class PathwayInstance implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core Identification
    private String instanceId;
    private String patientId;
    private String pathwayId;
    private String pathwayName;

    // State Tracking
    private InstanceStatus status;
    private String currentStepId;
    private List<StepExecution> completedSteps;
    private List<StepExecution> pendingSteps;

    // Timing
    private LocalDateTime startTime;
    private LocalDateTime endTime;
    private LocalDateTime lastUpdateTime;
    private Long totalDurationMinutes;

    // Deviation Tracking
    private List<Deviation> deviations;
    private boolean hasDeviations;
    private int deviationCount;

    // Clinical Context
    private String admissionDiagnosis;
    private String currentClinicalStatus;
    private Map<String, Object> patientData;    // Current patient data for decision-making

    // Quality and Outcomes
    private Map<String, Boolean> qualityMeasuresMet;
    private double adherenceScore;              // 0.0-1.0, pathway adherence
    private String finalOutcome;
    private String outcomeNotes;

    // Audit Trail
    private List<String> clinicianActions;
    private String initiatedBy;
    private String completedBy;

    /**
     * Instance status
     */
    public enum InstanceStatus {
        INITIATED,      // Pathway started but not actively executing
        IN_PROGRESS,    // Currently executing steps
        SUSPENDED,      // Temporarily paused (e.g., waiting for lab results)
        DEVIATED,       // Off pathway due to clinical judgment
        COMPLETED,      // Pathway completed successfully
        DISCONTINUED,   // Pathway stopped (patient improved, transferred, etc.)
        FAILED          // Pathway failed (adverse outcome, complications)
    }

    /**
     * Individual step execution record
     */
    public static class StepExecution implements Serializable {
        private static final long serialVersionUID = 1L;

        private String stepId;
        private String stepName;
        private LocalDateTime startTime;
        private LocalDateTime endTime;
        private Long durationMinutes;
        private boolean completed;
        private boolean onTime;             // Completed within expected duration
        private String executedBy;
        private List<String> actionsPerformed;
        private Map<String, Object> dataCollected;  // Data gathered during this step
        private String notes;

        public StepExecution(String stepId, String stepName) {
            this.stepId = stepId;
            this.stepName = stepName;
            this.startTime = LocalDateTime.now();
            this.actionsPerformed = new ArrayList<>();
            this.dataCollected = new HashMap<>();
            this.completed = false;
        }

        public void complete() {
            this.endTime = LocalDateTime.now();
            this.completed = true;
            this.durationMinutes = Duration.between(startTime, endTime).toMinutes();
        }

        // Getters and setters
        public String getStepId() { return stepId; }
        public void setStepId(String stepId) { this.stepId = stepId; }
        public String getStepName() { return stepName; }
        public void setStepName(String stepName) { this.stepName = stepName; }
        public LocalDateTime getStartTime() { return startTime; }
        public void setStartTime(LocalDateTime startTime) { this.startTime = startTime; }
        public LocalDateTime getEndTime() { return endTime; }
        public void setEndTime(LocalDateTime endTime) { this.endTime = endTime; }
        public Long getDurationMinutes() { return durationMinutes; }
        public void setDurationMinutes(Long durationMinutes) { this.durationMinutes = durationMinutes; }
        public boolean isCompleted() { return completed; }
        public void setCompleted(boolean completed) { this.completed = completed; }
        public boolean isOnTime() { return onTime; }
        public void setOnTime(boolean onTime) { this.onTime = onTime; }
        public String getExecutedBy() { return executedBy; }
        public void setExecutedBy(String executedBy) { this.executedBy = executedBy; }
        public List<String> getActionsPerformed() { return actionsPerformed; }
        public void setActionsPerformed(List<String> actionsPerformed) { this.actionsPerformed = actionsPerformed; }
        public Map<String, Object> getDataCollected() { return dataCollected; }
        public void setDataCollected(Map<String, Object> dataCollected) { this.dataCollected = dataCollected; }
        public String getNotes() { return notes; }
        public void setNotes(String notes) { this.notes = notes; }

        @Override
        public String toString() {
            return String.format("StepExecution{id='%s', name='%s', duration=%dmin, completed=%s}",
                stepId, stepName, durationMinutes, completed);
        }
    }

    /**
     * Deviation from expected pathway
     */
    public static class Deviation implements Serializable {
        private static final long serialVersionUID = 1L;

        private String deviationId;
        private DeviationType deviationType;
        private String stepId;
        private LocalDateTime deviationTime;
        private String reason;
        private String clinicalJustification;
        private Severity severity;
        private boolean wasResolved;
        private String resolution;

        public enum DeviationType {
            STEP_SKIPPED,           // Required step was skipped
            TIME_EXCEEDED,          // Step took longer than expected
            WRONG_SEQUENCE,         // Steps performed out of order
            MISSING_DATA,           // Required data not available
            CONTRAINDICATION,       // Clinical contraindication arose
            PATIENT_REFUSAL,        // Patient declined intervention
            RESOURCE_UNAVAILABLE,   // Required resource not available
            CLINICAL_OVERRIDE,      // Physician override for clinical reasons
            ADVERSE_EVENT           // Complication or adverse event
        }

        public enum Severity {
            LOW,        // Minor deviation, no clinical impact
            MODERATE,   // Notable deviation, potential impact
            HIGH,       // Significant deviation, likely impacted care
            CRITICAL    // Major deviation, immediate review needed
        }

        public Deviation(DeviationType type, String stepId, String reason) {
            this.deviationId = generateDeviationId();
            this.deviationType = type;
            this.stepId = stepId;
            this.reason = reason;
            this.deviationTime = LocalDateTime.now();
            this.wasResolved = false;
        }

        private String generateDeviationId() {
            return "DEV-" + System.currentTimeMillis();
        }

        // Getters and setters
        public String getDeviationId() { return deviationId; }
        public void setDeviationId(String deviationId) { this.deviationId = deviationId; }
        public DeviationType getDeviationType() { return deviationType; }
        public void setDeviationType(DeviationType deviationType) { this.deviationType = deviationType; }
        public String getStepId() { return stepId; }
        public void setStepId(String stepId) { this.stepId = stepId; }
        public LocalDateTime getDeviationTime() { return deviationTime; }
        public void setDeviationTime(LocalDateTime deviationTime) { this.deviationTime = deviationTime; }
        public String getReason() { return reason; }
        public void setReason(String reason) { this.reason = reason; }
        public String getClinicalJustification() { return clinicalJustification; }
        public void setClinicalJustification(String clinicalJustification) { this.clinicalJustification = clinicalJustification; }
        public Severity getSeverity() { return severity; }
        public void setSeverity(Severity severity) { this.severity = severity; }
        public boolean isWasResolved() { return wasResolved; }
        public void setWasResolved(boolean wasResolved) { this.wasResolved = wasResolved; }
        public String getResolution() { return resolution; }
        public void setResolution(String resolution) { this.resolution = resolution; }

        @Override
        public String toString() {
            return String.format("Deviation{type=%s, step='%s', severity=%s, resolved=%s}",
                deviationType, stepId, severity, wasResolved);
        }
    }

    // Constructors
    public PathwayInstance() {
        this.instanceId = generateInstanceId();
        this.status = InstanceStatus.INITIATED;
        this.startTime = LocalDateTime.now();
        this.lastUpdateTime = LocalDateTime.now();
        this.completedSteps = new ArrayList<>();
        this.pendingSteps = new ArrayList<>();
        this.deviations = new ArrayList<>();
        this.patientData = new HashMap<>();
        this.qualityMeasuresMet = new HashMap<>();
        this.clinicianActions = new ArrayList<>();
        this.hasDeviations = false;
        this.deviationCount = 0;
    }

    public PathwayInstance(String patientId, String pathwayId, String pathwayName) {
        this();
        this.patientId = patientId;
        this.pathwayId = pathwayId;
        this.pathwayName = pathwayName;
    }

    private String generateInstanceId() {
        return "PATHWAY-INST-" + System.currentTimeMillis();
    }

    /**
     * Start execution of a step
     */
    public StepExecution startStep(String stepId, String stepName) {
        StepExecution execution = new StepExecution(stepId, stepName);
        this.currentStepId = stepId;
        this.status = InstanceStatus.IN_PROGRESS;
        this.lastUpdateTime = LocalDateTime.now();
        return execution;
    }

    /**
     * Complete the current step
     */
    public void completeStep(StepExecution execution, boolean onTime) {
        execution.complete();
        execution.setOnTime(onTime);
        this.completedSteps.add(execution);
        this.lastUpdateTime = LocalDateTime.now();
    }

    /**
     * Record a deviation
     */
    public void recordDeviation(Deviation.DeviationType type, String stepId, String reason, Deviation.Severity severity) {
        Deviation deviation = new Deviation(type, stepId, reason);
        deviation.setSeverity(severity);
        this.deviations.add(deviation);
        this.hasDeviations = true;
        this.deviationCount++;
        this.lastUpdateTime = LocalDateTime.now();

        // Mark instance as deviated if severe
        if (severity == Deviation.Severity.HIGH || severity == Deviation.Severity.CRITICAL) {
            this.status = InstanceStatus.DEVIATED;
        }
    }

    /**
     * Complete the pathway
     */
    public void complete(String outcome) {
        this.status = InstanceStatus.COMPLETED;
        this.endTime = LocalDateTime.now();
        this.finalOutcome = outcome;
        this.totalDurationMinutes = Duration.between(startTime, endTime).toMinutes();
        calculateAdherenceScore();
    }

    /**
     * Discontinue the pathway
     */
    public void discontinue(String reason) {
        this.status = InstanceStatus.DISCONTINUED;
        this.endTime = LocalDateTime.now();
        this.outcomeNotes = reason;
        this.totalDurationMinutes = Duration.between(startTime, endTime).toMinutes();
    }

    /**
     * Calculate pathway adherence score
     */
    private void calculateAdherenceScore() {
        // Simple calculation: (steps completed on time) / (total steps completed)
        if (completedSteps == null || completedSteps.isEmpty()) {
            this.adherenceScore = 0.0;
            return;
        }

        long onTimeSteps = completedSteps.stream()
            .filter(StepExecution::isOnTime)
            .count();

        this.adherenceScore = (double) onTimeSteps / completedSteps.size();

        // Penalize for deviations
        if (hasDeviations) {
            double deviationPenalty = Math.min(0.1 * deviationCount, 0.3); // Max 30% penalty
            this.adherenceScore = Math.max(0.0, this.adherenceScore - deviationPenalty);
        }
    }

    /**
     * Get pathway progress percentage
     */
    public double getProgressPercentage() {
        int total = completedSteps.size() + pendingSteps.size();
        if (total == 0) return 0.0;
        return (double) completedSteps.size() / total * 100.0;
    }

    /**
     * Check if pathway is behind schedule
     */
    public boolean isBehindSchedule(Integer expectedDurationMinutes) {
        if (expectedDurationMinutes == null || startTime == null) {
            return false;
        }

        long actualDuration = Duration.between(startTime, LocalDateTime.now()).toMinutes();
        return actualDuration > expectedDurationMinutes;
    }

    // Getters and Setters
    public String getInstanceId() {
        return instanceId;
    }

    public void setInstanceId(String instanceId) {
        this.instanceId = instanceId;
    }

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

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

    public InstanceStatus getStatus() {
        return status;
    }

    public void setStatus(InstanceStatus status) {
        this.status = status;
    }

    public String getCurrentStepId() {
        return currentStepId;
    }

    public void setCurrentStepId(String currentStepId) {
        this.currentStepId = currentStepId;
    }

    public List<StepExecution> getCompletedSteps() {
        return completedSteps;
    }

    public void setCompletedSteps(List<StepExecution> completedSteps) {
        this.completedSteps = completedSteps;
    }

    public List<StepExecution> getPendingSteps() {
        return pendingSteps;
    }

    public void setPendingSteps(List<StepExecution> pendingSteps) {
        this.pendingSteps = pendingSteps;
    }

    public LocalDateTime getStartTime() {
        return startTime;
    }

    public void setStartTime(LocalDateTime startTime) {
        this.startTime = startTime;
    }

    public LocalDateTime getEndTime() {
        return endTime;
    }

    public void setEndTime(LocalDateTime endTime) {
        this.endTime = endTime;
    }

    public LocalDateTime getLastUpdateTime() {
        return lastUpdateTime;
    }

    public void setLastUpdateTime(LocalDateTime lastUpdateTime) {
        this.lastUpdateTime = lastUpdateTime;
    }

    public Long getTotalDurationMinutes() {
        return totalDurationMinutes;
    }

    public void setTotalDurationMinutes(Long totalDurationMinutes) {
        this.totalDurationMinutes = totalDurationMinutes;
    }

    public List<Deviation> getDeviations() {
        return deviations;
    }

    public void setDeviations(List<Deviation> deviations) {
        this.deviations = deviations;
    }

    public boolean isHasDeviations() {
        return hasDeviations;
    }

    public void setHasDeviations(boolean hasDeviations) {
        this.hasDeviations = hasDeviations;
    }

    public int getDeviationCount() {
        return deviationCount;
    }

    public void setDeviationCount(int deviationCount) {
        this.deviationCount = deviationCount;
    }

    public String getAdmissionDiagnosis() {
        return admissionDiagnosis;
    }

    public void setAdmissionDiagnosis(String admissionDiagnosis) {
        this.admissionDiagnosis = admissionDiagnosis;
    }

    public String getCurrentClinicalStatus() {
        return currentClinicalStatus;
    }

    public void setCurrentClinicalStatus(String currentClinicalStatus) {
        this.currentClinicalStatus = currentClinicalStatus;
    }

    public Map<String, Object> getPatientData() {
        return patientData;
    }

    public void setPatientData(Map<String, Object> patientData) {
        this.patientData = patientData;
    }

    public void updatePatientData(String key, Object value) {
        if (this.patientData == null) {
            this.patientData = new HashMap<>();
        }
        this.patientData.put(key, value);
        this.lastUpdateTime = LocalDateTime.now();
    }

    public Map<String, Boolean> getQualityMeasuresMet() {
        return qualityMeasuresMet;
    }

    public void setQualityMeasuresMet(Map<String, Boolean> qualityMeasuresMet) {
        this.qualityMeasuresMet = qualityMeasuresMet;
    }

    public void recordQualityMeasure(String measureName, boolean met) {
        if (this.qualityMeasuresMet == null) {
            this.qualityMeasuresMet = new HashMap<>();
        }
        this.qualityMeasuresMet.put(measureName, met);
    }

    public double getAdherenceScore() {
        return adherenceScore;
    }

    public void setAdherenceScore(double adherenceScore) {
        this.adherenceScore = adherenceScore;
    }

    public String getFinalOutcome() {
        return finalOutcome;
    }

    public void setFinalOutcome(String finalOutcome) {
        this.finalOutcome = finalOutcome;
    }

    public String getOutcomeNotes() {
        return outcomeNotes;
    }

    public void setOutcomeNotes(String outcomeNotes) {
        this.outcomeNotes = outcomeNotes;
    }

    public List<String> getClinicianActions() {
        return clinicianActions;
    }

    public void setClinicianActions(List<String> clinicianActions) {
        this.clinicianActions = clinicianActions;
    }

    public void recordAction(String action) {
        if (this.clinicianActions == null) {
            this.clinicianActions = new ArrayList<>();
        }
        this.clinicianActions.add(action);
    }

    public String getInitiatedBy() {
        return initiatedBy;
    }

    public void setInitiatedBy(String initiatedBy) {
        this.initiatedBy = initiatedBy;
    }

    public String getCompletedBy() {
        return completedBy;
    }

    public void setCompletedBy(String completedBy) {
        this.completedBy = completedBy;
    }

    @Override
    public String toString() {
        return String.format("PathwayInstance{id='%s', patient='%s', pathway='%s', status=%s, progress=%.1f%%, adherence=%.2f}",
            instanceId, patientId, pathwayName, status, getProgressPercentage(), adherenceScore);
    }
}
