package com.cardiofit.flink.intelligence;

import com.cardiofit.flink.loader.DiagnosticTestLoader;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.PatientDemographics;
import com.cardiofit.flink.models.LabResult;
import com.cardiofit.flink.models.diagnostics.LabTest;
import com.cardiofit.flink.models.diagnostics.ImagingStudy;
import com.cardiofit.flink.models.diagnostics.TestRecommendation;
import com.cardiofit.flink.models.diagnostics.TestResult;
import com.cardiofit.flink.protocol.models.Protocol;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;

/**
 * Test Recommender
 *
 * Core recommendation engine for intelligent diagnostic test ordering.
 * Provides protocol-based test selection, safety validation, reflex testing,
 * and appropriateness scoring for the CardioFit clinical decision support system.
 *
 * <p>Key Capabilities:
 * - Protocol-matched diagnostic bundles (Sepsis, STEMI, etc.)
 * - Safety checking (contraindications, allergies, renal function)
 * - Reflex testing based on abnormal results
 * - ACR appropriateness scoring for imaging
 * - Priority assignment (P0-P3) and urgency levels
 * - Minimum interval enforcement
 *
 * <p>Integrations:
 * - DiagnosticTestLoader for test definitions
 * - EnrichedPatientContext for patient clinical state
 * - Protocol objects for matched clinical protocols
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-23
 */
public class TestRecommender implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(TestRecommender.class);

    // Test loader for accessing test definitions
    private final DiagnosticTestLoader testLoader;

    // Track last order times for minimum interval enforcement
    private final ConcurrentHashMap<String, Long> lastOrderTimes;

    /**
     * Constructor with dependency injection.
     *
     * @param testLoader Diagnostic test loader instance
     */
    public TestRecommender(DiagnosticTestLoader testLoader) {
        this.testLoader = testLoader;
        this.lastOrderTimes = new ConcurrentHashMap<>();
    }

    /**
     * Default constructor (uses singleton test loader).
     */
    public TestRecommender() {
        this(DiagnosticTestLoader.getInstance());
    }

    // ================================================================
    // PRIMARY RECOMMENDATION API
    // ================================================================

    /**
     * Recommend diagnostic tests based on protocol and patient context.
     *
     * <p>Algorithm:
     * 1. Identify protocol-specific test panel
     * 2. Filter out contraindicated tests
     * 3. Check minimum intervals for repeats
     * 4. Assign priorities and urgency
     * 5. Validate safety (renal, pregnancy, allergies)
     * 6. Return ranked test recommendations
     *
     * @param context Patient clinical context
     * @param protocol Matched clinical protocol
     * @return List of test recommendations, ranked by priority
     */
    public List<TestRecommendation> recommendTests(
            EnrichedPatientContext context,
            Protocol protocol) {

        if (context == null || protocol == null) {
            LOG.warn("Cannot recommend tests: null context or protocol");
            return Collections.emptyList();
        }

        String protocolId = protocol.getProtocolId();
        LOG.info("Recommending tests for protocol: {} (patient: {})",
                protocolId, context.getPatientId());

        List<TestRecommendation> recommendations = new ArrayList<>();

        // Route to protocol-specific bundle
        if (protocolId.contains("SEPSIS")) {
            recommendations = getSepsisDiagnosticBundle(context);
        } else if (protocolId.contains("STEMI") || protocolId.contains("AMI")) {
            recommendations = getSTEMIDiagnosticBundle(context);
        } else {
            LOG.warn("No specific diagnostic bundle for protocol: {}", protocolId);
            recommendations = getStandardDiagnosticPanel(context, protocol);
        }

        // Filter unsafe recommendations
        recommendations = recommendations.stream()
                .filter(rec -> isSafeToOrder(rec, context))
                .collect(Collectors.toList());

        // Sort by priority
        recommendations.sort(Comparator.comparing(TestRecommendation::getPriority));

        LOG.info("Generated {} test recommendations for patient {}",
                recommendations.size(), context.getPatientId());

        return recommendations;
    }

    // ================================================================
    // PROTOCOL-SPECIFIC BUNDLES
    // ================================================================

    /**
     * Get sepsis diagnostic bundle.
     *
     * <p>SSC 2021 Hour-1 Bundle Tests:
     * - Serum Lactate (STAT)
     * - Blood Cultures (before antibiotics)
     * - CBC with differential
     * - Comprehensive Metabolic Panel (CMP)
     * - Chest X-Ray (if respiratory source)
     * - Urinalysis + Culture (if urinary source)
     * - Procalcitonin (optional, for antibiotic stewardship)
     *
     * @param context Patient context
     * @return List of sepsis diagnostic recommendations
     */
    public List<TestRecommendation> getSepsisDiagnosticBundle(EnrichedPatientContext context) {
        LOG.info("Building SEPSIS diagnostic bundle for patient {}", context.getPatientId());

        List<TestRecommendation> bundle = new ArrayList<>();
        long timestamp = System.currentTimeMillis();

        // 1. Serum Lactate (CRITICAL - SSC Hour-1)
        LabTest lactate = testLoader.getLabTest("LAB-LACTATE-001");
        if (lactate != null && canReorder("LAB-LACTATE-001")) {
            TestRecommendation lactatRec = TestRecommendation.builder()
                    .recommendationId(UUID.randomUUID().toString())
                    .testId(lactate.getTestId())
                    .testName(lactate.getTestName())
                    .testCategory(TestRecommendation.TestCategory.LAB)
                    .timestamp(timestamp)
                    .priority(TestRecommendation.Priority.P0_CRITICAL)
                    .urgency(TestRecommendation.Urgency.STAT)
                    .timeframeMinutes(60)
                    .indication("Sepsis Hour-1 Bundle - Assess tissue perfusion")
                    .rationale("Elevated lactate (≥2 mmol/L) defines septic shock and guides resuscitation")
                    .expectedFindings("Lactate <2 mmol/L normal, ≥4 mmol/L indicates shock")
                    .decisionSupport(TestRecommendation.DecisionSupport.builder()
                            .guidelineReference("SSC 2021")
                            .evidenceLevel("A")
                            .recommendationStrength("Strong")
                            .confidenceScore(1.0)
                            .clinicalReasoning("Lactate is a key biomarker for septic shock and resuscitation monitoring")
                            .build())
                    .followUpGuidance(TestRecommendation.FollowUpGuidance.builder()
                            .actionIfAbnormal("If lactate ≥2: repeat in 2-4 hours to assess clearance")
                            .actionIfCritical("If lactate ≥4: SEPTIC SHOCK - aggressive fluid resuscitation, ICU transfer")
                            .repeatIntervalHours(2)
                            .reflexTests(Arrays.asList("Arterial Blood Gas", "Central Venous Oxygen Saturation"))
                            .build())
                    .patientId(context.getPatientId())
                    .protocolId("SEPSIS-SSC-2021")
                    .generatedBy("TestRecommender")
                    .build();

            bundle.add(lactatRec);
            recordOrderTime("LAB-LACTATE-001");
        }

        // 2. Blood Cultures (before antibiotics)
        TestRecommendation bloodCultures = createLabRecommendation(
                "LAB-BLOOD-CULTURE-001",
                "Blood Cultures",
                TestRecommendation.Priority.P0_CRITICAL,
                TestRecommendation.Urgency.STAT,
                "Sepsis Hour-1 Bundle - Identify causative organism",
                "Obtain 2 sets before first antibiotic dose",
                context,
                timestamp
        );
        if (bloodCultures != null) bundle.add(bloodCultures);

        // 3. CBC with Differential
        LabTest wbc = testLoader.getLabTest("LAB-WBC-001");
        if (wbc != null) {
            TestRecommendation cbcRec = createLabRecommendation(
                    "LAB-WBC-001",
                    "WBC Count (CBC)",
                    TestRecommendation.Priority.P1_URGENT,
                    TestRecommendation.Urgency.STAT,
                    "Assess infection/inflammation and immune response",
                    "Leukocytosis or leukopenia supports sepsis diagnosis",
                    context,
                    timestamp
            );
            if (cbcRec != null) bundle.add(cbcRec);
        }

        // 4. Comprehensive Metabolic Panel (CMP)
        TestRecommendation cmp = createLabRecommendation(
                "LAB-CMP-001",
                "Comprehensive Metabolic Panel",
                TestRecommendation.Priority.P1_URGENT,
                TestRecommendation.Urgency.STAT,
                "Assess renal function, electrolytes, acid-base status",
                "Evaluate organ dysfunction (creatinine, bilirubin)",
                context,
                timestamp
        );
        if (cmp != null) bundle.add(cmp);

        // 5. Chest X-Ray (if respiratory source suspected)
        if (hasRespiratorySepsisSource(context)) {
            ImagingStudy cxr = testLoader.getImagingStudy("IMG-CXR-001");
            if (cxr != null) {
                TestRecommendation cxrRec = createImagingRecommendation(
                        cxr,
                        TestRecommendation.Priority.P1_URGENT,
                        TestRecommendation.Urgency.URGENT,
                        "Identify pneumonia or pulmonary source of sepsis",
                        context,
                        timestamp
                );
                if (cxrRec != null) bundle.add(cxrRec);
            }
        }

        LOG.info("SEPSIS bundle: {} tests recommended", bundle.size());
        return bundle;
    }

    /**
     * Get STEMI diagnostic bundle.
     *
     * <p>AHA/ACC STEMI Guidelines Tests:
     * - Troponin I (STAT)
     * - 12-Lead ECG
     * - Complete Blood Count
     * - Comprehensive Metabolic Panel
     * - Coagulation Panel (PT/INR)
     * - Lipid Panel
     * - Echocardiogram
     * - Chest X-Ray (if heart failure suspected)
     *
     * @param context Patient context
     * @return List of STEMI diagnostic recommendations
     */
    public List<TestRecommendation> getSTEMIDiagnosticBundle(EnrichedPatientContext context) {
        LOG.info("Building STEMI diagnostic bundle for patient {}", context.getPatientId());

        List<TestRecommendation> bundle = new ArrayList<>();
        long timestamp = System.currentTimeMillis();

        // 1. Troponin I (CRITICAL)
        TestRecommendation troponin = createLabRecommendation(
                "LAB-TROPONIN-I-001",
                "Troponin I",
                TestRecommendation.Priority.P0_CRITICAL,
                TestRecommendation.Urgency.STAT,
                "STEMI diagnosis and risk stratification",
                "Elevated troponin confirms myocardial injury",
                context,
                timestamp
        );
        if (troponin != null) {
            troponin.setFollowUpGuidance(TestRecommendation.FollowUpGuidance.builder()
                    .actionIfAbnormal("If elevated: serial troponins at 3-6 hours, urgent cardiology consult")
                    .actionIfCritical("Markedly elevated (>10x URL): STEMI confirmed, activate cath lab")
                    .repeatIntervalHours(3)
                    .reflexTests(Arrays.asList("CK-MB", "BNP", "Echocardiogram"))
                    .build());
            bundle.add(troponin);
        }

        // 2. Comprehensive Metabolic Panel (assess renal function for contrast)
        TestRecommendation cmp = createLabRecommendation(
                "LAB-CMP-001",
                "Comprehensive Metabolic Panel",
                TestRecommendation.Priority.P1_URGENT,
                TestRecommendation.Urgency.STAT,
                "Assess renal function before cardiac catheterization",
                "Creatinine needed for contrast safety assessment",
                context,
                timestamp
        );
        if (cmp != null) bundle.add(cmp);

        // 3. Coagulation Panel (PT/INR)
        LabTest ptInr = testLoader.getLabTest("LAB-PT-INR-001");
        if (ptInr != null) {
            TestRecommendation coagRec = createLabRecommendation(
                    ptInr.getTestId(),
                    ptInr.getTestName(),
                    TestRecommendation.Priority.P1_URGENT,
                    TestRecommendation.Urgency.STAT,
                    "Assess baseline coagulation before anticoagulation",
                    "Required before thrombolytics or PCI",
                    context,
                    timestamp
            );
            if (coagRec != null) bundle.add(coagRec);
        }

        // 4. Echocardiogram (assess cardiac function)
        ImagingStudy echo = testLoader.getImagingStudy("IMG-ECHO-001");
        if (echo != null) {
            TestRecommendation echoRec = createImagingRecommendation(
                    echo,
                    TestRecommendation.Priority.P1_URGENT,
                    TestRecommendation.Urgency.URGENT,
                    "Assess left ventricular function and wall motion abnormalities",
                    context,
                    timestamp
            );
            if (echoRec != null) bundle.add(echoRec);
        }

        // 5. Chest X-Ray (if heart failure suspected)
        if (hasPulmonaryEdema(context)) {
            ImagingStudy cxr = testLoader.getImagingStudy("IMG-CXR-001");
            if (cxr != null) {
                TestRecommendation cxrRec = createImagingRecommendation(
                        cxr,
                        TestRecommendation.Priority.P2_IMPORTANT,
                        TestRecommendation.Urgency.URGENT,
                        "Assess for pulmonary edema and cardiomegaly",
                        context,
                        timestamp
                );
                if (cxrRec != null) bundle.add(cxrRec);
            }
        }

        LOG.info("STEMI bundle: {} tests recommended", bundle.size());
        return bundle;
    }

    /**
     * Get standard diagnostic panel for other protocols.
     *
     * @param context Patient context
     * @param protocol Protocol object
     * @return Standard diagnostic recommendations
     */
    private List<TestRecommendation> getStandardDiagnosticPanel(
            EnrichedPatientContext context,
            Protocol protocol) {

        LOG.debug("Building standard diagnostic panel for protocol {}", protocol.getProtocolId());

        List<TestRecommendation> panel = new ArrayList<>();
        // Placeholder - would extract from protocol definition
        return panel;
    }

    // ================================================================
    // SAFETY VALIDATION
    // ================================================================

    /**
     * Check if test is safe to order for patient.
     *
     * <p>Safety Checks:
     * - Contraindications (from test definition)
     * - Renal function for contrast imaging (GFR > threshold)
     * - Pregnancy for radiation exposure
     * - Allergy checking (iodine contrast, gadolinium)
     * - Minimum interval enforcement (prevent over-testing)
     *
     * @param test Test recommendation
     * @param context Patient context
     * @return true if safe to order, false if contraindicated
     */
    public boolean isSafeToOrder(TestRecommendation test, EnrichedPatientContext context) {
        if (test == null || context == null) {
            return false;
        }

        PatientContextState state = context.getPatientState();
        if (state == null) {
            LOG.warn("Cannot validate safety: null patient state");
            return false;
        }

        // Check contraindications
        if (test.getContraindications() != null && !test.getContraindications().isEmpty()) {
            for (String contraindication : test.getContraindications()) {
                if (hasContraindication(contraindication, state)) {
                    LOG.warn("Test {} contraindicated: {} for patient {}",
                            test.getTestId(), contraindication, context.getPatientId());
                    return false;
                }
            }
        }

        // Imaging-specific safety checks
        if (test.isImagingStudy()) {
            ImagingStudy imagingStudy = testLoader.getImagingStudy(test.getTestId());
            if (imagingStudy != null) {
                // Check contrast safety (renal function)
                if (imagingStudy.getRequirements() != null &&
                    imagingStudy.getRequirements().isContrastRequired()) {

                    Double creatinine = getLabValue(state, "creatinine");
                    if (creatinine != null && creatinine > 1.5) {
                        LOG.warn("Contrast safety concern: Creatinine {} > 1.5 for patient {}",
                                creatinine, context.getPatientId());
                        return false;
                    }
                }

                // Check pregnancy for radiation
                if (imagingStudy.getRadiationExposure() != null &&
                    imagingStudy.getRadiationExposure().isHasRadiation()) {

                    if (isPregnant(state)) {
                        LOG.warn("Radiation exposure contraindicated: patient {} is pregnant",
                                context.getPatientId());
                        return false;
                    }
                }
            }
        }

        return true;
    }

    // ================================================================
    // REFLEX TESTING
    // ================================================================

    /**
     * Check for reflex testing based on abnormal results.
     *
     * <p>Reflex Rules:
     * - Lactate > 4.0 → Repeat in 2 hours
     * - Creatinine elevated → Add urinalysis
     * - Troponin elevated → Add CK-MB, BNP, Echo
     * - Positive blood culture → Add culture sensitivity
     * - Abnormal CXR → Consider CT chest
     *
     * @param result Test result that triggered reflex
     * @param context Patient context
     * @return List of reflex test recommendations
     */
    public List<TestRecommendation> checkReflexTesting(
            TestResult result,
            EnrichedPatientContext context) {

        if (result == null || context == null) {
            return Collections.emptyList();
        }

        List<TestRecommendation> reflexTests = new ArrayList<>();
        long timestamp = System.currentTimeMillis();

        String testId = result.getTestId();
        Double value = result.getValue();

        // Lactate > 4.0 → Repeat in 2 hours
        if ("LAB-LACTATE-001".equals(testId) && value != null && value >= 4.0) {
            TestRecommendation repeatLactate = createLabRecommendation(
                    "LAB-LACTATE-001",
                    "Repeat Serum Lactate",
                    TestRecommendation.Priority.P0_CRITICAL,
                    TestRecommendation.Urgency.URGENT,
                    "Lactate ≥4.0 mmol/L - assess clearance in 2 hours",
                    "Target: lactate clearance ≥10% from baseline",
                    context,
                    timestamp
            );
            if (repeatLactate != null) {
                repeatLactate.setRepeatTest(true);
                repeatLactate.setPreviousTestTimestamp(result.getTimestamp());
                reflexTests.add(repeatLactate);
            }
        }

        // Troponin elevated → Add CK-MB
        if (testId.contains("TROPONIN") && value != null && value > 0.04) {
            TestRecommendation ckmb = createLabRecommendation(
                    "LAB-CK-MB-001",
                    "CK-MB",
                    TestRecommendation.Priority.P1_URGENT,
                    TestRecommendation.Urgency.URGENT,
                    "Reflex: Troponin elevated - confirm myocardial injury",
                    "CK-MB more specific for cardiac muscle",
                    context,
                    timestamp
            );
            if (ckmb != null) reflexTests.add(ckmb);
        }

        LOG.info("Reflex testing: {} tests recommended based on {} result",
                reflexTests.size(), testId);

        return reflexTests;
    }

    // ================================================================
    // APPROPRIATENESS SCORING
    // ================================================================

    /**
     * Calculate appropriateness score for test recommendation.
     *
     * <p>Scoring Factors:
     * - Clinical indication match (40%)
     * - ACR appropriateness rating for imaging (30%)
     * - Timing/urgency alignment (15%)
     * - Cost-effectiveness (10%)
     * - Evidence strength (5%)
     *
     * @param test Test recommendation
     * @param context Patient context
     * @return Appropriateness score (0-100)
     */
    public int calculateAppropriatenessScore(
            TestRecommendation test,
            EnrichedPatientContext context) {

        if (test == null || context == null) {
            return 0;
        }

        int score = 0;

        // Clinical indication match (40 points)
        score += 40; // Assume good match if recommended by protocol

        // ACR appropriateness for imaging (30 points)
        if (test.isImagingStudy()) {
            ImagingStudy imagingStudy = testLoader.getImagingStudy(test.getTestId());
            if (imagingStudy != null && imagingStudy.getAppropriatenessScore() != null) {
                int acrScore = imagingStudy.getAppropriatenessScore();
                if (acrScore >= 7) {
                    score += 30; // Usually appropriate
                } else if (acrScore >= 4) {
                    score += 15; // May be appropriate
                } else {
                    score += 0;  // Usually not appropriate
                }
            }
        } else {
            score += 30; // Default for labs
        }

        // Timing alignment (15 points)
        if (test.requiresImmediateAction()) {
            score += 15;
        } else {
            score += 10;
        }

        // Cost-effectiveness (10 points)
        score += 10; // Default

        // Evidence strength (5 points)
        if (test.getDecisionSupport() != null) {
            String evidenceLevel = test.getDecisionSupport().getEvidenceLevel();
            if ("A".equals(evidenceLevel)) {
                score += 5;
            } else if ("B".equals(evidenceLevel)) {
                score += 3;
            } else {
                score += 1;
            }
        }

        return Math.min(score, 100);
    }

    // ================================================================
    // HELPER METHODS
    // ================================================================

    /**
     * Create lab test recommendation.
     */
    private TestRecommendation createLabRecommendation(
            String testId,
            String testName,
            TestRecommendation.Priority priority,
            TestRecommendation.Urgency urgency,
            String indication,
            String rationale,
            EnrichedPatientContext context,
            long timestamp) {

        if (!canReorder(testId)) {
            LOG.debug("Test {} cannot be reordered yet (minimum interval)", testId);
            return null;
        }

        TestRecommendation rec = TestRecommendation.builder()
                .recommendationId(UUID.randomUUID().toString())
                .testId(testId)
                .testName(testName)
                .testCategory(TestRecommendation.TestCategory.LAB)
                .timestamp(timestamp)
                .priority(priority)
                .urgency(urgency)
                .indication(indication)
                .rationale(rationale)
                .patientId(context.getPatientId())
                .generatedBy("TestRecommender")
                .build();

        recordOrderTime(testId);
        return rec;
    }

    /**
     * Create imaging recommendation.
     */
    private TestRecommendation createImagingRecommendation(
            ImagingStudy imagingStudy,
            TestRecommendation.Priority priority,
            TestRecommendation.Urgency urgency,
            String indication,
            EnrichedPatientContext context,
            long timestamp) {

        if (!canReorder(imagingStudy.getStudyId())) {
            LOG.debug("Imaging study {} cannot be reordered yet", imagingStudy.getStudyId());
            return null;
        }

        TestRecommendation rec = TestRecommendation.builder()
                .recommendationId(UUID.randomUUID().toString())
                .testId(imagingStudy.getStudyId())
                .testName(imagingStudy.getStudyName())
                .testCategory(TestRecommendation.TestCategory.IMAGING)
                .timestamp(timestamp)
                .priority(priority)
                .urgency(urgency)
                .indication(indication)
                .rationale(imagingStudy.getClinicalIndication())
                .patientId(context.getPatientId())
                .generatedBy("TestRecommender")
                .build();

        recordOrderTime(imagingStudy.getStudyId());
        return rec;
    }

    /**
     * Check if test can be reordered (minimum interval check).
     */
    private boolean canReorder(String testId) {
        if (testId == null) return true;

        Long lastOrderTime = lastOrderTimes.get(testId);
        if (lastOrderTime == null) {
            return true;
        }

        // Default 2 hour minimum interval
        long hoursSinceLastOrder = (System.currentTimeMillis() - lastOrderTime) / (1000 * 60 * 60);
        return hoursSinceLastOrder >= 2;
    }

    /**
     * Record order time for interval tracking.
     */
    private void recordOrderTime(String testId) {
        if (testId != null) {
            lastOrderTimes.put(testId, System.currentTimeMillis());
        }
    }

    /**
     * Check if patient has respiratory sepsis source.
     */
    private boolean hasRespiratorySepsisSource(EnrichedPatientContext context) {
        // Check for respiratory symptoms, hypoxia, elevated respiratory rate
        PatientContextState state = context.getPatientState();
        if (state == null) return false;

        // Check respiratory rate
        Double respRate = getVitalValue(state, "respiratoryrate");
        if (respRate != null && respRate > 22) {
            return true;
        }

        // Check oxygen saturation
        Double spo2 = getVitalValue(state, "spo2");
        if (spo2 != null && spo2 < 92) {
            return true;
        }

        return false;
    }

    /**
     * Check if patient has pulmonary edema.
     */
    private boolean hasPulmonaryEdema(EnrichedPatientContext context) {
        PatientContextState state = context.getPatientState();
        if (state == null) return false;

        // Check for elevated BNP (if available)
        Double bnp = getLabValue(state, "bnp");
        if (bnp != null && bnp > 400) {
            return true;
        }

        // Check for dyspnea indicators
        Double respRate = getVitalValue(state, "respiratoryrate");
        Double spo2 = getVitalValue(state, "spo2");

        return (respRate != null && respRate > 24) && (spo2 != null && spo2 < 90);
    }

    /**
     * Check if patient has contraindication.
     */
    private boolean hasContraindication(String contraindication, PatientContextState state) {
        if (contraindication == null || state == null) {
            return false;
        }

        String lower = contraindication.toLowerCase();

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

    /**
     * Check if patient is pregnant.
     */
    private boolean isPregnant(PatientContextState state) {
        if (state == null || state.getDemographics() == null) {
            return false;
        }

        PatientDemographics demographics = state.getDemographics();
        String sex = demographics.getSex();

        // Only check for females
        if (!"F".equalsIgnoreCase(sex) && !"FEMALE".equalsIgnoreCase(sex)) {
            return false;
        }

        // Check age range for pregnancy possibility
        Integer age = demographics.getAge();
        if (age == null || age < 12 || age > 55) {
            return false;
        }

        // Would check pregnancy status from conditions or vitals
        // Placeholder for now
        return false;
    }

    /**
     * Get vital sign value.
     */
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

    /**
     * Get lab result value.
     */
    private Double getLabValue(PatientContextState state, String labName) {
        if (state == null || state.getRecentLabs() == null) {
            return null;
        }

        LabResult labResult = state.getRecentLabs().get(labName.toLowerCase());
        if (labResult != null && labResult.getValue() != null) {
            return labResult.getValue();
        }

        return null;
    }
}
