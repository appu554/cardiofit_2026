package com.cardiofit.flink.cds.time;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.protocol.models.Protocol;
import com.cardiofit.flink.protocol.models.TimeConstraint;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.*;
import java.time.temporal.ChronoUnit;
import java.util.*;

/**
 * Tracks time constraints for clinical bundles and generates deadline alerts.
 *
 * <p>Handles:
 * - Hour-0/1/3 bundle tracking (e.g., sepsis bundles)
 * - Deadline calculation: trigger time + offset_minutes
 * - Alert generation: WARNING (<30min remaining), CRITICAL (deadline exceeded)
 * - Bundle compliance monitoring
 *
 * <p>Critical for time-sensitive protocols like sepsis, STEMI, stroke.
 *
 * <p>Example usage:
 * <pre>
 * TimeConstraintTracker tracker = new TimeConstraintTracker();
 * TimeConstraintStatus status = tracker.evaluateConstraints(protocol, context);
 * if (status.hasCriticalAlerts()) {
 *     // Handle critical time constraint violations
 * }
 * </pre>
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public class TimeConstraintTracker {
    private static final Logger logger = LoggerFactory.getLogger(TimeConstraintTracker.class);
    private static final int WARNING_THRESHOLD_MINUTES = 30;

    /**
     * Evaluates all time constraints for a protocol.
     *
     * @param protocol The protocol with time constraints
     * @param context The patient context (includes trigger time)
     * @return Status of all time constraints with alerts
     * @throws IllegalArgumentException if protocol or context is null
     */
    public TimeConstraintStatus evaluateConstraints(
        Protocol protocol,
        EnrichedPatientContext context) {

        if (protocol == null) {
            throw new IllegalArgumentException("Protocol cannot be null");
        }
        if (context == null) {
            throw new IllegalArgumentException("Patient context cannot be null");
        }

        TimeConstraintStatus status = new TimeConstraintStatus();
        status.setProtocolId(protocol.getProtocolId());

        Instant triggerTime = context.getTriggerTime();
        Instant currentTime = Instant.now();

        if (triggerTime == null) {
            logger.warn("No trigger time in context for protocol {}, using current time",
                protocol.getProtocolId());
            triggerTime = currentTime;
            context.setTriggerTime(triggerTime);
        }

        logger.debug("Evaluating time constraints for protocol {}: trigger={}, current={}",
            protocol.getProtocolId(), triggerTime, currentTime);

        List<TimeConstraint> constraints = protocol.getTimeConstraints();
        if (constraints == null || constraints.isEmpty()) {
            logger.debug("No time constraints for protocol {}", protocol.getProtocolId());
            return status;
        }

        for (TimeConstraint constraint : constraints) {
            ConstraintStatus constraintStatus = evaluateConstraint(
                constraint,
                triggerTime,
                currentTime);

            status.addConstraintStatus(constraintStatus);

            // Log critical alerts
            if (constraint.isCritical() && constraintStatus.getAlertLevel() == AlertLevel.CRITICAL) {
                logger.error("CRITICAL TIME CONSTRAINT EXCEEDED: {} - {}",
                    constraint.getBundleName(),
                    constraintStatus.getMessage());
            } else if (constraintStatus.getAlertLevel() == AlertLevel.WARNING) {
                logger.warn("TIME CONSTRAINT WARNING: {} - {}",
                    constraint.getBundleName(),
                    constraintStatus.getMessage());
            } else {
                logger.debug("Time constraint on track: {} - {}",
                    constraint.getBundleName(),
                    constraintStatus.getMessage());
            }
        }

        logger.info("Time constraint evaluation complete for protocol {}: {} constraints, {} critical alerts, {} warnings",
            protocol.getProtocolId(),
            status.getConstraintStatuses().size(),
            status.getCriticalAlerts().size(),
            status.getWarningAlerts().size());

        return status;
    }

    /**
     * Evaluates a single time constraint.
     *
     * @param constraint The time constraint
     * @param triggerTime When protocol was triggered
     * @param currentTime Current time
     * @return Status with alert level and message
     */
    private ConstraintStatus evaluateConstraint(
        TimeConstraint constraint,
        Instant triggerTime,
        Instant currentTime) {

        ConstraintStatus status = new ConstraintStatus();
        status.setConstraintId(constraint.getConstraintId());
        status.setBundleName(constraint.getBundleName());
        status.setCritical(constraint.isCritical());

        // Calculate deadline: trigger + offset_minutes
        Instant deadline = triggerTime.plus(constraint.getOffsetMinutes(), ChronoUnit.MINUTES);
        status.setDeadline(deadline);

        // Calculate time remaining
        Duration timeRemaining = Duration.between(currentTime, deadline);
        status.setTimeRemaining(timeRemaining);

        // Determine alert level
        AlertLevel alertLevel = determineAlertLevel(timeRemaining, constraint.isCritical());
        status.setAlertLevel(alertLevel);

        // Generate human-readable message
        String message = generateMessage(
            constraint.getBundleName(),
            timeRemaining,
            alertLevel);
        status.setMessage(message);

        logger.debug("Constraint {}: deadline={}, remaining={}, alert={}",
            constraint.getConstraintId(),
            deadline,
            formatDuration(timeRemaining),
            alertLevel);

        return status;
    }

    /**
     * Determines alert level based on time remaining.
     *
     * <p>Alert logic:
     * - CRITICAL: timeRemaining < 0 (deadline exceeded)
     * - WARNING: 0 ≤ timeRemaining ≤ 30 minutes
     * - INFO: timeRemaining > 30 minutes
     *
     * @param timeRemaining Duration until deadline
     * @param isCritical Whether this constraint is critical
     * @return AlertLevel (INFO, WARNING, CRITICAL)
     */
    private AlertLevel determineAlertLevel(Duration timeRemaining, boolean isCritical) {
        long minutesRemaining = timeRemaining.toMinutes();

        if (minutesRemaining < 0) {
            // Deadline exceeded - always CRITICAL
            return AlertLevel.CRITICAL;

        } else if (minutesRemaining <= WARNING_THRESHOLD_MINUTES) {
            // Within warning threshold (≤30 minutes)
            // If constraint is critical, issue WARNING, otherwise INFO
            return isCritical ? AlertLevel.WARNING : AlertLevel.INFO;

        } else {
            // On track (>30 minutes remaining)
            return AlertLevel.INFO;
        }
    }

    /**
     * Generates human-readable message for time constraint status.
     *
     * @param bundleName Name of the bundle (e.g., "Hour-1 Bundle")
     * @param timeRemaining Duration until deadline
     * @param alertLevel Current alert level
     * @return Formatted message string
     */
    private String generateMessage(String bundleName, Duration timeRemaining, AlertLevel alertLevel) {
        long minutes = Math.abs(timeRemaining.toMinutes());

        switch (alertLevel) {
            case CRITICAL:
                return String.format("%s deadline exceeded by %s",
                    bundleName, formatDuration(timeRemaining));

            case WARNING:
                return String.format("%s deadline in %s",
                    bundleName, formatDuration(timeRemaining));

            case INFO:
                return String.format("%s: %s remaining",
                    bundleName, formatDuration(timeRemaining));

            default:
                return String.format("%s status: %s",
                    bundleName, alertLevel);
        }
    }

    /**
     * Formats duration as human-readable string.
     *
     * <p>Format:
     * - "Xh Ym" if hours > 0
     * - "Xm" if only minutes
     *
     * @param duration Duration to format
     * @return Formatted string (e.g., "2h 15m", "45m")
     */
    private String formatDuration(Duration duration) {
        // Round to nearest minute for consistency with getMinutesRemaining()
        long seconds = Math.abs(duration.getSeconds());
        long minutes = (seconds + 30) / 60;  // Add 30 seconds for rounding
        long hours = minutes / 60;
        long mins = minutes % 60;

        if (hours > 0) {
            return String.format("%dh %dm", hours, mins);
        } else {
            return String.format("%dm", mins);
        }
    }
}
