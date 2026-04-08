package com.cardiofit.flink.operators;

import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6AlertFatigueTest {

    @Test
    void alertsUpTo10_notFatigued() {
        PatientAlertState state = new PatientAlertState("P001");
        for (int i = 0; i < 10; i++) {
            assertFalse(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE));
        }
        assertEquals(10, state.getAlertsInLast24Hours());
    }

    @Test
    void alert11_isFatigued() {
        PatientAlertState state = new PatientAlertState("P001");
        for (int i = 0; i < 10; i++) {
            AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE);
        }
        assertTrue(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE));
    }

    @Test
    void windowReset_after24Hours() {
        PatientAlertState state = new PatientAlertState("P001");
        state.setAlertsInLast24Hours(10);
        state.setAlertWindowStart(System.currentTimeMillis() - 25 * 60 * 60 * 1000L);
        assertFalse(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE));
        assertEquals(1, state.getAlertsInLast24Hours());
    }

    @Test
    void counterIncrements_correctly() {
        PatientAlertState state = new PatientAlertState("P001");
        AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE);
        assertEquals(1, state.getAlertsInLast24Hours());
        AlertLifecycleManager.checkAlertFatigue(state, ActionTier.SOFT_FLAG);
        assertEquals(2, state.getAlertsInLast24Hours());
    }

    // ── Critical #4: HALT always bypasses fatigue ──

    @Test
    void haltAlert_neverSuppressed_evenAfterFatigueThreshold() {
        PatientAlertState state = new PatientAlertState("P001");
        // Exhaust the fatigue window with 10 PAUSE alerts
        for (int i = 0; i < 10; i++) {
            AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE);
        }
        // 11th PAUSE alert IS suppressed
        assertTrue(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE),
            "Non-HALT alert should be suppressed at fatigue threshold");

        // HALT alert NEVER suppressed — patient safety overrides fatigue
        assertFalse(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.HALT),
            "HALT alert must NEVER be suppressed by fatigue protection");
    }

    @Test
    void haltAlert_stillIncrementsCounter() {
        PatientAlertState state = new PatientAlertState("P001");
        AlertLifecycleManager.checkAlertFatigue(state, ActionTier.HALT);
        assertEquals(1, state.getAlertsInLast24Hours(),
            "HALT alerts should still increment counter for monitoring");
    }

    // ── Interaction: HALT bypasses fatigue but NOT dedup ──

    @Test
    void haltBypassesFatigue_butDedupStillSuppresses() {
        // Simulate a patient with exhausted fatigue window
        PatientAlertState state = new PatientAlertState("P001");
        for (int i = 0; i < 10; i++) {
            AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE);
        }

        // HALT bypasses fatigue — always returns false
        assertFalse(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.HALT),
            "HALT must bypass fatigue");

        // But dedup should still suppress duplicate HALT for same category within 5min
        Module6CrossModuleDedup dedup = new Module6CrossModuleDedup();
        long now = System.currentTimeMillis();
        assertTrue(dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now),
            "First HALT for SEPSIS should emit");
        assertFalse(dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now + 60_000),
            "Duplicate HALT for SEPSIS within 5min window must be suppressed by dedup");

        // Different category is NOT suppressed — these are independent clinical situations
        assertTrue(dedup.shouldEmit("P001", ActionTier.HALT, "HYPERKALEMIA", now + 60_000),
            "HALT for different category should still emit");
    }

    @Test
    void fatigueReset_thenExactly10_notFatigued() {
        PatientAlertState state = new PatientAlertState("P001");
        state.setAlertsInLast24Hours(10);
        state.setAlertWindowStart(System.currentTimeMillis() - 25 * 60 * 60 * 1000L);

        // After reset, should be able to emit exactly 10 more
        for (int i = 0; i < 10; i++) {
            assertFalse(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE),
                "Alert " + (i + 1) + " after reset should not be suppressed");
        }
        // 11th should be suppressed
        assertTrue(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE),
            "11th alert after reset should be suppressed");
    }
}
