package com.cardiofit.flink.ml;

import com.cardiofit.flink.ml.features.ClinicalFeatureExtractor;
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import com.cardiofit.flink.ml.util.TestDataFactory;
import com.cardiofit.flink.models.MLPrediction;
import org.junit.jupiter.api.*;

import java.util.Arrays;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Quick ML Demo - Shows predictions on different patient scenarios!
 *
 * Run: mvn test -Dtest=QuickMLDemo
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
@DisplayName("Quick ML Prediction Demo")
public class QuickMLDemo {

    private static ONNXModelContainer sepsisModel;
    private static ONNXModelContainer deteriorationModel;
    private static ONNXModelContainer mortalityModel;
    private static ONNXModelContainer readmissionModel;

    private static ClinicalFeatureExtractor extractor;

    @BeforeAll
    public static void setup() throws Exception {
        System.out.println("\n");
        System.out.println("╔════════════════════════════════════════════════════════════════╗");
        System.out.println("║     CardioFit ML Inference - Quick Demo                       ║");
        System.out.println("║     All 4 Clinical Risk Models                                ║");
        System.out.println("╚════════════════════════════════════════════════════════════════╝");
        System.out.println();

        // Initialize feature extractor
        extractor = new ClinicalFeatureExtractor();

        // Load all 4 models
        sepsisModel = createModel(
            "models/sepsis_risk_v1.0.0.onnx",
            "sepsis_v1",
            "Sepsis Risk Predictor",
            ONNXModelContainer.ModelType.SEPSIS_ONSET
        );

        deteriorationModel = createModel(
            "models/deterioration_risk_v1.0.0.onnx",
            "deterioration_v1",
            "Clinical Deterioration Predictor",
            ONNXModelContainer.ModelType.CLINICAL_DETERIORATION
        );

        mortalityModel = createModel(
            "models/mortality_risk_v1.0.0.onnx",
            "mortality_v1",
            "30-Day Mortality Predictor",
            ONNXModelContainer.ModelType.MORTALITY_PREDICTION
        );

        readmissionModel = createModel(
            "models/readmission_risk_v1.0.0.onnx",
            "readmission_v1",
            "30-Day Readmission Predictor",
            ONNXModelContainer.ModelType.READMISSION_RISK
        );

        System.out.println("✅ All 4 ONNX models loaded successfully!");
        System.out.println();
    }

    private static ONNXModelContainer createModel(
        String path,
        String id,
        String name,
        ONNXModelContainer.ModelType type
    ) throws Exception {
        ModelConfig config = ModelConfig.builder()
            .modelPath(path)
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.7)
            .build();

        ONNXModelContainer model = ONNXModelContainer.builder()
            .modelId(id)
            .modelName(name)
            .modelType(type)
            .modelVersion("1.0.0")
            .inputFeatureNames(TestDataFactory.createFeatureNames())
            .outputNames(Arrays.asList("probability"))
            .config(config)
            .build();

        model.initialize();
        return model;
    }

    @AfterAll
    public static void cleanup() throws Exception {
        if (sepsisModel != null) sepsisModel.close();
        if (deteriorationModel != null) deteriorationModel.close();
        if (mortalityModel != null) mortalityModel.close();
        if (readmissionModel != null) readmissionModel.close();

        System.out.println();
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println("✅ Demo Complete!");
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println();
    }

    @Test
    @Order(1)
    @DisplayName("Scenario 1: HIGH-RISK Septic Patient")
    public void testHighRiskPatient() throws Exception {
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println("🔴 SCENARIO 1: HIGH-RISK Septic Patient");
        System.out.println("════════════════════════════════════════════════════════════════");

        // Use TestDataFactory to create a high-risk septic patient
        PatientContextSnapshot patient = TestDataFactory.createPatientContext("HIGH-RISK-001", true);

        System.out.println("\n📊 Patient Profile:");
        System.out.println("   Age: " + patient.getAge() + " years");
        System.out.println("   Heart Rate: " + patient.getHeartRate() + " bpm (normal: 60-100)");
        System.out.println("   Respiratory Rate: " + patient.getRespiratoryRate() + " /min (normal: 12-20)");
        System.out.println("   Temperature: " + patient.getTemperature() + "°C (normal: 36.5-37.5)");
        System.out.println("   Blood Pressure: " + patient.getSystolicBP() + "/" + patient.getDiastolicBP() + " mmHg");
        System.out.println("   O2 Saturation: " + patient.getOxygenSaturation() + "% (normal: >95%)");
        System.out.println("   WBC: " + patient.getWhiteBloodCells() + " (normal: 4.0-11.0)");
        System.out.println("   Lactate: " + patient.getLactate() + " mmol/L (normal: <2.0)");

        // Extract features and run predictions
        ClinicalFeatureVector features = extractor.extract(patient, null, null);
        float[] featureArray = features.toFloatArray();

        System.out.println("\n🤖 ML Model Predictions:");
        System.out.println("────────────────────────────────────────────────────────────────");

        MLPrediction sepsisPred = sepsisModel.predict(featureArray);
        printPrediction("Sepsis Risk", sepsisPred);

        MLPrediction detPred = deteriorationModel.predict(featureArray);
        printPrediction("Clinical Deterioration", detPred);

        MLPrediction mortPred = mortalityModel.predict(featureArray);
        printPrediction("30-Day Mortality", mortPred);

        MLPrediction readmPred = readmissionModel.predict(featureArray);
        printPrediction("30-Day Readmission", readmPred);

        System.out.println();

        assertThat(sepsisPred).isNotNull();
        assertThat(detPred).isNotNull();
        assertThat(mortPred).isNotNull();
        assertThat(readmPred).isNotNull();
    }

    @Test
    @Order(2)
    @DisplayName("Scenario 2: LOW-RISK Stable Patient")
    public void testLowRiskPatient() throws Exception {
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println("🟢 SCENARIO 2: LOW-RISK Stable Patient");
        System.out.println("════════════════════════════════════════════════════════════════");

        // Use TestDataFactory to create a stable patient
        PatientContextSnapshot patient = TestDataFactory.createPatientContext("LOW-RISK-001", false);

        System.out.println("\n📊 Patient Profile:");
        System.out.println("   Age: " + patient.getAge() + " years");
        System.out.println("   Heart Rate: " + patient.getHeartRate() + " bpm (normal: 60-100)");
        System.out.println("   Respiratory Rate: " + patient.getRespiratoryRate() + " /min (normal: 12-20)");
        System.out.println("   Temperature: " + patient.getTemperature() + "°C (normal: 36.5-37.5)");
        System.out.println("   Blood Pressure: " + patient.getSystolicBP() + "/" + patient.getDiastolicBP() + " mmHg");
        System.out.println("   O2 Saturation: " + patient.getOxygenSaturation() + "% (normal: >95%)");
        System.out.println("   WBC: " + patient.getWhiteBloodCells() + " (normal: 4.0-11.0)");
        System.out.println("   Lactate: " + patient.getLactate() + " mmol/L (normal: <2.0)");

        // Extract features and run predictions
        ClinicalFeatureVector features = extractor.extract(patient, null, null);
        float[] featureArray = features.toFloatArray();

        System.out.println("\n🤖 ML Model Predictions:");
        System.out.println("────────────────────────────────────────────────────────────────");

        MLPrediction sepsisPred = sepsisModel.predict(featureArray);
        printPrediction("Sepsis Risk", sepsisPred);

        MLPrediction detPred = deteriorationModel.predict(featureArray);
        printPrediction("Clinical Deterioration", detPred);

        MLPrediction mortPred = mortalityModel.predict(featureArray);
        printPrediction("30-Day Mortality", mortPred);

        MLPrediction readmPred = readmissionModel.predict(featureArray);
        printPrediction("30-Day Readmission", readmPred);

        System.out.println();

        assertThat(sepsisPred).isNotNull();
        assertThat(detPred).isNotNull();
        assertThat(mortPred).isNotNull();
        assertThat(readmPred).isNotNull();
    }

    @Test
    @Order(3)
    @DisplayName("Scenario 3: Critically Ill ICU Patient")
    public void testCriticallyIllPatient() throws Exception {
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println("🚨 SCENARIO 3: Critically Ill ICU Patient");
        System.out.println("════════════════════════════════════════════════════════════════");

        // Use TestDataFactory to create a critically ill patient (high risk)
        PatientContextSnapshot patient = TestDataFactory.createPatientContext("CRITICAL-001", true);

        System.out.println("\n📊 Patient Profile:");
        System.out.println("   Age: " + patient.getAge() + " years");
        System.out.println("   Heart Rate: " + patient.getHeartRate() + " bpm (CRITICAL)");
        System.out.println("   Respiratory Rate: " + patient.getRespiratoryRate() + " /min (CRITICAL)");
        System.out.println("   Temperature: " + patient.getTemperature() + "°C");
        System.out.println("   Blood Pressure: " + patient.getSystolicBP() + "/" + patient.getDiastolicBP() + " mmHg (HYPOTENSIVE)");
        System.out.println("   O2 Saturation: " + patient.getOxygenSaturation() + "% (HYPOXIC)");
        System.out.println("   WBC: " + patient.getWhiteBloodCells() + " (ELEVATED)");
        System.out.println("   Lactate: " + patient.getLactate() + " mmol/L (CRITICAL)");

        // Extract features and run predictions
        ClinicalFeatureVector features = extractor.extract(patient, null, null);
        float[] featureArray = features.toFloatArray();

        System.out.println("\n🤖 ML Model Predictions:");
        System.out.println("────────────────────────────────────────────────────────────────");

        MLPrediction sepsisPred = sepsisModel.predict(featureArray);
        printPrediction("Sepsis Risk", sepsisPred);

        MLPrediction detPred = deteriorationModel.predict(featureArray);
        printPrediction("Clinical Deterioration", detPred);

        MLPrediction mortPred = mortalityModel.predict(featureArray);
        printPrediction("30-Day Mortality", mortPred);

        MLPrediction readmPred = readmissionModel.predict(featureArray);
        printPrediction("30-Day Readmission", readmPred);

        System.out.println();

        assertThat(sepsisPred).isNotNull();
        assertThat(detPred).isNotNull();
        assertThat(mortPred).isNotNull();
        assertThat(readmPred).isNotNull();
    }

    // Helper method to print predictions with color-coded risk levels
    private void printPrediction(String modelName, MLPrediction prediction) {
        double score = prediction.getPrimaryScore();
        String riskLevel;
        String emoji;

        if (score >= 0.7) {
            riskLevel = "HIGH RISK";
            emoji = "🔴";
        } else if (score >= 0.3) {
            riskLevel = "MODERATE RISK";
            emoji = "🟡";
        } else {
            riskLevel = "LOW RISK";
            emoji = "🟢";
        }

        System.out.printf("   %s %-25s %.1f%% (%s)%n",
            emoji,
            modelName + ":",
            score * 100,
            riskLevel
        );
    }
}
