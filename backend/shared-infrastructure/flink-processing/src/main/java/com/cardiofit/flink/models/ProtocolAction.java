package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Protocol Action - Enhanced with Guideline Integration
 *
 * Individual actionable step within a clinical protocol, now linked to
 * evidence-based guidelines with full traceability to recommendations
 * and supporting citations.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 2.0
 * @since 2025-10-24
 */
public class ProtocolAction implements Serializable {
    private static final long serialVersionUID = 2L;

    // Core Action Properties
    @JsonProperty("action_id")
    private String actionId;

    @JsonProperty("action_type")
    private String actionType;  // MEDICATION, DIAGNOSTIC, PROCEDURE, CLINICAL_ASSESSMENT, CONSULTATION

    @JsonProperty("priority")
    private String priority;  // CRITICAL, HIGH, MEDIUM, LOW

    @JsonProperty("sequence_number")
    private int sequenceNumber;

    @JsonProperty("description")
    private String description;

    // Timing Properties
    @JsonProperty("timing")
    private ActionTiming timing;

    // Action-Specific Details
    @JsonProperty("medication")
    private MedicationDetails medication;

    @JsonProperty("diagnostic")
    private DiagnosticDetails diagnostic;

    @JsonProperty("procedure")
    private ProcedureDetails procedure;

    @JsonProperty("clinical_assessment")
    private ClinicalAssessmentDetails clinicalAssessment;

    @JsonProperty("consultation")
    private ConsultationDetails consultation;

    // PHASE 5 DAY 4: GUIDELINE INTEGRATION FIELDS

    /**
     * Reference to the guideline supporting this action
     * Example: "GUIDE-ACCAHA-STEMI-2023"
     */
    @JsonProperty("guideline_reference")
    private String guidelineReference;

    /**
     * Specific recommendation ID from the guideline
     * Example: "ACC-STEMI-2023-REC-003"
     */
    @JsonProperty("recommendation_id")
    private String recommendationId;

    /**
     * Complete evidence chain from action to citations
     */
    @JsonProperty("evidence_chain")
    private EvidenceChain evidenceChain;

    /**
     * Quality of evidence supporting this action (GRADE system)
     * HIGH, MODERATE, LOW, VERY_LOW
     */
    @JsonProperty("evidence_quality")
    private String evidenceQuality;

    /**
     * Strength of recommendation (GRADE system)
     * STRONG, WEAK, CONDITIONAL
     */
    @JsonProperty("recommendation_strength")
    private String recommendationStrength;

    /**
     * Class of recommendation (ACC/AHA system)
     * CLASS_I, CLASS_IIA, CLASS_IIB, CLASS_III
     */
    @JsonProperty("class_of_recommendation")
    private String classOfRecommendation;

    /**
     * Level of evidence (ACC/AHA system)
     * A (high quality), B-R (moderate randomized), B-NR (moderate non-randomized),
     * C-LD (limited data), C-EO (expert opinion)
     */
    @JsonProperty("level_of_evidence")
    private String levelOfEvidence;

    /**
     * Clinical rationale with evidence summary
     */
    @JsonProperty("clinical_rationale")
    private String clinicalRationale;

    /**
     * List of direct citation PMIDs supporting this action
     */
    @JsonProperty("citation_pmids")
    private List<String> citationPmids;

    // Contraindications and Safety
    @JsonProperty("contraindications")
    private List<String> contraindications;

    @JsonProperty("prerequisite_checks")
    private List<String> prerequisiteChecks;

    // Default constructor
    public ProtocolAction() {
        this.actionId = java.util.UUID.randomUUID().toString();
        this.citationPmids = new ArrayList<>();
        this.contraindications = new ArrayList<>();
        this.prerequisiteChecks = new ArrayList<>();
    }

    // Constructor with essential fields
    public ProtocolAction(String actionId, String actionType, int sequenceNumber) {
        this();
        this.actionId = actionId;
        this.actionType = actionType;
        this.sequenceNumber = sequenceNumber;
    }

    // Getters and Setters

    public String getActionId() { return actionId; }
    public void setActionId(String actionId) { this.actionId = actionId; }

    public String getActionType() { return actionType; }
    public void setActionType(String actionType) { this.actionType = actionType; }

    public String getPriority() { return priority; }
    public void setPriority(String priority) { this.priority = priority; }

    public int getSequenceNumber() { return sequenceNumber; }
    public void setSequenceNumber(int sequenceNumber) { this.sequenceNumber = sequenceNumber; }

    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }

    public ActionTiming getTiming() { return timing; }
    public void setTiming(ActionTiming timing) { this.timing = timing; }

    public MedicationDetails getMedication() { return medication; }
    public void setMedication(MedicationDetails medication) { this.medication = medication; }

    public DiagnosticDetails getDiagnostic() { return diagnostic; }
    public void setDiagnostic(DiagnosticDetails diagnostic) { this.diagnostic = diagnostic; }

    public ProcedureDetails getProcedure() { return procedure; }
    public void setProcedure(ProcedureDetails procedure) { this.procedure = procedure; }

    public ClinicalAssessmentDetails getClinicalAssessment() { return clinicalAssessment; }
    public void setClinicalAssessment(ClinicalAssessmentDetails clinicalAssessment) {
        this.clinicalAssessment = clinicalAssessment;
    }

    public ConsultationDetails getConsultation() { return consultation; }
    public void setConsultation(ConsultationDetails consultation) { this.consultation = consultation; }

    // Phase 5 Day 4 Getters and Setters

    public String getGuidelineReference() { return guidelineReference; }
    public void setGuidelineReference(String guidelineReference) { this.guidelineReference = guidelineReference; }

    public String getRecommendationId() { return recommendationId; }
    public void setRecommendationId(String recommendationId) { this.recommendationId = recommendationId; }

    public EvidenceChain getEvidenceChain() { return evidenceChain; }
    public void setEvidenceChain(EvidenceChain evidenceChain) { this.evidenceChain = evidenceChain; }

    public String getEvidenceQuality() { return evidenceQuality; }
    public void setEvidenceQuality(String evidenceQuality) { this.evidenceQuality = evidenceQuality; }

    public String getRecommendationStrength() { return recommendationStrength; }
    public void setRecommendationStrength(String recommendationStrength) {
        this.recommendationStrength = recommendationStrength;
    }

    public String getClassOfRecommendation() { return classOfRecommendation; }
    public void setClassOfRecommendation(String classOfRecommendation) {
        this.classOfRecommendation = classOfRecommendation;
    }

    public String getLevelOfEvidence() { return levelOfEvidence; }
    public void setLevelOfEvidence(String levelOfEvidence) { this.levelOfEvidence = levelOfEvidence; }

    public String getClinicalRationale() { return clinicalRationale; }
    public void setClinicalRationale(String clinicalRationale) { this.clinicalRationale = clinicalRationale; }

    public List<String> getCitationPmids() { return citationPmids; }
    public void setCitationPmids(List<String> citationPmids) { this.citationPmids = citationPmids; }

    public List<String> getContraindications() { return contraindications; }
    public void setContraindications(List<String> contraindications) { this.contraindications = contraindications; }

    public List<String> getPrerequisiteChecks() { return prerequisiteChecks; }
    public void setPrerequisiteChecks(List<String> prerequisiteChecks) {
        this.prerequisiteChecks = prerequisiteChecks;
    }

    // ================================================================
    // TEST COMPATIBILITY METHODS
    // ================================================================

    /**
     * Convenience setter for action type (test helper).
     * Delegates to setActionType() for backward compatibility.
     *
     * @param type The action type
     */
    public void setType(String type) {
        setActionType(type);
    }

    /**
     * Placeholder for medication selection result.
     * Tests may set this but it's not used in current implementation.
     */
    @JsonProperty("medication_selection")
    private Object medicationSelection;

    /**
     * Convenience setter for medication selection (test helper).
     *
     * @param selection The medication selection object
     */
    public void setMedicationSelection(Object selection) {
        this.medicationSelection = selection;
    }

    /**
     * Convenience getter for medication selection (test helper).
     *
     * @return The medication selection object
     */
    public Object getMedicationSelection() {
        return this.medicationSelection;
    }

    // Utility Methods

    /**
     * Check if action has guideline support
     */
    public boolean hasGuidelineSupport() {
        return guidelineReference != null && !guidelineReference.isEmpty();
    }

    /**
     * Check if action has high-quality evidence
     */
    public boolean hasHighQualityEvidence() {
        return "HIGH".equalsIgnoreCase(evidenceQuality);
    }

    /**
     * Check if action has strong recommendation
     */
    public boolean hasStrongRecommendation() {
        return "STRONG".equalsIgnoreCase(recommendationStrength) ||
               "CLASS_I".equalsIgnoreCase(classOfRecommendation);
    }

    /**
     * Check if action is time-critical
     */
    public boolean isTimeCritical() {
        return "CRITICAL".equalsIgnoreCase(priority) && timing != null &&
               timing.getMaxDelayMinutes() != null && timing.getMaxDelayMinutes() <= 60;
    }

    /**
     * Get quality badge for UI display
     * Returns emoji indicator of evidence quality
     */
    public String getQualityBadge() {
        if (evidenceChain != null && evidenceChain.isOutdated()) {
            return "⚠️ OUTDATED";
        }

        if ("HIGH".equalsIgnoreCase(evidenceQuality) && "STRONG".equalsIgnoreCase(recommendationStrength)) {
            return "🟢 STRONG";
        } else if ("MODERATE".equalsIgnoreCase(evidenceQuality)) {
            return "🟡 MODERATE";
        } else if ("LOW".equalsIgnoreCase(evidenceQuality) || "VERY_LOW".equalsIgnoreCase(evidenceQuality)) {
            return "🟠 WEAK";
        }

        return "⚪ UNGRADED";
    }

    /**
     * Get formatted evidence summary for display
     */
    public String getEvidenceSummary() {
        StringBuilder summary = new StringBuilder();

        if (recommendationId != null) {
            summary.append("Recommendation: ").append(recommendationId).append("\n");
        }

        if (classOfRecommendation != null && levelOfEvidence != null) {
            summary.append("Class ").append(classOfRecommendation)
                   .append(", Level ").append(levelOfEvidence).append("\n");
        }

        if (evidenceQuality != null && recommendationStrength != null) {
            summary.append("Evidence: ").append(evidenceQuality)
                   .append(", Strength: ").append(recommendationStrength).append("\n");
        }

        if (citationPmids != null && !citationPmids.isEmpty()) {
            summary.append("Citations: ").append(citationPmids.size())
                   .append(" studies (PMIDs: ").append(String.join(", ", citationPmids)).append(")");
        }

        return summary.toString();
    }

    @Override
    public String toString() {
        return "ProtocolAction{" +
            "actionId='" + actionId + '\'' +
            ", actionType='" + actionType + '\'' +
            ", priority='" + priority + '\'' +
            ", sequenceNumber=" + sequenceNumber +
            ", guidelineReference='" + guidelineReference + '\'' +
            ", recommendationId='" + recommendationId + '\'' +
            ", evidenceQuality='" + evidenceQuality + '\'' +
            ", recommendationStrength='" + recommendationStrength + '\'' +
            ", qualityBadge='" + getQualityBadge() + '\'' +
            '}';
    }

    // Nested classes for action-specific details

    public static class ActionTiming implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("window")
        private String window;  // IMMEDIATE, URGENT, ROUTINE

        @JsonProperty("max_delay_minutes")
        private Integer maxDelayMinutes;

        public String getWindow() { return window; }
        public void setWindow(String window) { this.window = window; }

        public Integer getMaxDelayMinutes() { return maxDelayMinutes; }
        public void setMaxDelayMinutes(Integer maxDelayMinutes) { this.maxDelayMinutes = maxDelayMinutes; }
    }

    public static class MedicationDetails implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("name")
        private String name;

        @JsonProperty("dose")
        private String dose;

        @JsonProperty("dose_unit")
        private String doseUnit;

        @JsonProperty("route")
        private String route;

        @JsonProperty("frequency")
        private String frequency;

        // Getters and setters
        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getDose() { return dose; }
        public void setDose(String dose) { this.dose = dose; }

        /**
         * Convenience getter for dosage (test helper).
         * Delegates to getDose() for backward compatibility.
         *
         * @return The dose value
         */
        public String getDosage() {
            return getDose();
        }

        public String getDoseUnit() { return doseUnit; }
        public void setDoseUnit(String doseUnit) { this.doseUnit = doseUnit; }

        public String getRoute() { return route; }
        public void setRoute(String route) { this.route = route; }

        public String getFrequency() { return frequency; }
        public void setFrequency(String frequency) { this.frequency = frequency; }
    }

    public static class DiagnosticDetails implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("test_name")
        private String testName;

        @JsonProperty("urgency")
        private String urgency;

        @JsonProperty("instructions")
        private String instructions;

        // Getters and setters
        public String getTestName() { return testName; }
        public void setTestName(String testName) { this.testName = testName; }

        public String getUrgency() { return urgency; }
        public void setUrgency(String urgency) { this.urgency = urgency; }

        public String getInstructions() { return instructions; }
        public void setInstructions(String instructions) { this.instructions = instructions; }
    }

    public static class ProcedureDetails implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("procedure_name")
        private String procedureName;

        @JsonProperty("procedure_code")
        private String procedureCode;

        @JsonProperty("urgency")
        private String urgency;

        // Getters and setters
        public String getProcedureName() { return procedureName; }
        public void setProcedureName(String procedureName) { this.procedureName = procedureName; }

        public String getProcedureCode() { return procedureCode; }
        public void setProcedureCode(String procedureCode) { this.procedureCode = procedureCode; }

        public String getUrgency() { return urgency; }
        public void setUrgency(String urgency) { this.urgency = urgency; }
    }

    public static class ClinicalAssessmentDetails implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("assessment_type")
        private String assessmentType;

        @JsonProperty("frequency")
        private String frequency;

        @JsonProperty("duration_hours")
        private Integer durationHours;

        // Getters and setters
        public String getAssessmentType() { return assessmentType; }
        public void setAssessmentType(String assessmentType) { this.assessmentType = assessmentType; }

        public String getFrequency() { return frequency; }
        public void setFrequency(String frequency) { this.frequency = frequency; }

        public Integer getDurationHours() { return durationHours; }
        public void setDurationHours(Integer durationHours) { this.durationHours = durationHours; }
    }

    public static class ConsultationDetails implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("specialty")
        private String specialty;

        @JsonProperty("urgency")
        private String urgency;

        @JsonProperty("reason")
        private String reason;

        // Getters and setters
        public String getSpecialty() { return specialty; }
        public void setSpecialty(String specialty) { this.specialty = specialty; }

        public String getUrgency() { return urgency; }
        public void setUrgency(String urgency) { this.urgency = urgency; }

        public String getReason() { return reason; }
        public void setReason(String reason) { this.reason = reason; }
    }
}
