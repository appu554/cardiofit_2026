package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module12TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static com.cardiofit.flink.builders.Module12TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Integration tests for concurrent intervention detection:
 * cross-referencing, domain classification, confounded status, 3+ concurrent.
 */
class Module12ConcurrentInterventionIntegrationTest {

    @Test
    void twoOverlappingWindows_crossReferenced() {
        InterventionWindowState state = emptyState("P1");

        // Open first intervention: metformin add, day 0–28
        state.openWindow("int-1", InterventionType.MEDICATION_ADD,
                medicationDetail("METFORMIN", "Metformin", "500mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        // Open second intervention: dietary change, day 10–24
        state.openWindow("int-2", InterventionType.NUTRITION_FOOD_CHANGE, null,
                14, daysAfter(BASE_TIME, 10), TrajectoryClassification.STABLE,
                "card-2", "APPROVED");

        // Detect concurrency for int-2
        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-2", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 10), daysAfter(BASE_TIME, 24),
                state.getActiveWindows());

        assertTrue(result.getConcurrentIds().contains("int-1"));
    }

    @Test
    void sameDomainGlucose_flaggedForAttribution() {
        InterventionWindowState state = emptyState("P1");

        // Metformin (GLUCOSE domain) and food change (GLUCOSE domain)
        state.openWindow("int-1", InterventionType.MEDICATION_ADD,
                medicationDetail("METFORMIN", "Metformin", "500mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-2", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19),
                state.getActiveWindows());

        assertTrue(result.isSameDomainConcurrent());
    }

    @Test
    void crossDomain_notConfounded() {
        InterventionWindowState state = emptyState("P1");

        // ACEi (BP domain) and dietary glucose change (GLUCOSE domain)
        state.openWindow("int-1", InterventionType.MEDICATION_ADD,
                medicationDetail("ACEI", "Enalapril", "10mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-2", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19),
                state.getActiveWindows());

        assertEquals(1, result.getConcurrentIds().size());
        assertFalse(result.isSameDomainConcurrent());
    }

    @Test
    void bidirectionalCrossReference_existingWindowUpdated() {
        // Simulate the handleInterventionApproved cross-reference logic:
        // 1. Window A opened (Amlodipine) — no concurrent yet
        // 2. Window B opened (Metformin) — detects A as concurrent
        // 3. Cross-reference: A should also list B as concurrent
        InterventionWindowState state = emptyState("P1");

        // Open Amlodipine first
        state.openWindow("int-aml", InterventionType.MEDICATION_ADD,
                medicationDetail("CCB", "Amlodipine", "5mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        // At this point, Amlodipine has no concurrent interventions
        assertEquals(0, state.getWindow("int-aml").concurrentInterventionIds.size(),
                "Amlodipine should have 0 concurrent at open time");

        // Open Metformin second (day 3, overlapping)
        state.openWindow("int-met", InterventionType.MEDICATION_DOSE_INCREASE,
                medicationDetail("METFORMIN", "Metformin", "1000mg"),
                28, daysAfter(BASE_TIME, 3), TrajectoryClassification.STABLE,
                "card-2", "APPROVED");

        // Detect concurrency for Metformin
        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-met", InterventionType.MEDICATION_DOSE_INCREASE,
                medicationDetail("METFORMIN", "Metformin", "1000mg"),
                daysAfter(BASE_TIME, 3), daysAfter(BASE_TIME, 31),
                state.getActiveWindows());

        // Metformin detects Amlodipine as concurrent
        assertTrue(result.getConcurrentIds().contains("int-aml"));

        // Simulate handleInterventionApproved cross-reference (lines 155-159)
        state.getWindow("int-met").concurrentInterventionIds.addAll(result.getConcurrentIds());
        for (String concurrentId : result.getConcurrentIds()) {
            InterventionWindowState.InterventionWindow existing = state.getWindow(concurrentId);
            if (existing != null && !existing.concurrentInterventionIds.contains("int-met")) {
                existing.concurrentInterventionIds.add("int-met");
            }
        }

        // VERIFY: Amlodipine now retroactively lists Metformin as concurrent
        assertEquals(1, state.getWindow("int-aml").concurrentInterventionIds.size(),
                "Amlodipine should now have 1 concurrent after cross-reference");
        assertTrue(state.getWindow("int-aml").concurrentInterventionIds.contains("int-met"),
                "Amlodipine should list Metformin as concurrent");

        // Metformin also has Amlodipine
        assertEquals(1, state.getWindow("int-met").concurrentInterventionIds.size());
        assertTrue(state.getWindow("int-met").concurrentInterventionIds.contains("int-aml"));
    }

    @Test
    void threePlusConcurrent_allDetected() {
        InterventionWindowState state = emptyState("P1");

        // Three overlapping interventions all starting within 7 days
        state.openWindow("int-1", InterventionType.MEDICATION_ADD,
                medicationDetail("METFORMIN", "Metformin", "500mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        state.openWindow("int-2", InterventionType.LIFESTYLE_ACTIVITY, null,
                14, daysAfter(BASE_TIME, 3), TrajectoryClassification.STABLE,
                "card-2", "APPROVED");

        // Detect concurrency for int-3
        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-3", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19),
                state.getActiveWindows());

        assertEquals(2, result.getConcurrentIds().size());
        assertTrue(result.getConcurrentIds().contains("int-1"));
        assertTrue(result.getConcurrentIds().contains("int-2"));
    }
}
