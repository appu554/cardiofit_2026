package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

import java.util.ArrayList;
import java.util.List;

/**
 * Pure-function test harness replicating Module 13's processElement pipeline
 * without requiring a Flink runtime or KeyedProcessFunction context.
 *
 * <p>Calls through the same chain as the real operator:
 * <ol>
 *   <li>{@code routeAndUpdateState} — route event to correct updateFrom* method</li>
 *   <li>{@code recordModuleSeen} — track source freshness</li>
 *   <li>{@code DataCompletenessMonitor.evaluate} — compute completeness</li>
 *   <li>{@code CKMRiskComputer.compute} — compute CKM velocity</li>
 *   <li>{@code StateChangeDetector.detect} — detect clinical state changes</li>
 *   <li>Update state velocity and timestamp</li>
 * </ol>
 */
public final class CrossModuleTestHarness {

    private CrossModuleTestHarness() {}

    /** Result of processing a single event through the Module 13 pipeline. */
    public static class ProcessResult {
        private final String sourceModule;
        private final CKMRiskVelocity velocity;
        private final List<ClinicalStateChangeEvent> changes;
        private final double completenessScore;

        public ProcessResult(String sourceModule, CKMRiskVelocity velocity,
                             List<ClinicalStateChangeEvent> changes, double completenessScore) {
            this.sourceModule = sourceModule;
            this.velocity = velocity;
            this.changes = changes;
            this.completenessScore = completenessScore;
        }

        public String getSourceModule() { return sourceModule; }
        public CKMRiskVelocity getVelocity() { return velocity; }
        public List<ClinicalStateChangeEvent> getChanges() { return changes; }
        public double getCompletenessScore() { return completenessScore; }
        public boolean hasChange(ClinicalStateChangeType type) {
            return changes.stream().anyMatch(c -> c.getChangeType() == type);
        }
    }

    /**
     * Process a single event through the full Module 13 pipeline.
     *
     * @param event the canonical event to process
     * @param state the mutable patient state (modified in place)
     * @param processingTime simulated processing time (ms)
     * @return result containing velocity, state changes, and completeness; null if unroutable
     */
    public static ProcessResult processEvent(CanonicalEvent event, ClinicalStateSummary state,
                                             long processingTime) {
        // 1. Route and update state
        String sourceModule = Module13_ClinicalStateSynchroniser.routeAndUpdateState(
                event, state, processingTime);
        if (sourceModule == null) return null;

        state.recordModuleSeen(sourceModule, event.getEventTime());

        // 2. Compute data completeness
        Module13DataCompletenessMonitor.Result completeness =
                Module13DataCompletenessMonitor.evaluate(state, processingTime);
        state.setDataCompletenessScore(completeness.getCompositeScore());

        // 3. Compute CKM risk velocity
        CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);

        // 4. Detect state changes
        List<ClinicalStateChangeEvent> changes =
                Module13StateChangeDetector.detect(state, velocity, processingTime);

        // 5. Update dedup timestamps
        for (ClinicalStateChangeEvent change : changes) {
            state.getLastEmittedChangeTimestamps().put(
                    change.getChangeType(), change.getProcessingTimestamp());
        }

        // 6. Persist velocity in state
        state.setLastComputedVelocity(velocity);
        state.setLastUpdated(processingTime);

        return new ProcessResult(sourceModule, velocity, changes, completeness.getCompositeScore());
    }

    /**
     * Process a sequence of events, accumulating all state changes.
     *
     * @param events ordered list of events to process
     * @param state  mutable patient state (modified across calls)
     * @return list of results (one per successfully routed event; nulls filtered out)
     */
    public static List<ProcessResult> processSequence(List<CanonicalEvent> events,
                                                       ClinicalStateSummary state) {
        List<ProcessResult> results = new ArrayList<>();
        for (CanonicalEvent event : events) {
            long processingTime = event.getEventTime(); // use event time as processing time
            ProcessResult result = processEvent(event, state, processingTime);
            if (result != null) results.add(result);
        }
        return results;
    }
}
