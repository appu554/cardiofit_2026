package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module10CurveClassifierTest {

    private static final long BASE = 1743552000000L;
    private static final long MIN_5 = 5 * 60_000L;

    @Test
    void rapidSpike_peakWithin30min() {
        GlucoseWindow window = cgmWindow(
            100, 130, 160, 180, 170, 140, 110, 105, 100, 98, 97, 96
        );
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.RAPID_SPIKE, shape);
    }

    @Test
    void slowRise_peakAfter60min() {
        GlucoseWindow window = cgmWindow(
            100, 105, 110, 115, 120, 125, 130, 135, 140, 150, 160, 155, 140, 120, 105
        );
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.SLOW_RISE, shape);
    }

    @Test
    void plateau_sustainedElevation() {
        GlucoseWindow window = cgmWindow(
            100, 120, 140, 150, 152, 151, 150, 149, 148, 147, 146, 145
        );
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.PLATEAU, shape);
    }

    @Test
    void doublePeak_twoPeaksSeparatedBy20min() {
        GlucoseWindow window = cgmWindow(
            100, 140, 160, 130, 120, 115, 120, 145, 155, 140, 120, 105
        );
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.DOUBLE_PEAK, shape);
    }

    @Test
    void flat_excursionBelow20() {
        GlucoseWindow window = cgmWindow(
            100, 102, 105, 108, 110, 112, 110, 108, 105, 103, 101, 100
        );
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.FLAT, shape);
    }

    @Test
    void unknown_insufficientReadings() {
        GlucoseWindow window = new GlucoseWindow();
        window.setBaseline(100.0);
        window.addReading(BASE, 100.0, "CGM");
        window.addReading(BASE + MIN_5, 130.0, "CGM");
        window.sortByTime();
        CurveShape shape = Module10CurveClassifier.classify(window);
        assertEquals(CurveShape.UNKNOWN, shape);
    }

    /** Helper: create CGM window with 5-min interval readings starting at BASE */
    private GlucoseWindow cgmWindow(double... values) {
        GlucoseWindow w = new GlucoseWindow();
        w.setBaseline(values[0]);
        w.setDataTier(DataTier.TIER_1_CGM);
        for (int i = 0; i < values.length; i++) {
            w.addReading(BASE + i * MIN_5, values[i], "CGM");
        }
        w.sortByTime();
        return w;
    }
}
