package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class Module12TrajectoryTrackerTest {

    private static final long DAY_MS = 86_400_000L;
    private static final long BASE_TIME = 1743552000000L;

    private static List<InterventionWindowState.TrajectoryDataPoint> makeReadings(
            String domain, double[] values, long startTime, long intervalMs) {
        List<InterventionWindowState.TrajectoryDataPoint> points = new ArrayList<>();
        for (int i = 0; i < values.length; i++) {
            points.add(new InterventionWindowState.TrajectoryDataPoint(
                    domain, values[i], startTime + i * intervalMs));
        }
        return points;
    }

    @Test
    void decliningFBG_classifiedAsImproving() {
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                makeReadings("FBG", new double[]{160, 155, 148, 142, 138}, BASE_TIME, 3 * DAY_MS);
        long windowStart = BASE_TIME + 14 * DAY_MS;

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "FBG", BASE_TIME, windowStart);

        assertEquals(TrajectoryClassification.IMPROVING, result);
    }

    @Test
    void risingSBP_classifiedAsDeteriorating() {
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                makeReadings("SBP", new double[]{130, 133, 137, 140, 144}, BASE_TIME, 3 * DAY_MS);
        long windowStart = BASE_TIME + 14 * DAY_MS;

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "SBP", BASE_TIME, windowStart);

        assertEquals(TrajectoryClassification.DETERIORATING, result);
    }

    @Test
    void flatReadingsWithNoise_classifiedAsStable() {
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                makeReadings("FBG", new double[]{120, 118, 122, 119, 121}, BASE_TIME, 3 * DAY_MS);
        long windowStart = BASE_TIME + 14 * DAY_MS;

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "FBG", BASE_TIME, windowStart);

        assertEquals(TrajectoryClassification.STABLE, result);
    }

    @Test
    void insufficientReadings_classifiedAsUnknown() {
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                makeReadings("FBG", new double[]{120, 125}, BASE_TIME, 3 * DAY_MS);

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "FBG", BASE_TIME, BASE_TIME + 7 * DAY_MS);

        assertEquals(TrajectoryClassification.UNKNOWN, result);
    }

    @Test
    void eGFR_rapidDecline_classifiedAsDeteriorating() {
        long yearMs = 365L * DAY_MS;
        List<InterventionWindowState.TrajectoryDataPoint> readings =
                makeReadings("EGFR", new double[]{55, 52, 48, 45}, BASE_TIME, yearMs / 4);
        long end = BASE_TIME + yearMs;

        TrajectoryClassification result = Module12TrajectoryTracker.classify(
                readings, "EGFR", BASE_TIME, end);

        assertEquals(TrajectoryClassification.DETERIORATING, result);
    }

    @Test
    void compositeWorstDomain_deterioratingWins() {
        InterventionWindowState state = new InterventionWindowState("P1");
        for (int i = 0; i < 5; i++) {
            state.addReading("FBG", 160 - i * 5.0, BASE_TIME + i * 3 * DAY_MS);
        }
        for (int i = 0; i < 5; i++) {
            state.addReading("SBP", 130 + i * 4.0, BASE_TIME + i * 3 * DAY_MS);
        }
        long windowEnd = BASE_TIME + 14 * DAY_MS;

        TrajectoryClassification result = Module12TrajectoryTracker.classifyComposite(
                state, BASE_TIME, windowEnd);

        assertEquals(TrajectoryClassification.DETERIORATING, result);
    }
}
