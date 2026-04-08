package com.cardiofit.flink.thresholds;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Validates that CIDThresholdSet.hardcodedDefaults() matches the constants
 * currently hardcoded in Module8 HALT/PAUSE/SOFT_FLAG evaluators.
 *
 * If an evaluator constant is changed without updating CIDThresholdSet,
 * these tests will fail — preventing silent threshold drift.
 */
class CIDThresholdSetTest {

    @Test
    void hardcodedDefaults_matchesHALTEvaluatorConstants() {
        CIDThresholdSet d = CIDThresholdSet.hardcodedDefaults();

        // CID-02: Module8HALTEvaluator.POTASSIUM_THRESHOLD
        assertEquals(5.3, d.getPotassiumThreshold(), 1e-9);
        // CID-03: Module8HALTEvaluator.GLUCOSE_HYPO_THRESHOLD
        assertEquals(60.0, d.getGlucoseHypoThreshold(), 1e-9);
        // CID-05: Module8HALTEvaluator.SBP_HYPOTENSION_THRESHOLD
        assertEquals(95.0, d.getSbpHypotensionThreshold(), 1e-9);
        // CID-05: Module8HALTEvaluator.SBP_DROP_THRESHOLD
        assertEquals(30.0, d.getSbpDropThreshold(), 1e-9);
        // CID-01: Module8HALTEvaluator.EGFR_DROP_THRESHOLD_PCT
        assertEquals(15.0, d.getEgfrDropThresholdPct(), 1e-9);
        // CID-01: Module8HALTEvaluator.WEIGHT_DROP_THRESHOLD_KG
        assertEquals(2.0, d.getWeightDropThresholdKg(), 1e-9);
        // CID-05: Module8HALTEvaluator.MIN_ANTIHYPERTENSIVES
        assertEquals(3, d.getMinAntihypertensives());
        // CID-01 inline at HALTEvaluator:88
        assertEquals(45.0, d.getEgfrCriticalRenalThreshold(), 1e-9);
    }

    @Test
    void hardcodedDefaults_matchesPAUSEEvaluatorConstants() {
        CIDThresholdSet d = CIDThresholdSet.hardcodedDefaults();

        assertEquals(15.0, d.getFbgDeltaThreshold(), 1e-9);      // CID-06
        assertEquals(25.0, d.getEgfrDeclineThresholdPct(), 1e-9); // CID-07
        assertEquals(6, d.getEgfrDipWindowWeeks());                // CID-07
        assertEquals(1.5, d.getWeightDropGIThresholdKg(), 1e-9);  // CID-09
        assertEquals(10.0, d.getGlucoseWorseningThreshold(), 1e-9); // CID-10
        assertEquals(10.0, d.getSbpWorseningThreshold(), 1e-9);     // CID-10
    }

    @Test
    void hardcodedDefaults_matchesSOFTFLAGEvaluatorConstants() {
        CIDThresholdSet d = CIDThresholdSet.hardcodedDefaults();

        assertEquals(8, d.getPolypharmacyThreshold());       // CID-12
        assertEquals(75, d.getElderlyAgeThreshold());        // CID-13
        assertEquals(130.0, d.getSbpTargetIntensive(), 1e-9); // CID-13
        assertEquals(30.0, d.getEgfrMetforminLow(), 1e-9);   // CID-14
        assertEquals(35.0, d.getEgfrMetforminHigh(), 1e-9);  // CID-14
        assertEquals(16, d.getFastingDurationThreshold());    // CID-17
    }

    @Test
    void hardcodedDefaults_hasVersionAndTimestamp() {
        CIDThresholdSet d = CIDThresholdSet.hardcodedDefaults();

        assertEquals("hardcoded-cid-v1.0.0", d.getVersion());
        assertTrue(d.getLoadedAtEpochMs() > 0);
    }

    @Test
    void settersOverrideDefaults() {
        CIDThresholdSet set = CIDThresholdSet.hardcodedDefaults();
        set.setPotassiumThreshold(5.0);
        set.setGlucoseHypoThreshold(70.0);
        set.setPolypharmacyThreshold(5);

        assertEquals(5.0, set.getPotassiumThreshold(), 1e-9);
        assertEquals(70.0, set.getGlucoseHypoThreshold(), 1e-9);
        assertEquals(5, set.getPolypharmacyThreshold());
    }
}
