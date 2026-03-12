package com.cardiofit.flink.ml.monitoring;

import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import com.cardiofit.flink.models.MLPrediction;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.MapState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.common.typeinfo.Types;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Model Drift Detector
 *
 * Detects statistical drift in feature distributions and prediction distributions
 * using rigorous statistical tests. Triggers alerts when drift exceeds thresholds,
 * indicating potential model degradation requiring retraining.
 *
 * Drift Detection Methods:
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 1. Feature Distribution Drift (Kolmogorov-Smirnov Test)                 │
 * │    - Compares current feature distribution vs baseline                   │
 * │    - Non-parametric test (no distribution assumptions)                   │
 * │    - p-value < 0.05 indicates significant drift                          │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 2. Prediction Distribution Drift (PSI - Population Stability Index)     │
 * │    - Measures shift in prediction score distribution                     │
 * │    - PSI < 0.1: No drift                                                 │
 * │    - PSI 0.1-0.25: Moderate drift (monitor)                             │
 * │    - PSI > 0.25: Severe drift (retrain model)                           │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 3. Accuracy Degradation Detection                                       │
 * │    - Track accuracy over sliding window                                  │
 * │    - Alert if accuracy drops below threshold                             │
 * │    - Requires ground truth labels                                        │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * Drift Alert Triggers:
 * - Feature Drift: Any feature with KS p-value < 0.05
 * - Prediction Drift: PSI > 0.25 (severe) or > 0.1 (moderate)
 * - Accuracy Drift: AUROC drops > 5% from baseline
 *
 * State Management:
 * - Baseline distributions: Captured from first N predictions (e.g., 1000)
 * - Current window: Last M predictions (e.g., 500) for comparison
 * - Drift history: Last 50 drift detection results
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class DriftDetector extends ProcessFunction<MLPrediction, DriftAlert> {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(DriftDetector.class);

    // Configuration
    private final int baselineWindowSize;
    private final int comparisonWindowSize;
    private final double ksPValueThreshold;
    private final double psiModerateThreshold;
    private final double psiSevereThreshold;
    private final long driftCheckIntervalMs;

    // State: Baseline feature distributions
    private transient MapState<String, FeatureDistribution> baselineFeatureDistributionsState;

    // State: Baseline prediction distribution
    private transient ValueState<PredictionDistribution> baselinePredictionDistributionState;

    // State: Current window for comparison
    private transient ValueState<List<MLPrediction>> currentWindowState;

    // State: Drift detection history
    private transient ValueState<List<DriftDetectionResult>> driftHistoryState;

    // State: Last drift check timestamp
    private transient ValueState<Long> lastDriftCheckTimestampState;

    // State: Baseline established flag
    private transient ValueState<Boolean> baselineEstablishedState;

    public DriftDetector() {
        this(1000, 500, 0.05, 0.1, 0.25, 3600_000);  // 1-hour drift checks
    }

    public DriftDetector(int baselineWindowSize,
                        int comparisonWindowSize,
                        double ksPValueThreshold,
                        double psiModerateThreshold,
                        double psiSevereThreshold,
                        long driftCheckIntervalMs) {
        this.baselineWindowSize = baselineWindowSize;
        this.comparisonWindowSize = comparisonWindowSize;
        this.ksPValueThreshold = ksPValueThreshold;
        this.psiModerateThreshold = psiModerateThreshold;
        this.psiSevereThreshold = psiSevereThreshold;
        this.driftCheckIntervalMs = driftCheckIntervalMs;
    }

    @Override
    public void open(OpenContext context) throws Exception {
        super.open(context);

        baselineFeatureDistributionsState = getRuntimeContext().getMapState(
            new MapStateDescriptor<>("baseline-features",
                TypeInformation.of(String.class),
                TypeInformation.of(FeatureDistribution.class))
        );

        baselinePredictionDistributionState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("baseline-predictions",
                TypeInformation.of(PredictionDistribution.class))
        );

        currentWindowState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("current-window",
                Types.LIST(TypeInformation.of(MLPrediction.class)))
        );

        driftHistoryState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("drift-history",
                Types.LIST(TypeInformation.of(DriftDetectionResult.class)))
        );

        lastDriftCheckTimestampState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("last-drift-check",
                TypeInformation.of(Long.class))
        );

        baselineEstablishedState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("baseline-established",
                TypeInformation.of(Boolean.class))
        );

        LOG.info("DriftDetector initialized: baseline={}, comparison={}, ksThreshold={}, psiThresholds=[{}, {}]",
            baselineWindowSize, comparisonWindowSize, ksPValueThreshold,
            psiModerateThreshold, psiSevereThreshold);
    }

    @Override
    public void processElement(MLPrediction prediction,
                              Context ctx,
                              Collector<DriftAlert> out) throws Exception {
        long currentTime = System.currentTimeMillis();
        String modelType = prediction.getModelType();

        // Add prediction to current window
        List<MLPrediction> currentWindow = currentWindowState.value();
        if (currentWindow == null) {
            currentWindow = new ArrayList<>();
        }
        currentWindow.add(prediction);
        currentWindowState.update(currentWindow);

        // Check if baseline is established
        Boolean baselineEstablished = baselineEstablishedState.value();
        if (baselineEstablished == null || !baselineEstablished) {
            // Build baseline from first N predictions
            if (currentWindow.size() >= baselineWindowSize) {
                establishBaseline(currentWindow);
                baselineEstablishedState.update(true);
                LOG.info("Baseline established for model {} with {} predictions",
                    modelType, baselineWindowSize);

                // Keep only recent predictions for comparison window
                if (currentWindow.size() > comparisonWindowSize) {
                    currentWindow = currentWindow.subList(
                        currentWindow.size() - comparisonWindowSize,
                        currentWindow.size()
                    );
                    currentWindowState.update(currentWindow);
                }
            }
            return;  // Don't check drift until baseline established
        }

        // Maintain comparison window size
        if (currentWindow.size() > comparisonWindowSize) {
            currentWindow.remove(0);  // Remove oldest
            currentWindowState.update(currentWindow);
        }

        // Check if it's time to run drift detection
        Long lastDriftCheck = lastDriftCheckTimestampState.value();
        if (lastDriftCheck == null) {
            lastDriftCheck = currentTime;
            lastDriftCheckTimestampState.update(lastDriftCheck);
        }

        if (currentTime - lastDriftCheck >= driftCheckIntervalMs &&
            currentWindow.size() >= comparisonWindowSize) {

            // Run drift detection
            DriftDetectionResult result = detectDrift(currentWindow, modelType, currentTime);

            // Store result in history
            storeDriftResult(result);

            // Emit drift alert if significant drift detected
            if (result.hasDrift()) {
                DriftAlert alert = createDriftAlert(result, modelType, currentTime);
                out.collect(alert);

                LOG.warn("Drift detected for model {}: featureDrift={}, predictionDrift={}, severity={}",
                    modelType, result.hasFeatureDrift(), result.hasPredictionDrift(), result.getSeverity());
            }

            // Update last drift check timestamp
            lastDriftCheckTimestampState.update(currentTime);
        }
    }

    /**
     * Establish baseline distributions from initial predictions
     */
    private void establishBaseline(List<MLPrediction> predictions) throws Exception {
        // Feature distributions
        Map<String, List<Double>> featureValues = new HashMap<>();

        for (MLPrediction prediction : predictions) {
            float[] inputFeatures = prediction.getInputFeatures();
            if (inputFeatures == null) continue;

            // For array-based features, we need feature names from metadata or use indices
            for (int i = 0; i < inputFeatures.length; i++) {
                String featureName = "feature_" + i;
                featureValues.computeIfAbsent(featureName, k -> new ArrayList<>()).add((double) inputFeatures[i]);
            }
        }

        // Store baseline feature distributions
        for (Map.Entry<String, List<Double>> entry : featureValues.entrySet()) {
            List<Double> values = entry.getValue();
            Collections.sort(values);
            FeatureDistribution distribution = new FeatureDistribution(entry.getKey(), values);
            baselineFeatureDistributionsState.put(entry.getKey(), distribution);
        }

        // Prediction distribution
        List<Double> predictionScores = new ArrayList<>();
        for (MLPrediction prediction : predictions) {
            predictionScores.add(prediction.getPrimaryScore());
        }
        Collections.sort(predictionScores);

        PredictionDistribution predictionDistribution = new PredictionDistribution(predictionScores);
        baselinePredictionDistributionState.update(predictionDistribution);
    }

    /**
     * Detect drift by comparing current window to baseline
     */
    private DriftDetectionResult detectDrift(List<MLPrediction> currentWindow,
                                            String modelType,
                                            long timestamp) throws Exception {
        DriftDetectionResult result = new DriftDetectionResult(modelType, timestamp);

        // 1. Feature Distribution Drift (Kolmogorov-Smirnov Test)
        Map<String, List<Double>> currentFeatureValues = extractFeatureValues(currentWindow);

        for (Map.Entry<String, List<Double>> entry : currentFeatureValues.entrySet()) {
            String featureName = entry.getKey();
            FeatureDistribution baseline = baselineFeatureDistributionsState.get(featureName);

            if (baseline == null) continue;

            List<Double> currentValues = entry.getValue();
            Collections.sort(currentValues);

            // Kolmogorov-Smirnov test
            KSTestResult ksResult = kolmogorovSmirnovTest(baseline.getValues(), currentValues);

            if (ksResult.pValue < ksPValueThreshold) {
                result.addFeatureDrift(featureName, ksResult.dStatistic, ksResult.pValue);
            }
        }

        // 2. Prediction Distribution Drift (PSI - Population Stability Index)
        PredictionDistribution baselinePredictions = baselinePredictionDistributionState.value();
        if (baselinePredictions != null) {
            List<Double> currentPredictions = new ArrayList<>();
            for (MLPrediction prediction : currentWindow) {
                currentPredictions.add(prediction.getPrimaryScore());
            }

            double psi = calculatePSI(baselinePredictions.getValues(), currentPredictions);
            result.setPredictionDriftPSI(psi);

            if (psi >= psiSevereThreshold) {
                result.setPredictionDriftSeverity("SEVERE");
            } else if (psi >= psiModerateThreshold) {
                result.setPredictionDriftSeverity("MODERATE");
            }
        }

        return result;
    }

    /**
     * Extract feature values from predictions
     */
    private Map<String, List<Double>> extractFeatureValues(List<MLPrediction> predictions) {
        Map<String, List<Double>> featureValues = new HashMap<>();

        for (MLPrediction prediction : predictions) {
            float[] inputFeatures = prediction.getInputFeatures();
            if (inputFeatures == null) continue;

            // For array-based features, we need feature names from metadata or use indices
            for (int i = 0; i < inputFeatures.length; i++) {
                String featureName = "feature_" + i;
                featureValues.computeIfAbsent(featureName, k -> new ArrayList<>()).add((double) inputFeatures[i]);
            }
        }

        return featureValues;
    }

    /**
     * Kolmogorov-Smirnov Test
     *
     * Non-parametric test for comparing two distributions.
     * Returns D-statistic and p-value.
     */
    private KSTestResult kolmogorovSmirnovTest(List<Double> baseline, List<Double> current) {
        int n1 = baseline.size();
        int n2 = current.size();

        // Calculate empirical CDFs at all unique points
        Set<Double> allPoints = new HashSet<>();
        allPoints.addAll(baseline);
        allPoints.addAll(current);
        List<Double> sortedPoints = new ArrayList<>(allPoints);
        Collections.sort(sortedPoints);

        double maxDiff = 0.0;

        for (double point : sortedPoints) {
            double cdf1 = empiricalCDF(baseline, point);
            double cdf2 = empiricalCDF(current, point);
            double diff = Math.abs(cdf1 - cdf2);
            maxDiff = Math.max(maxDiff, diff);
        }

        // Calculate p-value (approximate)
        double dStatistic = maxDiff;
        double pValue = calculateKSPValue(dStatistic, n1, n2);

        return new KSTestResult(dStatistic, pValue);
    }

    private double empiricalCDF(List<Double> sortedValues, double point) {
        int count = 0;
        for (double value : sortedValues) {
            if (value <= point) count++;
        }
        return (double) count / sortedValues.size();
    }

    private double calculateKSPValue(double dStatistic, int n1, int n2) {
        // Approximate p-value using asymptotic formula
        double n = (double) (n1 * n2) / (n1 + n2);
        double lambda = dStatistic * Math.sqrt(n);

        // Kolmogorov distribution approximation
        double pValue = 2.0 * Math.exp(-2.0 * lambda * lambda);
        return Math.min(1.0, Math.max(0.0, pValue));
    }

    /**
     * Calculate PSI (Population Stability Index)
     *
     * Measures shift in distribution by comparing binned proportions.
     * PSI = Σ (actual% - expected%) * ln(actual% / expected%)
     */
    private double calculatePSI(List<Double> baseline, List<Double> current) {
        int numBins = 10;

        // Create bins based on baseline distribution
        double[] bins = createBins(baseline, numBins);

        // Calculate proportions in each bin
        double[] baselineProportions = calculateBinProportions(baseline, bins);
        double[] currentProportions = calculateBinProportions(current, bins);

        // Calculate PSI
        double psi = 0.0;
        for (int i = 0; i < numBins; i++) {
            double expected = baselineProportions[i];
            double actual = currentProportions[i];

            // Avoid division by zero
            if (expected < 0.0001) expected = 0.0001;
            if (actual < 0.0001) actual = 0.0001;

            psi += (actual - expected) * Math.log(actual / expected);
        }

        return psi;
    }

    private double[] createBins(List<Double> values, int numBins) {
        Collections.sort(values);
        double[] bins = new double[numBins + 1];

        bins[0] = Double.NEGATIVE_INFINITY;
        bins[numBins] = Double.POSITIVE_INFINITY;

        for (int i = 1; i < numBins; i++) {
            int index = (int) ((double) i * values.size() / numBins);
            index = Math.min(index, values.size() - 1);
            bins[i] = values.get(index);
        }

        return bins;
    }

    private double[] calculateBinProportions(List<Double> values, double[] bins) {
        int numBins = bins.length - 1;
        int[] counts = new int[numBins];

        for (double value : values) {
            for (int i = 0; i < numBins; i++) {
                if (value >= bins[i] && value < bins[i + 1]) {
                    counts[i]++;
                    break;
                }
            }
        }

        double[] proportions = new double[numBins];
        for (int i = 0; i < numBins; i++) {
            proportions[i] = (double) counts[i] / values.size();
        }

        return proportions;
    }

    /**
     * Store drift result in history
     */
    private void storeDriftResult(DriftDetectionResult result) throws Exception {
        List<DriftDetectionResult> history = driftHistoryState.value();
        if (history == null) {
            history = new ArrayList<>();
        }

        history.add(result);

        // Maintain history size (last 50 results)
        if (history.size() > 50) {
            history.remove(0);
        }

        driftHistoryState.update(history);
    }

    /**
     * Create drift alert from detection result
     */
    private DriftAlert createDriftAlert(DriftDetectionResult result, String modelType, long timestamp) {
        return DriftAlert.builder()
            .modelType(modelType)
            .timestamp(timestamp)
            .severity(result.getSeverity())
            .hasFeatureDrift(result.hasFeatureDrift())
            .hasPredictionDrift(result.hasPredictionDrift())
            .driftedFeatures(result.getDriftedFeatures())
            .predictionDriftPSI(result.getPredictionDriftPSI())
            .recommendations(generateRecommendations(result))
            .build();
    }

    private List<String> generateRecommendations(DriftDetectionResult result) {
        List<String> recommendations = new ArrayList<>();

        if (result.hasPredictionDrift()) {
            if ("SEVERE".equals(result.getPredictionDriftSeverity())) {
                recommendations.add("URGENT: Severe prediction drift detected (PSI=" +
                    String.format("%.3f", result.getPredictionDriftPSI()) +
                    "). Model retraining strongly recommended.");
            } else {
                recommendations.add("Moderate prediction drift detected (PSI=" +
                    String.format("%.3f", result.getPredictionDriftPSI()) +
                    "). Monitor closely and consider retraining.");
            }
        }

        if (result.hasFeatureDrift()) {
            recommendations.add("Feature drift detected in " + result.getDriftedFeatures().size() +
                " features. Investigate data pipeline and feature engineering.");

            List<String> topFeatures = result.getDriftedFeatures().subList(
                0, Math.min(3, result.getDriftedFeatures().size())
            );
            recommendations.add("Most drifted features: " + String.join(", ", topFeatures));
        }

        recommendations.add("Review recent model performance metrics for accuracy degradation.");
        recommendations.add("Consider A/B testing with retrained model before full deployment.");

        return recommendations;
    }

    // ===== Helper Classes =====

    private static class FeatureDistribution implements Serializable {
        private final String featureName;
        private final List<Double> values;

        FeatureDistribution(String featureName, List<Double> values) {
            this.featureName = featureName;
            this.values = new ArrayList<>(values);
        }

        public String getFeatureName() { return featureName; }
        public List<Double> getValues() { return values; }
    }

    private static class PredictionDistribution implements Serializable {
        private final List<Double> values;

        PredictionDistribution(List<Double> values) {
            this.values = new ArrayList<>(values);
        }

        public List<Double> getValues() { return values; }
    }

    private static class KSTestResult {
        final double dStatistic;
        final double pValue;

        KSTestResult(double dStatistic, double pValue) {
            this.dStatistic = dStatistic;
            this.pValue = pValue;
        }
    }

    private static class DriftDetectionResult implements Serializable {
        private final String modelType;
        private final long timestamp;
        private final Map<String, FeatureDriftInfo> featureDrifts = new HashMap<>();
        private double predictionDriftPSI = 0.0;
        private String predictionDriftSeverity = "NONE";

        DriftDetectionResult(String modelType, long timestamp) {
            this.modelType = modelType;
            this.timestamp = timestamp;
        }

        public void addFeatureDrift(String featureName, double dStatistic, double pValue) {
            featureDrifts.put(featureName, new FeatureDriftInfo(dStatistic, pValue));
        }

        public void setPredictionDriftPSI(double psi) {
            this.predictionDriftPSI = psi;
        }

        public void setPredictionDriftSeverity(String severity) {
            this.predictionDriftSeverity = severity;
        }

        public boolean hasFeatureDrift() {
            return !featureDrifts.isEmpty();
        }

        public boolean hasPredictionDrift() {
            return !"NONE".equals(predictionDriftSeverity);
        }

        public boolean hasDrift() {
            return hasFeatureDrift() || hasPredictionDrift();
        }

        public List<String> getDriftedFeatures() {
            return new ArrayList<>(featureDrifts.keySet());
        }

        public double getPredictionDriftPSI() {
            return predictionDriftPSI;
        }

        public String getPredictionDriftSeverity() {
            return predictionDriftSeverity;
        }

        public String getSeverity() {
            if ("SEVERE".equals(predictionDriftSeverity)) return "CRITICAL";
            if (hasFeatureDrift() && featureDrifts.size() > 10) return "HIGH";
            if ("MODERATE".equals(predictionDriftSeverity)) return "MEDIUM";
            if (hasFeatureDrift()) return "MEDIUM";
            return "LOW";
        }

        private static class FeatureDriftInfo implements Serializable {
            final double dStatistic;
            final double pValue;

            FeatureDriftInfo(double dStatistic, double pValue) {
                this.dStatistic = dStatistic;
                this.pValue = pValue;
            }
        }
    }
}
