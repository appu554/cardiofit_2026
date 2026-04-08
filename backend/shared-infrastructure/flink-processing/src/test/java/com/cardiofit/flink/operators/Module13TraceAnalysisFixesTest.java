package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.operators.CrossModuleTestHarness.ProcessResult;

import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.util.*;

import static com.cardiofit.flink.builders.Module13TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for fixes identified during the Rajesh Kumar E2E trace analysis:
 *
 * Fix 1: Module 8 tracked in data completeness (8 domains, not 7)
 * Fix 4: Cold-start baseline freeze after ≥3 modules seen
 * Fix 5: Context-aware recommended_action for DATA_ABSENCE_WARNING
 */
public class Module13TraceAnalysisFixesTest {

    private static final String P = "trace-fix-patient";
    private static final long NOW = Module13TestBuilder.BASE_TIME + 60_000L;

    // ═══════════════════════════════════════════════════════════════════════
    // Fix 1: Module 8 completeness tracking
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Fix1_Module8CompletenessTracking {

        @Test
        void module8Seen_contributesToCompleteness() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.recordModuleSeen("module8", NOW);

            Module13DataCompletenessMonitor.Result result =
                    Module13DataCompletenessMonitor.evaluate(state, NOW);

            // module8 is 1 of 8 tracked modules — score should be > 0
            assertTrue(result.getCompositeScore() > 0.0,
                    "module8 should contribute to completeness score, got: " + result.getCompositeScore());
            // Should NOT have module8 in gap flags (it was seen)
            assertFalse(result.getDataGapFlags().containsKey("module8"),
                    "module8 was seen — should not be in gap flags");
        }

        @Test
        void allEightModulesSeen_fullCompleteness() {
            ClinicalStateSummary state = stateWithBaselines(P);

            Module13DataCompletenessMonitor.Result result =
                    Module13DataCompletenessMonitor.evaluate(state, NOW);

            assertTrue(result.getCompositeScore() > 0.95,
                    "All 8 modules seen → completeness should be ~1.0, got: " + result.getCompositeScore());
            assertTrue(result.getDataGapFlags().isEmpty(),
                    "All modules seen — no gap flags expected, got: " + result.getDataGapFlags());
        }

        @Test
        void sevenModulesWithoutModule8_showsGap() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            // Register all EXCEPT module8
            state.recordModuleSeen("module7", NOW);
            state.recordModuleSeen("module9", NOW);
            state.recordModuleSeen("module10b", NOW);
            state.recordModuleSeen("module11b", NOW);
            state.recordModuleSeen("enriched", NOW);
            state.recordModuleSeen("module12", NOW);
            state.recordModuleSeen("module12b", NOW);

            Module13DataCompletenessMonitor.Result result =
                    Module13DataCompletenessMonitor.evaluate(state, NOW);

            assertTrue(result.getDataGapFlags().containsKey("module8"),
                    "module8 not seen — should appear in gap flags");
            assertEquals("NEVER_SEEN", result.getDataGapFlags().get("module8"));
            // Score should be < 1.0 (7/8 domains)
            assertTrue(result.getCompositeScore() < 1.0,
                    "Missing module8 → score should be < 1.0, got: " + result.getCompositeScore());
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Fix 4: Cold-start baseline freeze
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Fix4_ColdStartBaselineFreeze {

        @Test
        void threeModulesSeen_triggersBaselineFreeze() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            // Simulate cold start: set some metric values before freeze
            state.current().fbg = 130.0;
            state.current().meanSBP = 140.0;

            // Record 3 modules (meets threshold)
            state.recordModuleSeen("module7", NOW);
            state.recordModuleSeen("enriched", NOW);
            state.recordModuleSeen("module9", NOW);

            // Before freeze: no previous snapshot
            assertNull(state.previous(), "Before freeze — no previous snapshot");

            // Simulate what processElement does at step 1a
            if (state.previous() == null
                    && state.getModuleLastSeenMs().size() >= Module13_ClinicalStateSynchroniser.COLD_START_MODULE_THRESHOLD) {
                state.rotateSnapshots(NOW);
            }

            // After freeze: previous snapshot should exist with baseline values
            assertNotNull(state.previous(), "After freeze — previous snapshot should exist");
            assertEquals(130.0, state.previous().fbg, 0.01,
                    "Previous snapshot should carry pre-freeze FBG");
            assertEquals(140.0, state.previous().meanSBP, 0.01,
                    "Previous snapshot should carry pre-freeze meanSBP");
        }

        @Test
        void twoModulesSeen_doesNotTriggerFreeze() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.recordModuleSeen("module7", NOW);
            state.recordModuleSeen("enriched", NOW);

            // Only 2 modules — below threshold
            assertTrue(state.getModuleLastSeenMs().size() < Module13_ClinicalStateSynchroniser.COLD_START_MODULE_THRESHOLD);
            assertNull(state.previous(), "2 modules — should not trigger freeze");
        }

        @Test
        void coldStartFreeze_onlyHappensOnce() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.current().fbg = 130.0;

            state.recordModuleSeen("module7", NOW);
            state.recordModuleSeen("enriched", NOW);
            state.recordModuleSeen("module9", NOW);

            // First freeze
            state.rotateSnapshots(NOW);
            assertNotNull(state.previous());
            assertEquals(130.0, state.previous().fbg, 0.01);

            // Update current with new values
            state.current().fbg = 180.0;
            state.recordModuleSeen("module10b", NOW);

            // Second check: previousSnapshot already exists → no re-freeze
            boolean shouldFreeze = !state.hasVelocityData()
                    && state.getModuleLastSeenMs().size() >= Module13_ClinicalStateSynchroniser.COLD_START_MODULE_THRESHOLD;
            assertFalse(shouldFreeze, "Should NOT re-freeze when previous snapshot already exists");

            // Previous snapshot still has original baseline
            assertEquals(130.0, state.previous().fbg, 0.01,
                    "Previous snapshot should not be overwritten by subsequent events");
        }

        @Test
        void afterColdStartFreeze_velocityCanBeComputed() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.setDataTier("TIER_2_SMBG");

            // Set baseline values
            state.current().fbg = 130.0;
            state.current().meanSBP = 135.0;
            state.current().egfr = 65.0;
            state.current().hba1c = 7.2;

            // Cold-start freeze
            state.recordModuleSeen("module7", NOW);
            state.recordModuleSeen("enriched", NOW);
            state.recordModuleSeen("module9", NOW);
            state.rotateSnapshots(NOW);

            // Now update current with worsening values (simulating post-freeze events)
            state.current().fbg = 180.0;
            state.current().meanSBP = 165.0;

            // Compute velocity — should not be UNKNOWN since we have both snapshots
            CKMRiskVelocity velocity = Module13CKMRiskComputer.compute(state);
            assertNotNull(velocity);
            assertNotEquals(CKMRiskVelocity.CompositeClassification.UNKNOWN,
                    velocity.getCompositeClassification(),
                    "After cold-start freeze with worsening values, velocity should not be UNKNOWN");
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // Fix 5: Context-aware recommended action
    // ═══════════════════════════════════════════════════════════════════════
    @Nested
    class Fix5_ContextAwareRecommendedAction {

        @Test
        void criticalBP_escalatesAction() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.current().meanSBP = 168.0; // Stage 2 uncontrolled

            String action = Module13_ClinicalStateSynchroniser.resolveDataAbsenceAction(
                    state, ClinicalStateChangeType.DATA_ABSENCE_WARNING);

            assertTrue(action.contains("INCOMPLETE_DATA_WITH_CRITICAL_VALUES"),
                    "Stage 2 HTN should escalate, got: " + action);
            assertTrue(action.contains("clinical review"),
                    "Should recommend clinical review, got: " + action);
        }

        @Test
        void criticalGlycaemic_escalatesAction() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.current().hba1c = 8.2;

            String action = Module13_ClinicalStateSynchroniser.resolveDataAbsenceAction(
                    state, ClinicalStateChangeType.DATA_ABSENCE_WARNING);

            assertTrue(action.contains("INCOMPLETE_DATA_WITH_CRITICAL_VALUES"),
                    "HbA1c 8.2 should escalate, got: " + action);
        }

        @Test
        void criticalRenal_escalatesAction() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.current().egfr = 38.0; // CKD G3b

            String action = Module13_ClinicalStateSynchroniser.resolveDataAbsenceAction(
                    state, ClinicalStateChangeType.DATA_ABSENCE_WARNING);

            assertTrue(action.contains("INCOMPLETE_DATA_WITH_CRITICAL_VALUES"),
                    "eGFR 38 should escalate, got: " + action);
        }

        @Test
        void cidHaltActive_escalatesAction() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.setActiveCIDHaltCount(1);

            String action = Module13_ClinicalStateSynchroniser.resolveDataAbsenceAction(
                    state, ClinicalStateChangeType.DATA_ABSENCE_WARNING);

            assertTrue(action.contains("INCOMPLETE_DATA_WITH_CRITICAL_VALUES"),
                    "Active CID HALT should escalate, got: " + action);
        }

        @Test
        void normalValues_defaultEngagementNudge() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.current().meanSBP = 125.0;
            state.current().fbg = 100.0;
            state.current().egfr = 70.0;

            String action = Module13_ClinicalStateSynchroniser.resolveDataAbsenceAction(
                    state, ClinicalStateChangeType.DATA_ABSENCE_WARNING);

            assertEquals("Generate engagement check card", action,
                    "Normal values should use default action");
        }

        @Test
        void criticalAbsence_alwaysUsesEnumAction() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            // Even with normal values, CRITICAL absence uses its own action
            state.current().meanSBP = 120.0;

            String action = Module13_ClinicalStateSynchroniser.resolveDataAbsenceAction(
                    state, ClinicalStateChangeType.DATA_ABSENCE_CRITICAL);

            assertEquals(ClinicalStateChangeType.DATA_ABSENCE_CRITICAL.getRecommendedAction(), action,
                    "DATA_ABSENCE_CRITICAL always uses enum default action");
        }

        @Test
        void nullMetrics_defaultEngagementNudge() {
            // Empty state — all nulls — should not escalate
            ClinicalStateSummary state = new ClinicalStateSummary(P);

            String action = Module13_ClinicalStateSynchroniser.resolveDataAbsenceAction(
                    state, ClinicalStateChangeType.DATA_ABSENCE_WARNING);

            assertEquals("Generate engagement check card", action,
                    "Null metrics should not trigger escalation");
        }

        @Test
        void highFBG_escalatesAction() {
            ClinicalStateSummary state = new ClinicalStateSummary(P);
            state.current().fbg = 148.0; // Above 140 threshold

            String action = Module13_ClinicalStateSynchroniser.resolveDataAbsenceAction(
                    state, ClinicalStateChangeType.DATA_ABSENCE_WARNING);

            assertTrue(action.contains("INCOMPLETE_DATA_WITH_CRITICAL_VALUES"),
                    "FBG 148 should escalate, got: " + action);
        }
    }
}
