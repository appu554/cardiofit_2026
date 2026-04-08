package com.cardiofit.flink.ml;

import com.cardiofit.flink.ml.PatientContextSnapshot;
import com.cardiofit.flink.models.MLPrediction;
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import com.cardiofit.flink.ml.features.ClinicalFeatureExtractor;
import com.cardiofit.flink.ml.ONNXModelContainer;
import com.cardiofit.flink.ml.ModelConfig;
import com.cardiofit.flink.ml.util.TestDataFactory;

import org.junit.jupiter.api.*;

import static org.assertj.core.api.Assertions.*;

import java.nio.file.Files;
import java.nio.file.Paths;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.stream.Collectors;

/**
 * Module 5 Performance Benchmark Suite
 *
 * Comprehensive performance benchmarks for ML inference pipeline:
 * - Latency profiling (p50, p95, p99 percentiles)
 * - Throughput measurement (predictions per second)
 * - Batch optimization (optimal batch size)
 * - Memory usage profiling
 * - Parallel speedup analysis
 *
 * Target Metrics:
 * - p99 latency: <15ms
 * - Throughput: >100 predictions/second
 * - Batch size 32: <50ms
 * - Memory usage: <500MB
 * - Parallel speedup: >2x
 *
 * Prerequisites:
 * - Mock ONNX models generated (sepsis, deterioration, mortality, readmission)
 * - ONNXModelContainer infrastructure
 * - ClinicalFeatureExtractor
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
@Disabled("Superseded by Module5OnnxIntegrationTest (55-feature v3.0.0 pipeline). " +
    "This benchmark uses the old 70-feature ClinicalFeatureExtractor and v1.0.0 model paths.")
@DisplayName("Module 5: ML Inference Performance Benchmarks")
public class Module5PerformanceBenchmark {

    private static final String MODELS_DIR = "models";
    private static final int WARMUP_ITERATIONS = 1000;

    // Models
    private static ONNXModelContainer sepsisModel;
    private static ONNXModelContainer deteriorationModel;
    private static ONNXModelContainer mortalityModel;
    private static ONNXModelContainer readmissionModel;

    // Test data
    private static List<PatientContextSnapshot> testPatients;
    private static ClinicalFeatureExtractor extractor;

    @BeforeAll
    static void setupBenchmarks() throws Exception {
        System.out.println("\n" + "=".repeat(70));
        System.out.println("MODULE 5 PERFORMANCE BENCHMARK SUITE");
        System.out.println("=".repeat(70));
        System.out.println();

        // Load all 4 models
        System.out.println("📦 Loading ONNX models...");

        ModelConfig sepsisConfig = ModelConfig.builder()
            .modelPath(MODELS_DIR + "/sepsis_risk_v1.0.0.onnx")
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.7)
            .build();

        sepsisModel = ONNXModelContainer.builder()
            .modelId("sepsis_risk_v1")
            .modelName("Sepsis Risk Predictor")
            .modelType(ONNXModelContainer.ModelType.SEPSIS_ONSET)
            .modelVersion("1.0.0")
            .inputFeatureNames(TestDataFactory.createFeatureNames())
            .outputNames(Arrays.asList("sepsis_probability"))
            .config(sepsisConfig)
            .build();
        sepsisModel.initialize();

        ModelConfig deteriorationConfig = ModelConfig.builder()
            .modelPath(MODELS_DIR + "/deterioration_risk_v1.0.0.onnx")
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.7)
            .build();

        deteriorationModel = ONNXModelContainer.builder()
            .modelId("deterioration_risk_v1")
            .modelName("Deterioration Risk Predictor")
            .modelType(ONNXModelContainer.ModelType.CLINICAL_DETERIORATION)
            .modelVersion("1.0.0")
            .inputFeatureNames(TestDataFactory.createFeatureNames())
            .outputNames(Arrays.asList("deterioration_probability"))
            .config(deteriorationConfig)
            .build();
        deteriorationModel.initialize();

        ModelConfig mortalityConfig = ModelConfig.builder()
            .modelPath(MODELS_DIR + "/mortality_risk_v1.0.0.onnx")
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.7)
            .build();

        mortalityModel = ONNXModelContainer.builder()
            .modelId("mortality_risk_v1")
            .modelName("Mortality Risk Predictor")
            .modelType(ONNXModelContainer.ModelType.MORTALITY_PREDICTION)
            .modelVersion("1.0.0")
            .inputFeatureNames(TestDataFactory.createFeatureNames())
            .outputNames(Arrays.asList("mortality_probability"))
            .config(mortalityConfig)
            .build();
        mortalityModel.initialize();

        ModelConfig readmissionConfig = ModelConfig.builder()
            .modelPath(MODELS_DIR + "/readmission_risk_v1.0.0.onnx")
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.7)
            .build();

        readmissionModel = ONNXModelContainer.builder()
            .modelId("readmission_risk_v1")
            .modelName("Readmission Risk Predictor")
            .modelType(ONNXModelContainer.ModelType.READMISSION_RISK)
            .modelVersion("1.0.0")
            .inputFeatureNames(TestDataFactory.createFeatureNames())
            .outputNames(Arrays.asList("readmission_probability"))
            .config(readmissionConfig)
            .build();
        readmissionModel.initialize();

        System.out.println("   ✅ All 4 models loaded successfully\n");

        // Create feature extractor
        extractor = new ClinicalFeatureExtractor();

        // Generate test patients
        System.out.println("👥 Generating test patients...");
        testPatients = generateTestPatients(10000);
        System.out.println("   ✅ Generated " + testPatients.size() + " test patients\n");

        // Warmup phase
        System.out.println("🔥 Warmup phase (" + WARMUP_ITERATIONS + " iterations)...");
        warmupModels();
        System.out.println("   ✅ Warmup complete\n");

        System.out.println("=".repeat(70));
        System.out.println("STARTING BENCHMARKS");
        System.out.println("=".repeat(70));
        System.out.println();
    }

    @AfterAll
    static void teardownBenchmarks() throws Exception {
        System.out.println("\n" + "=".repeat(70));
        System.out.println("BENCHMARK SUITE COMPLETE");
        System.out.println("=".repeat(70));

        // Close models
        if (sepsisModel != null) sepsisModel.close();
        if (deteriorationModel != null) deteriorationModel.close();
        if (mortalityModel != null) mortalityModel.close();
        if (readmissionModel != null) readmissionModel.close();
    }

    /**
     * Benchmark 1: Latency Profiling
     *
     * Run 10,000 predictions and measure latency distribution:
     * - p50 (median)
     * - p95
     * - p99
     *
     * Target: p99 < 15ms
     */
    @Test
    @Order(1)
    @DisplayName("Benchmark 1: Latency Profiling (10,000 predictions)")
    void benchmarkLatencyProfiling() throws Exception {
        System.out.println("\n📊 BENCHMARK 1: Latency Profiling");
        System.out.println("─".repeat(70));

        List<Long> latencies = new ArrayList<>(10000);

        // Run 10,000 predictions
        for (int i = 0; i < 10000; i++) {
            PatientContextSnapshot patient = testPatients.get(i % testPatients.size());
            ClinicalFeatureVector features = extractor.extract(patient, null, null);

            long startTime = System.nanoTime();
            float[] featureArray = features.toFloatArray();
            MLPrediction prediction = sepsisModel.predict(featureArray);
            long latencyNs = System.nanoTime() - startTime;

            latencies.add(latencyNs / 1_000_000); // Convert to ms

            // Progress indicator every 1000 iterations
            if ((i + 1) % 1000 == 0) {
                System.out.print(".");
                if ((i + 1) % 10000 == 0) {
                    System.out.println(" " + (i + 1));
                }
            }
        }

        // Calculate percentiles
        Collections.sort(latencies);
        long p50 = latencies.get((int) (latencies.size() * 0.50));
        long p95 = latencies.get((int) (latencies.size() * 0.95));
        long p99 = latencies.get((int) (latencies.size() * 0.99));
        long min = latencies.get(0);
        long max = latencies.get(latencies.size() - 1);
        double mean = latencies.stream().mapToLong(Long::longValue).average().orElse(0.0);

        // Results
        System.out.println();
        System.out.println("Results:");
        System.out.println("  Min:    " + min + " ms");
        System.out.println("  p50:    " + p50 + " ms");
        System.out.println("  p95:    " + p95 + " ms");
        System.out.println("  p99:    " + p99 + " ms (" + (p99 < 15 ? "✅ PASS" : "❌ FAIL") + ")");
        System.out.println("  Max:    " + max + " ms");
        System.out.println("  Mean:   " + String.format("%.2f", mean) + " ms");
        System.out.println();

        // Assertions
        assertThat(p99).isLessThan(15);
        assertThat(p95).isLessThan(10);
        assertThat(p50).isLessThan(5);
    }

    /**
     * Benchmark 2: Throughput Measurement
     *
     * Run continuous predictions for 60 seconds and measure:
     * - Total predictions processed
     * - Predictions per second
     *
     * Target: >100 predictions/second
     */
    @Test
    @Order(2)
    @DisplayName("Benchmark 2: Throughput Measurement (60 seconds)")
    void benchmarkThroughput() throws Exception {
        System.out.println("\n📊 BENCHMARK 2: Throughput Measurement");
        System.out.println("─".repeat(70));

        int totalPredictions = 0;
        long startTime = System.currentTimeMillis();
        long endTime = startTime + 60_000; // 60 seconds

        System.out.println("Running predictions for 60 seconds...");

        // Run predictions for 60 seconds
        while (System.currentTimeMillis() < endTime) {
            PatientContextSnapshot patient = testPatients.get(totalPredictions % testPatients.size());
            ClinicalFeatureVector features = extractor.extract(patient, null, null);
            float[] featureArray = features.toFloatArray();
            MLPrediction prediction = sepsisModel.predict(featureArray);
            totalPredictions++;

            // Progress indicator every 1000 predictions
            if (totalPredictions % 1000 == 0) {
                long elapsed = System.currentTimeMillis() - startTime;
                double currentThroughput = (totalPredictions * 1000.0) / elapsed;
                System.out.printf("  %d predictions in %.1fs (%.1f pred/sec)\n",
                    totalPredictions, elapsed / 1000.0, currentThroughput);
            }
        }

        long totalTime = System.currentTimeMillis() - startTime;
        double throughput = (totalPredictions * 1000.0) / totalTime;

        // Results
        System.out.println();
        System.out.println("Results:");
        System.out.println("  Total predictions: " + totalPredictions);
        System.out.println("  Total time:        " + (totalTime / 1000.0) + " seconds");
        System.out.println("  Throughput:        " + String.format("%.1f", throughput) + " pred/sec " +
            (throughput > 100 ? "✅ PASS" : "❌ FAIL"));
        System.out.println();

        // Assertions
        assertThat(throughput).isGreaterThan(100);
    }

    /**
     * Benchmark 3: Batch Optimization
     *
     * Test different batch sizes (8, 16, 32, 64) and measure:
     * - Total processing time
     * - Average time per prediction
     * - Optimal batch size
     *
     * Target: Batch size 32 < 50ms
     */
    @Test
    @Order(3)
    @DisplayName("Benchmark 3: Batch Optimization (8, 16, 32, 64)")
    void benchmarkBatchOptimization() throws Exception {
        System.out.println("\n📊 BENCHMARK 3: Batch Optimization");
        System.out.println("─".repeat(70));

        int[] batchSizes = {8, 16, 32, 64};
        Map<Integer, Double> batchResults = new LinkedHashMap<>();

        for (int batchSize : batchSizes) {
            System.out.println("\nTesting batch size: " + batchSize);

            // Prepare batch
            List<PatientContextSnapshot> batch = testPatients.subList(0, batchSize);
            List<ClinicalFeatureVector> features = new ArrayList<>();

            for (PatientContextSnapshot patient : batch) {
                features.add(extractor.extract(patient, null, null));
            }

            // Run 100 iterations to get stable measurement
            List<Long> batchLatencies = new ArrayList<>();
            for (int i = 0; i < 100; i++) {
                long startTime = System.nanoTime();

                // Process batch sequentially (simulate batch processing)
                for (int j = 0; j < features.size(); j++) {
                    float[] featureArray = features.get(j).toFloatArray();
                    MLPrediction prediction = sepsisModel.predict(featureArray);
                }

                long latencyNs = System.nanoTime() - startTime;
                batchLatencies.add(latencyNs / 1_000_000); // Convert to ms
            }

            // Calculate average
            double avgLatency = batchLatencies.stream()
                .mapToLong(Long::longValue)
                .average()
                .orElse(0.0);
            double avgPerPrediction = avgLatency / batchSize;

            batchResults.put(batchSize, avgLatency);

            System.out.println("  Total time:    " + String.format("%.2f", avgLatency) + " ms");
            System.out.println("  Per prediction: " + String.format("%.2f", avgPerPrediction) + " ms");
            System.out.println("  Status:        " +
                (batchSize == 32 && avgLatency < 50 ? "✅ PASS" :
                 batchSize == 32 && avgLatency >= 50 ? "❌ FAIL" : ""));
        }

        // Find optimal batch size (lowest per-prediction latency)
        System.out.println();
        System.out.println("Batch Size Comparison:");
        System.out.println("  Size | Total Time | Per Prediction");
        System.out.println("  " + "─".repeat(40));

        for (Map.Entry<Integer, Double> entry : batchResults.entrySet()) {
            int size = entry.getKey();
            double total = entry.getValue();
            double perPred = total / size;
            System.out.printf("  %-4d | %7.2f ms | %7.2f ms\n", size, total, perPred);
        }
        System.out.println();

        // Assertions
        assertThat(batchResults.get(32)).isLessThan(50.0);
    }

    /**
     * Benchmark 4: Memory Usage Profiling
     *
     * Measure heap memory usage with 4 models loaded:
     * - Before model loading
     * - After model loading
     * - During inference
     * - Peak memory usage
     *
     * Target: <500MB
     */
    @Test
    @Order(4)
    @DisplayName("Benchmark 4: Memory Usage Profiling")
    void benchmarkMemoryUsage() throws Exception {
        System.out.println("\n📊 BENCHMARK 4: Memory Usage Profiling");
        System.out.println("─".repeat(70));

        Runtime runtime = Runtime.getRuntime();

        // Force garbage collection
        System.gc();
        Thread.sleep(1000);

        long memoryBefore = runtime.totalMemory() - runtime.freeMemory();
        System.out.println("Memory before inference: " + formatMemory(memoryBefore));

        // Run 1000 predictions
        System.out.println("\nRunning 1000 predictions...");
        for (int i = 0; i < 1000; i++) {
            PatientContextSnapshot patient = testPatients.get(i % testPatients.size());
            ClinicalFeatureVector features = extractor.extract(patient, null, null);
            float[] featureArray = features.toFloatArray();
            MLPrediction prediction = sepsisModel.predict(featureArray);

            if ((i + 1) % 100 == 0) {
                System.out.print(".");
            }
        }
        System.out.println(" Done");

        long memoryAfter = runtime.totalMemory() - runtime.freeMemory();
        long memoryUsed = memoryAfter - memoryBefore;
        long peakMemory = runtime.maxMemory();

        // Results
        System.out.println();
        System.out.println("Results:");
        System.out.println("  Memory after inference:  " + formatMemory(memoryAfter));
        System.out.println("  Memory used:             " + formatMemory(memoryUsed));
        System.out.println("  Peak memory available:   " + formatMemory(peakMemory));
        System.out.println("  Status:                  " +
            (memoryAfter < 500_000_000 ? "✅ PASS (<500MB)" : "⚠️  Warning (>500MB)"));
        System.out.println();

        // Assertions (warning, not hard failure)
        assertThat(memoryAfter).isLessThan(1_000_000_000L); // 1GB hard limit
    }

    /**
     * Benchmark 5: Parallel Speedup Analysis
     *
     * Compare sequential vs parallel execution of 4 models:
     * - Sequential: Run 4 models one after another
     * - Parallel: Run 4 models concurrently
     * - Calculate speedup factor
     *
     * Target: >2x speedup
     */
    @Test
    @Order(5)
    @DisplayName("Benchmark 5: Parallel Speedup Analysis")
    void benchmarkParallelSpeedup() throws Exception {
        System.out.println("\n📊 BENCHMARK 5: Parallel Speedup Analysis");
        System.out.println("─".repeat(70));

        int iterations = 100;
        PatientContextSnapshot patient = testPatients.get(0);
        ClinicalFeatureVector features = extractor.extract(patient, null, null);
        float[] featureArray = features.toFloatArray();

        // Sequential execution
        System.out.println("\nSequential execution (" + iterations + " iterations)...");
        List<Long> sequentialLatencies = new ArrayList<>();

        for (int i = 0; i < iterations; i++) {
            long startTime = System.nanoTime();

            MLPrediction sepsis = sepsisModel.predict(featureArray);
            MLPrediction deterioration = deteriorationModel.predict(featureArray);
            MLPrediction mortality = mortalityModel.predict(featureArray);
            MLPrediction readmission = readmissionModel.predict(featureArray);

            long latencyNs = System.nanoTime() - startTime;
            sequentialLatencies.add(latencyNs / 1_000_000);

            if ((i + 1) % 10 == 0) {
                System.out.print(".");
            }
        }
        System.out.println(" Done");

        double avgSequential = sequentialLatencies.stream()
            .mapToLong(Long::longValue)
            .average()
            .orElse(0.0);

        // Parallel execution
        System.out.println("\nParallel execution (" + iterations + " iterations)...");
        List<Long> parallelLatencies = new ArrayList<>();
        ExecutorService executor = Executors.newFixedThreadPool(4);

        for (int i = 0; i < iterations; i++) {
            long startTime = System.nanoTime();

            // Submit all 4 models in parallel
            Future<MLPrediction> sepsisFuture = executor.submit(() ->
                sepsisModel.predict(featureArray));
            Future<MLPrediction> deteriorationFuture = executor.submit(() ->
                deteriorationModel.predict(featureArray));
            Future<MLPrediction> mortalityFuture = executor.submit(() ->
                mortalityModel.predict(featureArray));
            Future<MLPrediction> readmissionFuture = executor.submit(() ->
                readmissionModel.predict(featureArray));

            // Wait for all to complete
            sepsisFuture.get();
            deteriorationFuture.get();
            mortalityFuture.get();
            readmissionFuture.get();

            long latencyNs = System.nanoTime() - startTime;
            parallelLatencies.add(latencyNs / 1_000_000);

            if ((i + 1) % 10 == 0) {
                System.out.print(".");
            }
        }
        System.out.println(" Done");

        executor.shutdown();
        executor.awaitTermination(10, TimeUnit.SECONDS);

        double avgParallel = parallelLatencies.stream()
            .mapToLong(Long::longValue)
            .average()
            .orElse(0.0);

        double speedup = avgSequential / avgParallel;

        // Results
        System.out.println();
        System.out.println("Results:");
        System.out.println("  Sequential (4 models): " + String.format("%.2f", avgSequential) + " ms");
        System.out.println("  Parallel (4 models):   " + String.format("%.2f", avgParallel) + " ms");
        System.out.println("  Speedup factor:        " + String.format("%.2fx", speedup) + " " +
            (speedup > 2.0 ? "✅ PASS" : "⚠️  Warning (<2x)"));
        System.out.println("  Time saved:            " + String.format("%.2f", avgSequential - avgParallel) + " ms");
        System.out.println();

        // Assertions
        assertThat(speedup).isGreaterThan(1.5); // At least 1.5x speedup
    }

    // ========== Helper Methods ==========

    /**
     * Generate test patients with realistic clinical data
     */
    private static List<PatientContextSnapshot> generateTestPatients(int count) {
        List<PatientContextSnapshot> patients = new ArrayList<>();
        Random random = new Random(42);

        for (int i = 0; i < count; i++) {
            PatientContextSnapshot patient = new PatientContextSnapshot();
            patient.setPatientId("PAT-BENCH-" + String.format("%05d", i));
            patient.setTimestamp(java.time.Instant.ofEpochMilli(System.currentTimeMillis()));

            // Demographics
            patient.setAge(30 + random.nextInt(70));
            patient.setGender(random.nextBoolean() ? "M" : "F");

            // Vitals
            patient.setHeartRate((double)(60 + random.nextInt(40)));
            patient.setSystolicBP((double)(100 + random.nextInt(50)));
            patient.setDiastolicBP((double)(60 + random.nextInt(30)));
            patient.setRespiratoryRate((double)(12 + random.nextInt(12)));
            patient.setTemperature(36.0 + random.nextDouble() * 2.5);
            patient.setOxygenSaturation((double)(92 + random.nextInt(8)));

            // Labs
            patient.setWhiteBloodCells(4.0 + random.nextDouble() * 11.0);
            patient.setHemoglobin(10.0 + random.nextDouble() * 7.0);
            patient.setPlatelets((double)(100 + random.nextInt(300)));
            patient.setSodium((double)(130 + random.nextInt(20)));
            patient.setPotassium(3.0 + random.nextDouble() * 2.5);
            patient.setCreatinine(0.5 + random.nextDouble() * 2.5);
            patient.setLactate(0.5 + random.nextDouble() * 3.5);

            patients.add(patient);
        }

        return patients;
    }

    /**
     * Warmup models to ensure JIT compilation and cache warming
     */
    private static void warmupModels() throws Exception {
        PatientContextSnapshot patient = testPatients.get(0);
        ClinicalFeatureVector features = extractor.extract(patient, null, null);
        float[] featureArray = features.toFloatArray();

        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            sepsisModel.predict(featureArray);
            deteriorationModel.predict(featureArray);
            mortalityModel.predict(featureArray);
            readmissionModel.predict(featureArray);

            if ((i + 1) % 100 == 0) {
                System.out.print(".");
            }
        }
        System.out.println();
    }

    /**
     * Format memory in human-readable form
     */
    private static String formatMemory(long bytes) {
        if (bytes < 1024) {
            return bytes + " B";
        } else if (bytes < 1024 * 1024) {
            return String.format("%.2f KB", bytes / 1024.0);
        } else {
            return String.format("%.2f MB", bytes / (1024.0 * 1024.0));
        }
    }
}
