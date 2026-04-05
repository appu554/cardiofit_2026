package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.Optional;

/**
 * Detects engagement drops that warrant alerts.
 * Three detection modes:
 * 1. Level transition downward — WITH 3-DAY PERSISTENCE (R3 Fix, DD#8 Section 4.1)
 * 2. Sustained low (ORANGE for >= 5 consecutive days)
 * 3. Cliff drop (score delta > 0.30 in one day — fires immediately)
 *
 * Exception: Transitions directly to RED fire immediately (patient safety).
 */
public final class Module9DropDetector {

    private static final int SUSTAINED_LOW_THRESHOLD_DAYS = 5;
    private static final double CLIFF_DROP_THRESHOLD = 0.30;
    private static final int LEVEL_TRANSITION_PERSISTENCE_DAYS = 3;

    private Module9DropDetector() {}

    /**
     * Check if an engagement drop alert should be emitted.
     *
     * @param consecutiveDaysAtLevel How many days the patient has been at currentLevel
     */
    public static Optional<EngagementDropAlert> detect(
            String patientId,
            double currentScore,
            EngagementLevel currentLevel,
            Double previousScore,
            EngagementLevel previousLevel,
            int consecutiveLowDays,
            int consecutiveDaysAtLevel,
            EngagementState state,
            long currentTime) {

        // 1. Cliff drop detection (highest priority — fires immediately)
        if (previousScore != null) {
            double delta = previousScore - currentScore;
            if (delta >= CLIFF_DROP_THRESHOLD - 1e-9) {
                String key = EngagementDropAlert.DropType.CLIFF_DROP.name() + ":" + patientId;
                if (!state.isAlertSuppressed(key, currentTime)) {
                    return Optional.of(EngagementDropAlert.create(
                        patientId, EngagementDropAlert.DropType.CLIFF_DROP,
                        currentScore, previousScore, currentLevel, previousLevel,
                        consecutiveLowDays));
                }
            }
        }

        // 2. Level transition downward — WITH 3-DAY PERSISTENCE (R3 Fix)
        if (previousLevel != null && currentLevel.isAlertWorthy()) {
            boolean transitionedDown =
                (previousLevel == EngagementLevel.YELLOW && currentLevel == EngagementLevel.ORANGE) ||
                (previousLevel == EngagementLevel.ORANGE && currentLevel == EngagementLevel.RED) ||
                (previousLevel == EngagementLevel.YELLOW && currentLevel == EngagementLevel.RED) ||
                (previousLevel == EngagementLevel.GREEN && currentLevel == EngagementLevel.ORANGE) ||
                (previousLevel == EngagementLevel.GREEN && currentLevel == EngagementLevel.RED);

            if (transitionedDown) {
                boolean fireImmediately = (currentLevel == EngagementLevel.RED);

                boolean persistenceMet = fireImmediately
                    || consecutiveDaysAtLevel >= LEVEL_TRANSITION_PERSISTENCE_DAYS;

                if (persistenceMet) {
                    String key = EngagementDropAlert.DropType.LEVEL_TRANSITION.name()
                                 + ":" + patientId;
                    if (!state.isAlertSuppressed(key, currentTime)) {
                        return Optional.of(EngagementDropAlert.create(
                            patientId, EngagementDropAlert.DropType.LEVEL_TRANSITION,
                            currentScore, previousScore, currentLevel, previousLevel,
                            consecutiveLowDays));
                    }
                }
            }
        }

        // 3. Sustained low (ORANGE or RED for >= 5 consecutive days)
        if (currentLevel.isAlertWorthy()
            && consecutiveLowDays >= SUSTAINED_LOW_THRESHOLD_DAYS) {
            String key = EngagementDropAlert.DropType.SUSTAINED_LOW.name() + ":" + patientId;
            if (!state.isAlertSuppressed(key, currentTime)) {
                return Optional.of(EngagementDropAlert.create(
                    patientId, EngagementDropAlert.DropType.SUSTAINED_LOW,
                    currentScore, previousScore, currentLevel, previousLevel,
                    consecutiveLowDays));
            }
        }

        return Optional.empty();
    }
}
