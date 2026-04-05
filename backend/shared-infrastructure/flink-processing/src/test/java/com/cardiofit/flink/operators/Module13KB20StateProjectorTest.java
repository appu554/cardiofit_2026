package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;

import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class Module13KB20StateProjectorTest {

    @Test
    void project_bpVariabilityEvent_correctFieldMapping() {
        CanonicalEvent event = Module13TestBuilder.bpVariabilityEvent(
                "p1", Module13TestBuilder.BASE_TIME, 12.5, "MODERATE", 138.0, 85.0);

        List<KB20StateUpdate> updates = Module13KB20StateProjector.project(event);

        assertEquals(4, updates.size());
        // All should be REPLACE with sourceModule "module7"
        assertTrue(updates.stream().allMatch(u -> u.getOperation() == KB20StateUpdate.UpdateOperation.REPLACE));
        assertTrue(updates.stream().allMatch(u -> "module7".equals(u.getSourceModule())));
        // Check first update: bp_variability_arv
        KB20StateUpdate arvUpdate = updates.stream()
                .filter(u -> "bp_variability_arv".equals(u.getFieldPath()))
                .findFirst()
                .orElseThrow(() -> new AssertionError("bp_variability_arv not found"));
        assertEquals("p1", arvUpdate.getPatientId());
        assertEquals(12.5, (double) arvUpdate.getValue(), 0.001);
        assertEquals(Module13TestBuilder.BASE_TIME, arvUpdate.getUpdateTimestamp());
    }

    @Test
    void project_engagementEvent_correctFieldMapping() {
        CanonicalEvent event = Module13TestBuilder.engagementEvent(
                "p2", Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS,
                0.78, "GREEN", "CONSISTENT_TRACKER", "TIER_2_SMBG");

        List<KB20StateUpdate> updates = Module13KB20StateProjector.project(event);

        assertEquals(4, updates.size());
        assertTrue(updates.stream().allMatch(u -> u.getOperation() == KB20StateUpdate.UpdateOperation.REPLACE));
        assertTrue(updates.stream().allMatch(u -> "module9".equals(u.getSourceModule())));
        KB20StateUpdate scoreUpdate = updates.stream()
                .filter(u -> "engagement_composite_score".equals(u.getFieldPath()))
                .findFirst()
                .orElseThrow(() -> new AssertionError("engagement_composite_score not found"));
        assertEquals("p2", scoreUpdate.getPatientId());
        assertEquals(0.78, (double) scoreUpdate.getValue(), 0.001);
    }

    @Test
    void project_interventionWindowOpened_appendToActiveInterventions() {
        long windowStart = Module13TestBuilder.BASE_TIME;
        long windowEnd = Module13TestBuilder.BASE_TIME + 28 * Module13TestBuilder.DAY_MS;
        CanonicalEvent event = Module13TestBuilder.interventionWindowEvent(
                "p3", Module13TestBuilder.BASE_TIME, "iv-001", "OPENED", "LIFESTYLE", windowStart, windowEnd);

        List<KB20StateUpdate> updates = Module13KB20StateProjector.project(event);

        assertEquals(5, updates.size());
        assertTrue(updates.stream().allMatch(u -> u.getOperation() == KB20StateUpdate.UpdateOperation.APPEND));
        assertTrue(updates.stream().allMatch(u -> "module12".equals(u.getSourceModule())));
        assertTrue(updates.stream().allMatch(u -> u.getFieldPath().startsWith("active_interventions.")));
    }

    @Test
    void project_interventionDelta_appendToOutcomes() {
        CanonicalEvent event = Module13TestBuilder.interventionDeltaEvent(
                "p4", Module13TestBuilder.BASE_TIME + 28 * Module13TestBuilder.DAY_MS,
                "iv-001", "MEDICATION_DRIVEN", 0.85, -8.2, -5.0, 3.0);

        List<KB20StateUpdate> updates = Module13KB20StateProjector.project(event);

        assertEquals(6, updates.size());
        assertTrue(updates.stream().allMatch(u -> u.getOperation() == KB20StateUpdate.UpdateOperation.APPEND));
        assertTrue(updates.stream().allMatch(u -> "module12b".equals(u.getSourceModule())));
        assertTrue(updates.stream().allMatch(u -> u.getFieldPath().startsWith("intervention_outcomes.")));
    }

    @Test
    void project_unknownSourceModule_returnsEmptyList() {
        CanonicalEvent event = CanonicalEvent.builder()
                .id("evt-unknown")
                .patientId("p5")
                .eventType(EventType.UNKNOWN)
                .eventTime(Module13TestBuilder.BASE_TIME)
                .payload(new java.util.HashMap<String, Object>() {{
                    put("source_module", "module99_nonexistent");
                    put("some_field", "some_value");
                }})
                .build();

        List<KB20StateUpdate> updates = Module13KB20StateProjector.project(event);

        assertNotNull(updates);
        assertTrue(updates.isEmpty());
    }
}
