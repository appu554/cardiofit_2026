package com.cardiofit.flink.models.diagnostics;

import lombok.Data;
import lombok.Builder;
import lombok.NoArgsConstructor;
import lombok.AllArgsConstructor;
import java.io.Serializable;
import java.util.*;

/**
 * Imaging Study Definition
 *
 * Represents an imaging study (X-ray, CT, MRI, Ultrasound, Nuclear Medicine, Cardiac imaging)
 * with clinical metadata including ACR appropriateness criteria, radiation exposure, contrast
 * safety checks, and ordering guidance. Used for intelligent imaging test ordering and
 * appropriateness validation in the CardioFit clinical decision support system.
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-23
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ImagingStudy implements Serializable {
    private static final long serialVersionUID = 1L;

    // ================================================================
    // IDENTIFICATION
    // ================================================================
    private String studyId;              // e.g., "IMG-CT-CHEST-001"
    private String studyName;            // e.g., "CT Chest with Contrast"
    private String cptCode;              // e.g., "71260"
    private String loincCode;            // Optional LOINC for imaging
    private StudyType studyType;         // XRAY, CT, MRI, ULTRASOUND, NUCLEAR, CARDIAC
    private String modality;             // Specific modality (e.g., "CT", "MRI-3T")
    private String bodyRegion;           // CHEST, ABDOMEN, HEAD, etc.

    /**
     * Imaging Study Type Enumeration
     */
    public enum StudyType {
        XRAY,           // Plain radiography
        CT,             // Computed Tomography
        MRI,            // Magnetic Resonance Imaging
        ULTRASOUND,     // Ultrasound/Sonography
        NUCLEAR,        // Nuclear Medicine (PET, SPECT)
        CARDIAC,        // Cardiac imaging (Echo, Stress, Cath)
        FLUOROSCOPY,    // Fluoroscopic procedures
        MAMMOGRAPHY     // Breast imaging
    }

    // ================================================================
    // CLINICAL CONTEXT
    // ================================================================
    private String clinicalIndication;       // Primary reason for ordering
    private List<String> appropriateFor;     // Clinical scenarios where appropriate
    private List<String> inappropriateFor;   // Scenarios where not recommended
    private String expectedFindings;         // What the study should reveal
    private String interpretationGuidance;   // How to interpret results

    // ================================================================
    // ACR APPROPRIATENESS CRITERIA
    // ================================================================
    private ACRAppropriatenessRating acrRating;

    /**
     * ACR Appropriateness Rating
     * Based on American College of Radiology appropriateness criteria
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ACRAppropriatenessRating implements Serializable {
        private static final long serialVersionUID = 1L;

        private String clinicalScenario;         // Specific clinical scenario
        private Integer appropriatenessScore;    // 1-9 scale
        private String rating;                   // USUALLY_APPROPRIATE (7-9), MAY_BE_APPROPRIATE (4-6), USUALLY_NOT_APPROPRIATE (1-3)
        private String relativeRadiationLevel;   // None, Low, Medium, High
        private List<String> alternativeStudies; // More appropriate alternatives if score < 7
        private String evidenceBase;             // Supporting evidence summary
    }

    // ================================================================
    // IMAGING REQUIREMENTS
    // ================================================================
    private ImagingRequirements requirements;

    /**
     * Imaging Requirements
     * Technical and patient preparation requirements
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ImagingRequirements implements Serializable {
        private static final long serialVersionUID = 1L;

        private boolean contrastRequired;
        private String contrastType;             // IODINATED, GADOLINIUM, NONE
        private Double contrastVolume;           // mL
        private boolean oralContrastRequired;
        private String patientPreparation;       // Fasting, hydration, etc.
        private boolean sedationRequired;
        private Integer durationMinutes;         // Expected study duration
        private String positioning;              // Patient positioning requirements
        private String breathHoldInstructions;   // For CT/MRI
    }

    // ================================================================
    // RADIATION EXPOSURE
    // ================================================================
    private RadiationExposure radiationExposure;

    /**
     * Radiation Exposure Information
     * For ionizing radiation modalities (X-ray, CT, Nuclear)
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class RadiationExposure implements Serializable {
        private static final long serialVersionUID = 1L;

        private boolean hasRadiation;
        private String effectiveDose;            // mSv (millisieverts) - can be range or single value with description
        private String radiationLevel;           // NONE, LOW, MEDIUM, HIGH, VERY_HIGH
        private String pregnancyRisk;            // CONTRAINDICATED, CAUTION, SAFE
        private boolean requiresPregnancyTest;
        private String justificationRequired;    // When radiation exposure must be justified
    }

    // ================================================================
    // CONTRAST SAFETY CHECKS
    // ================================================================
    private ContrastSafety contrastSafety;

    /**
     * Contrast Safety Checks
     * Safety screening for contrast administration
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ContrastSafety implements Serializable {
        private static final long serialVersionUID = 1L;

        private boolean requiresRenalFunction;
        private Double minimumGFR;               // Minimum eGFR for safe contrast (typically 30-45)
        private boolean requiresAllergyScreen;
        private List<String> contraindications;  // Absolute contraindications
        private List<String> precautions;        // Relative contraindications/precautions
        private String premedication;            // Required premedication protocol
        private boolean requiresPostHydration;
        private String alternativeIfContraindicated; // Alternative study if contrast contraindicated
    }

    // ================================================================
    // TIMING & LOGISTICS
    // ================================================================
    private ImagingTiming timing;

    /**
     * Imaging Timing Information
     * Scheduling, turnaround, and availability
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ImagingTiming implements Serializable {
        private static final long serialVersionUID = 1L;

        private Integer schedulingLeadTime;      // Minimum days to schedule
        private Integer studyDurationMinutes;    // How long the study takes
        private Integer reportTurnaround;        // Hours until preliminary report
        private Integer finalReportTurnaround;   // Hours until final report
        private String availability;             // 24/7, Business hours, etc.
        private boolean portableAvailable;       // Can be done at bedside
        private String urgencyLevel;             // STAT, URGENT, ROUTINE
    }

    // ================================================================
    // ORDERING RULES
    // ================================================================
    private OrderingRules orderingRules;

    /**
     * Ordering Rules
     * Clinical criteria for appropriate ordering
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class OrderingRules implements Serializable {
        private static final long serialVersionUID = 1L;

        private List<String> indications;           // When to order
        private List<String> contraindications;     // When NOT to order
        private List<String> prerequisiteTests;     // Required before ordering
        private Integer minimumIntervalDays;        // Minimum time between repeat studies
        private boolean requiresPriorAuthorization;
        private boolean requiresRadiologistConsult;
        private String orderingGuidance;            // Special ordering instructions
    }

    // ================================================================
    // SAFETY CHECKS
    // ================================================================
    private SafetyChecks safetyChecks;

    /**
     * Safety Checks
     * Required safety screening before study
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SafetyChecks implements Serializable {
        private static final long serialVersionUID = 1L;

        private boolean requiresMRISafety;           // Check for metal implants
        private boolean requiresPregnancyCheck;
        private boolean requiresRenalFunction;
        private boolean requiresAllergyScrening;
        private List<String> implantChecks;          // Pacemaker, ICD, cochlear implant, etc.
        private List<String> medicationChecks;       // Metformin hold, anticoagulation, etc.
        private String consentRequired;              // Level of consent needed
    }

    // ================================================================
    // COSTS & UTILIZATION
    // ================================================================
    private CostData cost;

    /**
     * Cost Data
     * Financial and utilization information
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class CostData implements Serializable {
        private static final long serialVersionUID = 1L;

        private Double institutionalCost;
        private Double patientCharge;
        private boolean highCost;                    // Flag for expensive studies
        private boolean highUtilization;
        private String stewardshipRecommendations;   // Alternative lower-cost studies
        private boolean requiresUtilizationReview;
    }

    // ================================================================
    // EVIDENCE & GUIDELINES
    // ================================================================
    private List<String> evidenceReferences;         // PMIDs, ACR references
    private List<String> guidelineReferences;        // ACR, Fleischner, specialty societies
    private String evidenceLevel;                    // A, B, C

    // ================================================================
    // CLINICAL DECISION SUPPORT
    // ================================================================
    private CDSRules cdsRules;

    /**
     * CDS Rules
     * Automated guidance and alerts
     */
    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class CDSRules implements Serializable {
        private static final long serialVersionUID = 1L;

        private boolean alertOnInappropriate;        // Alert if ACR rating < 4
        private boolean suggestAlternatives;         // Suggest better alternatives
        private List<String> orderWith;              // Studies often ordered together
        private Map<String, String> protocolSelection; // If indication X, use protocol Y
        private String followUpGuidance;
        private boolean requiresClinicalDecisionSupport; // Trigger CDS intervention
    }

    // ================================================================
    // METADATA
    // ================================================================
    private String lastUpdated;
    private String source;                           // ACR, institutional protocols
    private String version;

    // ================================================================
    // HELPER METHODS
    // ================================================================

    /**
     * Check if study is appropriate for given clinical indication
     *
     * @param indication Clinical indication
     * @return true if study is appropriate
     */
    public boolean isAppropriate(String indication) {
        if (appropriateFor == null) return true;
        return appropriateFor.stream()
            .anyMatch(ind -> ind.toLowerCase().contains(indication.toLowerCase()));
    }

    /**
     * Check if contrast can be safely administered to patient
     *
     * @param gfr Patient's eGFR
     * @param hasContrastAllergy Patient has contrast allergy history
     * @return true if contrast is safe
     */
    public boolean isContrastSafe(Double gfr, boolean hasContrastAllergy) {
        if (contrastSafety == null || !requirements.isContrastRequired()) {
            return true;
        }

        // Check GFR if required
        if (contrastSafety.isRequiresRenalFunction() &&
            contrastSafety.getMinimumGFR() != null &&
            gfr != null && gfr < contrastSafety.getMinimumGFR()) {
            return false;
        }

        // Check allergy if required
        if (contrastSafety.isRequiresAllergyScreen() && hasContrastAllergy) {
            return false; // Would need premedication protocol
        }

        return true;
    }

    /**
     * Check if enough time has passed to repeat study
     *
     * @param lastStudyTimestamp Unix timestamp of last study in milliseconds
     * @return true if study can be repeated
     */
    public boolean canRepeat(long lastStudyTimestamp) {
        if (orderingRules == null || orderingRules.getMinimumIntervalDays() == null) {
            return true;
        }

        long daysSinceLastStudy =
            (System.currentTimeMillis() - lastStudyTimestamp) / (1000 * 60 * 60 * 24);

        return daysSinceLastStudy >= orderingRules.getMinimumIntervalDays();
    }

    /**
     * Get radiation exposure level category
     *
     * @return Radiation level string
     */
    public String getRadiationLevel() {
        if (radiationExposure == null || !radiationExposure.isHasRadiation()) {
            return "NONE";
        }
        return radiationExposure.getRadiationLevel();
    }

    /**
     * Check if study requires safety screening
     *
     * @return true if any safety checks are required
     */
    public boolean requiresSafetyScreening() {
        return safetyChecks != null && (
            safetyChecks.isRequiresMRISafety() ||
            safetyChecks.isRequiresPregnancyCheck() ||
            safetyChecks.isRequiresRenalFunction() ||
            safetyChecks.isRequiresAllergyScrening()
        );
    }

    /**
     * Get ACR appropriateness score
     *
     * @return Score 1-9, or null if not defined
     */
    public Integer getAppropriatenessScore() {
        return acrRating != null ? acrRating.getAppropriatenessScore() : null;
    }

    /**
     * Check if study is usually appropriate (ACR score 7-9)
     *
     * @return true if usually appropriate
     */
    public boolean isUsuallyAppropriate() {
        Integer score = getAppropriatenessScore();
        return score != null && score >= 7;
    }

    /**
     * Check if pregnant patients can safely undergo this study
     *
     * @return true if safe in pregnancy
     */
    public boolean isSafeInPregnancy() {
        if (radiationExposure == null) return true;
        return !radiationExposure.isHasRadiation() ||
               "SAFE".equals(radiationExposure.getPregnancyRisk());
    }
}
