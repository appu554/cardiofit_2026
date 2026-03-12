package com.cardiofit.flink.metrics;

import org.apache.flink.metrics.Counter;
import org.apache.flink.metrics.Gauge;
import org.apache.flink.metrics.Histogram;
import org.apache.flink.metrics.MetricGroup;
import org.apache.flink.dropwizard.metrics.DropwizardHistogramWrapper;
import com.codahale.metrics.SlidingTimeWindowReservoir;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Production-grade metrics framework for async I/O operations.
 *
 * Tracks critical metrics for FHIR and Neo4j async lookups:
 * - Request counters (total, success, failure)
 * - Latency histograms (P50, P95, P99)
 * - In-flight requests (backpressure indicator)
 * - Circuit breaker state
 * - Cache hit/miss rates
 *
 * Usage:
 * <pre>
 * AsyncIOMetrics metrics = new AsyncIOMetrics(getRuntimeContext().getMetricGroup(), "fhir");
 *
 * metrics.recordRequestStart();
 * CompletableFuture<Data> future = executeAsyncRequest()
 *     .whenComplete((result, throwable) -> {
 *         if (throwable == null) {
 *             metrics.recordSuccess(startTime);
 *         } else {
 *             metrics.recordFailure(startTime);
 *         }
 *     });
 * </pre>
 */
public class AsyncIOMetrics {

    private static final Logger LOG = LoggerFactory.getLogger(AsyncIOMetrics.class);

    private final String componentName;

    // Request Counters
    private final Counter requestsTotal;
    private final Counter requestsSuccess;
    private final Counter requestsFailure;
    private final Counter requestsTimeout;

    // Latency Histogram (60-second sliding window)
    private final Histogram latencyHistogram;

    // In-Flight Requests (Backpressure Indicator)
    private final AtomicLong inFlightRequests;
    private final Gauge<Long> inFlightGauge;

    // Cache Metrics (optional)
    private final Counter cacheHits;
    private final Counter cacheMisses;

    // Circuit Breaker State (optional)
    private final AtomicLong circuitBreakerState; // 0=closed, 1=open, 2=half-open
    private final Gauge<Long> circuitBreakerGauge;

    /**
     * Create metrics for an async I/O component.
     *
     * @param metricGroup Flink metric group (from getRuntimeContext().getMetricGroup())
     * @param componentName Component identifier (e.g., "fhir", "neo4j")
     */
    public AsyncIOMetrics(MetricGroup metricGroup, String componentName) {
        this.componentName = componentName;

        // Create component-specific metric group
        MetricGroup componentGroup = metricGroup.addGroup("async_io").addGroup(componentName);

        // Request Counters
        this.requestsTotal = componentGroup.counter("requests_total");
        this.requestsSuccess = componentGroup.counter("requests_success");
        this.requestsFailure = componentGroup.counter("requests_failure");
        this.requestsTimeout = componentGroup.counter("requests_timeout");

        // Latency Histogram (60-second sliding window)
        com.codahale.metrics.Histogram dropwizardHistogram = new com.codahale.metrics.Histogram(
            new SlidingTimeWindowReservoir(60, TimeUnit.SECONDS)
        );
        this.latencyHistogram = componentGroup.histogram("latency_ms",
            new DropwizardHistogramWrapper(dropwizardHistogram));

        // In-Flight Requests Gauge
        this.inFlightRequests = new AtomicLong(0);
        this.inFlightGauge = componentGroup.gauge("requests_inflight", () -> inFlightRequests.get());

        // Cache Metrics
        this.cacheHits = componentGroup.counter("cache_hits");
        this.cacheMisses = componentGroup.counter("cache_misses");

        // Circuit Breaker State Gauge
        this.circuitBreakerState = new AtomicLong(0); // 0=closed (healthy)
        this.circuitBreakerGauge = componentGroup.gauge("circuit_breaker_state", () -> circuitBreakerState.get());

        LOG.info("Initialized AsyncIOMetrics for component: {}", componentName);
    }

    /**
     * Record start of async request (increment in-flight counter).
     *
     * @return Current timestamp in nanoseconds (for latency calculation)
     */
    public long recordRequestStart() {
        requestsTotal.inc();
        inFlightRequests.incrementAndGet();
        return System.nanoTime();
    }

    /**
     * Record successful async request completion.
     *
     * @param startTimeNanos Start timestamp from recordRequestStart()
     */
    public void recordSuccess(long startTimeNanos) {
        long latencyNanos = System.nanoTime() - startTimeNanos;
        long latencyMs = latencyNanos / 1_000_000; // Convert to milliseconds

        requestsSuccess.inc();
        inFlightRequests.decrementAndGet();
        latencyHistogram.update(latencyMs);

        if (latencyMs > 500) {
            LOG.warn("[{}] Slow request detected: {}ms", componentName, latencyMs);
        }
    }

    /**
     * Record failed async request.
     *
     * @param startTimeNanos Start timestamp from recordRequestStart()
     */
    public void recordFailure(long startTimeNanos) {
        long latencyNanos = System.nanoTime() - startTimeNanos;
        long latencyMs = latencyNanos / 1_000_000;

        requestsFailure.inc();
        inFlightRequests.decrementAndGet();
        latencyHistogram.update(latencyMs);
    }

    /**
     * Record timeout on async request.
     *
     * @param startTimeNanos Start timestamp from recordRequestStart()
     */
    public void recordTimeout(long startTimeNanos) {
        requestsTimeout.inc();
        inFlightRequests.decrementAndGet();
        // Don't record latency for timeouts (would skew histogram)
    }

    /**
     * Record cache hit.
     */
    public void recordCacheHit() {
        cacheHits.inc();
    }

    /**
     * Record cache miss (will trigger async lookup).
     */
    public void recordCacheMiss() {
        cacheMisses.inc();
    }

    /**
     * Update circuit breaker state.
     *
     * @param state 0=closed (healthy), 1=open (failing), 2=half-open (testing)
     */
    public void setCircuitBreakerState(int state) {
        circuitBreakerState.set(state);

        if (state == 1) {
            LOG.error("[{}] Circuit breaker OPENED - {} is failing", componentName, componentName);
        } else if (state == 2) {
            LOG.warn("[{}] Circuit breaker HALF-OPEN - testing recovery", componentName);
        } else {
            LOG.info("[{}] Circuit breaker CLOSED - {} is healthy", componentName, componentName);
        }
    }

    /**
     * Get current in-flight request count (for backpressure monitoring).
     *
     * @return Number of requests currently in-flight
     */
    public long getInFlightRequests() {
        return inFlightRequests.get();
    }

    /**
     * Get component name for logging.
     */
    public String getComponentName() {
        return componentName;
    }
}
