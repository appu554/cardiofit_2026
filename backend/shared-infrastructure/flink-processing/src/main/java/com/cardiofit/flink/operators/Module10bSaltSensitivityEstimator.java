package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.MealPatternState;
import com.cardiofit.flink.models.SaltSensitivityClass;
import java.util.List;

/**
 * OLS linear regression estimator for salt sensitivity.
 *
 * Regresses SBP_excursion on sodium_mg from a 60-day rolling buffer.
 * Three division-by-zero guards:
 * 1. n = 0 → UNDETERMINED
 * 2. SS_xx = 0 (all sodium values identical) → UNDETERMINED
 * 3. SS_tot = 0 (all SBP excursions identical) → R² = 0
 */
public class Module10bSaltSensitivityEstimator {

    private Module10bSaltSensitivityEstimator() {}

    public static Result estimate(List<MealPatternState.SodiumSBPPair> pairs) {
        Result result = new Result();
        result.pairCount = pairs != null ? pairs.size() : 0;

        if (result.pairCount == 0) {
            result.classification = SaltSensitivityClass.UNDETERMINED;
            result.beta = 0.0;
            result.rSquared = 0.0;
            return result;
        }

        int n = result.pairCount;

        double sumX = 0, sumY = 0;
        for (MealPatternState.SodiumSBPPair p : pairs) {
            sumX += p.sodiumMg;
            sumY += p.sbpExcursion;
        }
        double meanX = sumX / n;
        double meanY = sumY / n;

        double ssXX = 0, ssXY = 0, ssTot = 0;
        for (MealPatternState.SodiumSBPPair p : pairs) {
            double dx = p.sodiumMg - meanX;
            double dy = p.sbpExcursion - meanY;
            ssXX += dx * dx;
            ssXY += dx * dy;
            ssTot += dy * dy;
        }

        if (ssXX < 1e-12) {
            result.beta = 0.0;
            result.rSquared = 0.0;
            result.classification = SaltSensitivityClass.UNDETERMINED;
            return result;
        }

        double beta = ssXY / ssXX;
        double ssReg = beta * beta * ssXX;
        double rSquared = (ssTot > 1e-12) ? ssReg / ssTot : 0.0;

        result.beta = beta;
        result.rSquared = rSquared;
        result.classification = SaltSensitivityClass.fromBetaAndR2(beta, rSquared, n);

        return result;
    }

    public static class Result {
        public SaltSensitivityClass classification;
        public double beta;
        public double rSquared;
        public int pairCount;
    }
}
