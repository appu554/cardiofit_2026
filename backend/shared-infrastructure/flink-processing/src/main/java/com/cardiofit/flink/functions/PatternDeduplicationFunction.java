package com.cardiofit.flink.functions;

import com.cardiofit.flink.models.PatternEvent;
import org.apache.flink.api.common.state.MapState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;

import java.util.*;

/**
 * Pattern Event Deduplication Function
 *
 * Prevents alert storms when multiple layers fire for the same patient.
 * Merges pattern events from different sources and boosts confidence.
 *
 * Deduplication Logic:
 * - Groups similar patterns within 5-minute window
 * - Merges evidence from multiple sources
 * - Increases confidence when layers agree
 * - Tracks which sources confirmed the pattern
 *
 * Implements Gap 1 from Gap Implementation Guide
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0
 */
public class PatternDeduplicationFunction
    extends KeyedProcessFunction<String, PatternEvent, PatternEvent> {

    // State to track last emitted pattern per patient
    private transient ValueState<PatternEvent> lastPatternState;

    // State to track recent patterns by type (for deduplication window)
    private transient MapState<String, Long> recentPatternsState;

    // Deduplication window: 5 minutes
    private static final long DEDUP_WINDOW_MS = 5 * 60 * 1000;

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        // Initialize last pattern state
        ValueStateDescriptor<PatternEvent> lastPatternDescriptor =
            new ValueStateDescriptor<>("last-pattern", PatternEvent.class);
        lastPatternState = getRuntimeContext().getState(lastPatternDescriptor);

        // Initialize recent patterns tracking
        MapStateDescriptor<String, Long> recentPatternsDescriptor =
            new MapStateDescriptor<>("recent-patterns", String.class, Long.class);
        recentPatternsState = getRuntimeContext().getMapState(recentPatternsDescriptor);
    }

    @Override
    public void processElement(
        PatternEvent pattern,
        Context ctx,
        Collector<PatternEvent> out) throws Exception {

        long now = System.currentTimeMillis();
        String patternKey = getPatternKey(pattern);

        // Check if similar pattern was recently fired
        Long lastFiredTime = recentPatternsState.get(patternKey);

        if (lastFiredTime != null && (now - lastFiredTime) < DEDUP_WINDOW_MS) {
            // Duplicate pattern within window - MERGE

            PatternEvent lastPattern = lastPatternState.value();

            if (lastPattern != null && shouldMerge(lastPattern, pattern)) {
                PatternEvent mergedPattern = mergePatterns(lastPattern, pattern);
                out.collect(mergedPattern);
                lastPatternState.update(mergedPattern);
                recentPatternsState.put(patternKey, now);
            } else {
                // Different enough to emit separately
                out.collect(pattern);
                lastPatternState.update(pattern);
                recentPatternsState.put(patternKey, now);
            }
        } else {
            // New pattern - emit immediately
            out.collect(pattern);
            lastPatternState.update(pattern);
            recentPatternsState.put(patternKey, now);
        }

        // Schedule cleanup timer
        ctx.timerService().registerProcessingTimeTimer(now + DEDUP_WINDOW_MS);
    }

    /**
     * Generate deduplication key from pattern
     * Key format: "{patternType}:{severity}"
     */
    private String getPatternKey(PatternEvent pattern) {
        return pattern.getPatternType() + ":" + pattern.getSeverity();
    }

    /**
     * Determine if two patterns should be merged
     * Merge if same type and similar severity
     */
    private boolean shouldMerge(PatternEvent existing, PatternEvent newPattern) {
        return existing.getPatternType().equals(newPattern.getPatternType()) &&
               existing.getSeverity().equals(newPattern.getSeverity());
    }

    /**
     * Merge two pattern events from different sources
     * Combines evidence and increases confidence
     */
    private PatternEvent mergePatterns(PatternEvent existing, PatternEvent newPattern) {

        // Build merged pattern
        PatternEvent merged = new PatternEvent();

        // Keep original ID and patient info
        merged.setId(existing.getId());
        merged.setPatientId(existing.getPatientId());
        merged.setEncounterId(existing.getEncounterId());
        merged.setPatternType(existing.getPatternType());
        merged.setCorrelationId(existing.getCorrelationId());

        // Use highest severity
        merged.setSeverity(getHighestSeverity(existing.getSeverity(), newPattern.getSeverity()));

        // Combine confidence (weighted average: existing 60%, new 40%)
        double combinedConfidence = Math.min(1.0,
            existing.getConfidence() * 0.6 + newPattern.getConfidence() * 0.4);
        merged.setConfidence(combinedConfidence);

        // Use earliest detection time
        merged.setDetectionTime(Math.min(
            existing.getDetectionTime(),
            newPattern.getDetectionTime()
        ));

        // Use earliest pattern start time
        merged.setPatternStartTime(Math.min(
            existing.getPatternStartTime() != null ? existing.getPatternStartTime() : Long.MAX_VALUE,
            newPattern.getPatternStartTime() != null ? newPattern.getPatternStartTime() : Long.MAX_VALUE
        ));

        // Use latest pattern end time
        merged.setPatternEndTime(Math.max(
            existing.getPatternEndTime() != null ? existing.getPatternEndTime() : Long.MIN_VALUE,
            newPattern.getPatternEndTime() != null ? newPattern.getPatternEndTime() : Long.MIN_VALUE
        ));

        // Merge involved events
        Set<String> allInvolvedEvents = new HashSet<>();
        if (existing.getInvolvedEvents() != null) {
            allInvolvedEvents.addAll(existing.getInvolvedEvents());
        }
        if (newPattern.getInvolvedEvents() != null) {
            allInvolvedEvents.addAll(newPattern.getInvolvedEvents());
        }
        merged.setInvolvedEvents(new ArrayList<>(allInvolvedEvents));

        // Merge recommended actions (deduplicate)
        Set<String> allActions = new LinkedHashSet<>();
        if (existing.getRecommendedActions() != null) {
            allActions.addAll(existing.getRecommendedActions());
        }
        if (newPattern.getRecommendedActions() != null) {
            allActions.addAll(newPattern.getRecommendedActions());
        }
        merged.setRecommendedActions(new ArrayList<>(allActions));

        // Use existing clinical context (most complete)
        merged.setClinicalContext(existing.getClinicalContext());

        // Merge pattern details
        Map<String, Object> mergedDetails = new HashMap<>();
        if (existing.getPatternDetails() != null) {
            mergedDetails.putAll(existing.getPatternDetails());
        }
        if (newPattern.getPatternDetails() != null) {
            mergedDetails.putAll(newPattern.getPatternDetails());
        }
        mergedDetails.put("mergedSources", Arrays.asList(
            getSourceFromMetadata(existing),
            getSourceFromMetadata(newPattern)
        ));
        mergedDetails.put("multiSourceConfirmation", true);
        merged.setPatternDetails(mergedDetails);

        // Update metadata
        PatternEvent.PatternMetadata mergedMetadata = new PatternEvent.PatternMetadata();
        mergedMetadata.setAlgorithm("MULTI_SOURCE_MERGED");
        mergedMetadata.setVersion("1.0.0");

        Map<String, Object> params = new HashMap<>();
        params.put("originalSource", getSourceFromMetadata(existing));
        params.put("confirmingSource", getSourceFromMetadata(newPattern));
        params.put("confidenceBoost", combinedConfidence - existing.getConfidence());
        mergedMetadata.setAlgorithmParameters(params);

        // Average processing time
        double avgProcessingTime = (
            existing.getPatternMetadata().getProcessingTime() +
            newPattern.getPatternMetadata().getProcessingTime()
        ) / 2.0;
        mergedMetadata.setProcessingTime(avgProcessingTime);
        mergedMetadata.setQualityScore("HIGH"); // Multi-source is always high quality

        merged.setPatternMetadata(mergedMetadata);

        // Merge tags
        Set<String> allTags = new HashSet<>();
        if (existing.getTags() != null) {
            allTags.addAll(existing.getTags());
        }
        if (newPattern.getTags() != null) {
            allTags.addAll(newPattern.getTags());
        }
        allTags.add("MULTI_SOURCE_CONFIRMED");
        merged.setTags(allTags);

        return merged;
    }

    /**
     * Get highest severity between two values
     */
    private String getHighestSeverity(String sev1, String sev2) {
        List<String> severityOrder = Arrays.asList("LOW", "MODERATE", "HIGH", "CRITICAL");
        int idx1 = severityOrder.indexOf(sev1.toUpperCase());
        int idx2 = severityOrder.indexOf(sev2.toUpperCase());

        if (idx1 < 0) idx1 = 0;
        if (idx2 < 0) idx2 = 0;

        return severityOrder.get(Math.max(idx1, idx2));
    }

    /**
     * Extract source algorithm from pattern metadata
     */
    private String getSourceFromMetadata(PatternEvent pattern) {
        if (pattern.getPatternMetadata() != null &&
            pattern.getPatternMetadata().getAlgorithm() != null) {
            return pattern.getPatternMetadata().getAlgorithm();
        }
        return "UNKNOWN_SOURCE";
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx, Collector<PatternEvent> out)
        throws Exception {
        // Cleanup expired pattern tracking
        Iterator<Map.Entry<String, Long>> iterator = recentPatternsState.iterator();
        while (iterator.hasNext()) {
            Map.Entry<String, Long> entry = iterator.next();
            if (timestamp - entry.getValue() > DEDUP_WINDOW_MS) {
                iterator.remove();
            }
        }
    }
}
