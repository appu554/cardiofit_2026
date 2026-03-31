package com.cardiofit.flink.models;

/**
 * Alert lifecycle state machine.
 *
 * ACTIVE → ACKNOWLEDGED → ACTIONED → RESOLVED
 * ACTIVE → AUTO_RESOLVED
 * ACTIVE → ESCALATED → ACTIONED → RESOLVED
 */
public enum AlertState {
    ACTIVE,
    ACKNOWLEDGED,
    ACTIONED,
    AUTO_RESOLVED,
    ESCALATED,
    RESOLVED;

    public boolean isTerminal() {
        return this == AUTO_RESOLVED || this == RESOLVED;
    }

    public boolean canTransitionTo(AlertState next) {
        return switch (this) {
            case ACTIVE -> next == ACKNOWLEDGED || next == AUTO_RESOLVED || next == ESCALATED;
            case ACKNOWLEDGED -> next == ACTIONED || next == RESOLVED;
            case ESCALATED -> next == ACTIONED || next == ACKNOWLEDGED || next == RESOLVED;
            case ACTIONED -> next == RESOLVED;
            case AUTO_RESOLVED, RESOLVED -> false;
        };
    }
}
