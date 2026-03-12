package com.cardiofit.flink.ml;

import com.cardiofit.flink.ml.features.ClinicalFeatureExtractor;
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import com.cardiofit.flink.ml.util.TestDataFactory;
import com.cardiofit.flink.models.MLPrediction;
import org.junit.jupiter.api.*;

import java.util.Arrays;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Custom Patient ML Test - Test with YOUR patient data!
 *
 * Edit the patient data in createCustomPatient() method below,
 * then run: mvn test -Dtest=CustomPatientMLTest
 */
@DisplayName("Custom Patient ML Test")
public class CustomPatientMLTest {

    private static ONNXModelContainer sepsisModel;
    private static ONNXModelContainer deteriorationModel;
    private static ONNXModelContainer mortalityModel;
    private static ONNXModelContainer readmissionModel;
    private static ClinicalFeatureExtractor extractor;

    @BeforeAll
    public static void setup() throws Exception {
        System.out.println("\n");
        System.out.println("╔════════════════════════════════════════════════════════════════╗");
        System.out.println("║     CardioFit ML - Custom Patient Testing                     ║");
        System.out.println("╚════════════════════════════════════════════════════════════════╝");
        System.out.println();

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
        System.out.println("✅ Test Complete!");
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println();
    }

    /**
     * ═══════════════════════════════════════════════════════════════════
     * 📝 EDIT YOUR PATIENT DATA HERE!
     * ═══════════════════════════════════════════════════════════════════
     */
    private PatientContextSnapshot createCustomPatient() {
        PatientContextSnapshot patient = new PatientContextSnapshot();

        // Patient Demographics
        patient.setPatientId("PAT-ROHAN-001");
        patient.setAge(42);
        patient.setGender("M");

        // Vital Signs (YOUR DATA HERE!)
        patient.setHeartRate(135.0);          // bpm (normal: 60-100)
        patient.setRespiratoryRate(28.0);     // /min (normal: 12-20)
        patient.setTemperature(39.5);         // °C (normal: 36.5-37.5)
        patient.setSystolicBP(85.0);          // mmHg (normal: 90-120)
        patient.setDiastolicBP(50.0);         // mmHg (normal: 60-80)
        patient.setOxygenSaturation(88.0);    // % (normal: >95)

        // Lab Values (YOUR DATA HERE!)
        patient.setWhiteBloodCells(18.0);     // 10⁹/L (normal: 4.0-11.0)
        patient.setLactate(5.2);              // mmol/L (normal: <2.0)
        patient.setHemoglobin(18.0);          // g/dL (normal: 12-16)
        patient.setPlatelets(130.0);          // 10⁹/L (normal: 150-400)
        patient.setCreatinine(2.5);           // mg/dL (normal: 0.6-1.2)
        patient.setGlucose(120.0);            // mg/dL (normal: 70-100)

        // Calculate derived metrics (don't modify this)
        patient.calculateDerivedMetrics();

        return patient;
    }

    @Test
    @DisplayName("Test ML Predictions on Custom Patient Data")
    public void testCustomPatient() throws Exception {
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println("🏥 Testing Custom Patient: PAT-ROHAN-001");
        System.out.println("════════════════════════════════════════════════════════════════");

        // Create your custom patient
        PatientContextSnapshot patient = createCustomPatient();

        System.out.println("\n📊 Patient Profile:");
        System.out.println("   Patient ID: " + patient.getPatientId());
        System.out.println("   Age: " + patient.getAge() + " years, Gender: " + patient.getGender());
        System.out.println();
        System.out.println("   Vital Signs:");
        System.out.println("      Heart Rate: " + patient.getHeartRate() + " bpm (normal: 60-100)");
        System.out.println("      Respiratory Rate: " + patient.getRespiratoryRate() + " /min (normal: 12-20)");
        System.out.println("      Temperature: " + patient.getTemperature() + "°C (normal: 36.5-37.5)");
        System.out.println("      Blood Pressure: " + patient.getSystolicBP() + "/" + patient.getDiastolicBP() + " mmHg");
        System.out.println("      O2 Saturation: " + patient.getOxygenSaturation() + "% (normal: >95%)");
        System.out.println();
        System.out.println("   Lab Values:");
        System.out.println("      WBC: " + patient.getWhiteBloodCells() + " (normal: 4.0-11.0)");
        System.out.println("      Lactate: " + patient.getLactate() + " mmol/L (normal: <2.0)");
        System.out.println("      Hemoglobin: " + patient.getHemoglobin() + " g/dL (normal: 12-16)");
        System.out.println("      Platelets: " + patient.getPlatelets() + " (normal: 150-400)");
        System.out.println("      Creatinine: " + patient.getCreatinine() + " mg/dL (normal: 0.6-1.2)");
        System.out.println("      Glucose: " + patient.getGlucose() + " mg/dL (normal: 70-100)");

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
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println("🔍 Clinical Assessment:");
        System.out.println("════════════════════════════════════════════════════════════════");

        // Clinical interpretation
        System.out.println("\n⚠️  CRITICAL FINDINGS:");
        if (patient.getHeartRate() > 100) {
            System.out.println("   🔴 Tachycardia (HR: " + patient.getHeartRate() + " bpm)");
        }
        if (patient.getRespiratoryRate() > 20) {
            System.out.println("   🔴 Tachypnea (RR: " + patient.getRespiratoryRate() + " /min)");
        }
        if (patient.getTemperature() > 38.0) {
            System.out.println("   🔴 Fever (Temp: " + patient.getTemperature() + "°C)");
        }
        if (patient.getSystolicBP() < 90) {
            System.out.println("   🔴 Hypotension (BP: " + patient.getSystolicBP() + "/" + patient.getDiastolicBP() + ")");
        }
        if (patient.getOxygenSaturation() < 95) {
            System.out.println("   🔴 Hypoxemia (O2 Sat: " + patient.getOxygenSaturation() + "%)");
        }
        if (patient.getWhiteBloodCells() > 12.0) {
            System.out.println("   🔴 Leukocytosis (WBC: " + patient.getWhiteBloodCells() + ")");
        }
        if (patient.getLactate() > 2.0) {
            System.out.println("   🔴 Elevated Lactate (" + patient.getLactate() + " mmol/L)");
        }
        if (patient.getCreatinine() > 1.2) {
            System.out.println("   🔴 Renal Dysfunction (Cr: " + patient.getCreatinine() + " mg/dL)");
        }

        System.out.println();

        // Verify predictions ran
        assertThat(sepsisPred).isNotNull();
        assertThat(detPred).isNotNull();
        assertThat(mortPred).isNotNull();
        assertThat(readmPred).isNotNull();
    }

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

        System.out.printf("   %s %-30s %.1f%% (%s)%n",
            emoji,
            modelName + ":",
            score * 100,
            riskLevel
        );
    }
}
