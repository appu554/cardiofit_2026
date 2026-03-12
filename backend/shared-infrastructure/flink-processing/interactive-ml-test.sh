#!/bin/bash

# Interactive ML Testing - Enter Your Own Patient Data
# This script lets you input custom patient clinical data and see ML predictions

cd "$(dirname "$0")"

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║     CardioFit ML - Interactive Patient Testing                ║"
echo "║     Enter Custom Patient Data for ML Predictions              ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Check if models exist
if [ ! -f "models/sepsis_risk_v1.0.0.onnx" ]; then
    echo "❌ Error: ONNX models not found"
    exit 1
fi

echo "✅ Found all 4 ONNX models"
echo ""
echo "📋 Enter Patient Clinical Data:"
echo "   (Press Enter for default values shown in brackets)"
echo ""

# Collect patient data
read -p "Patient ID [CUSTOM-PT-001]: " patient_id
patient_id=${patient_id:-CUSTOM-PT-001}

read -p "Age (years) [65]: " age
age=${age:-65}

read -p "Gender (M/F) [M]: " gender
gender=${gender:-M}

echo ""
echo "📊 Vital Signs:"

read -p "Heart Rate (bpm, normal: 60-100) [85]: " hr
hr=${hr:-85}

read -p "Respiratory Rate (/min, normal: 12-20) [18]: " rr
rr=${rr:-18}

read -p "Temperature (°C, normal: 36.5-37.5) [37.2]: " temp
temp=${temp:-37.2}

read -p "Systolic BP (mmHg, normal: 90-120) [120]: " sbp
sbp=${sbp:-120}

read -p "Diastolic BP (mmHg, normal: 60-80) [80]: " dbp
dbp=${dbp:-80}

read -p "O2 Saturation (%, normal: >95) [97]: " o2
o2=${o2:-97}

echo ""
echo "🧪 Lab Values:"

read -p "White Blood Cells (10⁹/L, normal: 4.0-11.0) [7.5]: " wbc
wbc=${wbc:-7.5}

read -p "Lactate (mmol/L, normal: <2.0) [1.5]: " lactate
lactate=${lactate:-1.5}

read -p "Hemoglobin (g/dL, normal: 12-16) [14.0]: " hgb
hgb=${hgb:-14.0}

read -p "Platelets (10⁹/L, normal: 150-400) [250]: " plt
plt=${plt:-250}

read -p "Creatinine (mg/dL, normal: 0.6-1.2) [1.0]: " cr
cr=${cr:-1.0}

read -p "Glucose (mg/dL, normal: 70-100) [90]: " glucose
glucose=${glucose:-90}

echo ""
echo "════════════════════════════════════════════════════════════════"
echo "📝 Your Patient Data Summary:"
echo "════════════════════════════════════════════════════════════════"
echo "Patient ID: $patient_id"
echo "Age: $age years, Gender: $gender"
echo ""
echo "Vital Signs:"
echo "  Heart Rate: $hr bpm"
echo "  Respiratory Rate: $rr /min"
echo "  Temperature: $temp °C"
echo "  Blood Pressure: $sbp/$dbp mmHg"
echo "  O2 Saturation: $o2 %"
echo ""
echo "Lab Values:"
echo "  WBC: $wbc"
echo "  Lactate: $lactate mmol/L"
echo "  Hemoglobin: $hgb g/dL"
echo "  Platelets: $plt"
echo "  Creatinine: $cr mg/dL"
echo "  Glucose: $glucose mg/dL"
echo "════════════════════════════════════════════════════════════════"
echo ""
read -p "🤖 Run ML predictions on this patient? (y/n) [y]: " confirm
confirm=${confirm:-y}

if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "❌ Cancelled."
    exit 0
fi

echo ""
echo "🚀 Running ML predictions..."
echo ""

# Create a temporary Java test file with the custom data
cat > CustomPatientTest.java << EOF
import com.cardiofit.flink.ml.*;
import com.cardiofit.flink.ml.features.ClinicalFeatureExtractor;
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import com.cardiofit.flink.ml.util.TestDataFactory;
import com.cardiofit.flink.models.MLPrediction;
import java.util.Arrays;

public class CustomPatientTest {
    public static void main(String[] args) throws Exception {
        // Create patient with custom data
        PatientContextSnapshot patient = new PatientContextSnapshot();
        patient.setPatientId("$patient_id");
        patient.setAge($age);
        patient.setGender("$gender");
        patient.setHeartRate($hr.0);
        patient.setRespiratoryRate($rr.0);
        patient.setTemperature($temp);
        patient.setSystolicBP($sbp.0);
        patient.setDiastolicBP($dbp.0);
        patient.setOxygenSaturation($o2.0);
        patient.setWhiteBloodCells($wbc);
        patient.setLactate($lactate);
        patient.setHemoglobin($hgb);
        patient.setPlatelets($plt.0);
        patient.setCreatinine($cr);
        patient.setGlucose($glucose.0);
        patient.calculateDerivedMetrics();

        // Load all 4 models
        ONNXModelContainer sepsisModel = createModel(
            "models/sepsis_risk_v1.0.0.onnx",
            "sepsis_v1",
            "Sepsis Risk",
            ONNXModelContainer.ModelType.SEPSIS_ONSET
        );

        ONNXModelContainer deteriorationModel = createModel(
            "models/deterioration_risk_v1.0.0.onnx",
            "deterioration_v1",
            "Clinical Deterioration",
            ONNXModelContainer.ModelType.CLINICAL_DETERIORATION
        );

        ONNXModelContainer mortalityModel = createModel(
            "models/mortality_risk_v1.0.0.onnx",
            "mortality_v1",
            "30-Day Mortality",
            ONNXModelContainer.ModelType.MORTALITY_PREDICTION
        );

        ONNXModelContainer readmissionModel = createModel(
            "models/readmission_risk_v1.0.0.onnx",
            "readmission_v1",
            "30-Day Readmission",
            ONNXModelContainer.ModelType.READMISSION_RISK
        );

        // Extract features
        ClinicalFeatureExtractor extractor = new ClinicalFeatureExtractor();
        ClinicalFeatureVector features = extractor.extract(patient, null, null);
        float[] featureArray = features.toFloatArray();

        // Run predictions
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println("🤖 ML Model Predictions for Patient: $patient_id");
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println();

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

        // Cleanup
        sepsisModel.close();
        deteriorationModel.close();
        mortalityModel.close();
        readmissionModel.close();
    }

    private static ONNXModelContainer createModel(String path, String id, String name, ONNXModelContainer.ModelType type) throws Exception {
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

    private static void printPrediction(String modelName, MLPrediction prediction) {
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

        System.out.printf("   %s %-30s %.1f%% (%s)%n", emoji, modelName + ":", score * 100, riskLevel);
    }
}
EOF

# Compile and run
echo "🔨 Compiling custom test..."
mvn exec:java -Dexec.mainClass="CustomPatientTest" -Dexec.classpathScope="test" -q 2>/dev/null

# Cleanup
rm -f CustomPatientTest.java

echo ""
echo "════════════════════════════════════════════════════════════════"
echo "✅ Prediction Complete!"
echo ""
echo "🔍 Understanding Your Results:"
echo "   🔴 HIGH RISK (≥70%):     Immediate intervention needed"
echo "   🟡 MODERATE RISK (30-70%): Enhanced monitoring required"
echo "   🟢 LOW RISK (<30%):      Routine care appropriate"
echo ""
echo "📖 For more information:"
echo "   See: claudedocs/MODULE5_HOW_TO_TEST_YOUR_MODELS.md"
echo "════════════════════════════════════════════════════════════════"
echo ""
