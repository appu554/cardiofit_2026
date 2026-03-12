package com.cardiofit.flink.examples;

import com.cardiofit.flink.loader.DiagnosticTestLoader;
import com.cardiofit.flink.recommender.TestRecommender;
import com.cardiofit.flink.rules.TestOrderingRules;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.PatientDemographics;
import com.cardiofit.flink.models.LabResult;
import com.cardiofit.flink.models.diagnostics.LabTest;
import com.cardiofit.flink.models.diagnostics.ImagingStudy;
import com.cardiofit.flink.models.diagnostics.TestRecommendation;
import com.cardiofit.flink.models.diagnostics.TestResult;
import com.cardiofit.flink.protocol.models.Protocol;

import java.util.*;

/**
 * Test Recommender Example
 *
 * Demonstrates usage of the Phase 4 Intelligence Layer components:
 * - DiagnosticTestLoader: Loading YAML test definitions
 * - TestRecommender: Intelligent test recommendation engine
 * - TestOrderingRules: Complex ordering logic and validation
 *
 * <p>Example Scenario: ROHAN - 67-year-old male with sepsis
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-23
 */
public class TestRecommenderExample {

    public static void main(String[] args) {
        System.out.println("=".repeat(80));
        System.out.println("TEST RECOMMENDER EXAMPLE - ROHAN SEPSIS CASE");
        System.out.println("=".repeat(80));
        System.out.println();

        // ================================================================
        // STEP 1: Initialize Components
        // ================================================================
        System.out.println("STEP 1: Initializing Phase 4 Intelligence Layer...");
        System.out.println("-".repeat(80));

        DiagnosticTestLoader testLoader = DiagnosticTestLoader.getInstance();
        TestRecommender recommender = new TestRecommender(testLoader);
        TestOrderingRules orderingRules = new TestOrderingRules(testLoader);

        System.out.println("✓ DiagnosticTestLoader initialized");
        System.out.println("  - Lab tests loaded: " + testLoader.getAllLabTests().size());
        System.out.println("  - Imaging studies loaded: " + testLoader.getAllImagingStudies().size());
        System.out.println("✓ TestRecommender initialized");
        System.out.println("✓ TestOrderingRules initialized");
        System.out.println();

        // ================================================================
        // STEP 2: Create Patient Context (ROHAN - Sepsis Patient)
        // ================================================================
        System.out.println("STEP 2: Creating patient context for ROHAN...");
        System.out.println("-".repeat(80));

        EnrichedPatientContext rohanContext = createRohanSepsisContext();

        System.out.println("Patient: ROHAN (67M)");
        System.out.println("  - Vital Signs:");
        System.out.println("    * Temperature: 39.2°C (fever)");
        System.out.println("    * Heart Rate: 118 bpm (tachycardia)");
        System.out.println("    * Systolic BP: 88 mmHg (hypotension)");
        System.out.println("    * Respiratory Rate: 28/min (tachypnea)");
        System.out.println("    * SpO2: 91% (hypoxia)");
        System.out.println("  - Clinical Presentation: Suspected sepsis with respiratory source");
        System.out.println();

        // ================================================================
        // STEP 3: Load Specific Test Definitions
        // ================================================================
        System.out.println("STEP 3: Loading specific test definitions...");
        System.out.println("-".repeat(80));

        LabTest lactate = testLoader.getLabTest("LAB-LACTATE-001");
        LabTest wbc = testLoader.getLabTestByLoinc("6690-2"); // WBC LOINC code
        ImagingStudy cxr = testLoader.getImagingStudy("IMG-CXR-001");

        if (lactate != null) {
            System.out.println("✓ Lactate Test Loaded:");
            System.out.println("  - Test Name: " + lactate.getTestName());
            System.out.println("  - LOINC Code: " + lactate.getLoincCode());
            System.out.println("  - TAT (STAT): " + lactate.getTiming().getUrgentTurnaroundMinutes() + " min");
            System.out.println("  - Critical High: " + lactate.getReferenceRanges().get("adult")
                    .getCritical().getCriticalHigh() + " mmol/L");
        }

        if (cxr != null) {
            System.out.println("✓ Chest X-Ray Loaded:");
            System.out.println("  - Study Name: " + cxr.getStudyName());
            System.out.println("  - CPT Code: " + cxr.getCptCode());
            System.out.println("  - Radiation Dose: " + cxr.getRadiationExposure().getEffectiveDose());
        }
        System.out.println();

        // ================================================================
        // STEP 4: Generate Test Recommendations
        // ================================================================
        System.out.println("STEP 4: Generating test recommendations...");
        System.out.println("-".repeat(80));

        Protocol sepsisProtocol = createSepsisProtocol();
        List<TestRecommendation> recommendations = recommender.recommendTests(
                rohanContext, sepsisProtocol);

        System.out.println("Generated " + recommendations.size() + " test recommendations:");
        System.out.println();

        for (int i = 0; i < recommendations.size(); i++) {
            TestRecommendation rec = recommendations.get(i);
            System.out.println((i + 1) + ". " + rec.getTestName());
            System.out.println("   Priority: " + rec.getPriority() + " | Urgency: " + rec.getUrgency());
            System.out.println("   Indication: " + rec.getIndication());
            System.out.println("   Rationale: " + rec.getRationale());

            if (rec.getDecisionSupport() != null) {
                System.out.println("   Evidence Level: " + rec.getDecisionSupport().getEvidenceLevel());
                System.out.println("   Guideline: " + rec.getDecisionSupport().getGuidelineReference());
            }

            // Safety validation
            boolean safe = recommender.isSafeToOrder(rec, rohanContext);
            System.out.println("   Safety Check: " + (safe ? "✓ SAFE" : "✗ CONTRAINDICATED"));

            // Appropriateness score
            int appropriateness = recommender.calculateAppropriatenessScore(rec, rohanContext);
            System.out.println("   Appropriateness Score: " + appropriateness + "/100");

            System.out.println();
        }

        // ================================================================
        // STEP 5: Check Ordering Rules
        // ================================================================
        System.out.println("STEP 5: Validating ordering rules...");
        System.out.println("-".repeat(80));

        if (lactate != null) {
            boolean meetsIndications = orderingRules.meetsIndications(lactate, rohanContext);
            System.out.println("Lactate Test - Meets Clinical Indications: " +
                    (meetsIndications ? "✓ YES" : "✗ NO"));

            List<String> contraindications = orderingRules.checkContraindications(
                    recommendations.get(0), rohanContext);
            System.out.println("Lactate Test - Contraindications: " +
                    (contraindications.isEmpty() ? "None" : contraindications));
        }
        System.out.println();

        // ================================================================
        // STEP 6: Auto-Ordering Bundles
        // ================================================================
        System.out.println("STEP 6: Checking auto-order bundles...");
        System.out.println("-".repeat(80));

        List<TestRecommendation> lactatBundle = orderingRules.getAutoOrderBundle("LAB-LACTATE-001");
        System.out.println("Auto-order bundle for Lactate:");
        for (TestRecommendation bundledTest : lactatBundle) {
            System.out.println("  - " + bundledTest.getTestName() + " (" + bundledTest.getPriority() + ")");
        }
        System.out.println();

        // ================================================================
        // STEP 7: Simulate Result and Reflex Testing
        // ================================================================
        System.out.println("STEP 7: Simulating test result and reflex testing...");
        System.out.println("-".repeat(80));

        TestResult lactateResult = TestResult.builder()
                .resultId(UUID.randomUUID().toString())
                .testId("LAB-LACTATE-001")
                .testName("Serum Lactate")
                .patientId("ROHAN-12345")
                .testType(TestResult.TestType.LAB)
                .numericValue(4.8) // Critically elevated
                .resultUnit("mmol/L")
                .timestamp(System.currentTimeMillis())
                .resultInterpretation(TestResult.ResultInterpretation.CRITICAL_HIGH)
                .isCritical(true)
                .status(TestResult.ResultStatus.FINAL)
                .build();

        System.out.println("Lactate Result: " + lactateResult.getNumericValue() + " " + lactateResult.getResultUnit());
        System.out.println("Interpretation: " + lactateResult.getResultInterpretation());
        System.out.println("Critical: " + (lactateResult.isCritical() ? "✗ YES - IMMEDIATE ACTION REQUIRED" : "No"));
        System.out.println();

        List<TestRecommendation> reflexTests = recommender.checkReflexTesting(
                lactateResult, rohanContext);

        System.out.println("Reflex Tests Triggered: " + reflexTests.size());
        for (TestRecommendation reflexTest : reflexTests) {
            System.out.println("  - " + reflexTest.getTestName());
            System.out.println("    Reason: " + reflexTest.getRationale());
            System.out.println("    Timeframe: " + reflexTest.getTimeframeMinutes() + " minutes");
        }
        System.out.println();

        // ================================================================
        // SUMMARY
        // ================================================================
        System.out.println("=".repeat(80));
        System.out.println("SUMMARY");
        System.out.println("=".repeat(80));
        System.out.println("Phase 4 Intelligence Layer successfully demonstrated:");
        System.out.println("✓ DiagnosticTestLoader loaded " + testLoader.getAllLabTests().size() + " lab tests");
        System.out.println("✓ TestRecommender generated " + recommendations.size() + " protocol-based recommendations");
        System.out.println("✓ TestOrderingRules validated clinical indications and safety");
        System.out.println("✓ Auto-ordering bundles identified " + lactatBundle.size() + " bundled tests");
        System.out.println("✓ Reflex testing triggered " + reflexTests.size() + " follow-up tests");
        System.out.println();
        System.out.println("Clinical Decision Support: READY FOR INTEGRATION");
        System.out.println("=".repeat(80));
    }

    /**
     * Create ROHAN's patient context (sepsis patient).
     */
    private static EnrichedPatientContext createRohanSepsisContext() {
        // Demographics
        PatientDemographics demographics = new PatientDemographics();
        demographics.setPatientId("ROHAN-12345");
        demographics.setAge(67);
        demographics.setSex("M");
        demographics.setWeight(85.0); // kg

        // Vital signs
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("temperature", 39.2);       // Celsius - fever
        vitals.put("heartrate", 118.0);        // bpm - tachycardia
        vitals.put("systolicbp", 88.0);        // mmHg - hypotension
        vitals.put("diastolicbp", 55.0);       // mmHg
        vitals.put("respiratoryrate", 28.0);   // /min - tachypnea
        vitals.put("spo2", 91.0);              // % - hypoxia

        // Lab results
        Map<String, LabResult> labs = new HashMap<>();
        labs.put("wbc", LabResult.builder()
                .name("WBC")
                .value(18.5)  // Leukocytosis
                .unit("K/uL")
                .build());
        labs.put("creatinine", LabResult.builder()
                .name("Creatinine")
                .value(1.2)   // Normal
                .unit("mg/dL")
                .build());

        // Patient state
        PatientContextState state = new PatientContextState();
        state.setDemographics(demographics);
        state.setLatestVitals(vitals);
        state.setRecentLabs(labs);
        state.setNews2Score(9);      // High NEWS2 score
        state.setQsofaScore(2);      // qSOFA positive

        // Enriched context
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId("ROHAN-12345");
        context.setPatientState(state);
        context.setEventType("VITAL_SIGN");
        context.setEventTime(System.currentTimeMillis());

        return context;
    }

    /**
     * Create sepsis protocol.
     */
    private static Protocol createSepsisProtocol() {
        Protocol protocol = new Protocol();
        protocol.setProtocolId("SEPSIS-SSC-2021");
        protocol.setName("Surviving Sepsis Campaign 2021");
        protocol.setCategory("INFECTIOUS_DISEASE");
        protocol.setSpecialty("CRITICAL_CARE");
        protocol.setVersion("2021.1");
        return protocol;
    }
}
