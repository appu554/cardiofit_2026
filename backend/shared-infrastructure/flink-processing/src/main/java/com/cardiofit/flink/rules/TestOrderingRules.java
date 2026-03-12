package com.cardiofit.flink.rules;

import com.cardiofit.flink.loader.DiagnosticTestLoader;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.diagnostics.LabTest;
import com.cardiofit.flink.models.diagnostics.ImagingStudy;
import com.cardiofit.flink.models.diagnostics.TestRecommendation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.time.Duration;
import java.time.Instant;
import java.util.*;

/**
 * Test Ordering Rules Engine
 *
 * Rule engine for complex diagnostic test ordering logic including clinical indications,
 * contraindications, prerequisite checking, minimum interval enforcement, and automated
 * test bundling. Provides safety validation and intelligent test sequencing for the
 * CardioFit clinical decision support system.
 *
 * <p>Key Capabilities:
 * - Clinical indication evaluation against patient context
 * - Contraindication checking (clinical, safety, cost)
 * - Minimum interval calculation and enforcement
 * - Prerequisite test validation
 * - Auto-ordering bundles (tests commonly ordered together)
 * - Evidence-based ordering appropriateness
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-23
 */
public class TestOrderingRules implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(TestOrderingRules.class);

    // Test loader for accessing test definitions
    private final DiagnosticTestLoader testLoader;

    // Common test bundles
    private final Map<String, List<String>> testBundles;

    /**
     * Constructor with dependency injection.
     *
     * @param testLoader Diagnostic test loader instance
     */
    public TestOrderingRules(DiagnosticTestLoader testLoader) {
        this.testLoader = testLoader;
        this.testBundles = initializeTestBundles();
    }

    /**
     * Default constructor (uses singleton test loader).
     */
    public TestOrderingRules() {
        this(DiagnosticTestLoader.getInstance());
    }

    /**
     * Initialize common test bundles.
     * Tests that are typically ordered together for efficiency and completeness.
     *
     * @return Map of primary test ID to list of bundled test IDs
     */
    private Map<String, List<String>> initializeTestBundles() {
        Map<String, List<String>> bundles = new HashMap<>();

        // Sepsis bundle
        bundles.put("LAB-LACTATE-001", Arrays.asList(
                "LAB-BLOOD-CULTURE-001",
                "LAB-WBC-001",
                "LAB-CMP-001"
        ));

        // Cardiac bundle (Troponin)
        bundles.put("LAB-TROPONIN-I-001", Arrays.asList(
                "LAB-CK-MB-001",
                "LAB-BNP-001",
                "LAB-PT-INR-001"
        ));

        // Coagulation bundle
        bundles.put("LAB-PT-INR-001", Arrays.asList(
                "LAB-PTT-001",
                "LAB-PLATELETS-001"
        ));

        // Renal function bundle
        bundles.put("LAB-CREATININE-001", Arrays.asList(
                "LAB-BUN-001",
                "LAB-URINALYSIS-001"
        ));

        // Liver function bundle
        bundles.put("LAB-ALT-001", Arrays.asList(
                "LAB-AST-001",
                "LAB-BILIRUBIN-001",
                "LAB-ALBUMIN-001",
                "LAB-ALP-001"
        ));

        return bundles;
    }

    // ================================================================
    // CLINICAL INDICATION EVALUATION
    // ================================================================

    /**
     * Check if test meets clinical indications for ordering.
     *
     * <p>Evaluates test-specific indications against patient clinical context:
     * - Symptoms matching indication
     * - Vital signs supporting indication
     * - Lab values supporting indication
     * - Diagnosis/condition matching indication
     *
     * @param test Test definition (LabTest or ImagingStudy)
     * @param context Patient context
     * @return true if clinical indications are met
     */
    public boolean meetsIndications(LabTest test, EnrichedPatientContext context) {
        if (test == null || context == null) {
            return false;
        }

        LabTest.OrderingRules orderingRules = test.getOrderingRules();
        if (orderingRules == null || orderingRules.getIndications() == null) {
            return true; // No specific indications defined
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            return false;
        }

        // Check each indication
        for (String indication : orderingRules.getIndications()) {
            if (evaluateIndication(indication, state)) {
                LOG.debug("Test {} meets indication: {}", test.getTestId(), indication);
                return true;
            }
        }

        LOG.debug("Test {} does not meet any clinical indications", test.getTestId());
        return false;
    }

    /**
     * Check if imaging study meets clinical indications.
     *
     * @param study Imaging study definition
     * @param context Patient context
     * @return true if clinical indications are met
     */
    public boolean meetsIndications(ImagingStudy study, EnrichedPatientContext context) {
        if (study == null || context == null) {
            return false;
        }

        ImagingStudy.OrderingRules orderingRules = study.getOrderingRules();
        if (orderingRules == null || orderingRules.getIndications() == null) {
            return true; // No specific indications defined
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            return false;
        }

        // Check each indication
        for (String indication : orderingRules.getIndications()) {
            if (evaluateIndication(indication, state)) {
                LOG.debug("Imaging study {} meets indication: {}", study.getStudyId(), indication);
                return true;
            }
        }

        LOG.debug("Imaging study {} does not meet any clinical indications", study.getStudyId());
        return false;
    }

    /**
     * Evaluate a specific clinical indication against patient state.
     *
     * @param indication Indication string to evaluate
     * @param state Patient clinical state
     * @return true if indication is met
     */
    private boolean evaluateIndication(String indication, PatientContextState state) {
        if (indication == null || state == null) {
            return false;
        }

        String lower = indication.toLowerCase();

        // Sepsis indicators
        if (lower.contains("sepsis") || lower.contains("infection")) {
            return hasSepsisIndicators(state);
        }

        // Respiratory indicators
        if (lower.contains("respiratory") || lower.contains("dyspnea") || lower.contains("hypoxia")) {
            return hasRespiratorySymptoms(state);
        }

        // Cardiac indicators
        if (lower.contains("chest pain") || lower.contains("cardiac") || lower.contains("ischemia")) {
            return hasCardiacSymptoms(state);
        }

        // Shock indicators
        if (lower.contains("shock") || lower.contains("hypotension")) {
            return hasShockIndicators(state);
        }

        // Metabolic acidosis
        if (lower.contains("acidosis") || lower.contains("metabolic")) {
            return hasMetabolicAcidosis(state);
        }

        // Renal failure
        if (lower.contains("renal") || lower.contains("kidney")) {
            return hasRenalDysfunction(state);
        }

        // Trauma
        if (lower.contains("trauma")) {
            return hasTraumaIndicators(state);
        }

        return false;
    }

    // ================================================================
    // CONTRAINDICATION CHECKING
    // ================================================================

    /**
     * Check for contraindications to ordering a test.
     *
     * <p>Evaluates:
     * - Absolute contraindications (never order)
     * - Relative contraindications (caution required)
     * - Patient-specific contraindications
     * - Safety contraindications
     *
     * @param test Test recommendation
     * @param context Patient context
     * @return List of contraindications found (empty if none)
     */
    public List<String> checkContraindications(
            TestRecommendation test,
            EnrichedPatientContext context) {

        List<String> contraindications = new ArrayList<>();

        if (test == null || context == null) {
            return contraindications;
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            return contraindications;
        }

        // Check test-defined contraindications
        if (test.getContraindications() != null) {
            for (String contraindication : test.getContraindications()) {
                if (hasContraindication(contraindication, state)) {
                    contraindications.add(contraindication);
                    LOG.warn("Contraindication found for test {}: {}",
                            test.getTestId(), contraindication);
                }
            }
        }

        // Imaging-specific contraindications
        if (test.isImagingStudy()) {
            contraindications.addAll(checkImagingContraindications(test, context));
        }

        // Lab-specific contraindications
        if (test.isLabTest()) {
            contraindications.addAll(checkLabContraindications(test, context));
        }

        return contraindications;
    }

    /**
     * Check imaging-specific contraindications.
     *
     * @param test Imaging test recommendation
     * @param context Patient context
     * @return List of contraindications
     */
    private List<String> checkImagingContraindications(
            TestRecommendation test,
            EnrichedPatientContext context) {

        List<String> contraindications = new ArrayList<>();

        ImagingStudy imagingStudy = testLoader.getImagingStudy(test.getTestId());
        if (imagingStudy == null) {
            return contraindications;
        }

        PatientContextState state = context.getPatientState();

        // Contrast contraindications (renal function)
        if (imagingStudy.getRequirements() != null &&
            imagingStudy.getRequirements().isContrastRequired()) {

            Double creatinine = getLabValue(state, "creatinine");
            if (creatinine != null && creatinine > 1.5) {
                contraindications.add("Impaired renal function (Cr > 1.5) - contrast risk");
            }

            // Check for contrast allergy
            if (hasContrastAllergy(state)) {
                contraindications.add("Known contrast allergy - premedication required");
            }
        }

        // Radiation contraindications (pregnancy)
        if (imagingStudy.getRadiationExposure() != null &&
            imagingStudy.getRadiationExposure().isHasRadiation()) {

            if (isPregnant(state)) {
                contraindications.add("Pregnancy - radiation exposure risk");
            }
        }

        // MRI contraindications (metal implants)
        if (ImagingStudy.StudyType.MRI.equals(imagingStudy.getStudyType())) {
            if (hasPacemaker(state)) {
                contraindications.add("Pacemaker/ICD present - MRI contraindicated");
            }
        }

        return contraindications;
    }

    /**
     * Check lab-specific contraindications.
     *
     * @param test Lab test recommendation
     * @param context Patient context
     * @return List of contraindications
     */
    private List<String> checkLabContraindications(
            TestRecommendation test,
            EnrichedPatientContext context) {

        List<String> contraindications = new ArrayList<>();

        // Example: Coagulation testing contraindicated if on certain meds
        // Placeholder for specific lab contraindications

        return contraindications;
    }

    // ================================================================
    // MINIMUM INTERVAL ENFORCEMENT
    // ================================================================

    /**
     * Calculate time until test can be reordered.
     *
     * <p>Enforces minimum intervals to prevent:
     * - Over-testing waste
     * - Unnecessary patient burden
     * - Increased healthcare costs
     *
     * @param testId Test identifier
     * @param lastOrderTime Time of last order (Instant)
     * @return Duration until test can be reordered (zero if can order now)
     */
    public Duration timeUntilCanReorder(String testId, Instant lastOrderTime) {
        if (testId == null || lastOrderTime == null) {
            return Duration.ZERO;
        }

        // Get test definition
        LabTest labTest = testLoader.getLabTest(testId);
        Integer minimumIntervalHours = null;

        if (labTest != null && labTest.getOrderingRules() != null) {
            minimumIntervalHours = labTest.getOrderingRules().getMinimumIntervalHours();
        } else {
            // Try imaging study
            ImagingStudy imagingStudy = testLoader.getImagingStudy(testId);
            if (imagingStudy != null && imagingStudy.getOrderingRules() != null) {
                Integer minimumIntervalDays = imagingStudy.getOrderingRules().getMinimumIntervalDays();
                if (minimumIntervalDays != null) {
                    minimumIntervalHours = minimumIntervalDays * 24;
                }
            }
        }

        // Default to 24 hours if not specified
        if (minimumIntervalHours == null) {
            minimumIntervalHours = 24;
        }

        // Calculate time elapsed
        Instant now = Instant.now();
        Duration elapsed = Duration.between(lastOrderTime, now);
        Duration minimumInterval = Duration.ofHours(minimumIntervalHours);

        // Return remaining time
        if (elapsed.compareTo(minimumInterval) >= 0) {
            return Duration.ZERO; // Can order now
        } else {
            return minimumInterval.minus(elapsed); // Time remaining
        }
    }

    // ================================================================
    // PREREQUISITE CHECKING
    // ================================================================

    /**
     * Get missing prerequisite tests.
     *
     * <p>Some tests require other tests to be completed first for:
     * - Safety (creatinine before contrast CT)
     * - Clinical appropriateness (pregnancy test before radiation)
     * - Diagnostic sequencing (CXR before CT chest)
     *
     * @param test Test recommendation
     * @param completedTests List of completed test IDs
     * @return List of missing prerequisite test IDs
     */
    public List<String> getMissingPrerequisites(
            TestRecommendation test,
            List<String> completedTests) {

        List<String> missing = new ArrayList<>();

        if (test == null || test.getPrerequisiteTests() == null) {
            return missing;
        }

        Set<String> completed = completedTests != null ?
                new HashSet<>(completedTests) : new HashSet<>();

        for (String prerequisite : test.getPrerequisiteTests()) {
            if (!completed.contains(prerequisite)) {
                missing.add(prerequisite);
                LOG.debug("Missing prerequisite for test {}: {}",
                        test.getTestId(), prerequisite);
            }
        }

        return missing;
    }

    // ================================================================
    // AUTO-ORDERING BUNDLES
    // ================================================================

    /**
     * Get auto-order bundle for a primary test.
     *
     * <p>When certain tests are ordered, other tests should be ordered together for:
     * - Clinical completeness (lactate + blood cultures for sepsis)
     * - Efficiency (CBC + CMP together)
     * - Cost-effectiveness (bundled pricing)
     * - Patient convenience (single blood draw)
     *
     * @param primaryTestId Primary test being ordered
     * @return List of test recommendations for bundle
     */
    public List<TestRecommendation> getAutoOrderBundle(String primaryTestId) {
        List<TestRecommendation> bundle = new ArrayList<>();

        if (primaryTestId == null) {
            return bundle;
        }

        // Check if test has defined bundle
        List<String> bundledTestIds = testBundles.get(primaryTestId);
        if (bundledTestIds == null || bundledTestIds.isEmpty()) {
            return bundle;
        }

        LOG.info("Auto-ordering bundle for primary test {}: {} tests",
                primaryTestId, bundledTestIds.size());

        long timestamp = System.currentTimeMillis();

        for (String bundledTestId : bundledTestIds) {
            LabTest labTest = testLoader.getLabTest(bundledTestId);
            if (labTest != null) {
                TestRecommendation rec = TestRecommendation.builder()
                        .recommendationId(UUID.randomUUID().toString())
                        .testId(labTest.getTestId())
                        .testName(labTest.getTestName())
                        .testCategory(TestRecommendation.TestCategory.LAB)
                        .timestamp(timestamp)
                        .priority(TestRecommendation.Priority.P2_IMPORTANT)
                        .urgency(TestRecommendation.Urgency.URGENT)
                        .indication("Auto-ordered with " + primaryTestId)
                        .rationale("Commonly ordered together for clinical completeness")
                        .generatedBy("TestOrderingRules")
                        .build();

                bundle.add(rec);
            }
        }

        return bundle;
    }

    // ================================================================
    // HELPER METHODS - CLINICAL INDICATORS
    // ================================================================

    private boolean hasSepsisIndicators(PatientContextState state) {
        // qSOFA ≥ 2 or high lactate or fever with leukocytosis
        Integer qsofa = state.getQsofaScore();
        if (qsofa != null && qsofa >= 2) {
            return true;
        }

        Double lactate = getLabValue(state, "lactate");
        if (lactate != null && lactate >= 2.0) {
            return true;
        }

        Double temp = getVitalValue(state, "temperature");
        Double wbc = getLabValue(state, "wbc");
        return (temp != null && temp > 38.0) && (wbc != null && (wbc > 12.0 || wbc < 4.0));
    }

    private boolean hasRespiratorySymptoms(PatientContextState state) {
        Double respRate = getVitalValue(state, "respiratoryrate");
        Double spo2 = getVitalValue(state, "spo2");

        return (respRate != null && respRate > 22) || (spo2 != null && spo2 < 92);
    }

    private boolean hasCardiacSymptoms(PatientContextState state) {
        // Would check for chest pain, elevated troponin, ECG changes
        Double troponin = getLabValue(state, "troponin");
        return troponin != null && troponin > 0.04;
    }

    private boolean hasShockIndicators(PatientContextState state) {
        Double sbp = getVitalValue(state, "systolicbp");
        Double lactate = getLabValue(state, "lactate");

        return (sbp != null && sbp < 90) || (lactate != null && lactate >= 4.0);
    }

    private boolean hasMetabolicAcidosis(PatientContextState state) {
        // Would check pH and bicarbonate
        return false; // Placeholder
    }

    private boolean hasRenalDysfunction(PatientContextState state) {
        Double creatinine = getLabValue(state, "creatinine");
        return creatinine != null && creatinine > 1.5;
    }

    private boolean hasTraumaIndicators(PatientContextState state) {
        // Would check encounter type, mechanism of injury
        return false; // Placeholder
    }

    private boolean hasContraindication(String contraindication, PatientContextState state) {
        if (contraindication == null || state == null) {
            return false;
        }

        String lower = contraindication.toLowerCase();

        // Check pregnancy
        if (lower.contains("pregnancy")) {
            return isPregnant(state);
        }

        // Check allergies
        if (lower.contains("allergy") && state.getAllergies() != null) {
            return state.getAllergies().stream()
                    .anyMatch(allergy -> allergy.toLowerCase().contains(lower));
        }

        // Check conditions
        if (state.getChronicConditions() != null) {
            return state.getChronicConditions().stream()
                    .anyMatch(cond -> cond.getCode() != null &&
                            cond.getCode().toLowerCase().contains(lower));
        }

        return false;
    }

    private boolean hasContrastAllergy(PatientContextState state) {
        if (state.getAllergies() == null) {
            return false;
        }

        return state.getAllergies().stream()
                .anyMatch(allergy ->
                        allergy.toLowerCase().contains("iodine") ||
                        allergy.toLowerCase().contains("contrast") ||
                        allergy.toLowerCase().contains("gadolinium"));
    }

    private boolean isPregnant(PatientContextState state) {
        // Would check pregnancy status
        return false; // Placeholder
    }

    private boolean hasPacemaker(PatientContextState state) {
        // Would check device list
        return false; // Placeholder
    }

    private Double getVitalValue(PatientContextState state, String vitalName) {
        if (state == null || state.getLatestVitals() == null) {
            return null;
        }

        Object value = state.getLatestVitals().get(vitalName.toLowerCase());
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }
        return null;
    }

    private Double getLabValue(PatientContextState state, String labName) {
        if (state == null || state.getRecentLabs() == null) {
            return null;
        }

        com.cardiofit.flink.models.LabResult labResult = state.getRecentLabs().get(labName.toLowerCase());
        if (labResult != null && labResult.getValue() != null) {
            return labResult.getValue();
        }

        return null;
    }
}
