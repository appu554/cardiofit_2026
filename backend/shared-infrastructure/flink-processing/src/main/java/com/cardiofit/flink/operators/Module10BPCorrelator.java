package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.BPWindow;

/**
 * BP correlation analysis for Module 10.
 *
 * Computes pre-meal vs post-meal BP excursion from a BPWindow.
 * Pre-meal BP is retroactively attached from state buffer (most recent within 60 min).
 * Post-meal BP is the first reading within 4h after meal.
 *
 * Stateless utility class.
 */
public class Module10BPCorrelator {

    private Module10BPCorrelator() {}

    public static Result analyze(BPWindow bpWindow) {
        if (bpWindow == null) return null;

        Result r = new Result();
        r.preMealSBP = bpWindow.getPreMealSBP();
        r.preMealDBP = bpWindow.getPreMealDBP();
        r.postMealSBP = bpWindow.getPostMealSBP();
        r.postMealDBP = bpWindow.getPostMealDBP();
        r.complete = bpWindow.isComplete();

        if (r.complete) {
            r.sbpExcursion = bpWindow.getSBPExcursion();
        }

        return r;
    }

    public static class Result {
        public Double preMealSBP;
        public Double preMealDBP;
        public Double postMealSBP;
        public Double postMealDBP;
        public Double sbpExcursion;
        public boolean complete;
    }
}
