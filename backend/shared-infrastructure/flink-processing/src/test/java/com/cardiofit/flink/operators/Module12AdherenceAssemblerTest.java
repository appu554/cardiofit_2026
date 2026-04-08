package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module12AdherenceAssembler: adherence signal assembly
 * with data quality tier classification.
 */
class Module12AdherenceAssemblerTest {

    @Test
    void medicationAdherence_fromReminderAcks() {
        Map<String, Object> signals = new HashMap<>();
        signals.put("reminder_ack_rate", 0.85);
        signals.put("refill_compliance", 0.70);

        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.MEDICATION_ADD, signals);

        // max(reminder_ack_rate, refill_compliance) = 0.85
        assertEquals(0.85, result.getAdherenceScore(), 0.01);
        assertEquals("HIGH", result.getDataQuality());
    }

    @Test
    void lifestyleActivity_fromWearableData() {
        Map<String, Object> signals = new HashMap<>();
        signals.put("activity_sessions_per_week", 3.0);
        signals.put("target_sessions_per_week", 5.0);
        signals.put("data_source", "WEARABLE");

        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.LIFESTYLE_ACTIVITY, signals);

        // 3/5 = 0.6
        assertEquals(0.60, result.getAdherenceScore(), 0.01);
        assertEquals("HIGH", result.getDataQuality());
    }

    @Test
    void nutritionFoodChange_fromMealLogs() {
        Map<String, Object> signals = new HashMap<>();
        signals.put("meals_with_prescribed_change", 8.0);
        signals.put("total_meals_logged", 14.0);
        signals.put("data_source", "APP_SELF_REPORT");

        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.NUTRITION_FOOD_CHANGE, signals);

        // 8/14 ≈ 0.571
        assertEquals(0.571, result.getAdherenceScore(), 0.01);
        assertEquals("MODERATE", result.getDataQuality());
    }

    @Test
    void noAdherenceData_defaultsToHalf() {
        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.MEDICATION_ADD, null);

        assertEquals(0.50, result.getAdherenceScore(), 0.01);
        assertEquals("LOW", result.getDataQuality());
    }

    @Test
    void adherenceScore_cappedAtOne() {
        Map<String, Object> signals = new HashMap<>();
        signals.put("activity_sessions_per_week", 7.0);
        signals.put("target_sessions_per_week", 5.0);

        Module12AdherenceAssembler.Result result = Module12AdherenceAssembler.assemble(
                InterventionType.LIFESTYLE_ACTIVITY, signals);

        // 7/5 = 1.4 → capped at 1.0
        assertEquals(1.0, result.getAdherenceScore(), 0.01);
    }
}
