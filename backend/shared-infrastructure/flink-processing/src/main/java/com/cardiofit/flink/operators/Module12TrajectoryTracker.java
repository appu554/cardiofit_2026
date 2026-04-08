package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.ArrayList;
import java.util.List;

public final class Module12TrajectoryTracker {

    private static final int MIN_READINGS = 3;
    private static final long MIN_SPAN_MS = 7L * 86_400_000L;
    private static final long WEEK_MS = 7L * 86_400_000L;
    private static final long YEAR_MS = 365L * 86_400_000L;

    private Module12TrajectoryTracker() {}

    public static TrajectoryClassification classify(
            List<InterventionWindowState.TrajectoryDataPoint> readings,
            String domain, long sinceMs, long untilMs) {

        List<InterventionWindowState.TrajectoryDataPoint> filtered = new ArrayList<>();
        for (InterventionWindowState.TrajectoryDataPoint p : readings) {
            if (domain.equals(p.domain) && p.timestamp >= sinceMs && p.timestamp <= untilMs) {
                filtered.add(p);
            }
        }

        if (filtered.size() < MIN_READINGS) {
            return TrajectoryClassification.UNKNOWN;
        }

        long span = filtered.get(filtered.size() - 1).timestamp - filtered.get(0).timestamp;
        if (span < MIN_SPAN_MS) {
            return TrajectoryClassification.UNKNOWN;
        }

        double slopePerMs = computeOLSSlope(filtered);
        return classifySlope(domain, slopePerMs);
    }

    public static TrajectoryClassification classifyComposite(
            InterventionWindowState state, long sinceMs, long untilMs) {

        List<InterventionWindowState.TrajectoryDataPoint> allReadings = state.getRecentReadings();

        TrajectoryClassification fbg = classify(allReadings, "FBG", sinceMs, untilMs);
        TrajectoryClassification sbp = classify(allReadings, "SBP", sinceMs, untilMs);

        if (fbg == TrajectoryClassification.DETERIORATING
                || sbp == TrajectoryClassification.DETERIORATING) {
            return TrajectoryClassification.DETERIORATING;
        }

        TrajectoryClassification egfr = classify(allReadings, "EGFR", sinceMs, untilMs);
        TrajectoryClassification weight = classify(allReadings, "WEIGHT", sinceMs, untilMs);

        if (egfr == TrajectoryClassification.DETERIORATING
                || weight == TrajectoryClassification.DETERIORATING) {
            return TrajectoryClassification.DETERIORATING;
        }

        boolean anyKnown = false;
        boolean allImproving = true;
        for (TrajectoryClassification tc : new TrajectoryClassification[]{fbg, sbp, egfr, weight}) {
            if (tc != TrajectoryClassification.UNKNOWN) {
                anyKnown = true;
                if (tc != TrajectoryClassification.IMPROVING) {
                    allImproving = false;
                }
            }
        }

        if (anyKnown && allImproving) {
            return TrajectoryClassification.IMPROVING;
        }

        return anyKnown ? TrajectoryClassification.STABLE : TrajectoryClassification.UNKNOWN;
    }

    static double computeOLSSlope(List<InterventionWindowState.TrajectoryDataPoint> points) {
        int n = points.size();
        double sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;
        for (InterventionWindowState.TrajectoryDataPoint p : points) {
            double x = p.timestamp;
            double y = p.value;
            sumX += x;
            sumY += y;
            sumXY += x * y;
            sumX2 += x * x;
        }
        double denominator = n * sumX2 - sumX * sumX;
        if (denominator == 0) return 0;
        return (n * sumXY - sumX * sumY) / denominator;
    }

    private static TrajectoryClassification classifySlope(String domain, double slopePerMs) {
        switch (domain) {
            case "FBG": {
                double slopePerWeek = slopePerMs * WEEK_MS;
                if (slopePerWeek < -3.0) return TrajectoryClassification.IMPROVING;
                if (slopePerWeek > 3.0) return TrajectoryClassification.DETERIORATING;
                return TrajectoryClassification.STABLE;
            }
            case "SBP": {
                double slopePerWeek = slopePerMs * WEEK_MS;
                if (slopePerWeek < -2.0) return TrajectoryClassification.IMPROVING;
                if (slopePerWeek > 2.0) return TrajectoryClassification.DETERIORATING;
                return TrajectoryClassification.STABLE;
            }
            case "EGFR": {
                double slopePerYear = slopePerMs * YEAR_MS;
                if (slopePerYear > 1.0) return TrajectoryClassification.IMPROVING;
                if (slopePerYear < -1.0) return TrajectoryClassification.DETERIORATING;
                return TrajectoryClassification.STABLE;
            }
            case "WEIGHT": {
                double slopePerWeek = slopePerMs * WEEK_MS;
                if (slopePerWeek < -0.3) return TrajectoryClassification.IMPROVING;
                if (slopePerWeek > 0.3) return TrajectoryClassification.DETERIORATING;
                return TrajectoryClassification.STABLE;
            }
            default:
                return TrajectoryClassification.UNKNOWN;
        }
    }
}
