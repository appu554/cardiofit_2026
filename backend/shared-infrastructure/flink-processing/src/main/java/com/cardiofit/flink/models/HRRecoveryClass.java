package com.cardiofit.flink.models;

public enum HRRecoveryClass {

    EXCELLENT,
    NORMAL,
    BLUNTED,
    ABNORMAL,
    INSUFFICIENT_DATA;

    public static final int MIN_RECOVERY_READINGS = 2;
    public static final long RECOVERY_WINDOW_MS = 2L * 60_000L;

    public static HRRecoveryClass fromHRR1(double hrr1) {
        if (hrr1 >= 25.0) return EXCELLENT;
        if (hrr1 >= 18.0) return NORMAL;
        if (hrr1 >= 12.0) return BLUNTED;
        return ABNORMAL;
    }

    public boolean isPrognosticFlag() {
        return this == ABNORMAL;
    }
}
