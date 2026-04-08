package com.cardiofit.flink.thresholds;

import com.cardiofit.flink.models.CIDSeverity;
import com.cardiofit.flink.thresholds.CIDThresholdResolver.ResolvedThreshold;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests three-layer resolution, HALT safety clamping, and provenance tracking.
 */
class CIDThresholdResolverTest {

    private static final double HARDCODED_K = 5.3;
    private static final double HARDCODED_GLUCOSE = 60.0;
    private static final double HARDCODED_EGFR_DROP = 15.0;

    // ═══════════════════════════════════════════════════════════════
    // Layer resolution order
    // ═══════════════════════════════════════════════════════════════

    @Nested
    class LayerResolution {

        @Test
        void layer1_personalizedTakesPrecedence() {
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "POTASSIUM_THRESHOLD", CIDSeverity.HALT,
                    5.0,            // Layer 1: personalised (more sensitive)
                    HARDCODED_K,    // Layer 2: guideline = hardcoded
                    HARDCODED_K);   // Layer 3: hardcoded

            assertEquals(5.0, r.getValue(), 1e-9);
            assertEquals("PATIENT", r.getProvenance().getLayer());
            assertEquals("KB-20", r.getProvenance().getSource());
        }

        @Test
        void layer2_guidelineUsedWhenNoPersonalized() {
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "POTASSIUM_THRESHOLD", CIDSeverity.HALT,
                    null,           // Layer 1: no personalised value
                    5.1,            // Layer 2: guideline (different from hardcoded)
                    HARDCODED_K);   // Layer 3: hardcoded

            assertEquals(5.1, r.getValue(), 1e-9);
            assertEquals("GUIDELINE", r.getProvenance().getLayer());
            assertEquals("KB-16", r.getProvenance().getSource());
        }

        @Test
        void layer3_hardcodedFallbackWhenNeitherAvailable() {
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "POTASSIUM_THRESHOLD", CIDSeverity.HALT,
                    null,           // Layer 1: absent
                    HARDCODED_K,    // Layer 2: same as hardcoded → treated as absent
                    HARDCODED_K);   // Layer 3: hardcoded

            assertEquals(HARDCODED_K, r.getValue(), 1e-9);
            assertEquals("HARDCODED", r.getProvenance().getLayer());
            assertEquals("HARDCODED", r.getProvenance().getSource());
        }

        @Test
        void layer1_overridesLayer2() {
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "POTASSIUM_THRESHOLD", CIDSeverity.PAUSE,
                    4.8,            // Layer 1
                    5.1,            // Layer 2
                    HARDCODED_K);   // Layer 3

            // Layer 1 wins over Layer 2
            assertEquals(4.8, r.getValue(), 1e-9);
            assertEquals("PATIENT", r.getProvenance().getLayer());
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // HALT safety clamping
    // ═══════════════════════════════════════════════════════════════

    @Nested
    class HALTSafetyClamping {

        @Test
        void halt_lowerIsSensitive_clampsPotassiumRelaxation() {
            // KB-20 sends 5.8 (more relaxed) for HALT rule → clamp to hardcoded 5.3
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "POTASSIUM_THRESHOLD", CIDSeverity.HALT,
                    5.8,            // Incorrectly relaxed
                    HARDCODED_K,
                    HARDCODED_K,
                    true);          // lower = more sensitive

            assertEquals(HARDCODED_K, r.getValue(), 1e-9, "HALT clamp should reject relaxation");
            assertEquals("PATIENT", r.getProvenance().getLayer());
        }

        @Test
        void halt_lowerIsSensitive_allowsTightening() {
            // KB-20 sends 5.0 (more sensitive) for HALT rule → allowed
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "POTASSIUM_THRESHOLD", CIDSeverity.HALT,
                    5.0,            // Tighter — allowed
                    HARDCODED_K,
                    HARDCODED_K,
                    true);

            assertEquals(5.0, r.getValue(), 1e-9, "HALT clamp should allow tightening");
        }

        @Test
        void halt_glucoseRelaxationClamped() {
            // Glucose hypo threshold: lower = more sensitive
            // Personalised 70 (relaxed) should clamp to 60 (hardcoded)
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "GLUCOSE_HYPO_THRESHOLD", CIDSeverity.HALT,
                    70.0,           // Relaxed — would miss hypos between 60-70
                    HARDCODED_GLUCOSE,
                    HARDCODED_GLUCOSE,
                    true);

            assertEquals(HARDCODED_GLUCOSE, r.getValue(), 1e-9);
        }

        @Test
        void halt_glucoseTighteningAllowed() {
            // Personalised 54 (tighter) is allowed for HALT
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "GLUCOSE_HYPO_THRESHOLD", CIDSeverity.HALT,
                    54.0,
                    HARDCODED_GLUCOSE,
                    HARDCODED_GLUCOSE,
                    true);

            assertEquals(54.0, r.getValue(), 1e-9);
        }

        @Test
        void halt_higherIsSensitive_egfrDropClamping() {
            // eGFR drop %: HIGHER % means "tolerate more decline before firing" = LESS sensitive
            // So we want lowerIsSensitive=false → clamp via Math.max
            // If personalized says 10% (less sensitive), clamp to 15% (hardcoded)
            // Wait — this is inverted. Let me think about this carefully:
            //
            // EGFR_DROP_THRESHOLD_PCT = 15 means "fire when decline >= 15%"
            // A LOWER threshold (10%) fires SOONER (at 10% decline) = MORE sensitive
            // So eGFR drop % IS lower-is-sensitive = true (same as potassium)
            //
            // Personalized 10% (more sensitive) → allowed
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "EGFR_DROP_THRESHOLD_PCT", CIDSeverity.HALT,
                    10.0,
                    HARDCODED_EGFR_DROP,
                    HARDCODED_EGFR_DROP,
                    true);

            assertEquals(10.0, r.getValue(), 1e-9, "Lower eGFR drop % = more sensitive = allowed");
        }

        @Test
        void halt_egfrDropRelaxationClamped() {
            // Personalized 25% (less sensitive — tolerates more decline) → clamp to 15%
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "EGFR_DROP_THRESHOLD_PCT", CIDSeverity.HALT,
                    25.0,
                    HARDCODED_EGFR_DROP,
                    HARDCODED_EGFR_DROP,
                    true);

            assertEquals(HARDCODED_EGFR_DROP, r.getValue(), 1e-9,
                    "Relaxed eGFR drop % should be clamped to hardcoded");
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // PAUSE/SOFT_FLAG: no clamping
    // ═══════════════════════════════════════════════════════════════

    @Nested
    class PAUSEAndSOFTFLAG {

        @Test
        void pause_allowsRelaxation() {
            // PAUSE rules may legitimately relax thresholds
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "FBG_DELTA_THRESHOLD", CIDSeverity.PAUSE,
                    20.0,           // Relaxed from 15 → 20 (fewer alerts)
                    15.0,
                    15.0);

            assertEquals(20.0, r.getValue(), 1e-9, "PAUSE should allow relaxation");
        }

        @Test
        void pause_allowsTightening() {
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "FBG_DELTA_THRESHOLD", CIDSeverity.PAUSE,
                    10.0,           // Tighter
                    15.0,
                    15.0);

            assertEquals(10.0, r.getValue(), 1e-9);
        }

        @Test
        void softFlag_allowsRelaxation() {
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "POLYPHARMACY_THRESHOLD", CIDSeverity.SOFT_FLAG,
                    10.0,           // Relaxed from 8 → 10
                    8.0,
                    8.0);

            assertEquals(10.0, r.getValue(), 1e-9, "SOFT_FLAG should allow relaxation");
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // Provenance tracking
    // ═══════════════════════════════════════════════════════════════

    @Nested
    class ProvenanceTracking {

        @Test
        void provenance_capturesThresholdKey() {
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "POTASSIUM_THRESHOLD", CIDSeverity.HALT,
                    5.0, HARDCODED_K, HARDCODED_K);

            assertEquals("POTASSIUM_THRESHOLD", r.getProvenance().getThresholdKey());
        }

        @Test
        void provenance_capturesDefaultValue() {
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "GLUCOSE_HYPO_THRESHOLD", CIDSeverity.HALT,
                    null, HARDCODED_GLUCOSE, HARDCODED_GLUCOSE);

            assertEquals(HARDCODED_GLUCOSE, r.getProvenance().getDefaultValue(), 1e-9);
            assertEquals(HARDCODED_GLUCOSE, r.getProvenance().getValueUsed(), 1e-9);
        }

        @Test
        void provenance_resolvedAtIsRecent() {
            long before = System.currentTimeMillis();
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "TEST", CIDSeverity.PAUSE, null, 5.0, 5.0);
            long after = System.currentTimeMillis();

            assertTrue(r.getProvenance().getResolvedAt() >= before);
            assertTrue(r.getProvenance().getResolvedAt() <= after);
        }

        @Test
        void provenance_clampedStillShowsPatientLayer() {
            // Even when clamped, provenance should show PATIENT layer
            // (the value was attempted from patient, just clamped)
            ResolvedThreshold r = CIDThresholdResolver.resolve(
                    "POTASSIUM_THRESHOLD", CIDSeverity.HALT,
                    5.8,            // Clamped
                    HARDCODED_K,
                    HARDCODED_K);

            assertEquals("PATIENT", r.getProvenance().getLayer());
            assertEquals(HARDCODED_K, r.getProvenance().getValueUsed(), 1e-9);
        }
    }

    // ═══════════════════════════════════════════════════════════════
    // Convenience overload
    // ═══════════════════════════════════════════════════════════════

    @Test
    void defaultOverload_assumesLowerIsSensitive() {
        // The 5-arg overload defaults to lowerIsSensitive=true
        ResolvedThreshold r = CIDThresholdResolver.resolve(
                "POTASSIUM_THRESHOLD", CIDSeverity.HALT,
                5.8,            // Relaxed — should be clamped
                HARDCODED_K,
                HARDCODED_K);

        assertEquals(HARDCODED_K, r.getValue(), 1e-9, "Default overload should clamp relaxation");
    }
}
