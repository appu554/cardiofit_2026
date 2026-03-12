package com.cardiofit.flink.models.protocol;

/**
 * Comparison operators for condition evaluation.
 *
 * Supports numeric comparisons and string operations.
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
public enum ComparisonOperator {
    /**
     * Greater than or equal (>=)
     */
    GREATER_THAN_OR_EQUAL(">="),

    /**
     * Less than or equal (<=)
     */
    LESS_THAN_OR_EQUAL("<="),

    /**
     * Greater than (>)
     */
    GREATER_THAN(">"),

    /**
     * Less than (<)
     */
    LESS_THAN("<"),

    /**
     * Equal (==)
     */
    EQUAL("=="),

    /**
     * Not equal (!=)
     */
    NOT_EQUAL("!="),

    /**
     * Contains substring (case-insensitive)
     */
    CONTAINS("CONTAINS"),

    /**
     * Does not contain substring (case-insensitive)
     */
    NOT_CONTAINS("NOT_CONTAINS");

    private final String symbol;

    ComparisonOperator(String symbol) {
        this.symbol = symbol;
    }

    public String getSymbol() {
        return symbol;
    }

    @Override
    public String toString() {
        return symbol;
    }
}
