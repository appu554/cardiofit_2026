package com.cardiofit.flink.models;

/**
 * Deployment channels with channel-specific engagement thresholds.
 * DD#8 Section 4.1: engagement expectations must be calibrated to patient context.
 * A Government patient at 0.35 is GREEN (engaging within structural constraints),
 * not ORANGE (which would fire a false-positive alert).
 */
public enum EngagementChannel {
    CORPORATE(0.70, 0.40, 0.20),
    GOVERNMENT(0.40, 0.25, 0.15),
    ACCHS(0.35, 0.20, 0.10);

    private final double greenThreshold;
    private final double yellowThreshold;
    private final double orangeThreshold;

    EngagementChannel(double green, double yellow, double orange) {
        this.greenThreshold = green;
        this.yellowThreshold = yellow;
        this.orangeThreshold = orange;
    }

    public double getGreenThreshold() { return greenThreshold; }
    public double getYellowThreshold() { return yellowThreshold; }
    public double getOrangeThreshold() { return orangeThreshold; }

    public static EngagementChannel fromString(String channel) {
        if (channel == null) return CORPORATE;
        switch (channel.toUpperCase()) {
            case "GOVERNMENT": case "GOV": return GOVERNMENT;
            case "ACCHS": case "COMMUNITY": return ACCHS;
            default: return CORPORATE;
        }
    }
}
