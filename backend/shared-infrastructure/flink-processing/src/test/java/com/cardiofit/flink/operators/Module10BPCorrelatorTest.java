package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module10BPCorrelatorTest {

    @Test
    void completeBP_computesExcursion() {
        BPWindow bpWindow = new BPWindow();
        bpWindow.setPreMealSBP(120.0);
        bpWindow.setPreMealDBP(80.0);
        bpWindow.setPostMealSBP(135.0);
        bpWindow.setPostMealDBP(85.0);

        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(bpWindow);

        assertNotNull(result);
        assertEquals(15.0, result.sbpExcursion);
        assertTrue(result.complete);
    }

    @Test
    void negativeExcursion_bpDropsPostMeal() {
        BPWindow bpWindow = new BPWindow();
        bpWindow.setPreMealSBP(140.0);
        bpWindow.setPreMealDBP(90.0);
        bpWindow.setPostMealSBP(125.0);
        bpWindow.setPostMealDBP(82.0);

        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(bpWindow);

        assertNotNull(result);
        assertEquals(-15.0, result.sbpExcursion);
        assertTrue(result.complete);
    }

    @Test
    void missingPreMeal_incompleteResult() {
        BPWindow bpWindow = new BPWindow();
        bpWindow.setPostMealSBP(130.0);
        bpWindow.setPostMealDBP(85.0);

        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(bpWindow);

        assertNotNull(result);
        assertNull(result.sbpExcursion);
        assertFalse(result.complete);
        assertEquals(130.0, result.postMealSBP);
    }

    @Test
    void missingPostMeal_incompleteResult() {
        BPWindow bpWindow = new BPWindow();
        bpWindow.setPreMealSBP(120.0);
        bpWindow.setPreMealDBP(80.0);

        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(bpWindow);

        assertNotNull(result);
        assertNull(result.sbpExcursion);
        assertFalse(result.complete);
        assertEquals(120.0, result.preMealSBP);
    }

    @Test
    void nullWindow_returnsNull() {
        Module10BPCorrelator.Result result = Module10BPCorrelator.analyze(null);
        assertNull(result);
    }
}
