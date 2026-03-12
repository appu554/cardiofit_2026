package com.cardiofit.flink.cds.pathways;

import java.time.LocalDateTime;
import java.time.Duration;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Phase 8 Module 3 - Clinical Pathways Engine
 *
 * Core execution engine for clinical pathways.
 * Manages pathway state, step transitions, branching logic, and deviation detection.
 *
 * Features:
 * - State machine-based step progression
 * - Conditional branching based on patient data
 * - Time-based deviation detection
 * - Quality measure tracking
 * - Automated alerting for delays and deviations
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class PathwayEngine {

    private static final String ENGINE_VERSION = "1.0.0";

    /**
     * Initiate a new pathway instance for a patient
     */
    public PathwayInstance initiatePathway(ClinicalPathway pathway, String patientId,
                                           Map<String, Object> initialPatientData) {
        // Validate pathway
        if (!pathway.validate()) {
            throw new IllegalArgumentException("Invalid pathway: " + pathway.getPathwayId());
        }

        // Check inclusion/exclusion criteria
        if (!pathway.meetsInclusionCriteria(initialPatientData)) {
            throw new IllegalStateException("Patient does not meet inclusion criteria for pathway: " +
                                           pathway.getPathwayName());
        }

        if (pathway.meetsExclusionCriteria(initialPatientData)) {
            throw new IllegalStateException("Patient meets exclusion criteria for pathway: " +
                                           pathway.getPathwayName());
        }

        // Create pathway instance
        PathwayInstance instance = new PathwayInstance(
            patientId,
            pathway.getPathwayId(),
            pathway.getPathwayName()
        );

        instance.setPatientData(new HashMap<>(initialPatientData));
        instance.setCurrentStepId(pathway.getInitialStepId());
        instance.setStatus(PathwayInstance.InstanceStatus.INITIATED);

        return instance;
    }

    /**
     * Execute the next step in the pathway
     */
    public PathwayInstance.StepExecution executeStep(ClinicalPathway pathway,
                                                      PathwayInstance instance,
                                                      String stepId) {
        // Find the step
        PathwayStep step = pathway.findStep(stepId);
        if (step == null) {
            throw new IllegalArgumentException("Step not found: " + stepId);
        }

        // Check entry conditions
        if (!step.canEnter(instance.getPatientData())) {
            String reason = "Entry conditions not met for step: " + step.getStepName();
            instance.recordDeviation(
                PathwayInstance.Deviation.DeviationType.WRONG_SEQUENCE,
                stepId,
                reason,
                PathwayInstance.Deviation.Severity.MODERATE
            );
            throw new IllegalStateException(reason);
        }

        // Start step execution
        PathwayInstance.StepExecution execution = instance.startStep(stepId, step.getStepName());

        // Record step initiation
        instance.recordAction("Started step: " + step.getStepName());

        return execution;
    }

    /**
     * Complete a step and determine next step
     */
    public String completeStep(ClinicalPathway pathway,
                               PathwayInstance instance,
                               PathwayInstance.StepExecution execution,
                               Map<String, Object> stepResults) {
        PathwayStep step = pathway.findStep(execution.getStepId());
        if (step == null) {
            throw new IllegalArgumentException("Step not found: " + execution.getStepId());
        }

        // Update patient data with step results
        if (stepResults != null) {
            for (Map.Entry<String, Object> entry : stepResults.entrySet()) {
                instance.updatePatientData(entry.getKey(), entry.getValue());
            }
        }

        // Check exit conditions
        if (!step.canExit(instance.getPatientData())) {
            String reason = "Exit conditions not met for step: " + step.getStepName();
            instance.recordDeviation(
                PathwayInstance.Deviation.DeviationType.MISSING_DATA,
                step.getStepId(),
                reason,
                PathwayInstance.Deviation.Severity.MODERATE
            );
            // Allow progression anyway but record deviation
        }

        // Check if completed on time
        boolean onTime = checkStepTiming(step, execution);

        // Complete the step
        instance.completeStep(execution, onTime);

        // Record quality measures if this step has them
        if (step.isCoreQualityMeasure()) {
            instance.recordQualityMeasure(step.getStepName(), true);
        }

        // Determine next step
        String nextStepId = determineNextStep(pathway, step, instance.getPatientData());

        // Update instance current step
        if (nextStepId != null) {
            instance.setCurrentStepId(nextStepId);
        } else {
            // No next step means pathway is complete
            instance.setStatus(PathwayInstance.InstanceStatus.COMPLETED);
        }

        instance.recordAction("Completed step: " + step.getStepName());

        return nextStepId;
    }

    /**
     * Determine the next step based on branching logic
     */
    private String determineNextStep(ClinicalPathway pathway, PathwayStep currentStep,
                                     Map<String, Object> patientData) {
        // Check if step has custom transitions
        if (currentStep.getTransitions() != null && !currentStep.getTransitions().isEmpty()) {
            return currentStep.determineNextStep(patientData);
        }

        // Use pathway-level decision points
        if (pathway.getDecisionPoints() != null) {
            for (Map.Entry<String, String> decision : pathway.getDecisionPoints().entrySet()) {
                String key = decision.getKey();
                if (key.startsWith(currentStep.getStepId() + ":")) {
                    return decision.getValue();
                }
            }
        }

        // Default: linear progression to next step in sequence
        List<PathwayStep> steps = pathway.getSteps();
        int currentIndex = -1;
        for (int i = 0; i < steps.size(); i++) {
            if (steps.get(i).getStepId().equals(currentStep.getStepId())) {
                currentIndex = i;
                break;
            }
        }

        if (currentIndex >= 0 && currentIndex < steps.size() - 1) {
            return steps.get(currentIndex + 1).getStepId();
        }

        return null; // No next step - pathway complete
    }

    /**
     * Check if step was completed within expected timeframe
     */
    private boolean checkStepTiming(PathwayStep step, PathwayInstance.StepExecution execution) {
        if (step.getExpectedDurationMinutes() == null || execution.getStartTime() == null) {
            return true; // No timing requirement
        }

        LocalDateTime now = LocalDateTime.now();
        long actualDuration = Duration.between(execution.getStartTime(), now).toMinutes();

        return actualDuration <= step.getExpectedDurationMinutes();
    }

    /**
     * Detect and record deviations in pathway execution
     */
    public List<PathwayInstance.Deviation> detectDeviations(ClinicalPathway pathway,
                                                             PathwayInstance instance) {
        List<PathwayInstance.Deviation> newDeviations = new ArrayList<>();

        // Check for time-critical steps that are overdue
        PathwayStep currentStep = pathway.findStep(instance.getCurrentStepId());
        if (currentStep != null && currentStep.isTimeCritical()) {
            if (currentStep.getMaxDurationMinutes() != null) {
                // Check if current step has exceeded max duration
                PathwayInstance.StepExecution currentExecution = findCurrentExecution(instance);
                if (currentExecution != null) {
                    long duration = Duration.between(
                        currentExecution.getStartTime(),
                        LocalDateTime.now()
                    ).toMinutes();

                    if (duration > currentStep.getMaxDurationMinutes()) {
                        PathwayInstance.Deviation deviation = new PathwayInstance.Deviation(
                            PathwayInstance.Deviation.DeviationType.TIME_EXCEEDED,
                            currentStep.getStepId(),
                            String.format("Step exceeded max duration: %d min (max: %d min)",
                                         duration, currentStep.getMaxDurationMinutes())
                        );
                        deviation.setSeverity(PathwayInstance.Deviation.Severity.HIGH);
                        newDeviations.add(deviation);
                        instance.getDeviations().add(deviation);
                    }
                }
            }
        }

        // Check overall pathway timing
        if (pathway.getMaxDurationMinutes() != null) {
            long totalDuration = Duration.between(
                instance.getStartTime(),
                LocalDateTime.now()
            ).toMinutes();

            if (totalDuration > pathway.getMaxDurationMinutes()) {
                PathwayInstance.Deviation deviation = new PathwayInstance.Deviation(
                    PathwayInstance.Deviation.DeviationType.TIME_EXCEEDED,
                    "PATHWAY",
                    String.format("Pathway exceeded max duration: %d min (max: %d min)",
                                 totalDuration, pathway.getMaxDurationMinutes())
                );
                deviation.setSeverity(PathwayInstance.Deviation.Severity.CRITICAL);
                newDeviations.add(deviation);
                instance.getDeviations().add(deviation);
            }
        }

        // Update deviation counts
        if (!newDeviations.isEmpty()) {
            instance.setHasDeviations(true);
            instance.setDeviationCount(instance.getDeviations().size());
        }

        return newDeviations;
    }

    /**
     * Find the currently executing step
     */
    private PathwayInstance.StepExecution findCurrentExecution(PathwayInstance instance) {
        // Check if there's a step in progress
        if (instance.getCompletedSteps() != null) {
            for (PathwayInstance.StepExecution exec : instance.getCompletedSteps()) {
                if (!exec.isCompleted()) {
                    return exec;
                }
            }
        }
        return null;
    }

    /**
     * Skip a step with clinical justification
     */
    public void skipStep(PathwayInstance instance, String stepId, String reason,
                        String clinicalJustification) {
        instance.recordDeviation(
            PathwayInstance.Deviation.DeviationType.STEP_SKIPPED,
            stepId,
            reason,
            PathwayInstance.Deviation.Severity.MODERATE
        );

        // Update the last deviation with clinical justification
        if (instance.getDeviations() != null && !instance.getDeviations().isEmpty()) {
            PathwayInstance.Deviation lastDeviation =
                instance.getDeviations().get(instance.getDeviations().size() - 1);
            lastDeviation.setClinicalJustification(clinicalJustification);
        }

        instance.recordAction("Skipped step: " + stepId + " - Reason: " + reason);
    }

    /**
     * Suspend pathway execution
     */
    public void suspendPathway(PathwayInstance instance, String reason) {
        instance.setStatus(PathwayInstance.InstanceStatus.SUSPENDED);
        instance.setOutcomeNotes(reason);
        instance.recordAction("Pathway suspended: " + reason);
    }

    /**
     * Resume suspended pathway
     */
    public void resumePathway(PathwayInstance instance) {
        if (instance.getStatus() != PathwayInstance.InstanceStatus.SUSPENDED) {
            throw new IllegalStateException("Cannot resume pathway that is not suspended");
        }

        instance.setStatus(PathwayInstance.InstanceStatus.IN_PROGRESS);
        instance.recordAction("Pathway resumed");
    }

    /**
     * Discontinue pathway with reason
     */
    public void discontinuePathway(PathwayInstance instance, String reason,
                                   String discontinuedBy) {
        instance.discontinue(reason);
        instance.setCompletedBy(discontinuedBy);
        instance.recordAction("Pathway discontinued: " + reason);
    }

    /**
     * Complete pathway successfully
     */
    public void completePathway(PathwayInstance instance, String outcome,
                               String completedBy) {
        instance.complete(outcome);
        instance.setCompletedBy(completedBy);
        instance.recordAction("Pathway completed with outcome: " + outcome);
    }

    /**
     * Generate pathway execution summary
     */
    public PathwaySummary generateSummary(PathwayInstance instance) {
        PathwaySummary summary = new PathwaySummary();
        summary.setInstanceId(instance.getInstanceId());
        summary.setPatientId(instance.getPatientId());
        summary.setPathwayName(instance.getPathwayName());
        summary.setStatus(instance.getStatus());
        summary.setStartTime(instance.getStartTime());
        summary.setEndTime(instance.getEndTime());
        summary.setTotalDurationMinutes(instance.getTotalDurationMinutes());

        // Calculate statistics
        summary.setTotalStepsCompleted(
            instance.getCompletedSteps() != null ? instance.getCompletedSteps().size() : 0
        );
        summary.setStepsCompletedOnTime(
            instance.getCompletedSteps() != null ?
            (int) instance.getCompletedSteps().stream().filter(s -> s.isOnTime()).count() : 0
        );
        summary.setTotalDeviations(instance.getDeviationCount());
        summary.setAdherenceScore(instance.getAdherenceScore());
        summary.setFinalOutcome(instance.getFinalOutcome());

        // Quality measures
        if (instance.getQualityMeasuresMet() != null) {
            long metCount = instance.getQualityMeasuresMet().values().stream()
                .filter(Boolean::booleanValue)
                .count();
            summary.setQualityMeasuresMet(
                (int) metCount,
                instance.getQualityMeasuresMet().size()
            );
        }

        return summary;
    }

    /**
     * Summary report of pathway execution
     */
    public static class PathwaySummary {
        private String instanceId;
        private String patientId;
        private String pathwayName;
        private PathwayInstance.InstanceStatus status;
        private LocalDateTime startTime;
        private LocalDateTime endTime;
        private Long totalDurationMinutes;
        private int totalStepsCompleted;
        private int stepsCompletedOnTime;
        private int totalDeviations;
        private double adherenceScore;
        private String finalOutcome;
        private int qualityMeasuresMet;
        private int totalQualityMeasures;

        public void setQualityMeasuresMet(int met, int total) {
            this.qualityMeasuresMet = met;
            this.totalQualityMeasures = total;
        }

        public double getQualityComplianceRate() {
            if (totalQualityMeasures == 0) return 0.0;
            return (double) qualityMeasuresMet / totalQualityMeasures * 100.0;
        }

        // Getters and setters
        public String getInstanceId() { return instanceId; }
        public void setInstanceId(String instanceId) { this.instanceId = instanceId; }
        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }
        public String getPathwayName() { return pathwayName; }
        public void setPathwayName(String pathwayName) { this.pathwayName = pathwayName; }
        public PathwayInstance.InstanceStatus getStatus() { return status; }
        public void setStatus(PathwayInstance.InstanceStatus status) { this.status = status; }
        public LocalDateTime getStartTime() { return startTime; }
        public void setStartTime(LocalDateTime startTime) { this.startTime = startTime; }
        public LocalDateTime getEndTime() { return endTime; }
        public void setEndTime(LocalDateTime endTime) { this.endTime = endTime; }
        public Long getTotalDurationMinutes() { return totalDurationMinutes; }
        public void setTotalDurationMinutes(Long totalDurationMinutes) { this.totalDurationMinutes = totalDurationMinutes; }
        public int getTotalStepsCompleted() { return totalStepsCompleted; }
        public void setTotalStepsCompleted(int totalStepsCompleted) { this.totalStepsCompleted = totalStepsCompleted; }
        public int getStepsCompletedOnTime() { return stepsCompletedOnTime; }
        public void setStepsCompletedOnTime(int stepsCompletedOnTime) { this.stepsCompletedOnTime = stepsCompletedOnTime; }
        public int getTotalDeviations() { return totalDeviations; }
        public void setTotalDeviations(int totalDeviations) { this.totalDeviations = totalDeviations; }
        public double getAdherenceScore() { return adherenceScore; }
        public void setAdherenceScore(double adherenceScore) { this.adherenceScore = adherenceScore; }
        public String getFinalOutcome() { return finalOutcome; }
        public void setFinalOutcome(String finalOutcome) { this.finalOutcome = finalOutcome; }
        public int getQualityMeasuresMet() { return qualityMeasuresMet; }
        public void setQualityMeasuresMet(int qualityMeasuresMet) { this.qualityMeasuresMet = qualityMeasuresMet; }
        public int getTotalQualityMeasures() { return totalQualityMeasures; }
        public void setTotalQualityMeasures(int totalQualityMeasures) { this.totalQualityMeasures = totalQualityMeasures; }

        @Override
        public String toString() {
            return String.format(
                "PathwaySummary{pathway='%s', status=%s, steps=%d, onTime=%d, deviations=%d, adherence=%.2f, quality=%.1f%%}",
                pathwayName, status, totalStepsCompleted, stepsCompletedOnTime,
                totalDeviations, adherenceScore, getQualityComplianceRate()
            );
        }
    }
}
