package com.cardiofit.flink.models.diagnostics;

import lombok.Data;
import lombok.Builder;
import java.io.Serializable;
import java.util.*;

/**
 * Test Recommendation
 *
 * Represents a clinical recommendation to order a diagnostic test (lab or imaging).
 * Links to either LabTest or ImagingStudy definitions and provides clinical context,
 * priority, urgency, and evidence-based rationale for the recommendation. Used in
 * intelligent test ordering workflows in the CardioFit clinical decision support system.
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-23
 */
@Data
@Builder
public class TestRecommendation implements Serializable {
    private static final long serialVersionUID = 1L;

    // ================================================================
    // IDENTIFICATION
    // ================================================================
    private String recommendationId;         // Unique ID for this recommendation
    private String testId;                   // Links to LabTest.testId or ImagingStudy.studyId
    private String testName;                 // Human-readable test name
    private TestCategory testCategory;       // LAB, IMAGING
    private Long timestamp;                  // When recommendation was generated

    /**
     * Test Category Enumeration
     */
    public enum TestCategory {
        LAB,            // Laboratory test
        IMAGING,        // Imaging study
        PROCEDURE,      // Procedural diagnostic
        MONITORING      // Ongoing monitoring
    }

    // ================================================================
    // PRIORITY & URGENCY
    // ================================================================
    private Priority priority;               // Clinical priority level
    private Urgency urgency;                 // How quickly test needs to be done

    /**
     * Priority Levels
     * Based on clinical impact and decision-making urgency
     */
    public enum Priority {
        P0_CRITICAL,        // Life-threatening, immediate action required
        P1_URGENT,          // Urgent, significant clinical impact
        P2_IMPORTANT,       // Important for diagnosis/management
        P3_ROUTINE          // Routine screening or monitoring
    }

    /**
     * Urgency Levels
     * Timeline for test completion
     */
    public enum Urgency {
        STAT,               // Within 1 hour
        URGENT,             // Within 4 hours
        TODAY,              // Same day
        ROUTINE,            // Within 24-48 hours
        SCHEDULED           // Schedule for future date
    }

    // ================================================================
    // CLINICAL CONTEXT
    // ================================================================
    private String indication;               // Clinical reason for ordering
    private String rationale;                // Why this test is recommended now
    private String expectedFindings;         // What we expect/hope to find
    private String interpretationGuidance;   // How to interpret results
    private List<String> differentialDiagnosis; // Diagnoses this test helps evaluate

    // ================================================================
    // TIMING & COLLECTION
    // ================================================================
    private Integer timeframeMinutes;        // How soon test should be done
    private String collectionTiming;         // Specific timing requirements
    private String optimalTiming;            // Best time for collection
    private boolean repeatTest;              // Is this a repeat/follow-up test
    private Long previousTestTimestamp;      // Timestamp of previous test if repeat

    // ================================================================
    // SAFETY & CONTRAINDICATIONS
    // ================================================================
    private List<String> contraindications;  // Reasons NOT to order this test
    private List<String> warnings;           // Important safety considerations
    private List<String> prerequisiteTests;  // Tests that should be done first
    private boolean requiresConsent;         // Does this test require informed consent
    private String safetyNotes;              // Additional safety information

    // ================================================================
    // DECISION SUPPORT
    // ================================================================
    private DecisionSupport decisionSupport;

    /**
     * Decision Support Information
     * Evidence and clinical reasoning behind the recommendation
     */
    @Data
    @Builder
    public static class DecisionSupport implements Serializable {
        private static final long serialVersionUID = 1L;

        private String guidelineReference;       // Which guideline recommends this
        private String evidenceLevel;            // A, B, C, D (strength of evidence)
        private String recommendationStrength;   // Strong, Moderate, Weak
        private Double confidenceScore;          // 0-1 confidence in recommendation
        private List<String> supportingEvidence; // PMIDs or guideline references
        private String clinicalReasoning;        // Explanation of clinical logic
    }

    // ================================================================
    // ORDERING INFORMATION
    // ================================================================
    private OrderingInformation orderingInfo;

    /**
     * Ordering Information
     * Practical details for ordering the test
     */
    @Data
    @Builder
    public static class OrderingInformation implements Serializable {
        private static final long serialVersionUID = 1L;

        private String orderCode;                // System order code
        private String loincCode;                // LOINC code if applicable
        private String cptCode;                  // CPT code if applicable
        private String specimenType;             // What specimen is needed
        private String collectionInstructions;   // How to collect
        private String transportInstructions;    // How to transport specimen
        private boolean fastingRequired;
        private Integer fastingHours;
        private String patientPreparation;
    }

    // ================================================================
    // ALTERNATIVES & OPTIMIZATION
    // ================================================================
    private List<TestAlternative> alternatives;  // Alternative tests to consider

    /**
     * Test Alternative
     * Alternative or complementary tests
     */
    @Data
    @Builder
    public static class TestAlternative implements Serializable {
        private static final long serialVersionUID = 1L;

        private String testId;
        private String testName;
        private String reason;                   // Why this might be better
        private boolean betterAppropriate;       // Is this more appropriate?
        private boolean lowerCost;               // Is this less expensive?
        private boolean fasterResult;            // Does this have faster TAT?
        private boolean lessSensitive;           // Trade-off: less sensitive?
    }

    // ================================================================
    // FOLLOW-UP & MONITORING
    // ================================================================
    private FollowUpGuidance followUpGuidance;

    /**
     * Follow-Up Guidance
     * What to do after the test is done
     */
    @Data
    @Builder
    public static class FollowUpGuidance implements Serializable {
        private static final long serialVersionUID = 1L;

        private String actionIfNormal;           // What to do if normal
        private String actionIfAbnormal;         // What to do if abnormal
        private String actionIfCritical;         // What to do if critical
        private List<String> reflexTests;        // Tests to order based on results
        private Integer repeatIntervalHours;     // How often to repeat
        private String monitoringPlan;           // Ongoing monitoring strategy
    }

    // ================================================================
    // COST & UTILIZATION
    // ================================================================
    private Double estimatedCost;            // Estimated patient cost
    private boolean highUtilization;         // Is this a commonly ordered test
    private String stewardshipNotes;         // Cost-effectiveness notes

    // ================================================================
    // CONTEXT & METADATA
    // ================================================================
    private String patientId;                // Patient this is for
    private String encounterId;              // Encounter context
    private String orderingProvider;         // Who should order
    private String generatedBy;              // System/protocol that generated this
    private String protocolId;               // Protocol this is part of
    private Map<String, String> additionalContext; // Additional context data

    // ================================================================
    // HELPER METHODS
    // ================================================================

    /**
     * Check if recommendation is still valid based on timing
     *
     * @return true if recommendation is still timely
     */
    public boolean isStillValid() {
        if (timestamp == null || timeframeMinutes == null) {
            return true;
        }

        long elapsedMinutes = (System.currentTimeMillis() - timestamp) / (1000 * 60);
        return elapsedMinutes < timeframeMinutes;
    }

    /**
     * Check if this is a high-priority recommendation
     *
     * @return true if priority is P0 or P1
     */
    public boolean isHighPriority() {
        return priority == Priority.P0_CRITICAL || priority == Priority.P1_URGENT;
    }

    /**
     * Check if this recommendation requires immediate action
     *
     * @return true if urgency is STAT or URGENT
     */
    public boolean requiresImmediateAction() {
        return urgency == Urgency.STAT || urgency == Urgency.URGENT;
    }

    /**
     * Check if patient has contraindications for this test
     *
     * @param patientConditions List of patient conditions
     * @return true if any contraindication matches patient conditions
     */
    public boolean hasContraindication(List<String> patientConditions) {
        if (contraindications == null || contraindications.isEmpty() ||
            patientConditions == null || patientConditions.isEmpty()) {
            return false;
        }

        return contraindications.stream()
            .anyMatch(contraindication ->
                patientConditions.stream()
                    .anyMatch(condition ->
                        condition.toLowerCase().contains(contraindication.toLowerCase())
                    )
            );
    }

    /**
     * Get the urgency deadline as timestamp
     *
     * @return Unix timestamp when test should be completed
     */
    public Long getUrgencyDeadline() {
        if (timestamp == null || urgency == null) {
            return null;
        }

        long additionalMillis;
        switch (urgency) {
            case STAT:
                additionalMillis = 60 * 60 * 1000L; // 1 hour
                break;
            case URGENT:
                additionalMillis = 4 * 60 * 60 * 1000L; // 4 hours
                break;
            case TODAY:
                additionalMillis = 24 * 60 * 60 * 1000L; // 24 hours
                break;
            case ROUTINE:
                additionalMillis = 48 * 60 * 60 * 1000L; // 48 hours
                break;
            case SCHEDULED:
            default:
                return null; // No specific deadline
        }

        return timestamp + additionalMillis;
    }

    /**
     * Get confidence score for this recommendation
     *
     * @return Confidence score 0-1, or null if not available
     */
    public Double getConfidenceScore() {
        return decisionSupport != null ? decisionSupport.getConfidenceScore() : null;
    }

    /**
     * Check if this is a lab test recommendation
     *
     * @return true if test category is LAB
     */
    public boolean isLabTest() {
        return TestCategory.LAB.equals(testCategory);
    }

    /**
     * Check if this is an imaging study recommendation
     *
     * @return true if test category is IMAGING
     */
    public boolean isImagingStudy() {
        return TestCategory.IMAGING.equals(testCategory);
    }

    /**
     * Get human-readable priority description
     *
     * @return Priority description string
     */
    public String getPriorityDescription() {
        if (priority == null) return "Unknown";

        switch (priority) {
            case P0_CRITICAL:
                return "Critical - Life-threatening";
            case P1_URGENT:
                return "Urgent - Significant clinical impact";
            case P2_IMPORTANT:
                return "Important - Needed for diagnosis";
            case P3_ROUTINE:
                return "Routine - Screening or monitoring";
            default:
                return "Unknown";
        }
    }

    /**
     * Get human-readable urgency description
     *
     * @return Urgency description string
     */
    public String getUrgencyDescription() {
        if (urgency == null) return "Unknown";

        switch (urgency) {
            case STAT:
                return "STAT - Within 1 hour";
            case URGENT:
                return "Urgent - Within 4 hours";
            case TODAY:
                return "Same Day - Within 24 hours";
            case ROUTINE:
                return "Routine - Within 48 hours";
            case SCHEDULED:
                return "Scheduled - Future appointment";
            default:
                return "Unknown";
        }
    }

    /**
     * Check if prerequisites are met
     *
     * @param completedTests List of test IDs that have been completed
     * @return true if all prerequisites are met
     */
    public boolean arePrerequisitesMet(List<String> completedTests) {
        if (prerequisiteTests == null || prerequisiteTests.isEmpty()) {
            return true;
        }

        if (completedTests == null) {
            return false;
        }

        return completedTests.containsAll(prerequisiteTests);
    }
}
