package com.cardiofit.flink.cds.time;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.protocol.models.Protocol;
import com.cardiofit.flink.protocol.models.TimeConstraint;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for TimeConstraintTracker.
 *
 * Test coverage:
 * - On-track tests (>30 min remaining, INFO alert): 2 tests
 * - Warning tests (10-30 min remaining, WARNING alert): 3 tests
 * - Critical tests (deadline exceeded, CRITICAL alert): 3 tests
 * - Bundle compliance tests: 2 tests
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
class TimeConstraintTrackerTest {

    private TimeConstraintTracker tracker;
    private Protocol protocol;
    private EnrichedPatientContext context;

    @BeforeEach
    void setUp() {
        tracker = new TimeConstraintTracker();
        protocol = createTestProtocol();
        context = createTestContext();
    }

    // ============================================================
    // ON-TRACK TESTS (>30 min remaining, INFO alert): 2 tests
    // ============================================================

    @Test
    void testEvaluateConstraint_OnTrack_50MinutesRemaining() {
        // Trigger 10 minutes ago, deadline 60 minutes from trigger (50 min remaining)
        Instant triggerTime = Instant.now().minus(10, ChronoUnit.MINUTES);
        context.setTriggerTime(triggerTime);

        TimeConstraint constraint = new TimeConstraint(
            "HOUR-1",
            "Hour-1 Bundle",
            60,
            true
        );
        protocol.addTimeConstraint(constraint);

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        assertEquals(1, status.getConstraintStatuses().size());
        ConstraintStatus constraintStatus = status.getConstraintStatuses().get(0);

        assertEquals(AlertLevel.INFO, constraintStatus.getAlertLevel());
        assertFalse(constraintStatus.isDeadlineExceeded());
        assertTrue(constraintStatus.getMinutesRemaining() > 30);
        assertTrue(constraintStatus.getMessage().contains("remaining"));
    }

    @Test
    void testEvaluateConstraint_OnTrack_90MinutesRemaining() {
        // Trigger 90 minutes ago, deadline 180 minutes from trigger (90 min remaining)
        Instant triggerTime = Instant.now().minus(90, ChronoUnit.MINUTES);
        context.setTriggerTime(triggerTime);

        TimeConstraint constraint = new TimeConstraint(
            "HOUR-3",
            "Hour-3 Bundle",
            180,
            false
        );
        protocol.addTimeConstraint(constraint);

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        ConstraintStatus constraintStatus = status.getConstraintStatuses().get(0);

        assertEquals(AlertLevel.INFO, constraintStatus.getAlertLevel());
        assertFalse(status.hasCriticalAlerts());
        assertFalse(status.hasWarningAlerts());
        assertTrue(constraintStatus.getMessage().contains("1h 30m"));
    }

    // ============================================================
    // WARNING TESTS (10-30 min remaining, WARNING alert): 3 tests
    // ============================================================

    @Test
    void testEvaluateConstraint_Warning_25MinutesRemaining() {
        // Trigger 35 minutes ago, deadline 60 minutes from trigger (25 min remaining)
        Instant triggerTime = Instant.now().minus(35, ChronoUnit.MINUTES);
        context.setTriggerTime(triggerTime);

        TimeConstraint constraint = new TimeConstraint(
            "HOUR-1",
            "Hour-1 Bundle",
            60,
            true  // Critical constraint
        );
        protocol.addTimeConstraint(constraint);

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        ConstraintStatus constraintStatus = status.getConstraintStatuses().get(0);

        assertEquals(AlertLevel.WARNING, constraintStatus.getAlertLevel());
        assertTrue(status.hasWarningAlerts());
        assertEquals(1, status.getWarningAlerts().size());
        assertTrue(constraintStatus.getMessage().contains("deadline in"));
        assertTrue(constraintStatus.getMinutesRemaining() <= 30);
        assertTrue(constraintStatus.getMinutesRemaining() > 0);
    }

    @Test
    void testEvaluateConstraint_Warning_15MinutesRemaining() {
        // Trigger 45 minutes ago, deadline 60 minutes from trigger (15 min remaining)
        Instant triggerTime = Instant.now().minus(45, ChronoUnit.MINUTES);
        context.setTriggerTime(triggerTime);

        TimeConstraint constraint = new TimeConstraint(
            "SEPSIS-HOUR-1",
            "Sepsis Hour-1 Bundle",
            60,
            true
        );
        protocol.addTimeConstraint(constraint);

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        ConstraintStatus constraintStatus = status.getConstraintStatuses().get(0);

        assertEquals(AlertLevel.WARNING, constraintStatus.getAlertLevel());
        assertTrue(constraintStatus.getMessage().contains("Sepsis Hour-1 Bundle deadline in"));
        assertEquals(1, status.getWarningAlerts().size());
    }

    @Test
    void testEvaluateConstraint_Warning_ExactlyAtThreshold() {
        // Trigger 30 minutes ago, deadline 60 minutes from trigger (30 min remaining exactly)
        Instant triggerTime = Instant.now().minus(30, ChronoUnit.MINUTES);
        context.setTriggerTime(triggerTime);

        TimeConstraint constraint = new TimeConstraint(
            "STEMI",
            "STEMI Door-to-Balloon",
            60,
            true
        );
        protocol.addTimeConstraint(constraint);

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        ConstraintStatus constraintStatus = status.getConstraintStatuses().get(0);

        // Should be WARNING when exactly at 30 minutes
        assertEquals(AlertLevel.WARNING, constraintStatus.getAlertLevel());
        assertEquals(30, constraintStatus.getMinutesRemaining());
    }

    // ============================================================
    // CRITICAL TESTS (deadline exceeded, CRITICAL alert): 3 tests
    // ============================================================

    @Test
    void testEvaluateConstraint_Critical_DeadlineExceededBy10Minutes() {
        // Trigger 70 minutes ago, deadline 60 minutes from trigger (10 min overdue)
        Instant triggerTime = Instant.now().minus(70, ChronoUnit.MINUTES);
        context.setTriggerTime(triggerTime);

        TimeConstraint constraint = new TimeConstraint(
            "HOUR-1",
            "Hour-1 Bundle",
            60,
            true
        );
        protocol.addTimeConstraint(constraint);

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        ConstraintStatus constraintStatus = status.getConstraintStatuses().get(0);

        assertEquals(AlertLevel.CRITICAL, constraintStatus.getAlertLevel());
        assertTrue(constraintStatus.isDeadlineExceeded());
        assertTrue(status.hasCriticalAlerts());
        assertEquals(1, status.getCriticalAlerts().size());
        assertTrue(constraintStatus.getMessage().contains("exceeded"));
        assertTrue(constraintStatus.getMinutesRemaining() < 0);
    }

    @Test
    void testEvaluateConstraint_Critical_DeadlineExceededBy45Minutes() {
        // Trigger 105 minutes ago, deadline 60 minutes from trigger (45 min overdue)
        Instant triggerTime = Instant.now().minus(105, ChronoUnit.MINUTES);
        context.setTriggerTime(triggerTime);

        TimeConstraint constraint = new TimeConstraint(
            "SEPSIS-HOUR-1",
            "Sepsis Hour-1 Bundle",
            60,
            true
        );
        protocol.addTimeConstraint(constraint);

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        ConstraintStatus constraintStatus = status.getConstraintStatuses().get(0);

        assertEquals(AlertLevel.CRITICAL, constraintStatus.getAlertLevel());
        assertTrue(constraintStatus.getMessage().contains("Sepsis Hour-1 Bundle deadline exceeded"));
        assertTrue(Math.abs(constraintStatus.getMinutesRemaining()) >= 40); // ~45 minutes
    }

    @Test
    void testEvaluateConstraint_Critical_MultipleConstraintsExceeded() {
        // Trigger 150 minutes ago
        Instant triggerTime = Instant.now().minus(150, ChronoUnit.MINUTES);
        context.setTriggerTime(triggerTime);

        // Hour-1 Bundle (60 min deadline) - will exceed
        TimeConstraint constraint1 = new TimeConstraint(
            "HOUR-1",
            "Hour-1 Bundle",
            60,
            true
        );

        // Hour-3 Bundle (180 min deadline) - still on track
        TimeConstraint constraint2 = new TimeConstraint(
            "HOUR-3",
            "Hour-3 Bundle",
            180,
            true
        );

        protocol.addTimeConstraint(constraint1);
        protocol.addTimeConstraint(constraint2);

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        assertEquals(2, status.getConstraintStatuses().size());
        assertTrue(status.hasCriticalAlerts());
        assertEquals(1, status.getCriticalAlerts().size());

        // Hour-1 should be CRITICAL
        ConstraintStatus hour1Status = status.getConstraintStatuses().stream()
            .filter(cs -> cs.getConstraintId().equals("HOUR-1"))
            .findFirst()
            .orElseThrow();
        assertEquals(AlertLevel.CRITICAL, hour1Status.getAlertLevel());

        // Hour-3 should be WARNING (30 min remaining)
        ConstraintStatus hour3Status = status.getConstraintStatuses().stream()
            .filter(cs -> cs.getConstraintId().equals("HOUR-3"))
            .findFirst()
            .orElseThrow();
        assertEquals(AlertLevel.WARNING, hour3Status.getAlertLevel());
    }

    // ============================================================
    // BUNDLE COMPLIANCE TESTS: 2 tests
    // ============================================================

    @Test
    void testEvaluateConstraints_SepsisBundleCompliance_AllOnTrack() {
        // Trigger 15 minutes ago
        Instant triggerTime = Instant.now().minus(15, ChronoUnit.MINUTES);
        context.setTriggerTime(triggerTime);

        // Sepsis Hour-0 Bundle (60 min)
        protocol.addTimeConstraint(new TimeConstraint(
            "SEPSIS-HOUR-0",
            "Sepsis Hour-0 Bundle",
            60,
            true
        ));

        // Sepsis Hour-1 Bundle (60 min from trigger, same as Hour-0 in this test)
        protocol.addTimeConstraint(new TimeConstraint(
            "SEPSIS-HOUR-1",
            "Sepsis Hour-1 Bundle",
            60,
            true
        ));

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        assertEquals(2, status.getConstraintStatuses().size());
        assertFalse(status.hasCriticalAlerts());
        assertFalse(status.hasWarningAlerts());

        // Both should be INFO (45 min remaining)
        for (ConstraintStatus cs : status.getConstraintStatuses()) {
            assertEquals(AlertLevel.INFO, cs.getAlertLevel());
            assertTrue(cs.getMinutesRemaining() > 30);
        }
    }

    @Test
    void testEvaluateConstraints_NoTriggerTime_UsesCurrentTime() {
        // No trigger time set - should use current time
        context.setTriggerTime(null);

        TimeConstraint constraint = new TimeConstraint(
            "HOUR-1",
            "Hour-1 Bundle",
            60,
            true
        );
        protocol.addTimeConstraint(constraint);

        TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);

        // Should not throw exception, should use current time as trigger
        assertNotNull(status);
        assertEquals(1, status.getConstraintStatuses().size());

        ConstraintStatus constraintStatus = status.getConstraintStatuses().get(0);
        assertEquals(AlertLevel.INFO, constraintStatus.getAlertLevel());

        // Deadline should be ~60 minutes from now
        assertTrue(constraintStatus.getMinutesRemaining() > 55);
        assertTrue(constraintStatus.getMinutesRemaining() <= 60);

        // Verify context now has trigger time set
        assertNotNull(context.getTriggerTime());
    }

    // ============================================================
    // HELPER METHODS
    // ============================================================

    private Protocol createTestProtocol() {
        Protocol p = new Protocol();
        p.setProtocolId("SEPSIS-BUNDLE-001");
        p.setName("Sepsis Management Bundle");
        p.setCategory("INFECTIOUS");
        return p;
    }

    private EnrichedPatientContext createTestContext() {
        EnrichedPatientContext ctx = new EnrichedPatientContext();
        ctx.setPatientId("TEST-PATIENT-001");
        ctx.setEventType("LAB_RESULT");
        ctx.setEventTime(System.currentTimeMillis());

        PatientContextState state = new PatientContextState();
        state.setPatientId("TEST-PATIENT-001");
        ctx.setPatientState(state);

        return ctx;
    }
}
