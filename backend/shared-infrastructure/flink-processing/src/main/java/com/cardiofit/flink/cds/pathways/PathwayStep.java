package com.cardiofit.flink.cds.pathways;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Phase 8 Module 3 - Clinical Pathways Engine
 *
 * Individual step within a clinical pathway.
 * Each step represents a clinical action, assessment, or decision point.
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class PathwayStep implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core Identification
    private String stepId;
    private String stepName;
    private int stepOrder;                  // Sequence number in pathway
    private StepType stepType;

    // Step Content
    private String description;
    private String clinicalRationale;       // Why this step is necessary
    private List<String> instructions;      // Detailed clinical instructions
    private List<String> requiredActions;   // Actions that must be completed

    // Time Constraints
    private Integer expectedDurationMinutes; // Expected time to complete step
    private Integer maxDurationMinutes;      // Maximum allowable time
    private boolean isTimeCritical;          // True if delays impact outcomes

    // Branching Logic
    private List<Condition> entryConditions;    // Conditions to enter this step
    private List<Condition> exitConditions;     // Conditions to exit this step
    private Map<String, String> transitions;    // Condition -> next step ID

    // Clinical Data Requirements
    private List<String> requiredVitals;        // Which vitals needed (e.g., "BP", "HR")
    private List<String> requiredLabs;          // Which labs needed (e.g., "Troponin", "Lactate")
    private List<String> requiredAssessments;   // Clinical assessments (e.g., "ECG", "Neuro exam")

    // Medications and Interventions
    private List<MedicationOrder> medications;  // Medications to administer
    private List<String> procedures;            // Procedures to perform
    private List<String> consultations;         // Specialist consultations needed

    // Decision Support
    private List<String> clinicalAlerts;        // Alerts to display
    private List<String> safeguards;            // Safety checks before proceeding
    private String decisionCriteria;            // Criteria for branching decisions

    // Documentation Requirements
    private List<String> requiredDocumentation; // What must be documented
    private boolean requiresPhysicianApproval;  // Needs MD sign-off

    // Quality and Compliance
    private String evidenceLevel;               // Evidence level for this step
    private List<String> qualityMeasures;       // Quality metrics tracked
    private boolean isCoreQualityMeasure;       // Part of core quality bundle

    /**
     * Step types in clinical pathways
     */
    public enum StepType {
        ASSESSMENT,         // Clinical assessment or evaluation
        DIAGNOSTIC,         // Diagnostic test or procedure
        THERAPEUTIC,        // Treatment or intervention
        MEDICATION,         // Medication administration
        MONITORING,         // Patient monitoring
        DECISION_POINT,     // Branching decision based on data
        CONSULTATION,       // Specialist consultation
        PATIENT_EDUCATION,  // Patient/family education
        DISPOSITION,        // Discharge or transfer planning
        DOCUMENTATION       // Required documentation
    }

    /**
     * Simple medication order for pathway steps
     */
    public static class MedicationOrder implements Serializable {
        private static final long serialVersionUID = 1L;

        private String medicationName;
        private String dose;
        private String route;           // "IV", "PO", "IM", etc.
        private String frequency;
        private boolean isStat;         // Immediate administration
        private String indication;

        public MedicationOrder(String medicationName, String dose, String route) {
            this.medicationName = medicationName;
            this.dose = dose;
            this.route = route;
        }

        // Getters and setters
        public String getMedicationName() { return medicationName; }
        public void setMedicationName(String medicationName) { this.medicationName = medicationName; }
        public String getDose() { return dose; }
        public void setDose(String dose) { this.dose = dose; }
        public String getRoute() { return route; }
        public void setRoute(String route) { this.route = route; }
        public String getFrequency() { return frequency; }
        public void setFrequency(String frequency) { this.frequency = frequency; }
        public boolean isStat() { return isStat; }
        public void setStat(boolean stat) { isStat = stat; }
        public String getIndication() { return indication; }
        public void setIndication(String indication) { this.indication = indication; }

        @Override
        public String toString() {
            return String.format("%s %s %s%s", medicationName, dose, route, isStat ? " STAT" : "");
        }
    }

    /**
     * Condition for pathway branching
     */
    public static class Condition implements Serializable {
        private static final long serialVersionUID = 1L;

        private String conditionId;
        private String description;
        private ConditionType conditionType;
        private String parameter;       // What to check (e.g., "troponin", "BP_systolic")
        private String operator;        // ">" "<" "=" ">=" "<=" "BETWEEN" "IN"
        private Object value;           // Comparison value
        private Object secondValue;     // For BETWEEN operator

        public enum ConditionType {
            LAB_VALUE,          // Laboratory result condition
            VITAL_SIGN,         // Vital sign condition
            CLINICAL_FINDING,   // Clinical assessment finding
            TIME_ELAPSED,       // Time-based condition
            MEDICATION_GIVEN,   // Medication administration status
            PROCEDURE_DONE,     // Procedure completion status
            CUSTOM              // Custom business logic
        }

        public Condition() {}

        public Condition(ConditionType type, String parameter, String operator, Object value) {
            this.conditionType = type;
            this.parameter = parameter;
            this.operator = operator;
            this.value = value;
        }

        /**
         * Evaluate this condition against patient data
         */
        public boolean evaluate(Map<String, Object> patientData) {
            if (parameter == null || operator == null || value == null) {
                return false;
            }

            Object actualValue = patientData.get(parameter);
            if (actualValue == null) {
                return false;
            }

            try {
                switch (operator) {
                    case ">":
                        return compareNumeric(actualValue, value) > 0;
                    case "<":
                        return compareNumeric(actualValue, value) < 0;
                    case ">=":
                        return compareNumeric(actualValue, value) >= 0;
                    case "<=":
                        return compareNumeric(actualValue, value) <= 0;
                    case "=":
                    case "==":
                        return actualValue.equals(value);
                    case "!=":
                        return !actualValue.equals(value);
                    case "BETWEEN":
                        if (secondValue == null) return false;
                        return compareNumeric(actualValue, value) >= 0 &&
                               compareNumeric(actualValue, secondValue) <= 0;
                    default:
                        return false;
                }
            } catch (Exception e) {
                return false;
            }
        }

        private int compareNumeric(Object a, Object b) {
            double aVal = ((Number) a).doubleValue();
            double bVal = ((Number) b).doubleValue();
            return Double.compare(aVal, bVal);
        }

        // Getters and setters
        public String getConditionId() { return conditionId; }
        public void setConditionId(String conditionId) { this.conditionId = conditionId; }
        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
        public ConditionType getConditionType() { return conditionType; }
        public void setConditionType(ConditionType conditionType) { this.conditionType = conditionType; }
        public String getParameter() { return parameter; }
        public void setParameter(String parameter) { this.parameter = parameter; }
        public String getOperator() { return operator; }
        public void setOperator(String operator) { this.operator = operator; }
        public Object getValue() { return value; }
        public void setValue(Object value) { this.value = value; }
        public Object getSecondValue() { return secondValue; }
        public void setSecondValue(Object secondValue) { this.secondValue = secondValue; }

        @Override
        public String toString() {
            return String.format("Condition{%s %s %s}", parameter, operator, value);
        }
    }

    // Constructors
    public PathwayStep() {
        this.instructions = new ArrayList<>();
        this.requiredActions = new ArrayList<>();
        this.entryConditions = new ArrayList<>();
        this.exitConditions = new ArrayList<>();
        this.transitions = new HashMap<>();
        this.requiredVitals = new ArrayList<>();
        this.requiredLabs = new ArrayList<>();
        this.requiredAssessments = new ArrayList<>();
        this.medications = new ArrayList<>();
        this.procedures = new ArrayList<>();
        this.consultations = new ArrayList<>();
        this.clinicalAlerts = new ArrayList<>();
        this.safeguards = new ArrayList<>();
        this.requiredDocumentation = new ArrayList<>();
        this.qualityMeasures = new ArrayList<>();
    }

    public PathwayStep(String stepId, String stepName, StepType stepType) {
        this();
        this.stepId = stepId;
        this.stepName = stepName;
        this.stepType = stepType;
    }

    /**
     * Check if all entry conditions are met
     */
    public boolean canEnter(Map<String, Object> patientData) {
        if (entryConditions == null || entryConditions.isEmpty()) {
            return true; // No entry conditions means step can always be entered
        }

        // All entry conditions must be met
        for (Condition condition : entryConditions) {
            if (!condition.evaluate(patientData)) {
                return false;
            }
        }
        return true;
    }

    /**
     * Check if all exit conditions are met
     */
    public boolean canExit(Map<String, Object> patientData) {
        if (exitConditions == null || exitConditions.isEmpty()) {
            return true; // No exit conditions means step can always be exited
        }

        // All exit conditions must be met
        for (Condition condition : exitConditions) {
            if (!condition.evaluate(patientData)) {
                return false;
            }
        }
        return true;
    }

    /**
     * Get the next step ID based on patient data
     */
    public String determineNextStep(Map<String, Object> patientData) {
        if (transitions == null || transitions.isEmpty()) {
            return null; // No branching - linear progression
        }

        // Evaluate each transition condition
        for (Map.Entry<String, String> transition : transitions.entrySet()) {
            String conditionKey = transition.getKey();
            String nextStepId = transition.getValue();

            // Simple condition evaluation (in production, would use Condition objects)
            // For now, return first match
            return nextStepId;
        }

        return null;
    }

    /**
     * Add a medication order to this step
     */
    public void addMedication(String name, String dose, String route, boolean isStat) {
        if (this.medications == null) {
            this.medications = new ArrayList<>();
        }
        MedicationOrder med = new MedicationOrder(name, dose, route);
        med.setStat(isStat);
        this.medications.add(med);
    }

    // Getters and Setters
    public String getStepId() {
        return stepId;
    }

    public void setStepId(String stepId) {
        this.stepId = stepId;
    }

    public String getStepName() {
        return stepName;
    }

    public void setStepName(String stepName) {
        this.stepName = stepName;
    }

    public int getStepOrder() {
        return stepOrder;
    }

    public void setStepOrder(int stepOrder) {
        this.stepOrder = stepOrder;
    }

    public StepType getStepType() {
        return stepType;
    }

    public void setStepType(StepType stepType) {
        this.stepType = stepType;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public String getClinicalRationale() {
        return clinicalRationale;
    }

    public void setClinicalRationale(String clinicalRationale) {
        this.clinicalRationale = clinicalRationale;
    }

    public List<String> getInstructions() {
        return instructions;
    }

    public void setInstructions(List<String> instructions) {
        this.instructions = instructions;
    }

    public void addInstruction(String instruction) {
        if (this.instructions == null) {
            this.instructions = new ArrayList<>();
        }
        this.instructions.add(instruction);
    }

    public List<String> getRequiredActions() {
        return requiredActions;
    }

    public void setRequiredActions(List<String> requiredActions) {
        this.requiredActions = requiredActions;
    }

    public void addRequiredAction(String action) {
        if (this.requiredActions == null) {
            this.requiredActions = new ArrayList<>();
        }
        this.requiredActions.add(action);
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

    public boolean isTimeCritical() {
        return isTimeCritical;
    }

    public void setTimeCritical(boolean timeCritical) {
        isTimeCritical = timeCritical;
    }

    public List<Condition> getEntryConditions() {
        return entryConditions;
    }

    public void setEntryConditions(List<Condition> entryConditions) {
        this.entryConditions = entryConditions;
    }

    public void addEntryCondition(Condition condition) {
        if (this.entryConditions == null) {
            this.entryConditions = new ArrayList<>();
        }
        this.entryConditions.add(condition);
    }

    public List<Condition> getExitConditions() {
        return exitConditions;
    }

    public void setExitConditions(List<Condition> exitConditions) {
        this.exitConditions = exitConditions;
    }

    public void addExitCondition(Condition condition) {
        if (this.exitConditions == null) {
            this.exitConditions = new ArrayList<>();
        }
        this.exitConditions.add(condition);
    }

    public Map<String, String> getTransitions() {
        return transitions;
    }

    public void setTransitions(Map<String, String> transitions) {
        this.transitions = transitions;
    }

    public void addTransition(String condition, String nextStepId) {
        if (this.transitions == null) {
            this.transitions = new HashMap<>();
        }
        this.transitions.put(condition, nextStepId);
    }

    public List<String> getRequiredVitals() {
        return requiredVitals;
    }

    public void setRequiredVitals(List<String> requiredVitals) {
        this.requiredVitals = requiredVitals;
    }

    public List<String> getRequiredLabs() {
        return requiredLabs;
    }

    public void setRequiredLabs(List<String> requiredLabs) {
        this.requiredLabs = requiredLabs;
    }

    public List<String> getRequiredAssessments() {
        return requiredAssessments;
    }

    public void setRequiredAssessments(List<String> requiredAssessments) {
        this.requiredAssessments = requiredAssessments;
    }

    public List<MedicationOrder> getMedications() {
        return medications;
    }

    public void setMedications(List<MedicationOrder> medications) {
        this.medications = medications;
    }

    public List<String> getProcedures() {
        return procedures;
    }

    public void setProcedures(List<String> procedures) {
        this.procedures = procedures;
    }

    public List<String> getConsultations() {
        return consultations;
    }

    public void setConsultations(List<String> consultations) {
        this.consultations = consultations;
    }

    public List<String> getClinicalAlerts() {
        return clinicalAlerts;
    }

    public void setClinicalAlerts(List<String> clinicalAlerts) {
        this.clinicalAlerts = clinicalAlerts;
    }

    public void addClinicalAlert(String alert) {
        if (this.clinicalAlerts == null) {
            this.clinicalAlerts = new ArrayList<>();
        }
        this.clinicalAlerts.add(alert);
    }

    public List<String> getSafeguards() {
        return safeguards;
    }

    public void setSafeguards(List<String> safeguards) {
        this.safeguards = safeguards;
    }

    public void addSafeguard(String safeguard) {
        if (this.safeguards == null) {
            this.safeguards = new ArrayList<>();
        }
        this.safeguards.add(safeguard);
    }

    public String getDecisionCriteria() {
        return decisionCriteria;
    }

    public void setDecisionCriteria(String decisionCriteria) {
        this.decisionCriteria = decisionCriteria;
    }

    public List<String> getRequiredDocumentation() {
        return requiredDocumentation;
    }

    public void setRequiredDocumentation(List<String> requiredDocumentation) {
        this.requiredDocumentation = requiredDocumentation;
    }

    public boolean isRequiresPhysicianApproval() {
        return requiresPhysicianApproval;
    }

    public void setRequiresPhysicianApproval(boolean requiresPhysicianApproval) {
        this.requiresPhysicianApproval = requiresPhysicianApproval;
    }

    public String getEvidenceLevel() {
        return evidenceLevel;
    }

    public void setEvidenceLevel(String evidenceLevel) {
        this.evidenceLevel = evidenceLevel;
    }

    public List<String> getQualityMeasures() {
        return qualityMeasures;
    }

    public void setQualityMeasures(List<String> qualityMeasures) {
        this.qualityMeasures = qualityMeasures;
    }

    public boolean isCoreQualityMeasure() {
        return isCoreQualityMeasure;
    }

    public void setCoreQualityMeasure(boolean coreQualityMeasure) {
        isCoreQualityMeasure = coreQualityMeasure;
    }

    @Override
    public String toString() {
        return String.format("PathwayStep{id='%s', name='%s', type=%s, order=%d, timeCritical=%s}",
            stepId, stepName, stepType, stepOrder, isTimeCritical);
    }
}
