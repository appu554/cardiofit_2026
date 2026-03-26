package com.cardiofit.flink.analytics;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class ARVCalculatorTest {

    @Test
    void arv_returnsNull_whenLessThan3Readings() {
        double[] sbpValues = {130.0, 125.0};
        assertNull(ARVCalculator.compute(sbpValues));
    }

    @Test
    void arv_computesCorrectly_forKnownSequence() {
        // |125-130| + |135-125| + |128-135| = 5 + 10 + 7 = 22; ARV = 22/3 = 7.33
        double[] sbpValues = {130.0, 125.0, 135.0, 128.0};
        Double arv = ARVCalculator.compute(sbpValues);
        assertNotNull(arv);
        assertEquals(7.33, arv, 0.01);
    }

    @Test
    void arv_classification_lowForBelow8() {
        assertEquals("LOW", ARVCalculator.classify(6.5));
    }

    @Test
    void arv_classification_veryHighForAbove15() {
        assertEquals("VERY_HIGH", ARVCalculator.classify(18.0));
    }
}
