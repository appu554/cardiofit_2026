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
            assertFalse(AlertLifecycleManager.checkAlertFatigue(state));
        }
        assertEquals(10, state.getAlertsInLast24Hours());
    }

    @Test
    void alert11_isFatigued() {
        PatientAlertState state = new PatientAlertState("P001");
        for (int i = 0; i < 10; i++) {
            AlertLifecycleManager.checkAlertFatigue(state);
        }
        assertTrue(AlertLifecycleManager.checkAlertFatigue(state));
    }

    @Test
    void windowReset_after24Hours() {
        PatientAlertState state = new PatientAlertState("P001");
        state.setAlertsInLast24Hours(10);
        state.setAlertWindowStart(System.currentTimeMillis() - 25 * 60 * 60 * 1000L);
        assertFalse(AlertLifecycleManager.checkAlertFatigue(state));
        assertEquals(1, state.getAlertsInLast24Hours());
    }

    @Test
    void counterIncrements_correctly() {
        PatientAlertState state = new PatientAlertState("P001");
        AlertLifecycleManager.checkAlertFatigue(state);
        assertEquals(1, state.getAlertsInLast24Hours());
        AlertLifecycleManager.checkAlertFatigue(state);
        assertEquals(2, state.getAlertsInLast24Hours());
    }
}
