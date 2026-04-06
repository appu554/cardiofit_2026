package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class Module13StateSyncLifecycleTest {

    // --- Test 1: State creation for new patient ---
    @Test
    void newPatient_stateCreatedWithPatientId() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");
        assertNotNull(state);
        assertEquals("p1", state.getPatientId());
        assertNull(state.current().fbg);
        assertNull(state.getLastComputedVelocity());
        assertTrue(state.getActiveInterventions().isEmpty());
        assertFalse(state.hasVelocityData(), "New patient should not have velocity data");
    }

    // --- Test 2: BP variability event updates correct fields ---
    @Test
    void bpVariabilityEvent_updatesStateFields() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");
        CanonicalEvent event = Module13TestBuilder.bpVariabilityEvent(
                "p1", Module13TestBuilder.BASE_TIME, 15.0, "ELEVATED", 148.0, 93.0);

        Map<String, Object> payload = event.getPayload();
        state.current().arv = ((Number) payload.get("arv")).doubleValue();
        state.current().meanSBP = ((Number) payload.get("mean_sbp")).doubleValue();
        state.current().meanDBP = ((Number) payload.get("mean_dbp")).doubleValue();

        assertEquals(15.0, state.current().arv);
        assertEquals(148.0, state.current().meanSBP);
        assertEquals(93.0, state.current().meanDBP);
    }

    // --- Test 3: Engagement event preserves previous score ---
    @Test
    void engagementEvent_preservesPreviousScore() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        Double before = state.current().engagementScore; // 0.72

        state.setPreviousEngagementScore(state.current().engagementScore);
        state.current().engagementScore = 0.45;

        assertEquals(before, state.getPreviousEngagementScore());
        assertEquals(0.45, state.current().engagementScore);
    }

    // --- Test 4: Intervention window OPENED adds to activeInterventions ---
    @Test
    void interventionWindowOpened_addsToActiveMap() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");
        assertTrue(state.getActiveInterventions().isEmpty());

        ClinicalStateSummary.InterventionWindowSummary iw = new ClinicalStateSummary.InterventionWindowSummary();
        iw.setInterventionId("int-001");
        iw.setStatus("OPENED");
        iw.setInterventionType(InterventionType.MEDICATION_ADD);
        state.getActiveInterventions().put("int-001", iw);

        assertEquals(1, state.getActiveInterventions().size());
        assertEquals("OPENED", state.getActiveInterventions().get("int-001").getStatus());
    }

    // --- Test 5: Intervention window CLOSED removes from activeInterventions ---
    @Test
    void interventionWindowClosed_removesFromActiveMap() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithActiveIntervention(
                "p1", "int-001", InterventionType.MEDICATION_ADD);
        assertEquals(1, state.getActiveInterventions().size());

        state.getActiveInterventions().remove("int-001");
        assertTrue(state.getActiveInterventions().isEmpty());
    }

    // --- Test 6: Recent intervention deltas capped at 10 ---
    @Test
    void interventionDeltas_cappedAt10() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");

        for (int i = 0; i < 12; i++) {
            ClinicalStateSummary.InterventionDeltaSummary d = new ClinicalStateSummary.InterventionDeltaSummary();
            d.setInterventionId("int-" + i);
            d.setAttribution(TrajectoryAttribution.IMPROVEMENT_CONTINUED);
            state.getRecentInterventionDeltas().add(d);
            while (state.getRecentInterventionDeltas().size() > 10) {
                state.getRecentInterventionDeltas().remove(0);
            }
        }

        assertEquals(10, state.getRecentInterventionDeltas().size());
        assertEquals("int-2", state.getRecentInterventionDeltas().get(0).getInterventionId());
    }

    // --- Test 7: Coalescing buffer accumulation and flush (ADAPTED for new API) ---
    @Test
    void coalescingBuffer_accumulatesAndFlushes() {
        ClinicalStateSummary state = Module13TestBuilder.emptyState("p1");
        assertEquals(-1L, state.getCoalescingTimerMs());
        assertTrue(state.getCoalescingBuffer().isEmpty());

        // Simulate buffering 3 KB-20 updates using NEW single-field API
        KB20StateUpdate u1 = KB20StateUpdate.builder().patientId("p1")
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("module7").fieldPath("arv").value(12.0)
                .updateTimestamp(Module13TestBuilder.BASE_TIME).build();
        KB20StateUpdate u2 = KB20StateUpdate.builder().patientId("p1")
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("module9").fieldPath("engagement_score").value(0.7)
                .updateTimestamp(Module13TestBuilder.BASE_TIME).build();
        KB20StateUpdate u3 = KB20StateUpdate.builder().patientId("p1")
                .operation(KB20StateUpdate.UpdateOperation.REPLACE)
                .sourceModule("module13").fieldPath("ckm_risk_velocity").value("STABLE")
                .updateTimestamp(Module13TestBuilder.BASE_TIME).build();

        state.getCoalescingBuffer().add(u1);
        state.getCoalescingBuffer().add(u2);
        state.getCoalescingBuffer().add(u3);
        state.setCoalescingTimerMs(Module13TestBuilder.BASE_TIME + 5000L);

        assertEquals(3, state.getCoalescingBuffer().size());
        assertTrue(state.getCoalescingTimerMs() > 0, "Timer should be registered");

        // Simulate timer fire: flush buffer
        List<KB20StateUpdate> flushed = new ArrayList<>(state.getCoalescingBuffer());
        state.getCoalescingBuffer().clear();
        state.setCoalescingTimerMs(-1L);

        assertEquals(3, flushed.size());
        assertTrue(state.getCoalescingBuffer().isEmpty(), "Buffer should be empty after flush");
        assertEquals(-1L, state.getCoalescingTimerMs(), "Timer should be reset after flush");
    }

    // --- Test 8: Snapshot rotation moves current → previous ---
    @Test
    void snapshotRotation_currentBecomesPrevious() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        assertFalse(state.hasVelocityData(), "Before rotation, no previous snapshot");

        state.current().fbg = 130.0;
        state.current().egfr = 65.0;

        state.rotateSnapshots(Module13TestBuilder.BASE_TIME + Module13TestBuilder.WEEK_MS);

        assertTrue(state.hasVelocityData(), "After rotation, previous snapshot exists");
        assertEquals(130.0, state.previous().fbg);
        assertEquals(65.0, state.previous().egfr);
        assertNotNull(state.current().fbg);
    }
}
