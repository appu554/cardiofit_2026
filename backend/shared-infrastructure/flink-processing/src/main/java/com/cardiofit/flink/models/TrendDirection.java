package com.cardiofit.flink.models;

/**
 * Clinical interpretation of trend direction and status ranges.
 *
 * This enum provides standardized categorization for TWO use cases:
 *
 * MODULE 4 - Trend Analysis (Time-Series):
 * Categorizes clinical parameter trends derived from linear regression analysis.
 * Slope thresholds:
 * - STABLE: |slope| < 0.01 (minimal change)
 * - RAPIDLY_INCREASING: slope > 0.1 (significant positive trend)
 * - INCREASING: 0 < slope ≤ 0.1 (moderate positive trend)
 * - RAPIDLY_DECREASING: slope < -0.1 (significant negative trend)
 * - DECREASING: -0.1 ≤ slope < 0 (moderate negative trend)
 *
 * MODULE 2 - Clinical Range Categorization (Point-in-Time):
 * Categorizes vital signs and lab values relative to normal physiological ranges.
 * - NORMAL: Within healthy reference range
 * - ELEVATED: Above normal but not critical
 * - LOW: Below normal but not critical
 * - CRITICALLY_LOW: Dangerously low (e.g., SpO2 <85%)
 * - BORDERLINE: Near threshold between normal/abnormal
 * - HYPOTHERMIA: Temperature <35°C
 * - FEVER: Temperature >38°C
 * - UNKNOWN: Insufficient data or not applicable
 *
 * Note: Thresholds may need adjustment based on clinical parameter type,
 * time window duration, and patient-specific baselines.
 */
public enum TrendDirection {
    // ===== MODULE 4: Time-Series Trend Analysis =====
    STABLE,
    INCREASING,
    RAPIDLY_INCREASING,
    DECREASING,
    RAPIDLY_DECREASING,

    // ===== MODULE 2: Clinical Range Categorization =====
    UNKNOWN,
    NORMAL,
    ELEVATED,
    LOW,
    CRITICALLY_LOW,
    BORDERLINE,
    HYPOTHERMIA,
    FEVER;

    /**
     * Determines trend direction from linear regression slope.
     *
     * @param slope Rate of change from linear regression (units per time)
     * @return TrendDirection enum value based on slope magnitude and sign
     */
    public static TrendDirection fromSlope(double slope) {
        if (Math.abs(slope) < 0.01) {
            return STABLE;
        } else if (slope > 0.1) {
            return RAPIDLY_INCREASING;
        } else if (slope > 0) {
            return INCREASING;
        } else if (slope < -0.1) {
            return RAPIDLY_DECREASING;
        } else {
            return DECREASING;
        }
    }

    /**
     * Gets human-readable description of the trend direction or clinical status.
     *
     * @return Clinical interpretation of the trend or status
     */
    public String getDescription() {
        switch (this) {
            // Module 4 trends
            case STABLE:
                return "No significant trend";
            case INCREASING:
                return "Gradual increase";
            case RAPIDLY_INCREASING:
                return "Rapid increase requiring attention";
            case DECREASING:
                return "Gradual decrease";
            case RAPIDLY_DECREASING:
                return "Rapid decrease requiring attention";

            // Module 2 clinical ranges
            case UNKNOWN:
                return "Status unknown";
            case NORMAL:
                return "Within normal range";
            case ELEVATED:
                return "Above normal range";
            case LOW:
                return "Below normal range";
            case CRITICALLY_LOW:
                return "Critically low - immediate attention required";
            case BORDERLINE:
                return "Borderline abnormal";
            case HYPOTHERMIA:
                return "Hypothermia detected";
            case FEVER:
                return "Fever detected";

            default:
                return "Unknown";
        }
    }

    /**
     * Determines if the trend requires clinical attention (Module 4 usage).
     * Rapid changes in either direction warrant closer monitoring.
     *
     * @return true if rapid change detected, false otherwise
     */
    public boolean requiresAttention() {
        return this == RAPIDLY_INCREASING || this == RAPIDLY_DECREASING;
    }

    /**
     * Determines if the trend indicates worsening condition (Module 4 usage).
     * Note: Clinical significance is context-dependent. For example:
     * - INCREASING creatinine suggests kidney function decline
     * - DECREASING hemoglobin suggests bleeding or anemia
     * This method should be used in conjunction with parameter-specific logic.
     *
     * @return true if showing significant change, false if stable
     */
    public boolean showsChange() {
        return this != STABLE && this != UNKNOWN;
    }

    /**
     * Determines if clinical value is abnormal (Module 2 usage).
     *
     * @return true if value is outside normal range
     */
    public boolean isAbnormal() {
        return this == ELEVATED || this == LOW || this == CRITICALLY_LOW ||
               this == HYPOTHERMIA || this == FEVER;
    }

    /**
     * Determines if clinical value is critical (Module 2 usage).
     *
     * @return true if value requires immediate attention
     */
    public boolean isCritical() {
        return this == CRITICALLY_LOW || this == HYPOTHERMIA || this == FEVER;
    }
}
