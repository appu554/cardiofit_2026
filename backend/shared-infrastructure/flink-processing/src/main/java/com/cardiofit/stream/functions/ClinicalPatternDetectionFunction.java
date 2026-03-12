package com.cardiofit.stream.functions;

import com.cardiofit.stream.models.EnrichedPatientEvent;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;

/**
 * ClinicalPatternDetectionFunction detects clinical patterns in patient events
 * using complex event processing and temporal analysis.
 */
public class ClinicalPatternDetectionFunction extends KeyedProcessFunction<String, EnrichedPatientEvent, EnrichedPatientEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(ClinicalPatternDetectionFunction.class);

    // State to track patient event history
    private transient ValueState<List<EnrichedPatientEvent>> eventHistoryState;
    private transient ValueState<LocalDateTime> lastProcessedState;

    // Pattern detection configuration
    private static final int MAX_HISTORY_SIZE = 100;
    private static final long PATTERN_WINDOW_HOURS = 24;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        // Initialize state descriptors
        ValueStateDescriptor<List<EnrichedPatientEvent>> historyDescriptor =
                new ValueStateDescriptor<>("eventHistory",
                    (Class<List<EnrichedPatientEvent>>) (Class<?>) List.class);
        eventHistoryState = getRuntimeContext().getState(historyDescriptor);

        ValueStateDescriptor<LocalDateTime> lastProcessedDescriptor =
                new ValueStateDescriptor<>("lastProcessed", LocalDateTime.class);
        lastProcessedState = getRuntimeContext().getState(lastProcessedDescriptor);
    }

    @Override
    public void processElement(
            EnrichedPatientEvent event,
            Context context,
            Collector<EnrichedPatientEvent> out) throws Exception {

        // Get current event history
        List<EnrichedPatientEvent> history = eventHistoryState.value();
        if (history == null) {
            history = new ArrayList<>();
        }

        // Add current event to history
        history.add(event);

        // Clean up old events (keep only recent events)
        LocalDateTime cutoffTime = event.getOriginalEvent().getTimestampAsDateTime().minusHours(PATTERN_WINDOW_HOURS);
        history.removeIf(e -> e.getOriginalEvent().getTimestampAsDateTime().isBefore(cutoffTime));

        // Limit history size
        if (history.size() > MAX_HISTORY_SIZE) {
            history = history.subList(history.size() - MAX_HISTORY_SIZE, history.size());
        }

        // Detect patterns
        List<String> detectedPatterns = detectPatterns(history, event);

        // Update event with detected patterns
        if (!detectedPatterns.isEmpty()) {
            // Convert String patterns to DetectedPattern objects
            List<EnrichedPatientEvent.DetectedPattern> patternObjects = new ArrayList<>();
            for (String patternName : detectedPatterns) {
                EnrichedPatientEvent.DetectedPattern pattern = new EnrichedPatientEvent.DetectedPattern();
                pattern.setPatternType("CLINICAL");
                pattern.setPatternName(patternName);
                pattern.setDescription("Detected clinical pattern: " + patternName);
                pattern.setSeverity(determinePatternSeverity(patternName));
                pattern.setConfidence(0.85); // Default confidence score
                pattern.setTimeWindow(PATTERN_WINDOW_HOURS + " hours");
                patternObjects.add(pattern);
            }
            event.setDetectedPatterns(patternObjects);
            LOG.info("Detected patterns for patient {}: {}",
                    event.getOriginalEvent().getPatientId(), detectedPatterns);
        }

        // Update state
        eventHistoryState.update(history);
        lastProcessedState.update(LocalDateTime.now());

        // Forward enriched event
        out.collect(event);
    }

    /**
     * Detect clinical patterns in the event history
     */
    private List<String> detectPatterns(List<EnrichedPatientEvent> history, EnrichedPatientEvent currentEvent) {
        List<String> patterns = new ArrayList<>();

        // Pattern 1: Rapid vital sign deterioration
        if (detectVitalSignDeterioration(history)) {
            patterns.add("VITAL_SIGN_DETERIORATION");
        }

        // Pattern 2: Medication adherence issues
        if (detectMedicationAdherenceIssues(history)) {
            patterns.add("MEDICATION_ADHERENCE_ISSUE");
        }

        // Pattern 3: Critical value sequence
        if (detectCriticalValueSequence(history)) {
            patterns.add("CRITICAL_VALUE_SEQUENCE");
        }

        // Pattern 4: Infection onset pattern
        if (detectInfectionOnsetPattern(history)) {
            patterns.add("INFECTION_ONSET");
        }

        // Pattern 5: Cardiac event risk pattern
        if (detectCardiacRiskPattern(history)) {
            patterns.add("CARDIAC_EVENT_RISK");
        }

        return patterns;
    }

    private boolean detectVitalSignDeterioration(List<EnrichedPatientEvent> history) {
        // Check for deteriorating vital signs over time
        int deterioratingCount = 0;
        for (EnrichedPatientEvent event : history) {
            if (event.getOriginalEvent().getEventType().contains("vital") &&
                event.getOriginalEvent().isCritical()) {
                deterioratingCount++;
            }
        }
        return deterioratingCount >= 3; // 3 or more critical vital signs
    }

    private boolean detectMedicationAdherenceIssues(List<EnrichedPatientEvent> history) {
        // Detect patterns indicating medication non-adherence
        int missedMedications = 0;
        for (EnrichedPatientEvent event : history) {
            if (event.getOriginalEvent().getEventType().contains("medication") &&
                event.getSemanticEnrichment() != null &&
                Boolean.FALSE.equals(event.getSemanticEnrichment().get("adherent"))) {
                missedMedications++;
            }
        }
        return missedMedications >= 2; // Multiple missed medications
    }

    private boolean detectCriticalValueSequence(List<EnrichedPatientEvent> history) {
        // Detect sequence of critical lab values
        int criticalLabCount = 0;
        for (EnrichedPatientEvent event : history) {
            if (event.getOriginalEvent().getEventType().contains("lab") &&
                event.getOriginalEvent().isCritical()) {
                criticalLabCount++;
            }
        }
        return criticalLabCount >= 2; // 2 or more critical labs
    }

    private boolean detectInfectionOnsetPattern(List<EnrichedPatientEvent> history) {
        // Detect pattern indicating possible infection
        boolean hasFever = false;
        boolean hasElevatedWBC = false;

        for (EnrichedPatientEvent event : history) {
            String eventType = event.getOriginalEvent().getEventType();
            if (eventType.contains("temperature") && event.getOriginalEvent().isCritical()) {
                hasFever = true;
            }
            if (eventType.contains("lab") &&
                event.getSemanticEnrichment() != null &&
                "WBC_ELEVATED".equals(event.getSemanticEnrichment().get("labType"))) {
                hasElevatedWBC = true;
            }
        }

        return hasFever && hasElevatedWBC;
    }

    private boolean detectCardiacRiskPattern(List<EnrichedPatientEvent> history) {
        // Detect patterns indicating cardiac event risk
        boolean hasChestPain = false;
        boolean hasECGChanges = false;
        boolean hasTroponinElevation = false;

        for (EnrichedPatientEvent event : history) {
            String eventType = event.getOriginalEvent().getEventType();
            if (eventType.contains("symptom") &&
                event.getSemanticEnrichment() != null &&
                "CHEST_PAIN".equals(event.getSemanticEnrichment().get("symptomType"))) {
                hasChestPain = true;
            }
            if (eventType.contains("ecg") && event.getOriginalEvent().isCritical()) {
                hasECGChanges = true;
            }
            if (eventType.contains("lab") &&
                event.getSemanticEnrichment() != null &&
                "TROPONIN_ELEVATED".equals(event.getSemanticEnrichment().get("labType"))) {
                hasTroponinElevation = true;
            }
        }

        // Any two of the three indicators suggest cardiac risk
        int indicators = (hasChestPain ? 1 : 0) + (hasECGChanges ? 1 : 0) + (hasTroponinElevation ? 1 : 0);
        return indicators >= 2;
    }

    /**
     * Determine the severity level for a detected pattern
     */
    private String determinePatternSeverity(String patternName) {
        switch (patternName) {
            case "CARDIAC_EVENT_RISK":
            case "CRITICAL_VALUE_SEQUENCE":
                return "CRITICAL";
            case "VITAL_SIGN_DETERIORATION":
            case "INFECTION_ONSET":
                return "HIGH";
            case "MEDICATION_ADHERENCE_ISSUE":
                return "MEDIUM";
            default:
                return "MEDIUM";
        }
    }
}