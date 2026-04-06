package com.cardiofit.flink.metrics;

import org.apache.flink.dropwizard.metrics.DropwizardHistogramWrapper;
import org.apache.flink.metrics.Counter;
import org.apache.flink.metrics.Gauge;
import org.apache.flink.metrics.MetricGroup;

import com.codahale.metrics.SlidingTimeWindowReservoir;

import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Module 13 Clinical State Synchroniser — Observability Metrics
 *
 * Tracks operational health, clinical quality, and performance for the
 * 7-source fan-in state synchroniser. Metrics are exposed via Flink's
 * Prometheus reporter under the {@code module13} metric group.
 *
 * Metric categories:
 *   1. Event processing: throughput, latency, source-module routing
 *   2. CKM velocity: classification distribution, domain health
 *   3. State changes: emission rates, dedup suppression, priority breakdown
 *   4. Data quality: completeness scores, absence alerts, idle patients
 *   5. A1 personalised targets: enrichment rate, fallback rate
 *   6. Write coalescing: buffer depth, flush rate, overflow evictions
 */
public class Module13Metrics {

    // --- Event processing ---
    private final Counter eventsProcessedTotal;
    private final Counter eventsUnroutableTotal;
    private final Counter[] moduleEventCounters;   // indexed by module ordinal
    private final DropwizardHistogramWrapper processElementLatencyMs;

    // --- CKM velocity ---
    private final Counter velocityImproving;
    private final Counter velocityStable;
    private final Counter velocityDeteriorating;
    private final Counter velocityUnknown;
    private final Counter crossDomainAmplificationTotal;
    private final DropwizardHistogramWrapper compositeScoreHistogram;

    // --- State changes ---
    private final Counter stateChangesEmittedTotal;
    private final Counter stateChangesDedupSuppressed;
    private final Counter stateChangesCriticalTotal;
    private final Counter stateChangesHighTotal;
    private final Counter stateChangesMediumTotal;
    private final Counter stateChangesInfoTotal;

    // --- Data quality ---
    private final DropwizardHistogramWrapper dataCompletenessHistogram;
    private final Counter dataAbsenceWarnings;
    private final Counter dataAbsenceCritical;
    private final AtomicLong idlePatientsGaugeValue = new AtomicLong(0);

    // --- A1 personalised targets ---
    private final Counter personalizedTargetsEnriched;
    private final Counter personalizedTargetsFallback;

    // --- Write coalescing ---
    private final Counter coalescingFlushTotal;
    private final Counter coalescingEvictionTotal;
    private final AtomicLong coalescingBufferDepthValue = new AtomicLong(0);
    private final Counter kb20UpdatesEmittedTotal;

    // --- Snapshot rotation ---
    private final Counter snapshotRotationsTotal;

    // Source module names for event routing counters
    private static final String[] MODULE_NAMES = {
            "module7", "module8", "module9", "module10b",
            "module11b", "module12", "module12b", "enriched"
    };

    public Module13Metrics(MetricGroup parentGroup) {
        MetricGroup g = parentGroup.addGroup("module13");

        // Event processing
        MetricGroup events = g.addGroup("events");
        this.eventsProcessedTotal = events.counter("processed_total");
        this.eventsUnroutableTotal = events.counter("unroutable_total");
        this.processElementLatencyMs = events.histogram("process_latency_ms",
                new DropwizardHistogramWrapper(
                        new com.codahale.metrics.Histogram(
                                new SlidingTimeWindowReservoir(60, TimeUnit.SECONDS))));

        MetricGroup routing = events.addGroup("routing");
        this.moduleEventCounters = new Counter[MODULE_NAMES.length];
        for (int i = 0; i < MODULE_NAMES.length; i++) {
            moduleEventCounters[i] = routing.counter(MODULE_NAMES[i] + "_total");
        }

        // CKM velocity
        MetricGroup ckm = g.addGroup("ckm_velocity");
        this.velocityImproving = ckm.counter("improving_total");
        this.velocityStable = ckm.counter("stable_total");
        this.velocityDeteriorating = ckm.counter("deteriorating_total");
        this.velocityUnknown = ckm.counter("unknown_total");
        this.crossDomainAmplificationTotal = ckm.counter("cross_domain_amplification_total");
        this.compositeScoreHistogram = ckm.histogram("composite_score",
                new DropwizardHistogramWrapper(
                        new com.codahale.metrics.Histogram(
                                new SlidingTimeWindowReservoir(60, TimeUnit.SECONDS))));

        // State changes
        MetricGroup changes = g.addGroup("state_changes");
        this.stateChangesEmittedTotal = changes.counter("emitted_total");
        this.stateChangesDedupSuppressed = changes.counter("dedup_suppressed_total");
        this.stateChangesCriticalTotal = changes.counter("critical_total");
        this.stateChangesHighTotal = changes.counter("high_total");
        this.stateChangesMediumTotal = changes.counter("medium_total");
        this.stateChangesInfoTotal = changes.counter("info_total");

        // Data quality
        MetricGroup quality = g.addGroup("data_quality");
        this.dataCompletenessHistogram = quality.histogram("completeness_score",
                new DropwizardHistogramWrapper(
                        new com.codahale.metrics.Histogram(
                                new SlidingTimeWindowReservoir(60, TimeUnit.SECONDS))));
        this.dataAbsenceWarnings = quality.counter("absence_warning_total");
        this.dataAbsenceCritical = quality.counter("absence_critical_total");
        quality.gauge("idle_patients", (Gauge<Long>) idlePatientsGaugeValue::get);

        // A1 personalised targets
        MetricGroup a1 = g.addGroup("personalized_targets");
        this.personalizedTargetsEnriched = a1.counter("enriched_total");
        this.personalizedTargetsFallback = a1.counter("fallback_total");

        // Write coalescing
        MetricGroup coalesce = g.addGroup("coalescing");
        this.coalescingFlushTotal = coalesce.counter("flush_total");
        this.coalescingEvictionTotal = coalesce.counter("eviction_total");
        coalesce.gauge("buffer_depth", (Gauge<Long>) coalescingBufferDepthValue::get);
        this.kb20UpdatesEmittedTotal = coalesce.counter("kb20_updates_emitted_total");

        // Snapshot
        MetricGroup snapshot = g.addGroup("snapshot");
        this.snapshotRotationsTotal = snapshot.counter("rotations_total");
    }

    // --- Recording methods ---

    public void recordEventProcessed(String sourceModule, long latencyMs) {
        eventsProcessedTotal.inc();
        processElementLatencyMs.update(latencyMs);
        for (int i = 0; i < MODULE_NAMES.length; i++) {
            if (MODULE_NAMES[i].equals(sourceModule)) {
                moduleEventCounters[i].inc();
                return;
            }
        }
    }

    public void recordUnroutableEvent() {
        eventsUnroutableTotal.inc();
    }

    public void recordVelocityClassification(String classification, double compositeScore,
                                              boolean crossDomainAmplification) {
        // Scale to integer for histogram (multiply by 1000 for 3 decimal places)
        compositeScoreHistogram.update((long) (compositeScore * 1000));
        switch (classification) {
            case "IMPROVING": velocityImproving.inc(); break;
            case "STABLE": velocityStable.inc(); break;
            case "DETERIORATING": velocityDeteriorating.inc(); break;
            default: velocityUnknown.inc(); break;
        }
        if (crossDomainAmplification) {
            crossDomainAmplificationTotal.inc();
        }
    }

    public void recordStateChangeEmitted(String priority) {
        stateChangesEmittedTotal.inc();
        switch (priority) {
            case "CRITICAL": stateChangesCriticalTotal.inc(); break;
            case "HIGH": stateChangesHighTotal.inc(); break;
            case "MEDIUM": stateChangesMediumTotal.inc(); break;
            case "INFO": stateChangesInfoTotal.inc(); break;
            default: break;
        }
    }

    public void recordDedupSuppressed() {
        stateChangesDedupSuppressed.inc();
    }

    public void recordDataCompleteness(double score) {
        dataCompletenessHistogram.update((long) (score * 100));
    }

    public void recordDataAbsenceWarning() { dataAbsenceWarnings.inc(); }
    public void recordDataAbsenceCritical() { dataAbsenceCritical.inc(); }
    public void setIdlePatientCount(long count) { idlePatientsGaugeValue.set(count); }

    public void recordPersonalizedTargetsEnriched() { personalizedTargetsEnriched.inc(); }
    public void recordPersonalizedTargetsFallback() { personalizedTargetsFallback.inc(); }

    public void recordCoalescingFlush(int bufferSize) {
        coalescingFlushTotal.inc();
        kb20UpdatesEmittedTotal.inc(bufferSize);
    }

    public void recordCoalescingEviction() { coalescingEvictionTotal.inc(); }
    public void setCoalescingBufferDepth(long depth) { coalescingBufferDepthValue.set(depth); }

    public void recordSnapshotRotation() { snapshotRotationsTotal.inc(); }
}
