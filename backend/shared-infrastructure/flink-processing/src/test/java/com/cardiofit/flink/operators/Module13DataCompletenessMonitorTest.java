package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class Module13DataCompletenessMonitorTest {

    @Test
    void allModulesRecentlyReporting_scoreIs1() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + Module13TestBuilder.DAY_MS;

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertEquals(1.0, result.getCompositeScore(), 0.01);
        assertTrue(result.getDataGapFlags().isEmpty());
    }

    @Test
    void oneModuleStale_reducedScore() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 10 * Module13TestBuilder.DAY_MS;
        state.recordModuleSeen("module7", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module9", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module10b", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("enriched", now - Module13TestBuilder.DAY_MS);

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(result.getCompositeScore() < 1.0);
        assertTrue(result.getCompositeScore() > 0.5);
        assertTrue(result.getDataGapFlags().containsKey("module11b"));
    }

    @Test
    void allModulesStale14Days_nearZeroScore() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 28 * Module13TestBuilder.DAY_MS;

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(result.getCompositeScore() < 0.15);
        assertTrue(result.isDataAbsenceCritical());
    }

    @Test
    void specificModuleAbsence_flagged() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 10 * Module13TestBuilder.DAY_MS;
        state.getModuleLastSeenMs().remove("module12");
        state.recordModuleSeen("module7", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module9", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module10b", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module11b", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("enriched", now - Module13TestBuilder.DAY_MS);

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(result.getDataGapFlags().containsKey("module12"));
        assertEquals("NEVER_SEEN", result.getDataGapFlags().get("module12"));
    }

    @Test
    void cgmTierPatient_mealGapPenalisedMore() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        state.setDataTier("TIER_1_CGM");
        long now = Module13TestBuilder.BASE_TIME + 10 * Module13TestBuilder.DAY_MS;
        state.recordModuleSeen("module7", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module9", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("module11b", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("enriched", now - Module13TestBuilder.DAY_MS);

        Module13DataCompletenessMonitor.Result cgmResult =
                Module13DataCompletenessMonitor.evaluate(state, now);

        state.setDataTier("TIER_2_SMBG");
        Module13DataCompletenessMonitor.Result smbgResult =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(cgmResult.getCompositeScore() < smbgResult.getCompositeScore(),
                "CGM patient should be penalised more for meal pattern gaps");
    }
}
