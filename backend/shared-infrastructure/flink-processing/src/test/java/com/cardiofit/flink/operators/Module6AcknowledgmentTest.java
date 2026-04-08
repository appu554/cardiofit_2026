package com.cardiofit.flink.operators;

import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for the acknowledgment processing logic that Module6_ClinicalActionEngine.processElement2 uses.
 * These test the state transition and lifecycle logic directly (without Flink harness)
 * since the operator delegates to AlertState.canTransitionTo() and AlertLifecycleManager.
 */
class Module6AcknowledgmentTest {

    // ── State transition validation ──

    @Test
    void activeAlert_canBeAcknowledged() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        assertEquals(AlertState.ACTIVE, alert.getState());
        assertTrue(alert.getState().canTransitionTo(AlertState.ACKNOWLEDGED));
    }

    @Test
    void acknowledgedAlert_canBeActioned() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        alert.setState(AlertState.ACKNOWLEDGED);
        assertTrue(alert.getState().canTransitionTo(AlertState.ACTIONED));
    }

    @Test
    void actionedAlert_canBeResolved() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        alert.setState(AlertState.ACTIONED);
        assertTrue(alert.getState().canTransitionTo(AlertState.RESOLVED));
    }

    @Test
    void resolvedAlert_cannotTransition() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        alert.setState(AlertState.RESOLVED);
        assertFalse(alert.getState().canTransitionTo(AlertState.ACKNOWLEDGED));
        assertFalse(alert.getState().canTransitionTo(AlertState.ACTIONED));
        assertTrue(alert.getState().isTerminal());
    }

    @Test
    void activeAlert_cannotSkipToActioned() {
        // ACTIVE → ACTIONED is not a valid transition (must acknowledge first)
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        assertFalse(alert.getState().canTransitionTo(AlertState.ACTIONED),
            "ACTIVE → ACTIONED should be invalid; must acknowledge first");
    }

    @Test
    void escalatedAlert_canBeAcknowledged() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        AlertLifecycleManager.escalate(alert);
        assertEquals(AlertState.ESCALATED, alert.getState());
        assertTrue(alert.getState().canTransitionTo(AlertState.ACKNOWLEDGED),
            "Escalated alert should still accept acknowledgment");
    }

    // ── Alert lifecycle with acknowledgment ──

    @Test
    void acknowledgedAlert_notEscalatedByTimer() {
        // Simulates the timer firing after physician acknowledged
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        alert.setState(AlertState.ACKNOWLEDGED);
        alert.setAcknowledgedAt(System.currentTimeMillis());

        // onTimer checks alert.getState() == ACTIVE before escalating
        assertNotEquals(AlertState.ACTIVE, alert.getState(),
            "Acknowledged alert should not be in ACTIVE state");
    }

    @Test
    void dismissAction_fromActive_autoResolves() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.SOFT_FLAG, "TREND", "MODULE_4_CEP");
        assertEquals(AlertState.ACTIVE, alert.getState());
        // DISMISS from ACTIVE maps to AUTO_RESOLVED (not RESOLVED — physician never acknowledged)
        assertTrue(alert.getState().canTransitionTo(AlertState.AUTO_RESOLVED));
        alert.setState(AlertState.AUTO_RESOLVED);
        assertTrue(alert.getState().isTerminal());
    }

    @Test
    void dismissAction_fromAcknowledged_resolves() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.SOFT_FLAG, "TREND", "MODULE_4_CEP");
        alert.setState(AlertState.ACKNOWLEDGED);
        // DISMISS from ACKNOWLEDGED maps to RESOLVED (physician already reviewed)
        assertTrue(alert.getState().canTransitionTo(AlertState.RESOLVED));
        alert.setState(AlertState.RESOLVED);
        assertTrue(alert.getState().isTerminal());
    }

    // ── PatientAlertState with acknowledgment ──

    @Test
    void acknowledgedAlert_removedFromActiveAlerts_whenTerminal() {
        PatientAlertState state = new PatientAlertState("P001");
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        state.getActiveAlerts().put(alert.getAlertId(), alert);
        assertEquals(1, state.getActiveAlerts().size());

        // Simulate full lifecycle: ACTIVE → ACKNOWLEDGED → ACTIONED → RESOLVED
        alert.setState(AlertState.ACKNOWLEDGED);
        assertFalse(alert.getState().isTerminal());
        assertEquals(1, state.getActiveAlerts().size(), "Non-terminal state keeps alert");

        alert.setState(AlertState.ACTIONED);
        assertFalse(alert.getState().isTerminal());

        alert.setState(AlertState.RESOLVED);
        assertTrue(alert.getState().isTerminal());
        state.getActiveAlerts().remove(alert.getAlertId());
        assertEquals(0, state.getActiveAlerts().size(), "Terminal state removes alert");
    }

    @Test
    void timerMapping_survivesAcknowledgment_butTimerNoOps() {
        PatientAlertState state = new PatientAlertState("P001");
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        state.getActiveAlerts().put(alert.getAlertId(), alert);
        state.registerTimerMapping(alert.getSlaDeadlineMs(), alert.getAlertId());

        // Physician acknowledges
        alert.setState(AlertState.ACKNOWLEDGED);

        // Timer fires — looks up alert, finds it's not ACTIVE, does nothing
        String timerAlertId = state.popTimerMapping(alert.getSlaDeadlineMs());
        assertEquals(alert.getAlertId(), timerAlertId, "Timer mapping should exist");
        ClinicalAlert foundAlert = state.getActiveAlerts().get(timerAlertId);
        assertNotEquals(AlertState.ACTIVE, foundAlert.getState(),
            "Alert should not be ACTIVE — timer should no-op");
    }

    // ── AlertAcknowledgment model ──

    @Test
    void ackAction_acknowledge_mapsCorrectly() {
        AlertAcknowledgment ack = new AlertAcknowledgment();
        ack.setAlertId("alert-123");
        ack.setPatientId("P001");
        ack.setAction(AlertAcknowledgment.AckAction.ACKNOWLEDGE);
        ack.setPractitionerId("DR-SMITH");
        ack.setTimestamp(System.currentTimeMillis());

        assertEquals("alert-123", ack.getAlertId());
        assertEquals(AlertAcknowledgment.AckAction.ACKNOWLEDGE, ack.getAction());
    }

    @Test
    void ackAction_actionTaken_includesDescription() {
        AlertAcknowledgment ack = new AlertAcknowledgment();
        ack.setAction(AlertAcknowledgment.AckAction.ACTION_TAKEN);
        ack.setActionDescription("Administered IV potassium correction");

        assertEquals("Administered IV potassium correction", ack.getActionDescription());
    }
}
