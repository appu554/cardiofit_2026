package com.cardiofit.flink.models.diagnostics;

import lombok.Data;
import lombok.Builder;
import lombok.NoArgsConstructor;
import lombok.AllArgsConstructor;
import lombok.extern.jackson.Jacksonized;
import java.io.Serializable;
import java.util.*;

/**
 * Laboratory Test Definition
 *
 * Represents a single lab test with all clinical metadata including specimen requirements,
 * reference ranges, interpretation guidance, and ordering rules. Used for intelligent
 * test ordering and result interpretation in the CardioFit clinical decision support system.
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-23
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class LabTest implements Serializable {
    private static final long serialVersionUID = 1L;

    // ================================================================
    // IDENTIFICATION
    // ================================================================
    private String testId;              // e.g., "LAB-LACTATE-001"
    private String testName;            // e.g., "Serum Lactate"
    private String loincCode;           // e.g., "2524-7"
    private String commonNames;         // Alternative names
    private String category;            // CHEMISTRY, HEMATOLOGY, MICROBIOLOGY, etc.

    // ================================================================
    // SPECIMEN & COLLECTION
    // ================================================================
    private SpecimenRequirements specimen;

    /**
     * Specimen Requirements
     * Defines how the specimen should be collected, handled, and transported
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SpecimenRequirements implements Serializable {
        private static final long serialVersionUID = 1L;

        private String specimenType;    // BLOOD, URINE, CSF, etc.
        private String collection;      // Venipuncture, arterial, etc.
        private String container;       // Red top, lavender top, etc.
        private Double volumeRequired;  // mL
        private String specialHandling; // Refrigerate, light-sensitive, etc.
        private boolean fastingRequired;
    }

    // ================================================================
    // TIMING & LOGISTICS
    // ================================================================
    private TestTiming timing;

    /**
     * Test Timing Information
     * Defines turnaround times, availability, and point-of-care options
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class TestTiming implements Serializable {
        private static final long serialVersionUID = 1L;

        private Integer turnaroundTimeMinutes;      // Typical TAT
        private Integer urgentTurnaroundMinutes;    // STAT TAT
        private Integer criticalResultMinutes;      // Critical value notification
        private String availability;                // 24/7, Business hours, etc.
        private boolean pointOfCareAvailable;
    }

    // ================================================================
    // REFERENCE RANGES
    // ================================================================
    private Map<String, ReferenceRange> referenceRanges;  // Key: population (adult, pediatric, etc.)

    /**
     * Reference Range
     * Population-specific normal and critical ranges with interpretation guidance
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ReferenceRange implements Serializable {
        private static final long serialVersionUID = 1L;

        private String population;      // adult, pediatric, neonatal, etc.
        private String sex;             // M, F, ALL
        private Integer ageMin;         // Years
        private Integer ageMax;         // Years

        private NormalRange normal;
        private CriticalRange critical;
        private String unit;

        /**
         * Normal Range
         * Defines the expected normal values for healthy patients
         */
        @Data
        @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class NormalRange implements Serializable {
            private static final long serialVersionUID = 1L;

            private Double min;
            private Double max;
            private String interpretation;
        }

        /**
         * Critical Range
         * Defines values that require immediate clinical action
         */
        @Data
        @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class CriticalRange implements Serializable {
            private static final long serialVersionUID = 1L;

            private Double criticalLow;
            private Double criticalHigh;
            private String criticalInterpretation;
        }
    }

    // ================================================================
    // CLINICAL INTERPRETATION
    // ================================================================
    private InterpretationGuidance interpretation;

    /**
     * Interpretation Guidance
     * Clinical significance and interpretation of test results
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class InterpretationGuidance implements Serializable {
        private static final long serialVersionUID = 1L;

        private Map<String, String> resultInterpretation;  // low, normal, high, critical
        private String clinicalSignificance;
        private List<String> commonCauses;
        private List<String> differentialDiagnosis;
        private String actionableFindings;
    }

    // ================================================================
    // ORDERING RULES
    // ================================================================
    private OrderingRules orderingRules;

    /**
     * Ordering Rules
     * Clinical criteria for when and how often a test should be ordered
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class OrderingRules implements Serializable {
        private static final long serialVersionUID = 1L;

        private List<String> indications;           // When to order
        private List<String> contraindications;     // When not to order
        private Integer minimumIntervalHours;       // How often can be repeated
        private boolean requiresConsent;
        private String appropriatenessLevel;        // USUALLY_APPROPRIATE, MAY_BE_APPROPRIATE, etc.
        private List<String> prerequisiteTests;     // Tests that should be done first
    }

    // ================================================================
    // QUALITY & INTERFERENCE
    // ================================================================
    private QualityFactors quality;

    /**
     * Quality Factors
     * Factors that can affect test quality and result validity
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class QualityFactors implements Serializable {
        private static final long serialVersionUID = 1L;

        private List<String> interferingFactors;    // Hemolysis, lipemia, etc.
        private List<String> interferingMedications; // Drugs that affect results
        private String stability;                    // How long sample is stable
        private String rejectionCriteria;            // Clotted, insufficient volume, etc.
    }

    // ================================================================
    // COSTS & UTILIZATION
    // ================================================================
    private CostData cost;

    /**
     * Cost Data
     * Financial and utilization information for test stewardship
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class CostData implements Serializable {
        private static final long serialVersionUID = 1L;

        private Double institutionalCost;            // Cost to hospital
        private Double patientCharge;                // Charge to patient
        private boolean highUtilization;             // Frequently ordered
        private String stewardshipRecommendations;   // Cost-effective alternatives
    }

    // ================================================================
    // EVIDENCE & GUIDELINES
    // ================================================================
    private List<String> evidenceReferences;        // PMIDs
    private List<String> guidelineReferences;       // Which guidelines recommend this test
    private String evidenceLevel;                   // A, B, C

    // ================================================================
    // CLINICAL DECISION SUPPORT
    // ================================================================
    private CDSRules cdsRules;

    /**
     * CDS Rules
     * Automated alerts and follow-up recommendations
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class CDSRules implements Serializable {
        private static final long serialVersionUID = 1L;

        private boolean alertOnCritical;
        private List<String> autoOrderWith;          // Tests commonly ordered together
        private Map<String, String> reflexTesting;   // If result X, order test Y
        private String followUpGuidance;
    }

    // ================================================================
    // METADATA
    // ================================================================
    private String lastUpdated;
    private String source;                          // Lab manual, guideline, etc.
    private String version;

    // ================================================================
    // HELPER METHODS
    // ================================================================

    /**
     * Interpret a numeric result value based on reference ranges
     *
     * @param value The numeric test result value
     * @param population The patient population (adult, pediatric, neonatal)
     * @return Interpretation string (CRITICAL_LOW, LOW, NORMAL, HIGH, CRITICAL_HIGH, UNKNOWN)
     */
    public String interpretResult(Double value, String population) {
        if (value == null || referenceRanges == null || referenceRanges.isEmpty()) {
            return "UNKNOWN";
        }

        ReferenceRange range = referenceRanges.get(population);
        if (range == null) {
            range = referenceRanges.get("adult");  // Default to adult
        }

        if (range == null) return "UNKNOWN";

        // Check critical ranges first
        if (range.getCritical() != null) {
            if (range.getCritical().getCriticalLow() != null &&
                value < range.getCritical().getCriticalLow()) {
                return "CRITICAL_LOW";
            }
            if (range.getCritical().getCriticalHigh() != null &&
                value > range.getCritical().getCriticalHigh()) {
                return "CRITICAL_HIGH";
            }
        }

        // Check normal range
        if (range.getNormal() != null) {
            if (range.getNormal().getMin() != null &&
                value < range.getNormal().getMin()) {
                return "LOW";
            }
            if (range.getNormal().getMax() != null &&
                value > range.getNormal().getMax()) {
                return "HIGH";
            }
        }

        return "NORMAL";
    }

    /**
     * Check if test can be ordered given patient context
     *
     * @param context Patient clinical context
     * @return true if test can be safely ordered, false if contraindicated
     */
    public boolean canOrder(PatientContext context) {
        if (orderingRules == null) return true;

        // Check contraindications
        if (orderingRules.getContraindications() != null && context != null) {
            for (String contraindication : orderingRules.getContraindications()) {
                if (context.hasCondition(contraindication)) {
                    return false;
                }
            }
        }

        return true;
    }

    /**
     * Check if enough time has passed since last order to allow reordering
     *
     * @param lastOrderTimestamp Unix timestamp of last order in milliseconds
     * @return true if test can be reordered, false if minimum interval not met
     */
    public boolean canReorder(long lastOrderTimestamp) {
        if (orderingRules == null || orderingRules.getMinimumIntervalHours() == null) {
            return true;
        }

        long hoursSinceLastOrder =
            (System.currentTimeMillis() - lastOrderTimestamp) / (1000 * 60 * 60);

        return hoursSinceLastOrder >= orderingRules.getMinimumIntervalHours();
    }

    /**
     * Get turnaround time based on urgency level
     *
     * @param isUrgent Whether this is a STAT/urgent order
     * @return Turnaround time in minutes
     */
    public int getTurnaroundTime(boolean isUrgent) {
        if (timing == null) return 120; // Default 2 hours

        if (isUrgent && timing.getUrgentTurnaroundMinutes() != null) {
            return timing.getUrgentTurnaroundMinutes();
        }

        return timing.getTurnaroundTimeMinutes() != null ?
               timing.getTurnaroundTimeMinutes() : 120;
    }

    /**
     * Check if result value is critical and requires immediate notification
     *
     * @param value The numeric test result value
     * @param population The patient population
     * @return true if value is in critical range
     */
    public boolean isCritical(Double value, String population) {
        String interpretation = interpretResult(value, population);
        return interpretation.startsWith("CRITICAL");
    }

    /**
     * Get appropriate reference range for patient age and sex
     *
     * @param ageYears Patient age in years
     * @param sex Patient sex (M, F)
     * @return Most appropriate reference range or null if none found
     */
    public ReferenceRange getReferenceRangeForPatient(Integer ageYears, String sex) {
        if (referenceRanges == null || referenceRanges.isEmpty()) {
            return null;
        }

        // Try to find exact match by age and sex
        for (ReferenceRange range : referenceRanges.values()) {
            if (ageYears != null &&
                ageYears >= range.getAgeMin() &&
                ageYears <= range.getAgeMax() &&
                (range.getSex().equals(sex) || range.getSex().equals("ALL"))) {
                return range;
            }
        }

        // Default to adult if available
        return referenceRanges.get("adult");
    }

    /**
     * Check if test requires patient preparation
     *
     * @return true if fasting or special preparation required
     */
    public boolean requiresPreparation() {
        return specimen != null && specimen.isFastingRequired();
    }

    /**
     * PatientContext interface for checking contraindications
     * Placeholder - actual implementation should come from patient service
     */
    public interface PatientContext {
        boolean hasCondition(String condition);
    }
}
