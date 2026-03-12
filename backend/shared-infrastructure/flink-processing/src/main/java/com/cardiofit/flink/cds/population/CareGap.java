package com.cardiofit.flink.cds.population;

import java.io.Serializable;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;

/**
 * Phase 8 Module 4 - Population Health Module
 *
 * Represents a gap in care for a patient or cohort.
 * Care gaps identify missing preventive services, overdue screenings,
 * uncontrolled chronic conditions, or medication non-adherence.
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class CareGap implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core Identification
    private String gapId;
    private String patientId;
    private String cohortId;
    private GapType gapType;
    private GapCategory category;

    // Gap Details
    private String gapName;
    private String description;
    private String clinicalReason;              // Why this gap matters
    private String recommendedAction;           // What should be done

    // Severity and Priority
    private GapSeverity severity;
    private int priority;                       // 1-10, 10 = highest
    private boolean isUrgent;

    // Timeline
    private LocalDate dueDate;                  // When action is due
    private LocalDate overdueDate;              // When gap became overdue
    private int daysOverdue;
    private LocalDateTime identifiedAt;
    private LocalDateTime closedAt;

    // Clinical Context
    private String relatedCondition;            // ICD-10 code
    private String relatedMedication;           // Drug name or class
    private String relatedProcedure;            // CPT code
    private String relatedLab;                  // LOINC code

    // Guidelines and Evidence
    private String guidelineReference;          // e.g., "ADA 2023 Standards"
    private String qualityMeasureId;            // HEDIS/CMS measure ID
    private String evidenceLevel;               // A, B, C

    // Status Tracking
    private GapStatus status;
    private String closureReason;
    private List<InterventionAttempt> interventions;

    // Quality Impact
    private boolean impactsQualityMeasure;
    private double financialImpact;             // Potential cost/savings

    /**
     * Care gap types
     */
    public enum GapType {
        PREVENTIVE_SCREENING,       // Missing preventive services
        CHRONIC_DISEASE_MONITORING, // Overdue chronic disease checks
        MEDICATION_ADHERENCE,       // Non-adherence to medications
        IMMUNIZATION,               // Missing vaccines
        DIAGNOSTIC_TESTING,         // Overdue diagnostic tests
        FOLLOW_UP_APPOINTMENT,      // Missing follow-up
        SPECIALIST_REFERRAL,        // Needed but not done
        CARE_COORDINATION,          // Coordination gaps
        PATIENT_EDUCATION,          // Education needs
        LIFESTYLE_MODIFICATION      // Diet, exercise, smoking cessation
    }

    /**
     * Care gap categories
     */
    public enum GapCategory {
        PREVENTIVE,                 // Preventive care
        CHRONIC_MANAGEMENT,         // Chronic disease management
        ACUTE_CARE,                 // Acute care needs
        MEDICATION,                 // Medication-related
        BEHAVIORAL_HEALTH,          // Mental health/substance abuse
        SOCIAL_DETERMINANTS         // SDOH-related gaps
    }

    /**
     * Gap severity levels
     */
    public enum GapSeverity {
        LOW(1, "Routine"),
        MODERATE(2, "Important"),
        HIGH(3, "Urgent"),
        CRITICAL(4, "Immediate action required");

        private final int level;
        private final String description;

        GapSeverity(int level, String description) {
            this.level = level;
            this.description = description;
        }

        public int getLevel() { return level; }
        public String getDescription() { return description; }
    }

    /**
     * Gap status
     */
    public enum GapStatus {
        IDENTIFIED,         // Newly identified
        NOTIFIED,           // Patient/provider notified
        INTERVENTION_SENT,  // Intervention attempted
        IN_PROGRESS,        // Being addressed
        CLOSED_COMPLETED,   // Gap closed - action taken
        CLOSED_INAPPROPRIATE,// Gap closed - not appropriate
        CLOSED_REFUSED,     // Gap closed - patient refused
        CLOSED_EXPIRED      // Gap closed - no longer relevant
    }

    /**
     * Intervention attempt record
     */
    public static class InterventionAttempt implements Serializable {
        private static final long serialVersionUID = 1L;

        private LocalDateTime attemptDate;
        private String interventionType;        // "Phone call", "Letter", "Portal message"
        private String outcome;
        private String notes;

        public InterventionAttempt() {}

        public InterventionAttempt(String interventionType, String outcome) {
            this.attemptDate = LocalDateTime.now();
            this.interventionType = interventionType;
            this.outcome = outcome;
        }

        // Getters and setters
        public LocalDateTime getAttemptDate() { return attemptDate; }
        public void setAttemptDate(LocalDateTime attemptDate) { this.attemptDate = attemptDate; }
        public String getInterventionType() { return interventionType; }
        public void setInterventionType(String interventionType) { this.interventionType = interventionType; }
        public String getOutcome() { return outcome; }
        public void setOutcome(String outcome) { this.outcome = outcome; }
        public String getNotes() { return notes; }
        public void setNotes(String notes) { this.notes = notes; }
    }

    // Constructors
    public CareGap() {
        this.gapId = generateGapId();
        this.identifiedAt = LocalDateTime.now();
        this.status = GapStatus.IDENTIFIED;
        this.interventions = new ArrayList<>();
    }

    public CareGap(String patientId, GapType gapType, String gapName) {
        this();
        this.patientId = patientId;
        this.gapType = gapType;
        this.gapName = gapName;
    }

    private String generateGapId() {
        return "GAP-" + System.currentTimeMillis();
    }

    /**
     * Calculate days overdue
     */
    public void calculateDaysOverdue() {
        if (dueDate != null && LocalDate.now().isAfter(dueDate)) {
            this.daysOverdue = (int) java.time.temporal.ChronoUnit.DAYS.between(dueDate, LocalDate.now());
            if (overdueDate == null) {
                this.overdueDate = dueDate.plusDays(1);
            }
        } else {
            this.daysOverdue = 0;
        }
    }

    /**
     * Add intervention attempt
     */
    public void addIntervention(String interventionType, String outcome) {
        InterventionAttempt attempt = new InterventionAttempt(interventionType, outcome);
        this.interventions.add(attempt);
        this.status = GapStatus.INTERVENTION_SENT;
    }

    /**
     * Close the care gap
     */
    public void closeGap(GapStatus closureStatus, String reason) {
        if (closureStatus != GapStatus.CLOSED_COMPLETED &&
            closureStatus != GapStatus.CLOSED_INAPPROPRIATE &&
            closureStatus != GapStatus.CLOSED_REFUSED &&
            closureStatus != GapStatus.CLOSED_EXPIRED) {
            throw new IllegalArgumentException("Invalid closure status: " + closureStatus);
        }

        this.status = closureStatus;
        this.closureReason = reason;
        this.closedAt = LocalDateTime.now();
    }

    /**
     * Check if gap is overdue
     */
    public boolean isOverdue() {
        return dueDate != null && LocalDate.now().isAfter(dueDate);
    }

    /**
     * Check if gap is open
     */
    public boolean isOpen() {
        return status != GapStatus.CLOSED_COMPLETED &&
               status != GapStatus.CLOSED_INAPPROPRIATE &&
               status != GapStatus.CLOSED_REFUSED &&
               status != GapStatus.CLOSED_EXPIRED;
    }

    // Getters and Setters
    public String getGapId() {
        return gapId;
    }

    public void setGapId(String gapId) {
        this.gapId = gapId;
    }

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getCohortId() {
        return cohortId;
    }

    public void setCohortId(String cohortId) {
        this.cohortId = cohortId;
    }

    public GapType getGapType() {
        return gapType;
    }

    public void setGapType(GapType gapType) {
        this.gapType = gapType;
    }

    public GapCategory getCategory() {
        return category;
    }

    public void setCategory(GapCategory category) {
        this.category = category;
    }

    public String getGapName() {
        return gapName;
    }

    public void setGapName(String gapName) {
        this.gapName = gapName;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public String getClinicalReason() {
        return clinicalReason;
    }

    public void setClinicalReason(String clinicalReason) {
        this.clinicalReason = clinicalReason;
    }

    public String getRecommendedAction() {
        return recommendedAction;
    }

    public void setRecommendedAction(String recommendedAction) {
        this.recommendedAction = recommendedAction;
    }

    public GapSeverity getSeverity() {
        return severity;
    }

    public void setSeverity(GapSeverity severity) {
        this.severity = severity;
    }

    public int getPriority() {
        return priority;
    }

    public void setPriority(int priority) {
        this.priority = Math.max(1, Math.min(10, priority)); // Clamp to 1-10
    }

    public boolean isUrgent() {
        return isUrgent;
    }

    public void setUrgent(boolean urgent) {
        isUrgent = urgent;
    }

    public LocalDate getDueDate() {
        return dueDate;
    }

    public void setDueDate(LocalDate dueDate) {
        this.dueDate = dueDate;
    }

    public LocalDate getOverdueDate() {
        return overdueDate;
    }

    public void setOverdueDate(LocalDate overdueDate) {
        this.overdueDate = overdueDate;
    }

    public int getDaysOverdue() {
        return daysOverdue;
    }

    public void setDaysOverdue(int daysOverdue) {
        this.daysOverdue = daysOverdue;
    }

    public LocalDateTime getIdentifiedAt() {
        return identifiedAt;
    }

    public void setIdentifiedAt(LocalDateTime identifiedAt) {
        this.identifiedAt = identifiedAt;
    }

    public LocalDateTime getClosedAt() {
        return closedAt;
    }

    public void setClosedAt(LocalDateTime closedAt) {
        this.closedAt = closedAt;
    }

    public String getRelatedCondition() {
        return relatedCondition;
    }

    public void setRelatedCondition(String relatedCondition) {
        this.relatedCondition = relatedCondition;
    }

    public String getRelatedMedication() {
        return relatedMedication;
    }

    public void setRelatedMedication(String relatedMedication) {
        this.relatedMedication = relatedMedication;
    }

    public String getRelatedProcedure() {
        return relatedProcedure;
    }

    public void setRelatedProcedure(String relatedProcedure) {
        this.relatedProcedure = relatedProcedure;
    }

    public String getRelatedLab() {
        return relatedLab;
    }

    public void setRelatedLab(String relatedLab) {
        this.relatedLab = relatedLab;
    }

    public String getGuidelineReference() {
        return guidelineReference;
    }

    public void setGuidelineReference(String guidelineReference) {
        this.guidelineReference = guidelineReference;
    }

    public String getQualityMeasureId() {
        return qualityMeasureId;
    }

    public void setQualityMeasureId(String qualityMeasureId) {
        this.qualityMeasureId = qualityMeasureId;
    }

    public String getEvidenceLevel() {
        return evidenceLevel;
    }

    public void setEvidenceLevel(String evidenceLevel) {
        this.evidenceLevel = evidenceLevel;
    }

    public GapStatus getStatus() {
        return status;
    }

    public void setStatus(GapStatus status) {
        this.status = status;
    }

    public String getClosureReason() {
        return closureReason;
    }

    public void setClosureReason(String closureReason) {
        this.closureReason = closureReason;
    }

    public List<InterventionAttempt> getInterventions() {
        return interventions;
    }

    public void setInterventions(List<InterventionAttempt> interventions) {
        this.interventions = interventions;
    }

    public boolean isImpactsQualityMeasure() {
        return impactsQualityMeasure;
    }

    public void setImpactsQualityMeasure(boolean impactsQualityMeasure) {
        this.impactsQualityMeasure = impactsQualityMeasure;
    }

    public double getFinancialImpact() {
        return financialImpact;
    }

    public void setFinancialImpact(double financialImpact) {
        this.financialImpact = financialImpact;
    }

    @Override
    public String toString() {
        return String.format("CareGap{id='%s', patient='%s', type=%s, severity=%s, daysOverdue=%d, status=%s}",
            gapId, patientId, gapType, severity, daysOverdue, status);
    }
}
