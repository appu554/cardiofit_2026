package com.cardiofit.flink.models.protocol;

/**
 * Match logic for combining multiple conditions.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public enum MatchLogic {
    /**
     * ALL_OF - AND logic, all conditions must be true
     */
    ALL_OF,

    /**
     * ANY_OF - OR logic, at least one condition must be true
     */
    ANY_OF
}
