package com.cardiofit.flink.models.diagnostics;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Comprehensive unit tests for ImagingStudy model (Phase 4).
 *
 * Test Coverage:
 * - Appropriateness checks (ACR criteria): 4 tests
 * - Contrast safety checks (renal function, allergies): 5 tests
 * - Radiation safety checks: 3 tests
 * - Timing and repeat study rules: 3 tests
 * - Helper methods: 5 tests
 *
 * Total: 20 unit tests for ImagingStudy
 *
 * @author Module 3 Phase 4 Team
 * @version 1.0
 * @since 2025-10-23
 */
@DisplayName("ImagingStudy Model Tests")
class ImagingStudyTest {

    private ImagingStudy chestXRay;
    private ImagingStudy ctChestWithContrast;

    @BeforeEach
    void setUp() {
        chestXRay = createChestXRay();
        ctChestWithContrast = createCTChestWithContrast();
    }

    // ============================================================
    // APPROPRIATENESS TESTS (4 tests)
    // ============================================================

    @Test
    @DisplayName("Is Appropriate: Chest X-ray for pneumonia (ACR 9)")
    void testIsAppropriate_ChestXRayForPneumonia() {
        // When: Check appropriateness for pneumonia
        boolean appropriate = chestXRay.isAppropriate("suspected pneumonia");

        // Then: Should be appropriate
        assertTrue(appropriate);
        assertTrue(chestXRay.isUsuallyAppropriate());
    }

    @Test
    @DisplayName("Is Appropriate: Chest X-ray for dyspnea")
    void testIsAppropriate_ChestXRayForDyspnea() {
        // When: Check appropriateness for dyspnea
        boolean appropriate = chestXRay.isAppropriate("acute dyspnea");

        // Then: Should be appropriate
        assertTrue(appropriate);
    }

    @Test
    @DisplayName("Is Appropriate: CT Chest for pulmonary embolism")
    void testIsAppropriate_CTForPulmonaryEmbolism() {
        // When: Check appropriateness for PE
        boolean appropriate = ctChestWithContrast.isAppropriate("pulmonary embolism");

        // Then: Should be appropriate
        assertTrue(appropriate);
    }

    @Test
    @DisplayName("ACR Score: Usually appropriate is score >= 7")
    void testIsUsuallyAppropriate_ScoreThreshold() {
        // Given: Set ACR score to 7 (threshold)
        ImagingStudy.ACRAppropriatenessRating rating =
            ImagingStudy.ACRAppropriatenessRating.builder()
                .appropriatenessScore(7)
                .build();
        chestXRay.setAcrRating(rating);

        // Then: Should be usually appropriate
        assertTrue(chestXRay.isUsuallyAppropriate());

        // When: Set score to 6
        rating.setAppropriatenessScore(6);

        // Then: Should NOT be usually appropriate
        assertFalse(chestXRay.isUsuallyAppropriate());
    }

    // ============================================================
    // CONTRAST SAFETY TESTS (5 tests)
    // ============================================================

    @Test
    @DisplayName("Contrast Safety: Safe with normal renal function")
    void testIsContrastSafe_NormalRenalFunction() {
        // Given: Patient with normal GFR (60 mL/min)
        Double gfr = 60.0;
        boolean hasAllergy = false;

        // When: Check contrast safety
        boolean safe = ctChestWithContrast.isContrastSafe(gfr, hasAllergy);

        // Then: Should be safe
        assertTrue(safe);
    }

    @Test
    @DisplayName("Contrast Safety: Unsafe with renal impairment (GFR < 30)")
    void testIsContrastSafe_RenalImpairment() {
        // Given: Patient with severe renal impairment (GFR = 25 mL/min)
        Double gfr = 25.0;
        boolean hasAllergy = false;

        // When: Check contrast safety
        boolean safe = ctChestWithContrast.isContrastSafe(gfr, hasAllergy);

        // Then: Should be UNSAFE
        assertFalse(safe);
    }

    @Test
    @DisplayName("Contrast Safety: Unsafe with contrast allergy")
    void testIsContrastSafe_ContrastAllergy() {
        // Given: Patient with normal renal function but contrast allergy
        Double gfr = 60.0;
        boolean hasAllergy = true;

        // When: Check contrast safety
        boolean safe = ctChestWithContrast.isContrastSafe(gfr, hasAllergy);

        // Then: Should be UNSAFE (requires premedication)
        assertFalse(safe);
    }

    @Test
    @DisplayName("Contrast Safety: Safe when contrast not required")
    void testIsContrastSafe_NoContrastRequired() {
        // Given: Study that doesn't require contrast
        Double gfr = 25.0; // Even with low GFR
        boolean hasAllergy = true; // And contrast allergy

        // When: Check contrast safety for chest X-ray (no contrast)
        boolean safe = chestXRay.isContrastSafe(gfr, hasAllergy);

        // Then: Should be safe (contrast not required)
        assertTrue(safe);
    }

    @Test
    @DisplayName("Contrast Safety: GFR at threshold (30 mL/min)")
    void testIsContrastSafe_GFRAtThreshold() {
        // Given: Patient with GFR exactly at threshold
        Double gfr = 30.0;
        boolean hasAllergy = false;

        // When: Check contrast safety
        boolean safe = ctChestWithContrast.isContrastSafe(gfr, hasAllergy);

        // Then: Should be safe (>= threshold)
        assertTrue(safe);
    }

    // ============================================================
    // RADIATION SAFETY TESTS (3 tests)
    // ============================================================

    @Test
    @DisplayName("Radiation Level: Chest X-ray is LOW radiation")
    void testGetRadiationLevel_ChestXRayLow() {
        // When: Get radiation level
        String level = chestXRay.getRadiationLevel();

        // Then: Should be LOW
        assertEquals("LOW", level);
    }

    @Test
    @DisplayName("Radiation Level: CT Chest is MEDIUM radiation")
    void testGetRadiationLevel_CTMedium() {
        // When: Get radiation level
        String level = ctChestWithContrast.getRadiationLevel();

        // Then: Should be MEDIUM
        assertEquals("MEDIUM", level);
    }

    @Test
    @DisplayName("Safe In Pregnancy: Chest X-ray with shielding is relatively safe")
    void testIsSafeInPregnancy_ChestXRayWithShielding() {
        // When: Check pregnancy safety
        boolean safe = chestXRay.isSafeInPregnancy();

        // Then: X-ray with low dose can be safe with shielding
        // (marked as CAUTION in YAML)
        assertFalse(safe); // CAUTION means not fully safe
    }

    // ============================================================
    // TIMING AND REPEAT STUDY TESTS (3 tests)
    // ============================================================

    @Test
    @DisplayName("Can Repeat: Minimum interval not met")
    void testCanRepeat_IntervalNotMet() {
        // Given: CT done 5 days ago (minimum interval is 7 days)
        long fiveDaysAgo = System.currentTimeMillis() - (1000L * 60 * 60 * 24 * 5);

        // When: Check if can repeat
        boolean canRepeat = ctChestWithContrast.canRepeat(fiveDaysAgo);

        // Then: Should NOT allow repeat yet
        assertFalse(canRepeat);
    }

    @Test
    @DisplayName("Can Repeat: Minimum interval met")
    void testCanRepeat_IntervalMet() {
        // Given: CT done 10 days ago (minimum interval is 7 days)
        long tenDaysAgo = System.currentTimeMillis() - (1000L * 60 * 60 * 24 * 10);

        // When: Check if can repeat
        boolean canRepeat = ctChestWithContrast.canRepeat(tenDaysAgo);

        // Then: Should allow repeat
        assertTrue(canRepeat);
    }

    @Test
    @DisplayName("Can Repeat: No minimum interval allows immediate repeat")
    void testCanRepeat_NoMinimumInterval() {
        // Given: X-ray done 1 hour ago, but no minimum interval set
        long oneHourAgo = System.currentTimeMillis() - (1000L * 60 * 60);

        // When: Check if can repeat
        boolean canRepeat = chestXRay.canRepeat(oneHourAgo);

        // Then: Should allow repeat (no restriction)
        assertTrue(canRepeat);
    }

    // ============================================================
    // HELPER METHOD TESTS (5 tests)
    // ============================================================

    @Test
    @DisplayName("Requires Safety Screening: CT with contrast requires screening")
    void testRequiresSafetyScreening_CTWithContrast() {
        // When: Check if requires safety screening
        boolean requiresScreening = ctChestWithContrast.requiresSafetyScreening();

        // Then: Should require screening (renal function, allergy)
        assertTrue(requiresScreening);
    }

    @Test
    @DisplayName("Requires Safety Screening: X-ray requires pregnancy check")
    void testRequiresSafetyScreening_XRayPregnancyCheck() {
        // When: Check if requires safety screening
        boolean requiresScreening = chestXRay.requiresSafetyScreening();

        // Then: Should require pregnancy check
        assertTrue(requiresScreening);
    }

    @Test
    @DisplayName("Get Appropriateness Score: Returns ACR score")
    void testGetAppropriatenessScore() {
        // When: Get appropriateness score
        Integer score = chestXRay.getAppropriatenessScore();

        // Then: Should return 9
        assertNotNull(score);
        assertEquals(9, score);
    }

    @Test
    @DisplayName("Builder Pattern: Complete study construction")
    void testBuilder_CompleteConstruction() {
        // When: Build complete imaging study
        ImagingStudy study = ImagingStudy.builder()
            .studyId("IMG-TEST-001")
            .studyName("Test Study")
            .cptCode("12345")
            .studyType(ImagingStudy.StudyType.CT)
            .bodyRegion("CHEST")
            .build();

        // Then: All fields should be set
        assertNotNull(study);
        assertEquals("IMG-TEST-001", study.getStudyId());
        assertEquals("Test Study", study.getStudyName());
        assertEquals("12345", study.getCptCode());
        assertEquals(ImagingStudy.StudyType.CT, study.getStudyType());
        assertEquals("CHEST", study.getBodyRegion());
    }

    @Test
    @DisplayName("YAML Integration: All critical fields populated")
    void testYAMLIntegration_AllFieldsPopulated() {
        // Then: Verify chest X-ray has all critical fields from YAML
        assertNotNull(chestXRay.getStudyId());
        assertNotNull(chestXRay.getStudyName());
        assertNotNull(chestXRay.getCptCode());
        assertNotNull(chestXRay.getStudyType());
        assertNotNull(chestXRay.getRequirements());
        assertNotNull(chestXRay.getRadiationExposure());
        assertNotNull(chestXRay.getTiming());
        assertNotNull(chestXRay.getAcrRating());

        // Verify ACR rating
        assertEquals(9, chestXRay.getAppropriatenessScore());

        // Verify CT has contrast safety info
        assertNotNull(ctChestWithContrast.getContrastSafety());
        assertEquals(30.0, ctChestWithContrast.getContrastSafety().getMinimumGFR());
    }

    // ============================================================
    // HELPER METHODS - TEST DATA CREATION
    // ============================================================

    private ImagingStudy createChestXRay() {
        return ImagingStudy.builder()
            .studyId("IMG-CXR-001")
            .studyName("Chest X-Ray (2-View)")
            .cptCode("71046")
            .studyType(ImagingStudy.StudyType.XRAY)
            .modality("RADIOGRAPHY")
            .bodyRegion("CHEST")
            .appropriateFor(Arrays.asList(
                "suspected pneumonia",
                "acute dyspnea",
                "chest trauma",
                "suspected pneumothorax"
            ))
            .inappropriateFor(Arrays.asList(
                "routine screening",
                "uncomplicated bronchitis"
            ))
            .acrRating(ImagingStudy.ACRAppropriatenessRating.builder()
                .clinicalScenario("Acute respiratory symptoms with fever")
                .appropriatenessScore(9)
                .rating("USUALLY_APPROPRIATE")
                .relativeRadiationLevel("Low")
                .evidenceBase("IDSA/ATS CAP Guidelines 2019")
                .build())
            .requirements(ImagingStudy.ImagingRequirements.builder()
                .contrastRequired(false)
                .sedationRequired(false)
                .durationMinutes(5)
                .patientPreparation("Remove metal objects")
                .build())
            .radiationExposure(ImagingStudy.RadiationExposure.builder()
                .hasRadiation(true)
                .effectiveDose("0.1 mSv") // Radiation dose as string
                .radiationLevel("LOW")
                .pregnancyRisk("CAUTION")
                .requiresPregnancyTest(false)
                .build())
            .safetyChecks(ImagingStudy.SafetyChecks.builder()
                .requiresMRISafety(false)
                .requiresPregnancyCheck(true)
                .requiresRenalFunction(false)
                .requiresAllergyScrening(false)
                .build())
            .timing(ImagingStudy.ImagingTiming.builder()
                .schedulingLeadTime(0)
                .studyDurationMinutes(5)
                .reportTurnaround(2) // hours
                .finalReportTurnaround(24) // hours
                .availability("24/7")
                .portableAvailable(true)
                .build())
            .orderingRules(ImagingStudy.OrderingRules.builder()
                .indications(Arrays.asList(
                    "Acute respiratory symptoms",
                    "Chest trauma",
                    "Suspected pneumothorax"
                ))
                .contraindications(Arrays.asList("Pregnancy (relative)"))
                .requiresPriorAuthorization(false)
                .build())
            .cost(ImagingStudy.CostData.builder()
                .institutionalCost(75.0)
                .patientCharge(250.0)
                .highCost(false)
                .highUtilization(true)
                .build())
            .evidenceLevel("A")
            .version("1.0")
            .build();
    }

    private ImagingStudy createCTChestWithContrast() {
        return ImagingStudy.builder()
            .studyId("IMG-CT-CHEST-001")
            .studyName("CT Chest with IV Contrast")
            .cptCode("71260")
            .studyType(ImagingStudy.StudyType.CT)
            .modality("CT")
            .bodyRegion("CHEST")
            .appropriateFor(Arrays.asList(
                "pulmonary embolism",
                "lung mass evaluation",
                "mediastinal mass"
            ))
            .inappropriateFor(Arrays.asList(
                "routine pneumonia",
                "simple bronchitis"
            ))
            .acrRating(ImagingStudy.ACRAppropriatenessRating.builder()
                .clinicalScenario("Suspected pulmonary embolism")
                .appropriatenessScore(9)
                .rating("USUALLY_APPROPRIATE")
                .relativeRadiationLevel("Medium")
                .evidenceBase("ACR Appropriateness Criteria")
                .build())
            .requirements(ImagingStudy.ImagingRequirements.builder()
                .contrastRequired(true)
                .contrastType("IODINATED")
                .contrastVolume(100.0)
                .sedationRequired(false)
                .durationMinutes(15)
                .patientPreparation("NPO 4 hours, hydration")
                .breathHoldInstructions("Hold breath for 10 seconds")
                .build())
            .radiationExposure(ImagingStudy.RadiationExposure.builder()
                .hasRadiation(true)
                .effectiveDose("7.0 mSv") // Radiation dose as string
                .radiationLevel("MEDIUM")
                .pregnancyRisk("CONTRAINDICATED")
                .requiresPregnancyTest(true)
                .justificationRequired("Yes - significant radiation exposure")
                .build())
            .contrastSafety(ImagingStudy.ContrastSafety.builder()
                .requiresRenalFunction(true)
                .minimumGFR(30.0)
                .requiresAllergyScreen(true)
                .contraindications(Arrays.asList(
                    "Severe renal impairment (GFR < 30)",
                    "Anaphylactic contrast allergy"
                ))
                .precautions(Arrays.asList(
                    "Diabetes with metformin",
                    "Mild renal impairment",
                    "Prior contrast reaction"
                ))
                .premedication("Prednisone 50mg PO x2 if prior reaction")
                .requiresPostHydration(true)
                .alternativeIfContraindicated("MRI Chest or V/Q scan")
                .build())
            .safetyChecks(ImagingStudy.SafetyChecks.builder()
                .requiresMRISafety(false)
                .requiresPregnancyCheck(true)
                .requiresRenalFunction(true)
                .requiresAllergyScrening(true)
                .medicationChecks(Arrays.asList("Metformin - hold 48 hours post-contrast"))
                .consentRequired("Informed consent for contrast")
                .build())
            .timing(ImagingStudy.ImagingTiming.builder()
                .schedulingLeadTime(0)
                .studyDurationMinutes(15)
                .reportTurnaround(2) // hours
                .finalReportTurnaround(24)
                .availability("24/7")
                .portableAvailable(false)
                .build())
            .orderingRules(ImagingStudy.OrderingRules.builder()
                .indications(Arrays.asList(
                    "Suspected pulmonary embolism",
                    "Lung mass evaluation",
                    "Complex pneumonia"
                ))
                .contraindications(Arrays.asList(
                    "Pregnancy",
                    "Severe renal failure",
                    "Anaphylactic contrast allergy without premedication"
                ))
                .prerequisiteTests(Arrays.asList("Creatinine/GFR", "Pregnancy test if applicable"))
                .minimumIntervalDays(7)
                .requiresPriorAuthorization(false)
                .build())
            .cost(ImagingStudy.CostData.builder()
                .institutionalCost(800.0)
                .patientCharge(3500.0)
                .highCost(true)
                .highUtilization(false)
                .stewardshipRecommendations("Consider chest X-ray first for simple cases")
                .requiresUtilizationReview(false)
                .build())
            .evidenceLevel("A")
            .version("1.0")
            .build();
    }
}
