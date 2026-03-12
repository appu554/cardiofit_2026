import com.cardiofit.flink.ml.*;
import com.cardiofit.flink.ml.features.ClinicalFeatureExtractor;
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import com.cardiofit.flink.ml.util.TestDataFactory;
import com.cardiofit.flink.ml.PatientContextSnapshot;
import com.cardiofit.flink.models.MLPrediction;

import java.util.Arrays;
import java.util.Scanner;

/**
 * Manual ML Testing Tool
 *
 * Run this to test ML predictions with custom patient data.
 *
 * Usage:
 *   javac -cp "target/classes:..." ManualMLTest.java
 *   java -cp ".:target/classes:..." ManualMLTest
 */
public class ManualMLTest {

    private static ONNXModelContainer sepsisModel;
    private static ONNXModelContainer deteriorationModel;
    private static ONNXModelContainer mortalityModel;
    private static ONNXModelContainer readmissionModel;
    private static ClinicalFeatureExtractor extractor;

    public static void main(String[] args) throws Exception {
        System.out.println("╔════════════════════════════════════════════════════════════════╗");
        System.out.println("║     CardioFit ML Inference - Manual Testing Tool              ║");
        System.out.println("║     Module 5: Clinical Risk Prediction                        ║");
        System.out.println("╚════════════════════════════════════════════════════════════════╝");
        System.out.println();

        // Load models
        loadModels();

        // Initialize extractor
        extractor = new ClinicalFeatureExtractor();

        // Run interactive test
        runInteractiveTest();
    }

    private static void loadModels() throws Exception {
        System.out.println("📦 Loading ONNX Models...");

        // Sepsis Model
        ModelConfig sepsisConfig = ModelConfig.builder()
            .modelPath("models/sepsis_risk_v1.0.0.onnx")
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.7)
            .build();

        sepsisModel = ONNXModelContainer.builder()
            .modelId("sepsis_v1")
            .modelName("Sepsis Risk Predictor")
            .modelType(ONNXModelContainer.ModelType.SEPSIS_ONSET)
            .modelVersion("1.0.0")
            .inputFeatureNames(TestDataFactory.createFeatureNames())
            .outputNames(Arrays.asList("sepsis_probability"))
            .config(sepsisConfig)
            .build();
        sepsisModel.initialize();
        System.out.println("   ✅ Sepsis model loaded");

        // Deterioration Model
        ModelConfig detConfig = ModelConfig.builder()
            .modelPath("models/deterioration_risk_v1.0.0.onnx")
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.7)
            .build();

        deteriorationModel = ONNXModelContainer.builder()
            .modelId("deterioration_v1")
            .modelName("Clinical Deterioration Predictor")
            .modelType(ONNXModelContainer.ModelType.CLINICAL_DETERIORATION)
            .modelVersion("1.0.0")
            .inputFeatureNames(TestDataFactory.createFeatureNames())
            .outputNames(Arrays.asList("deterioration_probability"))
            .config(detConfig)
            .build();
        deteriorationModel.initialize();
        System.out.println("   ✅ Deterioration model loaded");

        // Mortality Model
        ModelConfig mortConfig = ModelConfig.builder()
            .modelPath("models/mortality_risk_v1.0.0.onnx")
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.7)
            .build();

        mortalityModel = ONNXModelContainer.builder()
            .modelId("mortality_v1")
            .modelName("30-Day Mortality Predictor")
            .modelType(ONNXModelContainer.ModelType.MORTALITY_PREDICTION)
            .modelVersion("1.0.0")
            .inputFeatureNames(TestDataFactory.createFeatureNames())
            .outputNames(Arrays.asList("mortality_probability"))
            .config(mortConfig)
            .build();
        mortalityModel.initialize();
        System.out.println("   ✅ Mortality model loaded");

        // Readmission Model
        ModelConfig readmConfig = ModelConfig.builder()
            .modelPath("models/readmission_risk_v1.0.0.onnx")
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.7)
            .build();

        readmissionModel = ONNXModelContainer.builder()
            .modelId("readmission_v1")
            .modelName("30-Day Readmission Predictor")
            .modelType(ONNXModelContainer.ModelType.READMISSION_RISK)
            .modelVersion("1.0.0")
            .inputFeatureNames(TestDataFactory.createFeatureNames())
            .outputNames(Arrays.asList("readmission_probability"))
            .config(readmConfig)
            .build();
        readmissionModel.initialize();
        System.out.println("   ✅ Readmission model loaded");

        System.out.println("✅ All 4 models loaded successfully!\n");
    }

    private static void runInteractiveTest() throws Exception {
        Scanner scanner = new Scanner(System.in);

        while (true) {
            System.out.println("\n════════════════════════════════════════════════════════════════");
            System.out.println("Choose Test Scenario:");
            System.out.println("════════════════════════════════════════════════════════════════");
            System.out.println("1. High-Risk Sepsis Patient (preset)");
            System.out.println("2. Low-Risk Stable Patient (preset)");
            System.out.println("3. Enter Custom Patient Data");
            System.out.println("4. Quick Test (both high/low risk)");
            System.out.println("5. Exit");
            System.out.println("════════════════════════════════════════════════════════════════");
            System.out.print("Enter choice (1-5): ");

            String choice = scanner.nextLine().trim();

            switch (choice) {
                case "1":
                    testHighRiskPatient();
                    break;
                case "2":
                    testLowRiskPatient();
                    break;
                case "3":
                    testCustomPatient(scanner);
                    break;
                case "4":
                    testQuickComparison();
                    break;
                case "5":
                    System.out.println("\n👋 Goodbye!");
                    return;
                default:
                    System.out.println("❌ Invalid choice. Please enter 1-5.");
            }
        }
    }

    private static void testHighRiskPatient() throws Exception {
        System.out.println("\n╔════════════════════════════════════════════════════════════════╗");
        System.out.println("║  TEST 1: High-Risk Sepsis Patient                             ║");
        System.out.println("╚════════════════════════════════════════════════════════════════╝");

        PatientContextSnapshot patient = TestDataFactory.createPatientContext("HIGH-RISK-001", true);

        printPatientData(patient);
        runPredictions(patient);
    }

    private static void testLowRiskPatient() throws Exception {
        System.out.println("\n╔════════════════════════════════════════════════════════════════╗");
        System.out.println("║  TEST 2: Low-Risk Stable Patient                              ║");
        System.out.println("╚════════════════════════════════════════════════════════════════╝");

        PatientContextSnapshot patient = TestDataFactory.createPatientContext("LOW-RISK-001", false);

        printPatientData(patient);
        runPredictions(patient);
    }

    private static void testCustomPatient(Scanner scanner) throws Exception {
        System.out.println("\n╔════════════════════════════════════════════════════════════════╗");
        System.out.println("║  TEST 3: Custom Patient Data                                  ║");
        System.out.println("╚════════════════════════════════════════════════════════════════╝");

        PatientContextSnapshot patient = new PatientContextSnapshot();
        patient.setPatientId("CUSTOM-001");
        patient.setEncounterId("encounter-custom-001");

        // Demographics
        System.out.println("\n--- Demographics ---");
        System.out.print("Age (years): ");
        patient.setAge(Integer.parseInt(scanner.nextLine().trim()));

        System.out.print("Gender (male/female): ");
        patient.setGender(scanner.nextLine().trim());

        // Vital Signs
        System.out.println("\n--- Vital Signs ---");
        System.out.print("Heart Rate (bpm, normal: 60-100): ");
        patient.setHeartRate(Double.parseDouble(scanner.nextLine().trim()));

        System.out.print("Systolic BP (mmHg, normal: 90-120): ");
        patient.setSystolicBP(Double.parseDouble(scanner.nextLine().trim()));

        System.out.print("Diastolic BP (mmHg, normal: 60-80): ");
        patient.setDiastolicBP(Double.parseDouble(scanner.nextLine().trim()));

        System.out.print("Respiratory Rate (breaths/min, normal: 12-20): ");
        patient.setRespiratoryRate(Double.parseDouble(scanner.nextLine().trim()));

        System.out.print("Temperature (°C, normal: 36.5-37.5): ");
        patient.setTemperature(Double.parseDouble(scanner.nextLine().trim()));

        System.out.print("Oxygen Saturation (%, normal: >95): ");
        patient.setOxygenSaturation(Double.parseDouble(scanner.nextLine().trim()));

        // Calculate derived metrics
        patient.calculateDerivedMetrics();

        // Lab Values
        System.out.println("\n--- Lab Values ---");
        System.out.print("Lactate (mmol/L, normal: <2): ");
        patient.setLactate(Double.parseDouble(scanner.nextLine().trim()));

        System.out.print("Creatinine (mg/dL, normal: 0.7-1.3): ");
        patient.setCreatinine(Double.parseDouble(scanner.nextLine().trim()));

        System.out.print("White Blood Cells (K/μL, normal: 4-11): ");
        patient.setWhiteBloodCells(Double.parseDouble(scanner.nextLine().trim()));

        // Clinical Scores
        System.out.println("\n--- Clinical Scores ---");
        System.out.print("NEWS2 Score (0-20, high risk: >7): ");
        patient.setNews2Score(Integer.parseInt(scanner.nextLine().trim()));

        System.out.print("qSOFA Score (0-3, high risk: ≥2): ");
        patient.setQsofaScore(Integer.parseInt(scanner.nextLine().trim()));

        System.out.print("SOFA Score (0-24, high risk: >6): ");
        patient.setSofaScore(Integer.parseInt(scanner.nextLine().trim()));

        printPatientData(patient);
        runPredictions(patient);
    }

    private static void testQuickComparison() throws Exception {
        System.out.println("\n╔════════════════════════════════════════════════════════════════╗");
        System.out.println("║  TEST 4: Quick Comparison (High vs Low Risk)                  ║");
        System.out.println("╚════════════════════════════════════════════════════════════════╝");

        System.out.println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
        System.out.println("  HIGH-RISK PATIENT");
        System.out.println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
        PatientContextSnapshot highRisk = TestDataFactory.createPatientContext("HIGH-001", true);
        printPatientDataCompact(highRisk);
        runPredictionsCompact(highRisk, "HIGH RISK");

        System.out.println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
        System.out.println("  LOW-RISK PATIENT");
        System.out.println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
        PatientContextSnapshot lowRisk = TestDataFactory.createPatientContext("LOW-001", false);
        printPatientDataCompact(lowRisk);
        runPredictionsCompact(lowRisk, "LOW RISK");
    }

    private static void printPatientData(PatientContextSnapshot patient) {
        System.out.println("\n📋 Patient Clinical Data:");
        System.out.println("─────────────────────────────────────────────────────────────────");
        System.out.println("  Patient ID: " + patient.getPatientId());
        System.out.println("  Age: " + patient.getAge() + " years");
        System.out.println("  Gender: " + patient.getGender());
        System.out.println();

        System.out.println("  Vital Signs:");
        System.out.println("    • Heart Rate: " + String.format("%.1f", patient.getHeartRate()) + " bpm");
        System.out.println("    • Blood Pressure: " + String.format("%.0f", patient.getSystolicBP()) + "/" +
                          String.format("%.0f", patient.getDiastolicBP()) + " mmHg");
        System.out.println("    • Respiratory Rate: " + String.format("%.1f", patient.getRespiratoryRate()) + " /min");
        System.out.println("    • Temperature: " + String.format("%.1f", patient.getTemperature()) + " °C");
        System.out.println("    • O2 Saturation: " + String.format("%.1f", patient.getOxygenSaturation()) + " %");
        System.out.println();

        System.out.println("  Lab Values:");
        System.out.println("    • Lactate: " + String.format("%.1f", patient.getLactate()) + " mmol/L");
        System.out.println("    • Creatinine: " + String.format("%.1f", patient.getCreatinine()) + " mg/dL");
        System.out.println("    • WBC: " + String.format("%.1f", patient.getWhiteBloodCells()) + " K/μL");
        System.out.println();

        System.out.println("  Clinical Scores:");
        System.out.println("    • NEWS2: " + patient.getNews2Score() + " (high risk: >7)");
        System.out.println("    • qSOFA: " + patient.getQsofaScore() + " (high risk: ≥2)");
        System.out.println("    • SOFA: " + patient.getSofaScore() + " (high risk: >6)");
        System.out.println("─────────────────────────────────────────────────────────────────");
    }

    private static void printPatientDataCompact(PatientContextSnapshot patient) {
        System.out.printf("  Age: %d | HR: %.0f | BP: %.0f/%.0f | Temp: %.1f | SpO2: %.0f%%\n",
            patient.getAge(), patient.getHeartRate(), patient.getSystolicBP(),
            patient.getDiastolicBP(), patient.getTemperature(), patient.getOxygenSaturation());
        System.out.printf("  Lactate: %.1f | Creatinine: %.1f | WBC: %.1f\n",
            patient.getLactate(), patient.getCreatinine(), patient.getWhiteBloodCells());
        System.out.printf("  NEWS2: %d | qSOFA: %d | SOFA: %d\n",
            patient.getNews2Score(), patient.getQsofaScore(), patient.getSofaScore());
    }

    private static void runPredictions(PatientContextSnapshot patient) throws Exception {
        System.out.println("\n🤖 ML Model Predictions:");
        System.out.println("─────────────────────────────────────────────────────────────────");

        // Extract features
        ClinicalFeatureVector features = extractor.extract(patient, null, null);
        float[] featureArray = features.toFloatArray();

        // Run predictions on all 4 models
        MLPrediction sepsisPred = sepsisModel.predict(featureArray);
        MLPrediction detPred = deteriorationModel.predict(featureArray);
        MLPrediction mortPred = mortalityModel.predict(featureArray);
        MLPrediction readmPred = readmissionModel.predict(featureArray);

        // Print results
        printPrediction("Sepsis Risk (6-hour)", sepsisPred);
        printPrediction("Clinical Deterioration", detPred);
        printPrediction("30-Day Mortality", mortPred);
        printPrediction("30-Day Readmission", readmPred);

        System.out.println("─────────────────────────────────────────────────────────────────");
        System.out.println("\n💡 Clinical Interpretation:");
        interpretResults(sepsisPred, detPred, mortPred, readmPred);
    }

    private static void runPredictionsCompact(PatientContextSnapshot patient, String label) throws Exception {
        ClinicalFeatureVector features = extractor.extract(patient, null, null);
        float[] featureArray = features.toFloatArray();

        MLPrediction sepsisPred = sepsisModel.predict(featureArray);
        MLPrediction detPred = deteriorationModel.predict(featureArray);
        MLPrediction mortPred = mortalityModel.predict(featureArray);
        MLPrediction readmPred = readmissionModel.predict(featureArray);

        System.out.println("\n  📊 Predictions:");
        System.out.printf("    Sepsis:          %5.1f%%  %s\n", sepsisPred.getPrimaryScore() * 100, getRiskBadge(sepsisPred.getPrimaryScore()));
        System.out.printf("    Deterioration:   %5.1f%%  %s\n", detPred.getPrimaryScore() * 100, getRiskBadge(detPred.getPrimaryScore()));
        System.out.printf("    Mortality:       %5.1f%%  %s\n", mortPred.getPrimaryScore() * 100, getRiskBadge(mortPred.getPrimaryScore()));
        System.out.printf("    Readmission:     %5.1f%%  %s\n", readmPred.getPrimaryScore() * 100, getRiskBadge(readmPred.getPrimaryScore()));
    }

    private static void printPrediction(String modelName, MLPrediction prediction) {
        double riskPercent = prediction.getPrimaryScore() * 100;
        String riskLevel = getRiskLevel(prediction.getPrimaryScore());
        String badge = getRiskBadge(prediction.getPrimaryScore());

        System.out.printf("  %-30s %6.2f%%  %s  %s\n",
            modelName + ":",
            riskPercent,
            badge,
            riskLevel
        );
    }

    private static String getRiskLevel(double score) {
        if (score >= 0.7) return "[HIGH RISK]";
        if (score >= 0.3) return "[MODERATE RISK]";
        return "[LOW RISK]";
    }

    private static String getRiskBadge(double score) {
        if (score >= 0.7) return "🔴";
        if (score >= 0.3) return "🟡";
        return "🟢";
    }

    private static void interpretResults(MLPrediction sepsis, MLPrediction det, MLPrediction mort, MLPrediction readm) {
        System.out.println("─────────────────────────────────────────────────────────────────");

        // Sepsis interpretation
        if (sepsis.getPrimaryScore() >= 0.7) {
            System.out.println("⚠️  SEPSIS: HIGH RISK (6-hour horizon)");
            System.out.println("    → Action: Draw blood cultures STAT, start empiric antibiotics");
            System.out.println("    → Monitor: Lactate trend, vital signs q1h");
        } else if (sepsis.getPrimaryScore() >= 0.3) {
            System.out.println("⚠️  SEPSIS: MODERATE RISK");
            System.out.println("    → Action: Enhanced monitoring, trend lactate");
        } else {
            System.out.println("✅ SEPSIS: LOW RISK");
            System.out.println("    → Action: Routine monitoring");
        }

        System.out.println();

        // Deterioration interpretation
        if (det.getPrimaryScore() >= 0.7) {
            System.out.println("⚠️  DETERIORATION: HIGH RISK");
            System.out.println("    → Action: Notify rapid response team, increase monitoring frequency");
            System.out.println("    → Consider: ICU transfer, escalation of care");
        } else if (det.getPrimaryScore() >= 0.3) {
            System.out.println("⚠️  DETERIORATION: MODERATE RISK");
            System.out.println("    → Action: Enhanced vital signs monitoring");
        } else {
            System.out.println("✅ DETERIORATION: LOW RISK");
            System.out.println("    → Action: Standard nursing protocols");
        }

        System.out.println();

        // Mortality interpretation
        if (mort.getPrimaryScore() >= 0.7) {
            System.out.println("⚠️  MORTALITY: HIGH RISK (30-day)");
            System.out.println("    → Action: Goals of care discussion, consider palliative care consult");
            System.out.println("    → Assess: Advance directives, family meeting");
        }

        System.out.println();

        // Readmission interpretation
        if (readm.getPrimaryScore() >= 0.7) {
            System.out.println("⚠️  READMISSION: HIGH RISK (30-day)");
            System.out.println("    → Action: Enhanced discharge planning, home health services");
            System.out.println("    → Coordinate: Follow-up appointments, medication reconciliation");
        }

        System.out.println("─────────────────────────────────────────────────────────────────");
    }
}
