package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.ArrayList;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module10bSaltSensitivityTest {

    @Test
    void highSaltSensitivity_betaAbove005() {
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        for (int i = 0; i < 40; i++) {
            double sodium = 200 + i * 50;
            double sbpExcursion = 2.0 + sodium * 0.008;
            pairs.add(new MealPatternState.SodiumSBPPair(sodium, sbpExcursion, now - i * 86400000L));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        assertEquals(SaltSensitivityClass.HIGH, result.classification);
        assertTrue(result.beta >= 0.005);
        assertTrue(result.rSquared > 0.1);
        assertEquals(40, result.pairCount);
    }

    @Test
    void saltResistant_betaBelow0001() {
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        for (int i = 0; i < 35; i++) {
            double sodium = 300 + i * 40;
            double sbpExcursion = 5.0 + Math.random() * 0.5;
            pairs.add(new MealPatternState.SodiumSBPPair(sodium, sbpExcursion, now - i * 86400000L));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        assertEquals(SaltSensitivityClass.SALT_RESISTANT, result.classification);
        assertTrue(result.beta < 0.001);
    }

    @Test
    void undetermined_tooFewPairs() {
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        for (int i = 0; i < 15; i++) {
            pairs.add(new MealPatternState.SodiumSBPPair(500 + i * 100, 5.0 + i, now));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        assertEquals(SaltSensitivityClass.UNDETERMINED, result.classification);
        assertEquals(15, result.pairCount);
    }

    @Test
    void undetermined_lowRSquared() {
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        java.util.Random rng = new java.util.Random(42);
        for (int i = 0; i < 40; i++) {
            pairs.add(new MealPatternState.SodiumSBPPair(
                rng.nextDouble() * 2000, rng.nextDouble() * 30 - 15, now));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        if (result.rSquared < 0.1) {
            assertEquals(SaltSensitivityClass.UNDETERMINED, result.classification);
        }
    }

    @Test
    void emptyPairs_returnsUndetermined() {
        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(new ArrayList<>());

        assertEquals(SaltSensitivityClass.UNDETERMINED, result.classification);
        assertEquals(0, result.pairCount);
    }

    @Test
    void divisionByZeroGuard_allSameSodium() {
        List<MealPatternState.SodiumSBPPair> pairs = new ArrayList<>();
        long now = System.currentTimeMillis();
        for (int i = 0; i < 35; i++) {
            pairs.add(new MealPatternState.SodiumSBPPair(500.0, 5.0 + i * 0.1, now));
        }

        Module10bSaltSensitivityEstimator.Result result =
            Module10bSaltSensitivityEstimator.estimate(pairs);

        assertEquals(SaltSensitivityClass.UNDETERMINED, result.classification);
    }
}
