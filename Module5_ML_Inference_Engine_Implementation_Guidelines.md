# Module 5: ML Inference Engine — Implementation Guidelines

> **Pipeline Position:** Module 4 (Pattern Detection) → **Module 5 (ML Inference)** → Downstream Consumers
> **Operator Pattern:** `KeyedCoProcessFunction` with dual-input streams
> **Tech Stack:** Java 17, Flink 2.1.0, ONNX Runtime 1.17+, Jackson 2.17

---

## 1. Architecture Overview

Module 5 is the predictive intelligence layer. It consumes two streams — enriched CDS events from Module 3 and detected pattern events from Module 4 — fuses them into per-patient feature vectors, runs ONNX model inference, and emits clinical predictions with confidence scores and explainability metadata.

### Position in the DAG

```
Module 3 (CDS Engine)                    Module 5 (ML Inference)
comprehensive-cds-events.v1 ──────────►┌─────────────────────────┐
                                        │                         │──► ml-predictions.v1
Module 4 (Pattern Detection)            │  Feature Extraction     │──► high-risk-predictions.v1
clinical-patterns.v1 ──────────────────►│  ONNX Inference         │──► prediction-audit.v1
                                        │  Ensemble & Calibration │──► prediction-feedback.v1
                                        │  Explanation Generation │
                                        └─────────────────────────┘
```

### Five Prediction Categories

| Category | Clinical Value | Input Dependency | Latency Target |
|----------|---------------|------------------|----------------|
| **Readmission Risk** | 30-day readmission probability | CDS patient state + historical patterns | < 500ms |
| **Sepsis Onset** | Early sepsis detection (pre-SIRS) | Vital trends + lab trajectories + pattern events | < 200ms |
| **Clinical Deterioration** | 6-12hr deterioration forecast | CEP patterns + windowed trends + scoring trajectory | < 200ms |
| **Fall Risk** | Inpatient fall probability | Medication profile + mobility indicators + age | < 500ms |
| **Mortality Risk** | In-hospital mortality estimate | Acuity trajectory + organ failure indicators | < 500ms |

---

## 2. Input Contracts

### 2.1 Primary Input: CDS Events (from Module 3)

**Topic:** `comprehensive-cds-events.v1`
**Key:** `patientId`

This is the same event your Module 4 consumes. For Module 5, the critical fields are:

```
CDSEvent {
  patientId: String                    // keying field
  patientState: {
    latestVitals: Map<String, Object>  // heartrate, systolicbloodpressure, etc.
    recentLabs: Map<String, LabResult> // keyed by LOINC code
    riskIndicators: Map<String, Object>// boolean flags + trend directions
    news2Score: int
    qsofaScore: int
    combinedAcuityScore: double
    eventCount: int
  }
  eventType: String                    // VITAL_SIGN, LAB_RESULT, etc.
  eventTime: long
  processingTime: long
  semanticEnrichment: {
    semanticTags: List<String>         // e.g., "LOW_ACUITY_PATIENT"
    clinicalThresholds: Map            // scored thresholds with citations
    cepPatternFlags: Map               // pre-computed CEP readiness flags
  }
}
```

**CRITICAL — Schema lessons from E2E testing:**
- Vital sign keys are lowercase-no-separator: `heartrate`, `systolicbloodpressure`, `oxygensaturation` — NOT `heart_rate` or `systolic_bp`
- `latestVitals` may contain non-vital fields (`age`, `gender`) — filter these during feature extraction
- `recentLabs` is keyed by LOINC code (e.g., `"2524-7"` for lactate) — LabResult.getValue() CAN return null (auto-unboxing trap — always null-check before numeric comparison)
- `combinedAcuityScore` can be 0.0 for low-acuity patients — this is valid, not missing data
- Jackson deserializes with `FAIL_ON_UNKNOWN_PROPERTIES = false` — fields that don't match are silently dropped. Validate patientId is non-null at entry point

### 2.2 Secondary Input: Pattern Events (from Module 4)

**Topic:** `clinical-patterns.v1`
**Key:** `patientId`

```
PatternEvent {
  id: String
  patientId: String
  patternType: String         // CLINICAL_DETERIORATION, VITAL_SIGNS_TREND,
                              // CROSS_DOMAIN_DECLINE, MEDICATION_ADHERENCE,
                              // PATHWAY_COMPLIANCE, TREND_ANALYSIS,
                              // ANOMALY_DETECTION, PROTOCOL_MONITORING,
                              // EARLY_WARNING, SEPSIS_RISK, AKI_RISK
  severity: String            // LOW, MODERATE, HIGH, CRITICAL
  confidence: double          // 0.0 - 1.0
  detectionTime: long
  involvedEvents: List<String>
  patternDetails: Map<String, Object>  // trend_slope, deterioration_rate, etc.
  recommendedActions: List<String>
  tags: Set<String>           // may include SEVERITY_ESCALATION, MULTI_SOURCE_CONFIRMED
  patternMetadata: {
    algorithm: String         // CEP_DETERIORATION, MULTI_SOURCE_MERGED, etc.
    version: String
    processingTime: double
  }
}
```

**Key insight from Module 4 testing:** The 80→354 amplification ratio means each CDS event can generate ~4-5 pattern events (one per detection engine). Module 5 should NOT run inference on every individual pattern event. Instead, accumulate patterns in patient state and trigger inference on CDS events (which arrive at a lower, more stable rate).

---

## 3. Operator Architecture

### 3.1 Why KeyedCoProcessFunction (Not Union)

Module 4 used a simple union of streams. Module 5 needs `KeyedCoProcessFunction` because:
- CDS events and pattern events have different schemas requiring different feature extraction
- Pattern events should update patient state but NOT independently trigger inference (to avoid 4-5x inference amplification)
- The two streams need coordinated watermarking for consistent event-time processing

```java
public class Module5_MLInferenceEngine
    extends KeyedCoProcessFunction<String, CDSEvent, PatternEvent, MLPrediction> {

    // ── State ──
    private ValueState<PatientMLState> patientState;
    private ValueState<Long> lastInferenceTime;

    // ── Models ──
    private transient Map<String, OrtSession> modelSessions;  // prediction category → ONNX session
    private transient OrtEnvironment ortEnv;

    // ── Configuration ──
    private static final long MIN_INFERENCE_INTERVAL_MS = 30_000;  // 30s cooldown per patient
    private static final int MAX_PATTERN_BUFFER_SIZE = 20;         // per patient
}
```

### 3.2 Dual processElement Methods

```java
@Override
public void processElement1(CDSEvent cdsEvent, Context ctx, Collector<MLPrediction> out)
    throws Exception {
    // PRIMARY path — CDS events trigger inference
    // 1. Update patient state with latest clinical data
    // 2. Check inference cooldown (don't run on every event)
    // 3. Extract features from patient state + buffered patterns
    // 4. Run ONNX inference for applicable prediction categories
    // 5. Calibrate and emit predictions
    // 6. Clear pattern buffer after inference
}

@Override
public void processElement2(PatternEvent patternEvent, Context ctx, Collector<MLPrediction> out)
    throws Exception {
    // SECONDARY path — pattern events update state only
    // 1. Buffer pattern event in patient state (capped at MAX_PATTERN_BUFFER_SIZE)
    // 2. Update pattern-derived features (active pattern count, max severity, etc.)
    // 3. If CRITICAL severity + SEVERITY_ESCALATION tag → trigger immediate inference
    //    (bypass cooldown — this is a genuinely worsening patient)
    // 4. Otherwise, wait for next CDS event to trigger inference
}
```

### 3.3 Inference Cooldown Strategy

Without cooldown, a patient generating 10 CDS events/minute = 10 ONNX inferences/minute = wasted compute for nearly identical predictions. The cooldown should be:
- **30 seconds** default for stable patients (NEWS2 < 5)
- **10 seconds** for moderate-risk patients (NEWS2 5-6)
- **0 seconds** (every event) for high-risk patients (NEWS2 ≥ 7 OR qSOFA ≥ 2)
- **Bypassed** entirely when a CRITICAL severity escalation pattern arrives

```java
private boolean shouldRunInference(CDSEvent event, PatientMLState state) {
    Long lastRun = lastInferenceTime.value();
    if (lastRun == null) return true;  // first event for this patient

    long elapsed = event.getProcessingTime() - lastRun;
    int news2 = event.getPatientState().getNews2Score();
    int qsofa = event.getPatientState().getQsofaScore();

    if (news2 >= 7 || qsofa >= 2) return true;          // high-risk: always
    if (news2 >= 5) return elapsed >= 10_000;            // moderate: 10s cooldown
    return elapsed >= MIN_INFERENCE_INTERVAL_MS;          // stable: 30s cooldown
}
```

---

## 4. Patient ML State

```java
public class PatientMLState implements Serializable {
    private String patientId;

    // ── Latest clinical snapshot (from CDS events) ──
    private Map<String, Double> latestVitals;       // normalized vital signs
    private Map<String, Double> latestLabs;          // normalized lab values
    private int news2Score;
    private int qsofaScore;
    private double acuityScore;
    private List<String> semanticTags;
    private Map<String, Boolean> riskIndicators;

    // ── Temporal features (accumulated across events) ──
    private double[] news2History;                   // ring buffer, last 10 scores
    private int news2HistoryIndex;
    private double[] acuityHistory;                  // ring buffer, last 10 scores
    private int acuityHistoryIndex;
    private long firstEventTime;                     // for length-of-stay calculation
    private int totalEventCount;

    // ── Pattern features (from Module 4) ──
    private List<PatternSummary> recentPatterns;     // capped at MAX_PATTERN_BUFFER_SIZE
    private int deteriorationPatternCount;
    private int sepsisPatternCount;
    private String maxSeveritySeen;                  // highest severity in current window
    private boolean severityEscalationDetected;
    private long lastPatternTime;

    // ── Prediction tracking ──
    private Map<String, Double> lastPredictions;     // category → last score
    private long lastInferenceTime;
}
```

### State TTL

Apply 7-day TTL with `OnReadAndWrite` + `NeverReturnExpired`, same as Module 3. When a patient's state expires and they return, the first prediction will lack temporal features (NEWS2 history, acuity trajectory). Tag this in the output as `contextDepth: INITIAL` so downstream consumers know the prediction has lower confidence due to missing history.

```java
StateTtlConfig ttlConfig = StateTtlConfig.newBuilder(Duration.ofDays(7))
    .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
    .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
    .build();

ValueStateDescriptor<PatientMLState> descriptor =
    new ValueStateDescriptor<>("patient-ml-state", PatientMLState.class);
descriptor.enableTimeToLive(ttlConfig);
patientState = getRuntimeContext().getState(descriptor);
```

---

## 5. Feature Extraction

### 5.1 Feature Vector Structure

Each prediction category uses a shared base feature set plus category-specific features. The base set maps directly to what's available in the CDS event:

```java
public class FeatureExtractor {

    /**
     * Extract base features available for all prediction categories.
     * Returns a float array sized for ONNX input tensor.
     *
     * Feature layout (indices):
     * [0]  heart_rate (normalized 0-1, range 30-200)
     * [1]  systolic_bp (normalized 0-1, range 60-250)
     * [2]  diastolic_bp (normalized 0-1, range 30-150)
     * [3]  respiratory_rate (normalized 0-1, range 5-50)
     * [4]  oxygen_saturation (normalized 0-1, range 70-100)
     * [5]  temperature (normalized 0-1, range 34-42)
     * [6]  news2_score (normalized 0-1, range 0-20)
     * [7]  qsofa_score (normalized 0-1, range 0-3)
     * [8]  acuity_score (normalized 0-1, range 0-10)
     * [9]  event_count (log-scaled)
     * [10] hours_since_admission (capped at 720 = 30 days)
     * [11-20] news2_history (last 10 scores, 0-padded)
     * [21-30] acuity_history (last 10 scores, 0-padded)
     * [31] active_pattern_count
     * [32] deterioration_pattern_count
     * [33] max_severity_index (0=NONE, 1=LOW, 2=MODERATE, 3=HIGH, 4=CRITICAL)
     * [34] severity_escalation_flag (0 or 1)
     * [35-44] risk_indicator_flags (10 boolean flags as 0/1)
     */
    public static float[] extractBaseFeatures(PatientMLState state) {
        float[] features = new float[45];

        // Vital signs — normalize to 0-1 range
        features[0] = normalize(state.getLatestVitals().getOrDefault("heartrate", 0.0), 30, 200);
        features[1] = normalize(state.getLatestVitals().getOrDefault("systolicbloodpressure", 0.0), 60, 250);
        features[2] = normalize(state.getLatestVitals().getOrDefault("diastolicbloodpressure", 0.0), 30, 150);
        features[3] = normalize(state.getLatestVitals().getOrDefault("respiratoryrate", 0.0), 5, 50);
        features[4] = normalize(state.getLatestVitals().getOrDefault("oxygensaturation", 0.0), 70, 100);
        features[5] = normalize(state.getLatestVitals().getOrDefault("temperature", 0.0), 34, 42);

        // Note: vital key names match PRODUCTION schema (lowercase, no separator)
        // NOT the Module4TestBuilder format (snake_case)

        // Clinical scores
        features[6] = normalize(state.getNews2Score(), 0, 20);
        features[7] = normalize(state.getQsofaScore(), 0, 3);
        features[8] = normalize(state.getAcuityScore(), 0, 10);

        // ... (temporal + pattern features)

        return features;
    }

    private static float normalize(double value, double min, double max) {
        if (max == min) return 0.0f;
        return (float) Math.max(0.0, Math.min(1.0, (value - min) / (max - min)));
    }
}
```

### 5.2 Vital Sign Key Mapping

**This is the single most important lesson from Module 4 E2E testing.** Production CDS events use lowercase-no-separator keys. Your feature extractor MUST use these exact keys:

```java
// CORRECT — matches production CDS events
vitals.getOrDefault("heartrate", 0.0);
vitals.getOrDefault("systolicbloodpressure", 0.0);
vitals.getOrDefault("oxygensaturation", 0.0);

// WRONG — these keys don't exist in production data
vitals.getOrDefault("heart_rate", 0.0);      // Module4TestBuilder format
vitals.getOrDefault("systolic_bp", 0.0);     // SemanticEvent internal format
vitals.getOrDefault("HeartRate", 0.0);       // PascalCase FHIR format
```

If you also accept data through Module 4's SemanticEvent path (which uses `heart_rate` etc.), implement key normalization once, early, before feature extraction:

```java
private static final Map<String, String> VITAL_KEY_ALIASES = Map.ofEntries(
    Map.entry("heart_rate", "heartrate"),
    Map.entry("systolic_bp", "systolicbloodpressure"),
    Map.entry("diastolic_bp", "diastolicbloodpressure"),
    Map.entry("respiratory_rate", "respiratoryrate"),
    Map.entry("oxygen_saturation", "oxygensaturation")
);

private static Map<String, Double> normalizeVitalKeys(Map<String, Object> rawVitals) {
    Map<String, Double> normalized = new HashMap<>();
    for (Map.Entry<String, Object> entry : rawVitals.entrySet()) {
        String key = VITAL_KEY_ALIASES.getOrDefault(entry.getKey(), entry.getKey());
        // Skip non-vital fields that appear in production latestVitals
        if (key.equals("age") || key.equals("gender") || key.equals("bloodpressure")) continue;
        if (entry.getValue() instanceof Number) {
            normalized.put(key, ((Number) entry.getValue()).doubleValue());
        }
    }
    return normalized;
}
```

### 5.3 Category-Specific Features

**Sepsis prediction** adds: lactate value + trend, WBC value + trend, temperature trajectory, procalcitonin (if available), sepsisRisk flag, time-since-last-antibiotic

**Deterioration prediction** adds: NEWS2 slope (from history ring buffer), acuity slope, deterioration pattern count, trend_slope from TREND_ANALYSIS patterns, CEP deterioration confidence

**Readmission prediction** adds: prior admission count (from eventCount), length of current stay, medication count, comorbidity flags (hasDiabetes, hasChronicKidneyDisease, hasHeartFailure)

**Fall risk** adds: age, medication count (sedatives, antihypertensives), mobility indicators, consciousness level

**Mortality** adds: all organ failure indicators, ventilator status, vasopressor use, ICU flag, combined acuity trajectory

### 5.4 Null Safety in Feature Extraction

Apply the lesson from Module 2's NPE crash universally:

```java
// WRONG — NPE if getValue() returns null (auto-unboxing)
double lactate = state.getLatestLabs().get("2524-7").getValue();

// CORRECT — null-safe extraction
private static double safeLabValue(Map<String, LabResult> labs, String loincCode) {
    if (labs == null) return Double.NaN;
    LabResult result = labs.get(loincCode);
    if (result == null || result.getValue() == null) return Double.NaN;
    return result.getValue();
}

// In feature extraction, handle NaN as "missing"
double lactate = safeLabValue(labs, "2524-7");
features[idx] = Double.isNaN(lactate) ? -1.0f : normalize(lactate, 0, 20);
// Use -1 (out of 0-1 range) as "missing" signal for the model
```

---

## 6. ONNX Model Integration

### 6.1 Model Loading

Load models in `open()`, not in the constructor. ONNX Runtime sessions are not serializable — they must be created per-TaskManager after deserialization.

```java
@Override
public void open(OpenContext openContext) throws Exception {
    super.open(openContext);

    ortEnv = OrtEnvironment.getEnvironment();
    modelSessions = new HashMap<>();

    // Load each prediction model
    String modelBasePath = System.getenv("ML_MODEL_PATH");  // e.g., /opt/models/
    if (modelBasePath == null) modelBasePath = "/opt/flink/models";

    String[] categories = {"readmission", "sepsis", "deterioration", "fall", "mortality"};
    for (String category : categories) {
        String modelPath = modelBasePath + "/" + category + "/model.onnx";
        File modelFile = new File(modelPath);
        if (modelFile.exists()) {
            OrtSession.SessionOptions opts = new OrtSession.SessionOptions();
            opts.setIntraOpNumThreads(1);  // single thread per model — Flink manages parallelism
            opts.setOptimizationLevel(OrtSession.SessionOptions.OptLevel.ALL_OPT);
            modelSessions.put(category, ortEnv.createSession(modelPath, opts));
            LOG.info("Loaded ONNX model for {} from {}", category, modelPath);
        } else {
            LOG.warn("Model file not found for {}: {} — predictions disabled for this category",
                category, modelPath);
        }
    }

    // Initialize state descriptors
    // ... (with TTL config as described above)
}
```

### 6.2 Inference Execution

```java
private Map<String, MLPrediction> runInference(
    String patientId, PatientMLState state, CDSEvent triggerEvent) throws OrtException {

    Map<String, MLPrediction> predictions = new HashMap<>();
    float[] baseFeatures = FeatureExtractor.extractBaseFeatures(state);

    for (Map.Entry<String, OrtSession> entry : modelSessions.entrySet()) {
        String category = entry.getKey();
        OrtSession session = entry.getValue();

        // Build category-specific feature vector
        float[] fullFeatures = FeatureExtractor.appendCategoryFeatures(
            baseFeatures, state, category);

        // Create ONNX input tensor
        long[] shape = {1, fullFeatures.length};
        try (OnnxTensor inputTensor = OnnxTensor.createTensor(ortEnv, 
                new float[][]{fullFeatures})) {

            Map<String, OnnxTensor> inputs = Map.of("input", inputTensor);
            OrtSession.Result result = session.run(inputs);

            // Extract prediction
            float[][] output = (float[][]) result.get(0).getValue();
            float riskScore = output[0][0];  // assuming single output neuron (sigmoid)

            // Build prediction object
            MLPrediction prediction = new MLPrediction();
            prediction.setId(UUID.randomUUID().toString());
            prediction.setPatientId(patientId);
            prediction.setPredictionCategory(category);
            prediction.setRiskScore(riskScore);
            prediction.setConfidence(calculateConfidence(state, category));
            prediction.setTimestamp(System.currentTimeMillis());
            prediction.setTriggerEventId(triggerEvent.getEventId());
            prediction.setContextDepth(
                state.getTotalEventCount() > 3 ? "ESTABLISHED" : "INITIAL");
            prediction.setModelVersion(getModelVersion(category));

            // Calibrate
            prediction.setCalibratedScore(calibrate(riskScore, category));

            // Generate explanation
            prediction.setExplanation(
                ExplanationGenerator.explain(fullFeatures, riskScore, category));

            predictions.put(category, prediction);

        } catch (OrtException e) {
            LOG.error("ONNX inference failed for category {} patient {}: {}",
                category, patientId, e.getMessage());
            // Don't crash the pipeline — skip this category
        }
    }
    return predictions;
}
```

### 6.3 Latency Protection

ONNX inference should be fast (< 5ms per model for tabular features), but protect against outliers:

```java
private static final long INFERENCE_TIMEOUT_MS = 100;  // 100ms max per category

// In runInference, wrap each session.run() with timing
long start = System.nanoTime();
OrtSession.Result result = session.run(inputs);
long elapsedMs = (System.nanoTime() - start) / 1_000_000;

if (elapsedMs > INFERENCE_TIMEOUT_MS) {
    LOG.warn("ONNX inference slow for {} on patient {}: {}ms",
        category, patientId, elapsedMs);
}
// Emit latency as a Flink metric
getRuntimeContext().getMetricGroup()
    .gauge("inference_latency_ms_" + category, () -> elapsedMs);
```

### 6.4 Model Hot-Swapping via Broadcast State

Don't require a Flink restart to deploy new models. Use the same broadcast pattern as Module 3's KB CDC:

```java
// Broadcast stream from model-registry-updates topic
// When a new model version is published:
// 1. processBroadcastElement receives the update
// 2. Mark the category for reload
// 3. Next inference call for that category lazy-loads the new ONNX file
// 4. Old session is closed after successful load

private transient Set<String> pendingModelReloads = new HashSet<>();

// In processBroadcastElement for model updates:
public void processBroadcastElement(ModelUpdateEvent event, Context ctx,
    Collector<MLPrediction> out) throws Exception {
    pendingModelReloads.add(event.getCategory());
    LOG.info("Model update queued for category: {} version: {}",
        event.getCategory(), event.getNewVersion());
}

// In runInference, before session.run():
if (pendingModelReloads.contains(category)) {
    reloadModel(category);
    pendingModelReloads.remove(category);
}
```

---

## 7. Output Contract

### 7.1 MLPrediction Schema

```java
public class MLPrediction implements Serializable {
    private String id;
    private String patientId;
    private String encounterId;

    // ── Prediction ──
    private String predictionCategory;     // readmission, sepsis, deterioration, fall, mortality
    private double riskScore;              // raw model output (0.0 - 1.0)
    private double calibratedScore;        // after Platt scaling / isotonic calibration
    private double confidence;             // meta-confidence based on data completeness
    private String riskLevel;              // LOW, MODERATE, HIGH, CRITICAL (from calibrated score)

    // ── Context ──
    private String contextDepth;           // INITIAL or ESTABLISHED
    private int eventCountAtPrediction;
    private int patternCountAtPrediction;
    private String triggerEventId;         // which CDS event triggered this
    private String triggerSource;          // CDS_EVENT or SEVERITY_ESCALATION

    // ── Model metadata ──
    private String modelVersion;
    private String modelAlgorithm;         // e.g., "XGBoost_v3", "LSTM_v2"
    private long inferenceLatencyMs;

    // ── Explainability ──
    private Map<String, Double> featureImportances;  // top-5 contributing features
    private List<String> explanationTexts;            // human-readable explanations
    private List<String> recommendedActions;

    // ── Timestamps ──
    private long timestamp;
    private long predictionHorizonMs;      // how far ahead the prediction looks

    // ── Audit ──
    private Map<String, Object> inputSnapshot;  // feature vector used (for reproducibility)
}
```

### 7.2 Output Topics

| Topic | Content | Consumers |
|-------|---------|-----------|
| `ml-predictions.v1` | All predictions (all categories, all risk levels) | Analytics dashboards, audit trail |
| `high-risk-predictions.v1` | Only HIGH/CRITICAL predictions (riskLevel ≥ HIGH) | Clinical alerting, pager system |
| `prediction-audit.v1` | Full input snapshot + model version + feature vector | Compliance, model monitoring |
| `prediction-feedback.v1` | Prediction ID + outcome (for model retraining) | ML training pipeline |

### 7.3 Side-Output Routing

```java
// Output tags for side outputs
private static final OutputTag<MLPrediction> HIGH_RISK_TAG =
    new OutputTag<>("high-risk-predictions", TypeInformation.of(MLPrediction.class));
private static final OutputTag<MLPrediction> AUDIT_TAG =
    new OutputTag<>("prediction-audit", TypeInformation.of(MLPrediction.class));

// In processElement1 (after inference):
for (MLPrediction prediction : predictions.values()) {
    // Main output — all predictions
    out.collect(prediction);

    // Side output — high risk only
    if ("HIGH".equals(prediction.getRiskLevel()) ||
        "CRITICAL".equals(prediction.getRiskLevel())) {
        ctx.output(HIGH_RISK_TAG, prediction);
    }

    // Side output — audit trail (always)
    MLPrediction auditCopy = prediction.deepCopy();
    auditCopy.setInputSnapshot(currentFeatureSnapshot);
    ctx.output(AUDIT_TAG, auditCopy);
}
```

---

## 8. Calibration and Thresholds

### 8.1 Why Calibration Matters

Raw ONNX model output (sigmoid) is NOT a calibrated probability. A model outputting 0.7 does NOT mean "70% chance of sepsis." Clinical decisions require calibrated probabilities. Use Platt scaling (logistic calibration) or isotonic regression, trained on a held-out validation set.

```java
public class PredictionCalibrator {
    // Platt scaling parameters per category (trained offline, loaded from config)
    private final Map<String, double[]> plattParams;  // category → [A, B]

    public double calibrate(double rawScore, String category) {
        double[] params = plattParams.get(category);
        if (params == null) return rawScore;  // uncalibrated fallback
        // Platt scaling: P(y=1) = 1 / (1 + exp(A * rawScore + B))
        return 1.0 / (1.0 + Math.exp(params[0] * rawScore + params[1]));
    }
}
```

### 8.2 Clinical Risk Thresholds

These must be tuned per-category on clinical validation data. Starting points:

```java
private static String classifyRiskLevel(double calibratedScore, String category) {
    // Category-specific thresholds (loaded from config, not hardcoded)
    return switch (category) {
        case "sepsis" -> {
            // Sepsis: low threshold for HIGH because false negatives are fatal
            if (calibratedScore >= 0.60) yield "CRITICAL";
            if (calibratedScore >= 0.35) yield "HIGH";
            if (calibratedScore >= 0.15) yield "MODERATE";
            yield "LOW";
        }
        case "deterioration" -> {
            if (calibratedScore >= 0.70) yield "CRITICAL";
            if (calibratedScore >= 0.45) yield "HIGH";
            if (calibratedScore >= 0.20) yield "MODERATE";
            yield "LOW";
        }
        case "readmission" -> {
            // Readmission: higher thresholds (less immediately actionable)
            if (calibratedScore >= 0.80) yield "CRITICAL";
            if (calibratedScore >= 0.55) yield "HIGH";
            if (calibratedScore >= 0.30) yield "MODERATE";
            yield "LOW";
        }
        default -> {
            if (calibratedScore >= 0.75) yield "CRITICAL";
            if (calibratedScore >= 0.50) yield "HIGH";
            if (calibratedScore >= 0.25) yield "MODERATE";
            yield "LOW";
        }
    };
}
```

**IMPORTANT:** Sepsis thresholds are intentionally lower than other categories. A false-negative sepsis prediction (missing early sepsis) has far worse clinical outcomes than a false-positive (triggering an unnecessary blood culture). The threshold asymmetry reflects clinical utility, not model performance.

---

## 9. Explainability

### 9.1 Feature Importance

For tabular models (XGBoost, LightGBM exported to ONNX), pre-compute global feature importances during training and store them alongside the model. At inference time, multiply global importance by feature deviation from population mean:

```java
public class ExplanationGenerator {

    private static final Map<String, String[]> FEATURE_NAMES = Map.of(
        "base", new String[]{
            "Heart Rate", "Systolic BP", "Diastolic BP", "Respiratory Rate",
            "SpO2", "Temperature", "NEWS2", "qSOFA", "Acuity Score",
            "Event Count", "Hours Since Admission"
            // ... etc.
        }
    );

    public static Map<String, Double> explain(
        float[] features, float prediction, String category) {

        float[] importances = getGlobalImportances(category);
        // Compute local importance: global_importance × |feature_value - population_mean|
        Map<String, Double> localImportance = new TreeMap<>();
        for (int i = 0; i < Math.min(features.length, importances.length); i++) {
            double contribution = importances[i] * Math.abs(features[i] - getPopulationMean(i));
            localImportance.put(FEATURE_NAMES.get("base")[i], contribution);
        }

        // Return top 5 contributors
        return localImportance.entrySet().stream()
            .sorted(Map.Entry.<String, Double>comparingByValue().reversed())
            .limit(5)
            .collect(Collectors.toMap(Map.Entry::getKey, Map.Entry::getValue,
                (a, b) -> a, LinkedHashMap::new));
    }
}
```

### 9.2 Human-Readable Explanations

```java
// Generate clinical narrative from top features
public static List<String> generateTextExplanation(
    Map<String, Double> topFeatures, String category, double riskScore) {

    List<String> explanations = new ArrayList<>();

    if (topFeatures.containsKey("NEWS2") && topFeatures.get("NEWS2") > 0.3) {
        explanations.add("Elevated NEWS2 score contributing to increased " +
            category + " risk");
    }
    if (topFeatures.containsKey("Heart Rate") && topFeatures.get("Heart Rate") > 0.2) {
        explanations.add("Abnormal heart rate pattern detected");
    }
    // ... category-specific narratives

    return explanations;
}
```

---

## 10. Startup Without Models (Graceful Degradation)

Module 5 MUST NOT crash the pipeline if models are not yet available. This is the most common deployment scenario during initial setup and development.

```java
@Override
public void processElement1(CDSEvent event, Context ctx, Collector<MLPrediction> out)
    throws Exception {

    if (modelSessions.isEmpty()) {
        // No models loaded — pass through without predictions
        // Optionally emit a "NO_MODEL" prediction for monitoring
        LOG.debug("No ML models loaded — skipping inference for patient {}",
            event.getPatientId());
        return;
    }

    // ... normal inference path
}
```

---

## 11. Metrics and Monitoring

```java
// Register in open()
private transient Counter predictionsEmitted;
private transient Counter highRiskPredictions;
private transient Counter inferenceErrors;
private transient Gauge<Long> modelLoadTime;

@Override
public void open(OpenContext ctx) throws Exception {
    MetricGroup metrics = getRuntimeContext().getMetricGroup();
    predictionsEmitted = metrics.counter("predictions_emitted");
    highRiskPredictions = metrics.counter("high_risk_predictions");
    inferenceErrors = metrics.counter("inference_errors");

    // Per-category metrics
    for (String category : CATEGORIES) {
        metrics.gauge("model_loaded_" + category,
            () -> modelSessions.containsKey(category) ? 1L : 0L);
    }
}
```

---

## 12. Testing Strategy

Following Module 4's pattern — test extracted static logic without Flink runtime:

### 12.1 Test Classes

| Test Class | Count | Coverage |
|-----------|-------|----------|
| `Module5FeatureExtractionTest` | ~15 | Vital normalization, null safety, key mapping, demographic exclusion |
| `Module5CalibrationTest` | ~8 | Platt scaling, threshold classification, boundary values |
| `Module5ExplainabilityTest` | ~6 | Feature importance ranking, text generation |
| `Module5InferenceCooldownTest` | ~8 | Cooldown by risk level, escalation bypass, first-event |
| `Module5PatientStateTest` | ~6 | Ring buffer history, pattern buffering, state accumulation |
| `Module5RealDataMappingTest` | ~5 | Production CDS JSON → feature vector (same pattern as Module4RealCDSMappingTest) |

### 12.2 Highest-Priority Test (Write First)

```java
/**
 * Validates that real production CDS events produce valid feature vectors.
 * This is the Module 5 equivalent of Module4RealCDSMappingTest —
 * catches schema mismatches before they reach ONNX inference.
 */
@Test
void realCDSEvent_producesValidFeatureVector() {
    // Use actual JSON from comprehensive-cds-events.v1
    // Deserialize → build PatientMLState → extract features
    // Assert: no NaN, no Infinity, all values in [0,1] or [-1,1] range
    // Assert: vital keys resolve correctly
    // Assert: demographic fields (age, gender) excluded from vitals
}
```

---

## 13. File Structure

```
backend/shared-infrastructure/flink-processing/src/
├── main/java/com/cardiofit/flink/
│   ├── operators/Module5_MLInferenceEngine.java      # Main operator
│   ├── operators/Module5FeatureExtractor.java         # Feature extraction (testable)
│   ├── operators/Module5ClinicalScoring.java          # Risk classification (testable)
│   ├── models/MLPrediction.java                       # Output schema
│   ├── models/PatientMLState.java                     # Patient state for ML
│   ├── calibration/PredictionCalibrator.java           # Platt scaling
│   └── explanation/ExplanationGenerator.java           # Feature importance + text
├── test/java/com/cardiofit/flink/
│   ├── operators/Module5FeatureExtractionTest.java
│   ├── operators/Module5CalibrationTest.java
│   ├── operators/Module5ExplainabilityTest.java
│   ├── operators/Module5InferenceCooldownTest.java
│   ├── operators/Module5PatientStateTest.java
│   ├── operators/Module5RealDataMappingTest.java
│   └── builders/Module5TestBuilder.java               # Test data factory
└── resources/
    └── models/                                         # ONNX model files (gitignored)
        ├── sepsis/model.onnx
        ├── deterioration/model.onnx
        ├── readmission/model.onnx
        ├── fall/model.onnx
        └── mortality/model.onnx
```

---

## 14. Implementation Order

| Step | Task | Depends On |
|------|------|-----------|
| 1 | `MLPrediction.java` + `PatientMLState.java` — data models | Nothing |
| 2 | `Module5FeatureExtractor.java` — static feature extraction | Step 1 |
| 3 | `Module5RealDataMappingTest.java` — validate against production schemas | Step 2 |
| 4 | `Module5TestBuilder.java` — test data factory | Step 1 |
| 5 | `Module5FeatureExtractionTest.java` — full feature extraction tests | Steps 2, 4 |
| 6 | `PredictionCalibrator.java` + `Module5CalibrationTest.java` | Step 1 |
| 7 | `ExplanationGenerator.java` + `Module5ExplainabilityTest.java` | Step 2 |
| 8 | `Module5_MLInferenceEngine.java` — main operator | Steps 2, 6, 7 |
| 9 | `Module5InferenceCooldownTest.java` | Step 8 |
| 10 | E2E integration with Module 3 + 4 output | Steps 8, Module 4 E2E passing |

**Step 3 is non-negotiable before Step 8.** The Module 4 E2E exposed a snake_case vs camelCase serialization mismatch that silently dropped all events. Module 5 reads the same CDS events — validate the schema mapping against real production JSON before building the operator.

---

## 15. Lessons from Modules 1–4 (Apply to Module 5)

1. **Silent deserialization failures are the biggest risk.** Jackson with `FAIL_ON_UNKNOWN_PROPERTIES = false` will give you an object full of nulls and no error. Validate `patientId != null` at Module 5's entry point. Log and count rejected events.

2. **Null lab values cause NPEs.** Every `LabResult.getValue()` call must be null-checked before auto-unboxing. Apply the `safeLabValue()` pattern universally in feature extraction.

3. **Field name drift between modules is real.** Module 2 internal format, Module 3 CDS output format, and Module 4 SemanticEvent format all use different key conventions for the same vital signs. Module 5 must normalize keys at entry, not assume any particular format.

4. **FHIR calls inside Flink operators cause cascading failures.** Module 5 should NEVER make synchronous external calls during inference. All data needed for prediction must be in the CDS event or the patient state. If a feature requires data that isn't in the event, mark it as missing (-1.0) rather than fetching it.

5. **Checkpoint data + logs fill disk fast during restart loops.** Set `state.checkpoints.num-retained: 3` and configure log rotation before deploying Module 5. ONNX model files add to disk pressure — don't store multiple versions on the same node.

6. **Floating-point boundary comparisons need epsilon tolerance.** Module 4 discovered that `0.4 + 0.3 + 0.1 = 0.7999999999999999` in IEEE 754. Any threshold comparison in Module 5's risk classification should use `>= threshold - 1e-9`, not exact `>=`.
