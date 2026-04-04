package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.ArrayList;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module11bVO2maxEstimatorTest {

    @Test
    void estimateFromSubmaximalHR_healthyAdult() {
        // Age 40, resting HR 65, peak exercise HR 155, HR_max = 208-0.7*40 = 180
        Module11bVO2maxEstimator.Result result =
                Module11bVO2maxEstimator.estimate(155.0, 65.0, 180.0);
        assertNotNull(result);
        assertTrue(result.vo2max > 30.0 && result.vo2max < 55.0);
        assertNotNull(result.fitnessLevel);
    }

    @Test
    void estimateFromSubmaximalHR_highFitness() {
        // Athlete: resting HR 50, peak HR 175, HR_max 190
        Module11bVO2maxEstimator.Result result =
                Module11bVO2maxEstimator.estimate(175.0, 50.0, 190.0);
        assertTrue(result.vo2max > 40.0);
        assertTrue(result.fitnessLevel == FitnessLevel.GOOD
                || result.fitnessLevel == FitnessLevel.EXCELLENT);
    }

    @Test
    void estimateFromSubmaximalHR_deconditioned() {
        // Sedentary: resting HR 85, peak HR 140, HR_max 175
        Module11bVO2maxEstimator.Result result =
                Module11bVO2maxEstimator.estimate(140.0, 85.0, 175.0);
        assertTrue(result.vo2max < 40.0);
    }

    @Test
    void nullRestingHR_usesDefault() {
        Module11bVO2maxEstimator.Result result =
                Module11bVO2maxEstimator.estimate(160.0, null, 180.0);
        assertNotNull(result);
        // Uses default resting HR of 72
    }

    @Test
    void insufficientData_peakHRTooLow() {
        // Peak HR below 60% of HR_max — not valid submaximal test
        Module11bVO2maxEstimator.Result result =
                Module11bVO2maxEstimator.estimate(95.0, 70.0, 180.0);
        assertNull(result); // 95/180 = 53% < 60% threshold
    }

    @Test
    void averageFromMultipleSessions() {
        List<Double> peakHRs = List.of(155.0, 160.0, 158.0);
        Double restingHR = 65.0;
        double hrMax = 180.0;

        List<Module11bVO2maxEstimator.Result> results = new ArrayList<>();
        for (Double peakHR : peakHRs) {
            Module11bVO2maxEstimator.Result r =
                    Module11bVO2maxEstimator.estimate(peakHR, restingHR, hrMax);
            if (r != null) results.add(r);
        }
        assertEquals(3, results.size());
        // Averaging produces more stable estimate
        double avgVO2max = results.stream().mapToDouble(r -> r.vo2max).average().orElse(0);
        assertTrue(avgVO2max > 30.0);
    }
}
