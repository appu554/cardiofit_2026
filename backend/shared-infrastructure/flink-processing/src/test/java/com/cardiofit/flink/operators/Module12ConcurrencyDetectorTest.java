package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class Module12ConcurrencyDetectorTest {

    private static final long DAY_MS = 86_400_000L;
    private static final long BASE_TIME = 1743552000000L;

    private static long daysAfter(long base, int days) { return base + days * DAY_MS; }

    private static Map<String, Object> medicationDetail(String drugClass) {
        Map<String, Object> detail = new HashMap<>();
        detail.put("drug_class", drugClass);
        return detail;
    }

    private static InterventionWindowState.InterventionWindow makeWindow(
            String id, InterventionType type, long start, long end,
            Map<String, Object> detail) {
        InterventionWindowState.InterventionWindow w = new InterventionWindowState.InterventionWindow();
        w.interventionId = id;
        w.interventionType = type;
        w.interventionDetail = detail;
        w.observationStartMs = start;
        w.observationEndMs = end;
        w.status = "OBSERVING";
        return w;
    }

    @Test
    void noOverlap_emptyConcurrentList() {
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        active.put("int-A", makeWindow("int-A",
                InterventionType.LIFESTYLE_ACTIVITY, BASE_TIME, daysAfter(BASE_TIME, 14), null));

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 20), daysAfter(BASE_TIME, 34), active);

        assertTrue(result.getConcurrentIds().isEmpty());
        assertFalse(result.isSameDomainConcurrent());
    }

    @Test
    void partialOverlapBelowThreshold_emptyConcurrentList() {
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        active.put("int-A", makeWindow("int-A",
                InterventionType.MEDICATION_ADD, BASE_TIME, daysAfter(BASE_TIME, 28),
                medicationDetail("METFORMIN")));

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.MEDICATION_ADD, medicationDetail("SGLT2I"),
                daysAfter(BASE_TIME, 23), daysAfter(BASE_TIME, 51), active);

        assertTrue(result.getConcurrentIds().isEmpty());
    }

    @Test
    void partialOverlapAboveThreshold_bothCrossReferenced() {
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        active.put("int-A", makeWindow("int-A",
                InterventionType.LIFESTYLE_ACTIVITY, BASE_TIME, daysAfter(BASE_TIME, 28), null));

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 15), daysAfter(BASE_TIME, 29), active);

        assertEquals(1, result.getConcurrentIds().size());
        assertTrue(result.getConcurrentIds().contains("int-A"));
    }

    @Test
    void sameDomainConcurrent_flagged() {
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        active.put("int-A", makeWindow("int-A",
                InterventionType.MEDICATION_ADD, BASE_TIME, daysAfter(BASE_TIME, 28),
                medicationDetail("METFORMIN")));

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.NUTRITION_FOOD_CHANGE, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19), active);

        assertTrue(result.isSameDomainConcurrent());
    }

    @Test
    void crossDomainConcurrent_notFlaggedAsSameDomain() {
        Map<String, InterventionWindowState.InterventionWindow> active = new HashMap<>();
        active.put("int-A", makeWindow("int-A",
                InterventionType.MEDICATION_ADD, BASE_TIME, daysAfter(BASE_TIME, 28),
                medicationDetail("METFORMIN")));

        Module12ConcurrencyDetector.Result result = Module12ConcurrencyDetector.detect(
                "int-B", InterventionType.NUTRITION_SODIUM_REDUCTION, null,
                daysAfter(BASE_TIME, 5), daysAfter(BASE_TIME, 19), active);

        assertEquals(1, result.getConcurrentIds().size());
        assertFalse(result.isSameDomainConcurrent());
    }
}
