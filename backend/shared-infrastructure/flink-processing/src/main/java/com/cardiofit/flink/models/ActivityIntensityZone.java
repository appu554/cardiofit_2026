package com.cardiofit.flink.models;

/**
 * Heart rate training zone classification based on percentage of
 * age-predicted maximum heart rate (HR_max = 220 - age).
 *
 * Five-zone model aligned with ACSM guidelines and Karvonen formula.
 * Zone boundaries use % of HR_max (not HR reserve) for simplicity
 * in streaming computation where resting HR may be unavailable.
 *
 * Clinical significance:
 * - Zones 1-2: fat oxidation predominant, safe for cardiac rehab patients
 * - Zone 3: lactate threshold transition, maximal steady-state
 * - Zone 4-5: anaerobic, catecholamine-driven glucose spike expected
 */
public enum ActivityIntensityZone {

    ZONE_1_RECOVERY,   // 50–59% HR_max — warm-up, cool-down, very light
    ZONE_2_AEROBIC,    // 60–69% HR_max — base endurance, fat oxidation peak
    ZONE_3_TEMPO,      // 70–79% HR_max — lactate threshold, "comfortably hard"
    ZONE_4_THRESHOLD,  // 80–89% HR_max — anaerobic threshold, glycolytic
    ZONE_5_ANAEROBIC;  // ≥90% HR_max — VO2max effort, catecholamine surge

    /** Default HR_max estimation when age unknown (conservative for safety). */
    public static final double DEFAULT_HR_MAX = 180.0;

    public static ActivityIntensityZone fromHeartRate(double heartRate, double hrMax) {
        if (hrMax <= 0) hrMax = DEFAULT_HR_MAX;
        double pct = heartRate / hrMax;
        if (pct < 0.50) return ZONE_1_RECOVERY;
        if (pct < 0.60) return ZONE_1_RECOVERY;
        if (pct < 0.70) return ZONE_2_AEROBIC;
        if (pct < 0.80) return ZONE_3_TEMPO;
        if (pct < 0.90) return ZONE_4_THRESHOLD;
        return ZONE_5_ANAEROBIC;
    }

    public static double estimateHRMax(int age) {
        if (age <= 0) return DEFAULT_HR_MAX;
        return 208.0 - 0.7 * age;
    }

    public boolean isHighIntensity() {
        return this == ZONE_4_THRESHOLD || this == ZONE_5_ANAEROBIC;
    }

    public boolean isAerobic() {
        return this == ZONE_1_RECOVERY || this == ZONE_2_AEROBIC || this == ZONE_3_TEMPO;
    }
}
