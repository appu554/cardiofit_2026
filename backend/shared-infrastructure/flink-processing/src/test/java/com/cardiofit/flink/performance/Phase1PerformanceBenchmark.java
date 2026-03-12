package com.cardiofit.flink.performance;

import com.cardiofit.flink.alerts.SmartAlertGenerator;
import com.cardiofit.flink.indicators.EnhancedRiskIndicators;
import com.cardiofit.flink.models.PatientSnapshot;
import com.cardiofit.flink.scoring.*;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Performance Benchmark Tests for Phase 1 Components
 *
 * Validates against spec performance targets from MODULE2_ADVANCED_ENHANCEMENTS.md lines 495-500:
 * - Enrichment Latency: <100ms (P95)
 * - Alert Generation: <10ms
 * - Protocol Matching: <20ms
 * - Score Calculations: <50ms
 * - Neo4j Advanced Queries: <200ms
 *
 * This test measures single-threaded performance for Phase 1 clinical intelligence calculations.
 */
public class Phase1PerformanceBenchmark {

    private static final int WARMUP_ITERATIONS = 100;
    private static final int BENCHMARK_ITERATIONS = 1000;

    // Spec targets (in milliseconds)
    private static final double ALERT_GENERATION_TARGET = 10.0;
    private static final double SCORE_CALCULATION_TARGET = 50.0;
    private static final double ENRICHMENT_LATENCY_TARGET = 100.0;

    @Test
    public void benchmarkEnhancedRiskAssessment() {
        PatientSnapshot snapshot = createTestPatient();
        Map<String, Object> vitals = createTestVitals();

        // Warmup
        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            EnhancedRiskIndicators.assessRisk(snapshot, vitals);
        }

        // Benchmark
        long startTime = System.nanoTime();
        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            EnhancedRiskIndicators.assessRisk(snapshot, vitals);
        }
        long endTime = System.nanoTime();

        double avgTimeMs = (endTime - startTime) / (BENCHMARK_ITERATIONS * 1_000_000.0);

        System.out.printf("Enhanced Risk Assessment: %.3f ms/call%n", avgTimeMs);
        assertTrue(avgTimeMs < 5.0, "Risk assessment should be <5ms per call");
    }

    @Test
    public void benchmarkNEWS2Calculation() {
        Map<String, Object> vitals = createTestVitals();

        // Warmup
        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            NEWS2Calculator.calculate(vitals, false);
        }

        // Benchmark
        long startTime = System.nanoTime();
        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            NEWS2Calculator.calculate(vitals, false);
        }
        long endTime = System.nanoTime();

        double avgTimeMs = (endTime - startTime) / (BENCHMARK_ITERATIONS * 1_000_000.0);

        System.out.printf("NEWS2 Calculation: %.3f ms/call%n", avgTimeMs);
        assertTrue(avgTimeMs < 2.0, "NEWS2 calculation should be <2ms per call");
    }

    @Test
    public void benchmarkMetabolicAcuityCalculation() {
        PatientSnapshot snapshot = createTestPatient();
        Map<String, Object> vitals = createTestVitals();
        Map<String, Object> labs = createTestLabs();

        // Warmup
        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);
        }

        // Benchmark
        long startTime = System.nanoTime();
        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);
        }
        long endTime = System.nanoTime();

        double avgTimeMs = (endTime - startTime) / (BENCHMARK_ITERATIONS * 1_000_000.0);

        System.out.printf("Metabolic Acuity Calculation: %.3f ms/call%n", avgTimeMs);
        assertTrue(avgTimeMs < 5.0, "Metabolic acuity calculation should be <5ms per call");
    }

    @Test
    public void benchmarkCombinedAcuityCalculation() {
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(5);
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic = createMetabolicScore(3.0);

        // Warmup
        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            CombinedAcuityCalculator.calculate(news2, metabolic);
        }

        // Benchmark
        long startTime = System.nanoTime();
        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            CombinedAcuityCalculator.calculate(news2, metabolic);
        }
        long endTime = System.nanoTime();

        double avgTimeMs = (endTime - startTime) / (BENCHMARK_ITERATIONS * 1_000_000.0);

        System.out.printf("Combined Acuity Calculation: %.3f ms/call%n", avgTimeMs);
        assertTrue(avgTimeMs < 1.0, "Combined acuity calculation should be <1ms per call");
    }

    @Test
    public void benchmarkAlertGeneration() {
        String patientId = "TEST-PATIENT-001";
        EnhancedRiskIndicators.RiskAssessment risk = createRiskAssessment();
        NEWS2Calculator.NEWS2Score news2 = createNEWS2Score(7);
        Map<String, Object> vitals = createTestVitals();

        // Warmup
        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            SmartAlertGenerator.generateAlerts(patientId, risk, news2, vitals);
        }

        // Benchmark
        long startTime = System.nanoTime();
        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            SmartAlertGenerator.generateAlerts(patientId, risk, news2, vitals);
        }
        long endTime = System.nanoTime();

        double avgTimeMs = (endTime - startTime) / (BENCHMARK_ITERATIONS * 1_000_000.0);

        System.out.printf("Alert Generation: %.3f ms/call%n", avgTimeMs);
        assertTrue(avgTimeMs < ALERT_GENERATION_TARGET,
            String.format("Alert generation should be <%dms per call (spec target)", (int)ALERT_GENERATION_TARGET));
    }

    @Test
    public void benchmarkAllClinicalScores() {
        PatientSnapshot snapshot = createTestPatient();
        Map<String, Object> vitals = createTestVitals();
        Map<String, Object> labs = createTestLabs();

        // Warmup
        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            ClinicalScoreCalculators.calculateFraminghamScore(snapshot, labs);
            ClinicalScoreCalculators.calculateCHADS2VAScScore(snapshot);
            ClinicalScoreCalculators.calculateQSOFAScore(vitals);
            ClinicalScoreCalculators.calculateMetabolicSyndromeScore(snapshot, vitals, labs);
        }

        // Benchmark
        long startTime = System.nanoTime();
        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            ClinicalScoreCalculators.calculateFraminghamScore(snapshot, labs);
            ClinicalScoreCalculators.calculateCHADS2VAScScore(snapshot);
            ClinicalScoreCalculators.calculateQSOFAScore(vitals);
            ClinicalScoreCalculators.calculateMetabolicSyndromeScore(snapshot, vitals, labs);
        }
        long endTime = System.nanoTime();

        double avgTimeMs = (endTime - startTime) / (BENCHMARK_ITERATIONS * 1_000_000.0);

        System.out.printf("All Clinical Scores (4 calculators): %.3f ms/call%n", avgTimeMs);
        assertTrue(avgTimeMs < SCORE_CALCULATION_TARGET,
            String.format("All score calculations should be <%dms per call (spec target)", (int)SCORE_CALCULATION_TARGET));
    }

    @Test
    public void benchmarkCompleteClinicalIntelligence() {
        // Simulates complete Phase 1 pipeline
        PatientSnapshot snapshot = createTestPatient();
        Map<String, Object> vitals = createTestVitals();
        Map<String, Object> labs = createTestLabs();
        String patientId = "TEST-PATIENT-001";

        // Warmup
        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            calculateCompleteClinicalIntelligence(snapshot, vitals, labs, patientId);
        }

        // Benchmark
        long startTime = System.nanoTime();
        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            calculateCompleteClinicalIntelligence(snapshot, vitals, labs, patientId);
        }
        long endTime = System.nanoTime();

        double avgTimeMs = (endTime - startTime) / (BENCHMARK_ITERATIONS * 1_000_000.0);

        System.out.printf("Complete Clinical Intelligence (All Phase 1): %.3f ms/call%n", avgTimeMs);
        assertTrue(avgTimeMs < ENRICHMENT_LATENCY_TARGET,
            String.format("Complete Phase 1 enrichment should be <%dms per call (spec target)", (int)ENRICHMENT_LATENCY_TARGET));

        // Log P95 estimate (single-threaded approximation)
        System.out.printf("Estimated P95: %.3f ms (single-threaded)%n", avgTimeMs * 1.5);
    }

    @Test
    public void benchmarkP95Latency() {
        // Measure P95 latency for complete Phase 1 pipeline
        PatientSnapshot snapshot = createTestPatient();
        Map<String, Object> vitals = createTestVitals();
        Map<String, Object> labs = createTestLabs();
        String patientId = "TEST-PATIENT-001";

        double[] latencies = new double[BENCHMARK_ITERATIONS];

        // Warmup
        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            calculateCompleteClinicalIntelligence(snapshot, vitals, labs, patientId);
        }

        // Measure individual latencies
        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            long start = System.nanoTime();
            calculateCompleteClinicalIntelligence(snapshot, vitals, labs, patientId);
            long end = System.nanoTime();
            latencies[i] = (end - start) / 1_000_000.0; // Convert to ms
        }

        // Sort to find P95
        java.util.Arrays.sort(latencies);
        int p95Index = (int) (BENCHMARK_ITERATIONS * 0.95);
        double p95Latency = latencies[p95Index];

        System.out.printf("P95 Latency: %.3f ms%n", p95Latency);
        System.out.printf("P50 Latency: %.3f ms%n", latencies[BENCHMARK_ITERATIONS / 2]);
        System.out.printf("P99 Latency: %.3f ms%n", latencies[(int) (BENCHMARK_ITERATIONS * 0.99)]);
        System.out.printf("Max Latency: %.3f ms%n", latencies[BENCHMARK_ITERATIONS - 1]);

        assertTrue(p95Latency < ENRICHMENT_LATENCY_TARGET,
            String.format("P95 latency should be <%.0fms (spec target)", ENRICHMENT_LATENCY_TARGET));
    }

    // Simulation of complete Phase 1 pipeline
    private void calculateCompleteClinicalIntelligence(
            PatientSnapshot snapshot,
            Map<String, Object> vitals,
            Map<String, Object> labs,
            String patientId) {

        // 1. Enhanced Risk Assessment
        EnhancedRiskIndicators.RiskAssessment risk =
            EnhancedRiskIndicators.assessRisk(snapshot, vitals);

        // 2. NEWS2 Scoring
        NEWS2Calculator.NEWS2Score news2 =
            NEWS2Calculator.calculate(vitals, false);

        // 3. Metabolic Acuity
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolic =
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);

        // 4. Combined Acuity
        CombinedAcuityCalculator.CombinedAcuityScore combined =
            CombinedAcuityCalculator.calculate(news2, metabolic);

        // 5. Alert Generation
        List<SmartAlertGenerator.ClinicalAlert> alerts =
            SmartAlertGenerator.generateAlerts(patientId, risk, news2, vitals);

        // 6. Clinical Scores
        ClinicalScoreCalculators.calculateFraminghamScore(snapshot, labs);
        ClinicalScoreCalculators.calculateCHADS2VAScScore(snapshot);
        ClinicalScoreCalculators.calculateQSOFAScore(vitals);
        ClinicalScoreCalculators.calculateMetabolicSyndromeScore(snapshot, vitals, labs);

        // 7. Confidence Scoring
        ConfidenceScoreCalculator.calculateConfidence(snapshot, vitals, labs, "COMPREHENSIVE");
    }

    // Helper methods

    private PatientSnapshot createTestPatient() {
        PatientSnapshot snapshot = new PatientSnapshot("TEST-PATIENT-001");
        snapshot.setAge(58);
        snapshot.setGender("male");
        return snapshot;
    }

    private Map<String, Object> createTestVitals() {
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartRate", 115);
        vitals.put("systolicBP", 145);
        vitals.put("diastolicBP", 92);
        vitals.put("respiratoryRate", 18);
        vitals.put("oxygenSaturation", 96.0);
        vitals.put("temperature", 98.6);
        vitals.put("bmi", 32.0);
        vitals.put("timestamp", System.currentTimeMillis());
        return vitals;
    }

    private Map<String, Object> createTestLabs() {
        Map<String, Object> labs = new HashMap<>();
        labs.put("totalCholesterol", 240);
        labs.put("hdlCholesterol", 38);
        labs.put("ldlCholesterol", 165);
        labs.put("triglycerides", 185);
        labs.put("glucose", 115);
        return labs;
    }

    private NEWS2Calculator.NEWS2Score createNEWS2Score(int total) {
        NEWS2Calculator.NEWS2Score score = new NEWS2Calculator.NEWS2Score();
        score.setTotalScore(total);
        score.setRiskLevel(total >= 7 ? "HIGH" : total >= 5 ? "MEDIUM" : "LOW");
        return score;
    }

    private MetabolicAcuityCalculator.MetabolicAcuityScore createMetabolicScore(double value) {
        MetabolicAcuityCalculator.MetabolicAcuityScore score =
            new MetabolicAcuityCalculator.MetabolicAcuityScore();
        score.setScore(value);
        score.setRiskLevel(value >= 3 ? "HIGH" : value >= 2 ? "MODERATE" : "LOW");
        return score;
    }

    private EnhancedRiskIndicators.RiskAssessment createRiskAssessment() {
        PatientSnapshot snapshot = createTestPatient();
        Map<String, Object> vitals = createTestVitals();
        return EnhancedRiskIndicators.assessRisk(snapshot, vitals);
    }
}
