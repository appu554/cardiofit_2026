package com.cardiofit.flink.operators;

import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6AlertLifecycleTest {

    @Test
    void newAlert_startsAsActive() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        assertEquals(AlertState.ACTIVE, alert.getState());
        assertNotNull(alert.getAlertId());
        assertTrue(alert.getSlaDeadlineMs() > 0, "HALT must have SLA deadline");
    }

    @Test
    void haltSla_is30minutes() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        long expectedDeadline = alert.getCreatedAt() + (30 * 60 * 1000L);
        assertEquals(expectedDeadline, alert.getSlaDeadlineMs());
    }

    @Test
    void pauseSla_is24hours() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.PAUSE, "AKI", "MODULE_3_CDS");
        long expectedDeadline = alert.getCreatedAt() + (24 * 60 * 60 * 1000L);
        assertEquals(expectedDeadline, alert.getSlaDeadlineMs());
    }

    @Test
    void softFlagSla_isNegativeOne() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.SOFT_FLAG, "TREND", "MODULE_4_CEP");
        assertEquals(-1L, alert.getSlaDeadlineMs());
    }

    @Test
    void createAlert_withEventTime_usesEventTime() {
        long eventTime = 1700000000000L;
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS", eventTime);
        assertEquals(eventTime, alert.getCreatedAt());
        assertEquals(eventTime + (30 * 60 * 1000L), alert.getSlaDeadlineMs());
    }

    @Test
    void alertFatigue_capsAt10Per24Hours() {
        PatientAlertState state = new PatientAlertState("P001");
        for (int i = 0; i < 10; i++) {
            assertFalse(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE),
                "Alert " + (i+1) + " should not be fatigued");
        }
        assertTrue(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE),
            "Alert 11 should trigger fatigue protection");
    }

    @Test
    void alertFatigue_resetsAfter24Hours() {
        PatientAlertState state = new PatientAlertState("P001");
        state.setAlertsInLast24Hours(10);
        state.setAlertWindowStart(System.currentTimeMillis() - 25 * 60 * 60 * 1000L);
        assertFalse(AlertLifecycleManager.checkAlertFatigue(state, ActionTier.PAUSE),
            "Fatigue should reset after 24h window expires");
    }

    @Test
    void escalation_level1_goesToCareCoordinator() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        AlertLifecycleManager.escalate(alert);
        assertEquals(AlertState.ESCALATED, alert.getState());
        assertEquals(1, alert.getEscalationLevel());
        assertEquals("CARE_COORDINATOR", alert.getAssignedTo());
    }

    @Test
    void escalation_level2_goesToClinicalSupervisor() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        AlertLifecycleManager.escalate(alert); // level 1
        AlertLifecycleManager.escalate(alert); // level 2
        assertEquals(2, alert.getEscalationLevel());
        assertEquals("CLINICAL_SUPERVISOR", alert.getAssignedTo());
    }

    // ── Timer mapping tests ──

    @Test
    void timerMapping_registerAndPop() {
        PatientAlertState state = new PatientAlertState("P001");
        state.registerTimerMapping(1700000000000L, "alert-123");
        assertEquals("alert-123", state.popTimerMapping(1700000000000L));
        assertNull(state.popTimerMapping(1700000000000L), "Second pop should return null");
    }

    @Test
    void timerMapping_unknownTimestamp_returnsNull() {
        PatientAlertState state = new PatientAlertState("P001");
        assertNull(state.popTimerMapping(9999999999999L));
    }
}
