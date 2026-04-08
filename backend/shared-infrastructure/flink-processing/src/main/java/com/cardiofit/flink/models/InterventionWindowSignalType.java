package com.cardiofit.flink.models;

import java.io.Serializable;

public enum InterventionWindowSignalType implements Serializable {
    WINDOW_OPENED,
    WINDOW_MIDPOINT,
    WINDOW_CLOSED,
    /**
     * WINDOW_EXPIRED is NOT emitted by the streaming Module 12 KPF.
     * It is set by the batch IOR (Intervention Outcome Report) generator
     * when a closed window has insufficient data completeness for attribution.
     * Downstream consumers of the signal topic will only see OPENED, MIDPOINT,
     * CLOSED, and CANCELLED; EXPIRED is a batch-computed post-hoc status.
     */
    WINDOW_EXPIRED,
    WINDOW_CANCELLED
}
