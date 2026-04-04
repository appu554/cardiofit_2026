package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.ArrayList;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module11bExerciseDoseCalculatorTest {

    @Test
    void weeklyDose_sumsMetMinutes() {
        List<ActivityResponseRecord> records = new ArrayList<>();
        records.add(activityRecord(ExerciseType.AEROBIC, 6.0, 30.0)); // 180 MET-min
        records.add(activityRecord(ExerciseType.AEROBIC, 6.0, 45.0)); // 270 MET-min
        records.add(activityRecord(ExerciseType.RESISTANCE, 5.0, 40.0)); // 200 MET-min

        Module11bExerciseDoseCalculator.Result result =
                Module11bExerciseDoseCalculator.calculate(records);

        assertEquals(650.0, result.totalMetMinutes, 0.1);
        assertEquals(115.0, result.totalDurationMin, 0.1);
        assertTrue(result.meetsWHOModerate); // 650 >= 150 MET-min
    }

    @Test
    void whoBenchmark_belowMinimum() {
        List<ActivityResponseRecord> records = new ArrayList<>();
        records.add(activityRecord(ExerciseType.FLEXIBILITY, 2.5, 30.0)); // 75 MET-min

        Module11bExerciseDoseCalculator.Result result =
                Module11bExerciseDoseCalculator.calculate(records);

        assertEquals(75.0, result.totalMetMinutes, 0.1);
        assertFalse(result.meetsWHOModerate); // 75 < 150
    }

    @Test
    void emptyRecords_zeroResult() {
        Module11bExerciseDoseCalculator.Result result =
                Module11bExerciseDoseCalculator.calculate(new ArrayList<>());

        assertEquals(0.0, result.totalMetMinutes);
        assertEquals(0, result.activityCount);
        assertFalse(result.meetsWHOModerate);
    }

    @Test
    void perTypeBreakdown_correct() {
        List<ActivityResponseRecord> records = new ArrayList<>();
        records.add(activityRecord(ExerciseType.AEROBIC, 6.0, 30.0));
        records.add(activityRecord(ExerciseType.AEROBIC, 7.0, 45.0));
        records.add(activityRecord(ExerciseType.RESISTANCE, 5.0, 40.0));

        Module11bExerciseDoseCalculator.Result result =
                Module11bExerciseDoseCalculator.calculate(records);

        assertTrue(result.perTypeMetMinutes.containsKey(ExerciseType.AEROBIC));
        assertTrue(result.perTypeMetMinutes.containsKey(ExerciseType.RESISTANCE));
        assertEquals(495.0, result.perTypeMetMinutes.get(ExerciseType.AEROBIC), 0.1); // 180+315
        assertEquals(200.0, result.perTypeMetMinutes.get(ExerciseType.RESISTANCE), 0.1);
    }

    private ActivityResponseRecord activityRecord(ExerciseType type, double mets, double durationMin) {
        return ActivityResponseRecord.builder()
                .recordId("test-" + System.nanoTime())
                .patientId("P1")
                .activityEventId("act-" + System.nanoTime())
                .activityStartTime(System.currentTimeMillis())
                .exerciseType(type)
                .reportedMETs(mets)
                .activityDurationMin(durationMin)
                .metMinutes(mets * durationMin)
                .build();
    }
}
