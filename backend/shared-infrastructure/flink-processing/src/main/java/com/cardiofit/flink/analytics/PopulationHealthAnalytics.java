package com.cardiofit.flink.analytics;

import com.cardiofit.flink.models.MLPrediction;
import com.cardiofit.flink.models.PatientRiskProfile;
import com.cardiofit.flink.models.DepartmentStats;
import com.cardiofit.flink.models.PopulationMetrics;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.*;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Population Health Analytics for Module 6
 * Calculates department-level population health metrics from individual ML predictions
 *
 * Architecture:
 * - Input: ML predictions from Module 5 (inference-results.v1 topic)
 * - State: Patient risk profiles per department (MapState)
 * - State: Department aggregated statistics (ValueState)
 * - Output: Population metrics every minute (analytics-population-health topic)
 *
 * State Management:
 * - MapState<String, PatientRiskProfile>: Tracks individual patient risk profiles
 * - ValueState<DepartmentStats>: Maintains department-level aggregations
 * - Timer-based: Emits metrics every 60 seconds via processing time timers
 *
 * Risk Categories:
 * - LOW: 0.00-0.25
 * - MODERATE: 0.25-0.50
 * - HIGH: 0.50-0.75
 * - CRITICAL: 0.75-1.00
 *
 * Stale Data Handling:
 * - Patient profiles older than 24 hours are automatically removed
 * - Ensures metrics reflect current patient population
 *
 * Example Flow:
 * 1. ML prediction arrives for PAT-001 in ICU
 * 2. Update patient risk profile in MapState
 * 3. Recalculate department statistics (12 patients, 3 high-risk, avg mortality 0.18)
 * 4. On timer fire (every 60s), emit PopulationMetrics for ICU
 *
 * Usage:
 * DataStream<MLPrediction> predictions = ...
 * DataStream<PopulationMetrics> populationMetrics = predictions
 *     .keyBy(p -> extractDepartment(p))
 *     .process(new PopulationHealthAnalytics());
 */
public class PopulationHealthAnalytics
    extends KeyedProcessFunction<String, MLPrediction, PopulationMetrics> {

    private static final Logger LOG = LoggerFactory.getLogger(PopulationHealthAnalytics.class);
    private static final long METRIC_EMISSION_INTERVAL_MS = 60000L; // 1 minute
    private static final long STALE_PROFILE_AGE_MS = 24 * 3600 * 1000L; // 24 hours

    // State: Patient risk profiles per department
    private transient MapState<String, PatientRiskProfile> patientRiskState;

    // State: Department-level statistics
    private transient ValueState<DepartmentStats> departmentStatsState;

    // State: Track if timer has been registered
    private transient ValueState<Boolean> timerRegisteredState;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);
        LOG.info("Initializing Population Health Analytics state");

        // Initialize patient risk profile state
        MapStateDescriptor<String, PatientRiskProfile> patientRiskDescriptor =
            new MapStateDescriptor<>(
                "patient-risk-profiles",
                String.class,
                PatientRiskProfile.class
            );
        patientRiskState = getRuntimeContext().getMapState(patientRiskDescriptor);

        // Initialize department statistics state
        ValueStateDescriptor<DepartmentStats> statsDescriptor =
            new ValueStateDescriptor<>(
                "department-stats",
                DepartmentStats.class
            );
        departmentStatsState = getRuntimeContext().getState(statsDescriptor);

        // Initialize timer registration state
        ValueStateDescriptor<Boolean> timerDescriptor =
            new ValueStateDescriptor<>(
                "timer-registered",
                Boolean.class
            );
        timerRegisteredState = getRuntimeContext().getState(timerDescriptor);

        LOG.info("Population Health Analytics state initialized successfully");
    }

    @Override
    public void processElement(
        MLPrediction prediction,
        Context ctx,
        Collector<PopulationMetrics> out) throws Exception {

        String department = ctx.getCurrentKey();
        String patientId = prediction.getPatientId();

        LOG.debug("Processing ML prediction for patient {} in department {}", patientId, department);

        // Extract risk scores from prediction
        double mortalityRisk = extractRiskScore(prediction, "mortality");
        double sepsisRisk = extractRiskScore(prediction, "sepsis");
        double readmissionRisk = extractRiskScore(prediction, "readmission");

        // Calculate overall risk score (weighted average)
        double overallRiskScore = (mortalityRisk * 0.4) + (sepsisRisk * 0.4) + (readmissionRisk * 0.2);

        // Update patient risk profile in state
        PatientRiskProfile profile = PatientRiskProfile.builder()
            .patientId(patientId)
            .department(department)
            .mortalityRisk(mortalityRisk)
            .sepsisRisk(sepsisRisk)
            .readmissionRisk(readmissionRisk)
            .overallRiskScore(overallRiskScore)
            .lastUpdated(System.currentTimeMillis())
            .build();

        patientRiskState.put(patientId, profile);
        LOG.debug("Updated risk profile for patient {}: overall risk = {}", patientId, overallRiskScore);

        // Recalculate department statistics
        DepartmentStats stats = calculateDepartmentStats();
        departmentStatsState.update(stats);

        // Register timer for periodic metric emission (if not already registered)
        Boolean timerRegistered = timerRegisteredState.value();
        if (timerRegistered == null || !timerRegistered) {
            long nextTimerTime = System.currentTimeMillis() + METRIC_EMISSION_INTERVAL_MS;
            ctx.timerService().registerProcessingTimeTimer(nextTimerTime);
            timerRegisteredState.update(true);
            LOG.info("Registered population metrics timer for department {} at {}", department, nextTimerTime);
        }
    }

    @Override
    public void onTimer(
        long timestamp,
        OnTimerContext ctx,
        Collector<PopulationMetrics> out) throws Exception {

        String department = ctx.getCurrentKey();
        DepartmentStats stats = departmentStatsState.value();

        if (stats == null || stats.getTotalPatients() == 0) {
            LOG.debug("No active patients in department {}, skipping metric emission", department);
        } else {
            // Create and emit population metrics
            PopulationMetrics metrics = PopulationMetrics.builder()
                .department(department)
                .timestamp(timestamp)
                .totalPatients(stats.getTotalPatients())
                .highRiskPatients(stats.getHighRiskPatients())
                .criticalPatients(stats.getCriticalPatients())
                .avgMortalityRisk(stats.getAvgMortalityRisk())
                .avgSepsisRisk(stats.getAvgSepsisRisk())
                .riskDistribution(stats.getRiskDistribution())
                .trendIndicator(stats.getTrendIndicator())
                .build();

            out.collect(metrics);
            LOG.info("Emitted population metrics for department {}: {} total patients, {} high-risk, {} critical",
                     department, stats.getTotalPatients(), stats.getHighRiskPatients(), stats.getCriticalPatients());
        }

        // Register next timer
        long nextTimerTime = timestamp + METRIC_EMISSION_INTERVAL_MS;
        ctx.timerService().registerProcessingTimeTimer(nextTimerTime);
    }

    /**
     * Calculate department-level statistics from patient risk profiles
     * Iterates through all patient profiles in state and aggregates statistics
     */
    private DepartmentStats calculateDepartmentStats() throws Exception {
        int totalPatients = 0;
        int highRiskCount = 0;
        int criticalCount = 0;
        double mortalitySum = 0.0;
        double sepsisSum = 0.0;
        double readmissionSum = 0.0;

        Map<String, Integer> riskDistribution = new HashMap<>();
        riskDistribution.put("LOW", 0);
        riskDistribution.put("MODERATE", 0);
        riskDistribution.put("HIGH", 0);
        riskDistribution.put("CRITICAL", 0);

        // Iterate through all patient profiles
        Iterator<Map.Entry<String, PatientRiskProfile>> iterator =
            patientRiskState.iterator();

        while (iterator.hasNext()) {
            Map.Entry<String, PatientRiskProfile> entry = iterator.next();
            PatientRiskProfile profile = entry.getValue();

            // Remove stale profiles (>24 hours old)
            if (System.currentTimeMillis() - profile.getLastUpdated() > STALE_PROFILE_AGE_MS) {
                iterator.remove();
                LOG.debug("Removed stale patient profile: {}", profile.getPatientId());
                continue;
            }

            totalPatients++;
            mortalitySum += profile.getMortalityRisk();
            sepsisSum += profile.getSepsisRisk();
            readmissionSum += profile.getReadmissionRisk();

            double overallRisk = profile.getOverallRiskScore();

            // Categorize patient by risk level
            if (overallRisk >= 0.75) {
                criticalCount++;
                riskDistribution.put("CRITICAL", riskDistribution.get("CRITICAL") + 1);
            } else if (overallRisk >= 0.50) {
                highRiskCount++;
                riskDistribution.put("HIGH", riskDistribution.get("HIGH") + 1);
            } else if (overallRisk >= 0.25) {
                riskDistribution.put("MODERATE", riskDistribution.get("MODERATE") + 1);
            } else {
                riskDistribution.put("LOW", riskDistribution.get("LOW") + 1);
            }
        }

        // Build department statistics
        DepartmentStats stats = DepartmentStats.builder()
            .totalPatients(totalPatients)
            .highRiskPatients(highRiskCount)
            .criticalPatients(criticalCount)
            .avgMortalityRisk(totalPatients > 0 ? mortalitySum / totalPatients : 0.0)
            .avgSepsisRisk(totalPatients > 0 ? sepsisSum / totalPatients : 0.0)
            .riskDistribution(riskDistribution)
            .trendIndicator("STABLE")  // TODO: Calculate from historical data
            .build();

        LOG.debug("Calculated department stats: {} total patients, {} high-risk, {} critical",
                  totalPatients, highRiskCount, criticalCount);

        return stats;
    }

    /**
     * Extract risk score for a specific model type from ML prediction
     */
    private double extractRiskScore(MLPrediction prediction, String riskType) {
        // Check model type or model name
        String modelName = prediction.getModelName();
        String modelType = prediction.getModelType();

        if (modelName != null && modelName.toLowerCase().contains(riskType)) {
            return prediction.getPrimaryScore();
        } else if (modelType != null && modelType.toLowerCase().contains(riskType)) {
            return prediction.getPrimaryScore();
        }

        // Check prediction scores map
        Map<String, Double> scores = prediction.getPredictionScores();
        if (scores != null) {
            // Try various key patterns
            String[] keyPatterns = {
                riskType,
                riskType + "_risk",
                riskType + "_score",
                riskType + "_prediction",
                riskType + "_probability"
            };

            for (String key : keyPatterns) {
                Double score = scores.get(key);
                if (score != null) {
                    return score;
                }
            }
        }

        // Default to 0.0 if risk type not found
        return 0.0;
    }

    /**
     * Helper method to extract department from ML prediction
     * This should match the department field from enriched events
     */
    public static String extractDepartment(MLPrediction prediction) {
        // Try to get department from model metadata
        Map<String, Object> metadata = prediction.getModelMetadata();
        if (metadata != null) {
            Object dept = metadata.get("department");
            if (dept != null) {
                return dept.toString();
            }
        }

        // Default to UNKNOWN if department not found
        return "UNKNOWN";
    }
}
