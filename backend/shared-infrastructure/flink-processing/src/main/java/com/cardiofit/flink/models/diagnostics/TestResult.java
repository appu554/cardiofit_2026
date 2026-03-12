package com.cardiofit.flink.models.diagnostics;

import lombok.Data;
import lombok.Builder;
import java.io.Serializable;
import java.util.*;

/**
 * Test Result
 *
 * Represents the result of a diagnostic test (lab or imaging) with interpretation,
 * flags for abnormal/critical values, and automated clinical decision support.
 * Used for result storage, trending analysis, and triggering appropriate clinical
 * actions in the CardioFit clinical decision support system.
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-23
 */
@Data
@Builder
public class TestResult implements Serializable {
    private static final long serialVersionUID = 1L;

    // ================================================================
    // IDENTIFICATION
    // ================================================================
    private String resultId;                 // Unique ID for this result
    private String testId;                   // Links to LabTest or ImagingStudy
    private String testName;                 // Human-readable test name
    private String patientId;                // Patient identifier
    private String encounterId;              // Encounter context
    private String orderId;                  // Order that generated this result
    private TestType testType;               // LAB, IMAGING, PROCEDURE

    /**
     * Test Type Enumeration
     */
    public enum TestType {
        LAB,            // Laboratory test
        IMAGING,        // Imaging study
        PROCEDURE,      // Procedural result
        PATHOLOGY,      // Pathology result
        MICROBIOLOGY    // Culture/microbiology
    }

    // ================================================================
    // RESULT VALUES
    // ================================================================
    private String resultValue;              // String representation of result
    private Double numericValue;             // Numeric value if applicable
    private String resultUnit;               // Unit of measurement
    private String textValue;                // Text/narrative result (for imaging, path)
    private ResultStatus status;             // PRELIMINARY, FINAL, CORRECTED, CANCELLED

    /**
     * Result Status Enumeration
     */
    public enum ResultStatus {
        PRELIMINARY,        // Preliminary result
        FINAL,             // Final verified result
        CORRECTED,         // Corrected result
        CANCELLED,         // Cancelled/invalid
        PENDING            // Result pending
    }

    // ================================================================
    // TIMING
    // ================================================================
    private Long collectionTimestamp;        // When specimen was collected
    private Long resultTimestamp;            // When result became available
    private Long verificationTimestamp;      // When result was verified
    private Integer turnaroundTimeMinutes;   // Actual TAT

    // ================================================================
    // INTERPRETATION
    // ================================================================
    private ResultInterpretation resultInterpretation; // NORMAL, LOW, HIGH, CRITICAL_LOW, CRITICAL_HIGH

    /**
     * Result Interpretation Enumeration
     */
    public enum ResultInterpretation {
        NORMAL,             // Within normal range
        LOW,                // Below normal range
        HIGH,               // Above normal range
        CRITICAL_LOW,       // Critically low - immediate action needed
        CRITICAL_HIGH,      // Critically high - immediate action needed
        ABNORMAL,           // Abnormal (for non-numeric results)
        INDETERMINATE,      // Cannot be determined
        NOT_APPLICABLE      // Not applicable for this test
    }

    // ================================================================
    // REFERENCE RANGE
    // ================================================================
    private ReferenceRange referenceRange;

    /**
     * Reference Range
     * Normal and critical ranges for this result
     */
    @Data
    @Builder
    public static class ReferenceRange implements Serializable {
        private static final long serialVersionUID = 1L;

        private Double normalLow;            // Lower limit of normal
        private Double normalHigh;           // Upper limit of normal
        private Double criticalLow;          // Critical low threshold
        private Double criticalHigh;         // Critical high threshold
        private String unit;                 // Unit of measurement
        private String population;           // Adult, pediatric, etc.
        private String sex;                  // M, F, ALL
        private String interpretationText;   // Human-readable interpretation
    }

    // ================================================================
    // FLAGS & ALERTS
    // ================================================================
    private boolean isAbnormal;              // Is result outside normal range
    private boolean isCritical;              // Is result in critical range
    private boolean requiresFollowUp;        // Does result require follow-up action
    private boolean requiresPhysicianReview; // Does result need physician notification
    private List<String> flags;              // Additional flags (H, L, LL, HH, etc.)
    private String alertMessage;             // Alert message if critical

    // ================================================================
    // CLINICAL CONTEXT
    // ================================================================
    private String clinicalSignificance;     // What this result means clinically
    private String interpretation;           // Detailed interpretation
    private List<String> possibleCauses;     // Possible causes of abnormal result
    private String actionRequired;           // Recommended action based on result
    private List<String> differentialDiagnosis; // Differential diagnoses suggested

    // ================================================================
    // QUALITY INDICATORS
    // ================================================================
    private QualityIndicators qualityIndicators;

    /**
     * Quality Indicators
     * Factors affecting result quality and reliability
     */
    @Data
    @Builder
    public static class QualityIndicators implements Serializable {
        private static final long serialVersionUID = 1L;

        private boolean specimenAcceptable;      // Was specimen quality acceptable
        private List<String> specimenIssues;     // Hemolysis, lipemia, clotted, etc.
        private boolean interferenceDetected;    // Any interference detected
        private List<String> interferingFactors; // Factors affecting result
        private String specimenType;             // Actual specimen type
        private String collectionMethod;         // How specimen was collected
        private boolean deltaCheckPassed;        // Significant change from previous?
    }

    // ================================================================
    // COMPARISON & TRENDING
    // ================================================================
    private PreviousResults previousResults;

    /**
     * Previous Results
     * For trending and delta checks
     */
    @Data
    @Builder
    public static class PreviousResults implements Serializable {
        private static final long serialVersionUID = 1L;

        private Double previousValue;
        private Long previousTimestamp;
        private Double percentageChange;
        private String trend;                    // INCREASING, DECREASING, STABLE
        private boolean significantChange;       // Delta check flag
        private List<HistoricalValue> history;   // Historical values for trending

        @Data
        @Builder
        public static class HistoricalValue implements Serializable {
            private static final long serialVersionUID = 1L;
            private Double value;
            private Long timestamp;
            private String interpretation;
        }
    }

    // ================================================================
    // REFLEXIVE TESTING
    // ================================================================
    private ReflexActions reflexActions;

    /**
     * Reflex Actions
     * Automated follow-up tests triggered by this result
     */
    @Data
    @Builder
    public static class ReflexActions implements Serializable {
        private static final long serialVersionUID = 1L;

        private boolean reflexTriggered;         // Was reflex testing triggered
        private List<String> reflexTests;        // Tests automatically ordered
        private String reflexReason;             // Why reflex testing was triggered
        private String reflexProtocol;           // Protocol that triggered reflex
    }

    // ================================================================
    // REPORTING
    // ================================================================
    private String reportingProvider;        // Who reported the result
    private String verifyingProvider;        // Who verified the result
    private String reportNarrative;          // Full narrative report (for imaging)
    private String impression;               // Clinical impression
    private String recommendations;          // Provider recommendations

    // ================================================================
    // METADATA
    // ================================================================
    private String performingLab;            // Lab/imaging center that performed test
    private String performingLocation;       // Physical location
    private String methodology;              // Test methodology
    private String specimenSource;           // Where specimen came from
    private Map<String, String> additionalData; // Additional result data

    // ================================================================
    // HELPER METHODS
    // ================================================================

    /**
     * Interpret a numeric value based on reference range
     *
     * @return ResultInterpretation enum value
     */
    public ResultInterpretation interpretValue() {
        if (numericValue == null || referenceRange == null) {
            return ResultInterpretation.INDETERMINATE;
        }

        // Check critical ranges first
        if (referenceRange.getCriticalLow() != null &&
            numericValue < referenceRange.getCriticalLow()) {
            return ResultInterpretation.CRITICAL_LOW;
        }

        if (referenceRange.getCriticalHigh() != null &&
            numericValue > referenceRange.getCriticalHigh()) {
            return ResultInterpretation.CRITICAL_HIGH;
        }

        // Check normal range
        if (referenceRange.getNormalLow() != null &&
            numericValue < referenceRange.getNormalLow()) {
            return ResultInterpretation.LOW;
        }

        if (referenceRange.getNormalHigh() != null &&
            numericValue > referenceRange.getNormalHigh()) {
            return ResultInterpretation.HIGH;
        }

        return ResultInterpretation.NORMAL;
    }

    /**
     * Compare this result to reference range
     *
     * @param range Reference range to compare against
     * @return Interpretation of comparison
     */
    public ResultInterpretation compareToReference(ReferenceRange range) {
        if (numericValue == null || range == null) {
            return ResultInterpretation.INDETERMINATE;
        }

        // Check critical ranges
        if (range.getCriticalLow() != null && numericValue < range.getCriticalLow()) {
            return ResultInterpretation.CRITICAL_LOW;
        }
        if (range.getCriticalHigh() != null && numericValue > range.getCriticalHigh()) {
            return ResultInterpretation.CRITICAL_HIGH;
        }

        // Check normal ranges
        if (range.getNormalLow() != null && numericValue < range.getNormalLow()) {
            return ResultInterpretation.LOW;
        }
        if (range.getNormalHigh() != null && numericValue > range.getNormalHigh()) {
            return ResultInterpretation.HIGH;
        }

        return ResultInterpretation.NORMAL;
    }

    /**
     * Check if this result needs physician review
     *
     * @return true if physician notification required
     */
    public boolean needsPhysicianReview() {
        return isCritical ||
               requiresPhysicianReview ||
               (previousResults != null && previousResults.isSignificantChange());
    }

    /**
     * Calculate the change from previous value
     *
     * @return Percentage change, or null if no previous value
     */
    public Double calculatePercentageChange() {
        if (numericValue == null ||
            previousResults == null ||
            previousResults.getPreviousValue() == null ||
            previousResults.getPreviousValue() == 0) {
            return null;
        }

        return ((numericValue - previousResults.getPreviousValue()) /
                previousResults.getPreviousValue()) * 100.0;
    }

    /**
     * Get trend direction based on previous results
     *
     * @return INCREASING, DECREASING, or STABLE
     */
    public String getTrend() {
        if (previousResults == null || previousResults.getTrend() != null) {
            return previousResults != null ? previousResults.getTrend() : "UNKNOWN";
        }

        Double percentChange = calculatePercentageChange();
        if (percentChange == null) {
            return "UNKNOWN";
        }

        if (Math.abs(percentChange) < 5.0) {
            return "STABLE";
        } else if (percentChange > 0) {
            return "INCREASING";
        } else {
            return "DECREASING";
        }
    }

    /**
     * Check if result is within normal range
     *
     * @return true if result is normal
     */
    public boolean isNormal() {
        return !isAbnormal &&
               (resultInterpretation == ResultInterpretation.NORMAL ||
                resultInterpretation == ResultInterpretation.NOT_APPLICABLE);
    }

    /**
     * Check if specimen quality is acceptable
     *
     * @return true if specimen quality is good
     */
    public boolean isSpecimenAcceptable() {
        return qualityIndicators == null || qualityIndicators.isSpecimenAcceptable();
    }

    /**
     * Get age of result in hours
     *
     * @return Hours since result was generated
     */
    public long getAgeInHours() {
        if (resultTimestamp == null) {
            return 0;
        }
        return (System.currentTimeMillis() - resultTimestamp) / (1000 * 60 * 60);
    }

    /**
     * Check if result is recent (less than 24 hours old)
     *
     * @return true if result is recent
     */
    public boolean isRecent() {
        return getAgeInHours() < 24;
    }

    /**
     * Get a formatted result string with unit
     *
     * @return Formatted result string
     */
    public String getFormattedResult() {
        if (numericValue != null) {
            return String.format("%.2f %s", numericValue,
                                resultUnit != null ? resultUnit : "");
        }
        return resultValue != null ? resultValue : "N/A";
    }

    /**
     * Check if this is a critical result that needs immediate attention
     *
     * @return true if critical and requires immediate action
     */
    public boolean requiresImmediateAction() {
        return isCritical ||
               resultInterpretation == ResultInterpretation.CRITICAL_LOW ||
               resultInterpretation == ResultInterpretation.CRITICAL_HIGH;
    }

    /**
     * Get severity level for alerting
     *
     * @return Severity string (CRITICAL, HIGH, MODERATE, LOW)
     */
    public String getSeverity() {
        if (isCritical) {
            return "CRITICAL";
        }
        if (isAbnormal) {
            return "HIGH";
        }
        if (requiresFollowUp) {
            return "MODERATE";
        }
        return "LOW";
    }

    /**
     * Get numeric value (alias for compatibility).
     * Same as getNumericValue().
     *
     * @return Numeric value of the result
     */
    public Double getValue() {
        return numericValue;
    }

    /**
     * Get timestamp (alias for compatibility).
     * Same as getResultTimestamp().
     *
     * @return Result timestamp
     */
    public Long getTimestamp() {
        return resultTimestamp;
    }
}
