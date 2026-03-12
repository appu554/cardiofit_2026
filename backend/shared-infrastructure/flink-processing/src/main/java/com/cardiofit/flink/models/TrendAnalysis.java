package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * Statistical trend analysis result from linear regression.
 *
 * Provides mathematical characterization of clinical parameter trends over time.
 *
 * Linear Regression Model: y = slope * x + intercept
 * - slope: Rate of change per time unit (e.g., mg/dL per hour)
 * - intercept: Predicted value at time zero
 * - rSquared: Coefficient of determination (0-1), measures fit quality
 * - dataPointCount: Number of observations used in regression
 *
 * R-squared interpretation:
 * - >0.9: Very strong linear trend
 * - 0.7-0.9: Strong linear trend
 * - 0.5-0.7: Moderate linear trend
 * - <0.5: Weak or no linear trend (may indicate variability or non-linear pattern)
 *
 * Clinical applications:
 * - Creatinine trend: Slope >0.3 mg/dL/day suggests AKI progression
 * - Glucose trend: Detect hypo/hyperglycemia patterns
 * - Blood pressure trend: Identify hemodynamic instability
 * - Heart rate trend: Early sepsis detection (increasing trend)
 */
public class TrendAnalysis implements Serializable {
    private static final long serialVersionUID = 1L;

    private double slope;
    private double intercept;
    private double rSquared;
    private int dataPointCount;

    public TrendAnalysis() {}

    public TrendAnalysis(double slope, double intercept, double rSquared, int dataPointCount) {
        this.slope = slope;
        this.intercept = intercept;
        this.rSquared = rSquared;
        this.dataPointCount = dataPointCount;
    }

    // Getters and Setters
    public double getSlope() {
        return slope;
    }

    public void setSlope(double slope) {
        this.slope = slope;
    }

    public double getIntercept() {
        return intercept;
    }

    public void setIntercept(double intercept) {
        this.intercept = intercept;
    }

    public double getRSquared() {
        return rSquared;
    }

    public void setRSquared(double rSquared) {
        this.rSquared = rSquared;
    }

    public int getDataPointCount() {
        return dataPointCount;
    }

    public void setDataPointCount(int dataPointCount) {
        this.dataPointCount = dataPointCount;
    }

    /**
     * Determines if the trend is statistically reliable.
     * Requires sufficient data points and reasonable fit quality.
     */
    public boolean isReliable() {
        return dataPointCount >= 3 && rSquared >= 0.5;
    }

    /**
     * Determines if the trend shows strong linear relationship.
     */
    public boolean isStrongTrend() {
        return rSquared >= 0.7;
    }

    /**
     * Gets a human-readable interpretation of the R-squared value.
     */
    public String getFitQuality() {
        if (rSquared >= 0.9) {
            return "Very strong linear trend";
        } else if (rSquared >= 0.7) {
            return "Strong linear trend";
        } else if (rSquared >= 0.5) {
            return "Moderate linear trend";
        } else {
            return "Weak or no linear trend";
        }
    }

    @Override
    public String toString() {
        return "TrendAnalysis{" +
                "slope=" + slope +
                ", intercept=" + intercept +
                ", rSquared=" + rSquared +
                ", dataPointCount=" + dataPointCount +
                ", fitQuality='" + getFitQuality() + '\'' +
                '}';
    }
}
