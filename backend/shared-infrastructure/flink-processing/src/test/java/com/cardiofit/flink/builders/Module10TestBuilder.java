package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * Test builder for Module 10/10b tests.
 * Follows Module9TestBuilder pattern: static factory methods for events and state.
 */
public class Module10TestBuilder {

    private static final long DAY_MS = 86_400_000L;
    private static final long HOUR_MS = 3_600_000L;
    private static final long MIN_MS = 60_000L;
    public static final long BASE_TIME = 1743552000000L; // 2025-04-02 00:00:00 UTC

    public static long daysAgo(int days) { return BASE_TIME - (days * DAY_MS); }
    public static long hoursAgo(int hours) { return BASE_TIME - (hours * HOUR_MS); }
    public static long minutesAgo(int minutes) { return BASE_TIME - (minutes * MIN_MS); }
    public static long hoursAfter(long base, int hours) { return base + hours * HOUR_MS; }
    public static long minutesAfter(long base, int minutes) { return base + minutes * MIN_MS; }

    // --- Meal Events ---

    public static CanonicalEvent mealEvent(String patientId, long timestamp,
                                            double carbGrams, double proteinGrams) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "MEAL_LOG");
        payload.put("meal_type", "lunch");
        payload.put("carb_grams", carbGrams);
        payload.put("protein_grams", proteinGrams);
        payload.put("food_description", "test meal");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent mealWithSodium(String patientId, long timestamp,
                                                  double carbGrams, double sodiumMg) {
        CanonicalEvent event = mealEvent(patientId, timestamp, carbGrams, 20.0);
        event.getPayload().put("sodium_mg", sodiumMg);
        return event;
    }

    // --- CGM Glucose Events ---

    public static CanonicalEvent cgmReading(String patientId, long timestamp, double glucose) {
        CanonicalEvent event = baseEvent(patientId, EventType.DEVICE_READING, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("glucose_value", glucose);
        payload.put("data_tier", "TIER_1_CGM");
        payload.put("source", "CGM");
        event.setPayload(payload);
        return event;
    }

    // --- SMBG Glucose Events ---

    public static CanonicalEvent smbgReading(String patientId, long timestamp, double glucose) {
        CanonicalEvent event = baseEvent(patientId, EventType.LAB_RESULT, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "glucose");
        payload.put("value", glucose);
        payload.put("source", "SMBG");
        event.setPayload(payload);
        return event;
    }

    // --- BP Events ---

    public static CanonicalEvent bpReading(String patientId, long timestamp,
                                            double sbp, double dbp) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("systolic_bp", sbp);
        payload.put("diastolic_bp", dbp);
        event.setPayload(payload);
        return event;
    }

    // --- State Builders ---

    public static MealCorrelationState emptyState(String patientId) {
        return new MealCorrelationState(patientId);
    }

    public static MealCorrelationState stateWithTier(String patientId, DataTier tier) {
        MealCorrelationState state = new MealCorrelationState(patientId);
        state.setDataTier(tier);
        return state;
    }

    public static MealCorrelationState stateWithRecentBP(String patientId,
                                                          double sbp, double dbp,
                                                          long bpTimestamp) {
        MealCorrelationState state = new MealCorrelationState(patientId);
        state.addBPReading(bpTimestamp, sbp, dbp);
        return state;
    }

    // --- Glucose Window Builders ---

    /** Create a CGM window with 5-min interval readings */
    public static GlucoseWindow cgmWindow(long startTime, double baseline, double... values) {
        GlucoseWindow w = new GlucoseWindow();
        w.setBaseline(baseline);
        w.setDataTier(DataTier.TIER_1_CGM);
        w.setWindowOpenTime(startTime);
        for (int i = 0; i < values.length; i++) {
            w.addReading(startTime + i * 5 * MIN_MS, values[i], "CGM");
        }
        w.sortByTime();
        return w;
    }

    /** Create a rapid spike pattern (peak at 15 min) */
    public static GlucoseWindow rapidSpikeWindow(long startTime) {
        return cgmWindow(startTime, 100.0,
            100, 130, 160, 180, 170, 140, 110, 105, 100, 98, 97, 96);
    }

    /** Create a flat response pattern (excursion < 20) */
    public static GlucoseWindow flatWindow(long startTime) {
        return cgmWindow(startTime, 100.0,
            100, 102, 105, 108, 110, 112, 110, 108, 105, 103, 101, 100);
    }

    // --- MealResponseRecord Builder ---

    public static MealResponseRecord mealRecord(String patientId, long timestamp,
                                                  Double excursion, Double iauc,
                                                  Double sodiumMg, Double sbpExcursion) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("food_description", "test meal");

        return MealResponseRecord.builder()
            .recordId("test-" + UUID.randomUUID())
            .patientId(patientId)
            .mealEventId("meal-" + UUID.randomUUID())
            .mealTimestamp(timestamp)
            .mealTimeCategory(MealTimeCategory.fromTimestamp(timestamp))
            .glucoseExcursion(excursion)
            .iAUC(iauc)
            .sodiumMg(sodiumMg)
            .sbpExcursion(sbpExcursion)
            .bpComplete(sbpExcursion != null)
            .dataTier(DataTier.TIER_1_CGM)
            .mealPayload(payload)
            .qualityScore(0.8)
            .build();
    }

    private static CanonicalEvent baseEvent(String patientId, EventType type, long timestamp) {
        CanonicalEvent event = new CanonicalEvent();
        event.setId(UUID.randomUUID().toString());
        event.setPatientId(patientId);
        event.setEventType(type);
        event.setEventTime(timestamp);
        event.setProcessingTime(System.currentTimeMillis());
        event.setCorrelationId("test-" + UUID.randomUUID());
        return event;
    }
}
