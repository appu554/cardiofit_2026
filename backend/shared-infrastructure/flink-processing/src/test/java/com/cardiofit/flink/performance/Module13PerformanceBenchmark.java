package com.cardiofit.flink.performance;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.operators.Module13CKMRiskComputer;
import com.cardiofit.flink.operators.Module13DataCompletenessMonitor;
import com.cardiofit.flink.operators.Module13StateChangeDetector;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Module 13 Performance Benchmark — Week 7
 *
 * Validates that the pure-function pipeline (CKMRiskComputer + StateChangeDetector
 * + DataCompletenessMonitor) meets latency and throughput targets for production:
 *
 *   - Single-event P95 latency: <5ms
 *   - CKM velocity computation: <2ms
 *   - State change detection: <1ms
 *   - Throughput: >5,000 events/sec per thread
 *   - Memory stability: no growth over 10K iterations
 *
 * Benchmarks use real Module13TestBuilder state objects to simulate production
 * workloads with realistic metric distributions.
 */
class Module13PerformanceBenchmark {

    private static final int WARMUP_ITERATIONS = 1_000;
    private static final int BENCHMARK_ITERATIONS = 10_000;

    private static ClinicalStateSummary benchmarkState;

    @BeforeAll
    static void warmup() {
        // Pre-build state to avoid allocation noise in benchmarks
        benchmarkState = Module13TestBuilder.stateWithSnapshotPair("bench-p1");
        benchmarkState.current().fbg = 145.0;
        benchmarkState.current().hba1c = 7.8;
        benchmarkState.current().egfr = 58.0;
        benchmarkState.current().uacr = 60.0;
        benchmarkState.previous().uacr = 30.0;
        benchmarkState.current().arv = 14.0;
        benchmarkState.current().meanSBP = 148.0;
        benchmarkState.current().ldl = 125.0;
        benchmarkState.previous().ldl = 110.0;
        benchmarkState.current().morningSurgeMagnitude = 28.0;
        benchmarkState.previous().morningSurgeMagnitude = 20.0;
        benchmarkState.current().engagementScore = 0.65;
        benchmarkState.setLastComputedVelocity(CKMRiskVelocity.builder()
                .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                .compositeScore(0.1).dataCompleteness(1.0)
                .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

        // JIT warmup
        for (int i = 0; i < WARMUP_ITERATIONS; i++) {
            CKMRiskVelocity v = Module13CKMRiskComputer.compute(benchmarkState);
            Module13StateChangeDetector.detect(benchmarkState, v,
                    Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);
            Module13DataCompletenessMonitor.evaluate(benchmarkState,
                    Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);
        }
    }

    /**
     * Benchmark 1: CKM Risk Velocity computation latency.
     * Target: P95 < 2ms, Mean < 0.5ms.
     */
    @Test
    void ckmVelocityComputation_latencyWithinTarget() {
        long[] latencies = new long[BENCHMARK_ITERATIONS];

        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            long start = System.nanoTime();
            Module13CKMRiskComputer.compute(benchmarkState);
            latencies[i] = System.nanoTime() - start;
        }

        Arrays.sort(latencies);
        double p50Ms = latencies[(int) (BENCHMARK_ITERATIONS * 0.50)] / 1_000_000.0;
        double p95Ms = latencies[(int) (BENCHMARK_ITERATIONS * 0.95)] / 1_000_000.0;
        double p99Ms = latencies[(int) (BENCHMARK_ITERATIONS * 0.99)] / 1_000_000.0;
        double meanMs = Arrays.stream(latencies).average().orElse(0) / 1_000_000.0;

        System.out.printf("CKM Velocity: p50=%.3fms, p95=%.3fms, p99=%.3fms, mean=%.3fms%n",
                p50Ms, p95Ms, p99Ms, meanMs);

        assertTrue(p95Ms < 2.0, "CKM velocity P95 should be <2ms, got " + p95Ms);
        assertTrue(meanMs < 0.5, "CKM velocity mean should be <0.5ms, got " + meanMs);
    }

    /**
     * Benchmark 2: State change detection latency.
     * Target: P95 < 1ms, Mean < 0.2ms.
     */
    @Test
    void stateChangeDetection_latencyWithinTarget() {
        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(benchmarkState);
        long[] latencies = new long[BENCHMARK_ITERATIONS];

        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            long start = System.nanoTime();
            Module13StateChangeDetector.detect(benchmarkState, velocity,
                    Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS + i);
            latencies[i] = System.nanoTime() - start;
        }

        Arrays.sort(latencies);
        double p95Ms = latencies[(int) (BENCHMARK_ITERATIONS * 0.95)] / 1_000_000.0;
        double meanMs = Arrays.stream(latencies).average().orElse(0) / 1_000_000.0;

        System.out.printf("State Change Detection: p95=%.3fms, mean=%.3fms%n", p95Ms, meanMs);

        assertTrue(p95Ms < 1.0, "State change detection P95 should be <1ms, got " + p95Ms);
    }

    /**
     * Benchmark 3: Full pipeline (velocity + detection + completeness).
     * Target: P95 < 5ms combined.
     */
    @Test
    void fullPipeline_latencyWithinTarget() {
        long[] latencies = new long[BENCHMARK_ITERATIONS];

        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            long start = System.nanoTime();

            Module13DataCompletenessMonitor.Result completeness =
                    Module13DataCompletenessMonitor.evaluate(benchmarkState,
                            Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);
            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(benchmarkState);
            Module13StateChangeDetector.detect(benchmarkState, velocity,
                    Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS + i);

            latencies[i] = System.nanoTime() - start;
        }

        Arrays.sort(latencies);
        double p50Ms = latencies[(int) (BENCHMARK_ITERATIONS * 0.50)] / 1_000_000.0;
        double p95Ms = latencies[(int) (BENCHMARK_ITERATIONS * 0.95)] / 1_000_000.0;
        double p99Ms = latencies[(int) (BENCHMARK_ITERATIONS * 0.99)] / 1_000_000.0;
        double meanMs = Arrays.stream(latencies).average().orElse(0) / 1_000_000.0;

        System.out.printf("Full Pipeline: p50=%.3fms, p95=%.3fms, p99=%.3fms, mean=%.3fms%n",
                p50Ms, p95Ms, p99Ms, meanMs);

        assertTrue(p95Ms < 5.0, "Full pipeline P95 should be <5ms, got " + p95Ms);
    }

    /**
     * Benchmark 4: Throughput measurement over 5 seconds.
     * Target: >5,000 events/sec (single thread).
     */
    @Test
    void throughput_exceedsMinimum() {
        long durationMs = 5_000;
        long start = System.currentTimeMillis();
        long count = 0;

        while (System.currentTimeMillis() - start < durationMs) {
            CKMRiskVelocity v = Module13CKMRiskComputer.compute(benchmarkState);
            Module13StateChangeDetector.detect(benchmarkState, v,
                    Module13TestBuilder.BASE_TIME + count);
            count++;
        }

        double eventsPerSec = count / (durationMs / 1000.0);
        System.out.printf("Throughput: %.0f events/sec over %dms (%d total)%n",
                eventsPerSec, durationMs, count);

        assertTrue(eventsPerSec > 5_000,
                "Throughput should exceed 5,000 events/sec, got " + eventsPerSec);
    }

    /**
     * Benchmark 5: Memory stability — no significant heap growth over 10K iterations.
     * Target: <50MB growth (accounts for GC variance).
     */
    @Test
    void memoryStability_noSignificantGrowth() {
        System.gc();
        long heapBefore = Runtime.getRuntime().totalMemory() - Runtime.getRuntime().freeMemory();

        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            // Create fresh state each iteration to stress allocation
            ClinicalStateSummary state = Module13TestBuilder.stateWithSnapshotPair("mem-" + i);
            state.current().fbg = 130.0 + (i % 50);
            state.current().egfr = 65.0 - (i % 20);
            state.current().arv = 10.0 + (i % 10);
            state.setLastComputedVelocity(CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.STABLE)
                    .compositeScore(0.0).dataCompleteness(1.0)
                    .computationTimestamp(Module13TestBuilder.BASE_TIME).build());

            CKMRiskVelocity v = Module13CKMRiskComputer.compute(state);
            Module13StateChangeDetector.detect(state, v,
                    Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);
        }

        System.gc();
        long heapAfter = Runtime.getRuntime().totalMemory() - Runtime.getRuntime().freeMemory();
        long growthMB = (heapAfter - heapBefore) / (1024 * 1024);

        System.out.printf("Memory: before=%dMB, after=%dMB, growth=%dMB%n",
                heapBefore / (1024 * 1024), heapAfter / (1024 * 1024), growthMB);

        assertTrue(growthMB < 50,
                "Memory growth should be <50MB over 10K iterations, got " + growthMB + "MB");
    }

    /**
     * Benchmark 6: Snapshot rotation cost.
     * Target: P95 < 0.1ms per rotation.
     */
    @Test
    void snapshotRotation_latencyWithinTarget() {
        long[] latencies = new long[BENCHMARK_ITERATIONS];

        for (int i = 0; i < BENCHMARK_ITERATIONS; i++) {
            ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("rot-" + i);
            long start = System.nanoTime();
            state.rotateSnapshots(Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);
            latencies[i] = System.nanoTime() - start;
        }

        Arrays.sort(latencies);
        double p95Ms = latencies[(int) (BENCHMARK_ITERATIONS * 0.95)] / 1_000_000.0;
        double meanMs = Arrays.stream(latencies).average().orElse(0) / 1_000_000.0;

        System.out.printf("Snapshot Rotation: p95=%.3fms, mean=%.3fms%n", p95Ms, meanMs);

        assertTrue(p95Ms < 0.1, "Snapshot rotation P95 should be <0.1ms, got " + p95Ms);
    }
}
