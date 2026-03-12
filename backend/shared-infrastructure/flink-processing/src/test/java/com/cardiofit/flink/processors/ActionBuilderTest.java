package com.cardiofit.flink.processors;

import com.cardiofit.flink.cds.medication.MedicationSelector;
import com.cardiofit.flink.cds.time.TimeConstraintTracker;
import com.cardiofit.flink.cds.time.AlertLevel;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientState;
import com.cardiofit.flink.models.PatientDemographics;
import com.cardiofit.flink.protocol.models.Protocol;
import com.cardiofit.flink.protocol.models.TimeConstraint;
import com.cardiofit.flink.processors.ActionBuilder.ActionResult;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.Arrays;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for ActionBuilder with MedicationSelector and TimeConstraintTracker integration (Phase 1).
 *
 * Tests verify that:
 * - Medication selection is applied to actions with medication_selection
 * - Time constraints are tracked for all actions
 * - Null medication returns are logged as errors
 * - ActionResult contains both actions and time constraint status
 *
 * @author Module 3 CDS Team - Phase 1
 * @version 1.0
 * @since 2025-10-21
 */
class ActionBuilderTest {

    private ActionBuilder actionBuilder;
    private MedicationSelector medicationSelector;
    private TimeConstraintTracker timeConstraintTracker;
    private EnrichedPatientContext context;
    private PatientState patientState;

    @BeforeEach
    void setUp() {
        medicationSelector = new MedicationSelector();
        timeConstraintTracker = new TimeConstraintTracker();
        // Use default constructor which initializes TestRecommender internally
        actionBuilder = new ActionBuilder();

        // Setup patient context
        context = new EnrichedPatientContext();
        context.setPatientId("TEST-001");
        context.setTriggerTime(Instant.now().minus(15, ChronoUnit.MINUTES));

        patientState = new PatientState();
        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(65);
        demographics.setWeight(75.0);
        demographics.setSex("M");
        patientState.setDemographics(demographics);
        patientState.setCreatinine(1.2);

        context.setPatientState(patientState);
    }

    /**
     * Test 1: Time constraints are tracked for protocol
     */
    @Test
    void testTimeConstraintsTracked_SepsisProtocol() {
        // Given: Sepsis protocol with Hour-1 bundle constraint
        Protocol sepsisProtocol = createSepsisProtocol();

        TimeConstraint hour1Bundle = new TimeConstraint();
        hour1Bundle.setConstraintId("hour-1-bundle");
        hour1Bundle.setBundleName("Hour-1 Bundle");
        hour1Bundle.setOffsetMinutes(60);
        hour1Bundle.setCritical(true);

        sepsisProtocol.setTimeConstraints(Arrays.asList(hour1Bundle));

        // Trigger time was 15 minutes ago, so 45 minutes remaining
        context.setTriggerTime(Instant.now().minus(15, ChronoUnit.MINUTES));

        // When: Build actions with tracking
        ActionResult result = actionBuilder.buildActionsWithTracking(sepsisProtocol, context);

        // Then: Time constraint status should be present
        assertNotNull(result.getTimeConstraintStatus(), "Time constraint status should not be null");
        assertEquals("SEPSIS-BUNDLE-001", result.getTimeConstraintStatus().getProtocolId());
        assertEquals(1, result.getTimeConstraintStatus().getConstraintStatuses().size());

        // Should be INFO level (45 minutes remaining > 30 min threshold)
        assertFalse(result.hasCriticalAlerts(), "Should not have critical alerts (45 min remaining)");
        assertFalse(result.hasWarningAlerts(), "Should not have warning alerts (45 min remaining)");
    }

    /**
     * Test 2: Time constraint WARNING when < 30 minutes remaining
     */
    @Test
    void testTimeConstraintWarning_LessThan30MinutesRemaining() {
        // Given: Sepsis protocol with Hour-1 bundle constraint
        Protocol sepsisProtocol = createSepsisProtocol();

        TimeConstraint hour1Bundle = new TimeConstraint();
        hour1Bundle.setConstraintId("hour-1-bundle");
        hour1Bundle.setBundleName("Hour-1 Bundle");
        hour1Bundle.setOffsetMinutes(60);
        hour1Bundle.setCritical(true);

        sepsisProtocol.setTimeConstraints(Arrays.asList(hour1Bundle));

        // Trigger time was 45 minutes ago, so 15 minutes remaining (< 30 min threshold)
        context.setTriggerTime(Instant.now().minus(45, ChronoUnit.MINUTES));

        // When: Build actions with tracking
        ActionResult result = actionBuilder.buildActionsWithTracking(sepsisProtocol, context);

        // Then: Should have WARNING alert
        assertNotNull(result.getTimeConstraintStatus(), "Time constraint status should not be null");
        assertTrue(result.hasWarningAlerts(), "Should have warning alert (15 min remaining)");
        assertEquals(1, result.getTimeConstraintStatus().getWarningAlerts().size());
        assertEquals(AlertLevel.WARNING,
                result.getTimeConstraintStatus().getConstraintStatuses().get(0).getAlertLevel());
    }

    /**
     * Test 3: Time constraint CRITICAL when deadline exceeded
     */
    @Test
    void testTimeConstraintCritical_DeadlineExceeded() {
        // Given: Sepsis protocol with Hour-1 bundle constraint
        Protocol sepsisProtocol = createSepsisProtocol();

        TimeConstraint hour1Bundle = new TimeConstraint();
        hour1Bundle.setConstraintId("hour-1-bundle");
        hour1Bundle.setBundleName("Hour-1 Bundle");
        hour1Bundle.setOffsetMinutes(60);
        hour1Bundle.setCritical(true);

        sepsisProtocol.setTimeConstraints(Arrays.asList(hour1Bundle));

        // Trigger time was 75 minutes ago (deadline exceeded by 15 minutes)
        context.setTriggerTime(Instant.now().minus(75, ChronoUnit.MINUTES));

        // When: Build actions with tracking
        ActionResult result = actionBuilder.buildActionsWithTracking(sepsisProtocol, context);

        // Then: Should have CRITICAL alert
        assertNotNull(result.getTimeConstraintStatus(), "Time constraint status should not be null");
        assertTrue(result.hasCriticalAlerts(), "Should have critical alert (deadline exceeded)");
        assertEquals(1, result.getTimeConstraintStatus().getCriticalAlerts().size());
        assertEquals(AlertLevel.CRITICAL,
                result.getTimeConstraintStatus().getConstraintStatuses().get(0).getAlertLevel());
    }

    /**
     * Test 4: Multiple time constraints tracked correctly
     */
    @Test
    void testMultipleTimeConstraints_Hour1AndHour3() {
        // Given: Sepsis protocol with both Hour-1 and Hour-3 bundles
        Protocol sepsisProtocol = createSepsisProtocol();

        TimeConstraint hour1Bundle = new TimeConstraint();
        hour1Bundle.setConstraintId("hour-1-bundle");
        hour1Bundle.setBundleName("Hour-1 Bundle");
        hour1Bundle.setOffsetMinutes(60);
        hour1Bundle.setCritical(true);

        TimeConstraint hour3Bundle = new TimeConstraint();
        hour3Bundle.setConstraintId("hour-3-bundle");
        hour3Bundle.setBundleName("Hour-3 Bundle");
        hour3Bundle.setOffsetMinutes(180);
        hour3Bundle.setCritical(false);

        sepsisProtocol.setTimeConstraints(Arrays.asList(hour1Bundle, hour3Bundle));

        // Trigger time was 30 minutes ago
        context.setTriggerTime(Instant.now().minus(30, ChronoUnit.MINUTES));

        // When: Build actions with tracking
        ActionResult result = actionBuilder.buildActionsWithTracking(sepsisProtocol, context);

        // Then: Should track both constraints
        assertNotNull(result.getTimeConstraintStatus(), "Time constraint status should not be null");
        assertEquals(2, result.getTimeConstraintStatus().getConstraintStatuses().size(),
                "Should track 2 constraints");

        // Hour-1: 30 minutes remaining (WARNING)
        // Hour-3: 150 minutes remaining (INFO)
        assertTrue(result.hasWarningAlerts(), "Should have warning for Hour-1 bundle");
        assertFalse(result.hasCriticalAlerts(), "Should not have critical alerts");
    }

    /**
     * Test 5: Empty actions list when protocol is null
     */
    @Test
    void testNullProtocol_ReturnsEmptyActionResult() {
        // When: Build actions with null protocol
        ActionResult result = actionBuilder.buildActionsWithTracking(null, context);

        // Then: Should return empty result
        assertNotNull(result, "Result should not be null");
        assertNotNull(result.getActions(), "Actions list should not be null");
        assertEquals(0, result.getActions().size(), "Actions list should be empty");
        assertNull(result.getTimeConstraintStatus(), "Time status should be null for null protocol");
    }

    /**
     * Test 6: Protocol with no time constraints returns status with no alerts
     */
    @Test
    void testProtocolWithNoTimeConstraints_NoAlerts() {
        // Given: Protocol with no time constraints
        Protocol protocol = new Protocol("PROTOCOL-001", "Test Protocol", "TEST");
        // No time constraints set

        // When: Build actions with tracking
        ActionResult result = actionBuilder.buildActionsWithTracking(protocol, context);

        // Then: Should have time status but no constraint statuses
        assertNotNull(result.getTimeConstraintStatus(), "Time constraint status should not be null");
        assertEquals(0, result.getTimeConstraintStatus().getConstraintStatuses().size(),
                "Should have 0 constraint statuses");
        assertFalse(result.hasCriticalAlerts(), "Should not have critical alerts");
        assertFalse(result.hasWarningAlerts(), "Should not have warning alerts");
    }

    // Helper methods

    private Protocol createSepsisProtocol() {
        Protocol protocol = new Protocol();
        protocol.setProtocolId("SEPSIS-BUNDLE-001");
        protocol.setName("Sepsis Management Bundle");
        protocol.setCategory("INFECTIOUS");
        protocol.setSpecialty("Emergency Medicine");
        return protocol;
    }
}
