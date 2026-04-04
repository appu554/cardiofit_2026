package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.List;
import java.util.Map;

/**
 * Heart rate feature extraction for Module 11.
 *
 * Computes from an HRWindow:
 * 1. peakHR — maximum HR during active exercise phase
 * 2. meanActiveHR — mean HR during active phase only (excludes pre/recovery)
 * 3. hrr1 — HR drop from peak at ~1 min post-exercise (HRR₁)
 * 4. hrr2 — HR drop from peak at ~2 min post-exercise (HRR₂)
 * 5. hrRecoveryClass — classification from HRR₁ (Cole et al., NEJM 1999)
 * 6. dominantZone — zone with most time during active phase
 * 7. readingCount — total HR readings in window
 * 8. qualityScore — 0-1 based on reading density and phase coverage
 *
 * HRR algorithm:
 * - Find peak HR during active phase
 * - In recovery readings, find reading closest to 60s post-activity-end → HRR₁
 * - Find reading closest to 120s post-activity-end → HRR₂
 * - If no recovery reading within 30s of target time, mark as null
 *
 * Stateless utility class.
 */
public class Module11HRAnalyzer {

    private static final long HRR1_TARGET_MS = 60_000L;  // 1 minute
    private static final long HRR2_TARGET_MS = 120_000L; // 2 minutes
    private static final long HRR_TOLERANCE_MS = 30_000L; // ±30 seconds

    private Module11HRAnalyzer() {}

    public static Result analyze(HRWindow window) {
        if (window == null || window.isEmpty()) return null;

        window.sortByTime();

        List<HRWindow.HRReading> activeReadings = window.getActivePhaseReadings();
        List<HRWindow.HRReading> recoveryReadings = window.getRecoveryPhaseReadings();

        if (activeReadings.isEmpty()) return null;

        // Peak HR during active phase
        double peakHR = Double.NEGATIVE_INFINITY;
        for (HRWindow.HRReading r : activeReadings) {
            if (r.heartRate > peakHR) {
                peakHR = r.heartRate;
            }
        }

        // Mean active HR
        double sumHR = 0;
        for (HRWindow.HRReading r : activeReadings) {
            sumHR += r.heartRate;
        }
        double meanActiveHR = sumHR / activeReadings.size();

        // HRR₁ and HRR₂ from recovery phase
        Double hrr1 = null;
        Double hrr2 = null;
        HRRecoveryClass hrRecoveryClass = HRRecoveryClass.INSUFFICIENT_DATA;

        if (window.getActivityEndTime() != null && recoveryReadings.size() >= HRRecoveryClass.MIN_RECOVERY_READINGS) {
            long actEnd = window.getActivityEndTime();

            // Find reading closest to 1 min post-exercise
            HRWindow.HRReading closest1Min = findClosestReading(recoveryReadings, actEnd + HRR1_TARGET_MS);
            if (closest1Min != null
                    && Math.abs((closest1Min.timestamp - actEnd) - HRR1_TARGET_MS) <= HRR_TOLERANCE_MS) {
                hrr1 = peakHR - closest1Min.heartRate;
                hrRecoveryClass = HRRecoveryClass.fromHRR1(hrr1);
            }

            // Find reading closest to 2 min post-exercise
            HRWindow.HRReading closest2Min = findClosestReading(recoveryReadings, actEnd + HRR2_TARGET_MS);
            if (closest2Min != null
                    && Math.abs((closest2Min.timestamp - actEnd) - HRR2_TARGET_MS) <= HRR_TOLERANCE_MS) {
                hrr2 = peakHR - closest2Min.heartRate;
            }
        }

        // Dominant zone
        Map<ActivityIntensityZone, Long> zoneDistribution = window.computeZoneDistribution();
        ActivityIntensityZone dominantZone = ActivityIntensityZone.ZONE_1_RECOVERY;
        long maxTime = 0;
        for (Map.Entry<ActivityIntensityZone, Long> entry : zoneDistribution.entrySet()) {
            if (entry.getValue() > maxTime) {
                maxTime = entry.getValue();
                dominantZone = entry.getKey();
            }
        }

        // Quality score
        long activeDurationMs = window.getActivityEndTime() != null
                ? window.getActivityEndTime() - window.getActivityStartTime()
                : 0;
        double expectedReadings = Math.max(1, activeDurationMs / 60_000.0);
        double hrQuality = Math.min(1.0, activeReadings.size() / expectedReadings);
        double recoveryQuality = recoveryReadings.size() >= 2 ? 0.3 : 0.0;
        double qualityScore = Math.min(1.0, hrQuality * 0.7 + recoveryQuality);

        Result r = new Result();
        r.peakHR = peakHR;
        r.meanActiveHR = meanActiveHR;
        r.hrr1 = hrr1;
        r.hrr2 = hrr2;
        r.hrRecoveryClass = hrRecoveryClass;
        r.dominantZone = dominantZone;
        r.zoneDistribution = zoneDistribution;
        r.readingCount = window.size();
        r.qualityScore = qualityScore;
        return r;
    }

    /**
     * Compute Rate-Pressure Product from peak HR and peak SBP.
     * RPP = HR × SBP. Normal resting ~7,000; peak exercise ~25,000–40,000.
     */
    public static Double computeRPP(Double peakHR, Double peakSBP) {
        if (peakHR == null || peakSBP == null) return null;
        return peakHR * peakSBP;
    }

    private static HRWindow.HRReading findClosestReading(List<HRWindow.HRReading> readings, long targetTime) {
        HRWindow.HRReading closest = null;
        long minDiff = Long.MAX_VALUE;
        for (HRWindow.HRReading r : readings) {
            long diff = Math.abs(r.timestamp - targetTime);
            if (diff < minDiff) {
                minDiff = diff;
                closest = r;
            }
        }
        return closest;
    }

    public static class Result {
        public double peakHR;
        public double meanActiveHR;
        public Double hrr1;
        public Double hrr2;
        public HRRecoveryClass hrRecoveryClass;
        public ActivityIntensityZone dominantZone;
        public Map<ActivityIntensityZone, Long> zoneDistribution;
        public int readingCount;
        public double qualityScore;
    }
}
