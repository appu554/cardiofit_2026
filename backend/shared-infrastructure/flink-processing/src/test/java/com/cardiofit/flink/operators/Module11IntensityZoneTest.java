package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module11IntensityZoneTest {

    @Test
    void zoneClassification_allZones() {
        double hrMax = 180.0; // age ~40

        assertEquals(ActivityIntensityZone.ZONE_1_RECOVERY,
                ActivityIntensityZone.fromHeartRate(95, hrMax));  // 53%
        assertEquals(ActivityIntensityZone.ZONE_2_AEROBIC,
                ActivityIntensityZone.fromHeartRate(115, hrMax)); // 64%
        assertEquals(ActivityIntensityZone.ZONE_3_TEMPO,
                ActivityIntensityZone.fromHeartRate(135, hrMax)); // 75%
        assertEquals(ActivityIntensityZone.ZONE_4_THRESHOLD,
                ActivityIntensityZone.fromHeartRate(155, hrMax)); // 86%
        assertEquals(ActivityIntensityZone.ZONE_5_ANAEROBIC,
                ActivityIntensityZone.fromHeartRate(170, hrMax)); // 94%
    }

    @Test
    void tanakaFormula_correctForDifferentAges() {
        // Age 20: 208 - 0.7*20 = 194
        assertEquals(194.0, ActivityIntensityZone.estimateHRMax(20), 0.1);
        // Age 40: 208 - 0.7*40 = 180
        assertEquals(180.0, ActivityIntensityZone.estimateHRMax(40), 0.1);
        // Age 60: 208 - 0.7*60 = 166
        assertEquals(166.0, ActivityIntensityZone.estimateHRMax(60), 0.1);
    }

    @Test
    void isHighIntensity_zone4and5() {
        assertTrue(ActivityIntensityZone.ZONE_4_THRESHOLD.isHighIntensity());
        assertTrue(ActivityIntensityZone.ZONE_5_ANAEROBIC.isHighIntensity());
        assertFalse(ActivityIntensityZone.ZONE_3_TEMPO.isHighIntensity());
        assertFalse(ActivityIntensityZone.ZONE_1_RECOVERY.isHighIntensity());
    }

    @Test
    void exerciseType_fromString_variants() {
        assertEquals(ExerciseType.AEROBIC, ExerciseType.fromString("running"));
        assertEquals(ExerciseType.AEROBIC, ExerciseType.fromString("CYCLING"));
        assertEquals(ExerciseType.RESISTANCE, ExerciseType.fromString("weights"));
        assertEquals(ExerciseType.HIIT, ExerciseType.fromString("interval"));
        assertEquals(ExerciseType.FLEXIBILITY, ExerciseType.fromString("yoga"));
        assertEquals(ExerciseType.MIXED, ExerciseType.fromString(null));
        assertEquals(ExerciseType.MIXED, ExerciseType.fromString("unknown_activity"));
    }

    @Test
    void fitnessLevel_fromVO2max() {
        assertEquals(FitnessLevel.EXCELLENT, FitnessLevel.fromVO2max(48.0));
        assertEquals(FitnessLevel.GOOD, FitnessLevel.fromVO2max(38.0));
        assertEquals(FitnessLevel.AVERAGE, FitnessLevel.fromVO2max(28.0));
        assertEquals(FitnessLevel.BELOW_AVERAGE, FitnessLevel.fromVO2max(20.0));
        assertEquals(FitnessLevel.POOR, FitnessLevel.fromVO2max(15.0));
    }

    @Test
    void hrRecoveryClass_fromHRR1() {
        assertEquals(HRRecoveryClass.EXCELLENT, HRRecoveryClass.fromHRR1(30.0));
        assertEquals(HRRecoveryClass.NORMAL, HRRecoveryClass.fromHRR1(20.0));
        assertEquals(HRRecoveryClass.BLUNTED, HRRecoveryClass.fromHRR1(14.0));
        assertEquals(HRRecoveryClass.ABNORMAL, HRRecoveryClass.fromHRR1(8.0));
    }

    @Test
    void exerciseBPResponse_classifications() {
        assertEquals(ExerciseBPResponse.NORMAL,
                ExerciseBPResponse.classify(120.0, 160.0, 122.0));
        assertEquals(ExerciseBPResponse.EXAGGERATED,
                ExerciseBPResponse.classify(120.0, 215.0, 130.0));
        assertEquals(ExerciseBPResponse.POST_EXERCISE_HYPOTENSION,
                ExerciseBPResponse.classify(140.0, 170.0, 130.0));
        assertEquals(ExerciseBPResponse.HYPOTENSIVE_RESPONSE,
                ExerciseBPResponse.classify(140.0, 170.0, 115.0));
        assertEquals(ExerciseBPResponse.INCOMPLETE,
                ExerciseBPResponse.classify(null, 160.0, 130.0));
    }

    @Test
    void exerciseBPResponse_sexAwareThresholds() {
        // 200 mmHg peak: NORMAL for male (threshold 210), EXAGGERATED for female (threshold 190)
        assertEquals(ExerciseBPResponse.NORMAL,
                ExerciseBPResponse.classify(150.0, 200.0, 145.0, "M"));
        assertEquals(ExerciseBPResponse.EXAGGERATED,
                ExerciseBPResponse.classify(150.0, 200.0, 145.0, "F"));
        // Unknown sex uses conservative (female) threshold
        assertEquals(ExerciseBPResponse.EXAGGERATED,
                ExerciseBPResponse.classify(150.0, 200.0, 145.0, null));
    }
}
